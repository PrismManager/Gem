package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// Script represents an automation script
type Script struct {
	Name     string `json:"name"`
	File     string `json:"file"`
	Schedule string `json:"schedule"`
	Process  string `json:"process"`
}

// ScriptService handles script management
type ScriptService struct {
	cronScheduler *cron.Cron
}

// NewScriptService creates a new script service
func NewScriptService() *ScriptService {
	scheduler := cron.New(cron.WithSeconds())
	scheduler.Start()

	service := &ScriptService{
		cronScheduler: scheduler,
	}

	// Load and schedule all scripts
	service.loadScheduledScripts()

	return service
}

// loadScheduledScripts loads all scripts and schedules them
func (s *ScriptService) loadScheduledScripts() {
	scripts, err := s.List()
	if err != nil {
		return
	}

	for _, script := range scripts {
		if script.Schedule != "" {
			s.scheduleScript(script)
		}
	}
}

// scheduleScript adds a script to the cron scheduler
func (s *ScriptService) scheduleScript(script *Script) error {
	if script.Schedule == "" {
		return nil
	}

	_, err := s.cronScheduler.AddFunc(script.Schedule, func() {
		s.Run(script.Name)
	})

	if err != nil {
		return fmt.Errorf("failed to schedule script: %w", err)
	}

	return nil
}

// Add adds a new script
func (s *ScriptService) Add(name, file, schedule, process string) error {
	// Check if script already exists
	if _, err := s.Get(name); err == nil {
		return fmt.Errorf("script with name '%s' already exists", name)
	}

	// Check if file exists
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return fmt.Errorf("script file '%s' does not exist", file)
	}

	// If process is specified, check if it exists
	if process != "" {
		processService := NewProcessService()
		if _, err := processService.Get(process); err != nil {
			return fmt.Errorf("process '%s' does not exist", process)
		}
	}

	// Create script
	script := &Script{
		Name:     name,
		File:     file,
		Schedule: schedule,
		Process:  process,
	}

	// Save script configuration
	if err := s.saveScript(script); err != nil {
		return err
	}

	// Schedule script if it has a schedule
	if schedule != "" {
		if err := s.scheduleScript(script); err != nil {
			return err
		}
	}

	return nil
}

// Get gets a script by name
func (s *ScriptService) Get(name string) (*Script, error) {
	configFile := filepath.Join(scriptsDir, name+".json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("script '%s' not found", name)
	}

	var script Script
	if err := json.Unmarshal(data, &script); err != nil {
		return nil, fmt.Errorf("failed to parse script config: %w", err)
	}

	return &script, nil
}

// List lists all scripts
func (s *ScriptService) List() ([]*Script, error) {
	files, err := filepath.Glob(filepath.Join(scriptsDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list script files: %w", err)
	}

	var scripts []*Script
	for _, file := range files {
		// Extract script name from filename
		name := strings.TrimSuffix(filepath.Base(file), ".json")
		
		// Get script details
		script, err := s.Get(name)
		if err == nil {
			scripts = append(scripts, script)
		}
	}

	return scripts, nil
}

// Run runs a script
func (s *ScriptService) Run(name string) error {
	script, err := s.Get(name)
	if err != nil {
		return err
	}

	// Check if script file exists
	if _, err := os.Stat(script.File); os.IsNotExist(err) {
		return fmt.Errorf("script file '%s' does not exist", script.File)
	}

	// Get process working directory if process is specified
	var workDir string
	if script.Process != "" {
		processService := NewProcessService()
		process, err := processService.Get(script.Process)
		if err != nil {
			return fmt.Errorf("process '%s' not found", script.Process)
		}
		workDir = process.Cwd
	} else {
		// Use script file directory as working directory
		workDir = filepath.Dir(script.File)
	}

	// Create log file
	logFile := filepath.Join(logsDir, "script_"+script.Name+".log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer f.Close()

	// Write header to log
	timestamp := time.Now().Format(time.RFC3339)
	f.WriteString(fmt.Sprintf("\n=== Script '%s' executed at %s ===\n", script.Name, timestamp))

	// Determine how to execute the script based on file extension
	var cmd *exec.Cmd
	ext := strings.ToLower(filepath.Ext(script.File))
	
	switch ext {
	case ".sh":
		cmd = exec.Command("/bin/bash", script.File)
	case ".py":
		cmd = exec.Command("python3", script.File)
	case ".js":
		cmd = exec.Command("node", script.File)
	default:
		// Try to execute directly
		cmd = exec.Command(script.File)
	}

	cmd.Dir = workDir
	cmd.Stdout = f
	cmd.Stderr = f

	// Execute the script
	if err := cmd.Run(); err != nil {
		f.WriteString(fmt.Sprintf("Error: %v\n", err))
		return fmt.Errorf("failed to execute script: %w", err)
	}

	f.WriteString("Script execution completed successfully\n")
	return nil
}

// Remove removes a script
func (s *ScriptService) Remove(name string) error {
	// Check if script exists
	if _, err := s.Get(name); err != nil {
		return err
	}

	// Remove script config
	configFile := filepath.Join(scriptsDir, name+".json")
	if err := os.Remove(configFile); err != nil {
		return fmt.Errorf("failed to remove script config: %w", err)
	}

	return nil
}

// saveScript saves script configuration to file
func (s *ScriptService) saveScript(script *Script) error {
	data, err := json.MarshalIndent(script, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal script config: %w", err)
	}

	configFile := filepath.Join(scriptsDir, script.Name+".json")
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write script config: %w", err)
	}

	return nil
}
