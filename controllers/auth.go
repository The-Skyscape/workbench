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
// The returned prefix "auth" makes controller methods available in templates as {{auth.MethodName}}.
func Auth() (string, *AuthController) {
	// Create the base authentication controller with workbench cookie
	return "auth", &AuthController{
		Controller: models.Auth.Controller(
			authentication.WithCookie("workbench"),
		),
	}
}

// AuthController extends the devtools authentication controller with single-user logic.
// Unlike multi-user systems, this controller:
// - Allows only one admin user to be created
// - Renders auth forms inline rather than redirecting
// - Implements rate limiting on signin attempts
// - Uses 30-day session cookies for convenience
type AuthController struct {
	*authentication.Controller
}

// Setup initializes the authentication controller and registers HTTP routes.
// Called once during application startup. It intentionally does not call
// the parent Controller.Setup() to avoid route conflicts, instead registering
// only the POST endpoints needed for authentication actions.
// Routes registered:
// - POST /_auth/signup - Create the single admin user
// - POST /_auth/signin - Authenticate with rate limiting
// - POST /_auth/signout - Clear session and cookie
func (c *AuthController) Setup(app *application.App) {
	// Setup the base controller but don't call Controller.Setup to avoid route conflicts
	c.BaseController.Setup(app)

	// Register only the POST handlers for authentication
	http.HandleFunc("POST /_auth/signup", c.handleSignup)
	http.HandleFunc("POST /_auth/signin", c.handleSignin)
	http.HandleFunc("POST /_auth/signout", c.handleSignout)
}

// Handle prepares the controller for request-specific operations.
// Called for each HTTP request to set the request context in both
// the AuthController and its embedded authentication.Controller.
// This ensures template methods have access to the current request.
func (c AuthController) Handle(req *http.Request) application.Controller {
	// Update the request in both controllers
	c.Request = req
	c.Controller.Request = req
	return &c
}

// CurrentUser returns the currently authenticated user from the request context.
// Returns nil if no user is authenticated. This method is accessible in templates
// as {{auth.CurrentUser}} for displaying user information or conditional rendering.
func (c *AuthController) CurrentUser() *authentication.User {
	return c.Controller.CurrentUser()
}

// Required is an AccessCheck middleware that ensures a user is authenticated.
// Unlike traditional auth middleware that redirects, this renders auth forms
// inline for a seamless single-page experience with HTMX.
// Flow:
// 1. If no users exist → render signup form inline
// 2. If not authenticated → render signin form inline
// 3. If authenticated → allow request to proceed
// Returns true if authenticated, false if auth form was rendered.
func (c *AuthController) Required(app *application.App, w http.ResponseWriter, r *http.Request) bool {
	// If no users exist, render signup inline
	count := models.Auth.Users.Count("")
	if count == 0 {
		c.Render(w, r, "signup.html", nil)
		return false
	}

	// Check authentication
	if _, _, err := c.Authenticate(r); err != nil {
		// Not authenticated - render signin inline
		c.Render(w, r, "signin.html", nil)
		return false
	}

	// User is authenticated
	return true
}

// handleSignup handles POST /_auth/signup to create the single admin user.
// This endpoint can only be used once - when no users exist in the system.
// Validates all required fields, enforces password strength (min 8 chars),
// creates the admin user, establishes a session, and sets a 30-day cookie.
// Logs the signup activity for audit purposes.
func (c *AuthController) handleSignup(w http.ResponseWriter, r *http.Request) {
	// Check if any users exist (prevent multiple signups)
	count := models.Auth.Users.Count("")
	if count > 0 {
		c.Render(w, r, "error-message.html", errors.New("a user already exists"))
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		c.Render(w, r, "error-message.html", errors.New("invalid form data"))
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	handle := strings.TrimSpace(strings.ToLower(r.FormValue("handle")))
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	// Validate fields
	if name == "" || handle == "" || email == "" || password == "" {
		c.Render(w, r, "error-message.html", errors.New("all fields are required"))
		return
	}

	// Validate password strength
	if len(password) < 8 {
		c.Render(w, r, "error-message.html", errors.New("password must be at least 8 characters long"))
		return
	}

	// Create the single admin user with proper handle
	user, err := c.Signup(name, email, handle, password, true) // true = admin
	if err != nil {
		c.Render(w, r, "error-message.html", errors.New("failed to create user"))
		return
	}

	// Log signup activity
	internal.LogUserActivity("auth_signup", handle, "User account created")

	// Create session
	session, err := c.Sessions.Insert(&authentication.Session{UserID: user.ID})
	if err != nil {
		c.Render(w, r, "error-message.html", errors.New("failed to create session"))
		return
	}

	// Set auth cookie
	token, _ := session.Token()
	http.SetCookie(w, &http.Cookie{
		Name:     "workbench",
		Value:    token,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(time.Hour * 24 * 30), // 30 days for workbench
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
	})

	// Refresh the page - HTMX will reload
	c.Refresh(w, r)
}

// handleSignin handles POST /_auth/signin to authenticate the admin user.
// Implements rate limiting (5 attempts per minute per IP) to prevent brute force.
// Accepts either email or username in the handle field for flexibility.
// On success: creates a session, sets a 30-day cookie, and logs the activity.
// On failure: returns generic "invalid credentials" to avoid user enumeration.
func (c *AuthController) handleSignin(w http.ResponseWriter, r *http.Request) {
	// Check rate limit by IP
	clientIP := r.RemoteAddr
	if !internal.AuthRateLimiter.Allow(clientIP) {
		c.Render(w, r, "error-message.html", errors.New("too many login attempts, please wait a minute"))
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		c.Render(w, r, "error-message.html", errors.New("invalid form data"))
		return
	}

	handle := strings.TrimSpace(strings.ToLower(r.FormValue("handle")))
	password := r.FormValue("password")

	if handle == "" || password == "" {
		c.Render(w, r, "error-message.html", errors.New("email/username and password are required"))
		return
	}

	// Authenticate user - accepts email or username
	user, err := c.Signin(handle, password)
	if err != nil {
		c.Render(w, r, "error-message.html", errors.New("invalid credentials"))
		return
	}

	// Log signin activity
	internal.LogUserActivity("auth_signin", user.Handle, "User signed in")

	// Create session
	session, err := c.Sessions.Insert(&authentication.Session{UserID: user.ID})
	if err != nil {
		c.Render(w, r, "error-message.html", errors.New("failed to create session"))
		return
	}

	// Set auth cookie
	token, _ := session.Token()
	http.SetCookie(w, &http.Cookie{
		Name:     "workbench",
		Value:    token,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(time.Hour * 24 * 30), // 30 days for workbench
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
	})

	// Refresh the page - HTMX will reload
	c.Refresh(w, r)
}

// handleSignout handles POST /_auth/signout to end the user session.
// Clears the authentication cookie by setting MaxAge to -1, which instructs
// the browser to delete it immediately. The page is then refreshed via HTMX
// to return to the signin form.
func (c *AuthController) handleSignout(w http.ResponseWriter, r *http.Request) {
	// Clear the auth cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "workbench",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
	})

	// Refresh the page
	c.Refresh(w, r)
}
