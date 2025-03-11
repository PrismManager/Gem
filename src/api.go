// gem/api.go

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// APIServer provides a REST API for gem
type APIServer struct {
	port string
}

// NewAPIServer creates a new API server
func NewAPIServer(port string) *APIServer {
	return &APIServer{
		port: port,
	}
}

// Start starts the API server
func (s *APIServer) Start() error {
	// Set up routes
	http.HandleFunc("/api/processes", s.handleProcesses)
	http.HandleFunc("/api/processes/", s.handleProcess)
	http.HandleFunc("/api/scripts", s.handleScripts)
	http.HandleFunc("/api/scripts/", s.handleScript)

	// Start the server
	address := fmt.Sprintf(":%s", s.port)
	log.Printf("API server listening on %s", address)
	return http.ListenAndServe(address, nil)
}

// handleProcesses handles the /api/processes endpoint
func (s *APIServer) handleProcesses(w http.ResponseWriter, r *http.Request) {
	service := NewProcessService()

	switch r.Method {
	case http.MethodGet:
		// List all processes
		processes, err := service.List()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list processes: %v", err), http.StatusInternalServerError)
			return
		}
		s.jsonResponse(w, processes)

	case http.MethodPost:
		// Create a new process
		var request struct {
			Name        string   `json:"name"`
			Cmd         string   `json:"cmd"`
			Cwd         string   `json:"cwd"`
			Restart     bool     `json:"restart"`
			MaxRestarts int      `json:"maxRestarts"`
			Env         []string `json:"env"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		if request.Name == "" || request.Cmd == "" {
			http.Error(w, "Name and cmd are required", http.StatusBadRequest)
			return
		}

		// Set defaults
		if request.Cwd == "" {
			request.Cwd = "."
		}
		if request.MaxRestarts == 0 {
			request.MaxRestarts = 5
		}

		// Start the process
		process, err := service.Start(request.Name, request.Cmd, request.Cwd, request.Restart, request.MaxRestarts, request.Env)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to start process: %v", err), http.StatusInternalServerError)
			return
		}

		s.jsonResponse(w, process)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleProcess handles the /api/processes/{name} endpoint
func (s *APIServer) handleProcess(w http.ResponseWriter, r *http.Request) {
	service := NewProcessService()

	// Extract process name from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/processes/")
	parts := strings.Split(path, "/")
	name := parts[0]

	if name == "" {
		http.Error(w, "Process name is required", http.StatusBadRequest)
		return
	}

	// Handle actions
	if len(parts) > 1 && parts[1] != "" {
		action := parts[1]
		s.handleProcessAction(w, r, name, action)
		return
	}

	// Handle CRUD operations
	switch r.Method {
	case http.MethodGet:
		// Get process
		process, err := service.Get(name)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get process: %v", err), http.StatusNotFound)
			return
		}
		s.jsonResponse(w, process)

	case http.MethodDelete:
		// Stop and remove process
		if err := service.Stop(name); err != nil {
			http.Error(w, fmt.Sprintf("Failed to stop process: %v", err), http.StatusInternalServerError)
			return
		}
		s.jsonResponse(w, map[string]string{"status": "stopped", "message": fmt.Sprintf("Process '%s' stopped", name)})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleProcessAction handles process actions like start, stop, restart
func (s *APIServer) handleProcessAction(w http.ResponseWriter, r *http.Request, name, action string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	service := NewProcessService()

	switch action {
	case "start":
		// Get process to check if it exists
		process, err := service.Get(name)
		if err != nil {
			http.Error(w, fmt.Sprintf("Process not found: %v", err), http.StatusNotFound)
			return
		}

		// Start process
		process, err = service.Restart(name)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to start process: %v", err), http.StatusInternalServerError)
			return
		}
		s.jsonResponse(w, process)

	case "stop":
		// Stop process
		if err := service.Stop(name); err != nil {
			http.Error(w, fmt.Sprintf("Failed to stop process: %v", err), http.StatusInternalServerError)
			return
		}
		s.jsonResponse(w, map[string]string{"status": "stopped", "message": fmt.Sprintf("Process '%s' stopped", name)})

	case "restart":
		// Restart process
		process, err := service.Restart(name)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to restart process: %v", err), http.StatusInternalServerError)
			return
		}
		s.jsonResponse(w, process)

	case "logs":
		// Not implemented via API, return error
		http.Error(w, "Logs are not available via API", http.StatusNotImplemented)

	default:
		http.Error(w, fmt.Sprintf("Unknown action: %s", action), http.StatusBadRequest)
	}
}

// handleScripts handles the /api/scripts endpoint
func (s *APIServer) handleScripts(w http.ResponseWriter, r *http.Request) {
	service := NewScriptService()

	switch r.Method {
	case http.MethodGet:
		// List all scripts
		scripts, err := service.List()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list scripts: %v", err), http.StatusInternalServerError)
			return
		}
		s.jsonResponse(w, scripts)

	case http.MethodPost:
		// Create a new script
		var request struct {
			Name     string `json:"name"`
			File     string `json:"file"`
			Schedule string `json:"schedule"`
			Process  string `json:"process"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		if request.Name == "" || request.File == "" {
			http.Error(w, "Name and file are required", http.StatusBadRequest)
			return
		}

		// Add the script
		if err := service.Add(request.Name, request.File, request.Schedule, request.Process); err != nil {
			http.Error(w, fmt.Sprintf("Failed to add script: %v", err), http.StatusInternalServerError)
			return
		}

		s.jsonResponse(w, map[string]string{"status": "added", "message": fmt.Sprintf("Script '%s' added", request.Name)})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleScript handles the /api/scripts/{name} endpoint
func (s *APIServer) handleScript(w http.ResponseWriter, r *http.Request) {
	service := NewScriptService()

	// Extract script name from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/scripts/")
	parts := strings.Split(path, "/")
	name := parts[0]

	if name == "" {
		http.Error(w, "Script name is required", http.StatusBadRequest)
		return
	}

	// Handle actions
	if len(parts) > 1 && parts[1] != "" {
		action := parts[1]
		s.handleScriptAction(w, r, name, action)
		return
	}

	// Handle CRUD operations
	switch r.Method {
	case http.MethodGet:
		// Get script
		script, err := service.Get(name)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get script: %v", err), http.StatusNotFound)
			return
		}
		s.jsonResponse(w, script)

	case http.MethodDelete:
		// Remove script
		if err := service.Remove(name); err != nil {
			http.Error(w, fmt.Sprintf("Failed to remove script: %v", err), http.StatusInternalServerError)
			return
		}
		s.jsonResponse(w, map[string]string{"status": "removed", "message": fmt.Sprintf("Script '%s' removed", name)})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleScriptAction handles script actions like run
func (s *APIServer) handleScriptAction(w http.ResponseWriter, r *http.Request, name, action string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	service := NewScriptService()

	switch action {
	case "run":
		// Run script
		if err := service.Run(name); err != nil {
			http.Error(w, fmt.Sprintf("Failed to run script: %v", err), http.StatusInternalServerError)
			return
		}
		s.jsonResponse(w, map[string]string{"status": "success", "message": fmt.Sprintf("Script '%s' executed", name)})

	default:
		http.Error(w, fmt.Sprintf("Unknown action: %s", action), http.StatusBadRequest)
	}
}

// jsonResponse sends a JSON response
func (s *APIServer) jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	
	// Encode JSON
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}