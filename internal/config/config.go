package config

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"

	yaml "gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Jenkins  JenkinsConfig  `yaml:"jenkins"`
	API      APIConfig      `yaml:"api"`
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	Port           int      `yaml:"port"`
	Host           string   `yaml:"host"`
	AllowedOrigins []string `yaml:"allowed_origins"` // Empty slice means allow all origins (default, for backward compatibility)
	MaxBodySize    int64    `yaml:"max_body_size"`   // Maximum request body size in bytes (default: 1MB)
}

// DatabaseConfig represents the database configuration
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// JenkinsConfig represents the Jenkins configuration
type JenkinsConfig struct {
	Default   string                   `yaml:"default"`
	Instances map[string]JenkinsConfig `yaml:"instances"`
	URL       string                   `yaml:"url"`
	Username  string                   `yaml:"username"` // Jenkins username (optional, defaults to token if not provided)
	Token     string                   `yaml:"token"`
	Timeout   int                      `yaml:"timeout"` // Request timeout in seconds (default: 30)
}

// APIConfig represents the API configuration
type APIConfig struct {
	Keys []string `yaml:"keys"`
}

var instanceNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Load loads the configuration from the given file path
func Load(filePath string) (*Config, error) {
	// Read the YAML file
	data, err := os.ReadFile(filePath) //nolint:gosec // Trusted file path input
	if err != nil {
		return nil, err
	}

	// Parse the YAML into the Config struct
	config := &Config{}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	// Apply environment variables
	applyEnvVars(config)

	// Set default values if not provided
	setDefaults(config)

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// applyEnvVars applies environment variables to the configuration
func applyEnvVars(config *Config) {
	// Server configuration
	if port := os.Getenv("TRIGGERMESH_SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}
	if host := os.Getenv("TRIGGERMESH_SERVER_HOST"); host != "" {
		config.Server.Host = host
	}

	// Database configuration
	if path := os.Getenv("TRIGGERMESH_DATABASE_PATH"); path != "" {
		config.Database.Path = path
	}

}

// setDefaults sets default values for the configuration
func setDefaults(config *Config) {
	// Server defaults
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.MaxBodySize == 0 {
		config.Server.MaxBodySize = 1 << 20 // 1MB default
	}

	// Database defaults
	if config.Database.Path == "" {
		config.Database.Path = "./triggermesh.db"
	}

	// Jenkins defaults
	for name, target := range config.Jenkins.Instances {
		if target.Timeout == 0 {
			target.Timeout = 30
		}
		if target.Username == "" {
			target.Username = target.Token
		}
		config.Jenkins.Instances[name] = target
	}
}

// GetLogLevel returns the log level from the environment
func GetLogLevel() string {
	levelStr := os.Getenv("TRIGGERMESH_LOG_LEVEL")
	if levelStr == "" {
		return "info"
	}

	// Validate log level
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if _, ok := validLevels[levelStr]; ok {
		return levelStr
	}

	return "info"
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	// Validate server port
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server.port: %d (must be between 1 and 65535)", cfg.Server.Port)
	}

	// Validate max body size
	if cfg.Server.MaxBodySize < 0 {
		return fmt.Errorf("invalid server.max_body_size: %d (must be non-negative)", cfg.Server.MaxBodySize)
	}
	if cfg.Server.MaxBodySize > 100<<20 { // 100MB max
		return fmt.Errorf("invalid server.max_body_size: %d (must be less than 100MB)", cfg.Server.MaxBodySize)
	}

	// Validate Jenkins configuration
	if cfg.Jenkins.URL != "" || cfg.Jenkins.Token != "" || cfg.Jenkins.Username != "" || cfg.Jenkins.Timeout != 0 {
		return fmt.Errorf("jenkins credentials must be configured under jenkins.instances")
	}
	if _, ok := cfg.Jenkins.Instances[cfg.Jenkins.Default]; !ok {
		return fmt.Errorf("jenkins.default must name an instance in jenkins.instances")
	}
	for name, instance := range cfg.Jenkins.Instances {
		if !instanceNamePattern.MatchString(name) {
			return fmt.Errorf("jenkins.instances name %q must contain only letters, numbers, underscores, or hyphens", name)
		}
		if err := validateJenkinsInstance("jenkins.instances["+name+"]", instance); err != nil {
			return err
		}
	}

	// Validate API keys
	if len(cfg.API.Keys) == 0 {
		return fmt.Errorf("at least one api.key is required")
	}
	for i, key := range cfg.API.Keys {
		if key == "" {
			return fmt.Errorf("api.keys[%d] cannot be empty", i)
		}
	}

	return nil
}

func validateJenkinsInstance(name string, instance JenkinsConfig) error {
	if instance.Default != "" || len(instance.Instances) != 0 {
		return fmt.Errorf("%s cannot contain nested instances", name)
	}
	parsed, err := url.Parse(instance.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("%s.url must be an HTTP(S) URL", name)
	}
	if instance.Token == "" {
		return fmt.Errorf("%s.token is required", name)
	}
	if instance.Timeout < 1 {
		return fmt.Errorf("%s.timeout must be positive", name)
	}
	return nil
}
