package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/a8m/envsubst"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// InitLogger initializes the logger with default settings
func InitLogger() {
	// Set default log level from environment variable or default to info
	logLevel := os.Getenv("QUAY_LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// Set output format to JSON for better parsing
	log.SetFormatter(&logrus.JSONFormatter{})
}

// Config represents the root configuration structure
type Config struct {
	Projects []Project `yaml:"projects"`
}

// Project represents a single project configuration
type Project struct {
	Name        string            `yaml:"name"`
	Path        string            `yaml:"path"`
	ComposeFile string            `yaml:"compose_file"`
	Context     string            `yaml:"context,omitempty"`
	Ports       []PortMapping     `yaml:"ports,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	EnvFile     string            `yaml:"env_file,omitempty"`
	Ingress     *Ingress          `yaml:"ingress,omitempty"`
	Remote      *Remote           `yaml:"remote,omitempty"`
}

// PortMapping represents a port mapping between host and container
type PortMapping struct {
	Service   string `yaml:"service"`
	Host      string `yaml:"host"`
	Container string `yaml:"container"`
}

// Ingress represents ingress configuration for a project
type Ingress struct {
	Enabled  bool     `yaml:"enabled"`
	Hostname string   `yaml:"hostname"`
	Paths    []string `yaml:"paths"`
	SSL      *SSL     `yaml:"ssl,omitempty"`
}

// SSL represents SSL configuration for ingress
type SSL struct {
	Provider string `yaml:"provider"`
}

// Remote represents remote repository configuration
type Remote struct {
	Repo   string `yaml:"repo"`
	Branch string `yaml:"branch"`
}

// substituteEnvVars performs environment variable substitution on a string
// using the ${VAR:-default} syntax. If VAR is not set, default is used.
func substituteEnvVars(value string) (string, error) {
	log.WithField("value", value).Debug("Substituting environment variables")
	result, err := envsubst.String(value)
	if err != nil {
		log.WithError(err).WithField("value", value).Error("Failed to substitute environment variables")
		return "", err
	}
	log.WithFields(logrus.Fields{
		"original": value,
		"result":   result,
	}).Debug("Environment variable substitution completed")
	return result, nil
}

// processEnvVars performs environment variable substitution on all environment
// variables in the project configuration.
func (p *Project) processEnvVars() error {
	if p.Env == nil {
		log.WithField("project", p.Name).Debug("No environment variables to process")
		return nil
	}

	log.WithFields(logrus.Fields{
		"project": p.Name,
		"count":   len(p.Env),
	}).Debug("Processing environment variables")

	// Create a new map to store processed values
	processedEnv := make(map[string]string, len(p.Env))

	// Process each environment variable
	for key, value := range p.Env {
		processed, err := substituteEnvVars(value)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"project": p.Name,
				"key":     key,
			}).Error("Failed to substitute environment variables")
			return fmt.Errorf("failed to substitute environment variables for key %q: %w", key, err)
		}
		processedEnv[key] = processed
	}

	// Replace the original map with processed values
	p.Env = processedEnv
	log.WithField("project", p.Name).Debug("Environment variable processing completed")
	return nil
}

// loadEnvFile reads and parses a .env file into a map of environment variables.
// It returns an error if the file cannot be read or if the format is invalid.
func loadEnvFile(filePath string) (map[string]string, error) {
	log.WithField("file", filePath).Debug("Loading .env file")

	// Read the .env file
	envMap, err := godotenv.Read(filePath)
	if err != nil {
		log.WithError(err).WithField("file", filePath).Error("Failed to read .env file")
		return nil, fmt.Errorf("failed to read .env file %s: %w", filePath, err)
	}

	log.WithFields(logrus.Fields{
		"file":  filePath,
		"count": len(envMap),
	}).Debug("Successfully loaded .env file")
	return envMap, nil
}

// processEnvFile loads and merges environment variables from the specified .env file
// with the project's existing environment variables.
func (p *Project) processEnvFile() error {
	if p.EnvFile == "" {
		log.WithField("project", p.Name).Debug("No .env file specified")
		return nil
	}

	log.WithFields(logrus.Fields{
		"project": p.Name,
		"envFile": p.EnvFile,
	}).Debug("Processing .env file")

	// Resolve the .env file path relative to the project path
	envFilePath := p.EnvFile
	if !filepath.IsAbs(envFilePath) && p.Path != "" {
		envFilePath = filepath.Join(p.Path, envFilePath)
	}

	// Load environment variables from the .env file
	envMap, err := loadEnvFile(envFilePath)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"project": p.Name,
			"file":    envFilePath,
		}).Error("Failed to load .env file")
		return fmt.Errorf("failed to load .env file for project %q: %w", p.Name, err)
	}

	// Initialize the project's environment map if it's nil
	if p.Env == nil {
		p.Env = make(map[string]string)
	}

	// Merge environment variables, with project's env taking precedence
	overrideCount := 0
	for key, value := range envMap {
		if _, exists := p.Env[key]; !exists {
			p.Env[key] = value
		} else {
			overrideCount++
		}
	}

	log.WithFields(logrus.Fields{
		"project":       p.Name,
		"totalVars":     len(envMap),
		"overrides":     overrideCount,
		"finalVarCount": len(p.Env),
	}).Debug("Successfully merged environment variables")
	return nil
}

// validateProject validates a project configuration and returns an error if any required fields are missing
func (p *Project) validate() error {
	if p.Name == "" {
		return fmt.Errorf("project name is required")
	}
	if p.Path == "" {
		return fmt.Errorf("project path is required")
	}
	if p.ComposeFile == "" {
		return fmt.Errorf("compose_file is required")
	}

	// Validate ingress configuration if enabled
	if p.Ingress != nil && p.Ingress.Enabled {
		if p.Ingress.Hostname == "" {
			return fmt.Errorf("hostname is required when ingress is enabled")
		}
		if len(p.Ingress.Paths) == 0 {
			return fmt.Errorf("at least one path is required when ingress is enabled")
		}
	}

	// Validate remote configuration if provided
	if p.Remote != nil {
		if p.Remote.Repo == "" {
			return fmt.Errorf("repo is required when remote is configured")
		}
		if p.Remote.Branch == "" {
			p.Remote.Branch = "main" // Set default branch if not specified
		}
	}

	return nil
}

// processProjectPaths processes and validates project paths, making them absolute if necessary
func (p *Project) processProjectPaths(configDir string) error {
	// Make project path absolute if it's relative
	if !filepath.IsAbs(p.Path) {
		p.Path = filepath.Join(configDir, p.Path)
	}

	// Make compose file path absolute if it's relative
	if !filepath.IsAbs(p.ComposeFile) {
		p.ComposeFile = filepath.Join(p.Path, p.ComposeFile)
	}

	// Make context path absolute if it's relative and not empty
	if p.Context != "" && !filepath.IsAbs(p.Context) {
		p.Context = filepath.Join(p.Path, p.Context)
	}

	return nil
}

// processAllEnvVars processes environment variables in all relevant fields
func (p *Project) processAllEnvVars() error {
	// Process path fields
	if processed, err := substituteEnvVars(p.Path); err != nil {
		return fmt.Errorf("failed to process path: %w", err)
	} else {
		p.Path = processed
	}

	if processed, err := substituteEnvVars(p.ComposeFile); err != nil {
		return fmt.Errorf("failed to process compose_file: %w", err)
	} else {
		p.ComposeFile = processed
	}

	if p.Context != "" {
		if processed, err := substituteEnvVars(p.Context); err != nil {
			return fmt.Errorf("failed to process context: %w", err)
		} else {
			p.Context = processed
		}
	}

	// Process ingress fields if enabled
	if p.Ingress != nil && p.Ingress.Enabled {
		if processed, err := substituteEnvVars(p.Ingress.Hostname); err != nil {
			return fmt.Errorf("failed to process ingress hostname: %w", err)
		} else {
			p.Ingress.Hostname = processed
		}
	}

	// Process remote fields if configured
	if p.Remote != nil {
		if processed, err := substituteEnvVars(p.Remote.Repo); err != nil {
			return fmt.Errorf("failed to process remote repo: %w", err)
		} else {
			p.Remote.Repo = processed
		}

		if processed, err := substituteEnvVars(p.Remote.Branch); err != nil {
			return fmt.Errorf("failed to process remote branch: %w", err)
		} else {
			p.Remote.Branch = processed
		}
	}

	// Process environment variables last (after all other fields are processed)
	return p.processEnvVars()
}

// LoadConfig reads and parses a YAML configuration file into a Config struct.
// It returns an error if the file cannot be read or if the YAML is invalid.
func LoadConfig(filePath string) (*Config, error) {
	log.WithField("file", filePath).Info("Loading configuration file")

	// Get the absolute path and directory of the config file
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for config file: %w", err)
	}
	configDir := filepath.Dir(absPath)

	data, err := os.ReadFile(absPath)
	if err != nil {
		log.WithError(err).WithField("file", absPath).Error("Failed to read config file")
		return nil, fmt.Errorf("failed to read config file %s: %w", absPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.WithError(err).WithField("file", absPath).Error("Failed to parse YAML config")
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	if len(config.Projects) == 0 {
		log.WithField("file", absPath).Error("Config file contains no projects")
		return nil, fmt.Errorf("config file must contain at least one project")
	}

	log.WithField("projectCount", len(config.Projects)).Debug("Successfully parsed config file")

	// Process each project's configuration
	for i := range config.Projects {
		project := &config.Projects[i]
		log.WithField("project", project.Name).Debug("Processing project configuration")

		// Validate the project configuration
		if err := project.validate(); err != nil {
			return nil, fmt.Errorf("invalid configuration for project %q: %w", project.Name, err)
		}

		// Process all environment variables in the project configuration
		if err := project.processAllEnvVars(); err != nil {
			return nil, fmt.Errorf("failed to process environment variables for project %q: %w", project.Name, err)
		}

		// Process project paths
		if err := project.processProjectPaths(configDir); err != nil {
			return nil, fmt.Errorf("failed to process paths for project %q: %w", project.Name, err)
		}

		// Load and merge .env file if specified (after paths are processed)
		if err := project.processEnvFile(); err != nil {
			return nil, err
		}
	}

	log.WithField("file", absPath).Info("Successfully loaded and processed configuration")
	return &config, nil
}
