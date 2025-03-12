package cmd

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// Logs command flags
	linesFlag  int
	streamFlag string
	followFlag bool

	// Logs command
	logsCmd = &cobra.Command{
		Use:   "logs [process-name]",
		Short: "View process logs",
		Long:  `View logs for a process.`,
		Run:   runLogs,
	}
)

func init() {
	logsCmd.Flags().IntVarP(&linesFlag, "lines", "n", 100, "number of lines to show")
	logsCmd.Flags().StringVarP(&streamFlag, "stream", "s", "stdout", "log stream (stdout, stderr)")
	logsCmd.Flags().BoolVarP(&followFlag, "follow", "f", false, "follow log output")
}

func runLogs(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		logrus.Fatal("Process name is required")
	}

	name := args[0]

	// Validate stream
	if streamFlag != "stdout" && streamFlag != "stderr" {
		logrus.Fatal("Invalid stream, must be stdout or stderr")
	}

	// Get logs
	logs, err := processManager.GetLogs(name, streamFlag, linesFlag)
	if err != nil {
		logrus.Fatalf("Failed to get logs: %v", err)
	}

	// Print logs
	for _, line := range logs {
		fmt.Println(line)
	}

	// Follow logs if requested
	if followFlag {
		// TODO: Implement log following
		logrus.Warn("Log following not implemented yet")
	}
}
