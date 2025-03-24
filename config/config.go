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

// Config represents the root configuration structure.
type Config struct {
	Projects []Project `yaml:"projects"`
}

// Project represents a single project configuration.
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

// PortMapping represents a port mapping between host and container.
type PortMapping struct {
	Service   string `yaml:"service"`
	Host      string `yaml:"host"`
	Container string `yaml:"container"`
}

// Ingress represents ingress configuration for a project.
type Ingress struct {
	Enabled  bool     `yaml:"enabled"`
	Hostname string   `yaml:"hostname"`
	Paths    []string `yaml:"paths"`
	SSL      *SSL     `yaml:"ssl,omitempty"`
}

// SSL represents SSL configuration for ingress.
type SSL struct {
	Provider string `yaml:"provider"`
}

// Remote represents remote repository configuration.
type Remote struct {
	Repo   string `yaml:"repo"`
	Branch string `yaml:"branch"`
}

// substituteEnvVars performs environment variable substitution on a string
// using the ${VAR:-default} syntax. If VAR is not set, default is used.
func substituteEnvVars(value string) (string, error) {
	logger := logrus.WithFields(logrus.Fields{
		"function": "substituteEnvVars",
		"value":    value,
	})
	logger.Debug("Substituting environment variables")

	result, err := envsubst.String(value)
	if err != nil {
		logger.WithError(err).Error("Failed to substitute environment variables")
		return "", err
	}

	logger.Debug("Environment variable substitution completed")
	return result, nil
}

// processEnvVars performs environment variable substitution on all environment variables.
func (p *Project) processEnvVars() error {
	logger := logrus.WithFields(logrus.Fields{
		"function": "processEnvVars",
		"project":  p.Name,
	})
	if p.Env == nil {
		logger.Debug("No environment variables to process")
		return nil
	}

	logger.Debug("Processing environment variables")
	processedEnv := make(map[string]string, len(p.Env))

	// Process each environment variable
	for key, value := range p.Env {
		processed, err := substituteEnvVars(value)
		if err != nil {
			logger.WithError(err).Error("Failed to substitute environment variable")
			return fmt.Errorf("failed to substitute environment variable for key %q: %w", key, err)
		}
		processedEnv[key] = processed
	}

	// Replace the original map with processed values
	p.Env = processedEnv
	logger.Debug("Environment variable processing completed")
	return nil
}

// loadEnvFile reads and parses a .env file into a map of environment variables.
// It returns an error if the file cannot be read or if the format is invalid.
func loadEnvFile(filePath string) (map[string]string, error) {
	logger := logrus.WithFields(logrus.Fields{
		"function": "loadEnvFile",
		"file":     filePath,
	})
	logger.Debug("Loading .env file")

	// Read the .env file
	envMap, err := godotenv.Read(filePath)
	if err != nil {
		logger.WithError(err).Error("Failed to read .env file")
		return nil, fmt.Errorf("failed to read .env file %s: %w", filePath, err)
	}

	logger.Debug("Successfully loaded .env file")
	return envMap, nil
}

// processEnvFile loads and merges environment variables from the specified .env file.
func (p *Project) processEnvFile() error {
	logger := logrus.WithFields(logrus.Fields{
		"function": "processEnvFile",
		"project":  p.Name,
	})

	if p.EnvFile == "" {
		logger.Debug("No .env file specified")
		return nil
	}

	logger.Debug("Processing .env file")
	envFilePath := p.EnvFile
	if !filepath.IsAbs(envFilePath) && p.Path != "" {
		envFilePath = filepath.Join(p.Path, envFilePath)
	}

	// Load environment variables from the .env file
	envMap, err := loadEnvFile(envFilePath)
	if err != nil {
		logger.WithError(err).Error("Failed to load .env file")
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
	logger.Debug("Successfully merged environment variables")
	return nil
}

// validate validates a project configuration and returns an error if any required fields are missing
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
	logger := logrus.WithFields(logrus.Fields{
		"function": "processProjectPaths",
		"project":  p.Name,
	})

	if !filepath.IsAbs(p.Path) {
		p.Path = filepath.Join(configDir, p.Path)
		logger.Debugf("Converted project path to absolute: %s", p.Path)
	}

	// Make compose file path absolute if it's relative
	if !filepath.IsAbs(p.ComposeFile) {
		p.ComposeFile = filepath.Join(p.Path, p.ComposeFile)
		logger.Debugf("Converted compose file path to absolute: %s", p.ComposeFile)
	}

	// Make context path absolute if it's relative and not empty
	if p.Context != "" && !filepath.IsAbs(p.Context) {
		p.Context = filepath.Join(p.Path, p.Context)
		logger.Debugf("Converted context path to absolute: %s", p.Context)
	}

	return nil
}

// processAllEnvVars processes environment variables in all relevant fields.
func (p *Project) processAllEnvVars() error {
	logger := logrus.WithFields(logrus.Fields{
		"function": "processAllEnvVars",
		"project":  p.Name,
	})
	logger.Debug("Starting environment variable processing for all fields")

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

	logger.Debug("Completed environment variable processing for all fields")
	return p.processEnvVars()
}

// LoadConfig reads and parses a YAML configuration file into a Config struct.
// It returns an error if the file cannot be read or if the YAML is invalid.
func LoadConfig(filePath string) (*Config, error) {
	logger := logrus.WithFields(logrus.Fields{
		"function": "LoadConfig",
		"file":     filePath,
	})
	logger.Debug("Loading configuration file")

	// Get the absolute path and directory of the config file
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for config file: %w", err)
	}
	configDir := filepath.Dir(absPath)

	data, err := os.ReadFile(absPath)
	if err != nil {
		logger.WithError(err).Error("Failed to read config file")
		return nil, fmt.Errorf("failed to read config file %s: %w", absPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		logger.WithError(err).Error("Failed to parse YAML config")
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	if len(config.Projects) == 0 {
		logger.Error("Config file contains no projects")
		return nil, fmt.Errorf("config file must contain at least one project")
	}

	logger.Debug("Successfully parsed config file")

	// Process each project's configuration
	for i := range config.Projects {
		project := &config.Projects[i]
		logger.WithField("project", project.Name).Debug("Processing project configuration")

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

	logger.Info("Successfully loaded and processed configuration")
	return &config, nil
}
