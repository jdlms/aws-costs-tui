package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the SQLite database",
	Long:  "Create and initialize the SQLite database for storing cost explorer data",
	Run: func(cmd *cobra.Command, args []string) {
		initDatabase()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func getConfigDir() (string, error) {
	// Check for XDG_CONFIG_HOME first
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "cost-explorer"), nil
	}

	// Fall back to ~/.config on Unix-like systems
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".config", "cost-explorer"), nil
}

func initDatabase() {
	// Get XDG config directory
	configDir, err := getConfigDir()
	if err != nil {
		log.Fatalf("Failed to get config directory: %v", err)
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}

	dbPath := filepath.Join(configDir, "cost-explorer.db")

	// Check if database already exists
	if _, err := os.Stat(dbPath); err == nil {
		fmt.Printf("Database %s already exists. Do you want to recreate it? (y/N): ", dbPath)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Database initialization cancelled.")
			return
		}
		// Remove existing database
		if err := os.Remove(dbPath); err != nil {
			log.Fatalf("Failed to remove existing database: %v", err)
		}
	}

	// Create new database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test the connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Create basic tables (can be expanded later)
	createTables := `
	CREATE TABLE IF NOT EXISTS cost_data (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service TEXT NOT NULL,
		cost REAL NOT NULL,
		date TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS usage_data (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service TEXT NOT NULL,
		usage_type TEXT NOT NULL,
		amount REAL NOT NULL,
		unit TEXT NOT NULL,
		date TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := db.Exec(createTables); err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	fmt.Printf("Database %s initialized successfully!\n", dbPath)
	fmt.Println("Created tables: cost_data, usage_data")
}
