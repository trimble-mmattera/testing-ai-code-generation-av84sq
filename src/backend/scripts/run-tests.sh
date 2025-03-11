#!/bin/bash
# run-tests.sh - Test runner for Document Management Platform
# 
# This script runs different types of tests for the Document Management Platform:
# - Unit tests
# - Integration tests
# - End-to-end tests
#
# It can also generate coverage reports and check against coverage thresholds.

set -e

# Default values for environment variables
TEST_TYPE="${TEST_TYPE:-all}"
COVERAGE="${COVERAGE:-false}"
COVERAGE_DIR="${COVERAGE_DIR:-coverage}"
COVERAGE_THRESHOLD="${COVERAGE_THRESHOLD:-80}"
VERBOSE="${VERBOSE:-false}"
TEST_PATTERN="${TEST_PATTERN:-"./..."}"
GO_TEST_FLAGS="${GO_TEST_FLAGS:-"-v"}"
ENV="${ENV:-test}"

# Verify required tools are installed
check_required_tools() {
  if ! command -v go &> /dev/null; then
    echo "Error: go is not installed or not in PATH" >&2
    exit 1
  fi
  
  if [[ "$TEST_TYPE" == "integration" || "$TEST_TYPE" == "e2e" || "$TEST_TYPE" == "all" ]]; then
    if ! command -v docker-compose &> /dev/null; then
      echo "Error: docker-compose is not installed or not in PATH, required for integration and e2e tests" >&2
      exit 1
    fi
  fi
  
  if [[ "$COVERAGE" == "true" ]]; then
    if ! command -v jq &> /dev/null; then
      echo "Warning: jq is not installed or not in PATH, coverage reporting may be limited" >&2
    fi
  fi
}

# Print usage information
print_usage() {
  echo "Usage: $(basename "$0") [options]"
  echo
  echo "Options:"
  echo "  -t, --test-type TYPE     Test type to run (unit, integration, e2e, all) [default: all]"
  echo "  -c, --coverage           Generate code coverage reports [default: false]"
  echo "  -d, --coverage-dir DIR   Directory for coverage reports [default: coverage]"
  echo "  -p, --pattern PATTERN    Test file pattern [default: ./...]"
  echo "  -T, --threshold NUM      Coverage threshold percentage [default: 80]"
  echo "  -v, --verbose            Enable verbose output [default: false]"
  echo "  -e, --env ENV            Environment to use (test, dev, etc.) [default: test]"
  echo "  -h, --help               Print this help message"
  echo
  echo "Environment variables:"
  echo "  TEST_TYPE                Same as --test-type"
  echo "  COVERAGE                 Same as --coverage (set to 'true' to enable)"
  echo "  COVERAGE_DIR             Same as --coverage-dir"
  echo "  COVERAGE_THRESHOLD       Same as --threshold"
  echo "  VERBOSE                  Same as --verbose (set to 'true' to enable)"
  echo "  TEST_PATTERN             Same as --pattern"
  echo "  ENV                      Same as --env"
  echo "  GO_TEST_FLAGS            Additional flags to pass to go test"
}

# Parse command-line arguments
parse_args() {
  while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
      -t|--test-type)
        TEST_TYPE="$2"
        shift 2
        ;;
      -c|--coverage)
        COVERAGE="true"
        shift
        ;;
      -d|--coverage-dir)
        COVERAGE_DIR="$2"
        shift 2
        ;;
      -p|--pattern)
        TEST_PATTERN="$2"
        shift 2
        ;;
      -T|--threshold)
        COVERAGE_THRESHOLD="$2"
        shift 2
        ;;
      -v|--verbose)
        VERBOSE="true"
        shift
        ;;
      -e|--env)
        ENV="$2"
        shift 2
        ;;
      -h|--help)
        print_usage
        exit 0
        ;;
      *)
        echo "Unknown option: $1" >&2
        print_usage
        exit 1
        ;;
    esac
  done
  
  # Validate test type
  case $TEST_TYPE in
    unit|integration|e2e|all)
      # Valid test type
      ;;
    *)
      echo "Error: Invalid test type '$TEST_TYPE'. Must be one of: unit, integration, e2e, all" >&2
      exit 1
      ;;
  esac
}

# Check if dependencies are running
check_dependencies() {
  echo "Checking required dependencies..."
  local failed=0
  
  # Check PostgreSQL
  if ! pg_isready -h localhost -p 5432 -U postgres &> /dev/null; then
    echo "PostgreSQL is not running"
    failed=1
  else
    echo "PostgreSQL is running"
  fi
  
  # Check S3-compatible storage (using aws-cli)
  if command -v aws &> /dev/null; then
    if ! aws --endpoint-url=http://localhost:4566 s3 ls &> /dev/null; then
      echo "LocalStack S3 is not running"
      failed=1
    else
      echo "LocalStack S3 is running"
    fi
  else
    echo "Warning: aws-cli not found, skipping S3 check"
  fi
  
  # Check Elasticsearch
  if ! curl -s -f http://localhost:9200/_cluster/health &> /dev/null; then
    echo "Elasticsearch is not running"
    failed=1
  else
    echo "Elasticsearch is running"
  fi
  
  # For E2E tests, check ClamAV
  if [[ "$TEST_TYPE" == "e2e" || "$TEST_TYPE" == "all" ]]; then
    if ! echo PING | nc -w 2 localhost 3310 &> /dev/null; then
      echo "ClamAV is not running"
      failed=1
    else
      echo "ClamAV is running"
    fi
  fi
  
  return $failed
}

# Start dependencies using docker-compose
start_dependencies() {
  echo "Starting test dependencies with docker-compose..."
  
  # Use different docker-compose files based on test type
  local compose_file="docker-compose.test.yml"
  local services=""
  
  if [[ "$TEST_TYPE" == "integration" || "$TEST_TYPE" == "all" ]]; then
    services="postgres localstack elasticsearch redis"
  fi
  
  if [[ "$TEST_TYPE" == "e2e" || "$TEST_TYPE" == "all" ]]; then
    services="$services clamav"
  fi
  
  if [[ -n "$services" ]]; then
    docker-compose -f $compose_file up -d $services
    
    # Wait for services to be ready
    echo "Waiting for services to be ready..."
    sleep 10
    
    # Verify services are running
    if ! check_dependencies; then
      echo "Failed to start all required dependencies"
      return 1
    fi
  fi
  
  return 0
}

# Run unit tests
run_unit_tests() {
  echo "Running unit tests..."
  local exit_code=0
  local cover_args=""
  local test_pattern="./pkg/... ./domain/... ./application/..."
  
  # If TEST_PATTERN is set, use it instead of the default
  if [[ "$TEST_PATTERN" != "./..." ]]; then
    test_pattern="$TEST_PATTERN"
  fi
  
  # Set up coverage flags if enabled
  if [[ "$COVERAGE" == "true" ]]; then
    mkdir -p "$COVERAGE_DIR"
    cover_args="-coverprofile=$COVERAGE_DIR/unit.out -covermode=atomic"
  fi
  
  # Verbose output
  if [[ "$VERBOSE" == "true" ]]; then
    echo "Test pattern: $test_pattern"
    echo "Coverage enabled: $COVERAGE"
    if [[ "$COVERAGE" == "true" ]]; then
      echo "Coverage output: $COVERAGE_DIR/unit.out"
    fi
  fi
  
  # Run the tests
  echo "go test $GO_TEST_FLAGS $cover_args -tags=unit $test_pattern"
  if ! go test $GO_TEST_FLAGS $cover_args -tags=unit $test_pattern; then
    exit_code=1
  fi
  
  # Generate coverage report if enabled
  if [[ "$COVERAGE" == "true" && $exit_code -eq 0 ]]; then
    generate_coverage_report "$COVERAGE_DIR/unit.out" "$COVERAGE_DIR/unit-html"
    check_coverage "$COVERAGE_DIR/unit.out"
    exit_code=$?
  fi
  
  return $exit_code
}

# Run integration tests
run_integration_tests() {
  echo "Running integration tests..."
  local exit_code=0
  local cover_args=""
  local test_pattern="./test/integration/..."
  
  # If TEST_PATTERN is set, use it instead of the default
  if [[ "$TEST_PATTERN" != "./..." ]]; then
    test_pattern="$TEST_PATTERN"
  fi
  
  # Check if dependencies are running
  if ! check_dependencies; then
    echo "Starting required dependencies..."
    if ! start_dependencies; then
      echo "Failed to start dependencies for integration tests"
      return 1
    fi
  fi
  
  # Set up coverage flags if enabled
  if [[ "$COVERAGE" == "true" ]]; then
    mkdir -p "$COVERAGE_DIR"
    cover_args="-coverprofile=$COVERAGE_DIR/integration.out -covermode=atomic"
  fi
  
  # Set up environment variables for integration tests
  export GO_ENV="$ENV"
  export CONFIG_PATH="$(pwd)/config"
  
  # Verbose output
  if [[ "$VERBOSE" == "true" ]]; then
    echo "Test pattern: $test_pattern"
    echo "Coverage enabled: $COVERAGE"
    echo "Environment: $ENV"
    echo "Config path: $CONFIG_PATH"
    if [[ "$COVERAGE" == "true" ]]; then
      echo "Coverage output: $COVERAGE_DIR/integration.out"
    fi
  fi
  
  # Run the tests
  echo "go test $GO_TEST_FLAGS $cover_args -tags=integration $test_pattern"
  if ! go test $GO_TEST_FLAGS $cover_args -tags=integration $test_pattern; then
    exit_code=1
  fi
  
  # Generate coverage report if enabled
  if [[ "$COVERAGE" == "true" && $exit_code -eq 0 ]]; then
    generate_coverage_report "$COVERAGE_DIR/integration.out" "$COVERAGE_DIR/integration-html"
    check_coverage "$COVERAGE_DIR/integration.out"
    exit_code=$?
  fi
  
  return $exit_code
}

# Run end-to-end tests
run_e2e_tests() {
  echo "Running end-to-end tests..."
  local exit_code=0
  local test_pattern="./test/e2e/..."
  
  # If TEST_PATTERN is set, use it instead of the default
  if [[ "$TEST_PATTERN" != "./..." ]]; then
    test_pattern="$TEST_PATTERN"
  fi
  
  # Check if dependencies are running
  if ! check_dependencies; then
    echo "Starting required dependencies..."
    if ! start_dependencies; then
      echo "Failed to start dependencies for E2E tests"
      return 1
    fi
  fi
  
  # Set up environment variables for E2E tests
  export GO_ENV="$ENV"
  export CONFIG_PATH="$(pwd)/config"
  
  # Verbose output
  if [[ "$VERBOSE" == "true" ]]; then
    echo "Test pattern: $test_pattern"
    echo "Environment: $ENV"
    echo "Config path: $CONFIG_PATH"
  fi
  
  # Run the tests (note: we typically don't measure coverage for E2E tests)
  echo "go test $GO_TEST_FLAGS -tags=e2e $test_pattern"
  if ! go test $GO_TEST_FLAGS -tags=e2e $test_pattern; then
    exit_code=1
  fi
  
  return $exit_code
}

# Check if code coverage meets the threshold
check_coverage() {
  local coverage_file="$1"
  local exit_code=0
  
  if [[ ! -f "$coverage_file" ]]; then
    echo "Coverage file not found: $coverage_file"
    return 1
  fi
  
  # Extract coverage percentage
  local coverage_percent
  coverage_percent=$(go tool cover -func="$coverage_file" | grep total: | awk '{print $3}' | tr -d '%')
  
  # Compare with threshold
  local coverage_int=${coverage_percent%.*}
  if [[ -z "$coverage_int" ]]; then
    coverage_int=0
  fi
  
  if (( coverage_int < COVERAGE_THRESHOLD )); then
    echo "❌ Code coverage is below threshold: $coverage_percent% < $COVERAGE_THRESHOLD%"
    exit_code=1
  else
    echo "✅ Code coverage meets threshold: $coverage_percent% >= $COVERAGE_THRESHOLD%"
    exit_code=0
  fi
  
  return $exit_code
}

# Generate HTML coverage report
generate_coverage_report() {
  local coverage_file="$1"
  local output_dir="$2"
  local exit_code=0
  
  if [[ ! -f "$coverage_file" ]]; then
    echo "Coverage file not found: $coverage_file"
    return 1
  fi
  
  # Create output directory
  mkdir -p "$output_dir"
  
  # Generate HTML report
  echo "Generating HTML coverage report to $output_dir/coverage.html"
  if ! go tool cover -html="$coverage_file" -o="$output_dir/coverage.html"; then
    echo "Failed to generate HTML coverage report"
    exit_code=1
  fi
  
  return $exit_code
}

# Combine coverage profiles
combine_coverage_reports() {
  local output_file="$COVERAGE_DIR/coverage.out"
  local exit_code=0
  
  echo "Combining coverage reports..."
  
  # Use gocovmerge if available, otherwise use a simple cat approach
  if command -v gocovmerge &> /dev/null; then
    gocovmerge "$COVERAGE_DIR"/*.out > "$output_file"
  else
    # First file with full header
    if [[ -f "$COVERAGE_DIR/unit.out" ]]; then
      cp "$COVERAGE_DIR/unit.out" "$output_file"
    elif [[ -f "$COVERAGE_DIR/integration.out" ]]; then
      cp "$COVERAGE_DIR/integration.out" "$output_file"
    else
      echo "No coverage files found to combine"
      return 1
    fi
    
    # Append other files skipping the mode line
    for f in "$COVERAGE_DIR"/*.out; do
      if [[ "$f" != "$output_file" ]]; then
        tail -n +2 "$f" >> "$output_file"
      fi
    done
  fi
  
  # Generate combined report
  if [[ -f "$output_file" ]]; then
    generate_coverage_report "$output_file" "$COVERAGE_DIR/html"
    check_coverage "$output_file"
    exit_code=$?
  fi
  
  return $exit_code
}

# Main execution

# Check required tools
check_required_tools

# Parse arguments
parse_args "$@"

# Create coverage directory if needed
if [[ "$COVERAGE" == "true" ]]; then
  mkdir -p "$COVERAGE_DIR"
fi

# Run tests based on test type
exit_code=0

case $TEST_TYPE in
  unit)
    run_unit_tests
    exit_code=$?
    ;;
  integration)
    run_integration_tests
    exit_code=$?
    ;;
  e2e)
    run_e2e_tests
    exit_code=$?
    ;;
  all)
    # Run all test types, but continue even if one fails
    # We'll set the final exit code based on the combined results
    unit_result=0
    integration_result=0
    e2e_result=0
    
    run_unit_tests
    unit_result=$?
    
    run_integration_tests
    integration_result=$?
    
    run_e2e_tests
    e2e_result=$?
    
    # Combine coverage reports if enabled
    if [[ "$COVERAGE" == "true" ]]; then
      combine_coverage_reports
    fi
    
    # Set exit code to non-zero if any test failed
    if [[ $unit_result -ne 0 || $integration_result -ne 0 || $e2e_result -ne 0 ]]; then
      exit_code=1
    fi
    ;;
esac

# Print summary
echo
echo "Test Summary:"
echo "============="
echo "Test Type: $TEST_TYPE"
if [[ "$COVERAGE" == "true" ]]; then
  echo "Coverage Reports: $COVERAGE_DIR"
fi
echo "Exit Code: $exit_code"

exit $exit_code