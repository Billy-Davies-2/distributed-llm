# Test Configuration for Distributed LLM Project

## Test Categories

### Unit Tests
- **Location**: `pkg/`, `internal/`
- **Purpose**: Test individual components in isolation
- **Coverage Target**: 80%
- **Run Command**: `go test ./pkg/... ./internal/...`

### Integration Tests
- **Location**: `test/integration/`
- **Purpose**: Test component interactions
- **Dependencies**: May require network access, temporary files
- **Run Command**: `go test ./test/integration/...`

### End-to-End Tests
- **Location**: `test/e2e/`
- **Purpose**: Test complete workflows
- **Dependencies**: Built binaries, available ports
- **Run Command**: `go test ./test/e2e/...`

### Fuzz Tests
- **Location**: `test/fuzz/`
- **Purpose**: Test input validation and edge cases
- **Run Command**: `go test -fuzz=. ./test/fuzz/...`

## Test Requirements

### Prerequisites
- Go 1.19 or later
- Make
- Built binaries (for E2E tests)
- Available network ports (for integration/E2E tests)

### Optional Dependencies
- Kubernetes cluster (for K8s integration tests)
- Docker (for container tests)
- Minikube (for local cluster tests)

## Running Tests

### Quick Test Run
```bash
./scripts/run-tests.sh --quick
```

### Full Test Suite
```bash
./scripts/run-tests.sh
```

### Specific Test Categories
```bash
# Unit tests only
./scripts/run-tests.sh --unit-only

# Without fuzz tests
./scripts/run-tests.sh --no-fuzz

# Without E2E tests
./scripts/run-tests.sh --no-e2e
```

### Using Make Targets
```bash
# Run all tests
make test-all

# Run specific test types
make test-unit
make test-integration
make test-e2e
make test-fuzz

# Generate coverage report
make test-coverage
```

## Test Data and Fixtures

### Test Ports
- Tests automatically find available ports using `net.Listen(":0")`
- Integration tests use port ranges 8000-9000
- E2E tests use port ranges 9000-10000

### Test Models
- Small test model: 1GB, 12 layers
- Medium test model: 7GB, 32 layers  
- Large test model: 13GB, 40 layers

### Test Clusters
- Single node: Basic functionality
- 3 nodes: Distributed functionality
- 5 nodes: Failure scenarios

## Continuous Integration

### GitHub Actions
Tests are designed to run in CI environments:
- Unit tests: Always run
- Integration tests: Run if dependencies available
- E2E tests: Run if binaries can be built
- Fuzz tests: Run with short duration

### Performance Benchmarks
Benchmark tests track performance over time:
- Network operations
- Model loading
- Resource detection
- UI rendering

## Debugging Failed Tests

### Common Issues

1. **Port conflicts**: Tests find available ports automatically
2. **Build failures**: Run `make build` before E2E tests
3. **Network timeouts**: Increase timeout with `--timeout` flag
4. **Fuzz test failures**: Review fuzz inputs in testdata directory

### Verbose Output
Add `-v` flag for detailed test output:
```bash
go test -v ./...
```

### Test Specific Functions
```bash
go test -run TestSpecificFunction ./pkg/models
```

## Test Coverage

### Coverage Reports
- HTML report: `coverage.html`
- Terminal summary: Shown after unit tests
- Target coverage: 80%

### Improving Coverage
1. Identify uncovered functions in report
2. Add unit tests for missing paths
3. Add integration tests for cross-component flows
4. Add E2E tests for user workflows

## Mock and Test Utilities

### Available Mocks
- `MockBroadcaster`: For testing agent broadcasting
- Test cluster utilities for E2E tests
- Port finding utilities for network tests

### Test Helpers
- `findAvailablePort()`: Get available network port
- `TestCluster`: Manage test cluster lifecycle
- Various validation helpers in fuzz tests
