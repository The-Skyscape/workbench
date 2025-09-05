package internal

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"workbench/models"
	"workbench/services"
)

// CloneRepository clones a repository using git in the coder container
func CloneRepository(url, name string) error {
	if name == "" {
		// Auto-detect name from URL
		name = parseRepoName(url)
	}

	// Ensure repos directory exists
	services.CoderExec("mkdir -p /home/coder/repos")

	targetDir := filepath.Join("/home/coder/repos", name)

	// Execute git clone in the coder container
	cmd := fmt.Sprintf("git clone %s %s", url, targetDir)
	output, err := services.CoderExec(cmd)
	if err != nil {
		return fmt.Errorf("clone failed: %v\nOutput: %s", err, output)
	}

	// Get default branch
	branchCmd := fmt.Sprintf("cd %s && git branch --show-current", targetDir)
	branch, _ := services.CoderExec(branchCmd)
	branch = strings.TrimSpace(branch)
	if branch == "" {
		branch = "main"
	}

	// Save to database
	repo := &models.Repository{
		Name:          name,
		URL:           url,
		LocalPath:     targetDir,
		DefaultBranch: branch,
		LastPulled:    time.Now(),
		IsPrivate:     strings.Contains(url, "git@"),
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
		return fmt.Errorf("repository not found: %s", repoName)
	}
	cmd := fmt.Sprintf("cd %s && git pull", repo.LocalPath)
	output, err := services.CoderExec(cmd)
	if err != nil {
		return fmt.Errorf("pull failed: %v\nOutput: %s", err, output)
	}

	// Update last pulled time
	repo.LastPulled = time.Now()
	models.Repositories.Update(repo)

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
