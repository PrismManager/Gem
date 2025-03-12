package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/prism/gem/cmd"
	"github.com/prism/gem/config"
	"github.com/prism/gem/utils"
)

func main() {
	// Initialize configuration
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting user home directory: %v\n", err)
		os.Exit(1)
	}

	// Create Gem directory if it doesn't exist
	gemDir := filepath.Join(homeDir, ".gem")
	if err := os.MkdirAll(gemDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Gem directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := utils.InitLogger(filepath.Join(gemDir, "gem.log")); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing logger: %v\n", err)
		os.Exit(1)
	}

	// Load configuration
	if err := config.LoadConfig(gemDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Execute root command
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
