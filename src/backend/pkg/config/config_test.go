// Package config provides tests for the configuration package
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert" // v1.8.0+
	"github.com/stretchr/testify/require" // v1.8.0+

	"../errors"
	"../utils"
)

const testConfigDir = "./testdata"

// TestConfig is a test configuration struct for testing
type TestConfig struct {
	TestString string
	TestInt    int
	TestBool   bool
	Nested     TestNestedConfig
}

// TestNestedConfig is a nested configuration struct for testing
type TestNestedConfig struct {
	NestedString string
	NestedInt    int
}

// TestLoad tests the Load function with various scenarios
func TestLoad(t *testing.T) {
	// Set up test environment with temporary config files
	configDir := setupTestConfig(t)
	defer cleanupTestConfig(t, configDir)

	// Test loading valid configuration
	os.Setenv("ENV", "development")
	var cfg TestConfig
	err := Load(&cfg, configDir, "")
	require.NoError(t, err)
	assert.Equal(t, "development-value", cfg.TestString)
	assert.Equal(t, 42, cfg.TestInt)
	assert.True(t, cfg.TestBool)
	assert.Equal(t, "nested-dev", cfg.Nested.NestedString)
	assert.Equal(t, 123, cfg.Nested.NestedInt)

	// Test loading with environment variable overrides
	os.Setenv("TEST_STRING", "env-override")
	os.Setenv("TEST_INT", "100")
	os.Setenv("TEST_BOOL", "false")
	os.Setenv("NESTED_NESTED_STRING", "nested-env-override")
	
	cfg = TestConfig{}
	err = Load(&cfg, configDir, "")
	require.NoError(t, err)
	assert.Equal(t, "env-override", cfg.TestString)
	assert.Equal(t, 100, cfg.TestInt)
	assert.False(t, cfg.TestBool)
	assert.Equal(t, "nested-env-override", cfg.Nested.NestedString)

	// Clean up environment variables
	os.Unsetenv("TEST_STRING")
	os.Unsetenv("TEST_INT")
	os.Unsetenv("TEST_BOOL")
	os.Unsetenv("NESTED_NESTED_STRING")

	// Test loading with command-line flag overrides
	flags := map[string]string{
		"test-string": "flag-override",
		"test-int": "200",
		"test-bool": "true",
	}
	cfg = TestConfig{}
	err = LoadFromFlags(&cfg, flags)
	require.NoError(t, err)
	assert.Equal(t, "flag-override", cfg.TestString)
	assert.Equal(t, 200, cfg.TestInt)
	assert.True(t, cfg.TestBool)

	// Test loading with invalid configuration (validation errors)
	invalidConfigDir := filepath.Join(configDir, "invalid")
	require.NoError(t, os.MkdirAll(invalidConfigDir, 0755))
	
	invalidContent := `
testString: 
testInt: not-an-int
testBool: not-a-bool
`
	invalidPath := filepath.Join(invalidConfigDir, "development.yml")
	require.NoError(t, os.WriteFile(invalidPath, []byte(invalidContent), 0644))
	
	cfg = TestConfig{}
	err = Load(&cfg, invalidConfigDir, "")
	assert.Error(t, err)
	assert.True(t, errors.IsValidationError(err))

	// Test loading with non-existent files
	os.Setenv("ENV", "nonexistent")
	cfg = TestConfig{}
	err = Load(&cfg, configDir, "")
	assert.Error(t, err)
	
	// Clean up
	os.Unsetenv("ENV")
}

// TestLoadFromFile tests the LoadFromFile function
func TestLoadFromFile(t *testing.T) {
	// Set up test environment with temporary config files
	configDir := setupTestConfig(t)
	defer cleanupTestConfig(t, configDir)

	// Test loading from a valid YAML file
	var cfg TestConfig
	err := LoadFromFile(&cfg, filepath.Join(configDir, "development.yml"))
	require.NoError(t, err)
	assert.Equal(t, "development-value", cfg.TestString)
	assert.Equal(t, 42, cfg.TestInt)
	assert.True(t, cfg.TestBool)

	// Test loading from a non-existent file
	cfg = TestConfig{}
	err = LoadFromFile(&cfg, filepath.Join(configDir, "nonexistent.yml"))
	assert.Error(t, err)

	// Test loading from an invalid YAML file
	invalidFilePath := filepath.Join(configDir, "invalid.yml")
	err = os.WriteFile(invalidFilePath, []byte("this is not valid yaml:::::"), 0644)
	require.NoError(t, err)

	cfg = TestConfig{}
	err = LoadFromFile(&cfg, invalidFilePath)
	assert.Error(t, err)
}

// TestLoadFromEnv tests the LoadFromEnv function
func TestLoadFromEnv(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_STRING", "env-value")
	os.Setenv("TEST_INT", "123")
	os.Setenv("TEST_BOOL", "true")
	os.Setenv("NESTED_NESTED_STRING", "nested-env-value")
	os.Setenv("NESTED_NESTED_INT", "456")

	// Test loading configuration from environment variables
	var cfg TestConfig
	err := LoadFromEnv(&cfg)
	require.NoError(t, err)
	assert.Equal(t, "env-value", cfg.TestString)
	assert.Equal(t, 123, cfg.TestInt)
	assert.True(t, cfg.TestBool)
	assert.Equal(t, "nested-env-value", cfg.Nested.NestedString)
	assert.Equal(t, 456, cfg.Nested.NestedInt)

	// Test with various data types (string, int, bool)
	os.Setenv("TEST_STRING", "another string")
	os.Setenv("TEST_INT", "789")
	os.Setenv("TEST_BOOL", "false")
	
	cfg = TestConfig{}
	err = LoadFromEnv(&cfg)
	require.NoError(t, err)
	assert.Equal(t, "another string", cfg.TestString)
	assert.Equal(t, 789, cfg.TestInt)
	assert.False(t, cfg.TestBool)

	// Test with nested struct fields
	os.Setenv("NESTED_NESTED_STRING", "new nested value")
	os.Setenv("NESTED_NESTED_INT", "999")
	
	cfg = TestConfig{}
	err = LoadFromEnv(&cfg)
	require.NoError(t, err)
	assert.Equal(t, "new nested value", cfg.Nested.NestedString)
	assert.Equal(t, 999, cfg.Nested.NestedInt)

	// Test with invalid values
	os.Setenv("TEST_INT", "not-an-int")
	cfg = TestConfig{}
	err = LoadFromEnv(&cfg)
	assert.Error(t, err)

	// Clean up environment variables
	os.Unsetenv("TEST_STRING")
	os.Unsetenv("TEST_INT")
	os.Unsetenv("TEST_BOOL")
	os.Unsetenv("NESTED_NESTED_STRING")
	os.Unsetenv("NESTED_NESTED_INT")
}

// TestLoadFromFlags tests the LoadFromFlags function
func TestLoadFromFlags(t *testing.T) {
	// Set up test command-line arguments
	flags := map[string]string{
		"test-string": "flag-value",
		"test-int": "789",
		"test-bool": "true",
		"nested.nested-string": "nested-flag-value",
		"nested.nested-int": "987",
	}

	// Test loading configuration from command-line flags
	var cfg TestConfig
	err := LoadFromFlags(&cfg, flags)
	require.NoError(t, err)
	assert.Equal(t, "flag-value", cfg.TestString)
	assert.Equal(t, 789, cfg.TestInt)
	assert.True(t, cfg.TestBool)
	assert.Equal(t, "nested-flag-value", cfg.Nested.NestedString)
	assert.Equal(t, 987, cfg.Nested.NestedInt)

	// Test with various data types (string, int, bool)
	flags = map[string]string{
		"test-string": "another flag value",
		"test-int": "321",
		"test-bool": "false",
	}
	
	cfg = TestConfig{}
	err = LoadFromFlags(&cfg, flags)
	require.NoError(t, err)
	assert.Equal(t, "another flag value", cfg.TestString)
	assert.Equal(t, 321, cfg.TestInt)
	assert.False(t, cfg.TestBool)

	// Test with nested struct fields
	flags = map[string]string{
		"nested.nested-string": "new nested flag value",
		"nested.nested-int": "654",
	}
	
	cfg = TestConfig{}
	err = LoadFromFlags(&cfg, flags)
	require.NoError(t, err)
	assert.Equal(t, "new nested flag value", cfg.Nested.NestedString)
	assert.Equal(t, 654, cfg.Nested.NestedInt)

	// Test with invalid values
	flags = map[string]string{
		"test-int": "not-an-int",
	}
	
	cfg = TestConfig{}
	err = LoadFromFlags(&cfg, flags)
	assert.Error(t, err)
}

// TestValidate tests the Validate function
func TestValidate(t *testing.T) {
	// Create test configurations with valid and invalid values
	validCfg := TestConfig{
		TestString: "valid",
		TestInt:    42,
		TestBool:   true,
		Nested: TestNestedConfig{
			NestedString: "valid-nested",
			NestedInt:    123,
		},
	}

	// Test validation of valid configuration
	err := Validate(&validCfg)
	assert.NoError(t, err)

	// Test validation of configuration with missing required fields
	invalidCfg := TestConfig{
		TestString: "", // Assuming this is required
		TestInt:    -1, // Assuming this should be positive
	}

	err = Validate(&invalidCfg)
	assert.Error(t, err)
	assert.True(t, errors.IsValidationError(err))

	// Test validation of configuration with invalid field values
	invalidFieldCfg := TestConfig{
		TestString: "valid",
		TestInt:    -10, // Assuming positive values are required
		TestBool:   true,
	}

	err = Validate(&invalidFieldCfg)
	assert.Error(t, err)
	assert.True(t, errors.IsValidationError(err))

	// Verify that validation errors are properly returned
	cfg := TestConfig{
		TestInt: -5,
	}
	err = Validate(&cfg)
	assert.Error(t, err)
	assert.True(t, errors.IsValidationError(err))
}

// TestGetConfigFilePath tests the GetConfigFilePath function
func TestGetConfigFilePath(t *testing.T) {
	// Test with all parameters provided
	path := GetConfigFilePath("development", "/config", "config.yml")
	assert.Equal(t, "/config/development.yml", path)

	// Test with empty environment (should use DefaultEnv)
	path = GetConfigFilePath("", "/config", "config.yml")
	assert.Equal(t, "/config/development.yml", path) // Assuming DefaultEnv is "development"

	// Test with empty config directory (should use DefaultConfigPath)
	path = GetConfigFilePath("production", "", "config.yml")
	assert.Equal(t, "config/production.yml", path) // Assuming DefaultConfigPath is "config"

	// Verify correct path construction in all cases
	path = GetConfigFilePath("staging", "/custom/path", "settings.yml")
	assert.Equal(t, "/custom/path/staging.yml", path)
}

// TestGetEnv tests the getEnv function
func TestGetEnv(t *testing.T) {
	// Set up test environment with ENV variable
	os.Setenv("ENV", "production")

	// Test getting environment when ENV is set
	env := getEnv()
	assert.Equal(t, "production", env)

	// Unset ENV variable
	os.Unsetenv("ENV")

	// Test getting environment when ENV is not set (should return DefaultEnv)
	env = getEnv()
	assert.Equal(t, "development", env) // Assuming DefaultEnv is "development"

	// Clean up environment variables
	os.Unsetenv("ENV")
}

// TestSetField tests the setField function
func TestSetField(t *testing.T) {
	// Create test struct with various field types
	cfg := TestConfig{
		TestString: "original",
		TestInt:    42,
		TestBool:   false,
		Nested: TestNestedConfig{
			NestedString: "original-nested",
			NestedInt:    123,
		},
	}

	// Test setting string field
	err := setField(&cfg, "TestString", "new-value")
	require.NoError(t, err)
	assert.Equal(t, "new-value", cfg.TestString)

	// Test setting int field
	err = setField(&cfg, "TestInt", "99")
	require.NoError(t, err)
	assert.Equal(t, 99, cfg.TestInt)

	// Test setting bool field
	err = setField(&cfg, "TestBool", "true")
	require.NoError(t, err)
	assert.True(t, cfg.TestBool)

	// Test setting nested struct field
	err = setField(&cfg, "Nested.NestedString", "new-nested-value")
	require.NoError(t, err)
	assert.Equal(t, "new-nested-value", cfg.Nested.NestedString)

	// Test setting field with invalid value type
	err = setField(&cfg, "TestInt", "not-an-int")
	assert.Error(t, err)

	// Test setting non-existent field
	err = setField(&cfg, "NonExistentField", "value")
	assert.Error(t, err)
}

// setupTestConfig is a helper function to set up test configuration files
func setupTestConfig(t *testing.T) string {
	// Create temporary directory for test config files
	tempDir, err := os.MkdirTemp("", "config-test")
	require.NoError(t, err)

	// Create default.yml with basic configuration
	defaultContent := `
testString: default-value
testInt: 0
testBool: false
nested:
  nestedString: nested-default
  nestedInt: 0
`
	err = os.WriteFile(filepath.Join(tempDir, "default.yml"), []byte(defaultContent), 0644)
	require.NoError(t, err)

	// Create development.yml with development-specific configuration
	developmentContent := `
testString: development-value
testInt: 42
testBool: true
nested:
  nestedString: nested-dev
  nestedInt: 123
`
	err = os.WriteFile(filepath.Join(tempDir, "development.yml"), []byte(developmentContent), 0644)
	require.NoError(t, err)

	// Create staging.yml with staging-specific configuration
	stagingContent := `
testString: staging-value
testInt: 84
testBool: true
nested:
  nestedString: nested-staging
  nestedInt: 456
`
	err = os.WriteFile(filepath.Join(tempDir, "staging.yml"), []byte(stagingContent), 0644)
	require.NoError(t, err)

	// Create production.yml with production-specific configuration
	productionContent := `
testString: production-value
testInt: 100
testBool: false
nested:
  nestedString: nested-prod
  nestedInt: 789
`
	err = os.WriteFile(filepath.Join(tempDir, "production.yml"), []byte(productionContent), 0644)
	require.NoError(t, err)

	return tempDir
}

// cleanupTestConfig is a helper function to clean up test configuration files
func cleanupTestConfig(t *testing.T, configDir string) {
	// Remove temporary test config directory and all files
	err := os.RemoveAll(configDir)
	require.NoError(t, err)
}