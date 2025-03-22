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

// LoadConfig reads and parses a YAML configuration file into a Config struct.
// It returns an error if the file cannot be read or if the YAML is invalid.
func LoadConfig(filePath string) (*Config, error) {
	log.WithField("file", filePath).Info("Loading configuration file")

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.WithError(err).WithField("file", filePath).Error("Failed to read config file")
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.WithError(err).WithField("file", filePath).Error("Failed to parse YAML config")
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	if len(config.Projects) == 0 {
		log.WithField("file", filePath).Error("Config file contains no projects")
		return nil, fmt.Errorf("config file must contain at least one project")
	}

	log.WithField("projectCount", len(config.Projects)).Debug("Successfully parsed config file")

	// Process each project's configuration
	for i := range config.Projects {
		project := &config.Projects[i]
		log.WithField("project", project.Name).Debug("Processing project configuration")

		// First load and merge .env file if specified
		if err := project.processEnvFile(); err != nil {
			return nil, err
		}
		// Then process environment variable substitutions
		if err := project.processEnvVars(); err != nil {
			log.WithError(err).WithField("project", project.Name).Error("Failed to process environment variables")
			return nil, fmt.Errorf("failed to process environment variables for project %q: %w", project.Name, err)
		}
	}

	log.WithField("file", filePath).Info("Successfully loaded and processed configuration")
	return &config, nil
}
