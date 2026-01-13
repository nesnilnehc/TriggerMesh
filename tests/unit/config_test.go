package unit

import (
	"os"
	"strings"
	"testing"

	"triggermesh/internal/config"
)

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
	configContent := `
jenkins:
  url: https://test-jenkins.example.com
  token: test-token

api:
  keys:
    - test-api-key
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
	// Set environment variables
	os.Setenv("TRIGGERMESH_SERVER_PORT", "9090")
	os.Setenv("TRIGGERMESH_JENKINS_URL", "https://env-jenkins.example.com")
	defer os.Unsetenv("TRIGGERMESH_SERVER_PORT")
	defer os.Unsetenv("TRIGGERMESH_JENKINS_URL")

	// Create a minimal config file
	configContent := `
jenkins:
  url: https://test-jenkins.example.com
  token: test-token

api:
  keys:
    - test-api-key
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

	// Verify environment variables override config
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090 from env var, got %d", cfg.Server.Port)
	}
	if cfg.Jenkins.URL != "https://env-jenkins.example.com" {
		t.Errorf("Expected Jenkins URL from env var, got %s", cfg.Jenkins.URL)
	}
}

func TestGetLogLevel(t *testing.T) {
	// Test default log level
	os.Unsetenv("TRIGGERMESH_LOG_LEVEL")
	level := config.GetLogLevel()
	if level != "info" {
		t.Errorf("Expected default log level info, got %s", level)
	}

	// Test valid log levels
	validLevels := []string{"debug", "info", "warn", "error"}
	for _, validLevel := range validLevels {
		os.Setenv("TRIGGERMESH_LOG_LEVEL", validLevel)
		lvl := config.GetLogLevel()
		if lvl != validLevel {
			t.Errorf("Expected log level %s, got %s", validLevel, lvl)
		}
		os.Unsetenv("TRIGGERMESH_LOG_LEVEL")
	}

	// Test invalid log level (should default to info)
	os.Setenv("TRIGGERMESH_LOG_LEVEL", "invalid")
	level = config.GetLogLevel()
	if level != "info" {
		t.Errorf("Expected log level info for invalid value, got %s", level)
	}
	os.Unsetenv("TRIGGERMESH_LOG_LEVEL")
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
