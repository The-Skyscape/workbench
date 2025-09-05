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

// Workbench is a factory function that returns the controller prefix and instance.
// The prefix "workbench" makes controller methods available in templates as {{workbench.MethodName}}.
// This controller manages the main dashboard, repository operations, and VS Code integration.
func Workbench() (string, *WorkbenchController) {
	return "workbench", &WorkbenchController{}
}

// WorkbenchController manages the core workbench functionality including:
// - System dashboard with real-time monitoring
// - Git repository management (clone, pull, delete)
// - VS Code integration via code-server proxy
// - SSH key management for repository access
// - Activity logging and display
type WorkbenchController struct {
	application.BaseController
}

// Setup initializes the workbench controller during application startup.
// It registers HTTP routes for the dashboard and repository operations,
// sets up the VS Code proxy, and ensures SSH keys exist for Git operations.
// Routes registered:
// - GET / - Main dashboard with system stats
// - POST /repos/clone - Clone a new repository
// - POST /repos/pull/{name} - Pull latest changes
// - POST /repos/delete/{name} - Delete a repository
// - GET /partials/activity - Activity log partial for HTMX
// - /coder/* - Proxied VS Code server interface
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

// Handle prepares the controller for request-specific operations.
// Called for each HTTP request to set the request context, making it
// available to template helper methods that need request information
// (e.g., for timezone formatting or user context).
func (c WorkbenchController) Handle(req *http.Request) application.Controller {
	c.Request = req
	return &c
}

// verifySSHKeys ensures SSH keys exist for Git operations.
// Called during setup to generate keys if they don't exist.
// Keys are stored in the container's ~/.ssh directory and persist
// across restarts via the data volume mount.
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

// ============================================================================
// HTTP Handlers - Process repository management requests
// ============================================================================

// cloneRepo handles POST /repos/clone to clone a Git repository.
// Accepts URL (required) and name (optional, auto-detected from URL).
// Validates the Coder service is running, clones via Git in the container,
// saves repository metadata to database, and logs the activity.
// Returns error messages for duplicate names or clone failures.
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

// pullRepo handles POST /repos/pull/{name} to update a repository.
// Executes git pull in the repository directory within the Coder container.
// Updates the last pulled timestamp in the database and logs the activity.
// Returns error if repository doesn't exist or pull fails.
func (c *WorkbenchController) pullRepo(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	if err := internal.PullRepository(name); err != nil {
		c.Render(w, r, "error-message.html", err)
		return
	}

	c.Refresh(w, r)
}

// deleteRepo handles POST /repos/delete/{name} to remove a repository.
// Deletes both the repository directory from the filesystem and its
// database record. This action is permanent and cannot be undone.
// Logs the deletion activity for audit purposes.
func (c *WorkbenchController) deleteRepo(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	if err := internal.DeleteRepository(name); err != nil {
		c.Render(w, r, "error-message.html", err)
		return
	}

	c.Refresh(w, r)
}

// ============================================================================
// Template Helper Methods - Accessible in views as {{workbench.MethodName}}
// ============================================================================

// GetRecentActivity returns the 20 most recent activity log entries.
// Used in templates to display user actions and system events.
// Ordered by creation time descending (newest first).
// Template usage: {{range workbench.GetRecentActivity}}...{{end}}
func (c *WorkbenchController) GetRecentActivity() []*models.Activity {
	activities, err := models.Activities.Search("ORDER BY CreatedAt DESC LIMIT 20")
	if err != nil {
		log.Printf("Failed to fetch activities: %v", err)
	}
	return activities
}

// GetRepositories returns all cloned repositories alphabetically sorted.
// Used in dashboard to display repository list with actions.
// Template usage: {{range workbench.GetRepositories}}...{{end}}
func (c *WorkbenchController) GetRepositories() []*models.Repository {
	repos, _ := models.Repositories.Search("ORDER BY Name ASC")
	return repos
}

// HasRepositories returns true if at least one repository is cloned.
// Used for conditional rendering in templates to show empty state or list.
// Template usage: {{if workbench.HasRepositories}}...{{else}}...{{end}}
func (c *WorkbenchController) HasRepositories() bool {
	count := models.Repositories.Count("")
	return count > 0
}

// IsCoderRunning returns true if the VS Code server container is active.
// Used to conditionally enable/disable IDE features in the UI.
// Template usage: {{if workbench.IsCoderRunning}}...{{end}}
func (c *WorkbenchController) IsCoderRunning() bool {
	return services.Coder.IsRunning()
}

// GetPublicKey returns the SSH public key for repository authentication.
// Used in clone modal to allow users to copy key for Git server setup.
// Returns empty string if key doesn't exist or can't be read.
// Template usage: {{workbench.GetPublicKey}}
func (c *WorkbenchController) GetPublicKey() string {
	key, err := internal.GetPublicKey()
	if err != nil {
		return ""
	}
	return key
}

// FormatActivityTime converts UTC timestamps to user's local timezone.
// Detects timezone from request headers or defaults to UTC.
// Returns human-readable format like "Jan 2, 3:04 PM".
// Template usage: {{workbench.FormatActivityTime .CreatedAt}}
func (c *WorkbenchController) FormatActivityTime(t time.Time) string {
	return internal.FormatTimeInUserTZ(t, c.Request)
}
