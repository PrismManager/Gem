package cmd

import (
	"github.com/prism/gem/api"
	"github.com/prism/gem/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// API command flags
	apiPortFlag int

	// API command
	apiCmd = &cobra.Command{
		Use:   "api [start|stop]",
		Short: "Manage API server",
		Long:  `Start or stop the API server.`,
		Run:   runAPI,
	}
)

func init() {
	apiCmd.Flags().IntVarP(&apiPortFlag, "port", "p", 0, "API server port (default: from config)")
}

func runAPI(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		logrus.Fatal("Action is required (start or stop)")
	}

	action := args[0]

	switch action {
	case "start":
		// Get port from flag or config
		port := apiPortFlag
		if port == 0 {
			port = config.GlobalConfig.APIPort
		}

		// Create API server
		server := api.NewAPIServer(processManager)

		// Start API server
		logrus.Infof("Starting API server on port %d", port)
		if err := server.Start(port); err != nil {
			logrus.Fatalf("Failed to start API server: %v", err)
		}
	case "stop":
		logrus.Fatal("API server stop not implemented yet")
	default:
		logrus.Fatalf("Invalid action: %s", action)
	}
}
