package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yarlson/quay/config"
	"github.com/yarlson/quay/ingress"
)

var (
	configFile string
)

// ingressCmd represents the ingress command
var ingressCmd = &cobra.Command{
	Use:   "ingress",
	Short: "Configure and manage ingress routing with Nginx",
	Long: `Configure and manage ingress routing with Nginx.
This command reads the project configurations and sets up Nginx as a reverse proxy,
handling SSL certificates and routing traffic to the appropriate services.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load the configuration file
		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Convert project configurations to ingress configurations
		var ingressConfigs []ingress.IngressConfig
		for _, project := range cfg.Projects {
			if project.Ingress == nil || !project.Ingress.Enabled {
				continue
			}

			// Create ingress configuration for each path
			for _, path := range project.Ingress.Paths {
				ingressConfigs = append(ingressConfigs, ingress.IngressConfig{
					Hostname:   project.Ingress.Hostname,
					Paths:      []string{path},
					Upstream:   fmt.Sprintf("localhost:%s", project.Ports[0].Host), // Use the first port mapping
					SSLEnabled: project.Ingress.SSL != nil,
				})
			}
		}

		if len(ingressConfigs) == 0 {
			return fmt.Errorf("no ingress configurations found in the provided config file")
		}

		// Create Nginx manager
		nginxManager := ingress.NewNginxManager("/etc/nginx/conf.d", "quay.conf")

		// Run ingress setup
		if err := ingress.RunIngress(ingressConfigs, nginxManager); err != nil {
			return fmt.Errorf("failed to setup ingress: %w", err)
		}

		fmt.Println("Successfully configured ingress routing")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(ingressCmd)

	// Add flags
	ingressCmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Path to the configuration file")
	_ = ingressCmd.MarkFlagRequired("config")
}
