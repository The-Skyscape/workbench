package models

import (
	"time"
	
	"github.com/The-Skyscape/devtools/pkg/application"
)

// Activity represents a user activity (commits, file changes, etc)
type Activity struct {
	application.Model
	Type        string    // commit, file_change, repo_clone, etc
	Repository  string    // Repository name
	Description string    // Activity description
	Author      string    // Who performed the activity
	Timestamp   time.Time // When it happened
	Metadata    string    // JSON metadata for the activity
}

// Table returns the database table name
func (*Activity) Table() string {
	return "activities"
}