package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

// SystemService handles system service management
type SystemService struct{}

// NewSystemService creates a new system service
func NewSystemService() *SystemService {
	return &SystemService{}
}

// Install installs Gem as a system service
func (s *SystemService) Install() error {
	// Get executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create systemd service file
	serviceContent, err := s.generateSystemdService(exePath)
	if err != nil {
		return fmt.Errorf("failed to generate service file: %w", err)
	}

	// Write service file
	serviceFile := "/tmp/gem.service"
	if err := os.WriteFile(serviceFile, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Copy service file to systemd directory
	cmd := exec.Command("sudo", "cp", serviceFile, "/etc/systemd/system/gem.service")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy service file: %w", err)
	}

	// Reload systemd
	cmd = exec.Command("sudo", "systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Enable service
	cmd = exec.Command("sudo", "systemctl", "enable", "gem.service")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	// Start service
	cmd = exec.Command("sudo", "systemctl", "start", "gem.service")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Clean up temporary file
	os.Remove(serviceFile)

	return nil
}

// generateSystemdService generates a systemd service file
func (s *SystemService) generateSystemdService(exePath string) (string, error) {
	// Service template
	tmpl := `[Unit]
Description=Gem Process Manager
After=network.target

[Service]
ExecStart={{.ExecPath}} serve
Restart=always
RestartSec=5
User={{.User}}
Group={{.Group}}
WorkingDirectory={{.WorkDir}}
Environment=PATH=/usr/local/bin:/usr/bin:/bin

[Install]
WantedBy=multi-user.target
`

	// Get current user
	user := os.Getenv("USER")
	if user == "" {
		user = "root"
	}

	// Get current group
	group := user

	// Get working directory
	workDir := filepath.Dir(exePath)

	// Prepare template data
	data := struct {
		ExecPath string
		User     string
		Group    string
		WorkDir  string
	}{
		ExecPath: exePath,
		User:     user,
		Group:    group,
		WorkDir:  workDir,
	}

	// Parse and execute template
	t, err := template.New("service").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}