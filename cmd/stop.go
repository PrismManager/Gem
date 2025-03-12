package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// Stop command flags
	forceFlag bool

	// Stop command
	stopCmd = &cobra.Command{
		Use:   "stop [process-name]",
		Short: "Stop a process",
		Long:  `Stop a running process.`,
		Run:   runStop,
	}
)

func init() {
	stopCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "force stop (SIGKILL)")
}

func runStop(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		logrus.Fatal("Process name is required")
	}

	name := args[0]
	if err := processManager.StopProcess(name, forceFlag); err != nil {
		logrus.Fatalf("Failed to stop process: %v", err)
	}

	logrus.Infof("Process %s stopped", name)
}
