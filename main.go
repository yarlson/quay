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
	// Define flags
	composeFile := flag.String("f", "docker-compose.yml", "Path to docker-compose.yml file")
	composeCmd := flag.String("cmd", "up -d", "Docker compose command to run (e.g., 'up -d', 'down', etc.)")
	showConfig := flag.Bool("print", false, "Print filtered docker-compose config without executing")

	// Parse command line arguments
	flag.Parse()
	args := flag.Args()

	// Load the docker-compose project
	ctx := context.Background()
	projectOptions, err := cli.NewProjectOptions(
		[]string{*composeFile},
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

	// Start with the original project
	filteredProject := project

	// If services are specified, filter them
	if len(args) > 0 {
		// Create a map of requested services for quick lookup
		requestedServices := make(map[string]bool)
		for _, service := range args {
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
			fmt.Println("Warning: Some requested services were not found in the docker-compose.yml file:")
			for name := range requestedServices {
				if _, exists := project.Services[name]; !exists {
					fmt.Printf("  - %s\n", name)
				}
			}
		}

		// Update the filtered project's services
		filteredProject.Services = filteredServices
	}

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

	// Prepare docker-compose command
	cmdParts := strings.Split(*composeCmd, " ")
	cmdParts = append(cmdParts, "--remove-orphans")
	dockerComposeArgs := append([]string{"-f", "-"}, cmdParts...)

	// Add service names to the command if specified
	if len(args) > 0 && (strings.Contains(*composeCmd, "up") || strings.Contains(*composeCmd, "run") ||
		strings.Contains(*composeCmd, "start") || strings.Contains(*composeCmd, "restart")) {
		dockerComposeArgs = append(dockerComposeArgs, args...)
	}

	cmd := exec.Command("docker-compose", dockerComposeArgs...)
	cmd.Stdin = strings.NewReader(string(yamlData))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if len(args) > 0 {
		fmt.Printf("Executing: docker-compose %s with filtered services: %s\n",
			strings.Join(cmdParts, " "), strings.Join(args, ", "))
	} else {
		fmt.Printf("Executing: docker-compose %s with all services\n",
			strings.Join(cmdParts, " "))
	}

	err = cmd.Run()
	if err != nil {
		log.Fatalf("Error executing docker-compose: %v", err)
	}
}
