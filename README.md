# [Experimental] Glean OpenTelemetry Receiver

[![Tests](https://github.com/mozilla/glean-otel-receiver/actions/workflows/test.yml/badge.svg)](https://github.com/mozilla/glean-otel-receiver/actions/workflows/test.yml)
[![Build](https://github.com/mozilla/glean-otel-receiver/actions/workflows/build.yml/badge.svg)](https://github.com/mozilla/glean-otel-receiver/actions/workflows/build.yml)

A custom & experimental OpenTelemetry Collector receiver for Mozilla Glean telemetry pings.

## Overview

This receiver accepts Glean telemetry pings via HTTP POST requests and converts them to OpenTelemetry metrics and event logs:

- **Metrics**: Glean counters, quantities, distributions, rates, and other metric types are converted to appropriate OpenTelemetry metric types
- **Event Logs**: Glean events are converted to OpenTelemetry event logs with `event.name` and `event.domain` attributes
- **Path-based Routing**: Extracts namespace, document type, version, and ID from URL path
- **Resource Attributes**: Client info and device metadata mapped to OTel resource attributes
- **Scope Attributes**: Ping metadata (seq, type, reason) added to instrumentation scope

## Quick Start

```bash
# Using Makefile
make quick-test     # Build, run, test, and stop collector

# Or manually
make build-collector
make run-bg
make send-ping
make stop

# Run demo with colored JSON output
make demo
```

## Configuration

Add the receiver to your OpenTelemetry Collector configuration:

```yaml
receivers:
  glean:
    # HTTP server endpoint (default: localhost:9888)
    endpoint: localhost:9888

    # Optional: Path where Glean pings are received
    # Default: "/submit/{namespace}/{document_type}/{document_version}/{document_id}"
    # path: ""

    # Optional: Configure read timeout
    read_header_timeout: 20s

exporters:
  debug:
    verbosity: detailed
  prometheus:
    endpoint: localhost:8889

service:
  pipelines:
    metrics:
      receivers: [glean]
      exporters: [debug, prometheus]
    logs:
      receivers: [glean]
      exporters: [debug]
```

### Path Parameters

The receiver extracts metadata from the URL path:

- `{namespace}` - Glean application namespace (e.g., `glean`, `firefox`)
- `{document_type}` - Ping type (e.g., `metrics`, `events`, `deletion-request`)
- `{document_version}` - Schema version (e.g., `1`, `2`)
- `{document_id}` - Unique document identifier (UUID)

## Data Mapping

### Client Info → Resource Attributes

Glean `client_info` fields are mapped to OpenTelemetry **resource attributes**:

- `client_id` → `client.id`
- `session_id` → `session.id`
- `session_count` → `session.count`
- `app_build` → `service.version`
- `app_display_version` → `app.version`
- `app_channel` → `app.channel`
- `architecture` → `host.arch`
- `device_manufacturer` → `device.manufacturer`
- `device_model` → `device.model.name`
- `os` → `os.type`
- `os_version` → `os.version`
- `locale` → `host.locale`

### Ping Info → Scope Attributes

Glean `ping_info` fields are mapped to OpenTelemetry **scope attributes**:

- `seq` → `ping.seq`
- `ping_type` → `ping.type`
- `reason` → `ping.reason`

### Metrics Mapping

Glean metric types are converted as follows:

| Glean Type | OpenTelemetry Type | Notes |
|------------|-------------------|-------|
| `counter` | Counter (monotonic sum) | Cumulative integer values |
| `quantity` | Gauge | Non-monotonic integer values |
| `boolean` | Gauge | 0.0 or 1.0 |
| `string` | Gauge | Value stored as attribute |
| `string_list` | Gauge | Multiple data points with index |
| `timing_distribution` | Histogram | With sum and bucket counts |
| `memory_distribution` | Histogram | With sum and bucket counts |
| `custom_distribution` | Histogram | With sum and bucket counts |
| `rate` | Gauge | Ratio with numerator/denominator attributes |

### Events → Event Logs

Glean events are converted to OpenTelemetry event logs:

- Event timestamp is calculated relative to ping start time
- Log body contains the event name
- `event.name` attribute set to event name
- `event.domain` attribute set to event category
- Event `extra` fields are added as top-level attributes
- Resource attributes include client and ping info

## Building

To use this receiver in your collector:

1. Add this module to your collector's `go.mod`
2. Import the receiver in your collector's `components.go`:

```go
import (
    gleanreceiver "github.com/mozilla/gleanotelreceiver"
)

func components() (otelcol.Factories, error) {
    factories, err := otelcolDefault.Components()
    if err != nil {
        return otelcol.Factories{}, err
    }

    receivers, err := receiver.MakeFactoryMap(
        append(
            factories.Receivers,
            gleanreceiver.NewFactory(),
        )...,
    )
    if err != nil {
        return otelcol.Factories{}, err
    }
    factories.Receivers = receivers

    return factories, nil
}
```

## Sending Glean Pings

Send Glean pings to the receiver via HTTP POST:

```bash
curl -X POST http://localhost:9888/submit/glean/metrics/1/c641eacf-c30c-4171-b403-f077724e848a \
  -H "Content-Type: application/json" \
  -d @example-ping.json

# Using Makefile (with example-ping.json)
make send-ping
```

The URL path follows the pattern:
```
/submit/{namespace}/{document_type}/{document_version}/{document_id}
```

Example Glean ping structure:

```json
{
  "client_info": {
    "client_id": "c641eacf-c30c-4171-b403-f077724e848a",
    "app_build": "1.0.0",
    "app_display_version": "1.0",
    "os": "Android",
    "os_version": "11"
  },
  "ping_info": {
    "seq": 1,
    "start_time": "2024-01-28T10:00:00Z",
    "end_time": "2024-01-28T10:01:00Z",
    "ping_type": "metrics"
  },
  "metrics": {
    "counter": {
      "app.opened": 5
    },
    "quantity": {
      "network.bytes_sent": 12345
    }
  },
  "events": [
    {
      "timestamp": 1000,
      "category": "ui",
      "name": "button_clicked",
      "extra": {
        "button_id": "submit"
      }
    }
  ]
}
```

## Development

### Makefile Targets

```bash
# Build and test
make test              # Run all tests
make test-coverage     # Generate coverage report (coverage.html)
make build             # Build receiver module
make build-collector   # Build custom collector
make verify            # Run all verification checks

# Run collector
make run              # Run in foreground
make run-bg           # Run in background
make stop             # Stop background collector
make logs             # Show logs

# Quick testing
make quick-test       # Integration test (build→run→test→stop)
make demo             # Run demo with colored output

# Docker
make docker-build     # Build Docker image
make docker-run       # Run container
make docker-stop      # Stop container

# Utilities
make clean            # Clean artifacts
make fmt              # Format code
make tidy             # Tidy modules
make help             # Show all targets
```

### Testing

```bash
# Using Makefile
make test              # Run all tests
make test-coverage     # Generate HTML coverage report
make quick-test        # Full integration test

# Or directly with Go
go test -v ./...
go test -v -cover ./...

# Run specific test
go test -v -run TestGetPath
```

### Building Custom Collector

Using OpenTelemetry Collector Builder:

```bash
# Install builder
make install-tools

# Build collector
make build-collector

# Or manually
~/go/bin/builder --config=builder-config.yaml
```

### Docker

```bash
# Build image
make docker-build
# Or: docker build -t glean-otelcol:latest .

# Run container
make docker-run
# Or: docker run -p 9888:9888 glean-otelcol:latest

# Send test ping
make send-ping
# Or: curl ...
```

## References
- [OpenTelemetry Collector Documentation](https://opentelemetry.io/docs/collector/)
- [Building Custom Components](https://opentelemetry.io/docs/collector/extend/custom-component/)
- [Glean Documentation](https://mozilla.github.io/glean/)

## License

MIT
