package gleanreceiver

import (
	"errors"

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

// Validate checks if the receiver configuration is valid
func (cfg *Config) Validate() error {
	if cfg.Path == "" {
		return errors.New("path cannot be empty")
	}
	return nil
}
