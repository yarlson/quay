package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// PortMapping represents a port mapping override for a service.
type PortMapping struct {
	Service   string // Name of the service to modify
	Host      string // Host port to map to
	Container string // Container port to map from
}

// EnvVar is a custom type for environment variables that prevents quoting in YAML
type EnvVar string

func (e EnvVar) MarshalYAML() (interface{}, error) {
	return string(e), nil
}

// LoadComposeFile loads and parses a Docker Compose file from the provided path.
// It returns a types.Project containing the parsed configuration and any error encountered.
func LoadComposeFile(path string) (*types.Project, error) {
	logger := logrus.WithField("compose_file", path)

	// Ensure the file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("compose file not found at %s", path)
	}

	// Get the absolute path of the compose file
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get the directory containing the compose file
	workingDir := filepath.Dir(absPath)

	// Load the compose file
	config, err := loader.Load(types.ConfigDetails{
		WorkingDir: workingDir,
		ConfigFiles: []types.ConfigFile{
			{
				Filename: absPath,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load compose file: %w", err)
	}

	logger.Debug("Successfully loaded compose file")
	return config, nil
}

// ApplyPortMappings applies the provided port mapping overrides to the project's services.
// It returns an error if any service specified in the mappings is not found in the project.
func ApplyPortMappings(project *types.Project, mappings []PortMapping) error {
	logger := logrus.WithField("mappings_count", len(mappings))

	// Create a map of service names for quick lookup
	serviceMap := make(map[string]*types.ServiceConfig)
	for i := range project.Services {
		serviceMap[project.Services[i].Name] = &project.Services[i]
	}

	for _, mapping := range mappings {
		service, exists := serviceMap[mapping.Service]
		if !exists {
			return fmt.Errorf("service %s not found in compose file", mapping.Service)
		}

		// Convert container port to integer for validation
		containerPort, err := strconv.Atoi(mapping.Container)
		if err != nil {
			return fmt.Errorf("invalid container port %s for service %s: %w", mapping.Container, mapping.Service, err)
		}

		// Find and update the port mapping
		found := false
		for i := range service.Ports {
			if service.Ports[i].Target == uint32(containerPort) {
				service.Ports[i].Published = mapping.Host
				found = true
				logger.WithFields(logrus.Fields{
					"service":   mapping.Service,
					"host":      mapping.Host,
					"container": mapping.Container,
				}).Debug("Updated port mapping")
				break
			}
		}

		if !found {
			// Add new port mapping if not found
			service.Ports = append(service.Ports, types.ServicePortConfig{
				Mode:      "host",
				Published: mapping.Host,
				Target:    uint32(containerPort),
			})
			logger.WithFields(logrus.Fields{
				"service":   mapping.Service,
				"host":      mapping.Host,
				"container": mapping.Container,
			}).Debug("Added new port mapping")
		}
	}

	return nil
}

// ServiceConfig represents a service configuration in the YAML output
type ServiceConfig struct {
	Image       string            `yaml:"image,omitempty"`
	Ports       []string          `yaml:"ports,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
}

// ComposeConfig represents the root YAML configuration
type ComposeConfig struct {
	Services map[string]ServiceConfig `yaml:"services"`
}

func GenerateYAML(project *types.Project) (string, error) {
	config := ComposeConfig{
		Services: make(map[string]ServiceConfig),
	}

	// Convert each service to our internal format
	for _, service := range project.Services {
		serviceConfig := ServiceConfig{
			Image: service.Image,
		}

		// Add ports if any
		if len(service.Ports) > 0 {
			ports := make([]string, len(service.Ports))
			for i, port := range service.Ports {
				ports[i] = fmt.Sprintf("%s:%d", port.Published, port.Target)
			}
			serviceConfig.Ports = ports
		}

		// Add environment variables if any
		if len(service.Environment) > 0 {
			envMap := make(map[string]string)
			for k, v := range service.Environment {
				if v != nil {
					envMap[k] = *v
				}
			}
			if len(envMap) > 0 {
				serviceConfig.Environment = envMap
			}
		}

		// Add volumes if any
		if len(service.Volumes) > 0 {
			volumes := make([]string, len(service.Volumes))
			for i, vol := range service.Volumes {
				volumes[i] = fmt.Sprintf("%s:%s", vol.Source, vol.Target)
			}
			serviceConfig.Volumes = volumes
		}

		config.Services[service.Name] = serviceConfig
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Post-process the YAML to remove quotes from environment values
	yamlStr := string(yamlData)
	lines := strings.Split(yamlStr, "\n")
	for i, line := range lines {
		if strings.Contains(line, "environment:") {
			continue
		}
		if strings.Contains(line, ": \"") && strings.HasSuffix(line, "\"") {
			// Remove quotes from environment values
			lines[i] = strings.Replace(line, ": \"", ": ", 1)
			lines[i] = lines[i][:len(lines[i])-1]
		}
	}

	return strings.Join(lines, "\n"), nil
}
