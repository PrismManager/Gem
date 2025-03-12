package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/prism/gem/config"
	"github.com/prism/gem/core"
	"github.com/sirupsen/logrus"
)

// APIServer represents the API server
type APIServer struct {
	router         *gin.Engine
	processManager *core.ProcessManager
	upgrader       websocket.Upgrader
}

// NewAPIServer creates a new API server
func NewAPIServer(processManager *core.ProcessManager) *APIServer {
	// Set Gin to release mode in production
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(loggerMiddleware())

	server := &APIServer{
		router:         router,
		processManager: processManager,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins
			},
		},
	}

	server.setupRoutes()
	return server
}

// Start starts the API server
func (s *APIServer) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	logrus.Infof("Starting API server on %s", addr)
	return s.router.Run(addr)
}

// setupRoutes sets up the API routes
func (s *APIServer) setupRoutes() {
	// API version
	v1 := s.router.Group("/api/v1")

	// Process management
	processes := v1.Group("/processes")
	{
		processes.GET("", s.listProcesses)
		processes.POST("", s.startProcess)
		processes.GET("/:name", s.getProcess)
		processes.DELETE("/:name", s.stopProcess)
		processes.POST("/:name/restart", s.restartProcess)
		processes.GET("/:name/logs/:stream", s.getLogs)
		processes.GET("/:name/shell", s.shellWebsocket)
	}

	// Cluster management
	clusters := v1.Group("/clusters")
	{
		clusters.GET("", s.listClusters)
		clusters.GET("/:name", s.getCluster)
	}

	// System information
	v1.GET("/system", s.getSystemInfo)

	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

// API handlers

// listProcesses lists all processes
func (s *APIServer) listProcesses(c *gin.Context) {
	processes := s.processManager.ListProcesses()
	
	// Filter out cluster workers from top-level list
	filteredProcesses := make([]*core.ManagedProcess, 0)
	for _, proc := range processes {
		if !isClusterWorker(proc.Config.Name) {
			filteredProcesses = append(filteredProcesses, proc)
		}
	}
	
	c.JSON(http.StatusOK, filteredProcesses)
}

// startProcess starts a new process
func (s *APIServer) startProcess(c *gin.Context) {
	var procConfig config.ProcessConfig
	if err := c.ShouldBindJSON(&procConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	proc, err := s.processManager.StartProcess(&procConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, proc)
}

// getProcess gets information about a process
func (s *APIServer) getProcess(c *gin.Context) {
	name := c.Param("name")
	
	procInfo, err := s.processManager.GetProcessInfo(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, procInfo)
}

// stopProcess stops a process
func (s *APIServer) stopProcess(c *gin.Context) {
	name := c.Param("name")
	force := c.DefaultQuery("force", "false") == "true"
	
	if err := s.processManager.StopProcess(name, force); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

// restartProcess restarts a process
func (s *APIServer) restartProcess(c *gin.Context) {
	name := c.Param("name")
	
	if err := s.processManager.RestartProcess(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "restarting"})
}

// getLogs gets logs for a process
func (s *APIServer) getLogs(c *gin.Context) {
	name := c.Param("name")
	stream := c.Param("stream")
	
	if stream != "stdout" && stream != "stderr" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stream, must be stdout or stderr"})
		return
	}
	
	lines, err := strconv.Atoi(c.DefaultQuery("lines", "100"))
	if err != nil {
		lines = 100
	}
	
	logs, err := s.processManager.GetLogs(name, stream, lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// shellWebsocket handles shell access via websocket
func (s *APIServer) shellWebsocket(c *gin.Context) {
	name := c.Param("name")
	
	// Upgrade to websocket connection
	ws, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Errorf("Failed to upgrade to websocket: %v", err)
		return
	}
	defer ws.Close()
	
	// Attach shell to process
	pty, err := s.processManager.AttachShell(name)
	if err != nil {
		ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
		return
	}
	defer s.processManager.DetachShell(name)
	
	// Set up bidirectional communication
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := pty.Read(buf)
			if err != nil {
				break
			}
			if err := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				break
			}
		}
	}()
	
	// Read from websocket and write to pty
	for {
		messageType, p, err := ws.ReadMessage()
		if err != nil {
			break
		}
		if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
			if _, err := pty.Write(p); err != nil {
				break
			}
		}
	}
}

// listClusters lists all clusters
func (s *APIServer) listClusters(c *gin.Context) {
	processes := s.processManager.ListProcesses()
	
	// Filter only cluster masters
	clusters := make([]*core.ManagedProcess, 0)
	for _, proc := range processes {
		if len(proc.ClusterProcs) > 0 {
			clusters = append(clusters, proc)
		}
	}
	
	c.JSON(http.StatusOK, clusters)
}

// getCluster gets information about a cluster
func (s *APIServer) getCluster(c *gin.Context) {
	name := c.Param("name")
	
	proc, err := s.processManager.GetProcess(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	
	if len(proc.ClusterProcs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not a cluster"})
		return
	}
	
	c.JSON(http.StatusOK, proc)
}

// getSystemInfo gets system information
func (s *APIServer) getSystemInfo(c *gin.Context) {
	// TODO: Implement system information
	c.JSON(http.StatusOK, gin.H{
		"version": "1.0.0",
		"uptime":  "unknown",
	})
}

// Helper functions

// loggerMiddleware returns a gin middleware for logging requests
func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		
		// Process request
		c.Next()
		
		// Log request
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		
		logrus.Infof("%s | %3d | %12v | %s | %s",
			method,
			statusCode,
			latency,
			clientIP,
			path,
		)
	}
}

// isClusterWorker checks if a process name is a cluster worker
func isClusterWorker(name string) bool {
	return len(name) > 8 && name[len(name)-8:] == "-worker-"
}
