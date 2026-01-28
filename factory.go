package gleanreceiver

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

const (
	typeStr   = "glean"
	stability = component.StabilityLevelAlpha
)

var (
	sharedReceivers = make(map[component.ID]*gleanReceiver)
	receiversMux    sync.Mutex
)

// NewFactory creates a factory for Glean receiver
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		component.MustNewType(typeStr),
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, stability),
		receiver.WithLogs(createLogsReceiver, stability),
	)
}

// createDefaultConfig creates the default configuration for Glean receiver
func createDefaultConfig() component.Config {
	return &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint:          "localhost:9888",
			ReadHeaderTimeout: 20 * time.Second,
		},
		Path: "/submit/telemetry",
	}
}

// createMetricsReceiver creates a metrics receiver based on provided config
func createMetricsReceiver(
	ctx context.Context,
	set receiver.Settings,
	cfg component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	return getOrCreateReceiver(set, cfg, consumer, nil)
}

// createLogsReceiver creates a logs receiver based on provided config
func createLogsReceiver(
	ctx context.Context,
	set receiver.Settings,
	cfg component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	return getOrCreateReceiver(set, cfg, nil, consumer)
}

// getOrCreateReceiver returns a shared receiver instance
func getOrCreateReceiver(
	set receiver.Settings,
	cfg component.Config,
	metricsConsumer consumer.Metrics,
	logsConsumer consumer.Logs,
) (*gleanReceiver, error) {
	receiversMux.Lock()
	defer receiversMux.Unlock()

	rCfg := cfg.(*Config)

	// Check if a receiver already exists for this config
	rcvr, exists := sharedReceivers[set.ID]
	if exists {
		// Update consumers if provided
		if metricsConsumer != nil {
			rcvr.metricsConsumer = metricsConsumer
		}
		if logsConsumer != nil {
			rcvr.logsConsumer = logsConsumer
		}
		return rcvr, nil
	}

	// Create new receiver
	rcvr, err := newGleanReceiver(rCfg, set, metricsConsumer, logsConsumer)
	if err != nil {
		return nil, err
	}

	sharedReceivers[set.ID] = rcvr
	return rcvr, nil
}
