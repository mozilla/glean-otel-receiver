package gleanreceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/config/confighttp"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: func() *Config {
				cfg := confighttp.NewDefaultServerConfig()
				cfg.NetAddr.Endpoint = "localhost:9888"
				return &Config{
					ServerConfig: cfg,
					Path:         "/submit/telemetry",
				}
			}(),
			wantErr: false,
		},
		{
			name: "empty path",
			config: func() *Config {
				cfg := confighttp.NewDefaultServerConfig()
				cfg.NetAddr.Endpoint = "localhost:9888"
				return &Config{
					ServerConfig: cfg,
					Path:         "",
				}
			}(),
			wantErr: true,
		},
		{
			name: "valid config with forwarding",
			config: func() *Config {
				cfg := confighttp.NewDefaultServerConfig()
				cfg.NetAddr.Endpoint = "localhost:9888"
				return &Config{
					ServerConfig:   cfg,
					Path:           "/submit/telemetry",
					ForwardURL:     "https://downstream.example.com/ingest",
					ForwardTimeout: 30 * time.Second,
					ForwardHeaders: map[string]string{
						"Authorization": "Bearer token",
					},
				}
			}(),
			wantErr: false,
		},
		{
			name: "invalid forward URL - no protocol",
			config: func() *Config {
				cfg := confighttp.NewDefaultServerConfig()
				cfg.NetAddr.Endpoint = "localhost:9888"
				return &Config{
					ServerConfig: cfg,
					Path:         "/submit/telemetry",
					ForwardURL:   "downstream.example.com/ingest",
				}
			}(),
			wantErr: true,
		},
		{
			name: "invalid forward URL - invalid protocol",
			config: func() *Config {
				cfg := confighttp.NewDefaultServerConfig()
				cfg.NetAddr.Endpoint = "localhost:9888"
				return &Config{
					ServerConfig: cfg,
					Path:         "/submit/telemetry",
					ForwardURL:   "ftp://downstream.example.com/ingest",
				}
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig().(*Config)

	assert.Equal(t, "localhost:9888", cfg.ServerConfig.NetAddr.Endpoint)
	assert.Equal(t, "/submit/{namespace}/{document_type}/{document_version}/{document_id}", cfg.Path)
	assert.Equal(t, 20*time.Second, cfg.ServerConfig.ReadHeaderTimeout)
}

func TestGetPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "full path with all parameters",
			path:     "/submit/{namespace}/{document_type}/{document_version}/{document_id}",
			expected: "/submit/{namespace}/{document_type}/{document_version}/{document_id}",
		},
		{
			name:     "path without parameters",
			path:     "/submit",
			expected: "/submit/{namespace}/{document_type}/{document_version}/{document_id}",
		},
		{
			name:     "path with only namespace",
			path:     "/submit/{namespace}",
			expected: "/submit/{namespace}/{document_type}/{document_version}/{document_id}",
		},
		{
			name:     "path with namespace and document_type",
			path:     "/submit/{namespace}/{document_type}",
			expected: "/submit/{namespace}/{document_type}/{document_version}/{document_id}",
		},
		{
			name:     "path with trailing slash",
			path:     "/submit/",
			expected: "/submit/{namespace}/{document_type}/{document_version}/{document_id}",
		},
		{
			name:     "simple path",
			path:     "/test",
			expected: "/test/{namespace}/{document_type}/{document_version}/{document_id}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Path: tt.path}
			result := cfg.GetPath()
			assert.Equal(t, tt.expected, result)
		})
	}
}
