package ingress

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// Config represents the configuration for a single ingress endpoint
type Config struct {
	Hostname    string            // The hostname to match
	Paths       []string          // URL path patterns to route
	Upstream    string            // The address (host:port) of the service to route to
	SSLEnabled  bool              // Whether SSL is enabled for this endpoint
	SSLCertPath string            // Path to the SSL certificate file
	SSLKeyPath  string            // Path to the SSL key file
	ForceSSL    bool              // Whether to redirect HTTP to HTTPS
	WebSocket   bool              // Whether to enable WebSocket support
	RateLimit   *RateLimit        // Rate limiting configuration
	Headers     map[string]string // Custom response headers
	ErrorPages  map[int]string    // Custom error pages (status code -> path)
}

// RateLimit represents rate limiting configuration
type RateLimit struct {
	Requests int    // Number of requests allowed
	Window   string // Time window (e.g., "1m", "1h")
	Key      string // Key to use for rate limiting (e.g., "$binary_remote_addr")
}

// GenerateNginxConfig generates an Nginx configuration string from a slice of Config objects.
// It creates server blocks for each configuration, including SSL configuration when enabled.
func GenerateNginxConfig(ingressConfigs []Config) (string, error) {
	logger := logrus.WithFields(logrus.Fields{
		"function": "GenerateNginxConfig",
		"configs":  len(ingressConfigs),
	})

	if len(ingressConfigs) == 0 {
		logger.Error("no ingress configurations provided")
		return "", fmt.Errorf("no ingress configurations provided")
	}

	// Validate SSL configurations first
	for _, ing := range ingressConfigs {
		if ing.SSLEnabled {
			if ing.SSLCertPath == "" || ing.SSLKeyPath == "" {
				logger.WithFields(logrus.Fields{
					"hostname": ing.Hostname,
				}).Error("SSL certificate paths not set")
				return "", fmt.Errorf("SSL certificate paths not set for hostname %s", ing.Hostname)
			}
		}
	}

	logger.Debug("Starting Nginx configuration generation")
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

    # Security headers
    add_header X-Content-Type-Options nosniff;
    add_header X-Frame-Options SAMEORIGIN;
    add_header X-XSS-Protection "1; mode=block";

    # SSL settings
    ssl_session_timeout 1d;
    ssl_session_cache shared:SSL:50m;
    ssl_session_tickets off;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;

    # HSTS settings
    add_header Strict-Transport-Security "max-age=63072000" always;

    # Logging
    access_log /var/log/nginx/access.log;
    error_log /var/log/nginx/error.log;

`)

	// Generate server blocks for each ingress configuration
	for i, ing := range ingressConfigs {
		// If SSL is enabled and ForceSSL is true, create a redirect server
		if ing.SSLEnabled && ing.ForceSSL {
			config.WriteString(fmt.Sprintf("    # HTTP redirect server for %s\n", ing.Hostname))
			config.WriteString("    server {\n")
			config.WriteString(fmt.Sprintf("        server_name %s;\n", ing.Hostname))
			config.WriteString("        listen 80;\n")
			config.WriteString("        return 301 https://$server_name$request_uri;\n")
			config.WriteString("    }\n\n")
		}

		// Write server block header
		config.WriteString(fmt.Sprintf("    # Server block for %s\n", ing.Hostname))
		config.WriteString("    server {\n")
		config.WriteString(fmt.Sprintf("        server_name %s;\n", ing.Hostname))

		// Add SSL configuration if enabled
		if ing.SSLEnabled {
			config.WriteString(fmt.Sprintf("        ssl_certificate %s;\n", ing.SSLCertPath))
			config.WriteString(fmt.Sprintf("        ssl_certificate_key %s;\n", ing.SSLKeyPath))
			config.WriteString("        listen 443 ssl http2;\n")
		} else {
			config.WriteString("        listen 80;\n")
		}

		// Add custom error pages if configured
		for code, page := range ing.ErrorPages {
			config.WriteString(fmt.Sprintf("        error_page %d /%s;\n", code, page))
		}

		// Add custom headers if configured
		for key, value := range ing.Headers {
			config.WriteString(fmt.Sprintf("        add_header %s %s always;\n", key, value))
		}

		// Add rate limiting if configured
		if ing.RateLimit != nil {
			config.WriteString(fmt.Sprintf("        limit_req_zone %s zone=one:%s rate=%dr/%s;\n",
				ing.RateLimit.Key, ing.RateLimit.Window, ing.RateLimit.Requests, ing.RateLimit.Window))
		}

		// Add location blocks for each path
		for _, path := range ing.Paths {
			config.WriteString("\n")
			config.WriteString(fmt.Sprintf("        location %s {\n", path))

			// Add rate limiting if configured
			if ing.RateLimit != nil {
				config.WriteString("            limit_req zone=one burst=5 nodelay;\n")
			}

			config.WriteString("            proxy_pass http://" + ing.Upstream + ";\n")
			config.WriteString("            proxy_set_header Host $host;\n")
			config.WriteString("            proxy_set_header X-Real-IP $remote_addr;\n")
			config.WriteString("            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
			config.WriteString("            proxy_set_header X-Forwarded-Proto $scheme;\n")

			// Add WebSocket support if enabled
			if ing.WebSocket {
				config.WriteString("            proxy_http_version 1.1;\n")
				config.WriteString("            proxy_set_header Upgrade $http_upgrade;\n")
				config.WriteString("            proxy_set_header Connection \"upgrade\";\n")
				config.WriteString("            proxy_read_timeout 86400;\n")
			}

			config.WriteString("        }\n")
		}

		config.WriteString("    }\n")
		if i < len(ingressConfigs)-1 {
			config.WriteString("\n")
		}
	}

	// Close the http block without adding a newline
	config.WriteString("}")

	logger.Info("Successfully generated Nginx configuration")
	return config.String(), nil
}

// RunIngress orchestrates the complete ingress setup process:
// 1. Generates SSL certificates for configurations that require them
// 2. Generates the Nginx configuration
// 3. Writes the configuration to a file
// 4. Reloads Nginx to apply the changes
func RunIngress(configs []Config, projectDir string) error {
	logger := logrus.WithFields(logrus.Fields{
		"function": "RunIngress",
		"configs":  len(configs),
	})

	if len(configs) == 0 {
		logger.Error("no ingress configurations provided")
		return fmt.Errorf("no ingress configurations provided")
	}

	logger.Debug("Starting ingress setup process")

	// Generate SSL certificates for configurations that require them
	for i := range configs {
		logger.WithFields(logrus.Fields{
			"hostname": configs[i].Hostname,
		}).Debug("Generating SSL certificates")

		if err := GenerateSSLCerts(&configs[i], projectDir); err != nil {
			logger.WithFields(logrus.Fields{
				"hostname": configs[i].Hostname,
				"error":    err,
			}).Error("Failed to generate SSL certificates")
			return fmt.Errorf("failed to generate SSL certificates for %s: %w", configs[i].Hostname, err)
		}
	}

	// Generate Nginx configuration
	logger.Debug("Generating Nginx configuration")
	config, err := GenerateNginxConfig(configs)
	if err != nil {
		logger.WithError(err).Error("Failed to generate Nginx configuration")
		return fmt.Errorf("failed to generate Nginx configuration: %w", err)
	}

	// Create Nginx manager
	nginxManager, err := NewNginxManager(projectDir)
	if err != nil {
		logger.WithError(err).Error("Failed to create Nginx manager")
		return fmt.Errorf("failed to create Nginx manager: %w", err)
	}

	// Write the configuration file
	logger.Debug("Writing Nginx configuration")
	if err := nginxManager.WriteConfig(config); err != nil {
		logger.WithError(err).Error("Failed to write Nginx configuration")
		return fmt.Errorf("failed to write Nginx configuration: %w", err)
	}

	// Check if Nginx is running
	logger.Debug("Checking Nginx status")
	if err := nginxManager.CheckNginxStatus(); err != nil {
		// If Nginx is not running, try to restart it
		logger.Info("Nginx not running, attempting to start")
		if err := nginxManager.RestartNginx(); err != nil {
			logger.WithError(err).Error("Failed to start Nginx")
			return fmt.Errorf("failed to start Nginx: %w", err)
		}
		logger.Info("Successfully started Nginx")
	} else {
		// If Nginx is running, reload the configuration
		logger.Info("Reloading Nginx configuration")
		if err := nginxManager.ReloadNginx(); err != nil {
			logger.WithError(err).Error("Failed to reload Nginx")
			return fmt.Errorf("failed to reload Nginx: %w", err)
		}
		logger.Info("Successfully reloaded Nginx")
	}

	logger.Info("Successfully completed ingress setup")
	return nil
}
