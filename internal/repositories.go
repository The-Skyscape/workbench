// Package internal contains the business logic for the workbench application.
// It provides repository management, SSH key handling, system monitoring,
// activity logging, and utility functions used by controllers.
package internal

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"
	"workbench/models"
	"workbench/services"
)

// CloneRepository clones a Git repository into the VS Code server container.
// Parameters:
//   - url: The repository URL (HTTPS or SSH format)
//   - name: Optional repository name (auto-detected from URL if empty)
//
// The function:
// 1. Validates the repository doesn't already exist (case-insensitive)
// 2. Creates the repos directory if needed
// 3. Executes git clone in the container
// 4. Saves repository metadata to the database
// 5. Logs the activity for audit purposes
//
// Returns user-friendly error messages for common Git failures.
func CloneRepository(url, name string) error {
	if name == "" {
		// Auto-detect name from URL
		name = parseRepoName(url)
	}

	// Log for debugging
	log.Printf("Attempting to clone repository: URL=%s, Name=%s", url, name)

	// Validate name is not empty
	if name == "" {
		return fmt.Errorf("repository name cannot be empty")
	}

	// Check if repository already exists (case-insensitive)
	existing, err := models.Repositories.Find("WHERE LOWER(Name) = LOWER(?)", name)
	if err == nil && existing != nil && existing.Name != "" {
		log.Printf("Repository already exists in database: %s (found: %s)", name, existing.Name)
		return fmt.Errorf("a repository named '%s' already exists", existing.Name)
	}
	log.Printf("No existing repository found for name: %s (err: %v)", name, err)

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
	go models.Activities.Insert(&models.Activity{
		Type:        "repo_clone",
		Repository:  name,
		Description: fmt.Sprintf("Cloned repository %s", name),
		Author:      "System",
		Timestamp:   time.Now(),
	})

	return nil
}

// PullRepository fetches and merges latest changes from the remote repository.
// If the local directory is missing (e.g., after container rebuild), it attempts
// to re-clone the repository automatically.
//
// Parameters:
//   - repoName: The name of the repository in the database
//
// Common error scenarios handled:
//   - Missing local directory → automatic re-clone
//   - Authentication failures → SSH key reminder
//   - Merge conflicts → manual resolution required
//   - Uncommitted changes → stash or commit first
//
// Returns detailed error messages to guide user actions.
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
		log.Printf("Repository directory missing, attempting to re-clone: %s", repoName)
		services.CoderExec("mkdir -p /home/coder/repos")
		cmd := fmt.Sprintf("git clone %s %s 2>&1", repo.URL, repo.LocalPath)
		_, err := services.CoderExec(cmd)
		if err != nil {
			return fmt.Errorf("repository directory was missing and re-clone failed")
		}

		go models.Activities.Insert(&models.Activity{
			Type:        "repo_pull",
			Repository:  repo.Name,
			Description: fmt.Sprintf("Re-cloned missing repository %s", repoName),
			Author:      "System",
			Timestamp:   time.Now(),
		})

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
	go models.Activities.Insert(&models.Activity{
		Type:        "repo_pull",
		Repository:  repoName,
		Description: fmt.Sprintf("Synced repository %s", repoName),
		Author:      "System",
		Timestamp:   time.Now(),
	})

	return nil
}

// DeleteRepository permanently removes a repository from both filesystem and database.
// This operation cannot be undone. The function:
// 1. Verifies the repository exists in the database
// 2. Deletes the repository directory and all contents
// 3. Removes the database record
// 4. Logs the deletion for audit purposes
//
// Parameters:
//   - name: The repository name to delete
//
// Returns error if repository not found or deletion fails.
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
	go models.Activities.Insert(&models.Activity{
		Type:        "repo_delete",
		Repository:  name,
		Description: fmt.Sprintf("Deleted repository %s", name),
		Author:      "System",
		Timestamp:   time.Now(),
	})

	return nil
}

// parseRepoName extracts a clean repository name from various Git URL formats.
// Handles:
//   - HTTPS URLs: https://github.com/user/repo.git → "repo"
//   - SSH URLs: git@github.com:user/repo.git → "repo"
//   - URLs with or without .git extension
//   - URLs with trailing slashes
//
// Returns empty string if URL is invalid or name cannot be extracted.
func parseRepoName(url string) string {
	// Handle empty URL
	if url == "" {
		return ""
	}

	// Clean up the URL
	url = strings.TrimSpace(url)
	url = strings.TrimSuffix(url, "/") // Remove trailing slash
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
		return ""
	}

	// Handle HTTPS URLs
	parts := strings.Split(url, "/")
	if len(parts) > 0 && parts[len(parts)-1] != "" {
		return parts[len(parts)-1]
	}

	return ""
}
