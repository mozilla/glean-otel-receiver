# Contributing to Glean OpenTelemetry Receiver

Thank you for your interest in contributing! This document provides guidelines for contributing to the Glean OpenTelemetry Receiver.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR-USERNAME/glean-otel-receiver`
3. Add upstream remote: `git remote add upstream https://github.com/mozilla/glean-otel-receiver`
4. Create a feature branch: `git checkout -b feature/your-feature-name`

## Development Setup

### Prerequisites

- Go 1.22 or later
- OpenTelemetry Collector Builder (`go install go.opentelemetry.io/collector/cmd/builder@v0.113.0`)

### Building

```bash
# Build the receiver module
go build -v ./...

# Build the custom collector
~/go/bin/builder --config=builder-config.yaml
```

### Testing

```bash
# Run all tests
go test -v ./...

# Run with coverage
go test -v -cover ./...

# Run specific test
go test -v -run TestConvertToMetrics
```

See [TESTING.md](TESTING.md) for detailed testing documentation.

## Making Changes

### Code Style

- Follow standard Go conventions and [Effective Go](https://golang.org/doc/effective_go.html)
- Run `go fmt` before committing
- Use meaningful variable and function names
- Add comments for exported functions and types

### Commit Messages

- Use clear, descriptive commit messages
- Start with a verb in present tense (e.g., "Add", "Fix", "Update")
- Reference issues when applicable (e.g., "Fix #123: Handle empty metrics")

### Pull Requests

1. Ensure all tests pass: `go test -v ./...`
2. Add tests for new functionality
3. Update documentation if needed
4. Push to your fork and create a pull request
5. Fill out the pull request template
6. Wait for review from maintainers

### PR Checklist

- [ ] Tests pass locally
- [ ] New tests added for new functionality
- [ ] Documentation updated
- [ ] Code follows project style
- [ ] Commit messages are clear

## Testing Guidelines

- Write unit tests for all new code
- Aim for >70% code coverage
- Test both success and error cases
- Use table-driven tests when appropriate
- Mock external dependencies

## Documentation

- Update README.md for user-facing changes
- Update TESTING.md for test-related changes
- Add inline comments for complex logic
- Include examples in documentation

## Reporting Issues

When reporting issues, please include:

- Go version (`go version`)
- Operating system
- Steps to reproduce
- Expected behavior
- Actual behavior
- Error messages or logs

## Code of Conduct

This project follows Mozilla's [Community Participation Guidelines](https://www.mozilla.org/en-US/about/governance/policies/participation/).

## Questions?

- Open an issue for bugs or feature requests
- Join the discussion in existing issues
- Reach out to the Glean team

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
