package main

import (
	"log"
	"os"

	"cost-explorer/internal/app"
	"cost-explorer/internal/aws"
	"cost-explorer/internal/types"
)

func main() {
	// Setup logging to file to avoid interfering with TUI
	logFile, err := os.OpenFile("cost-explorer.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file:", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Create AWS client
	client, err := aws.NewClient()
	if err != nil {
		log.Fatalf("Unable to create AWS client: %v", err)
	}

	// Create app state with client
	initialState := &types.AppState{
		Client: client,
	}

	// Create and run the application
	appState := app.CreateApp(initialState)
	if err := appState.App.Run(); err != nil {
		panic(err)
	}
}
