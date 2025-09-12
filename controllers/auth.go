package controllers

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"workbench/internal"
	"workbench/models"

	"github.com/The-Skyscape/devtools/pkg/application"
	"github.com/The-Skyscape/devtools/pkg/authentication"
)

// Auth is a factory function that returns the controller prefix and instance.
// It creates an authentication controller configured for single-user operation
// with a persistent "workbench" cookie for session management.
func Auth() (string, *AuthController) {
	// Create new auth toolkit
	auth := authentication.New("") // Uses AUTH_SECRET env var

	return "auth", &AuthController{
		Controller: application.Controller{},
		auth:       auth,
		cookieName: "workbench",
		Collection: models.Auth, // For backward compatibility
	}
}

// AuthController provides single-user authentication using devtools primitives.
// Unlike multi-user systems, this controller:
// - Allows only one admin user to be created
// - Renders auth forms inline rather than redirecting
// - Implements rate limiting on signin attempts
// - Uses 30-day session cookies for convenience
type AuthController struct {
	application.Controller
	*authentication.Collection // Embed for backward compatibility
	auth                       *authentication.Auth
	cookieName                 string
}

// Setup initializes the authentication controller and registers HTTP routes.
func (c *AuthController) Setup(app *application.App) {
	c.Controller.Setup(app)

	// Register only the POST handlers for authentication
	http.HandleFunc("POST /_auth/signup", c.handleSignup)
	http.HandleFunc("POST /_auth/signin", c.handleSignin)
	http.HandleFunc("POST /_auth/signout", c.handleSignout)
}

// Handle prepares the controller for request-specific operations.
func (c AuthController) Handle(req *http.Request) application.Handler {
	c.Request = req
	return &c
}

// handleSignup handles the signup form submission (single user only)
func (c *AuthController) handleSignup(w http.ResponseWriter, r *http.Request) {
	c.SetRequest(r)

	// Check if a user already exists (single-user system)
	if c.Collection.Users.Count("") > 0 {
		c.RenderError(w, r, errors.New("A user already exists. This is a single-user system."))
		return
	}

	// Rate limiting check
	clientIP := r.RemoteAddr // Simple IP for single-user system
	if !internal.AuthRateLimiter.Allow(clientIP + ":signup") {
		c.RenderError(w, r, errors.New("Too many attempts. Please wait a minute and try again."))
		return
	}

	// Get form values
	name := strings.TrimSpace(r.FormValue("name"))
	handle := strings.TrimSpace(strings.ToLower(r.FormValue("handle")))
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	// Validate all fields
	if name == "" || handle == "" || email == "" || password == "" {
		c.RenderError(w, r, errors.New("All fields are required"))
		return
	}

	// Create the single admin user
	user, err := c.Collection.Signup(name, email, handle, password, true)
	if err != nil {
		c.RenderError(w, r, err)
		return
	}

	// Generate session token
	token, err := c.auth.GenerateSessionToken(user.ID, 30*24*time.Hour)
	if err != nil {
		c.RenderError(w, r, err)
		return
	}

	// Set cookie
	c.auth.SetCookie(w, c.cookieName, token, time.Now().Add(30*24*time.Hour), r.TLS != nil)

	// Create session record
	models.Auth.Sessions.Insert(&authentication.Session{
		UserID: user.ID,
	})

	// Log the activity
	internal.LogUserActivity("user_signup", name, "User created account")

	// Refresh the page (HTMX will handle the update)
	c.Refresh(w, r)
}

// handleSignin processes signin form submission with rate limiting
func (c *AuthController) handleSignin(w http.ResponseWriter, r *http.Request) {
	c.SetRequest(r)

	// Rate limiting check - 5 attempts per minute per IP
	clientIP := r.RemoteAddr // Simple IP for single-user system
	if !internal.AuthRateLimiter.Allow(clientIP + ":signin") {
		c.RenderError(w, r, errors.New("Too many signin attempts. Please wait a minute and try again."))
		internal.LogActivity("signin_rate_limited", "Signin rate limited")
		return
	}

	handle := strings.TrimSpace(strings.ToLower(r.FormValue("handle")))
	password := r.FormValue("password")

	if handle == "" || password == "" {
		c.RenderError(w, r, errors.New("Email/username and password are required"))
		return
	}

	// Authenticate user
	user, err := c.Collection.Signin(handle, password)
	if err != nil {
		internal.LogActivity("signin_failed", "Failed signin attempt")
		c.RenderError(w, r, errors.New("Invalid credentials"))
		return
	}

	// Generate session token
	token, err := c.auth.GenerateSessionToken(user.ID, 30*24*time.Hour)
	if err != nil {
		c.RenderError(w, r, err)
		return
	}

	// Set cookie
	c.auth.SetCookie(w, c.cookieName, token, time.Now().Add(30*24*time.Hour), r.TLS != nil)

	// Create session record
	models.Auth.Sessions.Insert(&authentication.Session{
		UserID: user.ID,
	})

	// Log successful signin
	internal.LogUserActivity("user_signin", user.Handle, "User signed in")

	// Refresh the page
	c.Refresh(w, r)
}

// handleSignout processes signout
func (c *AuthController) handleSignout(w http.ResponseWriter, r *http.Request) {
	c.SetRequest(r)

	// Get current user before clearing
	user := c.CurrentUser()

	// Clear cookie
	c.auth.ClearCookie(w, c.cookieName)

	// Log the signout
	if user != nil {
		internal.LogUserActivity("user_signout", user.Handle, "User signed out")
	}

	// Refresh to show signin page
	c.Refresh(w, r)
}

// CurrentUser returns the currently authenticated user
func (c *AuthController) CurrentUser() *authentication.User {
	// Try to get token from cookie
	token, err := c.auth.GetTokenFromCookie(c.Request, c.cookieName)
	if err != nil {
		return nil
	}

	// Validate token
	claims, err := c.auth.ValidateToken(token)
	if err != nil {
		return nil
	}

	// Get user ID from claims
	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil
	}

	// Get user from database
	user, err := c.Collection.Users.Get(userID)
	if err != nil {
		return nil
	}

	return user
}

// IsAuthenticated checks if a user is authenticated
func (c *AuthController) IsAuthenticated() bool {
	return c.CurrentUser() != nil
}

// Required is an AccessCheck middleware that ensures a user is authenticated.
// Unlike traditional auth middleware that redirects, this renders auth forms
// inline for a seamless single-page experience with HTMX.
func (c *AuthController) Required(app *application.App, w http.ResponseWriter, r *http.Request) bool {
	// If no users exist, render signup form inline
	if c.Collection.Users.Count("") == 0 {
		app.Render(w, r, "signup.html", nil)
		return false
	}

	// Check authentication
	user := c.GetAuthenticatedUser(r)
	if user == nil {
		// Not authenticated - render signin form inline
		app.Render(w, r, "signin.html", nil)
		return false
	}

	// User is authenticated
	return true
}

// Optional always returns true (public access)
func (c *AuthController) Optional(app *application.App, w http.ResponseWriter, r *http.Request) bool {
	return true
}

// GetAuthenticatedUser gets the current user from a request
func (c *AuthController) GetAuthenticatedUser(r *http.Request) *authentication.User {
	// Try to get token from cookie
	token, err := c.auth.GetTokenFromCookie(r, c.cookieName)
	if err != nil {
		return nil
	}

	// Validate token
	claims, err := c.auth.ValidateToken(token)
	if err != nil {
		return nil
	}

	// Get user ID from claims
	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil
	}

	// Get user from database
	user, err := c.Collection.Users.Get(userID)
	if err != nil {
		return nil
	}

	return user
}
