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
	flagSet := flag.NewFlagSet("compose-filter", flag.ExitOnError)
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
	cmdOptions, services := parseRemainingArgs(args[1:])

	composePath, err := findComposeFile(*composeFile)
	if err != nil {
		return err
	}

	if len(services) == 0 {
		return executePassthroughCommand(composePath, args)
	}

	return executeFilteredCommand(composePath, composeCmd, cmdOptions, services)
}

// printUsage displays command line usage information and exits the program
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

// parseRemainingArgs separates command options from service names in the argument list
// It returns two slices: command options (starting with -) and service names
func parseRemainingArgs(args []string) (cmdOptions, services []string) {
	serviceMode := false
	for _, arg := range args {
		if !serviceMode && strings.HasPrefix(arg, "-") {
			cmdOptions = append(cmdOptions, arg)
		} else {
			serviceMode = true
			services = append(services, arg)
		}
	}
	return cmdOptions, services
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
func executeFilteredCommand(composePath, composeCmd string, cmdOptions, services []string) error {
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

	filteredProject, missingServices := filterServices(project, services)

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

	if needsServiceNames(composeCmd) {
		dockerComposeArgs = append(dockerComposeArgs, services...)
	}

	cmd := exec.Command("docker-compose", dockerComposeArgs...)
	cmd.Stdin = strings.NewReader(string(yamlData))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// filterServices creates a filtered version of the project containing only the requested services
// and returns a list of any services that were requested but not found
func filterServices(project *types.Project, requestedServices []string) (*types.Project, []string) {
	requestedServicesMap := make(map[string]bool)
	for _, service := range requestedServices {
		requestedServicesMap[service] = true
	}

	filteredServices := types.Services{}
	var missingServices []string

	for name, service := range project.Services {
		if requestedServicesMap[name] {
			filteredServices[name] = service
			delete(requestedServicesMap, name)
		}
	}

	for name := range requestedServicesMap {
		missingServices = append(missingServices, name)
	}

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

// needsServiceNames determines if a Docker Compose command should have service names appended
func needsServiceNames(cmd string) bool {
	serviceCommands := []string{"up", "run", "start", "restart"}
	for _, serviceCmd := range serviceCommands {
		if strings.Contains(cmd, serviceCmd) {
			return true
		}
	}
	return false
}
