package internal

import (
	"time"
	"workbench/models"
)

// LogActivity logs a general activity to the database
func LogActivity(activityType, description string) {
	activity := &models.Activity{
		Type:        activityType,
		Repository:  "",
		Description: description,
		Author:      "System",
		Timestamp:   time.Now(),
	}
	models.Activities.Insert(activity)
}

// LogUserActivity logs a user-specific activity
func LogUserActivity(activityType, username, description string) {
	activity := &models.Activity{
		Type:        activityType,
		Repository:  "",
		Description: description,
		Author:      username,
		Timestamp:   time.Now(),
	}
	models.Activities.Insert(activity)
}

// LogRepoActivity logs a repository-specific activity
func LogRepoActivity(activityType, repository, description string) {
	activity := &models.Activity{
		Type:        activityType,
		Repository:  repository,
		Description: description,
		Author:      "System",
		Timestamp:   time.Now(),
	}
	models.Activities.Insert(activity)
}