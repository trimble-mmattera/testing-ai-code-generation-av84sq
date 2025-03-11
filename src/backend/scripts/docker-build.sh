#!/bin/bash
#
# This script builds Docker images for the Document Management Platform services.
# It handles the build process with proper tagging, versioning, and build arguments
# to create optimized container images for deployment.
#
# Usage: ./docker-build.sh [options]
#
# Options:
#   -h | --help: Show usage information
#   -s | --service <service_name>: Specify the service to build (api or worker).
#                                  If not specified, builds all services.
#   -r | --registry <docker_registry>: Specify the Docker registry to use.
#                                      If not specified, defaults to 'document-mgmt'.
#
# Environment Variables:
#   REGISTRY: Docker registry to use (default: document-mgmt)
#   VERSION:  Version tag for the image (default: latest)
#   DOCKERFILE: Path to the Dockerfile (default: Dockerfile)
#   CONTEXT: Build context path (default: .)

set -euo pipefail

# Global variables with default values
REGISTRY="${REGISTRY:-document-mgmt}"
VERSION="${VERSION:-latest}"
GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
BUILD_DATE="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
DOCKERFILE="${DOCKERFILE:-Dockerfile}"
CONTEXT="${CONTEXT:-.}"

# Check for required tools
if ! command -v docker &> /dev/null; then
  echo "Error: docker is required but not installed."
  exit 1
fi

if ! command -v git &> /dev/null; then
  echo "Warning: git is not installed. GIT_COMMIT will be 'unknown'."
fi

# Available services
declare -A available_services=(
  ["api"]="API Service"
  ["worker"]="Worker Service"
)

# Function to build a Docker image for a service
build_image() {
  local service_name="$1"

  # Check if the service is valid
  if [[ -z "${available_services[$service_name]}" ]]; then
    echo "Error: Invalid service name: $service_name"
    print_usage
    exit 1
  fi

  echo "Building image for ${available_services[$service_name]} ($service_name)..."

  # Construct the image name
  local image_name="${REGISTRY}/document-mgmt-${service_name}:${VERSION}"

  # Set up build arguments
  local build_args=(
    "--build-arg" "SERVICE=${service_name}"
    "--build-arg" "VERSION=${VERSION}"
    "--build-arg" "GIT_COMMIT=${GIT_COMMIT}"
    "--build-arg" "BUILD_DATE=${BUILD_DATE}"
  )

  # Execute docker build command
  docker build -f "${DOCKERFILE}" "${build_args[@]}" -t "${image_name}" "${CONTEXT}"

  # Check the exit code
  if [ $? -eq 0 ]; then
    echo "Successfully built image: $image_name"
  else
    echo "Error: Failed to build image: $image_name"
    return 1
  fi
}

# Function to print usage information
print_usage() {
  echo "Usage: $0 [options]"
  echo ""
  echo "Options:"
  echo "  -h | --help             Show usage information"
  echo "  -s | --service <service>  Specify the service to build (api or worker)"
  echo "  -r | --registry <registry> Specify the Docker registry to use"
  echo ""
  echo "Available services:"
  for service in "${!available_services[@]}"; do
    echo "  - $service: ${available_services[$service]}"
  done
  echo ""
  echo "Environment Variables:"
  echo "  REGISTRY=<registry>   Docker registry to use (default: document-mgmt)"
  echo "  VERSION=<version>     Version tag for the image (default: latest)"
  echo "  DOCKERFILE=<path>    Path to the Dockerfile (default: Dockerfile)"
  echo "  CONTEXT=<path>       Build context path (default: .)"
}

# Function to parse command-line arguments
parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -h | --help)
        print_usage
        exit 0
        ;;
      -s | --service)
        if [[ -n "$2" ]]; then
          SERVICE="$2"
          shift 2
        else
          echo "Error: --service requires a value"
          print_usage
          exit 1
        fi
        ;;
      -r | --registry)
        if [[ -n "$2" ]]; then
          REGISTRY="$2"
          shift 2
        else
          echo "Error: --registry requires a value"
          print_usage
          exit 1
        fi
        ;;
      *)
        echo "Error: Unknown parameter: $1"
        print_usage
        exit 1
        ;;
    esac
  done
}

# Main script execution
# Parse command-line arguments
parse_args "$@"

# Determine which services to build
if [[ -n "${SERVICE}" ]]; then
  services=(${SERVICE})
else
  services=("${!available_services[@]}")
fi

# Build images for each service
build_status=0
for service in "${services[@]}"; do
  if ! build_image "$service"; then
    build_status=1
  fi
done

# Exit with appropriate status code
exit $build_status