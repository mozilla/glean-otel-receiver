package gleanreceiver

import (
	"errors"
	"path"
	"strings"
	"time"

	"go.opentelemetry.io/collector/config/confighttp"
)

// Config defines the configuration for the Glean receiver
type Config struct {
	// ServerConfig contains HTTP server settings
	confighttp.ServerConfig `mapstructure:",squash"`

	// Path is the HTTP path where Glean pings are received
	// Default: /submit/telemetry
	Path string `mapstructure:"path"`

	// ForwardURL is the downstream HTTP endpoint to forward raw Glean ping JSON
	// If empty, forwarding is disabled
	ForwardURL string `mapstructure:"forward_url"`

	// ForwardHeaders contains custom HTTP headers to send with forwarded requests
	ForwardHeaders map[string]string `mapstructure:"forward_headers"`

	// ForwardTimeout is the HTTP client timeout for forwarding requests
	// Default: 30s
	ForwardTimeout time.Duration `mapstructure:"forward_timeout"`
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

	// Validate forward URL if provided
	if cfg.ForwardURL != "" {
		if !strings.HasPrefix(cfg.ForwardURL, "http://") && !strings.HasPrefix(cfg.ForwardURL, "https://") {
			return errors.New("forward_url must start with http:// or https://")
		}
	}

	return nil
}
