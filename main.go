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

func main() {
	// Define a custom flag set to handle Docker Compose-style flags
	flagSet := flag.NewFlagSet("compose-filter", flag.ExitOnError)
	composeFile := flagSet.String("f", "", "Path to docker-compose file")
	showConfig := flagSet.Bool("print", false, "Print filtered docker-compose config without executing")

	// Parse arguments
	err := flagSet.Parse(os.Args[1:])
	if err != nil {
		log.Fatalf("Error parsing arguments: %v", err)
	}

	// Get the remaining arguments (Docker Compose command and potential service names)
	args := flagSet.Args()
	if len(args) == 0 {
		fmt.Println("Usage: compose-filter [options] COMMAND [SERVICE...]")
		flagSet.PrintDefaults()
		os.Exit(1)
	}

	// The first arg is the Docker Compose command
	composeCmd := args[0]

	// The remaining args might be Docker Compose command options or service names
	var cmdOptions []string
	var services []string

	// Try to separate command options from service names
	serviceMode := false
	for _, arg := range args[1:] {
		// If arg starts with dash, it's likely an option for the command
		if !serviceMode && strings.HasPrefix(arg, "-") {
			cmdOptions = append(cmdOptions, arg)
		} else {
			// Once we see a non-option, consider all remaining args as services
			serviceMode = true
			services = append(services, arg)
		}
	}

	// If no compose file specified, look for default files
	composePath := *composeFile
	if composePath == "" {
		// Check for docker-compose.yml and docker-compose.yaml in the current directory
		for _, filename := range []string{"docker-compose.yml", "docker-compose.yaml"} {
			if _, err := os.Stat(filename); err == nil {
				composePath = filename
				break
			}
		}

		if composePath == "" {
			log.Fatalf("No docker-compose file found")
		}
	}

	// If no services specified, just pass through to docker-compose
	if len(services) == 0 {
		// Prepare the docker-compose command
		dockerComposeArgs := []string{}
		if composePath != "" {
			dockerComposeArgs = append(dockerComposeArgs, "-f", composePath)
		}
		dockerComposeArgs = append(dockerComposeArgs, args...)

		// Execute docker-compose directly
		cmd := exec.Command("docker-compose", dockerComposeArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		fmt.Printf("Executing: docker-compose %s\n", strings.Join(dockerComposeArgs, " "))

		err = cmd.Run()
		if err != nil {
			log.Fatalf("Error executing docker-compose: %v", err)
		}
		return
	}

	// If services are specified, apply filtering

	// Load the docker-compose project
	ctx := context.Background()

	projectOptions, err := cli.NewProjectOptions(
		[]string{composePath},
		cli.WithOsEnv,
		cli.WithDotEnv,
	)
	if err != nil {
		log.Fatalf("Error creating project options: %v", err)
	}

	project, err := projectOptions.LoadProject(ctx)
	if err != nil {
		log.Fatalf("Error loading project: %v", err)
	}

	// Create a map of requested services for quick lookup
	requestedServices := make(map[string]bool)
	for _, service := range services {
		requestedServices[service] = true
	}

	// Filter services - create a new Services map
	filteredServices := types.Services{}
	for name, service := range project.Services {
		if requestedServices[name] {
			filteredServices[name] = service
		}
	}

	// Verify that all requested services were found
	if len(filteredServices) != len(requestedServices) {
		fmt.Println("Warning: Some requested services were not found in the docker-compose file:")
		for name := range requestedServices {
			if _, exists := project.Services[name]; !exists {
				fmt.Printf("  - %s\n", name)
			}
		}
	}

	// Update the filtered project's services
	filteredProject := project
	filteredProject.Services = filteredServices

	// Marshal the filtered project to YAML
	yamlData, err := yaml.Marshal(filteredProject)
	if err != nil {
		log.Fatalf("Error marshaling filtered project: %v", err)
	}

	// If print flag is set, just print the config and exit
	if *showConfig {
		fmt.Println(string(yamlData))
		return
	}

	// Prepare docker-compose command with any cmd options
	dockerComposeArgs := append([]string{"-f", "-", composeCmd}, cmdOptions...)

	// Add --remove-orphans flag for "up" command if not already specified
	if composeCmd == "up" && !containsRemoveOrphans(cmdOptions) {
		dockerComposeArgs = append(dockerComposeArgs, "--remove-orphans")
	}

	// Add service names only for commands that work with service names
	if strings.Contains(composeCmd, "up") || strings.Contains(composeCmd, "run") ||
		strings.Contains(composeCmd, "start") || strings.Contains(composeCmd, "restart") {
		dockerComposeArgs = append(dockerComposeArgs, services...)
	}

	cmd := exec.Command("docker-compose", dockerComposeArgs...)
	cmd.Stdin = strings.NewReader(string(yamlData))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Executing: docker-compose %s with filtered services: %s\n",
		strings.Join(dockerComposeArgs, " "), strings.Join(services, ", "))

	err = cmd.Run()
	if err != nil {
		log.Fatalf("Error executing docker-compose: %v", err)
	}
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
