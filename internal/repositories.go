package internal

import (
	"fmt"
	"path/filepath"
	"strings"
	"workbench/models"
	"workbench/services"
)

// CloneRepository clones a repository using git in the coder container
func CloneRepository(url, name string) error {
	if name == "" {
		// Auto-detect name from URL
		name = parseRepoName(url)
	}

	// Check if repository already exists
	existing, _ := models.Repositories.Find("WHERE Name = ?", name)
	if existing != nil {
		return fmt.Errorf("a repository named '%s' already exists", name)
	}

	// Ensure repos directory exists
	services.CoderExec("mkdir -p /home/coder/repos")

	targetDir := filepath.Join("/home/coder/repos", name)
	
	// Check if directory already exists
	checkCmd := fmt.Sprintf("test -d %s && echo exists", targetDir)
	exists, _ := services.CoderExec(checkCmd)
	if strings.TrimSpace(exists) == "exists" {
		return fmt.Errorf("directory %s already exists - please choose a different name", name)
	}

	// Execute git clone in the coder container
	cmd := fmt.Sprintf("git clone %s %s 2>&1", url, targetDir)
	output, err := services.CoderExec(cmd)
	if err != nil {
		// Parse common git errors for better messages
		outputStr := string(output)
		if strings.Contains(outputStr, "Permission denied") || strings.Contains(outputStr, "Could not read from remote") {
			return fmt.Errorf("authentication failed - for private repos, add your SSH key to the git provider")
		}
		if strings.Contains(outputStr, "does not exist") || strings.Contains(outputStr, "not found") {
			return fmt.Errorf("repository not found - check the URL is correct")
		}
		if strings.Contains(outputStr, "Could not resolve") || strings.Contains(outputStr, "unable to access") {
			return fmt.Errorf("network error - check your connection and try again")
		}
		// Generic error
		return fmt.Errorf("failed to clone repository")
	}

	// Save to database
	repo := &models.Repository{
		Name:      name,
		URL:       url,
		LocalPath: targetDir,
		IsPrivate: strings.Contains(url, "git@"),
	}
	_, err = models.Repositories.Insert(repo)
	if err != nil {
		return fmt.Errorf("failed to save repository: %w", err)
	}

	// Log activity
	LogRepoActivity("repo_clone", name, fmt.Sprintf("Cloned repository %s", name))

	return nil
}

// PullRepository pulls latest changes for a repository
func PullRepository(repoName string) error {
	repo, err := models.Repositories.Find("WHERE Name = ?", repoName)
	if err != nil {
		return fmt.Errorf("repository '%s' not found", repoName)
	}
	
	// Check if directory exists
	checkCmd := fmt.Sprintf("test -d %s && echo exists", repo.LocalPath)
	exists, _ := services.CoderExec(checkCmd)
	if strings.TrimSpace(exists) != "exists" {
		// Try to re-clone if directory is missing
		Log.Warn("Repository directory missing, attempting to re-clone: %s", repoName)
		services.CoderExec("mkdir -p /home/coder/repos")
		cmd := fmt.Sprintf("git clone %s %s 2>&1", repo.URL, repo.LocalPath)
		_, err := services.CoderExec(cmd)
		if err != nil {
			return fmt.Errorf("repository directory was missing and re-clone failed")
		}
		LogRepoActivity("repo_pull", repoName, fmt.Sprintf("Re-cloned missing repository %s", repoName))
		return nil
	}
	
	cmd := fmt.Sprintf("cd %s && git pull 2>&1", repo.LocalPath)
	output, err := services.CoderExec(cmd)
	if err != nil {
		outputStr := string(output)
		// Check for common issues
		if strings.Contains(outputStr, "Permission denied") {
			return fmt.Errorf("authentication failed - check your SSH key is added to the git provider")
		}
		if strings.Contains(outputStr, "merge conflict") || strings.Contains(outputStr, "Merge conflict") {
			return fmt.Errorf("merge conflicts detected - resolve manually in VS Code")
		}
		if strings.Contains(outputStr, "uncommitted changes") || strings.Contains(outputStr, "Your local changes") {
			return fmt.Errorf("uncommitted changes - commit or stash them first")
		}
		// Generic error
		return fmt.Errorf("failed to pull latest changes")
	}

	// Log activity
	LogRepoActivity("repo_pull", repoName, fmt.Sprintf("Synced repository %s", repoName))

	return nil
}

// DeleteRepository removes a repository
func DeleteRepository(name string) error {
	repo, err := models.Repositories.Find("WHERE Name = ?", name)
	if err != nil {
		return fmt.Errorf("repository not found: %s", name)
	}

	// Remove from filesystem
	cmd := fmt.Sprintf("rm -rf %s", repo.LocalPath)
	if _, err := services.CoderExec(cmd); err != nil {
		return fmt.Errorf("failed to delete repository files: %w", err)
	}

	// Remove from database
	if err := models.Repositories.Delete(repo); err != nil {
		return fmt.Errorf("failed to delete repository record: %w", err)
	}

	// Log activity
	LogRepoActivity("repo_delete", name, fmt.Sprintf("Deleted repository %s", name))

	return nil
}


// GetRepositorySize returns the size of a repository in bytes
func GetRepositorySize(name string) (int64, error) {
	repo, err := models.Repositories.Find("WHERE Name = ?", name)
	if err != nil {
		return 0, err
	}

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

// parseRepoName extracts repository name from URL
func parseRepoName(url string) string {
	// Handle empty URL
	if url == "" {
		return "repository"
	}
	
	// Remove .git suffix if present
	url = strings.TrimSuffix(url, ".git")

	// Handle SSH URLs (git@github.com:user/repo)
	if strings.HasPrefix(url, "git@") {
		parts := strings.Split(url, ":")
		if len(parts) > 1 {
			path := parts[1]
			parts = strings.Split(path, "/")
			if len(parts) > 0 && parts[len(parts)-1] != "" {
				return parts[len(parts)-1]
			}
		}
	}

	// Handle HTTPS URLs
	parts := strings.Split(url, "/")
	if len(parts) > 0 && parts[len(parts)-1] != "" {
		return parts[len(parts)-1]
	}

	return "repository"
}
