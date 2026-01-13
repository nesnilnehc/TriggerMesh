package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"triggermesh/internal/api"
	"triggermesh/internal/config"
	"triggermesh/internal/engine/jenkins"
	"triggermesh/internal/logger"
	"triggermesh/internal/storage"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to the configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	loggerLevel := config.GetLogLevel()
	logger.Init(loggerLevel)
	logger.Info("Starting TriggerMesh service", "log_level", loggerLevel)

	// Initialize database
	if err := storage.Init(cfg.Database.Path); err != nil {
		logger.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer storage.Close()

	// Initialize Jenkins client and engine
	jenkinsClient := jenkins.NewClient(cfg.Jenkins)
	jenkinsEngine := jenkins.NewTrigger(jenkinsClient)

	// Initialize router
	router := api.NewRouter(*cfg, jenkinsEngine)

	// Read PORT from environment variable if set
	port := cfg.Server.Port
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil && p > 0 {
			port = p
		}
	}

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, port),
		Handler: router,
	}

	// Start the server in a goroutine
	go func() {
		logger.Info("Server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create a context with timeout for graceful shutdown
	// Use 30 seconds for production to allow long-running requests to complete
	shutdownTimeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	logger.Info("Initiating graceful shutdown", "timeout", shutdownTimeout.String())

	// Shutdown the server gracefully
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err, "timeout", shutdownTimeout.String())
	} else {
		logger.Info("Server shutdown gracefully")
	}

	// Close the database connection
	if err := storage.Close(); err != nil {
		logger.Error("Failed to close database connection", "error", err)
	}

	logger.Info("Server stopped")
}
