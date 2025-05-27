#!/bin/bash

# Comprehensive test runner for distributed LLM project

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "🧪 Running comprehensive test suite for distributed LLM project"
echo "============================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
UNIT_TESTS=true
INTEGRATION_TESTS=true
E2E_TESTS=true
FUZZ_TESTS=true
BENCHMARK_TESTS=true
COVERAGE_REPORT=true
TIMEOUT="5m"
FUZZ_TIME="30s"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --unit-only)
      INTEGRATION_TESTS=false
      E2E_TESTS=false
      FUZZ_TESTS=false
      shift
      ;;
    --no-fuzz)
      FUZZ_TESTS=false
      shift
      ;;
    --no-e2e)
      E2E_TESTS=false
      shift
      ;;
    --no-integration)
      INTEGRATION_TESTS=false
      shift
      ;;
    --quick)
      FUZZ_TIME="10s"
      E2E_TESTS=false
      shift
      ;;
    --timeout)
      TIMEOUT="$2"
      shift
      shift
      ;;
    --fuzz-time)
      FUZZ_TIME="$2"
      shift
      shift
      ;;
    --help)
      echo "Usage: $0 [options]"
      echo "Options:"
      echo "  --unit-only       Run only unit tests"
      echo "  --no-fuzz         Skip fuzz tests"
      echo "  --no-e2e          Skip end-to-end tests"
      echo "  --no-integration  Skip integration tests"
      echo "  --quick           Quick test run (shorter fuzz time, no e2e)"
      echo "  --timeout TIME    Set test timeout (default: 5m)"
      echo "  --fuzz-time TIME  Set fuzz test duration (default: 30s)"
      echo "  --help            Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown option $1"
      exit 1
      ;;
  esac
done

# Functions
run_test_category() {
  local category="$1"
  local description="$2"
  local command="$3"
  
  echo -e "\n${BLUE}=== $description ===${NC}"
  echo "Command: $command"
  
  start_time=$(date +%s)
  if eval "$command"; then
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    echo -e "${GREEN}✅ $category completed successfully (${duration}s)${NC}"
    return 0
  else
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    echo -e "${RED}❌ $category failed (${duration}s)${NC}"
    return 1
  fi
}

# Check prerequisites
echo -e "${BLUE}🔍 Checking prerequisites...${NC}"

if ! command -v go &> /dev/null; then
  echo -e "${RED}❌ Go is not installed${NC}"
  exit 1
fi

echo -e "${GREEN}✅ Go version: $(go version)${NC}"

# Check if we can build
if ! make build &> /dev/null; then
  echo -e "${YELLOW}⚠️  Build failed - some E2E tests may be skipped${NC}"
fi

# Initialize test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Run unit tests
if [ "$UNIT_TESTS" = true ]; then
  TOTAL_TESTS=$((TOTAL_TESTS + 1))
  if run_test_category "Unit Tests" "Running unit tests with coverage" \
    "go test -timeout $TIMEOUT -coverprofile=coverage.out -v ./pkg/... ./internal/..."; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
  else
    FAILED_TESTS=$((FAILED_TESTS + 1))
  fi
fi

# Run integration tests
if [ "$INTEGRATION_TESTS" = true ]; then
  TOTAL_TESTS=$((TOTAL_TESTS + 1))
  if run_test_category "Integration Tests" "Running integration tests" \
    "go test -timeout $TIMEOUT -v ./test/integration/..."; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
  else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    echo -e "${YELLOW}⚠️  Integration test failures may be due to missing dependencies${NC}"
  fi
fi

# Run end-to-end tests
if [ "$E2E_TESTS" = true ]; then
  TOTAL_TESTS=$((TOTAL_TESTS + 1))
  if run_test_category "End-to-End Tests" "Running end-to-end tests" \
    "go test -timeout $TIMEOUT -v ./test/e2e/..."; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
  else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    echo -e "${YELLOW}⚠️  E2E test failures may be due to missing binaries or network issues${NC}"
  fi
fi

# Run fuzz tests
if [ "$FUZZ_TESTS" = true ]; then
  TOTAL_TESTS=$((TOTAL_TESTS + 1))
  if run_test_category "Fuzz Tests" "Running fuzz tests" \
    "go test -timeout $TIMEOUT -fuzz=. -fuzztime=$FUZZ_TIME ./test/fuzz/..."; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
  else
    FAILED_TESTS=$((FAILED_TESTS + 1))
  fi
fi

# Run benchmark tests
if [ "$BENCHMARK_TESTS" = true ]; then
  TOTAL_TESTS=$((TOTAL_TESTS + 1))
  if run_test_category "Benchmark Tests" "Running benchmark tests" \
    "go test -timeout $TIMEOUT -bench=. -benchmem ./pkg/... ./internal/..."; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
  else
    FAILED_TESTS=$((FAILED_TESTS + 1))
  fi
fi

# Generate coverage report
if [ "$COVERAGE_REPORT" = true ] && [ -f coverage.out ]; then
  echo -e "\n${BLUE}=== Coverage Report ===${NC}"
  
  # Generate HTML coverage report
  go tool cover -html=coverage.out -o coverage.html
  
  # Show coverage summary
  coverage_percent=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | sed 's/%//')
  
  if (( $(echo "$coverage_percent >= 80" | bc -l) )); then
    echo -e "${GREEN}✅ Coverage: $coverage_percent%${NC}"
  elif (( $(echo "$coverage_percent >= 60" | bc -l) )); then
    echo -e "${YELLOW}⚠️  Coverage: $coverage_percent%${NC}"
  else
    echo -e "${RED}❌ Coverage: $coverage_percent% (target: 80%)${NC}"
  fi
  
  echo -e "${BLUE}📊 Coverage report generated: coverage.html${NC}"
  
  # Show top uncovered functions
  echo -e "\n${BLUE}Top uncovered functions:${NC}"
  go tool cover -func=coverage.out | grep -E "^[^[:space:]]+.*[[:space:]]+0\.0%" | head -10 || echo "No uncovered functions found"
fi

# Test summary
echo -e "\n${BLUE}=============================================================${NC}"
echo -e "${BLUE}🧪 Test Suite Summary${NC}"
echo -e "${BLUE}=============================================================${NC}"

echo "Total test categories: $TOTAL_TESTS"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"

if [ "$FAILED_TESTS" -eq 0 ]; then
  echo -e "\n${GREEN}🎉 All test categories passed!${NC}"
  exit 0
else
  echo -e "\n${RED}💥 Some test categories failed${NC}"
  exit 1
fi
