package gleanreceiver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type gleanPingForwarder struct {
	cfg    *Config
	logger *zap.Logger
	host   component.Host
	client *http.Client
}

// newGleanPingForwarder creates a new instance of gleanPingForwarder
func newGleanPingForwarder(cfg *Config, set receiver.Settings) (*gleanPingForwarder, error) {
	// Create HTTP client with timeout
	timeout := cfg.ForwardTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	return &gleanPingForwarder{
		cfg:    cfg,
		logger: set.Logger,
		client: client,
	}, nil
}

// forwardRawPing forwards the raw Glean ping JSON to the configured downstream endpoint
func (r *gleanPingForwarder) forwardRawPing(ctx context.Context, gleanReq GleanPingRequest, body []byte) error {
	if r.cfg.ForwardURL == "" {
		return nil // Forwarding not configured, skip
	}

	req, err := r.buildRequest(ctx, gleanReq, body)
	if err != nil {
		return err
	}

	// Send request
	go func() {
		if err := r.sendRequest(req); err != nil {
			r.logger.Error("Failed to forward ping to downstream",
				zap.Error(err),
				zap.String("downstream_url", r.cfg.ForwardURL))
		}
	}()
	return nil
}

// Create the full url with glean ping document paths (ns, type, version, id)
func (r *gleanPingForwarder) makeURL(gleanReq GleanPingRequest) (*url.URL, error) {
	baseURL, err := url.Parse(r.cfg.ForwardURL)
	if err != nil {
		return nil, err
	}

	fullURL := baseURL.JoinPath(gleanReq.Namespace, gleanReq.DocumentType, gleanReq.DocumentVersion, gleanReq.DocumentID)
	return fullURL, nil
}

func (r *gleanPingForwarder) buildRequest(ctx context.Context, gleanReq GleanPingRequest, body []byte) (*http.Request, error) {
	// Create POST request with raw body
	fullURL, err := r.makeURL(gleanReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create full url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create forward request: %w", err)
	}

	// Set cloned headers
	req.Header = gleanReq.Headers

	// Add custom headers from config
	for key, value := range r.cfg.ForwardHeaders {
		req.Header.Set(key, value)
	}

	return req, err
}

func (r *gleanPingForwarder) sendRequest(req *http.Request) error {
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to forward ping: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("downstream returned error %d: %s", resp.StatusCode, string(respBody))
	}

	r.logger.Debug("Successfully forwarded raw Glean ping",
		zap.String("url", r.cfg.ForwardURL),
		zap.Int("status", resp.StatusCode))

	return nil
}
