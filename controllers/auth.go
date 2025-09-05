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

// Auth returns the authentication controller with single-user logic
func Auth() (string, *AuthController) {
	// Create the base authentication controller with workbench cookie
	return "auth", &AuthController{
		Controller: models.Auth.Controller(
			authentication.WithCookie("workbench"),
		),
	}
}

// AuthController wraps devtools auth with single-user logic
type AuthController struct {
	*authentication.Controller
}

// Setup registers auth routes with custom single-user handlers
func (c *AuthController) Setup(app *application.App) {
	// Setup the base controller but don't call Controller.Setup to avoid route conflicts
	c.BaseController.Setup(app)

	// Register only the POST handlers for authentication
	http.HandleFunc("POST /_auth/signup", c.handleSignup)
	http.HandleFunc("POST /_auth/signin", c.handleSignin)
	http.HandleFunc("POST /_auth/signout", c.handleSignout)
}

// Handle prepares the controller for each request
func (c AuthController) Handle(req *http.Request) application.Controller {
	// Update the request in both controllers
	c.Request = req
	c.Controller.Request = req
	return &c
}

// CurrentUser returns the current user
func (c *AuthController) CurrentUser() *authentication.User {
	return c.Controller.CurrentUser()
}

// Required is an AccessCheck that ensures user is authenticated (inline rendering)
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

// handleSignup creates the single admin user
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

// handleSignin authenticates the single user
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

// handleSignout clears the session
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
