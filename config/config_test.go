package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "gem-config-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Load config
	err = LoadConfig(tempDir)
	assert.NoError(t, err)

	// Check default values
	assert.Equal(t, "info", GlobalConfig.LogLevel)
	assert.Equal(t, 3456, GlobalConfig.APIPort)
	assert.Equal(t, filepath.Join(tempDir, "gem.sock"), GlobalConfig.SocketPath)
	assert.Equal(t, filepath.Join(tempDir, "processes"), GlobalConfig.ProcessesPath)
	assert.Equal(t, filepath.Join(tempDir, "logs"), GlobalConfig.LogsPath)
	assert.False(t, GlobalConfig.ClusterMode)
	assert.Empty(t, GlobalConfig.ClusterNodes)

	// Check if config file was created
	configFile := filepath.Join(tempDir, "config.yaml")
	_, err = os.Stat(configFile)
	assert.NoError(t, err)

	// Check if directories were created
	_, err = os.Stat(GlobalConfig.ProcessesPath)
	assert.NoError(t, err)
	_, err = os.Stat(GlobalConfig.LogsPath)
	assert.NoError(t, err)
}

func TestLoadProcessConfig(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "gem-process-config-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test config file
	configFile := filepath.Join(tempDir, "test.gem")
	content := `
name: test-process
cmd: echo
args:
  - "hello"
  - "world"
cwd: /tmp
env:
  TEST_VAR: "test"
restart: always
max_restarts: 5
restart_delay: 2
cluster:
  instances: 3
  mode: fork
log:
  stdout: ./logs/test.out.log
  stderr: ./logs/test.err.log
  rotate: true
  max_size: 10M
  max_files: 3
autostart: true
user: nobody
scripts:
  pre_start: echo "Starting..."
  post_start: echo "Started!"
  pre_stop: echo "Stopping..."
  post_stop: echo "Stopped!"
`
	err = os.WriteFile(configFile, []byte(content), 0644)
	assert.NoError(t, err)

	// Load process config
	procConfig, err := LoadProcessConfig(configFile)
	assert.NoError(t, err)
	assert.NotNil(t, procConfig)

	// Check values
	assert.Equal(t, "test-process", procConfig.Name)
	assert.Equal(t, "echo", procConfig.Command)
	assert.Equal(t, []string{"hello", "world"}, procConfig.Args)
	assert.Equal(t, "/tmp", procConfig.WorkingDir)
	assert.Equal(t, "test", procConfig.Environment["TEST_VAR"])
	assert.Equal(t, "always", procConfig.Restart)
	assert.Equal(t, 5, procConfig.MaxRestarts)
	assert.Equal(t, 2, procConfig.RestartDelay)
	assert.Equal(t, 3, procConfig.Cluster.Instances)
	assert.Equal(t, "fork", procConfig.Cluster.Mode)
	assert.Equal(t, "./logs/test.out.log", procConfig.Log.Stdout)
	assert.Equal(t, "./logs/test.err.log", procConfig.Log.Stderr)
	assert.True(t, procConfig.Log.Rotate)
	assert.Equal(t, "10M", procConfig.Log.MaxSize)
	assert.Equal(t, 3, procConfig.Log.MaxFiles)
	assert.True(t, procConfig.AutoStart)
	assert.Equal(t, "nobody", procConfig.User)
	assert.Equal(t, "echo \"Starting...\"", procConfig.Scripts.PreStart)
	assert.Equal(t, "echo \"Started!\"", procConfig.Scripts.PostStart)
	assert.Equal(t, "echo \"Stopping...\"", procConfig.Scripts.PreStop)
	assert.Equal(t, "echo \"Stopped!\"", procConfig.Scripts.PostStop)
}

func TestProcessConfigDefaults(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "gem-process-defaults-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create minimal config file
	configFile := filepath.Join(tempDir, "minimal.gem")
	content := `
name: minimal
cmd: echo
`
	err = os.WriteFile(configFile, []byte(content), 0644)
	assert.NoError(t, err)

	// Load process config
	procConfig, err := LoadProcessConfig(configFile)
	assert.NoError(t, err)
	assert.NotNil(t, procConfig)

	// Check default values
	assert.Equal(t, "minimal", procConfig.Name)
	assert.Equal(t, "echo", procConfig.Command)
	assert.Equal(t, "on-failure", procConfig.Restart)
	assert.Equal(t, 10, procConfig.MaxRestarts)
	assert.Equal(t, 3, procConfig.RestartDelay)
}
