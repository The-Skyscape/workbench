package models

import (
	"github.com/The-Skyscape/devtools/pkg/authentication"
	"github.com/The-Skyscape/devtools/pkg/database"
	"github.com/The-Skyscape/devtools/pkg/database/local"
)

var (
	// DB is the application's database
	DB = local.Database("workbench.db")

	// Auth is the DB's authentication collection (devtools authentication)
	Auth = authentication.Manage(DB)

	// Application collections
	Repositories = database.Manage(DB, new(Repository))
	Activities   = database.Manage(DB, new(Activity))
	Settings     = database.Manage(DB, new(Setting))
)

func init() {
	// Create database indexes for common queries
	createIndexes()
}

// createIndexes creates database indexes for common queries
func createIndexes() {
	// Workbench is single-user, so fewer indexes needed
	
	// Activity tracking
	Activities.Index("CreatedAt") // For ordering recent activities
	
	// Settings lookup
	Settings.Index("Key") // For key-value lookups
	
	// Repository management
	Repositories.Index("CreatedAt") // For ordering repositories
}

// InitializeForTesting reinitializes the global repositories with a test database
func InitializeForTesting(testDB *database.DynamicDB) {
	DB = testDB
	Auth = authentication.Manage(testDB)
	Repositories = database.Manage(testDB, new(Repository))
	Activities = database.Manage(testDB, new(Activity))
	Settings = database.Manage(testDB, new(Setting))
}