package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
)

// Default Docker Compose file names to check when none specified
const (
	defaultComposeFile1 = "docker-compose.yml"
	defaultComposeFile2 = "docker-compose.yaml"
)

// main is the entry point for the application that handles Docker Compose filtering
func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// run processes command line arguments and executes Docker Compose commands
// with optional service filtering
func run() error {
	flagSet := flag.NewFlagSet("quay", flag.ExitOnError)
	composeFile := flagSet.String("f", "", "Path to docker-compose file")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parsing arguments: %w", err)
	}

	args := flagSet.Args()

	if len(args) == 0 {
		printUsage(flagSet)
		return nil
	}

	composeCmd := args[0]
	cmdOptions, includeServices, excludeServices, portMappings := parseRemainingArgs(args[1:])

	if len(includeServices) > 0 && len(excludeServices) > 0 {
		return fmt.Errorf("cannot use both --include and --exclude options together")
	}

	composePath, err := findComposeFile(*composeFile)
	if err != nil {
		return err
	}

	if len(includeServices) == 0 && len(excludeServices) == 0 && len(portMappings) == 0 {
		return executePassthroughCommand(composePath, args)
	}

	return executeFilteredCommand(composePath, composeCmd, cmdOptions, includeServices, excludeServices, portMappings)
}

// PortMapping represents a port mapping for a service
type PortMapping struct {
	ServiceName   string
	HostPort      string
	ContainerPort string
}

// printUsage displays command line usage information and exits the program
func printUsage(flagSet *flag.FlagSet) {
	fmt.Println("Usage: quay [options] COMMAND [command options]")
	fmt.Println("\nOptions:")
	flagSet.PrintDefaults()
	fmt.Println("\nCommand options:")
	fmt.Println("  --include SERVICE    Service to include (can be used multiple times)")
	fmt.Println("  --exclude SERVICE    Service to exclude (can be used multiple times)")
	fmt.Println("  --port SERVICE:HOST_PORT:CONTAINER_PORT    Redefine published port for a service")
	fmt.Println("\nNote: --include and --exclude options cannot be used together")
	fmt.Println("\nExamples:")
	fmt.Println("  quay up -d                           # Run all services")
	fmt.Println("  quay up -d --include web --include db  # Run only web and db services")
	fmt.Println("  quay up -d --exclude web               # Run all services except web")
	fmt.Println("  quay -f custom.yml up --include redis  # Use custom compose file")
	fmt.Println("  quay up -d --port web:8080:80          # Run with web service port 80 published to host port 8080")
	os.Exit(1)
}

// parseRemainingArgs separates command options from service names in the argument list
// It extracts services specified with --include/--exclude and returns command options and services
func parseRemainingArgs(args []string) (cmdOptions, includeServices, excludeServices []string, portMappings []PortMapping) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--include" && i+1 < len(args) {
			includeServices = append(includeServices, args[i+1])
			i++ // Skip the next argument as it's the service name
		} else if args[i] == "--exclude" && i+1 < len(args) {
			excludeServices = append(excludeServices, args[i+1])
			i++ // Skip the next argument as it's the service name
		} else if args[i] == "--port" && i+1 < len(args) {
			// Parse port mapping in format service:host_port:container_port
			portMapping, err := parsePortMapping(args[i+1])
			if err != nil {
				fmt.Printf("Warning: Invalid port mapping format '%s': %v\n", args[i+1], err)
			} else {
				portMappings = append(portMappings, portMapping)
			}
			i++ // Skip the next argument as it's the port mapping
		} else {
			cmdOptions = append(cmdOptions, args[i])
		}
	}
	return cmdOptions, includeServices, excludeServices, portMappings
}

// parsePortMapping parses a port mapping string in the format service:host_port:container_port
func parsePortMapping(mapping string) (PortMapping, error) {
	re := regexp.MustCompile(`^([^:]+):(\d+):(\d+)$`)
	matches := re.FindStringSubmatch(mapping)

	if matches == nil || len(matches) != 4 {
		return PortMapping{}, fmt.Errorf("invalid format, expected SERVICE:HOST_PORT:CONTAINER_PORT")
	}

	serviceName := matches[1]
	hostPort := matches[2]
	containerPort := matches[3]

	// Validate port numbers
	if _, err := strconv.Atoi(hostPort); err != nil {
		return PortMapping{}, fmt.Errorf("invalid host port: %s", hostPort)
	}

	if _, err := strconv.Atoi(containerPort); err != nil {
		return PortMapping{}, fmt.Errorf("invalid container port: %s", containerPort)
	}

	return PortMapping{
		ServiceName:   serviceName,
		HostPort:      hostPort,
		ContainerPort: containerPort,
	}, nil
}

// findComposeFile locates a Docker Compose file to use, either the specified file
// or one of the default files if none is specified
func findComposeFile(specifiedFile string) (string, error) {
	if specifiedFile != "" {
		return specifiedFile, nil
	}

	for _, filename := range []string{defaultComposeFile1, defaultComposeFile2} {
		if _, err := os.Stat(filename); err == nil {
			return filename, nil
		}
	}

	return "", fmt.Errorf("no docker-compose file found")
}

// executePassthroughCommand runs docker-compose with all arguments passed through
// without any service filtering
func executePassthroughCommand(composePath string, args []string) error {
	dockerComposeArgs := []string{"-f", composePath}
	dockerComposeArgs = append(dockerComposeArgs, args...)

	cmd := exec.Command("docker-compose", dockerComposeArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// executeFilteredCommand loads a Docker Compose project, filters it to only include
// the specified services, and then runs docker-compose with those services
func executeFilteredCommand(composePath, composeCmd string, cmdOptions, includeServices, excludeServices []string, portMappings []PortMapping) error {
	ctx := context.Background()

	projectOptions, err := cli.NewProjectOptions(
		[]string{composePath},
		cli.WithOsEnv,
		cli.WithDotEnv,
	)
	if err != nil {
		return fmt.Errorf("creating project options: %w", err)
	}

	project, err := projectOptions.LoadProject(ctx)
	if err != nil {
		return fmt.Errorf("loading project: %w", err)
	}

	filteredProject, missingServices := filterServices(project, includeServices, excludeServices)

	// Apply port mappings to filtered project
	missingPortServices := applyPortMappings(filteredProject, portMappings)
	missingServices = append(missingServices, missingPortServices...)

	if len(missingServices) > 0 {
		fmt.Println("Warning: Some requested services were not found in the docker-compose file:")
		for _, name := range missingServices {
			fmt.Printf("  - %s\n", name)
		}
	}

	yamlData, err := yaml.Marshal(filteredProject)
	if err != nil {
		return fmt.Errorf("marshaling filtered project: %w", err)
	}

	dockerComposeArgs := []string{"-f", "-", composeCmd}
	dockerComposeArgs = append(dockerComposeArgs, cmdOptions...)

	if composeCmd == "up" && !containsRemoveOrphans(cmdOptions) {
		dockerComposeArgs = append(dockerComposeArgs, "--remove-orphans")
	}

	cmd := exec.Command("docker-compose", dockerComposeArgs...)
	cmd.Stdin = strings.NewReader(string(yamlData))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// applyPortMappings modifies service port mappings in the filtered project
// and returns a list of services that were requested but not found
func applyPortMappings(project *types.Project, portMappings []PortMapping) []string {
	var missingServices []string

	for _, mapping := range portMappings {
		service, exists := project.Services[mapping.ServiceName]
		if !exists {
			missingServices = append(missingServices, mapping.ServiceName)
			continue
		}

		// Parse string ports to integers
		containerPort, _ := strconv.ParseUint(mapping.ContainerPort, 10, 32)
		containerPortUint32 := uint32(containerPort)

		// Create or update the ports configuration for the service
		newPort := types.ServicePortConfig{
			Published: mapping.HostPort,
			Target:    containerPortUint32,
			Protocol:  "tcp", // Default to TCP protocol
		}

		// Check if there's an existing port mapping for the container port
		portUpdated := false
		for i, port := range service.Ports {
			if port.Target == containerPortUint32 {
				// Update the existing port mapping
				service.Ports[i].Published = mapping.HostPort
				portUpdated = true
				break
			}
		}

		// If no existing mapping was found, add a new one
		if !portUpdated {
			service.Ports = append(service.Ports, newPort)
		}

		// Update the service in the project
		project.Services[mapping.ServiceName] = service
	}

	return missingServices
}

// filterServices creates a filtered version of the project containing only the requested services
// and returns a list of any services that were requested but not found
func filterServices(project *types.Project, includeServices, excludeServices []string) (*types.Project, []string) {
	// Convert include and exclude services to maps for quick lookup
	includeMap := make(map[string]bool)
	for _, service := range includeServices {
		includeMap[service] = true
	}

	excludeMap := make(map[string]bool)
	for _, service := range excludeServices {
		excludeMap[service] = true
	}

	// Track which services we couldn't find
	missingIncludeServices := make(map[string]bool)
	for service := range includeMap {
		missingIncludeServices[service] = true
	}

	missingExcludeServices := make(map[string]bool)
	for service := range excludeMap {
		missingExcludeServices[service] = true
	}

	// Create a filtered version of the project services
	filteredServices := types.Services{}

	// If include services are specified, only include those services
	// If only exclude services are specified, include all except those
	usingIncludeMode := len(includeServices) > 0

	for name, service := range project.Services {
		if usingIncludeMode {
			// Include mode: only add services that are explicitly included
			if includeMap[name] {
				filteredServices[name] = service
				delete(missingIncludeServices, name)
			}
		} else {
			// Exclude mode: add all services except those explicitly excluded
			if !excludeMap[name] {
				filteredServices[name] = service
			} else {
				delete(missingExcludeServices, name)
			}
		}
	}

	// Collect missing services for error reporting
	var missingServices []string
	for service := range missingIncludeServices {
		missingServices = append(missingServices, service)
	}
	for service := range missingExcludeServices {
		missingServices = append(missingServices, service)
	}

	// Create a filtered project with the selected services
	filteredProject := *project
	filteredProject.Services = filteredServices

	return &filteredProject, missingServices
}

// containsRemoveOrphans checks if the --remove-orphans flag is present in the options list
func containsRemoveOrphans(options []string) bool {
	for _, opt := range options {
		if opt == "--remove-orphans" {
			return true
		}
	}
	return false
}
