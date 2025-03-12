package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/prism/gem/config"
	"github.com/prism/gem/core"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// Global process manager
	processManager *core.ProcessManager

	// Global flags
	configDir string
	verbose   bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gem",
	Short: "Gem - A lightweight process manager",
	Long: `Gem is a lightweight, fast process manager for Linux/Ubuntu systems.
It allows you to manage processes, view tasks, access shells, automate scripts,
and view logs with ease.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Set log level
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		}

		// Initialize process manager
		processManager = core.NewProcessManager(
			config.GlobalConfig.ProcessesPath,
			config.GlobalConfig.LogsPath,
		)

		// Load running processes
		if err := processManager.LoadRunningProcesses(); err != nil {
			logrus.Warnf("Failed to load running processes: %v", err)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting user home directory: %v\n", err)
		os.Exit(1)
	}

	// Set default config directory
	defaultConfigDir := filepath.Join(homeDir, ".gem")

	// Add persistent flags
	rootCmd.PersistentFlags().StringVar(&configDir, "config-dir", defaultConfigDir, "config directory")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Add commands
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(apiCmd)
}
