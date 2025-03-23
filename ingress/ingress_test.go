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
			expectedConfig: `events {
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

    # Server block for example.com
    server {
        server_name example.com;
        listen 80;

        location / {
            proxy_pass http://localhost:8080;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}`,
		},
		{
			name: "single https server",
			configs: []Config{
				{
					Hostname:    "secure.example.com",
					Paths:       []string{"/"},
					Upstream:    "localhost:8443",
					SSLEnabled:  true,
					SSLCertPath: "/etc/nginx/ssl/cert.pem",
					SSLKeyPath:  "/etc/nginx/ssl/key.pem",
				},
			},
			expectedError: false,
			expectedConfig: `events {
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

    # Server block for secure.example.com
    server {
        server_name secure.example.com;
        ssl_certificate /etc/nginx/ssl/cert.pem;
        ssl_certificate_key /etc/nginx/ssl/key.pem;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers HIGH:!aNULL:!MD5;
        listen 443 ssl;

        location / {
            proxy_pass http://localhost:8443;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}`,
		},
		{
			name: "multiple paths",
			configs: []Config{
				{
					Hostname: "api.example.com",
					Paths:    []string{"/v1", "/v2"},
					Upstream: "localhost:3000",
				},
			},
			expectedError: false,
			expectedConfig: `events {
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

    # Server block for api.example.com
    server {
        server_name api.example.com;
        listen 80;

        location /v1 {
            proxy_pass http://localhost:3000;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        location /v2 {
            proxy_pass http://localhost:3000;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}`,
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
			assert.Equal(s.T(), tt.expectedConfig, config)
		})
	}
}

func TestIngressSuite(t *testing.T) {
	suite.Run(t, new(IngressTestSuite))
}
