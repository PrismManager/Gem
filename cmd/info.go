package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// Info command
	infoCmd = &cobra.Command{
		Use:   "info [process-name]",
		Short: "Show process information",
		Long:  `Show detailed information about a process.`,
		Run:   runInfo,
	}
)

func runInfo(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		logrus.Fatal("Process name is required")
	}

	name := args[0]
	proc, err := processManager.GetProcess(name)
	if err != nil {
		logrus.Fatalf("Failed to get process: %v", err)
	}

	// Get process info
	info, err := processManager.GetProcessInfo(name)
	if err != nil {
		logrus.Fatalf("Failed to get process info: %v", err)
	}

	// Print process information
	fmt.Printf("Process: %s\n", info.Name)
	fmt.Printf("Status: %s\n", info.Status)
	fmt.Printf("PID: %d\n", info.PID)
	fmt.Printf("CPU: %.1f%%\n", info.CPU)
	fmt.Printf("Memory: %.1f MB\n", info.Memory)
	fmt.Printf("Uptime: %s\n", info.Uptime)
	fmt.Printf("Restarts: %d\n", proc.Restarts)
	fmt.Printf("Command: %s\n", info.Command)
	fmt.Printf("User: %s\n", info.User)

	// Print environment variables
	if len(proc.Config.Environment) > 0 {
		fmt.Println("\nEnvironment Variables:")
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Key", "Value"})
		table.SetBorder(false)
		table.SetColumnSeparator(" ")

		for k, v := range proc.Config.Environment {
			table.Append([]string{k, v})
		}

		table.Render()
	}

	// Print cluster information
	if info.Instances > 0 {
		fmt.Printf("\nCluster Mode: %s\n", proc.Config.Cluster.Mode)
		fmt.Printf("Instances: %d\n", info.Instances)

		// Print worker processes
		fmt.Println("\nWorker Processes:")
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "PID", "Status", "CPU", "Memory", "Uptime"})
		table.SetBorder(false)
		table.SetColumnSeparator(" ")

		for _, worker := range proc.ClusterProcs {
			workerInfo, err := processManager.GetProcessInfo(worker.Config.Name)
			if err != nil {
				continue
			}

			table.Append([]string{
				workerInfo.Name,
				strconv.Itoa(int(workerInfo.PID)),
				workerInfo.Status,
				fmt.Sprintf("%.1f%%", workerInfo.CPU),
				fmt.Sprintf("%.1f MB", workerInfo.Memory),
				workerInfo.Uptime,
			})
		}

		table.Render()
	}
}

