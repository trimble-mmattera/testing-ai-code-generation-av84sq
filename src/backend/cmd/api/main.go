// Package main is the entry point for the Document Management Platform API service.
package main

import (
	"context" // standard library
	"fmt"     // standard library
	"log"     // standard library
	"net/http" // standard library
	"os"      // standard library
	"os/signal" // standard library
	"syscall"   // standard library
	"time"      // standard library

	"src/backend/api/router" // For setting up API routes
	"src/backend/application/usecases" // For document use case implementation
	"src/backend/infrastructure/auth/jwt" // For JWT authentication
	"src/backend/infrastructure/persistence/postgres" // For database connection and management
	"src/backend/infrastructure/search/elasticsearch" // For Elasticsearch connection and search functionality
	"src/backend/infrastructure/storage/s3" // For S3 document storage
	"src/backend/pkg/config" // For loading and accessing application configuration
	"src/backend/pkg/logger" // For application logging
	"src/backend/pkg/metrics" // For application metrics collection
	documentrepo "src/backend/infrastructure/persistence/postgres"
	folderrepo "src/backend/infrastructure/persistence/postgres"
	searchusecase "src/backend/application/usecases"
	tenantrepo "src/backend/infrastructure/persistence/postgres"
	userrepo "src/backend/infrastructure/persistence/postgres"
	webhookrepo "src/backend/infrastructure/persistence/postgres"
	documentusecase "src/backend/application/usecases"
	folderusecase "src/backend/application/usecases"
	webhookusecase "src/backend/application/usecases"
)

func main() {
	// Load application configuration using config.Load
	var cfg config.Config
	if err := config.Load(&cfg); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger with configuration using logger.Init
	if err := logger.Init(cfg.Log); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Shutdown()

	// Initialize metrics collection using metrics.Init
	if err := metrics.Init(cfg.Metrics); err != nil {
		logger.Error("Failed to initialize metrics", "error", err)
	}

	// Initialize database connection using db.Init
	if err := postgres.Init(cfg.Database); err != nil {
		logger.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer postgres.Close()

	// Run database migrations using db.Migrate for all domain models
	if err := postgres.Migrate(
		&models.Document{},
		&models.DocumentMetadata{},
		&models.DocumentVersion{},
		&models.Folder{},
		&models.Permission{},
		&models.Tag{},
		&models.Tenant{},
		&models.User{},
		&models.Webhook{},
		&models.WebhookDelivery{},
	); err != nil {
		logger.Error("Failed to run database migrations", "error", err)
		os.Exit(1)
	}

	// Initialize Elasticsearch client using es.NewElasticsearchClient
	esClient, err := elasticsearch.NewElasticsearchClient(cfg.Elasticsearch)
	if err != nil {
		logger.Error("Failed to initialize Elasticsearch client", "error", err)
		os.Exit(1)
	}

	// Initialize document index using es.NewDocumentIndex
	docIndex, err := elasticsearch.NewDocumentIndex(esClient, cfg.Elasticsearch)
	if err != nil {
		logger.Error("Failed to initialize Elasticsearch document index", "error", err)
		os.Exit(1)
	}

	// Initialize S3 storage service using s3storage.NewS3Storage
	s3StorageService := s3storage.NewS3Storage(cfg.S3)

	// Initialize repositories (document, folder, user, tenant, webhook)
	documentRepo, err := documentrepo.NewDocumentRepository(postgres.GetDB())
	if err != nil {
		logger.Error("Failed to initialize document repository", "error", err)
		os.Exit(1)
	}

	folderRepo := folderrepo.NewFolderRepository(postgres.GetDB())
	userRepo, err := userrepo.NewUserRepository(postgres.GetDB())
	if err != nil {
		logger.Error("Failed to initialize user repository", "error", err)
		os.Exit(1)
	}

	tenantRepo := tenantrepo.NewTenantRepository(postgres.GetDB())
	webhookRepo := webhookrepo.NewWebhookRepository()

	// Initialize JWT authentication service using jwtauth.NewJWTService
	jwtService, err := jwt.NewJWTService(userRepo, tenantRepo, cfg.JWT)
	if err != nil {
		logger.Error("Failed to initialize JWT service", "error", err)
		os.Exit(1)
	}

	// Initialize use cases (document, folder, search, webhook)
	documentUseCase, err := documentusecase.NewDocumentUseCase(documentRepo, s3StorageService, nil, nil, folderRepo, nil, jwtService, nil)
	if err != nil {
		logger.Error("Failed to initialize document use case", "error", err)
		os.Exit(1)
	}

	folderUseCase := folderusecase.NewFolderUseCase(folderRepo, nil, nil, jwtService, nil)
	searchUseCase, err := searchusecase.NewSearchUseCase(nil, nil, documentRepo)
	if err != nil {
		logger.Error("Failed to initialize search use case", "error", err)
		os.Exit(1)
	}

	webhookUseCase, err := webhookusecase.NewWebhookUseCase(nil, nil)
	if err != nil {
		logger.Error("Failed to initialize webhook use case", "error", err)
		os.Exit(1)
	}

	// Set up API router with all routes and middleware using router.SetupRouter
	apiRouter := router.SetupRouter(
		cfg,
		documentUseCase,
		folderUseCase,
		searchUseCase,
		webhookUseCase,
		jwtService,
	)

	// Create HTTP server with configured timeouts and address
	httpServer := createHTTPServer(cfg, apiRouter)

	// Set up graceful shutdown with signal handling
	setupGracefulShutdown(httpServer)

	// Start HTTP server in a goroutine
	go func() {
		logger.Info("Starting HTTP server", "address", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server ListenAndServe error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	<-shutdownSignal

	// Perform graceful shutdown of HTTP server
	logger.Info("Shutting down HTTP server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	// Close database connection
	if err := postgres.Close(); err != nil {
		logger.Error("Database close error", "error", err)
	}

	// Shutdown logger
	logger.Info("Service shutdown complete")
}

var shutdownSignal chan os.Signal

// setupGracefulShutdown sets up graceful shutdown handling for the server
func setupGracefulShutdown(server *http.Server) {
	shutdownSignal = make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-shutdownSignal
		logger.Info("Shutdown signal received", "signal", sig)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("HTTP server shutdown error", "error", err)
		}

		logger.Info("HTTP server shutdown complete")
	}()
}

// createHTTPServer creates and configures the HTTP server
func createHTTPServer(cfg config.Config, handler http.Handler) *http.Server {
	serverAddress := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	readTimeout, err := time.ParseDuration(cfg.Server.ReadTimeout)
	if err != nil {
		logger.Error("Failed to parse read timeout", "error", err)
		readTimeout = 5 * time.Second // Default value
	}

	writeTimeout, err := time.ParseDuration(cfg.Server.WriteTimeout)
	if err != nil {
		logger.Error("Failed to parse write timeout", "error", err)
		writeTimeout = 10 * time.Second // Default value
	}

	idleTimeout, err := time.ParseDuration(cfg.Server.IdleTimeout)
	if err != nil {
		logger.Error("Failed to parse idle timeout", "error", err)
		idleTimeout = 30 * time.Second // Default value
	}

	return &http.Server{
		Addr:         serverAddress,
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
}