package models

import (
	"time"
	
	"github.com/The-Skyscape/devtools/pkg/application"
)

// Repository represents a git repository
type Repository struct {
	application.Model
	Name          string
	URL           string
	LocalPath     string
	Description   string
	LastPulled    time.Time
	IsPrivate     bool
	DefaultBranch string
}

// Table returns the database table name
func (*Repository) Table() string {
	return "repositories"
}