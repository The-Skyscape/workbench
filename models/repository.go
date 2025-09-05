// Package models defines the database entities for the workbench application.
// All models embed application.Model for standard fields (ID, CreatedAt, UpdatedAt).
package models

import (
	"github.com/The-Skyscape/devtools/pkg/application"
)

// Repository represents a cloned Git repository in the workbench.
// Tracks both the remote URL and local filesystem path within the container.
// The LocalPath is typically /home/coder/repos/{name} in the VS Code container.
type Repository struct {
	application.Model
	Name        string
	URL         string
	LocalPath   string
	Description string
	IsPrivate   bool
}

// Table returns the database table name for the Repository model.
// Required by the devtools ORM for database operations.
func (*Repository) Table() string {
	return "repositories"
}