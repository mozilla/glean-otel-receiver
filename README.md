# Glean OpenTelemetry Receiver

[![Tests](https://github.com/mozilla/glean-otel-receiver/actions/workflows/test.yml/badge.svg)](https://github.com/mozilla/glean-otel-receiver/actions/workflows/test.yml)
[![Build](https://github.com/mozilla/glean-otel-receiver/actions/workflows/build.yml/badge.svg)](https://github.com/mozilla/glean-otel-receiver/actions/workflows/build.yml)

A custom OpenTelemetry Collector receiver for Mozilla Glean telemetry pings.

## Overview

This receiver accepts Glean telemetry pings via HTTP POST requests and converts them to OpenTelemetry metrics and event logs:

- **Metrics**: Glean counters, quantities, distributions, rates, and other metric types are converted to appropriate OpenTelemetry metric types
- **Event Logs**: Glean events are converted to OpenTelemetry event logs with `event.name` and `event.domain` attributes

## Configuration

Add the receiver to your OpenTelemetry Collector configuration:

```yaml
receivers:
  glean:
    # HTTP server endpoint (default: localhost:8888)
    endpoint: localhost:8888

    # Path where Glean pings are received (default: /submit/telemetry)
    endpoint: /submit/telemetry

    # Optional: Configure read timeout
    read_header_timeout: 20s

exporters:
  # Configure your exporters
  logging:
    loglevel: debug

service:
  pipelines:
    metrics:
      receivers: [glean]
      exporters: [logging]
    logs:
      receivers: [glean]
      exporters: [logging]
```

## Data Mapping

### Client Info → Resource Attributes

Glean `client_info` fields are mapped to OpenTelemetry resource attributes:

- `client_id` → `client.id`
- `app_build` → `service.version`
- `app_display_version` → `app.version`
- `app_channel` → `app.channel`
- `os` → `os.type`
- `os_version` → `os.version`
- `device_model` → `device.model.name`
- And more...

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
curl -X POST http://localhost:8888/submit/example/metrics/1/1235123 \
  -H "Content-Type: application/json" \
  -d @glean_ping.json
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

## Testing

The receiver includes comprehensive unit tests:

```bash
# Run all tests
go test -v ./...

# Run with coverage
go test -v -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Current test coverage: **76.8%**

See [TESTING.md](TESTING.md) for detailed testing documentation.

## License

MIT
