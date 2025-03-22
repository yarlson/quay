package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yarlson/quay/compose"
)

var (
	composeFile  string
	portMappings []string
)

// composeCmd represents the compose command
var composeCmd = &cobra.Command{
	Use:   "compose [command]",
	Short: "Execute Docker Compose operations with port mapping support",
	Long: `Execute Docker Compose operations with support for port mapping overrides.
Supported commands: up, down, ps

Example:
  quay compose up -f docker-compose.yml --port web:8080:80
  quay compose down -f docker-compose.yml
  quay compose ps -f docker-compose.yml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("command is required (up, down, or ps)")
		}

		// Parse port mappings
		var mappings []compose.PortMapping
		for _, mapping := range portMappings {
			parts := strings.Split(mapping, ":")
			if len(parts) != 3 {
				return fmt.Errorf("invalid port mapping format: %s (expected service:host:container)", mapping)
			}
			mappings = append(mappings, compose.PortMapping{
				Service:   parts[0],
				Host:      parts[1],
				Container: parts[2],
			})
		}

		// Execute the compose operation
		return compose.RunComposeOperation(composeFile, mappings, args)
	},
}

func init() {
	rootCmd.AddCommand(composeCmd)

	// Add flags
	composeCmd.Flags().StringVarP(&composeFile, "file", "f", "docker-compose.yml", "Path to the Docker Compose file")
	composeCmd.Flags().StringSliceVarP(&portMappings, "port", "p", nil, "Port mapping in format service:host:container (e.g., web:8080:80)")
	_ = composeCmd.MarkFlagRequired("file")
}
