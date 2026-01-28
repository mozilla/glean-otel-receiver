package gleanreceiver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

// gleanReceiver implements the receiver.Metrics and receiver.Logs interfaces
type gleanReceiver struct {
	cfg             *Config
	logger          *zap.Logger
	metricsConsumer consumer.Metrics
	logsConsumer    consumer.Logs
	server          *http.Server
	host            component.Host
	startOnce       sync.Once
	shutdownOnce    sync.Once
}

// newGleanReceiver creates a new instance of gleanReceiver
func newGleanReceiver(
	cfg *Config,
	set receiver.Settings,
	metricsConsumer consumer.Metrics,
	logsConsumer consumer.Logs,
) (*gleanReceiver, error) {
	if metricsConsumer == nil && logsConsumer == nil {
		return nil, errors.New("at least one consumer (metrics or logs) must be provided")
	}

	return &gleanReceiver{
		cfg:             cfg,
		logger:          set.Logger,
		metricsConsumer: metricsConsumer,
		logsConsumer:    logsConsumer,
	}, nil
}

// Start starts the HTTP server for receiving Glean pings
func (r *gleanReceiver) Start(ctx context.Context, host component.Host) error {
	var startErr error
	r.startOnce.Do(func() {
		r.host = host

		mux := http.NewServeMux()
		mux.HandleFunc(r.cfg.Path, r.handleGleanPing)

		r.server = &http.Server{
			Addr:              r.cfg.ServerConfig.Endpoint,
			Handler:           mux,
			ReadHeaderTimeout: r.cfg.ServerConfig.ReadHeaderTimeout,
		}

		r.logger.Info("Starting Glean receiver",
			zap.String("endpoint", r.cfg.ServerConfig.Endpoint),
			zap.String("path", r.cfg.Path))

		go func() {
			if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				r.logger.Error("Error starting HTTP server", zap.Error(err))
			}
		}()
	})

	return startErr
}

// Shutdown stops the HTTP server
func (r *gleanReceiver) Shutdown(ctx context.Context) error {
	var shutdownErr error
	r.shutdownOnce.Do(func() {
		if r.server != nil {
			r.logger.Info("Shutting down Glean receiver")
			shutdownErr = r.server.Shutdown(ctx)
		}
	})
	return shutdownErr
}

// handleGleanPing processes incoming Glean ping requests
func (r *gleanReceiver) handleGleanPing(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		r.logger.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var ping GleanPing
	if err := json.Unmarshal(body, &ping); err != nil {
		r.logger.Error("Failed to parse Glean ping", zap.Error(err))
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Convert to metrics if metrics consumer is available
	if r.metricsConsumer != nil && ping.Metrics != nil {
		metrics, err := convertToMetrics(&ping)
		if err != nil {
			r.logger.Error("Failed to convert to metrics", zap.Error(err))
			http.Error(w, "Failed to process metrics", http.StatusInternalServerError)
			return
		}

		if err := r.metricsConsumer.ConsumeMetrics(req.Context(), metrics); err != nil {
			r.logger.Error("Failed to consume metrics", zap.Error(err))
			http.Error(w, "Failed to process metrics", http.StatusInternalServerError)
			return
		}
	}

	// Convert to event logs if logs consumer is available
	if r.logsConsumer != nil && len(ping.Events) > 0 {
		logs, err := convertToEventLogs(&ping)
		if err != nil {
			r.logger.Error("Failed to convert to event logs", zap.Error(err))
			http.Error(w, "Failed to process event logs", http.StatusInternalServerError)
			return
		}

		if err := r.logsConsumer.ConsumeLogs(req.Context(), logs); err != nil {
			r.logger.Error("Failed to consume event logs", zap.Error(err))
			http.Error(w, "Failed to process event logs", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}
