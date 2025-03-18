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
	cmdOptions, includeServices, excludeServices := parseRemainingArgs(args[1:])

	if len(includeServices) > 0 && len(excludeServices) > 0 {
		return fmt.Errorf("cannot use both --include and --exclude options together")
	}

	composePath, err := findComposeFile(*composeFile)
	if err != nil {
		return err
	}

	if len(includeServices) == 0 && len(excludeServices) == 0 {
		return executePassthroughCommand(composePath, args)
	}

	return executeFilteredCommand(composePath, composeCmd, cmdOptions, includeServices, excludeServices)
}

// printUsage displays command line usage information and exits the program
func printUsage(flagSet *flag.FlagSet) {
	fmt.Println("Usage: quay [options] COMMAND [command options]")
	fmt.Println("\nOptions:")
	flagSet.PrintDefaults()
	fmt.Println("\nCommand options:")
	fmt.Println("  --include SERVICE    Service to include (can be used multiple times)")
	fmt.Println("  --exclude SERVICE    Service to exclude (can be used multiple times)")
	fmt.Println("\nNote: --include and --exclude options cannot be used together")
	fmt.Println("\nExamples:")
	fmt.Println("  quay up -d                           # Run all services")
	fmt.Println("  quay up -d --include web --include db  # Run only web and db services")
	fmt.Println("  quay up -d --exclude web               # Run all services except web")
	fmt.Println("  quay -f custom.yml up --include redis  # Use custom compose file")
	os.Exit(1)
}

// parseRemainingArgs separates command options from service names in the argument list
// It extracts services specified with --include/--exclude and returns command options and services
func parseRemainingArgs(args []string) (cmdOptions, includeServices, excludeServices []string) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--include" && i+1 < len(args) {
			includeServices = append(includeServices, args[i+1])
			i++ // Skip the next argument as it's the service name
		} else if args[i] == "--exclude" && i+1 < len(args) {
			excludeServices = append(excludeServices, args[i+1])
			i++ // Skip the next argument as it's the service name
		} else {
			cmdOptions = append(cmdOptions, args[i])
		}
	}
	return cmdOptions, includeServices, excludeServices
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
func executeFilteredCommand(composePath, composeCmd string, cmdOptions, includeServices, excludeServices []string) error {
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
