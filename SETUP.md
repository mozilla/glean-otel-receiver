# Glean OpenTelemetry Receiver - Setup Guide

## Quick Start

### 1. Build the Custom Collector

The project includes a pre-configured builder setup. Simply run:

```bash
go install go.opentelemetry.io/collector/cmd/builder@v0.113.0
~/go/bin/builder --config=builder-config.yaml
```

This creates a custom collector binary at `./dist/glean-otelcol`.

### 2. Run the Collector

```bash
./dist/glean-otelcol --config=collector-config.yaml
```

The collector will start with:
- **Glean receiver** on `http://localhost:9888/submit/telemetry`
- **Collector metrics** on `http://localhost:8888/metrics`
- **ZPages diagnostics** on `http://localhost:55679/debug`

### 3. Send a Test Ping

```bash
curl -X POST http://localhost:9888/submit/telemetry \
  -H "Content-Type: application/json" \
  -d @example-ping.json
```

Or use the test script:

```bash
chmod +x test.sh
./test.sh
```

## What You'll See

### Metrics Output

All Glean metrics are converted to OpenTelemetry metrics:

```
Resource attributes:
     -> client.id: c641eacf-c30c-4171-b403-f077724e848a
     -> app.version: 1.2.3-beta
     -> os.type: iOS
     -> device.model.name: iPhone 15

Metrics:
     -> counter.app.opened: 5
     -> quantity.memory.used_mb: 256
     -> rate.network.error_rate.rate: 0.039370
     -> timing_distribution.page.load_time: Histogram(count=100, sum=15000)
```

### Event Logs Output

Glean events are converted to OpenTelemetry event logs:

```
LogRecord:
     Timestamp: 2024-01-28 10:00:01 +0000 UTC
     Body: button_clicked
     Attributes:
          -> event.name: button_clicked
          -> event.domain: ui
          -> button_id: submit
          -> screen: home
```

## Configuration

### Collector Configuration (`collector-config.yaml`)

```yaml
receivers:
  glean:
    endpoint: localhost:9888      # HTTP server address
    path: /submit/telemetry       # HTTP path for pings
    read_header_timeout: 20s
```

### Builder Configuration (`builder-config.yaml`)

Defines the custom collector components:
- Glean receiver (custom)
- OTLP receiver
- Batch processor
- Debug exporter
- OTLP exporter
- ZPages extension

## Customization

### Adding More Exporters

Edit `builder-config.yaml` to add exporters like Prometheus, Jaeger, etc.:

```yaml
exporters:
  - gomod: go.opentelemetry.io/collector/exporter/prometheusexporter v0.113.0
```

Then update `collector-config.yaml`:

```yaml
exporters:
  prometheus:
    endpoint: "0.0.0.0:8889"

service:
  pipelines:
    metrics:
      receivers: [glean]
      processors: [batch]
      exporters: [debug, prometheus]
```

### Changing the Receiver Port

Edit `collector-config.yaml`:

```yaml
receivers:
  glean:
    endpoint: localhost:8080  # Your desired port
```

## Troubleshooting

### Port Already in Use

If you see "bind: address already in use":
1. Check if another process is using the port: `lsof -i :9888`
2. Change the port in `collector-config.yaml`
3. Rebuild: `rm -rf dist && ~/go/bin/builder --config=builder-config.yaml`

### Invalid JSON

Ensure your Glean pings match the schema. Check `example-ping.json` for the correct structure.

### Missing Dependencies

Run `go mod tidy` to ensure all dependencies are downloaded.

## Next Steps

- Integrate with your Glean telemetry pipeline
- Configure production exporters (Prometheus, OTLP, etc.)
- Set up authentication if needed (using confighttp auth settings)
- Add custom processing logic in the receiver
- Deploy to your infrastructure

## Learn More

- [OpenTelemetry Collector Documentation](https://opentelemetry.io/docs/collector/)
- [Building Custom Components](https://opentelemetry.io/docs/collector/extend/custom-component/)
- [Glean Documentation](https://mozilla.github.io/glean/)
