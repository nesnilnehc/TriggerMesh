package config

import (
	"os"
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
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// DatabaseConfig represents the database configuration
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// JenkinsConfig represents the Jenkins configuration
type JenkinsConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"` // Jenkins username (optional, defaults to token if not provided)
	Token    string `yaml:"token"`
	Timeout  int    `yaml:"timeout"` // Request timeout in seconds (default: 30)
}

// APIConfig represents the API configuration
type APIConfig struct {
	Keys []string `yaml:"keys"`
}

// Load loads the configuration from the given file path
func Load(filePath string) (*Config, error) {
	// Read the YAML file
	data, err := os.ReadFile(filePath) //nolint:gosec // Trusted file path input
	if err != nil {
		return nil, err
	}

	// Parse the YAML into the Config struct
	config := &Config{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	// Apply environment variables
	applyEnvVars(config)

	// Set default values if not provided
	setDefaults(config)

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

	// Jenkins configuration
	if url := os.Getenv("TRIGGERMESH_JENKINS_URL"); url != "" {
		config.Jenkins.URL = url
	}
	if username := os.Getenv("TRIGGERMESH_JENKINS_USERNAME"); username != "" {
		config.Jenkins.Username = username
	}
	if token := os.Getenv("TRIGGERMESH_JENKINS_TOKEN"); token != "" {
		config.Jenkins.Token = token
	}
	if timeout := os.Getenv("TRIGGERMESH_JENKINS_TIMEOUT"); timeout != "" {
		if t, err := strconv.Atoi(timeout); err == nil && t > 0 {
			config.Jenkins.Timeout = t
		}
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

	// Database defaults
	if config.Database.Path == "" {
		config.Database.Path = "./triggermesh.db"
	}

	// Jenkins defaults
	if config.Jenkins.Timeout == 0 {
		config.Jenkins.Timeout = 30 // 30 seconds default timeout
	}
	if config.Jenkins.Username == "" {
		// If username is not provided, use token as username (Jenkins API token authentication)
		config.Jenkins.Username = config.Jenkins.Token
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
