package ingress

import (
	"fmt"
	"strings"
)

// IngressConfig represents the configuration for a single ingress endpoint
type IngressConfig struct {
	Hostname    string   // The hostname to match
	Paths       []string // URL path patterns to route
	Upstream    string   // The address (host:port) of the service to route to
	SSLEnabled  bool     // Whether SSL is enabled for this endpoint
	SSLCertPath string   // Path to the SSL certificate file
	SSLKeyPath  string   // Path to the SSL key file
}

// GenerateNginxConfig generates an Nginx configuration string from a slice of IngressConfig objects.
// It creates server blocks for each configuration, including SSL configuration when enabled.
func GenerateNginxConfig(ingressConfigs []IngressConfig) (string, error) {
	if len(ingressConfigs) == 0 {
		return "", fmt.Errorf("no ingress configurations provided")
	}

	// Validate SSL configurations first
	for _, ing := range ingressConfigs {
		if ing.SSLEnabled {
			if ing.SSLCertPath == "" || ing.SSLKeyPath == "" {
				return "", fmt.Errorf("SSL certificate paths not set for hostname %s", ing.Hostname)
			}
		}
	}

	var config strings.Builder

	// Write the main Nginx configuration header
	config.WriteString(`events {
    worker_connections 1024;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    # Basic settings
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;

    # Logging
    access_log /var/log/nginx/access.log;
    error_log /var/log/nginx/error.log;

`)

	// Generate server blocks for each ingress configuration
	for i, ing := range ingressConfigs {
		// Write server block header
		config.WriteString(fmt.Sprintf("    # Server block for %s\n", ing.Hostname))
		config.WriteString("    server {\n")

		// Set server_name
		config.WriteString(fmt.Sprintf("        server_name %s;\n", ing.Hostname))

		// Add SSL configuration if enabled
		if ing.SSLEnabled {
			config.WriteString(fmt.Sprintf("        ssl_certificate %s;\n", ing.SSLCertPath))
			config.WriteString(fmt.Sprintf("        ssl_certificate_key %s;\n", ing.SSLKeyPath))
			config.WriteString("        ssl_protocols TLSv1.2 TLSv1.3;\n")
			config.WriteString("        ssl_ciphers HIGH:!aNULL:!MD5;\n")
			config.WriteString("        listen 443 ssl;\n")
		} else {
			config.WriteString("        listen 80;\n")
		}

		// Add location blocks for each path
		for _, path := range ing.Paths {
			config.WriteString("\n")
			config.WriteString(fmt.Sprintf("        location %s {\n", path))
			config.WriteString("            proxy_pass http://" + ing.Upstream + ";\n")
			config.WriteString("            proxy_set_header Host $host;\n")
			config.WriteString("            proxy_set_header X-Real-IP $remote_addr;\n")
			config.WriteString("            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
			config.WriteString("            proxy_set_header X-Forwarded-Proto $scheme;\n")
			config.WriteString("        }\n")
		}

		config.WriteString("    }\n")
		if i < len(ingressConfigs)-1 {
			config.WriteString("\n")
		}
	}

	// Close the http block without adding a newline
	config.WriteString("}")

	return config.String(), nil
}

// RunIngress orchestrates the complete ingress setup process:
// 1. Generates SSL certificates for configurations that require them
// 2. Generates the Nginx configuration
// 3. Writes the configuration to a file
// 4. Reloads Nginx to apply the changes
func RunIngress(configs []IngressConfig, nginxManager *NginxManager) error {
	if len(configs) == 0 {
		return fmt.Errorf("no ingress configurations provided")
	}

	// Generate SSL certificates for configurations that require them
	for i := range configs {
		if err := GenerateSSLCerts(&configs[i]); err != nil {
			return fmt.Errorf("failed to generate SSL certificates for %s: %w", configs[i].Hostname, err)
		}
	}

	// Generate Nginx configuration
	config, err := GenerateNginxConfig(configs)
	if err != nil {
		return fmt.Errorf("failed to generate Nginx configuration: %w", err)
	}

	// Write the configuration file
	if err := nginxManager.WriteConfig(config); err != nil {
		return fmt.Errorf("failed to write Nginx configuration: %w", err)
	}

	// Check if Nginx is running
	if err := nginxManager.CheckNginxStatus(); err != nil {
		// If Nginx is not running, try to restart it
		if err := nginxManager.RestartNginx(); err != nil {
			return fmt.Errorf("failed to start Nginx: %w", err)
		}
	} else {
		// If Nginx is running, reload the configuration
		if err := nginxManager.ReloadNginx(); err != nil {
			return fmt.Errorf("failed to reload Nginx: %w", err)
		}
	}

	return nil
}
