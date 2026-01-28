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
			config: &Config{
				ServerConfig: confighttp.ServerConfig{
					Endpoint: "localhost:9888",
				},
				Path: "/submit/telemetry",
			},
			wantErr: false,
		},
		{
			name: "empty path",
			config: &Config{
				ServerConfig: confighttp.ServerConfig{
					Endpoint: "localhost:9888",
				},
				Path: "",
			},
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

	assert.Equal(t, "localhost:9888", cfg.ServerConfig.Endpoint)
	assert.Equal(t, "/submit/telemetry", cfg.Path)
	assert.Equal(t, 20*time.Second, cfg.ServerConfig.ReadHeaderTimeout)
}
