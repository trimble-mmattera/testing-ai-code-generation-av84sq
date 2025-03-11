// Package config provides configuration management for the Document Management Platform.
// It handles loading configuration from multiple sources including YAML files, environment
// variables, and command-line flags, with support for environment-specific configurations
// and validation.
package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3" // v3.0.1

	"../errors"   // For creating standardized validation errors
	"../utils"    // For file existence checks
	"../validator" // For validating configuration values
)

// Default configuration constants
const (
	// DefaultConfigPath is the default path for configuration files
	DefaultConfigPath = "./config"
	
	// DefaultEnv is the default environment to use if not specified
	DefaultEnv = "development"
	
	// EnvPrefix is the prefix for environment variables
	EnvPrefix = "DMP_"
)

// Config is the main configuration struct that holds all application configurations
type Config struct {
	// Env represents the current environment (development, staging, production)
	Env string

	// Server configuration for the HTTP server
	Server ServerConfig

	// Database configuration for PostgreSQL
	Database DatabaseConfig

	// S3 configuration for AWS S3 document storage
	S3 S3Config

	// Elasticsearch configuration for document search
	Elasticsearch ElasticsearchConfig

	// JWT configuration for authentication
	JWT JWTConfig

	// Log configuration for application logging
	Log LogConfig

	// ClamAV configuration for virus scanning
	ClamAV ClamAVConfig

	// SQS configuration for AWS SQS message queues
	SQS SQSConfig

	// SNS configuration for AWS SNS event publishing
	SNS SNSConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	// Host to bind the server to
	Host string

	// Port to listen on
	Port int

	// ReadTimeout for HTTP requests
	ReadTimeout string

	// WriteTimeout for HTTP responses
	WriteTimeout string

	// IdleTimeout for keep-alive connections
	IdleTimeout string

	// TLS enables HTTPS if true
	TLS bool

	// CertFile path for TLS certificate
	CertFile string

	// KeyFile path for TLS private key
	KeyFile string
}

// DatabaseConfig holds PostgreSQL database configuration
type DatabaseConfig struct {
	// Host of the database server
	Host string

	// Port of the database server
	Port int

	// User for database authentication
	User string

	// Password for database authentication
	Password string

	// DBName is the database name
	DBName string

	// SSLMode for database connection
	SSLMode string

	// MaxOpenConns is the maximum number of open connections
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections
	MaxIdleConns int

	// ConnMaxLifetime is the maximum lifetime of a connection
	ConnMaxLifetime string
}

// S3Config holds AWS S3 configuration for document storage
type S3Config struct {
	// Region is the AWS region
	Region string

	// Endpoint is the S3 endpoint URL (for custom endpoints)
	Endpoint string

	// AccessKey for S3 authentication
	AccessKey string

	// SecretKey for S3 authentication
	SecretKey string

	// Bucket is the main bucket for document storage
	Bucket string

	// TempBucket is the bucket for temporary document storage
	TempBucket string

	// QuarantineBucket is the bucket for quarantined documents
	QuarantineBucket string

	// UseSSL enables SSL for S3 connections
	UseSSL bool

	// ForcePathStyle enables path-style S3 URLs
	ForcePathStyle bool
}

// ElasticsearchConfig holds Elasticsearch configuration for document search
type ElasticsearchConfig struct {
	// Addresses is a list of Elasticsearch nodes
	Addresses []string

	// Username for Elasticsearch authentication
	Username string

	// Password for Elasticsearch authentication
	Password string

	// EnableSniff enables sniffing for Elasticsearch nodes
	EnableSniff bool

	// IndexPrefix is the prefix for Elasticsearch indices
	IndexPrefix string
}

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	// Secret is the JWT signing secret (for HMAC algorithms)
	Secret string

	// PublicKey is the path to the RSA public key for verification
	PublicKey string

	// PrivateKey is the path to the RSA private key for signing
	PrivateKey string

	// Issuer is the JWT issuer claim
	Issuer string

	// ExpirationTime is the JWT expiration time
	ExpirationTime string

	// Algorithm is the JWT signing algorithm (HS256, RS256, etc.)
	Algorithm string
}

// LogConfig holds logging configuration
type LogConfig struct {
	// Level is the log level (debug, info, warn, error)
	Level string

	// Format is the log format (json, text)
	Format string

	// Output is the log output destination (stdout, file)
	Output string

	// EnableConsole enables logging to console
	EnableConsole bool

	// EnableFile enables logging to file
	EnableFile bool

	// FilePath is the path for log files
	FilePath string
}

// ClamAVConfig holds ClamAV virus scanning configuration
type ClamAVConfig struct {
	// Host of the ClamAV server
	Host string

	// Port of the ClamAV server
	Port int

	// Timeout for scan operations in seconds
	Timeout int
}

// SQSConfig holds AWS SQS configuration for message queues
type SQSConfig struct {
	// Region is the AWS region
	Region string

	// Endpoint is the SQS endpoint URL (for custom endpoints)
	Endpoint string

	// AccessKey for SQS authentication
	AccessKey string

	// SecretKey for SQS authentication
	SecretKey string

	// DocumentQueueURL is the URL for document processing queue
	DocumentQueueURL string

	// ScanQueueURL is the URL for virus scanning queue
	ScanQueueURL string

	// IndexQueueURL is the URL for document indexing queue
	IndexQueueURL string

	// UseSSL enables SSL for SQS connections
	UseSSL bool
}

// SNSConfig holds AWS SNS configuration for event publishing
type SNSConfig struct {
	// Region is the AWS region
	Region string

	// Endpoint is the SNS endpoint URL (for custom endpoints)
	Endpoint string

	// AccessKey for SNS authentication
	AccessKey string

	// SecretKey for SNS authentication
	SecretKey string

	// DocumentTopicARN is the ARN for document events topic
	DocumentTopicARN string

	// EventTopicARN is the ARN for general events topic
	EventTopicARN string

	// UseSSL enables SSL for SNS connections
	UseSSL bool
}

// Load loads the configuration from all sources
func Load(cfg interface{}) error {
	// Ensure cfg is a pointer to a struct
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.NewValidationError("config must be a pointer to a struct")
	}

	// Get the environment
	env := getEnv()

	// Load default configuration
	defaultConfigPath := GetConfigFilePath(DefaultConfigPath, "", "default.yml")
	if utils.IsFile(defaultConfigPath) {
		if err := LoadFromFile(defaultConfigPath, cfg); err != nil {
			return err
		}
	}

	// Load environment-specific configuration
	envConfigPath := GetConfigFilePath(DefaultConfigPath, env, env+".yml")
	if utils.IsFile(envConfigPath) {
		if err := LoadFromFile(envConfigPath, cfg); err != nil {
			return err
		}
	}

	// Override with environment variables
	if err := LoadFromEnv(cfg, EnvPrefix); err != nil {
		return err
	}

	// Override with command-line flags
	if err := LoadFromFlags(cfg); err != nil {
		return err
	}

	// Validate configuration
	return Validate(cfg)
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(filePath string, cfg interface{}) error {
	// Check if file exists
	if !utils.IsFile(filePath) {
		return errors.NewResourceNotFoundError("configuration file not found: " + filePath)
	}

	// Read file content
	data, err := utils.ReadFileToBytes(filePath)
	if err != nil {
		return errors.NewInternalError("failed to read configuration file: " + err.Error())
	}

	// Unmarshal YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return errors.NewValidationError("failed to parse configuration file: " + err.Error())
	}

	return nil
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv(cfg interface{}, prefix string) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.NewValidationError("config must be a pointer to a struct")
	}
	v = v.Elem()

	// Get all environment variables
	envVars := os.Environ()

	// Process environment variables
	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Only process variables with the correct prefix
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		// Remove prefix and convert to struct field name
		fieldPath := strings.TrimPrefix(key, prefix)
		fieldPath = strings.Replace(fieldPath, "_", ".", -1)

		// Set the field value
		if err := setNestedField(v, fieldPath, value); err != nil {
			return err
		}
	}

	return nil
}

// LoadFromFlags loads configuration from command-line flags
func LoadFromFlags(cfg interface{}) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.NewValidationError("config must be a pointer to a struct")
	}
	v = v.Elem()

	// Define a map to hold flag values
	flagValues := make(map[string]interface{})

	// Use reflection to define flags
	defineFlags(v, "", flagValues)

	// Parse flags
	flag.Parse()

	// Set struct fields from flag values
	for path, value := range flagValues {
		if err := setNestedField(v, path, value); err != nil {
			return err
		}
	}

	return nil
}

// defineFlags recursively defines command-line flags based on struct fields
func defineFlags(v reflect.Value, prefix string, flagValues map[string]interface{}) {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		fieldName := field.Name
		flagName := strings.ToLower(fieldName)
		if prefix != "" {
			flagName = prefix + "." + flagName
		}

		// Handle nested structs
		if fieldValue.Kind() == reflect.Struct {
			defineFlags(fieldValue, flagName, flagValues)
			continue
		}

		// Define flags based on field type
		switch fieldValue.Kind() {
		case reflect.String:
			ptr := flag.String(flagName, fieldValue.String(), "")
			flagValues[flagName] = ptr
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			ptr := flag.Int(flagName, int(fieldValue.Int()), "")
			flagValues[flagName] = ptr
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			ptr := flag.Uint(flagName, uint(fieldValue.Uint()), "")
			flagValues[flagName] = ptr
		case reflect.Float32, reflect.Float64:
			ptr := flag.Float64(flagName, fieldValue.Float(), "")
			flagValues[flagName] = ptr
		case reflect.Bool:
			ptr := flag.Bool(flagName, fieldValue.Bool(), "")
			flagValues[flagName] = ptr
		case reflect.Slice:
			// For string slices only
			if fieldValue.Type().Elem().Kind() == reflect.String {
				var strVal string
				if fieldValue.Len() > 0 {
					strSlice := make([]string, fieldValue.Len())
					for j := 0; j < fieldValue.Len(); j++ {
						strSlice[j] = fieldValue.Index(j).String()
					}
					strVal = strings.Join(strSlice, ",")
				}
				ptr := flag.String(flagName, strVal, "")
				flagValues[flagName] = ptr
			}
		}
	}
}

// setNestedField sets a nested field value in a struct
func setNestedField(v reflect.Value, path string, value interface{}) error {
	parts := strings.Split(path, ".")
	current := v

	// Navigate to the parent struct
	for i := 0; i < len(parts)-1; i++ {
		fieldName := parts[i]
		// Convert to exported field name (capitalize first letter)
		if len(fieldName) > 0 {
			fieldName = strings.ToUpper(fieldName[:1]) + fieldName[1:]
		}

		field := current.FieldByName(fieldName)
		if !field.IsValid() {
			return nil // Field doesn't exist, silently ignore
		}

		if field.Kind() != reflect.Struct {
			return errors.NewValidationError("path refers to non-struct field: " + path)
		}

		current = field
	}

	// Get the final field
	fieldName := parts[len(parts)-1]
	// Convert to exported field name
	if len(fieldName) > 0 {
		fieldName = strings.ToUpper(fieldName[:1]) + fieldName[1:]
	}

	field := current.FieldByName(fieldName)
	if !field.IsValid() {
		return nil // Field doesn't exist, silently ignore
	}

	if !field.CanSet() {
		return errors.NewValidationError("field cannot be set: " + path)
	}

	// Set field value based on the type of value
	switch v := value.(type) {
	case *string:
		if v != nil && field.Kind() == reflect.String {
			field.SetString(*v)
		}
	case *int:
		if v != nil && field.Kind() == reflect.Int {
			field.SetInt(int64(*v))
		}
	case *uint:
		if v != nil && field.Kind() == reflect.Uint {
			field.SetUint(uint64(*v))
		}
	case *float64:
		if v != nil && field.Kind() == reflect.Float64 {
			field.SetFloat(*v)
		}
	case *bool:
		if v != nil && field.Kind() == reflect.Bool {
			field.SetBool(*v)
		}
	case string:
		return setField(field, fieldName, v)
	default:
		// Unknown type
		return errors.NewValidationError("unsupported value type")
	}

	return nil
}

// setField sets a field value from a string with type conversion
func setField(v reflect.Value, name string, value string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return errors.NewValidationError(fmt.Sprintf("invalid integer value for %s: %s", name, value))
		}
		v.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return errors.NewValidationError(fmt.Sprintf("invalid unsigned integer value for %s: %s", name, value))
		}
		v.SetUint(i)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return errors.NewValidationError(fmt.Sprintf("invalid float value for %s: %s", name, value))
		}
		v.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return errors.NewValidationError(fmt.Sprintf("invalid boolean value for %s: %s", name, value))
		}
		v.SetBool(b)
	case reflect.Slice:
		// Handle string slices
		if v.Type().Elem().Kind() == reflect.String {
			parts := strings.Split(value, ",")
			slice := reflect.MakeSlice(v.Type(), len(parts), len(parts))
			for i, part := range parts {
				slice.Index(i).SetString(strings.TrimSpace(part))
			}
			v.Set(slice)
		} else {
			return errors.NewValidationError(fmt.Sprintf("unsupported slice type for %s", name))
		}
	default:
		return errors.NewValidationError(fmt.Sprintf("unsupported field type for %s", name))
	}

	return nil
}

// Validate validates the configuration
func Validate(cfg interface{}) error {
	return validator.Validate(cfg)
}

// GetConfigFilePath gets the configuration file path
func GetConfigFilePath(configDir, env, fileName string) string {
	if env == "" {
		env = DefaultEnv
	}

	if configDir == "" {
		configDir = DefaultConfigPath
	}

	return filepath.Join(configDir, fileName)
}

// getEnv gets the current environment
func getEnv() string {
	env := os.Getenv("ENV")
	if env == "" {
		return DefaultEnv
	}
	return env
}