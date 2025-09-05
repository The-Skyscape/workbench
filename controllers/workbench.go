package controllers

import (
	"log"
	"net/http"
	"time"
	"workbench/internal"
	"workbench/models"
	"workbench/services"

	"github.com/The-Skyscape/devtools/pkg/application"
)

// Workbench is a factory function with the prefix and instance
func Workbench() (string, *WorkbenchController) {
	return "workbench", &WorkbenchController{}
}

// WorkbenchController is the unified controller for workbench management
type WorkbenchController struct {
	application.BaseController
}

// Setup is called when the application is started
func (c *WorkbenchController) Setup(app *application.App) {
	c.BaseController.Setup(app)

	auth := app.Use("auth").(*AuthController)

	// Dashboard route
	http.Handle("/", app.Serve("dashboard.html", auth.Required))

	// Repository API routes (for dashboard)
	http.Handle("POST /repos/clone", app.ProtectFunc(c.cloneRepo, auth.Required))
	http.Handle("POST /repos/pull/{name}", app.ProtectFunc(c.pullRepo, auth.Required))
	http.Handle("POST /repos/delete/{name}", app.ProtectFunc(c.deleteRepo, auth.Required))
	
	// Partial routes for HTMX lazy loading
	http.Handle("GET /partials/activity", app.Serve("activity-log.html", auth.Required))

	// Coder proxy route
	http.Handle("/coder/", http.StripPrefix("/coder/", app.Protect(services.CoderProxy(), auth.Required)))

	// Ensure SSH key exists
	c.verifySSHKeys()
}

// Handle is called when each request is handled
func (c WorkbenchController) Handle(req *http.Request) application.Controller {
	c.Request = req
	return &c
}

// verifySSHKeys checks if SSH keys are valid
func (c *WorkbenchController) verifySSHKeys() {
	if internal.HasSSHKey() {
		log.Println("SSH key already exists")
		return
	}

	if err := internal.GenerateSSHKeyForUser(); err != nil {
		log.Printf("Failed to generate SSH key: %v", err)
	} else {
		log.Println("SSH key generated successfully")
	}
}

// HTTP Handlers

// cloneRepo clones a new repository
func (c *WorkbenchController) cloneRepo(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	name := r.FormValue("name")

	if url == "" {
		c.Render(w, r, "error-message.html", "Repository URL is required")
		return
	}

	// Make sure coder is running
	if !services.Coder.IsRunning() {
		c.Render(w, r, "error-message.html", "Coder service is not running")
		return
	}

	// Use internal package for business logic
	if err := internal.CloneRepository(url, name); err != nil {
		c.Render(w, r, "error-message.html", err)
		return
	}

	// Refresh the page
	c.Refresh(w, r)
}

// pullRepo pulls latest changes for a repository
func (c *WorkbenchController) pullRepo(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	if err := internal.PullRepository(name); err != nil {
		c.Render(w, r, "error-message.html", err)
		return
	}

	c.Refresh(w, r)
}

// deleteRepo deletes a repository
func (c *WorkbenchController) deleteRepo(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	if err := internal.DeleteRepository(name); err != nil {
		c.Render(w, r, "error-message.html", err)
		return
	}

	c.Refresh(w, r)
}

// Template Helper Methods

// GetRecentActivity returns recent activities (for templates)
func (c *WorkbenchController) GetRecentActivity() []*models.Activity {
	activities, err := models.Activities.Search("ORDER BY CreatedAt DESC LIMIT 20")
	if err != nil {
		log.Printf("Failed to fetch activities: %v", err)
	}
	return activities
}

// GetRepositories returns all repositories (for templates)
func (c *WorkbenchController) GetRepositories() []*models.Repository {
	repos, _ := models.Repositories.Search("")
	return repos
}

// HasRepositories checks if any repositories exist
func (c *WorkbenchController) HasRepositories() bool {
	count := models.Repositories.Count("")
	return count > 0
}

// IsCoderRunning checks if coder is running
func (c *WorkbenchController) IsCoderRunning() bool {
	return services.Coder.IsRunning()
}

// AppName returns the application name for templates
func (c *WorkbenchController) AppName() string {
	return "Workbench"
}

// AppDescription returns the application description for templates
func (c *WorkbenchController) AppDescription() string {
	return "Personal development environment with integrated VS Code"
}

// GetPublicKey returns the SSH public key for templates
func (c *WorkbenchController) GetPublicKey() string {
	key, err := internal.GetPublicKey()
	if err != nil {
		return ""
	}
	return key
}

// FormatActivityTime formats activity time in user's timezone
func (c *WorkbenchController) FormatActivityTime(t time.Time) string {
	return internal.FormatTimeInUserTZ(t, c.Request)
}
