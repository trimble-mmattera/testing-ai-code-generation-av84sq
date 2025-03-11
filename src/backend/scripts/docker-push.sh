#!/bin/bash
#
# This script pushes Docker images to a container registry for the Document Management Platform.
# It handles authentication with the registry, tagging images appropriately, and pushing them
# to the target repository with proper versioning.
#
# Usage: ./docker-push.sh [options]
#
# Options:
#   -h | --help: Show usage information
#   -s | --service <service_name>: Specify the service to push (api or worker).
#                                  If not specified, pushes all services.
#   -r | --registry <docker_registry>: Specify the Docker registry to use.
#                                      If not specified, defaults to 'document-mgmt'.
#
# Environment Variables:
#   REGISTRY: Docker registry to use (default: document-mgmt)
#   VERSION:  Version tag for the image (default: latest)
#   AWS_REGION: AWS region for ECR authentication (default: us-west-2)
#   AWS_ACCOUNT_ID: AWS account ID for ECR authentication (required for ECR)
#   ECR_REPOSITORY: ECR repository name (default: document-mgmt)

set -euo pipefail

# Source the docker-build.sh script to reuse variables and naming conventions
# This ensures consistency between the build and push processes
source "$(dirname "$0")/docker-build.sh"

# Global variables with default values
REGISTRY="${REGISTRY:-document-mgmt}"
VERSION="${VERSION:-latest}"
AWS_REGION="${AWS_REGION:-us-west-2}"
AWS_ACCOUNT_ID="${AWS_ACCOUNT_ID:-}"
ECR_REPOSITORY="${ECR_REPOSITORY:-document-mgmt}"

# Check for required tools
if ! command -v docker &> /dev/null; then
  echo "Error: docker is required but not installed."
  exit 1
fi

if [[ "$REGISTRY" == *".amazonaws.com"* ]] && ! command -v aws &> /dev/null; then
  echo "Error: aws-cli is required for pushing to AWS ECR but not installed."
  exit 1
fi

# Available services
declare -A available_services=(
  ["api"]="API Service"
  ["worker"]="Worker Service"
)

# Function to push a Docker image to the specified registry
push_image() {
  local service_name="$1"

  # Check if the service is valid
  if [[ -z "${available_services[$service_name]}" ]]; then
    echo "Error: Invalid service name: $service_name"
    print_usage
    exit 1
  fi

  echo "Pushing image for ${available_services[$service_name]} ($service_name)..."

  # Construct the image name with registry and service
  local image_name="${REGISTRY}/document-mgmt-${service_name}:${VERSION}"

  # Check if pushing to ECR and authenticate if needed
  if [[ "$REGISTRY" == *".amazonaws.com"* ]]; then
    ecr_login
    if [ $? -ne 0 ]; then
      echo "Error: Failed to authenticate with AWS ECR"
      return 1
    fi
  fi

  # Execute docker push command with appropriate tags
  docker push "$image_name"

  # Check the exit code
  if [ $? -eq 0 ]; then
    echo "Successfully pushed image: $image_name"
  else
    echo "Error: Failed to push image: $image_name"
    return 1
  fi
}

# Function to authenticate with AWS ECR registry
ecr_login() {
  # Check if AWS CLI is installed
  if ! command -v aws &> /dev/null; then
    echo "Error: aws-cli is required but not installed."
    return 1
  fi

  # Verify AWS_REGION and AWS_ACCOUNT_ID are set
  if [[ -z "${AWS_REGION}" ]]; then
    echo "Error: AWS_REGION environment variable must be set for ECR authentication."
    return 1
  fi
  if [[ -z "${AWS_ACCOUNT_ID}" ]]; then
    echo "Error: AWS_ACCOUNT_ID environment variable must be set for ECR authentication."
    return 1
  fi

  # Execute AWS ECR get-login-password command
  aws ecr get-login-password --region "$AWS_REGION" | docker login --username AWS --password-stdin "$AWS_ACCOUNT_ID".dkr.ecr."$AWS_REGION".amazonaws.com

  # Check the exit code
  if [ $? -eq 0 ]; then
    echo "Successfully authenticated with AWS ECR"
  else
    echo "Error: Failed to authenticate with AWS ECR"
    return 1
  fi
}

# Function to print usage information
print_usage() {
  echo "Usage: $0 [options]"
  echo ""
  echo "Options:"
  echo "  -h | --help             Show usage information"
  echo "  -s | --service <service>  Specify the service to push (api or worker)"
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
  echo "  AWS_REGION=<region>   AWS region for ECR authentication (default: us-west-2)"
  echo "  AWS_ACCOUNT_ID=<account_id> AWS account ID for ECR authentication (required for ECR)"
  echo "  ECR_REPOSITORY=<repository> ECR repository name (default: document-mgmt)"
  echo ""
  echo "Examples:"
  echo "  Push all services to Docker Hub:"
  echo "  ./docker-push.sh"
  echo ""
  echo "  Push only the API service to a custom registry:"
  echo "  ./docker-push.sh -s api -r my-docker-registry.com"
  echo ""
  echo "  Push all services to AWS ECR:"
  echo "  export AWS_REGION=us-west-2"
  echo "  export AWS_ACCOUNT_ID=123456789012"
  echo "  ./docker-push.sh -r 123456789012.dkr.ecr.us-west-2.amazonaws.com/document-mgmt"
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

# Determine which services to push
if [[ -n "${SERVICE}" ]]; then
  services=(${SERVICE})
else
  services=("${!available_services[@]}")
fi

# Push images for each service
push_status=0
for service in "${services[@]}"; do
  if ! push_image "$service"; then
    push_status=1
  fi
done

# Exit with appropriate status code
exit $push_status