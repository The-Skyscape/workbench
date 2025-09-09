package main

import (
	"embed"
	"net/http"

	"github.com/The-Skyscape/devtools/pkg/application"

	"workbench/controllers"
	_ "workbench/internal/commander" // Initialize Commander client
)

//go:embed all:views
var views embed.FS

func main() {
	// Health check endpoint
	http.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"healthy"}`))
	})

	// Start application
	application.Serve(views,
		application.WithDaisyTheme("dark"),
		application.WithController(controllers.Auth()),
		application.WithController(controllers.Workbench()),
		application.WithController(controllers.Monitoring()),
	)
}
