package cmd

import (
	"log"
	"os"

	"cost-explorer/internal/app"
	"cost-explorer/internal/aws"
	"cost-explorer/internal/types"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cost-explorer",
	Short: "AWS Cost Explorer TUI application",
	Long:  "A terminal user interface for exploring AWS costs and usage data",
	Run: func(cmd *cobra.Command, args []string) {
		// This is the default behavior - start the TUI
		startTUI()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func startTUI() {
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
