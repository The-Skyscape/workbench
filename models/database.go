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