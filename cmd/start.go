package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/prism/gem/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// Start command flags
	startCmd = &cobra.Command{
		Use:   "start [process-name]",
		Short: "Start a process",
		Long:  `Start a new process or load from a .gem configuration file.`,
		Run:   runStart,
	}

	// Command flags
	cmdFlag        string
	argsFlag       []string
	cwdFlag        string
	envFlag        []string
	restartFlag    string
	maxRestartsFlag int
	configFileFlag string
	clusterFlag    int
	clusterModeFlag string
	autoStartFlag  bool
	userFlag       string
	groupFlag      string
)

func init() {
	startCmd.Flags().StringVarP(&cmdFlag, "cmd", "c", "", "command to run")
	startCmd.Flags().StringSliceVarP(&argsFlag, "args", "a", nil, "command arguments")
	startCmd.Flags().StringVarP(&cwdFlag, "cwd", "d", "", "working directory")
	startCmd.Flags().StringSliceVarP(&envFlag, "env", "e", nil, "environment variables (KEY=VALUE)")
	startCmd.Flags().StringVarP(&restartFlag, "restart", "r", "on-failure", "restart policy (always, on-failure, no)")
	startCmd.Flags().IntVarP(&maxRestartsFlag, "max-restarts", "m", 10, "maximum number of restarts")
	startCmd.Flags().StringVarP(&configFileFlag, "file", "f", "", "configuration file (.gem)")
	startCmd.Flags().IntVarP(&clusterFlag, "cluster", "n", 0, "number of instances to run in cluster mode")
	startCmd.Flags().StringVar(&clusterModeFlag, "cluster-mode", "fork", "cluster mode (fork, cluster)")
	startCmd.Flags().BoolVar(&autoStartFlag, "autostart", false, "automatically start on daemon startup")
	startCmd.Flags().StringVar(&userFlag, "user", "", "user to run the process as")
	startCmd.Flags().StringVar(&groupFlag, "group", "", "group to run the process as")
}

func runStart(cmd *cobra.Command, args []string) {
	var procConfig *config.ProcessConfig

	// Check if we're loading from a config file
	if configFileFlag != "" {
		var err error
		procConfig, err = config.LoadProcessConfig(configFileFlag)
		if err != nil {
			logrus.Fatalf("Failed to load configuration file: %v", err)
		}
	} else {
		// Check if we have a process name
		if len(args) == 0 {
			logrus.Fatal("Process name is required")
		}

		// Check if we have a command
		if cmdFlag == "" {
			logrus.Fatal("Command is required")
		}

		// Create process config
		procConfig = &config.ProcessConfig{
			Name:        args[0],
			Command:     cmdFlag,
			Args:        argsFlag,
			WorkingDir:  cwdFlag,
			Restart:     restartFlag,
			MaxRestarts: maxRestartsFlag,
			AutoStart:   autoStartFlag,
			User:        userFlag,
			Group:       groupFlag,
		}

		// Parse environment variables
		if len(envFlag) > 0 {
			procConfig.Environment = make(map[string]string)
			for _, env := range envFlag {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) != 2 {
					logrus.Fatalf("Invalid environment variable: %s", env)
				}
				procConfig.Environment[parts[0]] = parts[1]
			}
		}

		// Set up cluster if requested
		if clusterFlag > 0 {
			procConfig.Cluster = config.ClusterConfig{
				Instances: clusterFlag,
				Mode:      clusterModeFlag,
			}
		}
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(config.GlobalConfig.LogsPath, 0755); err != nil {
		logrus.Fatalf("Failed to create log directory: %v", err)
	}

	// Set up logging
	if procConfig.Log.Stdout == "" {
		procConfig.Log.Stdout = filepath.Join(config.GlobalConfig.LogsPath, fmt.Sprintf("%s.out.log", procConfig.Name))
	}
	if procConfig.Log.Stderr == "" {
		procConfig.Log.Stderr = filepath.Join(config.GlobalConfig.LogsPath, fmt.Sprintf("%s.err.log", procConfig.Name))
	}

	// Start the process
	proc, err := processManager.StartProcess(procConfig)
	if err != nil {
		logrus.Fatalf("Failed to start process: %v", err)
	}

	logrus.Infof("Started process %s (PID: %d)", procConfig.Name, proc.PID)
}
