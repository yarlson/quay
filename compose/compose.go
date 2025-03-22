package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"github.com/sirupsen/logrus"
)

// PortMapping represents a port mapping override for a service.
type PortMapping struct {
	Service   string // Name of the service to modify
	Host      string // Host port to map to
	Container string // Container port to map from
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
