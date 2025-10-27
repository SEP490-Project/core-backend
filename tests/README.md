# Test Directory Structure

This directory contains all tests for the core-backend project, organized by test type and scope.

## Directory Overview

```
tests/
├── unit_tests/                 # Unit tests for individual functions/methods
│   ├── utils/                  # Tests for pkg/utils functions
│   │   └── payment_cycle_calculator_test.go
│   └── service/                # Tests for service layer helpers
│       └── campaign_service_helpers_test.go
│
├── integration_tests/          # Integration tests for end-to-end flows
│   ├── campaign/               # Campaign suggestion integration tests
│   │   ├── advertising_test.go
│   │   ├── affiliate_test.go
│   │   ├── brand_ambassador_test.go
│   │   └── co_producing_test.go
│   └── contract_payment/       # Contract payment integration tests
│       └── payment_creation_test.go
│
├── performance_tests/          # Load and performance tests
│   ├── campaign_suggestion_bench_test.go
│   └── concurrent_load_test.go
│
├── fixtures/                   # Reusable test fixtures and factories
│   └── contracts/              # Contract model fixtures
│       ├── advertising.go
│       ├── affiliate.go
│       ├── brand_ambassador.go
│       └── co_producing.go
│
└── testdata/                   # Static test data files (JSON, YAML, etc.)
    ├── advertising_contract.json
    ├── affiliate_contract.json
    ├── brand_ambassador_contract.json
    └── co_producing_contract.json
```

## Test Types

### Unit Tests (`unit_tests/`)

Test individual functions in isolation with mocked dependencies.

**Scope:**
- `pkg/utils/payment_cycle_calculator.go` functions
- `internal/application/service/campaign_service_helpers.go` functions
- Pure functions with no external dependencies

**Run unit tests:**
```bash
go test ./tests/unit_tests/... -v
```

### Integration Tests (`integration_tests/`)

Test complete workflows with real or test database connections.

**Scope:**
- Campaign suggestion for each contract type (ADVERTISING, AFFILIATE, BRAND_AMBASSADOR, CO_PRODUCING)
- Contract payment creation
- Milestone-payment alignment validation
- End-to-end request-response flows

**Run integration tests:**
```bash
go test ./tests/integration_tests/... -v
```

**Prerequisites:**
- Test database connection (PostgreSQL)
- Test fixtures loaded
- Environment variables configured

### Performance Tests (`performance_tests/`)

Benchmark and load tests for critical operations.

**Scope:**
- Campaign suggestion response times by deliverable count
- Concurrent request handling (100+ simultaneous requests)
- Memory usage profiling
- Database query optimization validation

**Run performance tests:**
```bash
go test ./tests/performance_tests/... -bench=. -benchmem
```

**Run specific benchmark:**
```bash
go test ./tests/performance_tests/ -bench=BenchmarkCampaignSuggestion -benchtime=10s
```

### Fixtures (`fixtures/`)

Reusable test data factories for creating model instances.

**Usage:**
```go
import "core-backend/tests/fixtures/contracts"

// Create test contract
contract := contracts.NewAdvertisingContract(
    contracts.WithStartDate(time.Now()),
    contracts.WithItemCount(10),
)
```

### Test Data (`testdata/`)

Static JSON/YAML files representing realistic contract data.

**Usage:**
```go
import "core-backend/tests/testdata"

// Load test contract from JSON
contractJSON := testdata.LoadAdvertisingContract()
var contract model.Contract
json.Unmarshal(contractJSON, &contract)
```

## Running Tests

### Run all tests:
```bash
go test ./tests/... -v
```

### Run specific test suite:
```bash
# Unit tests only
go test ./tests/unit_tests/... -v

# Integration tests only
go test ./tests/integration_tests/... -v

# Performance tests only
go test ./tests/performance_tests/... -bench=. -benchmem
```

### Run with coverage:
```bash
go test ./tests/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run specific test:
```bash
go test ./tests/unit_tests/utils -run TestCalculateMonthlyPaymentDates -v
```

## Test Naming Conventions

### Test Files:
- Unit tests: `*_test.go` (same package as tested code or `_test` suffix)
- Integration tests: `*_integration_test.go` or in `integration_tests/` directory
- Benchmark tests: `*_bench_test.go` or use `Benchmark*` function prefix

### Test Functions:
- Unit tests: `TestFunctionName_Scenario`
- Table-driven tests: `TestFunctionName` with subtests
- Benchmarks: `BenchmarkFunctionName`

**Examples:**
```go
func TestCalculateMonthlyPaymentDates_ValidInput(t *testing.T) { }
func TestCalculateMonthlyPaymentDates_YearBoundary(t *testing.T) { }
func TestExtractAdvertisingTasks_EmptyDeliverables(t *testing.T) { }
func BenchmarkCampaignSuggestion_10Items(b *testing.B) { }
```

## Test Coverage Goals

- **Unit Tests**: 80%+ coverage for utility functions and helpers
- **Integration Tests**: All critical user flows covered
- **Performance Tests**: Response time targets met:
  - 1-10 items: < 100ms
  - 11-50 items: < 300ms
  - 51-100 items: < 500ms

## CI/CD Integration

Tests are automatically run on:
- Pull request creation
- Push to main branch
- Pre-deployment validation

**CI Pipeline:**
1. Run unit tests (fast feedback)
2. Run integration tests (requires test DB)
3. Run performance tests (nightly/weekly)
4. Generate coverage reports
5. Enforce minimum coverage thresholds

## Adding New Tests

1. **Determine test type** (unit, integration, performance)
2. **Create test file** in appropriate directory
3. **Write test cases** following naming conventions
4. **Add fixtures** if reusable data needed
5. **Update this README** if new patterns introduced

## Test Dependencies

### Testing Libraries:
- `testing` - Standard Go testing package
- `github.com/stretchr/testify` - Assertions and mocking
  - `assert` - Assertions
  - `mock` - Mock generation
  - `suite` - Test suites
- `github.com/DATA-DOG/go-sqlmock` - SQL mock for database tests (optional)

### Install dependencies:
```bash
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/mock
go get github.com/stretchr/testify/suite
```

## Best Practices

1. **Isolation**: Each test should be independent
2. **Cleanup**: Always clean up test data (use `t.Cleanup()`)
3. **Readability**: Use table-driven tests for multiple scenarios
4. **Speed**: Unit tests should be fast (<10ms each)
5. **Reliability**: Tests should be deterministic (no random data without seed)
6. **Documentation**: Comment complex test scenarios
7. **Fixtures**: Reuse fixtures, don't duplicate test data

## Troubleshooting

### Tests fail with database connection errors:
- Ensure test database is running
- Check DATABASE_URL environment variable
- Verify database migrations are up to date

### Performance tests show high latency:
- Check if running with `-race` flag (adds overhead)
- Ensure test database has indexes
- Profile with `go test -cpuprofile=cpu.out`

### Coverage reports missing files:
- Ensure all packages are tested
- Run with `-covermode=atomic` for accurate coverage
- Check `.gitignore` doesn't exclude test files
