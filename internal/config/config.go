package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/guessi/docker-parallel-pull/internal/security"
)

// Security constants
const (
	MaxConcurrency = 20               // Hard limit on concurrency
	MaxTimeout     = 30 * time.Minute // Maximum timeout
	MaxRetries     = 10               // Maximum retries
)

// Config holds all configuration options for the application
type Config struct {
	ContainerFile    string        `yaml:"container_file"`
	CleanupAfterTest bool          `yaml:"cleanup_after_test"`
	ShowPullDetail   bool          `yaml:"show_pull_detail"`
	MaxConcurrency   int           `yaml:"max_concurrency"`
	Timeout          time.Duration `yaml:"timeout"`
	MaxRetries       int           `yaml:"max_retries"`
	RetryDelay       time.Duration `yaml:"retry_delay"`
	ShowProgress     bool          `yaml:"show_progress"`
	OutputFormat     string        `yaml:"output_format"`
}

// LoadConfig loads configuration from a YAML file with defaults
func LoadConfig(filename string) (*Config, error) {
	config, err := LoadConfigFile(filename)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	// Set defaults for missing values
	if config.ContainerFile == "" {
		config.ContainerFile = "containers.yaml"
	}
	if config.MaxConcurrency == 0 {
		config.MaxConcurrency = 5
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 2 * time.Second
	}
	if config.OutputFormat == "" {
		config.OutputFormat = "text"
	}
	// ShowProgress and CleanupAfterTest default to true if not set
	// (YAML unmarshaling will set them to false if not specified)

	return config, nil
}

// LoadConfigFile loads configuration from a YAML file with security validation
func LoadConfigFile(filename string) (*Config, error) {
	data, err := security.SecureReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true) // Reject unknown fields for security

	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}



// Validate checks if the configuration parameters are valid and secure
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if c.ContainerFile == "" {
		return fmt.Errorf("container file path cannot be empty")
	}

	if err := security.ValidateFilePath(c.ContainerFile); err != nil {
		return fmt.Errorf("invalid container file path: %w", err)
	}

	if _, err := os.Stat(c.ContainerFile); os.IsNotExist(err) {
		return fmt.Errorf("container file does not exist: %s", security.SanitizeLogMessage(c.ContainerFile))
	}

	if c.MaxConcurrency <= 0 {
		return fmt.Errorf("max concurrency must be greater than 0, got: %d", c.MaxConcurrency)
	}

	if c.MaxConcurrency > MaxConcurrency {
		return fmt.Errorf("max concurrency too high (>%d), got: %d", MaxConcurrency, c.MaxConcurrency)
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0, got: %v", c.Timeout)
	}

	if c.Timeout > MaxTimeout {
		return fmt.Errorf("timeout too high (>%v), got: %v", MaxTimeout, c.Timeout)
	}

	if c.Timeout < 30*time.Second {
		return fmt.Errorf("timeout too short, minimum 30 seconds recommended, got: %v", c.Timeout)
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative, got: %d", c.MaxRetries)
	}

	if c.MaxRetries > MaxRetries {
		return fmt.Errorf("max retries too high (>%d), got: %d", MaxRetries, c.MaxRetries)
	}

	if c.RetryDelay < 0 {
		return fmt.Errorf("retry delay cannot be negative, got: %v", c.RetryDelay)
	}

	if c.OutputFormat != "text" && c.OutputFormat != "json" {
		return fmt.Errorf("output format must be 'text' or 'json', got: %s", c.OutputFormat)
	}

	return nil
}
