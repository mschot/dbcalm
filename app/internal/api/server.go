package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/martijn/dbcalm/internal/api/handler"
	"github.com/martijn/dbcalm/internal/api/middleware"
	"github.com/martijn/dbcalm/internal/core/repository"
	"github.com/martijn/dbcalm/internal/core/service"
	"github.com/martijn/dbcalm/pkg/config"
)

type Server struct {
	router *gin.Engine
	srv    *http.Server
	config *config.Config
}

// NewServer creates a new API server
func NewServer(
	cfg *config.Config,
	authService *service.AuthService,
	processService *service.ProcessService,
	backupService *service.BackupService,
	restoreService *service.RestoreService,
	scheduleService *service.ScheduleService,
	cleanupService *service.CleanupService,
	clientRepo repository.ClientRepository,
	scheduleRepo repository.ScheduleRepository,
	backupRepo repository.BackupRepository,
) *Server {
	// Set Gin mode
	if !cfg.IsDevMode() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.ErrorHandlerMiddleware())
	router.Use(middleware.CORSMiddleware(cfg.CORSOrigins))

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	backupHandler := handler.NewBackupHandler(backupService, scheduleRepo)
	restoreHandler := handler.NewRestoreHandler(restoreService, backupRepo)
	scheduleHandler := handler.NewScheduleHandler(scheduleService)
	processHandler := handler.NewProcessHandler(processService)
	clientHandler := handler.NewClientHandler(clientRepo, authService)
	cleanupHandler := handler.NewCleanupHandler(cleanupService)

	// Public routes (no auth required)
	auth := router.Group("/auth")
	{
		auth.POST("/authorize", authHandler.Authorize)
		auth.POST("/token", authHandler.Token)
	}

	// Protected routes (auth required)
	authMiddleware := middleware.AuthMiddleware(authService)

	// Backups
	backups := router.Group("/backups")
	backups.Use(authMiddleware)
	{
		backups.POST("", backupHandler.CreateBackup)
		backups.GET("", backupHandler.ListBackups)
		backups.GET("/:id", backupHandler.GetBackup)
	}

	// Restores
	restores := router.Group("/restores")
	restores.Use(authMiddleware)
	{
		restores.POST("", restoreHandler.CreateRestore)
		restores.GET("", restoreHandler.ListRestores)
		restores.GET("/:id", restoreHandler.GetRestore)
	}

	// Alternative restore endpoint (Python compatibility)
	router.POST("/restore", authMiddleware, restoreHandler.CreateRestore)

	// Schedules
	schedules := router.Group("/schedules")
	schedules.Use(authMiddleware)
	{
		schedules.POST("", scheduleHandler.CreateSchedule)
		schedules.GET("", scheduleHandler.ListSchedules)
		schedules.GET("/:id", scheduleHandler.GetSchedule)
		schedules.PUT("/:id", scheduleHandler.UpdateSchedule)
		schedules.DELETE("/:id", scheduleHandler.DeleteSchedule)
	}

	// Processes
	processes := router.Group("/processes")
	processes.Use(authMiddleware)
	{
		processes.GET("", processHandler.ListProcesses)
		processes.GET("/:id", processHandler.GetProcess)
	}

	// Process status by command ID
	router.GET("/status/:command_id", authMiddleware, processHandler.GetProcessByCommandID)

	// Clients
	clients := router.Group("/clients")
	clients.Use(authMiddleware)
	{
		clients.POST("", clientHandler.CreateClient)
		clients.GET("", clientHandler.ListClients)
		clients.GET("/:id", clientHandler.GetClient)
		clients.PUT("/:id", clientHandler.UpdateClient)
		clients.DELETE("/:id", clientHandler.DeleteClient)
	}

	// Cleanup
	router.POST("/cleanup", authMiddleware, cleanupHandler.Cleanup)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	server := &Server{
		router: router,
		config: cfg,
	}

	return server
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.APIHost, s.config.APIPort)

	s.srv = &http.Server{
		Addr:           addr,
		Handler:        s.router,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Start with or without SSL
	if s.config.SSLCert != "" && s.config.SSLKey != "" {
		fmt.Printf("Starting HTTPS server on %s\n", addr)
		return s.srv.ListenAndServeTLS(s.config.SSLCert, s.config.SSLKey)
	}

	fmt.Printf("Starting HTTP server on %s\n", addr)
	return s.srv.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.srv != nil {
		return s.srv.Shutdown(ctx)
	}
	return nil
}
