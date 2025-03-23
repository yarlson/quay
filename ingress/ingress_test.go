package ingress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IngressTestSuite struct {
	suite.Suite
}

func (s *IngressTestSuite) TestGenerateNginxConfig() {
	tests := []struct {
		name           string
		configs        []Config
		expectedError  bool
		expectedConfig string
	}{
		{
			name: "single http server",
			configs: []Config{
				{
					Hostname: "example.com",
					Paths:    []string{"/"},
					Upstream: "localhost:8080",
				},
			},
			expectedError: false,
		},
		{
			name: "https server with force ssl",
			configs: []Config{
				{
					Hostname:    "secure.example.com",
					Paths:       []string{"/"},
					Upstream:    "localhost:8443",
					SSLEnabled:  true,
					SSLCertPath: "/etc/nginx/ssl/cert.pem",
					SSLKeyPath:  "/etc/nginx/ssl/key.pem",
					ForceSSL:    true,
				},
			},
			expectedError: false,
		},
		{
			name: "websocket support and custom headers",
			configs: []Config{
				{
					Hostname:  "ws.example.com",
					Paths:     []string{"/socket"},
					Upstream:  "localhost:8080",
					WebSocket: true,
					Headers: map[string]string{
						"X-Custom-Header": "value",
					},
				},
			},
			expectedError: false,
		},
		{
			name: "rate limiting and error pages",
			configs: []Config{
				{
					Hostname: "api.example.com",
					Paths:    []string{"/api"},
					Upstream: "localhost:3000",
					RateLimit: &RateLimit{
						Requests: 10,
						Window:   "1m",
						Key:      "$binary_remote_addr",
					},
					ErrorPages: map[int]string{
						404: "404.html",
						500: "500.html",
					},
				},
			},
			expectedError: false,
		},
		{
			name:          "empty configs",
			configs:       []Config{},
			expectedError: true,
		},
		{
			name: "ssl enabled without cert paths",
			configs: []Config{
				{
					Hostname:   "invalid.example.com",
					Paths:      []string{"/"},
					Upstream:   "localhost:8080",
					SSLEnabled: true,
				},
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			config, err := GenerateNginxConfig(tt.configs)
			if tt.expectedError {
				assert.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)

			// Basic assertions for each feature
			switch tt.name {
			case "single http server":
				assert.Contains(s.T(), config, "server_name example.com;")
				assert.Contains(s.T(), config, "listen 80;")
				assert.Contains(s.T(), config, "proxy_pass http://localhost:8080;")

			case "https server with force ssl":
				assert.Contains(s.T(), config, "return 301 https://$server_name$request_uri;")
				assert.Contains(s.T(), config, "listen 443 ssl http2;")
				assert.Contains(s.T(), config, "ssl_certificate /etc/nginx/ssl/cert.pem;")

			case "websocket support and custom headers":
				assert.Contains(s.T(), config, "proxy_set_header Upgrade $http_upgrade;")
				assert.Contains(s.T(), config, "proxy_set_header Connection \"upgrade\";")
				assert.Contains(s.T(), config, "add_header X-Custom-Header value always;")

			case "rate limiting and error pages":
				assert.Contains(s.T(), config, "limit_req_zone $binary_remote_addr zone=one:1m rate=10r/1m;")
				assert.Contains(s.T(), config, "error_page 404 /404.html;")
				assert.Contains(s.T(), config, "error_page 500 /500.html;")
			}

			// Common assertions for all valid configs
			assert.Contains(s.T(), config, "# Security headers")
			assert.Contains(s.T(), config, "add_header X-Content-Type-Options nosniff;")
			assert.Contains(s.T(), config, "ssl_protocols TLSv1.2 TLSv1.3;")
		})
	}
}

func TestIngressSuite(t *testing.T) {
	suite.Run(t, new(IngressTestSuite))
}
