// Package services manages external service integrations for the workbench.
// Currently provides VS Code server (code-server) integration via Docker containers.
package services

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/The-Skyscape/devtools/pkg/containers"
)

// Coder is the VS Code server (code-server) container configuration.
// Provides a full VS Code IDE accessible via web browser.
// Configuration:
//   - No authentication (handled by workbench)
//   - Binds to port 8080 internally
//   - Uses host network for simplicity
//   - Mounts persistent directories for code and config
//   - Auto-restarts on failure
var Coder = &containers.Service{
	Host:          containers.Local(),
	Name:          "workbench-coder",
	Image:         "codercom/code-server:latest",
	Command:       "--auth none --bind-addr 0.0.0.0:8080",
	Network:       "skyscape-internal",
	RestartPolicy: "always",
	Mounts: map[string]string{
		"/home/.ssh":                                 "/home/.ssh",          // SSH keys for Git
		"/mnt/data/services/workbench-coder/":        "/home/coder",         // Main workspace
		"/mnt/data/services/workbench-coder/.config": "/home/coder/.config", // VS Code config
	},
}

// init automatically starts the VS Code server container during package initialization.
// Skipped during tests to avoid Docker dependencies.
// Creates necessary directories with proper permissions and launches the container.
// If container already exists and is running, reuses it.
func init() {
	// Skip initialization during tests
	if strings.HasSuffix(os.Args[0], ".test") {
		return
	}

	log.Println("Initializing Coder service...")

	// Check if container already exists
	existing := containers.Local().Service("workbench-coder")
	if existing != nil && existing.IsRunning() {
		log.Println("Coder service already running")
		Coder = existing
		return
	}

	prepareScript := `
		mkdir -p /mnt/data/services/workbench-coder
		mkdir -p /mnt/data/services/workbench-coder/.config
		mkdir -p /mnt/data/services/workbench-coder/repos
		chmod -R 777 /mnt/data/services/workbench-coder
		chown -R 1000:1000 /mnt/data/services/workbench-coder || true
	`

	if err := containers.Local().Exec("bash", "-c", prepareScript); err != nil {
		log.Fatalf("Failed to prepare coder directrories: %v", err)
	}

	// Launch the container
	log.Println("Starting Coder container...")
	if err := containers.Launch(containers.Local(), Coder); err != nil {
		log.Fatalf("failed to start coder service: %v", err)
	}

	log.Println("Coder service started successfully")
}

// CoderExec executes a shell command inside the VS Code server container.
// Used for Git operations, file management, and system commands.
// Commands are run with bash -c for proper shell expansion.
//
// Parameters:
//   - command: Shell command to execute
//
// Returns:
//   - Command output (stdout and stderr combined)
//   - Error if container not running or command fails
func CoderExec(command string) (string, error) {
	if Coder == nil {
		return "", fmt.Errorf("coder service not initialized")
	}

	if !Coder.IsRunning() {
		return "", fmt.Errorf("coder service not running")
	}

	// Use containers.Service ExecInContainerWithOutput method
	return Coder.ExecInContainerWithOutput("/bin/bash", "-c", command)
}

// CoderProxy returns an HTTP reverse proxy to the VS Code server.
// Forwards requests from /coder/* to the code-server on port 8080.
// Used to expose VS Code through the workbench with authentication.
// Returns error handler if service is not initialized.
func CoderProxy() http.Handler {
	if Coder == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Coder service not initialized", http.StatusServiceUnavailable)
		})
	}

	// Use container's proxy method
	return Coder.Proxy(8080)
}

// CoderRestart performs a graceful restart of the VS Code server container.
// Stops the container (if running) and starts it again.
// Used when configuration changes or to recover from issues.
// Logs all operations for debugging.
func CoderRestart() error {
	if Coder == nil {
		return fmt.Errorf("coder service not initialized")
	}

	log.Println("Restarting Coder service...")
	if err := Coder.Stop(); err != nil {
		log.Printf("Error stopping coder service: %v", err)
	}

	return Coder.Start()
}
