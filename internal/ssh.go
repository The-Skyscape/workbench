package internal

import (
	"fmt"
	"strings"
	"workbench/models"
	"workbench/services"
)

// GenerateSSHKeyForUser creates an SSH key pair for Git authentication.
// Uses the email from settings or defaults to "user@workbench.local".
// This is called during application startup if no key exists.
// The generated key persists across container restarts via volume mount.
func GenerateSSHKeyForUser() error {
	email, _ := models.GetSetting("git_user_email")
	if email == "" {
		email = "user@workbench.local"
	}

	_, err := GenerateSSHKey(email)
	return err
}

// GenerateSSHKey generates an SSH key pair in the VS Code container.
// Attempts to create an Ed25519 key first (modern, secure, fast),
// falls back to RSA 4096-bit if Ed25519 is not supported.
//
// Parameters:
//   - email: Email address to associate with the key
//
// The function:
// 1. Creates ~/.ssh directory with proper permissions (700)
// 2. Generates key without passphrase for automation
// 3. Configures known_hosts for common Git providers
// 4. Saves public key to database settings
//
// Returns the public key content for display to user.
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

// GetPublicKey retrieves the SSH public key from the container.
// Checks for Ed25519 key first (preferred), then RSA key.
// The key content is suitable for adding to Git provider SSH settings.
//
// Returns:
//   - Public key content (e.g., "ssh-ed25519 AAAA... email@example.com")
//   - Error if no key exists
func GetPublicKey() (string, error) {
	// Try ed25519 first, then RSA
	cmd := "cat ~/.ssh/id_ed25519.pub 2>/dev/null || cat ~/.ssh/id_rsa.pub 2>/dev/null"
	publicKey, err := services.CoderExec(cmd)
	if err != nil {
		return "", fmt.Errorf("no SSH key found")
	}

	return strings.TrimSpace(publicKey), nil
}

// ConfigureSSHHosts pre-populates SSH known_hosts with common Git providers.
// This prevents "Host key verification failed" errors during git operations.
// Scans RSA keys from:
//   - github.com
//   - gitlab.com
//   - bitbucket.org
//   - codeberg.org
//
// Removes duplicate entries to keep the file clean.
// Non-critical: failures are logged but don't stop execution.
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

// HasSSHKey checks whether an SSH key pair exists in the container.
// Used during startup to determine if key generation is needed.
// Returns true if either Ed25519 or RSA key is present.
func HasSSHKey() bool {
	_, err := GetPublicKey()
	return err == nil
}
