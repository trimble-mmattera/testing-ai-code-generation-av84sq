package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"../../pkg/config"
	"../../pkg/logger"
	"../../pkg/metrics"
	"../../infrastructure/messaging/sqs/sqsclient"
	"../../infrastructure/messaging/sqs/documentqueue"
	"../../infrastructure/virus_scanning/clamav"
	"../../infrastructure/virus_scanning/clamav/virusscanner"
	"../../infrastructure/storage/s3/s3storage"
	"../../infrastructure/messaging/sns/eventpublisher"
)

// Number of documents to process in a batch
const batchSize = 10

// Time to wait between processing batches
const processingInterval = 5 * time.Second

// Timeout duration for graceful shutdown
const shutdownTimeout = 30 * time.Second

func main() {
	// Load application configuration
	var cfg config.Config
	err := config.Load(&cfg)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger with configuration
	err = logger.Init(logger.LogConfig{
		Level:         cfg.Log.Level,
		Format:        cfg.Log.Format,
		Output:        cfg.Log.Output,
		EnableConsole: cfg.Log.EnableConsole,
		EnableFile:    cfg.Log.EnableFile,
		FilePath:      cfg.Log.FilePath,
	})
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Shutdown()

	// Initialize metrics collection
	err = metrics.Init(metrics.MetricsConfig{
		Enabled:         true,
		EnableEndpoint:  true,
		EndpointAddress: "0.0.0.0",
		EndpointPort:    9090,
		Namespace:       "document_mgmt",
	})
	if err != nil {
		logger.Error("Failed to initialize metrics", "error", err)
		os.Exit(1)
	}
	defer metrics.Shutdown()

	// Log worker startup
	logger.Info("Document scanning worker starting up", "version", "1.0.0")

	// Initialize SQS client
	sqsClient, err := sqsclient.NewSQSClient(context.Background(), cfg.SQS)
	if err != nil {
		logger.Error("Failed to initialize SQS client", "error", err)
		os.Exit(1)
	}

	// Initialize document scan queue
	scanQueue, err := documentqueue.NewDocumentScanQueue(context.Background(), sqsClient, cfg)
	if err != nil {
		logger.Error("Failed to initialize document scan queue", "error", err)
		os.Exit(1)
	}

	// Initialize ClamAV client
	clamAVClient, err := clamav.NewClamAVClient(fmt.Sprintf("%s:%d", cfg.ClamAV.Host, cfg.ClamAV.Port))
	if err != nil {
		logger.Error("Failed to initialize ClamAV client", "error", err)
		os.Exit(1)
	}

	// Initialize S3 storage service
	storageService := s3storage.NewS3Storage(cfg.S3)
	if storageService == nil {
		logger.Error("Failed to initialize S3 storage service")
		os.Exit(1)
	}

	// Initialize event publisher
	eventPublisher, err := eventpublisher.NewEventPublisher(context.Background(), cfg.SNS)
	if err != nil {
		logger.Error("Failed to initialize event publisher", "error", err)
		os.Exit(1)
	}

	// Initialize virus scanner service
	virusScanner, err := virusscanner.NewVirusScanner(clamAVClient, scanQueue, storageService, eventPublisher, cfg)
	if err != nil {
		logger.Error("Failed to initialize virus scanner service", "error", err)
		os.Exit(1)
	}

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Set up signal handling for graceful shutdown
	setupSignalHandling(cancel)

	// Start the main processing loop
	logger.Info("Starting document processing loop", "batch_size", batchSize)
	go processDocuments(ctx, virusScanner)

	// Wait for shutdown signal
	<-ctx.Done()

	// Perform graceful shutdown
	gracefulShutdown(context.Background())
}

// setupSignalHandling sets up signal handling for graceful shutdown
func setupSignalHandling(cancel context.CancelFunc) {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-signalCh
		logger.Info("Received shutdown signal", "signal", sig.String())
		cancel()
	}()
}

// processDocuments is the main processing loop for scanning documents
func processDocuments(ctx context.Context, scanner virusscanner.VirusScanningService) {
	for {
		// Process the scan queue with the specified batch size
		count, err := scanner.ProcessScanQueue(ctx, batchSize)
		if err != nil {
			logger.Error("Error processing scan queue", "error", err)
		} else {
			logger.Info("Processed documents from queue", "count", count)
		}

		// Sleep for the processing interval
		select {
		case <-time.After(processingInterval):
			// Continue processing after interval
		case <-ctx.Done():
			// Context is cancelled, exit the loop
			logger.Info("Stopping document processing")
			return
		}
	}
}

// gracefulShutdown performs graceful shutdown of worker components
func gracefulShutdown(ctx context.Context) {
	// Create a context with timeout for shutdown operations
	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	logger.Info("Shutting down worker", "timeout", shutdownTimeout)

	// Shutdown metrics collection
	if err := metrics.Shutdown(); err != nil {
		logger.Error("Error shutting down metrics", "error", err)
	}

	// Shutdown logger
	if err := logger.Shutdown(); err != nil {
		// Can't log this error since logger is being shut down
		fmt.Printf("Error shutting down logger: %v\n", err)
	}

	fmt.Println("Worker shutdown complete")
}