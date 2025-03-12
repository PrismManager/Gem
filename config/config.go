package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Config holds the global configuration for Gem
type Config struct {
	LogLevel      string `mapstructure:"log_level"`
	APIPort       int    `mapstructure:"api_port"`
	SocketPath    string `mapstructure:"socket_path"`
	ProcessesPath string `mapstructure:"processes_path"`
	LogsPath      string `mapstructure:"logs_path"`
	ClusterMode   bool   `mapstructure:"cluster_mode"`
	ClusterNodes  []string `mapstructure:"cluster_nodes"`
}

// Global configuration instance
var GlobalConfig Config

// LoadConfig loads the configuration from the specified directory
func LoadConfig(configDir string) error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	// Set default values
	viper.SetDefault("log_level", "info")
	viper.SetDefault("api_port", 3456)
	viper.SetDefault("socket_path", filepath.Join(configDir, "gem.sock"))
	viper.SetDefault("processes_path", filepath.Join(configDir, "processes"))
	viper.SetDefault("logs_path", filepath.Join(configDir, "logs"))
	viper.SetDefault("cluster_mode", false)
	viper.SetDefault("cluster_nodes", []string{})

	// Create config file if it doesn't exist
	configFile := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return err
		}
		if err := viper.SafeWriteConfigAs(configFile); err != nil {
			return err
		}
		logrus.Infof("Created default configuration file at %s", configFile)
	}

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	// Unmarshal the config into GlobalConfig
	if err := viper.Unmarshal(&GlobalConfig); err != nil {
		return err
	}

	// Create necessary directories
	dirs := []string{
		GlobalConfig.ProcessesPath,
		GlobalConfig.LogsPath,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// ProcessConfig represents the configuration for a process
type ProcessConfig struct {
	Name         string            `yaml:"name" json:"name"`
	Command      string            `yaml:"cmd" json:"cmd"`
	Args         []string          `yaml:"args,omitempty" json:"args,omitempty"`
	WorkingDir   string            `yaml:"cwd,omitempty" json:"cwd,omitempty"`
	Environment  map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Restart      string            `yaml:"restart,omitempty" json:"restart,omitempty"` // "always", "on-failure", "no"
	MaxRestarts  int               `yaml:"max_restarts,omitempty" json:"max_restarts,omitempty"`
	RestartDelay int               `yaml:"restart_delay,omitempty" json:"restart_delay,omitempty"` // in seconds
	Cluster      ClusterConfig     `yaml:"cluster,omitempty" json:"cluster,omitempty"`
	Log          LogConfig         `yaml:"log,omitempty" json:"log,omitempty"`
	AutoStart    bool              `yaml:"autostart,omitempty" json:"autostart,omitempty"`
	User         string            `yaml:"user,omitempty" json:"user,omitempty"`
	Group        string            `yaml:"group,omitempty" json:"group,omitempty"`
	Scripts      ScriptsConfig     `yaml:"scripts,omitempty" json:"scripts,omitempty"`
}

// ClusterConfig represents cluster configuration for a process
type ClusterConfig struct {
	Instances int    `yaml:"instances,omitempty" json:"instances,omitempty"`
	Mode      string `yaml:"mode,omitempty" json:"mode,omitempty"` // "fork" or "cluster"
}

// LogConfig represents logging configuration for a process
type LogConfig struct {
	Stdout   string `yaml:"stdout,omitempty" json:"stdout,omitempty"`
	Stderr   string `yaml:"stderr,omitempty" json:"stderr,omitempty"`
	Rotate   bool   `yaml:"rotate,omitempty" json:"rotate,omitempty"`
	MaxSize  string `yaml:"max_size,omitempty" json:"max_size,omitempty"`
	MaxFiles int    `yaml:"max_files,omitempty" json:"max_files,omitempty"`
}

// ScriptsConfig represents scripts configuration for a process
type ScriptsConfig struct {
	PreStart  string `yaml:"pre_start,omitempty" json:"pre_start,omitempty"`
	PostStart string `yaml:"post_start,omitempty" json:"post_start,omitempty"`
	PreStop   string `yaml:"pre_stop,omitempty" json:"pre_stop,omitempty"`
	PostStop  string `yaml:"post_stop,omitempty" json:"post_stop,omitempty"`
}

// LoadProcessConfig loads a process configuration from a .gem file
func LoadProcessConfig(filePath string) (*ProcessConfig, error) {
	v := viper.New()
	v.SetConfigFile(filePath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("unsupported config type: %s", filepath.Ext(filePath))
		}
		return nil, err
	}

	var config ProcessConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Set default values if not provided
	if config.Restart == "" {
		config.Restart = "on-failure"
	}
	if config.MaxRestarts == 0 {
		config.MaxRestarts = 10
	}
	if config.RestartDelay == 0 {
		config.RestartDelay = 3
	}

	return &config, nil
}
