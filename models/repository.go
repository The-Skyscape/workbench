// Package models defines the database entities for the workbench application.
// All models embed application.Model for standard fields (ID, CreatedAt, UpdatedAt).
package models

import (
	"fmt"
	"strings"
	"workbench/services"

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

// GetRepositorySize calculates the total disk usage of a repository.
// Uses the 'du' command in the container to get accurate size including
// all files, git history, and working tree.
func (repo *Repository) Size() (int64, error) {
	// Get size using du command in coder container
	cmd := fmt.Sprintf("du -sb %s | cut -f1", repo.LocalPath)
	output, err := services.CoderExec(cmd)
	if err != nil {
		return 0, err
	}

	// Parse the size
	var size int64
	fmt.Sscanf(strings.TrimSpace(output), "%d", &size)
	return size, nil
}
