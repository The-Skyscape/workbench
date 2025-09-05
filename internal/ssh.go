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
	services.CoderExec("mkdir -p ~/.ssh && chmod 700 ~/.ssh")

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
	ConfigureSSHHosts()

	// Save to settings
	models.SetSetting("ssh_public_key", publicKey, "ssh_key")

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

// GetPrivateKeyPath returns the path to the private key
func GetPrivateKeyPath() (string, error) {
	// Check which key exists
	cmd := "test -f ~/.ssh/id_ed25519 && echo '~/.ssh/id_ed25519' || test -f ~/.ssh/id_rsa && echo '~/.ssh/id_rsa'"
	path, err := services.CoderExec(cmd)
	if err != nil {
		return "", fmt.Errorf("no SSH key found")
	}

	return strings.TrimSpace(path), nil
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
		services.CoderExec(cmd)
	}

	// Remove duplicates
	services.CoderExec("sort -u ~/.ssh/known_hosts -o ~/.ssh/known_hosts 2>/dev/null")

	return nil
}

// TestSSHConnection tests SSH connection to a git host
func TestSSHConnection(host string) (bool, string) {
	// Extract hostname from git URL if needed
	if strings.Contains(host, "@") {
		parts := strings.Split(host, "@")
		if len(parts) > 1 {
			host = strings.Split(parts[1], ":")[0]
		}
	}

	cmd := fmt.Sprintf("ssh -T git@%s 2>&1", host)
	output, _ := services.CoderExec(cmd)

	// GitHub returns "Hi username!" on successful auth
	// GitLab returns "Welcome to GitLab"
	// Even with exit code 1, these indicate successful auth
	if strings.Contains(output, "Hi ") ||
		strings.Contains(output, "Welcome") ||
		strings.Contains(output, "authenticated") {
		return true, output
	}

	return false, output
}

// ImportSSHKey imports an existing SSH private key
func ImportSSHKey(privateKey string) error {
	// Ensure .ssh directory exists
	services.CoderExec("mkdir -p ~/.ssh && chmod 700 ~/.ssh")

	// Detect key type
	keyType := "id_rsa"
	if strings.Contains(privateKey, "BEGIN OPENSSH PRIVATE KEY") ||
		strings.Contains(privateKey, "BEGIN EC PRIVATE KEY") {
		keyType = "id_ed25519"
	}

	// Write the private key
	cmd := fmt.Sprintf("echo '%s' > ~/.ssh/%s && chmod 600 ~/.ssh/%s", privateKey, keyType, keyType)
	if _, err := services.CoderExec(cmd); err != nil {
		return fmt.Errorf("failed to import SSH key: %w", err)
	}

	// Generate public key from private key
	cmd = fmt.Sprintf("ssh-keygen -y -f ~/.ssh/%s > ~/.ssh/%s.pub", keyType, keyType)
	if _, err := services.CoderExec(cmd); err != nil {
		return fmt.Errorf("failed to generate public key: %w", err)
	}

	// Configure SSH hosts
	ConfigureSSHHosts()

	// Save public key to settings
	publicKey, _ := GetPublicKey()
	if publicKey != "" {
		models.SetSetting("ssh_public_key", publicKey, "ssh_key")
	}

	return nil
}

// HasSSHKey checks if an SSH key exists
func HasSSHKey() bool {
	_, err := GetPublicKey()
	return err == nil
}
