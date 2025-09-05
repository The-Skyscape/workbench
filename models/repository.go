package models

import (
	"github.com/The-Skyscape/devtools/pkg/application"
)

// Repository represents a git repository
type Repository struct {
	application.Model
	Name        string
	URL         string
	LocalPath   string
	Description string
	IsPrivate   bool
}

// Table returns the database table name
func (*Repository) Table() string {
	return "repositories"
}