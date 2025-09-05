package services

import (
	"fmt"
	"log"
	"net/http"

	"github.com/The-Skyscape/devtools/pkg/database"

	"github.com/The-Skyscape/devtools/pkg/containers"
)

var Coder = &containers.Service{
	Host:          containers.Local(),
	Name:          "workbench-coder",
	Image:         "codercom/code-server:latest",
	Command:       "--auth none --bind-addr 0.0.0.0:8080",
	Network:       "host",
	RestartPolicy: "always",
	Mounts: map[string]string{
		"/home/.ssh": "/home/.ssh",
		fmt.Sprintf("%s/coder", database.DataDir()):         "/home/coder",
		fmt.Sprintf("%s/coder/.config", database.DataDir()): "/home/coder/.config",
	},
}

// Init initializes and starts the coder service
func init() {
	log.Println("Initializing Coder service...")

	// Check if container already exists
	existing := containers.Local().Service("workbench-coder")
	if existing != nil && existing.IsRunning() {
		log.Println("Coder service already running")
		Coder = existing
		return
	}

	prepareScript := fmt.Sprintf(`
		mkdir -p %[1]s/coder
		mkdir -p %[1]s/coder/.config
		mkdir -p %[1]s/coder/repos
		chmod -R 777 %[1]s/coder
		chown -R 1000:1000 %[1]s/coder || true
	`, database.DataDir())

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

// Execute runs a command in the coder container
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

// Proxy returns HTTP proxy to coder service
func CoderProxy() http.Handler {
	if Coder == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Coder service not initialized", http.StatusServiceUnavailable)
		})
	}

	// Use container's proxy method
	return Coder.Proxy(8080)
}

// Restart restarts the coder service
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
