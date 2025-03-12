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
	// List command
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all processes",
		Long:  `List all running processes.`,
		Run:   runList,
	}
)

func runList(cmd *cobra.Command, args []string) {
	processes := processManager.ListProcesses()
	if len(processes) == 0 {
		fmt.Println("No processes running")
		return
	}

	// Create table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "PID", "Status", "CPU", "Memory", "Uptime", "Restarts"})
	table.SetBorder(false)
	table.SetColumnSeparator(" ")

	// Add rows
	for _, proc := range processes {
		// Get process info
		info, err := processManager.GetProcessInfo(proc.Config.Name)
		if err != nil {
			logrus.Warnf("Failed to get process info for %s: %v", proc.Config.Name, err)
			continue
		}

		// Format CPU and memory
		cpu := fmt.Sprintf("%.1f%%", info.CPU)
		mem := fmt.Sprintf("%.1f MB", info.Memory)

		// Add row
		table.Append([]string{
			info.Name,
			strconv.Itoa(int(info.PID)),
			info.Status,
			cpu,
			mem,
			info.Uptime,
			strconv.Itoa(proc.Restarts),
		})
	}

	table.Render()
}
