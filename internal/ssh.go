package internal

import (
	"fmt"
	"strings"
	"workbench/models"
	"workbench/services"
)

// GenerateSSHKeyForUser creates a new SSH key using the configured user email
func GenerateSSHKeyForUser() error {
	email, _ := models.GetSetting("git_user_email")
	if email == "" {
		email = "user@workbench.local"
	}

	_, err := GenerateSSHKey(email)
	return err
}

// GenerateSSHKey creates a new SSH key in the container
func GenerateSSHKey(email string) (publicKey string, err error) {
	// First, ensure .ssh directory exists
	if _, err := services.CoderExec("mkdir -p ~/.ssh && chmod 700 ~/.ssh"); err != nil {
		return "", fmt.Errorf("failed to create SSH directory: %w", err)
	}

	// Generate the key
	cmd := fmt.Sprintf(`ssh-keygen -t ed25519 -C "%s" -f ~/.ssh/id_ed25519 -N "" -q`, email)
	if _, err := services.CoderExec(cmd); err != nil {
		// Try RSA if ed25519 fails
		cmd = fmt.Sprintf(`ssh-keygen -t rsa -b 4096 -C "%s" -f ~/.ssh/id_rsa -N "" -q`, email)
		if _, err := services.CoderExec(cmd); err != nil {
			return "", fmt.Errorf("failed to generate SSH key: %w", err)
		}
	}

	// Get the public key
	publicKey, err = GetPublicKey()
	if err != nil {
		return "", err
	}

	// Configure SSH for common git hosts
	if err := ConfigureSSHHosts(); err != nil {
		// Log but don't fail - this is not critical
		fmt.Printf("Warning: failed to configure SSH hosts: %v\n", err)
	}

	// Save to settings
	if err := models.SetSetting("ssh_public_key", publicKey, "ssh_key"); err != nil {
		// Log but don't fail - key was still generated
		fmt.Printf("Warning: failed to save SSH key to settings: %v\n", err)
	}

	return publicKey, nil
}

// GetPublicKey retrieves the current public key
func GetPublicKey() (string, error) {
	// Try ed25519 first, then RSA
	cmd := "cat ~/.ssh/id_ed25519.pub 2>/dev/null || cat ~/.ssh/id_rsa.pub 2>/dev/null"
	publicKey, err := services.CoderExec(cmd)
	if err != nil {
		return "", fmt.Errorf("no SSH key found")
	}

	return strings.TrimSpace(publicKey), nil
}

// ConfigureSSHHosts adds common git hosts to known_hosts
func ConfigureSSHHosts() error {
	hosts := []string{
		"github.com",
		"gitlab.com",
		"bitbucket.org",
		"codeberg.org",
	}

	for _, host := range hosts {
		cmd := fmt.Sprintf("ssh-keyscan -t rsa %s >> ~/.ssh/known_hosts 2>/dev/null", host)
		if _, err := services.CoderExec(cmd); err != nil {
			// Continue with other hosts even if one fails
			continue
		}
	}

	// Remove duplicates
	_, err := services.CoderExec("sort -u ~/.ssh/known_hosts -o ~/.ssh/known_hosts 2>/dev/null")
	return err
}

// HasSSHKey checks if an SSH key exists
func HasSSHKey() bool {
	_, err := GetPublicKey()
	return err == nil
}
