package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/a8m/envsubst"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

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
	return envsubst.String(value)
}

// processEnvVars performs environment variable substitution on all environment
// variables in the project configuration.
func (p *Project) processEnvVars() error {
	if p.Env == nil {
		return nil
	}

	// Create a new map to store processed values
	processedEnv := make(map[string]string, len(p.Env))

	// Process each environment variable
	for key, value := range p.Env {
		processed, err := substituteEnvVars(value)
		if err != nil {
			return fmt.Errorf("failed to substitute environment variables for key %q: %w", key, err)
		}
		processedEnv[key] = processed
	}

	// Replace the original map with processed values
	p.Env = processedEnv
	return nil
}

// loadEnvFile reads and parses a .env file into a map of environment variables.
// It returns an error if the file cannot be read or if the format is invalid.
func loadEnvFile(filePath string) (map[string]string, error) {
	// Read the .env file
	envMap, err := godotenv.Read(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read .env file %s: %w", filePath, err)
	}
	return envMap, nil
}

// processEnvFile loads and merges environment variables from the specified .env file
// with the project's existing environment variables.
func (p *Project) processEnvFile() error {
	if p.EnvFile == "" {
		return nil
	}

	// Resolve the .env file path relative to the project path
	envFilePath := p.EnvFile
	if !filepath.IsAbs(envFilePath) && p.Path != "" {
		envFilePath = filepath.Join(p.Path, envFilePath)
	}

	// Load environment variables from the .env file
	envMap, err := loadEnvFile(envFilePath)
	if err != nil {
		return fmt.Errorf("failed to load .env file for project %q: %w", p.Name, err)
	}

	// Initialize the project's environment map if it's nil
	if p.Env == nil {
		p.Env = make(map[string]string)
	}

	// Merge environment variables, with project's env taking precedence
	for key, value := range envMap {
		if _, exists := p.Env[key]; !exists {
			p.Env[key] = value
		}
	}

	return nil
}

// LoadConfig reads and parses a YAML configuration file into a Config struct.
// It returns an error if the file cannot be read or if the YAML is invalid.
func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	if len(config.Projects) == 0 {
		return nil, fmt.Errorf("config file must contain at least one project")
	}

	// Process each project's configuration
	for i := range config.Projects {
		// First load and merge .env file if specified
		if err := config.Projects[i].processEnvFile(); err != nil {
			return nil, err
		}
		// Then process environment variable substitutions
		if err := config.Projects[i].processEnvVars(); err != nil {
			return nil, fmt.Errorf("failed to process environment variables for project %q: %w", config.Projects[i].Name, err)
		}
	}

	return &config, nil
}
