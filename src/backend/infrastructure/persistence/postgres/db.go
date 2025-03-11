// Package postgres provides database connection management for PostgreSQL in the Document Management Platform.
// It implements connection initialization, configuration, pooling, and metrics tracking for database operations,
// ensuring proper resource management and monitoring.
package postgres

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm" // v1.25.0+
	"gorm.io/driver/postgres" // v1.5.0+

	"../../../pkg/config"  // For database configuration settings
	"../../../pkg/logger"  // For logging database operations
	"../../../pkg/metrics" // For tracking database performance
	"../../../pkg/errors"  // For standardized error handling
)

var (
	// instance is the singleton database connection
	instance *gorm.DB
	
	// mu is used for thread-safe database initialization
	mu sync.Mutex
	
	// initialized indicates if the database has been initialized
	initialized bool
	
	// queryHistogram tracks database query execution times
	queryHistogram *prometheus.HistogramVec
	
	// connectionGauge tracks active database connections
	connectionGauge *prometheus.GaugeVec
)

// Init initializes the database connection with the provided configuration
func Init(dbConfig config.DatabaseConfig) error {
	mu.Lock()
	defer mu.Unlock()

	if initialized {
		return nil
	}

	// Build DSN from config
	dsn := buildDSN(dbConfig)

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: NewGormLogger(),
	})
	if err != nil {
		return errors.NewDependencyError(fmt.Sprintf("failed to connect to database: %v", err))
	}

	// Get underlying SQL DB to configure pool
	sqlDB, err := db.DB()
	if err != nil {
		return errors.NewDependencyError(fmt.Sprintf("failed to get database connection: %v", err))
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)
	sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)
	
	// Parse connection max lifetime from string
	connMaxLifetime, err := time.ParseDuration(dbConfig.ConnMaxLifetime)
	if err != nil {
		connMaxLifetime = 1 * time.Hour // Default to 1 hour if parsing fails
	}
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	// Register metrics
	registerMetrics()

	// Set the global instance
	instance = db
	initialized = true

	logger.Info("Database initialized successfully", 
		"host", dbConfig.Host, 
		"port", dbConfig.Port, 
		"database", dbConfig.DBName)

	return nil
}

// GetDB returns the database connection instance
func GetDB() (*gorm.DB, error) {
	if !initialized {
		return nil, errors.NewDependencyError("database not initialized")
	}
	return instance, nil
}

// Close closes the database connection and releases resources
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if !initialized {
		return nil
	}

	sqlDB, err := instance.DB()
	if err != nil {
		return errors.NewDependencyError(fmt.Sprintf("failed to get database connection: %v", err))
	}

	err = sqlDB.Close()
	if err != nil {
		return errors.NewDependencyError(fmt.Sprintf("failed to close database connection: %v", err))
	}

	initialized = false
	logger.Info("Database connection closed")
	return nil
}

// WithTransaction executes a function within a database transaction
func WithTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.NewDependencyError(fmt.Sprintf("failed to begin transaction: %v", tx.Error))
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // Re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return errors.NewDependencyError(fmt.Sprintf("failed to commit transaction: %v", err))
	}

	return nil
}

// Migrate runs database migrations to ensure schema is up to date
func Migrate(models ...interface{}) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	logger.Info("Running database migrations")
	err = db.AutoMigrate(models...)
	if err != nil {
		return errors.NewDependencyError(fmt.Sprintf("failed to run migrations: %v", err))
	}
	
	logger.Info("Database migrations completed successfully")
	return nil
}

// registerMetrics registers database-related metrics with Prometheus
func registerMetrics() {
	// Query execution time histogram
	queryHistogram = metrics.RegisterCustomHistogram(
		"db_query_duration_seconds",
		"Database query execution time in seconds",
		[]string{"operation"},
		[]float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5},
	)

	// Active connections gauge
	connectionGauge = metrics.RegisterCustomGauge(
		"db_connections",
		"Database connections",
		[]string{"state"},
	)
	
	// Update metrics periodically
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		
		for range ticker.C {
			if initialized {
				updateMetrics()
			}
		}
	}()
}

// updateMetrics updates database metrics with current values
func updateMetrics() {
	if !initialized {
		return
	}

	sqlDB, err := instance.DB()
	if err != nil {
		logger.Error("Failed to get database stats", "error", err)
		return
	}

	stats := sqlDB.Stats()
	connectionGauge.WithLabelValues("open").Set(float64(stats.OpenConnections))
	connectionGauge.WithLabelValues("idle").Set(float64(stats.Idle))
	connectionGauge.WithLabelValues("in_use").Set(float64(stats.InUse))
	connectionGauge.WithLabelValues("wait_count").Set(float64(stats.WaitCount))
	connectionGauge.WithLabelValues("wait_duration").Set(float64(stats.WaitDuration.Seconds()))
}

// buildDSN builds a PostgreSQL connection string from configuration
func buildDSN(config config.DatabaseConfig) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.DBName,
		config.SSLMode,
	)
}

// GormLogger is a custom logger for GORM that integrates with our application logging
type GormLogger struct {
	LogLevel      gorm.LogLevel
	SlowThreshold time.Duration
}

// NewGormLogger creates a new GormLogger instance
func NewGormLogger() *GormLogger {
	return &GormLogger{
		LogLevel:      gorm.InfoLevel,
		SlowThreshold: 200 * time.Millisecond,
	}
}

// LogMode sets the log level for the logger
func (l *GormLogger) LogMode(level gorm.LogLevel) gorm.Logger {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info logs info messages from GORM
func (l *GormLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gorm.InfoLevel {
		logger.InfoContext(ctx, fmt.Sprintf(msg, args...))
	}
}

// Warn logs warning messages from GORM
func (l *GormLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gorm.WarnLevel {
		logger.WarnContext(ctx, fmt.Sprintf(msg, args...))
	}
}

// Error logs error messages from GORM
func (l *GormLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gorm.ErrorLevel {
		logger.ErrorContext(ctx, fmt.Sprintf(msg, args...))
	}
}

// Trace logs SQL queries with execution time
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gorm.Silent {
		return
	}
	
	elapsed := time.Since(begin)
	
	// Record query duration in metrics
	if queryHistogram != nil {
		queryHistogram.WithLabelValues("query").Observe(elapsed.Seconds())
	}
	
	sql, rows := fc()
	
	// Log based on error and query time
	if err != nil {
		logger.ErrorContext(ctx, "Database error",
			"error", err,
			"elapsed", elapsed.String(),
			"rows", rows,
			"sql", sql,
		)
		return
	}
	
	if elapsed > l.SlowThreshold && l.LogLevel >= gorm.WarnLevel {
		logger.WarnContext(ctx, "Slow SQL query",
			"elapsed", elapsed.String(),
			"rows", rows,
			"sql", sql,
		)
		return
	}
	
	if l.LogLevel >= gorm.InfoLevel {
		logger.DebugContext(ctx, "SQL query",
			"elapsed", elapsed.String(),
			"rows", rows,
			"sql", sql,
		)
	}
}