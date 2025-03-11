package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

var (
	configDir  string
	logsDir    string
	pidDir     string
	scriptsDir string
)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Failed to get user home directory:", err)
	}

	// Setup directories
	configDir = filepath.Join(homeDir, ".gem")
	logsDir = filepath.Join(configDir, "logs")
	pidDir = filepath.Join(configDir, "pids")
	scriptsDir = filepath.Join(configDir, "scripts")

	// Ensure directories exist
	dirs := []string{configDir, logsDir, pidDir, scriptsDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatal("Failed to create directory:", dir, err)
		}
	}
}

func main() {
	app := &cli.App{
		Name:    "gem",
		Usage:   "A process manager for Linux/Ubuntu systems",
		Version: "1.0.0",
		Commands: []*cli.Command{
			{
				Name:    "start",
				Aliases: []string{"s"},
				Usage:   "Start a process",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "name",
						Aliases: []string{"n"},
						Usage:   "Name of the process",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "cmd",
						Aliases: []string{"c"},
						Usage:   "Command to run",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "cwd",
						Usage:   "Working directory",
						Value:   ".",
					},
					&cli.BoolFlag{
						Name:    "restart",
						Aliases: []string{"r"},
						Usage:   "Restart on failure",
						Value:   false,
					},
					&cli.IntFlag{
						Name:    "max-restarts",
						Usage:   "Maximum number of restarts",
						Value:   5,
					},
					&cli.StringSliceFlag{
						Name:    "env",
						Aliases: []string{"e"},
						Usage:   "Environment variables (KEY=VALUE)",
					},
				},
				Action: startProcess,
			},
			{
				Name:    "stop",
				Aliases: []string{"p"},
				Usage:   "Stop a process",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "name",
						Aliases: []string{"n"},
						Usage:   "Name of the process",
						Required: true,
					},
				},
				Action: stopProcess,
			},
			{
				Name:    "restart",
				Aliases: []string{"r"},
				Usage:   "Restart a process",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "name",
						Aliases: []string{"n"},
						Usage:   "Name of the process",
						Required: true,
					},
				},
				Action: restartProcess,
			},
			{
				Name:    "list",
				Aliases: []string{"l", "ls", "status"},
				Usage:   "List all processes",
				Action:  listProcesses,
			},
			{
				Name:  "logs",
				Usage: "View process logs",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "name",
						Aliases: []string{"n"},
						Usage:   "Name of the process",
						Required: true,
					},
					&cli.BoolFlag{
						Name:    "follow",
						Aliases: []string{"f"},
						Usage:   "Follow log output",
					},
					&cli.IntFlag{
						Name:    "lines",
						Aliases: []string{"n"},
						Usage:   "Number of lines to show",
						Value:   100,
					},
				},
				Action: viewLogs,
			},
			{
				Name:  "shell",
				Usage: "Enter shell for a process",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "name",
						Aliases: []string{"n"},
						Usage:   "Name of the process",
						Required: true,
					},
				},
				Action: enterShell,
			},
			{
				Name:  "script",
				Usage: "Manage automation scripts",
				Subcommands: []*cli.Command{
					{
						Name:  "add",
						Usage: "Add a new script",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Aliases:  []string{"n"},
								Usage:    "Name of the script",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "file",
								Aliases:  []string{"f"},
								Usage:    "Path to script file",
								Required: true,
							},
							&cli.StringFlag{
								Name:    "schedule",
								Aliases: []string{"s"},
								Usage:   "Cron schedule expression",
							},
							&cli.StringFlag{
								Name:    "process",
								Aliases: []string{"p"},
								Usage:   "Process to run script against",
							},
						},
						Action: addScript,
					},
					{
						Name:  "list",
						Usage: "List all scripts",
						Action: listScripts,
					},
					{
						Name:  "run",
						Usage: "Run a script",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Aliases:  []string{"n"},
								Usage:    "Name of the script",
								Required: true,
							},
						},
						Action: runScript,
					},
					{
						Name:  "remove",
						Usage: "Remove a script",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Aliases:  []string{"n"},
								Usage:    "Name of the script",
								Required: true,
							},
						},
						Action: removeScript,
					},
				},
			},
			{
				Name:  "serve",
				Usage: "Start the Gem API server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "port",
						Aliases: []string{"p"},
						Usage:   "Port to listen on",
						Value:   "8080",
					},
				},
				Action: startAPIServer,
			},
			{
				Name:   "install",
				Usage:  "Install Gem as a system service",
				Action: installService,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func startProcess(c *cli.Context) error {
	name := c.String("name")
	cmd := c.String("cmd")
	cwd := c.String("cwd")
	restart := c.Bool("restart")
	maxRestarts := c.Int("max-restarts")
	env := c.StringSlice("env")

	service := NewProcessService()
	process, err := service.Start(name, cmd, cwd, restart, maxRestarts, env)
	if err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	fmt.Printf("Started process %s (PID: %d)\n", process.Name, process.Pid)
	return nil
}

func stopProcess(c *cli.Context) error {
	name := c.String("name")

	service := NewProcessService()
	if err := service.Stop(name); err != nil {
		return fmt.Errorf("failed to stop process: %w", err)
	}

	fmt.Printf("Stopped process %s\n", name)
	return nil
}

func restartProcess(c *cli.Context) error {
	name := c.String("name")

	service := NewProcessService()
	process, err := service.Restart(name)
	if err != nil {
		return fmt.Errorf("failed to restart process: %w", err)
	}

	fmt.Printf("Restarted process %s (PID: %d)\n", process.Name, process.Pid)
	return nil
}

func listProcesses(c *cli.Context) error {
	service := NewProcessService()
	processes, err := service.List()
	if err != nil {
		return fmt.Errorf("failed to list processes: %w", err)
	}

	if len(processes) == 0 {
		fmt.Println("No processes running")
		return nil
	}

	fmt.Println("Running processes:")
	fmt.Printf("%-20s %-10s %-20s %-10s %-15s\n", "NAME", "PID", "STATUS", "RESTARTS", "UPTIME")
	fmt.Println(strings.Repeat("-", 80))

	for _, p := range processes {
		uptime := time.Since(p.StartTime).Round(time.Second)
		fmt.Printf("%-20s %-10d %-20s %-10d %-15s\n", 
			p.Name, p.Pid, p.Status, p.Restarts, uptime)
	}

	return nil
}

func viewLogs(c *cli.Context) error {
	name := c.String("name")
	follow := c.Bool("follow")
	lines := c.Int("lines")

	service := NewProcessService()
	if err := service.ShowLogs(name, follow, lines); err != nil {
		return fmt.Errorf("failed to view logs: %w", err)
	}

	return nil
}

func enterShell(c *cli.Context) error {
	name := c.String("name")

	service := NewProcessService()
	if err := service.EnterShell(name); err != nil {
		return fmt.Errorf("failed to enter shell: %w", err)
	}

	return nil
}

func addScript(c *cli.Context) error {
	name := c.String("name")
	file := c.String("file")
	schedule := c.String("schedule")
	process := c.String("process")

	service := NewScriptService()
	if err := service.Add(name, file, schedule, process); err != nil {
		return fmt.Errorf("failed to add script: %w", err)
	}

	fmt.Printf("Added script %s\n", name)
	return nil
}

func listScripts(c *cli.Context) error {
	service := NewScriptService()
	scripts, err := service.List()
	if err != nil {
		return fmt.Errorf("failed to list scripts: %w", err)
	}

	if len(scripts) == 0 {
		fmt.Println("No scripts configured")
		return nil
	}

	fmt.Println("Configured scripts:")
	fmt.Printf("%-20s %-30s %-20s %-20s\n", "NAME", "FILE", "SCHEDULE", "PROCESS")
	fmt.Println(strings.Repeat("-", 90))

	for _, s := range scripts {
		fmt.Printf("%-20s %-30s %-20s %-20s\n", 
			s.Name, s.File, s.Schedule, s.Process)
	}

	return nil
}

func runScript(c *cli.Context) error {
	name := c.String("name")

	service := NewScriptService()
	if err := service.Run(name); err != nil {
		return fmt.Errorf("failed to run script: %w", err)
	}

	fmt.Printf("Executed script %s\n", name)
	return nil
}

func removeScript(c *cli.Context) error {
	name := c.String("name")

	service := NewScriptService()
	if err := service.Remove(name); err != nil {
		return fmt.Errorf("failed to remove script: %w", err)
	}

	fmt.Printf("Removed script %s\n", name)
	return nil
}

func startAPIServer(c *cli.Context) error {
	port := c.String("port")

	server := NewAPIServer(port)
	fmt.Printf("Starting API server on port %s\n", port)
	return server.Start()
}

func installService(c *cli.Context) error {
	service := NewSystemService()
	if err := service.Install(); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	fmt.Println("Installed Gem as a system service")
	return nil
}
