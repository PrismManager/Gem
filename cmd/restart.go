package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// Restart command
	restartCmd = &cobra.Command{
		Use:   "restart [process-name]",
		Short: "Restart a process",
		Long:  `Restart a running process.`,
		Run:   runRestart,
	}
)

func runRestart(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		logrus.Fatal("Process name is required")
	}

	name := args[0]
	if err := processManager.RestartProcess(name); err != nil {
		logrus.Fatalf("Failed to restart process: %v", err)
	}

	logrus.Infof("Process %s restarted", name)
}
