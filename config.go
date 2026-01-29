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
	// Required path parameters in order
	requiredParams := []string{"{namespace}", "{document_type}", "{document_version}", "{document_id}"}

	// Check if all required parameters are already present
	hasAllParams := true
	for _, param := range requiredParams {
		if !strings.Contains(cfg.Path, param) {
			hasAllParams = false
			break
		}
	}

	// If all parameters are present, return as-is
	if hasAllParams {
		return cfg.Path
	}

	// Otherwise, append missing parameters
	result := cfg.Path
	for _, param := range requiredParams {
		if !strings.Contains(result, param) {
			result = path.Join(result, param)
		}
	}

	return result
}

// Validate checks if the receiver configuration is valid
func (cfg *Config) Validate() error {
	if cfg.Path == "" {
		return errors.New("path cannot be empty")
	}
	return nil
}
