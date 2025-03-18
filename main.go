package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
)

const (
	defaultComposeFile1 = "docker-compose.yml"
	defaultComposeFile2 = "docker-compose.yaml"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	// Define a custom flag set to handle Docker Compose-style flags
	flagSet := flag.NewFlagSet("compose-filter", flag.ExitOnError)
	composeFile := flagSet.String("f", "", "Path to docker-compose file")

	// Parse arguments
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parsing arguments: %w", err)
	}

	// Get the remaining arguments (Docker Compose command and potential service names)
	args := flagSet.Args()

	// We need a command
	if len(args) == 0 {
		printUsage(flagSet)
		return nil
	}

	// The first arg is the Docker Compose command
	composeCmd := args[0]

	// The remaining args might be Docker Compose command options or service names
	cmdOptions, services := parseRemainingArgs(args[1:])

	// If no compose file specified, look for default files
	composePath, err := findComposeFile(*composeFile)
	if err != nil {
		return err
	}

	// If no services specified, just pass through to docker-compose
	if len(services) == 0 {
		return executePassthroughCommand(composePath, args)
	}

	// If services are specified, apply filtering
	return executeFilteredCommand(composePath, composeCmd, cmdOptions, services)
}

func printUsage(flagSet *flag.FlagSet) {
	fmt.Println("Usage: compose-filter [options] COMMAND [SERVICE...]")
	fmt.Println("\nOptions:")
	flagSet.PrintDefaults()
	fmt.Println("\nExamples:")
	fmt.Println("  compose-filter up -d                     # Run all services")
	fmt.Println("  compose-filter up -d web db              # Run only web and db services")
	fmt.Println("  compose-filter -f custom.yml up redis    # Use custom compose file")
	os.Exit(1)
}

func parseRemainingArgs(args []string) (cmdOptions, services []string) {
	serviceMode := false
	for _, arg := range args {
		// If arg starts with dash, it's likely an option for the command
		if !serviceMode && strings.HasPrefix(arg, "-") {
			cmdOptions = append(cmdOptions, arg)
		} else {
			// Once we see a non-option, consider all remaining args as services
			serviceMode = true
			services = append(services, arg)
		}
	}
	return cmdOptions, services
}

func findComposeFile(specifiedFile string) (string, error) {
	if specifiedFile != "" {
		return specifiedFile, nil
	}

	// Check for default files in the current directory
	for _, filename := range []string{defaultComposeFile1, defaultComposeFile2} {
		if _, err := os.Stat(filename); err == nil {
			return filename, nil
		}
	}

	return "", fmt.Errorf("no docker-compose file found")
}

func executePassthroughCommand(composePath string, args []string) error {
	// Prepare the docker-compose command
	dockerComposeArgs := []string{"-f", composePath}
	dockerComposeArgs = append(dockerComposeArgs, args...)

	// Execute docker-compose directly
	cmd := exec.Command("docker-compose", dockerComposeArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func executeFilteredCommand(composePath, composeCmd string, cmdOptions, services []string) error {
	// Load the docker-compose project
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

	// Filter services and verify all requested services exist
	filteredProject, missingServices := filterServices(project, services)

	// Warn about missing services
	if len(missingServices) > 0 {
		fmt.Println("Warning: Some requested services were not found in the docker-compose file:")
		for _, name := range missingServices {
			fmt.Printf("  - %s\n", name)
		}
	}

	// Marshal the filtered project to YAML
	yamlData, err := yaml.Marshal(filteredProject)
	if err != nil {
		return fmt.Errorf("marshaling filtered project: %w", err)
	}

	// Prepare docker-compose command with any cmd options
	dockerComposeArgs := []string{"-f", "-", composeCmd}
	dockerComposeArgs = append(dockerComposeArgs, cmdOptions...)

	// Add --remove-orphans flag for "up" command if not already specified
	if composeCmd == "up" && !containsRemoveOrphans(cmdOptions) {
		dockerComposeArgs = append(dockerComposeArgs, "--remove-orphans")
	}

	// Add service names only for commands that work with service names
	if needsServiceNames(composeCmd) {
		dockerComposeArgs = append(dockerComposeArgs, services...)
	}

	cmd := exec.Command("docker-compose", dockerComposeArgs...)
	cmd.Stdin = strings.NewReader(string(yamlData))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func filterServices(project *types.Project, requestedServices []string) (*types.Project, []string) {
	// Create a map of requested services for quick lookup
	requestedServicesMap := make(map[string]bool)
	for _, service := range requestedServices {
		requestedServicesMap[service] = true
	}

	// Filter services - create a new Services map
	filteredServices := types.Services{}
	var missingServices []string

	for name, service := range project.Services {
		if requestedServicesMap[name] {
			filteredServices[name] = service
			delete(requestedServicesMap, name) // Remove from map to track missing services
		}
	}

	// Any services remaining in the map weren't found
	for name := range requestedServicesMap {
		missingServices = append(missingServices, name)
	}

	// Create a copy of the project with filtered services
	filteredProject := *project
	filteredProject.Services = filteredServices

	return &filteredProject, missingServices
}

// Helper function to check if --remove-orphans flag is present in options
func containsRemoveOrphans(options []string) bool {
	for _, opt := range options {
		if opt == "--remove-orphans" {
			return true
		}
	}
	return false
}

// Helper function to determine if a command needs service names
func needsServiceNames(cmd string) bool {
	serviceCommands := []string{"up", "run", "start", "restart"}
	for _, serviceCmd := range serviceCommands {
		if strings.Contains(cmd, serviceCmd) {
			return true
		}
	}
	return false
}
