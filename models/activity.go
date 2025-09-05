package models

import (
	"time"
	
	"github.com/The-Skyscape/devtools/pkg/application"
)

// Activity represents an audit log entry for user and system actions.
// Used to track all significant events in the workbench for security
// and debugging purposes. Activities are displayed in the dashboard
// to provide visibility into recent operations.
type Activity struct {
	application.Model
	Type        string    // Activity type: repo_clone, repo_pull, repo_delete, auth_signin, etc.
	Repository  string    // Repository name if applicable, empty for system activities
	Description string    // Human-readable description of what happened
	Author      string    // User handle or "system" for automated actions
	Timestamp   time.Time // When the activity occurred (UTC)
	Metadata    string    // Optional JSON data for additional context
}

// Table returns the database table name for the Activity model.
// Required by the devtools ORM for database operations.
func (*Activity) Table() string {
	return "activities"
}