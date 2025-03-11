package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vektra/mockery/v2" // v2.20.0+
	"github.com/vektra/mockery/v2/pkg/config" // v2.20.0+
)

// interfacesToMock defines all the interfaces that need mock implementations
// generated for testing purposes
var interfacesToMock = []string{
	"DocumentRepository",
	"FolderRepository",
	"UserRepository",
	"TenantRepository",
	"TagRepository",
	"PermissionRepository",
	"WebhookRepository",
	"EventRepository",
	"DocumentService",
	"FolderService",
	"StorageService",
	"SearchService",
	"VirusScanningService",
	"ThumbnailService",
	"EventServiceInterface",
	"AuthService",
}

// configureMockery sets up mockery with appropriate configuration settings
// for the Document Management Platform project
func configureMockery() *config.Config {
	// Create a new configuration
	cfg := &config.Config{}
	
	// Get the project root directory
	rootDir, err := filepath.Abs("../../")
	if err != nil {
		fmt.Printf("Error finding project root: %v\n", err)
		os.Exit(1)
	}
	
	// Configure mockery settings
	cfg.Dir = rootDir
	cfg.Output = "../mocks"
	cfg.PackageName = "mocks"
	cfg.Recursive = true
	cfg.KeepTree = false
	cfg.FileName = "{{.InterfaceName}}.go"
	cfg.Exported = true
	cfg.InPackage = false
	cfg.All = false
	
	return cfg
}

// generateMocks runs mockery to generate mock implementations
// for all interfaces in the interfacesToMock list
func generateMocks(cfg *config.Config) error {
	totalGenerated := 0
	
	// Process each interface
	for _, interfaceName := range interfacesToMock {
		// Set the current interface to mock
		cfg.Name = interfaceName
		
		// Create a mockery instance
		m, err := mockery.New(
			mockery.WithConfig(cfg),
			mockery.WithName(interfaceName),
		)
		if err != nil {
			return fmt.Errorf("error creating mockery for %s: %w", interfaceName, err)
		}
		
		// Generate the mock implementation
		if err := m.Generate(); err != nil {
			return fmt.Errorf("error generating mock for %s: %w", interfaceName, err)
		}
		
		fmt.Printf("Generated mock for: %s\n", interfaceName)
		totalGenerated++
	}
	
	fmt.Printf("Successfully generated %d mocks\n", totalGenerated)
	return nil
}

// main is the entry point for the mock generation process
func main() {
	fmt.Println("Starting mock generation process...")
	
	// Configure mockery
	cfg := configureMockery()
	
	// Generate mocks
	if err := generateMocks(cfg); err != nil {
		fmt.Printf("Error generating mocks: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("Mock generation completed successfully")
}