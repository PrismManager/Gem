package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/prism/gem/config"
	"github.com/stretchr/testify/assert"
)

func TestNewProcessManager(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "gem-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	processesPath := filepath.Join(tempDir, "processes")
	logsPath := filepath.Join(tempDir, "logs")

	// Create process manager
	pm := NewProcessManager(processesPath, logsPath)
	assert.NotNil(t, pm)
	assert.Equal(t, processesPath, pm.processesPath)
	assert.Equal(t, logsPath, pm.logsPath)
	assert.Empty(t, pm.processes)
}

func TestStartStopProcess(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "gem-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	processesPath := filepath.Join(tempDir, "processes")
	logsPath := filepath.Join(tempDir, "logs")

	// Create process manager
	pm := NewProcessManager(processesPath, logsPath)

	// Create process config
	procConfig := &config.ProcessConfig{
		Name:    "test-process",
		Command: "sleep",
		Args:    []string{"10"},
	}

	// Start process
	proc, err := pm.StartProcess(procConfig)
	assert.NoError(t, err)
	assert.NotNil(t, proc)
	assert.Equal(t, "running", proc.Status)
	assert.Greater(t, proc.PID, 0)

	// Check if process is running
	assert.True(t, proc.Cmd.Process != nil)

	// Get process
	retrievedProc, err := pm.GetProcess("test-process")
	assert.NoError(t, err)
	assert.Equal(t, proc, retrievedProc)

	// Stop process
	err = pm.StopProcess("test-process", false)
	assert.NoError(t, err)

	// Wait for process to stop
	time.Sleep(1 * time.Second)

	// Check if process is removed from map
	_, err = pm.GetProcess("test-process")
	assert.Error(t, err)
}

func TestListProcesses(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "gem-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	processesPath := filepath.Join(tempDir, "processes")
	logsPath := filepath.Join(tempDir, "logs")

	// Create process manager
	pm := NewProcessManager(processesPath, logsPath)

	// No processes initially
	processes := pm.ListProcesses()
	assert.Empty(t, processes)

	// Start a process
	procConfig := &config.ProcessConfig{
		Name:    "test-process",
		Command: "sleep",
		Args:    []string{"10"},
	}

	_, err = pm.StartProcess(procConfig)
	assert.NoError(t, err)

	// Check if process is listed
	processes = pm.ListProcesses()
	assert.Len(t, processes, 1)
	assert.Equal(t, "test-process", processes[0].Config.Name)

	// Clean up
	err = pm.StopProcess("test-process", true)
	assert.NoError(t, err)
}

func TestClusterProcess(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "gem-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	processesPath := filepath.Join(tempDir, "processes")
	logsPath := filepath.Join(tempDir, "logs")

	// Create process manager
	pm := NewProcessManager(processesPath, logsPath)

	// Create process config with cluster
	procConfig := &config.ProcessConfig{
		Name:    "test-cluster",
		Command: "sleep",
		Args:    []string{"10"},
		Cluster: config.ClusterConfig{
			Instances: 2,
			Mode:      "fork",
		},
	}

	// Start cluster
	proc, err := pm.StartProcess(procConfig)
	assert.NoError(t, err)
	assert.NotNil(t, proc)
	assert.Equal(t, "running", proc.Status)
	assert.Len(t, proc.ClusterProcs, 2)

	// Get cluster info
	info, err := pm.GetProcessInfo("test-cluster")
	assert.NoError(t, err)
	assert.Equal(t, 2, info.Instances)

	// Stop cluster
	err = pm.StopProcess("test-cluster", true)
	assert.NoError(t, err)

	// Wait for processes to stop
	time.Sleep(1 * time.Second)

	// Check if cluster is removed from map
	_, err = pm.GetProcess("test-cluster")
	assert.Error(t, err)
}
