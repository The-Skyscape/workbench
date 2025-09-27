package main

import (
	"embed"

	"github.com/The-Skyscape/devtools/pkg/application"

	"workbench/controllers"
)

//go:embed all:views
var views embed.FS

func main() {
	// Start application
	application.Serve(views,
		application.WithDaisyTheme("dark"),
		application.WithController(controllers.Auth()),
		application.WithController(controllers.Workbench()),
		application.WithController(controllers.Monitoring()),
	)
}
