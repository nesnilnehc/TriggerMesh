package unit

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"triggermesh/internal/config"
)

// testMinimalConfigContent is a shared minimal configuration content used across
// multiple test functions to avoid duplication (DRY principle). This constant
// contains the minimum required fields for a valid TriggerMesh configuration.
const testMinimalConfigContent = `
jenkins:
  url: https://test-jenkins.example.com
  token: test-token

api:
  keys:
    - test-api-key
`

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	configContent := `
server:
  port: 8080
  host: "0.0.0.0"

database:
  path: ./test.db

jenkins:
  url: https://test-jenkins.example.com
  token: test-token
  timeout: 30

api:
  keys:
    - test-api-key-1
    - test-api-key-2
`

	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, writeErr := tmpFile.WriteString(configContent); writeErr != nil {
		t.Fatalf("Failed to write config: %v", writeErr)
	}
	tmpFile.Close()

	// Load the config
	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify server config
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected host 0.0.0.0, got %s", cfg.Server.Host)
	}

	// Verify database config
	if cfg.Database.Path != "./test.db" {
		t.Errorf("Expected database path ./test.db, got %s", cfg.Database.Path)
	}

	// Verify Jenkins config
	if cfg.Jenkins.URL != "https://test-jenkins.example.com" {
		t.Errorf("Expected Jenkins URL https://test-jenkins.example.com, got %s", cfg.Jenkins.URL)
	}
	if cfg.Jenkins.Token != "test-token" {
		t.Errorf("Expected Jenkins token test-token, got %s", cfg.Jenkins.Token)
	}
	if cfg.Jenkins.Timeout != 30 {
		t.Errorf("Expected Jenkins timeout 30, got %d", cfg.Jenkins.Timeout)
	}

	// Verify API config
	if len(cfg.API.Keys) != 2 {
		t.Errorf("Expected 2 API keys, got %d", len(cfg.API.Keys))
	}
	if cfg.API.Keys[0] != "test-api-key-1" {
		t.Errorf("Expected first API key test-api-key-1, got %s", cfg.API.Keys[0])
	}
}

func TestConfigDefaults(t *testing.T) {
	// Create a minimal config file
	configContent := testMinimalConfigContent

	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, writeErr := tmpFile.WriteString(configContent); writeErr != nil {
		t.Fatalf("Failed to write config: %v", writeErr)
	}
	tmpFile.Close()

	// Load the config
	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify defaults
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default host 0.0.0.0, got %s", cfg.Server.Host)
	}
	if cfg.Database.Path != "./triggermesh.db" {
		t.Errorf("Expected default database path ./triggermesh.db, got %s", cfg.Database.Path)
	}
	if cfg.Jenkins.Timeout != 30 {
		t.Errorf("Expected default Jenkins timeout 30, got %d", cfg.Jenkins.Timeout)
	}
	// Username should default to token if not provided
	if cfg.Jenkins.Username != cfg.Jenkins.Token {
		t.Errorf("Expected Jenkins username to default to token, got %s", cfg.Jenkins.Username)
	}
}

func TestConfigEnvVars(t *testing.T) {
	// Set environment variables (t.Setenv automatically cleans up after test)
	t.Setenv("TRIGGERMESH_SERVER_PORT", "9090")
	t.Setenv("TRIGGERMESH_JENKINS_URL", "https://env-jenkins.example.com")

	// Create a minimal config file
	configContent := testMinimalConfigContent

	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, writeErr := tmpFile.WriteString(configContent); writeErr != nil {
		t.Fatalf("Failed to write config: %v", writeErr)
	}
	tmpFile.Close()

	// Load the config
	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify environment variables override config
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090 from env var, got %d", cfg.Server.Port)
	}
	if cfg.Jenkins.URL != "https://env-jenkins.example.com" {
		t.Errorf("Expected Jenkins URL from env var, got %s", cfg.Jenkins.URL)
	}
}

func TestGetLogLevel(t *testing.T) {
	// Test default log level (unset env var)
	t.Setenv("TRIGGERMESH_LOG_LEVEL", "")
	level := config.GetLogLevel()
	if level != "info" {
		t.Errorf("Expected default log level info, got %s", level)
	}

	// Test valid log levels
	validLevels := []string{"debug", "info", "warn", "error"}
	for _, validLevel := range validLevels {
		t.Setenv("TRIGGERMESH_LOG_LEVEL", validLevel)
		lvl := config.GetLogLevel()
		if lvl != validLevel {
			t.Errorf("Expected log level %s, got %s", validLevel, lvl)
		}
	}

	// Test invalid log level (should default to info)
	t.Setenv("TRIGGERMESH_LOG_LEVEL", "invalid")
	level = config.GetLogLevel()
	if level != "info" {
		t.Errorf("Expected log level info for invalid value, got %s", level)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		expectError   bool
		errorContains string
	}{
		{
			name: "Valid config",
			configContent: `
jenkins:
  url: https://test-jenkins.example.com
  token: test-token
api:
  keys:
    - test-api-key
`,
			expectError: false,
		},
		{
			name: "Missing Jenkins URL",
			configContent: `
jenkins:
  token: test-token
api:
  keys:
    - test-api-key
`,
			expectError:   true,
			errorContains: "jenkins.url is required",
		},
		{
			name: "Missing Jenkins Token",
			configContent: `
jenkins:
  url: https://test-jenkins.example.com
api:
  keys:
    - test-api-key
`,
			expectError:   true,
			errorContains: "jenkins.token is required",
		},
		{
			name: "Invalid Jenkins URL",
			configContent: `
jenkins:
  url: "://invalid-url"
  token: test-token
api:
  keys:
    - test-api-key
`,
			expectError:   true,
			errorContains: "invalid jenkins.url",
		},
		{
			name: "Missing API Keys",
			configContent: `
jenkins:
  url: https://test-jenkins.example.com
  token: test-token
api:
  keys: []
`,
			expectError:   true,
			errorContains: "at least one api.key is required",
		},
		{
			name: "Invalid Port",
			configContent: `
server:
  port: 70000
jenkins:
  url: https://test-jenkins.example.com
  token: test-token
api:
  keys:
    - test-api-key
`,
			expectError:   true,
			errorContains: "invalid server.port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "config-validation-test-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, writeErr := tmpFile.WriteString(tt.configContent); writeErr != nil {
				t.Fatalf("Failed to write config: %v", writeErr)
			}
			tmpFile.Close()

			cfg, err := config.Load(tmpFile.Name())
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if cfg == nil {
					t.Error("Config should not be nil")
				}
			}
		})
	}
}

func TestConfigEnvVarsAll(t *testing.T) {
	// Test all environment variables (t.Setenv automatically cleans up after test)
	t.Setenv("TRIGGERMESH_SERVER_PORT", "9090")
	t.Setenv("TRIGGERMESH_SERVER_HOST", "127.0.0.1")
	t.Setenv("TRIGGERMESH_DATABASE_PATH", "/tmp/test.db")
	t.Setenv("TRIGGERMESH_JENKINS_URL", "https://env-jenkins.example.com")
	t.Setenv("TRIGGERMESH_JENKINS_USERNAME", "env-user")
	t.Setenv("TRIGGERMESH_JENKINS_TOKEN", "env-token")
	t.Setenv("TRIGGERMESH_JENKINS_TIMEOUT", "60")

	// Create a minimal config file
	configContent := testMinimalConfigContent

	tmpFile, err := os.CreateTemp("", "config-env-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, writeErr := tmpFile.WriteString(configContent); writeErr != nil {
		t.Fatalf("Failed to write config: %v", writeErr)
	}
	tmpFile.Close()

	// Load the config
	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify environment variables override config
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090 from env var, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Expected host 127.0.0.1 from env var, got %s", cfg.Server.Host)
	}
	if cfg.Database.Path != "/tmp/test.db" {
		t.Errorf("Expected database path /tmp/test.db from env var, got %s", cfg.Database.Path)
	}
	if cfg.Jenkins.URL != "https://env-jenkins.example.com" {
		t.Errorf("Expected Jenkins URL from env var, got %s", cfg.Jenkins.URL)
	}
	if cfg.Jenkins.Username != "env-user" {
		t.Errorf("Expected Jenkins username from env var, got %s", cfg.Jenkins.Username)
	}
	if cfg.Jenkins.Token != "env-token" {
		t.Errorf("Expected Jenkins token from env var, got %s", cfg.Jenkins.Token)
	}
	if cfg.Jenkins.Timeout != 60 {
		t.Errorf("Expected Jenkins timeout 60 from env var, got %d", cfg.Jenkins.Timeout)
	}
}

func TestConfigValidationMaxBodySize(t *testing.T) {
	tests := []struct {
		name          string
		maxBodySize   int64
		expectError   bool
		errorContains string
	}{
		{
			name:        "Valid max body size",
			maxBodySize: 10 * 1024 * 1024, // 10MB
			expectError: false,
		},
		{
			name:          "Negative max body size",
			maxBodySize:   -1,
			expectError:   true,
			errorContains: "invalid server.max_body_size",
		},
		{
			name:          "Too large max body size",
			maxBodySize:   200 * 1024 * 1024, // 200MB, exceeds 100MB limit
			expectError:   true,
			errorContains: "invalid server.max_body_size",
		},
		{
			name:        "Zero max body size (valid)",
			maxBodySize: 0,
			expectError: false, // Will use default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configContent := fmt.Sprintf(`
server:
  max_body_size: %d
jenkins:
  url: https://test-jenkins.example.com
  token: test-token
api:
  keys:
    - test-api-key
`, tt.maxBodySize)

			tmpFile, err := os.CreateTemp("", "config-maxbody-test-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, writeErr := tmpFile.WriteString(configContent); writeErr != nil {
				t.Fatalf("Failed to write config: %v", writeErr)
			}
			tmpFile.Close()

			cfg, err := config.Load(tmpFile.Name())
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if cfg != nil && tt.maxBodySize > 0 && cfg.Server.MaxBodySize != tt.maxBodySize {
					t.Errorf("Expected max body size %d, got %d", tt.maxBodySize, cfg.Server.MaxBodySize)
				}
			}
		})
	}
}

func TestConfigValidationEmptyAPIKey(t *testing.T) {
	configContent := `
jenkins:
  url: https://test-jenkins.example.com
  token: test-token
api:
  keys:
    - test-api-key
    - ""  # Empty key should fail validation
`

	tmpFile, err := os.CreateTemp("", "config-empty-key-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, writeErr := tmpFile.WriteString(configContent); writeErr != nil {
		t.Fatalf("Failed to write config: %v", writeErr)
	}
	tmpFile.Close()

	_, err = config.Load(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for empty API key, got nil")
	} else if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("Expected error about empty API key, got %q", err.Error())
	}
}

func TestConfigDefaultsMaxBodySize(t *testing.T) {
	// Test that max body size defaults to 1MB
	configContent := `
jenkins:
  url: https://test-jenkins.example.com
  token: test-token
api:
  keys:
    - test-api-key
`

	tmpFile, err := os.CreateTemp("", "config-defaults-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, writeErr := tmpFile.WriteString(configContent); writeErr != nil {
		t.Fatalf("Failed to write config: %v", writeErr)
	}
	tmpFile.Close()

	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// MaxBodySize should default to 1MB (1 << 20)
	expectedMaxBodySize := int64(1 << 20)
	if cfg.Server.MaxBodySize != expectedMaxBodySize {
		t.Errorf("Expected default max body size %d, got %d", expectedMaxBodySize, cfg.Server.MaxBodySize)
	}
}

func TestConfigEnvVarsInvalidValues(t *testing.T) {
	// Test that invalid environment variable values are ignored
	t.Setenv("TRIGGERMESH_SERVER_PORT", "invalid-port")
	t.Setenv("TRIGGERMESH_JENKINS_TIMEOUT", "invalid-timeout")

	configContent := `
server:
  port: 8080
jenkins:
  url: https://test-jenkins.example.com
  token: test-token
  timeout: 30
api:
  keys:
    - test-api-key
`

	tmpFile, err := os.CreateTemp("", "config-invalid-env-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, writeErr := tmpFile.WriteString(configContent); writeErr != nil {
		t.Fatalf("Failed to write config: %v", writeErr)
	}
	tmpFile.Close()

	// Should load successfully, invalid env vars should be ignored
	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Should use config file values, not invalid env vars
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected port 8080 from config file (invalid env var ignored), got %d", cfg.Server.Port)
	}
	if cfg.Jenkins.Timeout != 30 {
		t.Errorf("Expected timeout 30 from config file (invalid env var ignored), got %d", cfg.Jenkins.Timeout)
	}
}

func TestConfigEnvVarsNegativeTimeout(t *testing.T) {
	// Test that negative timeout is ignored
	t.Setenv("TRIGGERMESH_JENKINS_TIMEOUT", "-10")

	configContent := `
jenkins:
  url: https://test-jenkins.example.com
  token: test-token
  timeout: 30
api:
  keys:
    - test-api-key
`

	tmpFile, err := os.CreateTemp("", "config-negative-timeout-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, writeErr := tmpFile.WriteString(configContent); writeErr != nil {
		t.Fatalf("Failed to write config: %v", writeErr)
	}
	tmpFile.Close()

	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Negative timeout should be ignored, use config file value
	if cfg.Jenkins.Timeout != 30 {
		t.Errorf("Expected timeout 30 from config file (negative env var ignored), got %d", cfg.Jenkins.Timeout)
	}
}

func TestConfigLoadFileNotFound(t *testing.T) {
	// Test loading non-existent config file
	// Use a path that's guaranteed not to exist
	nonexistentPath := "/tmp/triggermesh-test-nonexistent-" + t.Name() + ".yaml"
	_, err := config.Load(nonexistentPath)
	if err == nil {
		t.Error("Expected error loading non-existent config file, got nil")
	}
}

func TestConfigLoadInvalidYAML(t *testing.T) {
	// Test loading invalid YAML file
	invalidYAML := `invalid: yaml: content: [`

	tmpFile, err := os.CreateTemp("", "config-invalid-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, writeErr := tmpFile.WriteString(invalidYAML); writeErr != nil {
		t.Fatalf("Failed to write config: %v", writeErr)
	}
	tmpFile.Close()

	_, err = config.Load(tmpFile.Name())
	if err == nil {
		t.Error("Expected error loading invalid YAML, got nil")
	}
}
