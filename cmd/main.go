package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
)

func main() {
	// Setup logging to file to avoid interfering with TUI
	logFile, err := os.OpenFile("cost-explorer-tui.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file:", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Unable to load SDK config: %v", err)
	}

	client := costexplorer.NewFromConfig(cfg)

	appState := createApp(client)
	if err := appState.app.Run(); err != nil {
		panic(err)
	}
}
