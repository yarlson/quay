package compose

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/types"
	"github.com/sirupsen/logrus"
)

// PortMapping represents a port mapping override for a service.
type PortMapping struct {
	Service   string // Name of the service to modify
	Host      string // Host port to map to
	Container string // Container port to map from
}

type commandType string

const (
	commandUp   commandType = "up"
	commandDown commandType = "down"
	commandPs   commandType = "ps"
)

type executeOptions struct {
	WorkingDir string      // The working directory for the command
	Command    commandType // The command to execute
	Detach     bool        // Whether to run containers in the background
}

// RunComposeOperation orchestrates the complete Docker Compose operation workflow.
// It loads the compose file, applies port mappings, and executes the command with the updated configuration.
func RunComposeOperation(composeFile string, mappings []PortMapping, cmdArgs []string) error {
	logger := logrus.WithFields(logrus.Fields{
		"compose_file": composeFile,
		"mappings":     len(mappings),
		"cmd_args":     cmdArgs,
	})

	// Load the compose file.
	logger.Debug("Loading compose file")
	project, err := loadComposeFile(composeFile)
	if err != nil {
		return fmt.Errorf("failed to load compose file: %w", err)
	}

	// Apply port mappings if any.
	if len(mappings) > 0 {
		logger.Debug("Applying port mappings")
		if err := applyPortMappings(project, mappings); err != nil {
			return fmt.Errorf("failed to apply port mappings: %w", err)
		}
	}

	// Generate updated YAML configuration.
	logger.Debug("Generating updated YAML configuration")
	if _, err := generateYAML(project); err != nil {
		return fmt.Errorf("failed to generate YAML: %w", err)
	}

	// Parse command arguments.
	var cmdType commandType
	var detach bool
	for _, arg := range cmdArgs {
		switch arg {
		case "up":
			cmdType = commandUp
		case "down":
			cmdType = commandDown
		case "ps":
			cmdType = commandPs
		case "-d", "--detach":
			detach = true
		}
	}

	if cmdType == "" {
		return fmt.Errorf("invalid command arguments: %v", cmdArgs)
	}

	// Execute the command.
	logger.Debug("Executing docker-compose command")
	opts := executeOptions{
		WorkingDir: filepath.Dir(composeFile),
		Command:    cmdType,
		Detach:     detach,
	}

	if err := executeCommand(project, opts); err != nil {
		return fmt.Errorf("failed to execute docker-compose command: %w", err)
	}

	logger.Debug("Successfully completed compose operation")
	return nil
}

// loadComposeFile loads a docker compose file from the given path and returns a Project.
func loadComposeFile(path string) (*types.Project, error) {
	logger := logrus.WithField("compose_file", path)

	// Ensure the file exists.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("compose file not found at %s", path)
	}

	// Get the absolute path of the compose file.
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Read the raw YAML to get the project name.
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file: %w", err)
	}

	var rawConfig struct {
		Name string `yaml:"name"`
	}
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Create project options with environment variable support.
	options, err := cli.NewProjectOptions(
		[]string{absPath},
		cli.WithOsEnv,
		cli.WithDotEnv,
		cli.WithName(rawConfig.Name),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create project options: %w", err)
	}

	// Load the project.
	project, err := cli.ProjectFromOptions(options)
	if err != nil {
		return nil, fmt.Errorf("failed to load project: %w", err)
	}

	logger.Debug("Successfully loaded compose file")
	return project, nil
}

// applyPortMappings applies the provided port mappings to the project.
func applyPortMappings(project *types.Project, mappings []PortMapping) error {
	logger := logrus.WithField("mappings_count", len(mappings))

	// Create a map of service names for quick lookup.
	serviceMap := make(map[string]*types.ServiceConfig)
	for i := range project.Services {
		serviceMap[project.Services[i].Name] = &project.Services[i]
	}

	for _, mapping := range mappings {
		service, exists := serviceMap[mapping.Service]
		if !exists {
			return fmt.Errorf("service %s not found in compose file", mapping.Service)
		}

		// Convert container port to integer for validation.
		containerPort, err := strconv.Atoi(mapping.Container)
		if err != nil {
			return fmt.Errorf("invalid container port %s for service %s: %w", mapping.Container, mapping.Service, err)
		}

		// Find and update the port mapping.
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
			// Add new port mapping if not found.
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

// generateYAML generates a YAML string from a project configuration.
func generateYAML(project *types.Project) (string, error) {
	logger := logrus.WithFields(logrus.Fields{
		"function": "generateYAML",
		"project":  project.Name,
	})
	logger.Debug("Starting YAML generation")

	type serviceConfig struct {
		Image       string            `yaml:"image,omitempty"`
		Ports       []string          `yaml:"ports,omitempty"`
		Environment map[string]string `yaml:"environment,omitempty"`
		Volumes     []string          `yaml:"volumes,omitempty"`
	}

	type config struct {
		Services map[string]serviceConfig `yaml:"services"`
	}

	cfg := config{
		Services: make(map[string]serviceConfig),
	}

	// Convert each service to our internal format.
	for _, service := range project.Services {
		logger.Debugf("Processing service: %s", service.Name)
		svcConfig := serviceConfig{
			Image: service.Image,
		}

		// Add ports if any.
		if len(service.Ports) > 0 {
			ports := make([]string, len(service.Ports))
			for i, port := range service.Ports {
				ports[i] = fmt.Sprintf("%s:%d", port.Published, port.Target)
			}
			svcConfig.Ports = ports
			logger.Debugf("Service %s ports: %v", service.Name, ports)
		}

		// Add environment variables if any.
		if len(service.Environment) > 0 {
			envMap := make(map[string]string)
			for k, v := range service.Environment {
				if v != nil {
					envMap[k] = *v
				}
			}
			if len(envMap) > 0 {
				svcConfig.Environment = envMap
				logger.Debugf("Service %s environment: %v", service.Name, envMap)
			}
		}

		// Add volumes if any.
		if len(service.Volumes) > 0 {
			volumes := make([]string, len(service.Volumes))
			for i, vol := range service.Volumes {
				volumes[i] = fmt.Sprintf("%s:%s", vol.Source, vol.Target)
			}
			svcConfig.Volumes = volumes
			logger.Debugf("Service %s volumes: %v", service.Name, volumes)
		}

		cfg.Services[service.Name] = svcConfig
	}

	// Marshal to YAML.
	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		logger.Errorf("Failed to marshal YAML: %v", err)
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Post-process the YAML to remove quotes from environment values.
	yamlStr := string(yamlData)
	lines := strings.Split(yamlStr, "\n")
	for i, line := range lines {
		if strings.Contains(line, "environment:") {
			continue
		}
		if strings.Contains(line, ": \"") && strings.HasSuffix(line, "\"") {
			lines[i] = strings.Replace(line, ": \"", ": ", 1)
			lines[i] = lines[i][:len(lines[i])-1]
		}
	}

	result := strings.Join(lines, "\n")
	logger.Debug("Successfully generated YAML configuration")
	return result, nil
}

// executeCommand executes the docker-compose command with the generated YAML configuration.
func executeCommand(project *types.Project, opts executeOptions) error {
	logger := logrus.WithFields(logrus.Fields{
		"project": project.Name,
		"command": opts.Command,
	})

	// Generate the YAML configuration.
	yamlData, err := generateYAML(project)
	if err != nil {
		return fmt.Errorf("failed to generate YAML: %w", err)
	}

	// Build the command arguments.
	args := []string{"-f", "-"} // Use "-" to read from stdin.
	switch opts.Command {
	case commandUp:
		args = append(args, "up")
		if opts.Detach {
			args = append(args, "-d")
		}
	case commandDown:
		args = append(args, "down")
	case commandPs:
		args = append(args, "ps")
	default:
		return fmt.Errorf("unsupported command type: %s", opts.Command)
	}

	// Set up the command.
	cmd := exec.Command("docker-compose", args...)
	cmd.Dir = opts.WorkingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Create a pipe for stdin.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start the command.
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker-compose command: %w", err)
	}

	// Write the YAML configuration to stdin.
	if _, err := stdin.Write([]byte(yamlData)); err != nil {
		return fmt.Errorf("failed to write YAML to stdin: %w", err)
	}

	// Close stdin to signal end of input.
	if err := stdin.Close(); err != nil {
		return fmt.Errorf("failed to close stdin: %w", err)
	}

	// Wait for the command to complete.
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("failed to execute docker-compose command: %w", err)
	}

	logger.Debug("Successfully executed docker-compose command")
	return nil
}
