package config

import (
	"fmt"
	"os"

	"github.com/a8m/envsubst"
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

	// Process environment variables for each project
	for i := range config.Projects {
		if err := config.Projects[i].processEnvVars(); err != nil {
			return nil, fmt.Errorf("failed to process environment variables for project %q: %w", config.Projects[i].Name, err)
		}
	}

	return &config, nil
}
