package core

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/prism/gem/config"
	"github.com/prism/gem/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// ProcessManager handles process lifecycle management
type ProcessManager struct {
	processes     map[string]*ManagedProcess
	processesPath string
	logsPath      string
	mutex         sync.RWMutex
}

// ManagedProcess represents a process managed by Gem
type ManagedProcess struct {
	Config       *config.ProcessConfig
	Cmd          *exec.Cmd
	PID          int
	Status       string // "running", "stopped", "restarting", "failed"
	StartTime    time.Time
	Restarts     int
	LogFiles     map[string]*os.File
	ClusterProcs []*ManagedProcess // For cluster mode
	PTY          *os.File          // For interactive shell
	mu           sync.RWMutex
}

// NewProcessManager creates a new process manager
func NewProcessManager(processesPath, logsPath string) *ProcessManager {
	return &ProcessManager{
		processes:     make(map[string]*ManagedProcess),
		processesPath: processesPath,
		logsPath:      logsPath,
		mutex:         sync.RWMutex{},
	}
}

// LoadRunningProcesses loads all running processes from PID files
func (pm *ProcessManager) LoadRunningProcesses() error {
	runningProcesses, err := utils.GetRunningProcesses(pm.processesPath)
	if err != nil {
		return err
	}

	for name, pid := range runningProcesses {
		configPath := filepath.Join(pm.processesPath, fmt.Sprintf("%s.gem", name))
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			logrus.Warnf("Process %s is running but no config file found", name)
			continue
		}

		procConfig, err := config.LoadProcessConfig(configPath)
		if err != nil {
			logrus.Warnf("Failed to load config for process %s: %v", name, err)
			continue
		}

		proc := &ManagedProcess{
			Config:    procConfig,
			PID:       int(pid),
			Status:    "running",
			StartTime: time.Now(), // Approximate
			LogFiles:  make(map[string]*os.File),
		}

		pm.mutex.Lock()
		pm.processes[name] = proc
		pm.mutex.Unlock()

		logrus.Infof("Loaded running process: %s (PID: %d)", name, pid)
	}

	return nil
}

// StartProcess starts a new process
func (pm *ProcessManager) StartProcess(procConfig *config.ProcessConfig) (*ManagedProcess, error) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Check if process already exists
	if proc, exists := pm.processes[procConfig.Name]; exists {
		if proc.Status == "running" {
			return nil, fmt.Errorf("process %s is already running", procConfig.Name)
		}
	}

	// Handle cluster mode
	if procConfig.Cluster.Instances > 1 {
		return pm.startClusterProcess(procConfig)
	}

	// Run pre-start script if defined
	if procConfig.Scripts.PreStart != "" {
		if err := runScript(procConfig.Scripts.PreStart); err != nil {
			return nil, fmt.Errorf("pre-start script failed: %v", err)
		}
	}

	// Create command
	cmd := exec.Command(procConfig.Command, procConfig.Args...)

	// Set working directory
	if procConfig.WorkingDir != "" {
		cmd.Dir = procConfig.WorkingDir
	}

	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range procConfig.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set up user/group if specified
	if procConfig.User != "" {
		if err := setProcessUser(cmd, procConfig.User, procConfig.Group); err != nil {
			return nil, err
		}
	}

	// Set up logging
	logFiles, err := setupLogging(procConfig, pm.logsPath)
	if err != nil {
		return nil, err
	}

	// Set up stdout/stderr
	stdout, stderr := logFiles["stdout"], logFiles["stderr"]
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// Start the process
	if err := cmd.Start(); err != nil {
		closeLogFiles(logFiles)
		return nil, err
	}

	// Create managed process
	proc := &ManagedProcess{
		Config:    procConfig,
		Cmd:       cmd,
		PID:       cmd.Process.Pid,
		Status:    "running",
		StartTime: time.Now(),
		LogFiles:  logFiles,
	}

	// Save PID file
	if err := utils.WritePIDFile(proc.PID, procConfig.Name, pm.processesPath); err != nil {
		logrus.Warnf("Failed to write PID file: %v", err)
	}

	// Save config file
	configPath := filepath.Join(pm.processesPath, fmt.Sprintf("%s.gem", procConfig.Name))
	if err := saveConfigFile(procConfig, configPath); err != nil {
		logrus.Warnf("Failed to save config file: %v", err)
	}

	// Store process
	pm.processes[procConfig.Name] = proc

	// Run post-start script if defined
	if procConfig.Scripts.PostStart != "" {
		go func() {
			if err := runScript(procConfig.Scripts.PostStart); err != nil {
				logrus.Warnf("Post-start script failed: %v", err)
			}
		}()
	}

	// Monitor process in background
	go pm.monitorProcess(proc)

	logrus.Infof("Started process %s (PID: %d)", procConfig.Name, proc.PID)
	return proc, nil
}

// startClusterProcess starts a process in cluster mode
func (pm *ProcessManager) startClusterProcess(procConfig *config.ProcessConfig) (*ManagedProcess, error) {
	instances := procConfig.Cluster.Instances
	if instances <= 0 {
		instances = 1
	}

	// Create master process
	masterProc := &ManagedProcess{
		Config:       procConfig,
		Status:       "running",
		StartTime:    time.Now(),
		ClusterProcs: make([]*ManagedProcess, 0, instances),
	}

	// Start worker processes
	for i := 0; i < instances; i++ {
		// Clone the config for this instance
		instanceConfig := *procConfig
		instanceConfig.Name = fmt.Sprintf("%s-worker-%d", procConfig.Name, i)
		instanceConfig.Cluster.Instances = 0 // Prevent recursive cluster creation

		// Start the worker process
		proc, err := pm.StartProcess(&instanceConfig)
		if err != nil {
			logrus.Errorf("Failed to start worker %d for cluster %s: %v", i, procConfig.Name, err)
			continue
		}

		// Add to cluster processes
		masterProc.ClusterProcs = append(masterProc.ClusterProcs, proc)
	}

	// Store master process
	pm.processes[procConfig.Name] = masterProc

	logrus.Infof("Started cluster %s with %d instances", procConfig.Name, len(masterProc.ClusterProcs))
	return masterProc, nil
}

// StopProcess stops a running process
func (pm *ProcessManager) StopProcess(name string, force bool) error {
	pm.mutex.Lock()
	proc, exists := pm.processes[name]
	pm.mutex.Unlock()

	if !exists {
		return fmt.Errorf("process %s not found", name)
	}

	// Handle cluster mode
	if len(proc.ClusterProcs) > 0 {
		for _, workerProc := range proc.ClusterProcs {
			if err := pm.StopProcess(workerProc.Config.Name, force); err != nil {
				logrus.Warnf("Failed to stop worker %s: %v", workerProc.Config.Name, err)
			}
		}

		// Update master process status
		proc.mu.Lock()
		proc.Status = "stopped"
		proc.mu.Unlock()

		// Remove from processes map
		pm.mutex.Lock()
		delete(pm.processes, name)
		pm.mutex.Unlock()

		return nil
	}

	// Run pre-stop script if defined
	if proc.Config.Scripts.PreStop != "" {
		if err := runScript(proc.Config.Scripts.PreStop); err != nil {
			logrus.Warnf("Pre-stop script failed: %v", err)
		}
	}

	// Stop the process
	var err error
	if force {
		err = proc.Cmd.Process.Kill()
	} else {
		err = proc.Cmd.Process.Signal(syscall.SIGTERM)
	}

	if err != nil {
		return err
	}

	// Update process status
	proc.mu.Lock()
	proc.Status = "stopped"
	proc.mu.Unlock()

	// Wait for process to exit
	go func() {
		proc.Cmd.Wait()

		// Close log files
		closeLogFiles(proc.LogFiles)

		// Delete PID file
		utils.DeletePIDFile(name, pm.processesPath)

		// Run post-stop script if defined
		if proc.Config.Scripts.PostStop != "" {
			if err := runScript(proc.Config.Scripts.PostStop); err != nil {
				logrus.Warnf("Post-stop script failed: %v", err)
			}
		}

		// Remove from processes map
		pm.mutex.Lock()
		delete(pm.processes, name)
		pm.mutex.Unlock()

		logrus.Infof("Process %s stopped", name)
	}()

	return nil
}

// RestartProcess restarts a running process
func (pm *ProcessManager) RestartProcess(name string) error {
	pm.mutex.RLock()
	proc, exists := pm.processes[name]
	pm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("process %s not found", name)
	}

	// Handle cluster mode
	if len(proc.ClusterProcs) > 0 {
		for _, workerProc := range proc.ClusterProcs {
			if err := pm.RestartProcess(workerProc.Config.Name); err != nil {
				logrus.Warnf("Failed to restart worker %s: %v", workerProc.Config.Name, err)
			}
		}
		return nil
	}

	// Stop the process
	if err := pm.StopProcess(name, false); err != nil {
		return err
	}

	// Wait a moment for the process to stop
	time.Sleep(time.Duration(proc.Config.RestartDelay) * time.Second)

	// Start the process again
	_, err := pm.StartProcess(proc.Config)
	return err
}

// GetProcess returns a process by name
func (pm *ProcessManager) GetProcess(name string) (*ManagedProcess, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	proc, exists := pm.processes[name]
	if !exists {
		return nil, fmt.Errorf("process %s not found", name)
	}

	return proc, nil
}

// ListProcesses returns a list of all managed processes
func (pm *ProcessManager) ListProcesses() []*ManagedProcess {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	processes := make([]*ManagedProcess, 0, len(pm.processes))
	for _, proc := range pm.processes {
		processes = append(processes, proc)
	}

	return processes
}

// GetProcessInfo returns detailed information about a process
func (pm *ProcessManager) GetProcessInfo(name string) (*utils.ProcessInfo, error) {
	proc, err := pm.GetProcess(name)
	if err != nil {
		return nil, err
	}

	// Handle cluster mode
	if len(proc.ClusterProcs) > 0 {
		// Get info for master process
		info := &utils.ProcessInfo{
			Name:      proc.Config.Name,
			Status:    proc.Status,
			StartTime: proc.StartTime,
			Uptime:    time.Since(proc.StartTime).Round(time.Second).String(),
			Command:   proc.Config.Command,
			Instances: len(proc.ClusterProcs),
		}

		return info, nil
	}

	// Get detailed process info
	return utils.GetProcessInfo(int32(proc.PID))
}

// AttachShell attaches an interactive shell to a running process
func (pm *ProcessManager) AttachShell(name string) (*os.File, error) {
	proc, err := pm.GetProcess(name)
	if err != nil {
		return nil, err
	}

	// Handle cluster mode
	if len(proc.ClusterProcs) > 0 {
		return nil, fmt.Errorf("cannot attach shell to cluster master, specify a worker instance")
	}

	// Check if process is running
	if proc.Status != "running" {
		return nil, fmt.Errorf("process %s is not running", name)
	}

	// Create a new command for the shell
	cmd := exec.Command("sh")

	// Set the same working directory as the process
	cmd.Dir = proc.Config.WorkingDir

	// Set the same environment variables
	cmd.Env = os.Environ()
	for k, v := range proc.Config.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Create a pseudoterminal
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	// Store the PTY
	proc.mu.Lock()
	proc.PTY = ptmx
	proc.mu.Unlock()

	return ptmx, nil
}

// DetachShell detaches an interactive shell from a process
func (pm *ProcessManager) DetachShell(name string) error {
	proc, err := pm.GetProcess(name)
	if err != nil {
		return err
	}

	proc.mu.Lock()
	defer proc.mu.Unlock()

	if proc.PTY != nil {
		proc.PTY.Close()
		proc.PTY = nil
	}

	return nil
}

// GetLogs returns the logs for a process
func (pm *ProcessManager) GetLogs(name string, stream string, lines int) ([]string, error) {
	proc, err := pm.GetProcess(name)
	if err != nil {
		return nil, err
	}

	// Handle cluster mode
	if len(proc.ClusterProcs) > 0 {
		return nil, fmt.Errorf("cannot get logs for cluster master, specify a worker instance")
	}

	// Determine log file path
	var logPath string
	if stream == "stdout" {
		if proc.Config.Log.Stdout != "" {
			logPath = proc.Config.Log.Stdout
		} else {
			logPath = filepath.Join(pm.logsPath, fmt.Sprintf("%s.out.log", proc.Config.Name))
		}
	} else if stream == "stderr" {
		if proc.Config.Log.Stderr != "" {
			logPath = proc.Config.Log.Stderr
		} else {
			logPath = filepath.Join(pm.logsPath, fmt.Sprintf("%s.err.log", proc.Config.Name))
		}
	} else {
		return nil, fmt.Errorf("invalid stream: %s", stream)
	}

	// Read the log file
	return readLastLines(logPath, lines)
}

// monitorProcess monitors a process and handles restarts
func (pm *ProcessManager) monitorProcess(proc *ManagedProcess) {
	// Wait for the process to exit
	err := proc.Cmd.Wait()

	// Process has exited
	proc.mu.Lock()
	proc.Status = "stopped"
	proc.mu.Unlock()

	// Close log files
	closeLogFiles(proc.LogFiles)

	// Check if we should restart the process
	shouldRestart := false
	if proc.Config.Restart == "always" {
		shouldRestart = true
	} else if proc.Config.Restart == "on-failure" && err != nil {
		shouldRestart = true
	}

	// Check max restarts
	if shouldRestart && (proc.Config.MaxRestarts == 0 || proc.Restarts < proc.Config.MaxRestarts) {
		logrus.Infof("Process %s exited, restarting in %d seconds", proc.Config.Name, proc.Config.RestartDelay)

		// Wait before restarting
		time.Sleep(time.Duration(proc.Config.RestartDelay) * time.Second)

		// Increment restart counter
		proc.mu.Lock()
		proc.Restarts++
		proc.mu.Unlock()

		// Restart the process
		_, err := pm.StartProcess(proc.Config)
		if err != nil {
			logrus.Errorf("Failed to restart process %s: %v", proc.Config.Name, err)
		}
	} else {
		// Process won't be restarted, clean up
		utils.DeletePIDFile(proc.Config.Name, pm.processesPath)

		pm.mutex.Lock()
		delete(pm.processes, proc.Config.Name)
		pm.mutex.Unlock()

		logrus.Infof("Process %s exited and won't be restarted", proc.Config.Name)
	}
}

// Helper functions

// setupLogging sets up logging for a process
func setupLogging(procConfig *config.ProcessConfig, logsPath string) (map[string]*os.File, error) {
	logFiles := make(map[string]*os.File)

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logsPath, 0755); err != nil {
		return nil, err
	}

	// Set up stdout log
	stdoutPath := procConfig.Log.Stdout
	if stdoutPath == "" {
		stdoutPath = filepath.Join(logsPath, fmt.Sprintf("%s.out.log", procConfig.Name))
	} else if !filepath.IsAbs(stdoutPath) {
		stdoutPath = filepath.Join(logsPath, stdoutPath)
	}

	// Create stdout log file
	stdout, err := os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	logFiles["stdout"] = stdout

	// Set up stderr log
	stderrPath := procConfig.Log.Stderr
	if stderrPath == "" {
		stderrPath = filepath.Join(logsPath, fmt.Sprintf("%s.err.log", procConfig.Name))
	} else if !filepath.IsAbs(stderrPath) {
		stderrPath = filepath.Join(logsPath, stderrPath)
	}

	// Create stderr log file
	stderr, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		closeLogFiles(logFiles)
		return nil, err
	}
	logFiles["stderr"] = stderr

	return logFiles, nil
}

// closeLogFiles closes all log files
func closeLogFiles(logFiles map[string]*os.File) {
	for _, file := range logFiles {
		file.Close()
	}
}

// setProcessUser sets the user and group for a process
func setProcessUser(cmd *exec.Cmd, username, groupname string) error {
	// Get user info
	u, err := user.Lookup(username)
	if err != nil {
		return err
	}

	// Parse user ID
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return err
	}

	// Set up credentials
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid)}

	// Set group if specified
	if groupname != "" {
		g, err := user.LookupGroup(groupname)
		if err != nil {
			return err
		}

		gid, err := strconv.Atoi(g.Gid)
		if err != nil {
			return err
		}

		cmd.SysProcAttr.Credential.Gid = uint32(gid)
	}

	return nil
}

// saveConfigFile saves a process configuration to a file
func saveConfigFile(procConfig *config.ProcessConfig, filePath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Marshal config to YAML
	data, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer data.Close()

	// Use viper to write the config
	v := viper.New()
	v.SetConfigType("yaml")

	// Set values
	v.Set("name", procConfig.Name)
	v.Set("cmd", procConfig.Command)

	if len(procConfig.Args) > 0 {
		v.Set("args", procConfig.Args)
	}
	if procConfig.WorkingDir != "" {
		v.Set("cwd", procConfig.WorkingDir)
	}
	if len(procConfig.Environment) > 0 {
		v.Set("env", procConfig.Environment)
	}
	if procConfig.Restart != "" {
		v.Set("restart", procConfig.Restart)
	}
	if procConfig.MaxRestarts > 0 {
		v.Set("max_restarts", procConfig.MaxRestarts)
	}
	if procConfig.RestartDelay > 0 {
		v.Set("restart_delay", procConfig.RestartDelay)
	}
	if procConfig.Cluster.Instances > 0 {
		v.Set("cluster.instances", procConfig.Cluster.Instances)
		v.Set("cluster.mode", procConfig.Cluster.Mode)
	}
	if procConfig.Log.Stdout != "" || procConfig.Log.Stderr != "" || procConfig.Log.Rotate {
		if procConfig.Log.Stdout != "" {
			v.Set("log.stdout", procConfig.Log.Stdout)
		}
		if procConfig.Log.Stderr != "" {
			v.Set("log.stderr", procConfig.Log.Stderr)
		}
		if procConfig.Log.Rotate {
			v.Set("log.rotate", procConfig.Log.Rotate)
			v.Set("log.max_size", procConfig.Log.MaxSize)
			v.Set("log.max_files", procConfig.Log.MaxFiles)
		}
	}
	if procConfig.AutoStart {
		v.Set("autostart", procConfig.AutoStart)
	}
	if procConfig.User != "" {
		v.Set("user", procConfig.User)
	}
	if procConfig.Group != "" {
		v.Set("group", procConfig.Group)
	}
	if procConfig.Scripts.PreStart != "" || procConfig.Scripts.PostStart != "" ||
		procConfig.Scripts.PreStop != "" || procConfig.Scripts.PostStop != "" {
		if procConfig.Scripts.PreStart != "" {
			v.Set("scripts.pre_start", procConfig.Scripts.PreStart)
		}
		if procConfig.Scripts.PostStart != "" {
			v.Set("scripts.post_start", procConfig.Scripts.PostStart)
		}
		if procConfig.Scripts.PreStop != "" {
			v.Set("scripts.pre_stop", procConfig.Scripts.PreStop)
		}
		if procConfig.Scripts.PostStop != "" {
			v.Set("scripts.post_stop", procConfig.Scripts.PostStop)
		}
	}

	return v.WriteConfig()
}

// runScript runs a script
func runScript(script string) error {
	cmd := exec.Command("sh", "-c", script)
	return cmd.Run()
}

// readLastLines reads the last n lines from a file
func readLastLines(filePath string, n int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// If n is 0 or negative, return all lines
	if n <= 0 {
		var lines []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		return lines, scanner.Err()
	}

	// Read all lines
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Return last n lines
	if len(lines) <= n {
		return lines, nil
	}
	return lines[len(lines)-n:], nil
}
