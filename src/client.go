package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a client for the Gem API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Gem client
func NewClient(host string, port string) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://%s:%s/api", host, port),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ListProcesses lists all processes
func (c *Client) ListProcesses() ([]*Process, error) {
	url := fmt.Sprintf("%s/processes", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var processes []*Process
	if err := json.NewDecoder(resp.Body).Decode(&processes); err != nil {
		return nil, err
	}

	return processes, nil
}

// GetProcess gets a process by name
func (c *Client) GetProcess(name string) (*Process, error) {
	url := fmt.Sprintf("%s/processes/%s", c.baseURL, name)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var process Process
	if err := json.NewDecoder(resp.Body).Decode(&process); err != nil {
		return nil, err
	}

	return &process, nil
}

// StartProcess starts a new process
func (c *Client) StartProcess(name string, cmd string, cwd string, restart bool, maxRestarts int, env []string) (*Process, error) {
	url := fmt.Sprintf("%s/processes", c.baseURL)

	// Prepare request body
	data := map[string]interface{}{
		"name":        name,
		"cmd":         cmd,
		"cwd":         cwd,
		"restart":     restart,
		"maxRestarts": maxRestarts,
		"env":         env,
	}

	// Send request
	resp, err := c.sendJSON("POST", url, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var process Process
	if err := json.NewDecoder(resp.Body).Decode(&process); err != nil {
		return nil, err
	}

	return &process, nil
}

// StopProcess stops a process
func (c *Client) StopProcess(name string) error {
	url := fmt.Sprintf("%s/processes/%s/stop", c.baseURL, name)
	resp, err := c.sendJSON("POST", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// RestartProcess restarts a process
func (c *Client) RestartProcess(name string) (*Process, error) {
	url := fmt.Sprintf("%s/processes/%s/restart", c.baseURL, name)
	resp, err := c.sendJSON("POST", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var process Process
	if err := json.NewDecoder(resp.Body).Decode(&process); err != nil {
		return nil, err
	}

	return &process, nil
}

// DeleteProcess deletes a process
func (c *Client) DeleteProcess(name string) error {
	url := fmt.Sprintf("%s/processes/%s", c.baseURL, name)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// ListScripts lists all scripts
func (c *Client) ListScripts() ([]*Script, error) {
	url := fmt.Sprintf("%s/scripts", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var scripts []*Script
	if err := json.NewDecoder(resp.Body).Decode(&scripts); err != nil {
		return nil, err
	}

	return scripts, nil
}

// GetScript gets a script by name
func (c *Client) GetScript(name string) (*Script, error) {
	url := fmt.Sprintf("%s/scripts/%s", c.baseURL, name)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var script Script
	if err := json.NewDecoder(resp.Body).Decode(&script); err != nil {
		return nil, err
	}

	return &script, nil
}

// AddScript adds a new script
func (c *Client) AddScript(name string, file string, schedule string, process string) error {
	url := fmt.Sprintf("%s/scripts", c.baseURL)

	// Prepare request body
	data := map[string]interface{}{
		"name":     name,
		"file":     file,
		"schedule": schedule,
		"process":  process,
	}

	// Send request
	resp, err := c.sendJSON("POST", url, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// RunScript runs a script
func (c *Client) RunScript(name string) error {
	url := fmt.Sprintf("%s/scripts/%s/run", c.baseURL, name)
	resp, err := c.sendJSON("POST", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// RemoveScript removes a script
func (c *Client) RemoveScript(name string) error {
	url := fmt.Sprintf("%s/scripts/%s", c.baseURL, name)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// sendJSON sends a JSON request
func (c *Client) sendJSON(method string, url string, data interface{}) (*http.Response, error) {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// parseError parses an error response
func (c *Client) parseError(resp *http.Response) error {
	var errResponse struct {
		Error string `json:"error"`
	}

	err := json.NewDecoder(resp.Body).Decode(&errResponse)
	if err != nil {
		return fmt.Errorf("Failed to parse error response: %v", err)
	}

	return fmt.Errorf("%s", errResponse.Error)
}
