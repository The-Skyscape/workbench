package controllers

import (
	"errors"
	"net/http"
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
	return "auth", &AuthController{
		Controller: models.Auth.Controller(),
	}
}

// AuthController provides single-user authentication using devtools primitives.
// Unlike multi-user systems, this controller:
// - Allows only one admin user to be created
// - Renders auth forms inline rather than redirecting
// - Implements rate limiting on signin attempts
// - Uses 30-day session cookies for convenience
type AuthController struct {
	*authentication.Controller // Embed for backward compatibility
}

// Setup initializes the authentication controller and registers HTTP routes.
func (c *AuthController) Setup(app *application.App) {
	c.Controller.Controller.Setup(app)

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
	// Check if a user already exists (single-user system)
	if c.Collection.Users.Count("") > 0 {
		c.RenderError(w, r, errors.New("a user already exists. This is a single-user system"))
		return
	}

	// Rate limiting check
	clientIP := r.RemoteAddr // Simple IP for single-user system
	if !internal.AuthRateLimiter.Allow(clientIP + ":signup") {
		c.RenderError(w, r, errors.New("too many attempts. Please wait a minute and try again"))
		return
	}

	c.Controller.HandleSignup(w, r)
}

// handleSignin processes signin form submission with rate limiting
func (c *AuthController) handleSignin(w http.ResponseWriter, r *http.Request) {
	// Rate limiting check - 5 attempts per minute per IP
	clientIP := r.RemoteAddr // Simple IP for single-user system
	if !internal.AuthRateLimiter.Allow(clientIP + ":signin") {
		c.RenderError(w, r, errors.New("too many signin attempts. Please wait a minute and try again"))
		internal.LogActivity("signin_rate_limited", "Signin rate limited")
		return
	}

	c.Controller.HandleSignin(w, r)
}

// handleSignout processes signout
func (c *AuthController) handleSignout(w http.ResponseWriter, r *http.Request) {
	c.Controller.HandleSignout(w, r)
}
