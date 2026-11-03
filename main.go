package main

import (
	"embed"

	"log"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Resolve the user's home once, here at the app edge; the skills package
	// takes it as an argument so it stays testable.
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	app := application.New(application.Options{
		Name:        "agent-skills-manager",
		Description: "Read-only inventory of coding-agent skills",
		Services: []application.Service{
			application.NewService(NewInventoryService(home)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Agent Skills",
		// Wide enough for the matrix: skill, status, and six root columns.
		Width:  1280,
		Height: 800,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(6, 7, 15),
		URL:              "/",
	})

	// Run the application. This blocks until the application has been exited.
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
