package gleanreceiver

import (
	"errors"
	"path"
	"strings"

	"go.opentelemetry.io/collector/config/confighttp"
)

// Config defines the configuration for the Glean receiver
type Config struct {
	// ServerConfig contains HTTP server settings
	confighttp.ServerConfig `mapstructure:",squash"`

	// Path is the HTTP path where Glean pings are received
	// Default: /submit/telemetry
	Path string `mapstructure:"path"`
}

func (cfg *Config) GetPath() string {
	if strings.HasSuffix(cfg.Path, "{document_id}") {
		return cfg.Path
	}
	return path.Join(cfg.Path, "{document_id}")
}

// Validate checks if the receiver configuration is valid
func (cfg *Config) Validate() error {
	if cfg.Path == "" {
		return errors.New("path cannot be empty")
	}
	return nil
}
