# Testing Guide

## Running Tests

### Run All Tests

```bash
go test -v ./...
```

### Run Tests with Coverage

```bash
go test -v -cover ./...
```

### Generate Coverage Report

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Specific Test

```bash
go test -v -run TestConvertToMetrics
```

## Test Structure

### Config Tests (`config_test.go`)
- **TestConfigValidate**: Tests configuration validation
  - Valid config
  - Empty path (should fail validation)
- **TestCreateDefaultConfig**: Verifies default configuration values

### Factory Tests (`factory_test.go`)
- **TestNewFactory**: Verifies factory creation and type
- **TestCreateMetricsReceiver**: Tests metrics receiver creation
- **TestCreateLogsReceiver**: Tests logs receiver creation
- **TestCreateReceiverWithInvalidConfig**: Tests receiver creation with invalid config
- **TestSharedReceiverInstance**: Verifies that metrics and logs receivers share the same instance

### Receiver Tests (`receiver_test.go`)
- **TestReceiverStartStop**: Tests receiver lifecycle (start/stop)
- **TestReceiverHandleInvalidMethod**: Tests HTTP method validation (only POST allowed)
- **TestReceiverHandleInvalidJSON**: Tests invalid JSON payload handling
- **TestReceiverHandleValidPing**: End-to-end test with valid Glean ping
  - Verifies metrics consumption
  - Verifies logs consumption
- **TestReceiverMultipleStarts**: Tests that multiple Start() calls are safe (sync.Once)

### Converter Tests (`converter_test.go`)
- **TestConvertToMetrics**: Tests Glean → OTel metrics conversion
  - Resource attributes
  - Counter metrics
  - Boolean metrics
  - String metrics
  - Quantity metrics
- **TestConvertToEventLogs**: Tests Glean events → OTel event logs conversion
  - Resource attributes
  - Event name and domain attributes
  - Extra fields as attributes
  - Timestamp calculation
- **TestConvertDistributionMetric**: Tests distribution (histogram) conversion
  - Sum and count
  - Bucket values
- **TestConvertRateMetric**: Tests rate metric conversion
  - Calculated ratio
  - Numerator/denominator attributes
- **TestBoolToFloat**: Tests boolean to float conversion
- **TestToFloat64**: Tests type conversion helper function

## Test Coverage

Current coverage: **76.8%**

### Coverage by File
- `config.go`: Configuration and validation
- `factory.go`: Factory functions and shared receiver pattern
- `receiver.go`: HTTP server and request handling
- `converter.go`: Glean to OTel conversion logic

### Uncovered Areas
Some areas with lower coverage:
- Error paths in complex metric conversion
- Edge cases in HTTP error handling
- Shutdown error handling

## Adding New Tests

### Test Template

```go
func TestNewFeature(t *testing.T) {
    // Arrange
    input := createTestInput()

    // Act
    result, err := functionUnderTest(input)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, expectedValue, result)
}
```

### Testing HTTP Endpoints

```go
func TestNewEndpoint(t *testing.T) {
    // Create receiver with unique port
    cfg := &Config{
        Path: "/test",
    }
    cfg.ServerConfig.Endpoint = "localhost:19999" // Use unique port

    receiver, err := newGleanReceiver(cfg, ...)
    require.NoError(t, err)

    // Start receiver
    ctx := context.Background()
    err = receiver.Start(ctx, componenttest.NewNopHost())
    require.NoError(t, err)
    defer receiver.Shutdown(ctx)

    // Give server time to start
    time.Sleep(100 * time.Millisecond)

    // Make HTTP request
    resp, err := http.Post("http://localhost:19999/test", ...)
    require.NoError(t, err)
    defer resp.Body.Close()

    // Assert response
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

### Testing Conversions

```go
func TestNewConversion(t *testing.T) {
    ping := &GleanPing{
        ClientInfo: ClientInfo{...},
        PingInfo: PingInfo{...},
        Metrics: map[string]interface{}{
            "new_metric_type": map[string]interface{}{
                "metric_name": expectedValue,
            },
        },
    }

    metrics, err := convertToMetrics(ping)
    require.NoError(t, err)

    // Navigate to the metric
    scopeMetrics := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0)
    metric := scopeMetrics.Metrics().At(0)

    // Assert metric properties
    assert.Equal(t, "new_metric_type.metric_name", metric.Name())
}
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'

      - name: Run tests
        run: go test -v -cover ./...

      - name: Generate coverage
        run: go test -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
```

## Benchmarking

### Run Benchmarks

```bash
go test -bench=. -benchmem
```

### Example Benchmark

```go
func BenchmarkConvertToMetrics(b *testing.B) {
    ping := createTestPing()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = convertToMetrics(ping)
    }
}
```

## Integration Testing

For full integration tests with a running collector:

```bash
# Start collector
./dist/glean-otelcol --config=collector-config.yaml &
COLLECTOR_PID=$!

# Run integration tests
go test -tags=integration -v ./...

# Cleanup
kill $COLLECTOR_PID
```
