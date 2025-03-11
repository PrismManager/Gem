package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Process represents a managed process
type Process struct {
	Name        string    `json:"name"`
	Cmd         string    `json:"cmd"`
	Cwd         string    `json:"cwd"`
	Pid         int       `json:"pid"`
	Status      string    `json:"status"`
	Restart     bool      `json:"restart"`
	MaxRestarts int       `json:"maxRestarts"`
	Restarts    int       `json:"restarts"`
	Env         []string  `json:"env"`
	StartTime   time.Time `json:"startTime"`
}

// ProcessService handles process management
type ProcessService struct{}

// NewProcessService creates a new process service
func NewProcessService() *ProcessService {
	return &ProcessService{}
}

// Start starts a new process
func (s *ProcessService) Start(name, cmd, cwd string, restart bool, maxRestarts int, env []string) (*Process, error) {
	// Check if process already exists
	if _, err := s.Get(name); err == nil {
		return nil, fmt.Errorf("process with name '%s' already exists", name)
	}

	// Create process
	process := &Process{
		Name:        name,
		Cmd:         cmd,
		Cwd:         cwd,
		Status:      "starting",
		Restart:     restart,
		MaxRestarts: maxRestarts,
		Env:         env,
		StartTime:   time.Now(),
	}

	// Save process configuration
	if err := s.saveProcess(process); err != nil {
		return nil, err
	}

	// Start the actual process
	if err := s.startProcess(process); err != nil {
		// Clean up on failure
		s.removeProcess(name)
		return nil, err
	}

	return process, nil
}

// startProcess launches the actual process
func (s *ProcessService) startProcess(process *Process) error {
	// Parse command and arguments
	cmdParts := strings.Fields(process.Cmd)
	if len(cmdParts) == 0 {
		return fmt.Errorf("invalid command")
	}

	// Create command
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Dir = process.Cwd
	cmd.Env = append(os.Environ(), process.Env...)

	// Setup log files
	stdout, err := os.Create(filepath.Join(logsDir, process.Name+".out.log"))
	if err != nil {
		return fmt.Errorf("failed to create stdout log file: %w", err)
	}

	stderr, err := os.Create(filepath.Join(logsDir, process.Name+".err.log"))
	if err != nil {
		return fmt.Errorf("failed to create stderr log file: %w", err)
	}

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	// Update process info
	process.Pid = cmd.Process.Pid
	process.Status = "running"

	// Save the updated process
	if err := s.saveProcess(process); err != nil {
		return err
	}

	// Create PID file
	pidFile := filepath.Join(pidDir, process.Name+".pid")
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(process.Pid)), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Handle process completion and restart if necessary
	go func() {
		cmd.Wait()

		// Process exited, update status
		process.Status = "stopped"
		
		if process.Restart && process.Restarts < process.MaxRestarts {
			process.Restarts++
			process.Status = "restarting"
			s.saveProcess(process)
			
			// Log restart event
			msg := fmt.Sprintf("[%s] Process exited, restarting (%d/%d)...\n", 
				time.Now().Format(time.RFC3339), process.Restarts, process.MaxRestarts)
			
			f, err := os.OpenFile(filepath.Join(logsDir, process.Name+".out.log"), 
				os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				f.WriteString(msg)
				f.Close()
			}
			
			// Restart after a small delay
			time.Sleep(1 * time.Second)
			s.startProcess(process)
		} else {
			// Clean up PID file
			os.Remove(pidFile)
			s.saveProcess(process)
		}
	}()

	return nil
}

// Stop stops a process
func (s *ProcessService) Stop(name string) error {
	process, err := s.Get(name)
	if err != nil {
		return err
	}

	// Check if process is already stopped
	if process.Status == "stopped" {
		return fmt.Errorf("process '%s' is already stopped", name)
	}

	// Try to kill the process
	proc, err := os.FindProcess(process.Pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// First try SIGTERM for graceful shutdown
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		// If SIGTERM failed, try SIGKILL
		if err := proc.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	// Update process status
	process.Status = "stopped"
	if err := s.saveProcess(process); err != nil {
		return err
	}

	// Clean up PID file
	pidFile := filepath.Join(pidDir, process.Name+".pid")
	os.Remove(pidFile)

	return nil
}

// Restart restarts a process
func (s *ProcessService) Restart(name string) (*Process, error) {
	process, err := s.Get(name)
	if err != nil {
		return nil, err
	}

	// Stop process if it's running
	if process.Status == "running" {
		if err := s.Stop(name); err != nil {
			return nil, fmt.Errorf("failed to stop process: %w", err)
		}
	}

	// Reset restart counter
	process.Restarts = 0
	process.StartTime = time.Now()
	
	// Start process again
	if err := s.startProcess(process); err != nil {
		return nil, fmt.Errorf("failed to restart process: %w", err)
	}

	return process, nil
}

// Get gets a process by name
func (s *ProcessService) Get(name string) (*Process, error) {
	configFile := filepath.Join(configDir, name+".json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("process '%s' not found", name)
	}

	var process Process
	if err := json.Unmarshal(data, &process); err != nil {
		return nil, fmt.Errorf("failed to parse process config: %w", err)
	}

	// Verify process is actually running if status is "running"
	if process.Status == "running" {
		if !isProcessRunning(process.Pid) {
			process.Status = "dead"
			s.saveProcess(&process)
		}
	}

	return &process, nil
}

// List lists all processes
func (s *ProcessService) List() ([]*Process, error) {
	files, err := filepath.Glob(filepath.Join(configDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list process files: %w", err)
	}

	var processes []*Process
	for _, file := range files {
		// Extract process name from filename
		name := strings.TrimSuffix(filepath.Base(file), ".json")
		
		// Get process details
		process, err := s.Get(name)
		if err == nil {
			processes = append(processes, process)
		}
	}

	return processes, nil
}

// ShowLogs displays logs for a process
func (s *ProcessService) ShowLogs(name string, follow bool, lines int) error {
	// Check if process exists
	if _, err := s.Get(name); err != nil {
		return err
	}

	// Get log files
	stdoutLog := filepath.Join(logsDir, name+".out.log")
	stderrLog := filepath.Join(logsDir, name+".err.log")

	// Show stdout logs
	fmt.Printf("=== STDOUT (%s) ===\n", name)
	if err := tailFile(stdoutLog, lines); err != nil {
		fmt.Printf("Error reading stdout log: %v\n", err)
	}

	fmt.Printf("\n=== STDERR (%s) ===\n", name)
	if err := tailFile(stderrLog, lines); err != nil {
		fmt.Printf("Error reading stderr log: %v\n", err)
	}

	// If follow option is enabled, keep showing new log entries
	if follow {
		fmt.Println("\nFollowing logs (Ctrl+C to stop)...")
		
		// Open stdout for following
		stdout, err := os.Open(stdoutLog)
		if err != nil {
			return fmt.Errorf("failed to open stdout log: %w", err)
		}
		defer stdout.Close()

		// Seek to end of file
		if _, err := stdout.Seek(0, io.SeekEnd); err != nil {
			return fmt.Errorf("failed to seek in stdout log: %w", err)
		}

		// Create scanner
		scanner := bufio.NewScanner(stdout)
		
		// Keep scanning for new lines
		for {
			if scanner.Scan() {
				fmt.Println(scanner.Text())
			} else {
				// No new lines, sleep and continue
				time.Sleep(500 * time.Millisecond)
			}
		}
	}

	return nil
}

// EnterShell starts a shell in the process working directory
func (s *ProcessService) EnterShell(name string) error {
	process, err := s.Get(name)
	if err != nil {
		return err
	}

	// Get the shell from environment or use /bin/bash as default
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	// Create command
	cmd := exec.Command(shell)
	cmd.Dir = process.Cwd
	cmd.Env = append(os.Environ(), process.Env...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Print info
	fmt.Printf("Entering shell for process '%s' in directory '%s'\n", process.Name, process.Cwd)
	fmt.Println("Type 'exit' to return to Gem")

	// Start interactive shell
	return cmd.Run()
}

// saveProcess saves process configuration to file
func (s *ProcessService) saveProcess(process *Process) error {
	data, err := json.MarshalIndent(process, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal process config: %w", err)
	}

	configFile := filepath.Join(configDir, process.Name+".json")
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write process config: %w", err)
	}

	return nil
}

// removeProcess removes process configuration
func (s *ProcessService) removeProcess(name string) error {
	configFile := filepath.Join(configDir, name+".json")
	if err := os.Remove(configFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove process config: %w", err)
	}
	return nil
}

// isProcessRunning checks if a process with the given PID exists
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds, so we need to send signal 0
	// to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// tailFile shows the last n lines of a file
func tailFile(file string, lines int) error {
	// Check if file exists
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return fmt.Errorf("log file does not exist")
	}

	// Run tail command
	cmd := exec.Command("tail", "-n", strconv.Itoa(lines), file)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}