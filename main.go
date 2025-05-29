/*
Package main is the entry point for the Skynet Agent application.

This package initializes and starts the Skynet Agent server, which provides
AI-powered agent capabilities through a REST API. The server is built using
the Echo web framework and includes proper configuration loading, logging,
graceful shutdown, and error handling.

The application follows these initialization steps:
1. Load configuration from environment variables and files
2. Initialize structured logging
3. Create the core server instance with dependencies
4. Set up HTTP middleware (logging, recovery, CORS)
5. Register API routes
6. Start the server with graceful shutdown support

Author: Skynet Agent Team
*/
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"skynet/core"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// main is the application entry point that initializes and starts the Skynet Agent server.
// It handles the complete lifecycle of the application including:
// - Configuration loading
// - Dependency initialization
// - HTTP server setup
// - Graceful shutdown on interrupt signals
func main() {
	// Load configuration from environment variables and config files
	config := core.LoadConfig()

	// Initialize structured logger with the loaded configuration
	logger := core.InitializeLogger(config)
	logger.Info("Starting Skynet Agent server")

	// Create the core server instance with all dependencies
	server, err := core.NewServer(config, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create server")
	}

	// Create Echo web framework instance
	e := echo.New()

	// Configure middleware stack for request processing
	e.Use(middleware.Logger())  // HTTP request logging
	e.Use(middleware.Recover()) // Panic recovery
	e.Use(middleware.CORS())    // Cross-Origin Resource Sharing

	// Register all API routes and handlers
	server.RegisterRoutes(e)

	// Start the HTTP server in a separate goroutine to allow for graceful shutdown
	go func() {
		logger.WithField("port", config.Port).Info("Starting server")
		if err := e.Start(fmt.Sprintf(":%s", config.Port)); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Set up graceful shutdown handling
	// Create a channel to receive OS interrupt signals
	quit := make(chan os.Signal, 1)
	// Register the channel to receive specific signals (SIGINT, SIGTERM)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	// Block until a signal is received
	<-quit

	logger.Info("Shutting down server...")

	// Create a context with timeout for graceful shutdown
	// This gives the server 30 seconds to finish processing ongoing requests
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := e.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Failed to gracefully shutdown server")
	} else {
		logger.Info("Server shutdown complete")
	}
}
