# Quay CLI

A CLI tool for managing Docker Compose operations with port mapping support and ingress routing.

## Development

### Prerequisites

- Go 1.24 or later
- Docker (for integration tests)
- mkcert (for SSL certificate generation)

### Running Tests

The project uses [Testify](https://github.com/stretchr/testify) for unit tests and [testcontainers-go](https://github.com/testcontainers/testcontainers-go) for integration tests.

To run all tests:

```bash
go test ./...
```

To run tests for a specific package:

```bash
go test ./config/...  # Run config package tests
go test ./compose/... # Run compose package tests
go test ./ingress/... # Run ingress package tests
```

To run tests with verbose output:

```bash
go test -v ./...
```

To run tests with coverage:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out  # View coverage in browser
```

### Test Structure

The project uses a common test suite (`testutil.BaseSuite`) that provides shared functionality for all test suites:

- Temporary directory creation and cleanup
- Helper functions for creating test files
- Common assertions and utilities

Each package contains:
- Unit tests in `*_test.go` files
- A smoke test in `smoke_test.go` that verifies basic functionality
- Integration tests where appropriate (using testcontainers-go)

### Continuous Integration

Tests are automatically run in CI on each pull request and push to main. The CI pipeline:

1. Runs all unit tests
2. Runs integration tests
3. Checks test coverage
4. Runs linters

## License

[MIT License](LICENSE) 