package internal

import (
	"fmt"
	"workbench/models"
	"workbench/services"
)

// ConfigureGitUser sets up git user configuration
func ConfigureGitUser(name, email string) error {
	cmds := []string{
		fmt.Sprintf(`git config --global user.name "%s"`, name),
		fmt.Sprintf(`git config --global user.email "%s"`, email),
		`git config --global init.defaultBranch main`,
	}

	for _, cmd := range cmds {
		if _, err := services.CoderExec(cmd); err != nil {
			return fmt.Errorf("failed to configure git: %w", err)
		}
	}

	// Save to settings
	models.SetSetting("git_user_name", name, "git_config")
	models.SetSetting("git_user_email", email, "git_config")

	return nil
}
