// Package main is the entry point for the Document Management Platform.
// It delegates to the appropriate service (API or worker) based on command-line arguments.
// This file serves as a unified entry point that can start either the API server or the background
// worker process depending on the provided command.
package main

import (
	"flag" // standard library - For parsing command-line flags
	"fmt"  // standard library - For formatted output
	"os"   // standard library - For accessing command-line arguments and environment

	"src/backend/cmd/api"    // For starting the API server
	"src/backend/cmd/worker" // For starting the worker process
	"src/backend/pkg/config" // For loading application configuration
	"src/backend/pkg/logger" // For application logging
)

// version is the application version
const version = "1.0.0"

// main is the entry point for the Document Management Platform
func main() {
	// Define command-line flags for service type (api or worker)
	var serviceType string
	flag.StringVar(&serviceType, "service", "api", "Service type (api or worker)")

	// Parse command-line flags
	flag.Parse()

	// Load common configuration
	var cfg config.Config
	if err := config.Load(&cfg); err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init(cfg.Log); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Shutdown()

	// Log application startup with version information
	logger.Info("Starting Document Management Platform", "version", version)

	// Determine which service to start based on service type flag
	switch serviceType {
	case "api":
		// If service type is 'api', call api.main()
		logger.Info("Starting API service")
		api.Main()
	case "worker":
		// If service type is 'worker', call worker.main()
		logger.Info("Starting worker service")
		worker.Main()
	case "version":
		// If service type is 'version', print version information
		printVersion()
	default:
		// If service type is invalid, log error and exit with non-zero status
		logger.Error("Invalid service type", "serviceType", serviceType)
		fmt.Println("Invalid service type. Use 'api' or 'worker'.")
		os.Exit(1)
	}
}

// printVersion prints the application version information
func printVersion() {
	// Print the application name and version
	fmt.Printf("Document Management Platform\nVersion: %s\n", version)

	// Print build information (if available)
	// In a real-world scenario, this would include build date, git commit, etc.
	fmt.Println("Build Information: N/A")
}