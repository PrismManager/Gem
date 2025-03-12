package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// ProcessInfo represents information about a running process
type ProcessInfo struct {
	PID        int32     `json:"pid"`
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	CPU        float64   `json:"cpu"`
	Memory     float64   `json:"memory"`
	StartTime  time.Time `json:"start_time"`
	Uptime     string    `json:"uptime"`
	Command    string    `json:"command"`
	Restarts   int       `json:"restarts"`
	User       string    `json:"user"`
	ClusterID  int       `json:"cluster_id,omitempty"`
	Instances  int       `json:"instances,omitempty"`
}

// GetProcessInfo retrieves information about a process by PID
func GetProcessInfo(pid int32) (*ProcessInfo, error) {
	proc, err := process.NewProcess(pid)
	if err != nil {
		return nil, err
	}

	name, err := proc.Name()
	if err != nil {
		name = "unknown"
	}

	status, err := proc.Status()
	if err != nil {
		status = []string{"unknown"}
	}

	cpuPercent, err := proc.CPUPercent()
	if err != nil {
		cpuPercent = 0
	}

	memInfo, err := proc.MemoryInfo()
	var memPercent float64
	if err != nil || memInfo == nil {
		memPercent = 0
	} else {
		memPercent = float64(memInfo.RSS) / (1024 * 1024) // Convert to MB
	}

	createTime, err := proc.CreateTime()
	startTime := time.Now()
	if err == nil {
		startTime = time.Unix(createTime/1000, 0)
	}

	uptime := time.Since(startTime).Round(time.Second).String()

	cmdline, err := proc.Cmdline()
	if err != nil {
		cmdline = "unknown"
	}

	username, err := proc.Username()
	if err != nil {
		username = "unknown"
	}

	return &ProcessInfo{
		PID:       pid,
		Name:      name,
		Status:    status[0],
		CPU:       cpuPercent,
		Memory:    memPercent,
		StartTime: startTime,
		Uptime:    uptime,
		Command:   cmdline,
		User:      username,
	}, nil
}

// IsProcessRunning checks if a process with the given PID is running
func IsProcessRunning(pid int32) bool {
	_, err := process.NewProcess(pid)
	return err == nil
}

// WritePIDFile writes a PID to a file
func WritePIDFile(pid int, name string, processesDir string) error {
	pidFile := filepath.Join(processesDir, fmt.Sprintf("%s.pid", name))
	return os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
}

// ReadPIDFile reads a PID from a file
func ReadPIDFile(name string, processesDir string) (int32, error) {
	pidFile := filepath.Join(processesDir, fmt.Sprintf("%s.pid", name))
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}

	return int32(pid), nil
}

// DeletePIDFile deletes a PID file
func DeletePIDFile(name string, processesDir string) error {
	pidFile := filepath.Join(processesDir, fmt.Sprintf("%s.pid", name))
	return os.Remove(pidFile)
}

// GetRunningProcesses returns a list of all running processes managed by Gem
func GetRunningProcesses(processesDir string) (map[string]int32, error) {
	processes := make(map[string]int32)

	files, err := os.ReadDir(processesDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".pid") {
			name := strings.TrimSuffix(file.Name(), ".pid")
			pid, err := ReadPIDFile(name, processesDir)
			if err != nil {
				continue
			}

			if IsProcessRunning(pid) {
				processes[name] = pid
			} else {
				// Clean up stale PID file
				DeletePIDFile(name, processesDir)
			}
		}
	}

	return processes, nil
}

