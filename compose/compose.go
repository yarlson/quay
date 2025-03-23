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

// EnvVar is a custom type for environment variables that prevents quoting in YAML
type EnvVar string

func (e EnvVar) MarshalYAML() (interface{}, error) {
	return string(e), nil
}

// ValidateServiceConfig validates a service configuration and returns an error if any required fields are missing or invalid
func ValidateServiceConfig(service *types.ServiceConfig) error {
	if service.Name == "" {
		return fmt.Errorf("service name is required")
	}
	if service.Image == "" {
		return fmt.Errorf("image is required for service %s", service.Name)
	}

	// Validate ports
	for i, port := range service.Ports {
		if port.Target == 0 {
			return fmt.Errorf("container port is required for service %s port mapping %d", service.Name, i)
		}
		if port.Published == "" {
			return fmt.Errorf("host port is required for service %s port mapping %d", service.Name, i)
		}
	}

	// Validate volumes
	for i, volume := range service.Volumes {
		if volume.Target == "" {
			return fmt.Errorf("volume target is required for service %s volume %d", service.Name, i)
		}
	}

	return nil
}

// ValidateProject validates a project configuration and returns an error if any required fields are missing or invalid
func ValidateProject(project *types.Project) error {
	if project.Name == "" {
		return fmt.Errorf("project name is required")
	}
	if len(project.Services) == 0 {
		return fmt.Errorf("project must contain at least one service")
	}

	for i := range project.Services {
		if err := ValidateServiceConfig(&project.Services[i]); err != nil {
			return err
		}
	}

	return nil
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

	// Read the raw YAML to get the project name
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

	// Create project options with environment variable support
	options, err := cli.NewProjectOptions(
		[]string{absPath},
		cli.WithOsEnv,
		cli.WithDotEnv,
		cli.WithName(rawConfig.Name),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create project options: %w", err)
	}

	// Load the project
	project, err := cli.ProjectFromOptions(options)
	if err != nil {
		return nil, fmt.Errorf("failed to load project: %w", err)
	}

	// Validate the project configuration
	if err := ValidateProject(project); err != nil {
		return nil, fmt.Errorf("invalid project configuration: %w", err)
	}

	logger.Debug("Successfully loaded compose file")
	return project, nil
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

// Config represents the root YAML configuration
type Config struct {
	Services map[string]ServiceConfig `yaml:"services"`
}

func GenerateYAML(project *types.Project) (string, error) {
	config := Config{
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

// CommandType represents the type of Docker Compose command to execute
type CommandType string

const (
	CommandUp   CommandType = "up"
	CommandDown CommandType = "down"
	CommandPs   CommandType = "ps"
)

// ExecuteOptions represents options for executing a Docker Compose command
type ExecuteOptions struct {
	WorkingDir string      // The working directory for the command
	Command    CommandType // The command to execute
	Detach     bool        // Whether to run containers in the background
}

// CommandExecutor is an interface for executing commands
type CommandExecutor interface {
	Run(cmd *exec.Cmd) error
}

// DefaultCommandExecutor is the default implementation of CommandExecutor
type DefaultCommandExecutor struct{}

func (e *DefaultCommandExecutor) Run(cmd *exec.Cmd) error {
	return cmd.Run()
}

// ExecuteCommand executes a Docker Compose command with the provided project configuration.
// The modified YAML configuration is piped directly to the docker-compose command's stdin.
func ExecuteCommand(project *types.Project, opts ExecuteOptions) error {
	logger := logrus.WithFields(logrus.Fields{
		"project": project.Name,
		"command": opts.Command,
	})

	// Generate the YAML configuration
	yamlData, err := GenerateYAML(project)
	if err != nil {
		return fmt.Errorf("failed to generate YAML: %w", err)
	}

	// Build the command arguments
	args := []string{"-f", "-"} // Use "-" to read from stdin

	// Add command-specific arguments
	switch opts.Command {
	case CommandUp:
		args = append(args, "up")
		if opts.Detach {
			args = append(args, "-d")
		}
	case CommandDown:
		args = append(args, "down")
	case CommandPs:
		args = append(args, "ps")
	default:
		return fmt.Errorf("unsupported command type: %s", opts.Command)
	}

	// Set up the command
	cmd := exec.Command("docker-compose", args...)
	cmd.Dir = opts.WorkingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Create a pipe for stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker-compose command: %w", err)
	}

	// Write the YAML configuration to stdin
	if _, err := stdin.Write([]byte(yamlData)); err != nil {
		return fmt.Errorf("failed to write YAML to stdin: %w", err)
	}

	// Close stdin to signal end of input
	if err := stdin.Close(); err != nil {
		return fmt.Errorf("failed to close stdin: %w", err)
	}

	// Wait for the command to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("failed to execute docker-compose command: %w", err)
	}

	logger.Debug("Successfully executed docker-compose command")
	return nil
}

// RunComposeOperation orchestrates the complete Docker Compose operation workflow.
// It loads the compose file, applies port mappings, and executes the command with the updated configuration.
func RunComposeOperation(composeFile string, mappings []PortMapping, cmdArgs []string) error {
	logger := logrus.WithFields(logrus.Fields{
		"compose_file": composeFile,
		"mappings":     len(mappings),
		"cmd_args":     cmdArgs,
	})

	// Load the compose file
	logger.Debug("Loading compose file")
	project, err := LoadComposeFile(composeFile)
	if err != nil {
		return fmt.Errorf("failed to load compose file: %w", err)
	}

	// Apply port mappings if any
	if len(mappings) > 0 {
		logger.Debug("Applying port mappings")
		if err := ApplyPortMappings(project, mappings); err != nil {
			return fmt.Errorf("failed to apply port mappings: %w", err)
		}
	}

	// Generate updated YAML configuration
	logger.Debug("Generating updated YAML configuration")
	if _, err := GenerateYAML(project); err != nil {
		return fmt.Errorf("failed to generate YAML: %w", err)
	}

	// Parse command arguments
	var cmdType CommandType
	var detach bool
	for _, arg := range cmdArgs {
		switch arg {
		case "up":
			cmdType = CommandUp
		case "down":
			cmdType = CommandDown
		case "ps":
			cmdType = CommandPs
		case "-d", "--detach":
			detach = true
		}
	}

	if cmdType == "" {
		return fmt.Errorf("invalid command arguments: %v", cmdArgs)
	}

	// Execute the command
	logger.Debug("Executing docker-compose command")
	opts := ExecuteOptions{
		WorkingDir: filepath.Dir(composeFile),
		Command:    cmdType,
		Detach:     detach,
	}

	if err := ExecuteCommand(project, opts); err != nil {
		return fmt.Errorf("failed to execute docker-compose command: %w", err)
	}

	logger.Debug("Successfully completed compose operation")
	return nil
}
