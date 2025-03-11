#!/bin/bash

# generate-mock.sh
# Script to generate mock implementations for interfaces used in the Document Management Platform
# Version: 1.0
# Requirements: mockery v2.20.0+ (github.com/vektra/mockery/v2)

set -e

# Constants
SOURCE_DIR=$(pwd)
OUTPUT_DIR="${SOURCE_DIR}/test/mocks"
MOCKERY_CMD="mockery"

# Function to display usage information
usage() {
    echo "Usage: $0 [options]"
    echo "Generate mock implementations for interfaces in the Document Management Platform"
    echo ""
    echo "Options:"
    echo "  -h, --help           Show this help message"
    echo "  -v, --verbose        Enable verbose output"
    echo "  -o, --output DIR     Set custom output directory (default: ./test/mocks)"
    echo "  -s, --source DIR     Set source directory (default: current directory)"
    echo "  -p, --package PKG    Generate mocks for a specific package"
    echo "  -i, --interface NAME Generate mock for a specific interface"
    echo ""
    echo "Examples:"
    echo "  $0                   Generate all mocks"
    echo "  $0 -p domain/repository  Generate mocks for repository interfaces only"
    echo "  $0 -p domain/service -i DocumentService  Generate mock for DocumentService interface only"
    exit 1
}

# Function to check if mockery is installed
check_mockery() {
    echo "Checking if mockery is installed..."
    
    if ! command -v $MOCKERY_CMD &> /dev/null; then
        echo "mockery not found. Attempting to install..."
        go install github.com/vektra/mockery/v2@latest
        
        # Check if installation was successful
        if ! command -v $MOCKERY_CMD &> /dev/null; then
            echo "Failed to install mockery. Please install it manually: go install github.com/vektra/mockery/v2@latest"
            return 1
        fi
        
        echo "mockery installed successfully."
    else
        echo "mockery already installed."
    fi
    
    return 0
}

# Function to generate mocks for all interfaces in a package
generate_mocks_for_package() {
    local package_path=$1
    echo "Generating mocks for package: $package_path"
    
    $MOCKERY_CMD --dir="$SOURCE_DIR" --output="$OUTPUT_DIR" --all --case=underscore --with-expecter --keeptree --packageprefix=mock --outpkg=mocks --name=".*" --recursive=false --testonly=false --packages="$package_path"
    
    if [ $? -ne 0 ]; then
        echo "Failed to generate mocks for package: $package_path"
        return 1
    fi
    
    echo "Successfully generated mocks for package: $package_path"
    return 0
}

# Function to generate a mock for a specific interface
generate_mock_for_interface() {
    local package_path=$1
    local interface_name=$2
    
    echo "Generating mock for interface: $interface_name in package: $package_path"
    
    $MOCKERY_CMD --dir="$SOURCE_DIR" --output="$OUTPUT_DIR" --case=underscore --with-expecter --keeptree --packageprefix=mock --outpkg=mocks --name="$interface_name" --recursive=false --testonly=false --packages="$package_path"
    
    if [ $? -ne 0 ]; then
        echo "Failed to generate mock for interface: $interface_name in package: $package_path"
        return 1
    fi
    
    echo "Successfully generated mock for interface: $interface_name in package: $package_path"
    return 0
}

# Main function
main() {
    local specific_package=""
    local specific_interface=""
    local verbose=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                ;;
            -v|--verbose)
                verbose=true
                shift
                ;;
            -o|--output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            -s|--source)
                SOURCE_DIR="$2"
                shift 2
                ;;
            -p|--package)
                specific_package="$2"
                shift 2
                ;;
            -i|--interface)
                specific_interface="$2"
                shift 2
                ;;
            *)
                echo "Unknown option: $1"
                usage
                ;;
        esac
    done
    
    echo "===================================================="
    echo "  Generating mocks for Document Management Platform"
    echo "===================================================="
    echo "  Source directory: $SOURCE_DIR"
    echo "  Output directory: $OUTPUT_DIR"
    if [ -n "$specific_package" ]; then
        echo "  Specific package: $specific_package"
    fi
    if [ -n "$specific_interface" ]; then
        echo "  Specific interface: $specific_interface"
    fi
    echo "===================================================="
    
    # Check if mockery is installed
    check_mockery
    if [ $? -ne 0 ]; then
        echo "Aborting: mockery is required to generate mocks."
        return 1
    fi
    
    # Create output directory if it doesn't exist
    mkdir -p "$OUTPUT_DIR"
    
    # Define package paths for the Document Management Platform
    MODULE_PREFIX="github.com/document-mgmt"
    DOMAIN_PKG="${MODULE_PREFIX}/domain"
    DOMAIN_REPO_PKG="${DOMAIN_PKG}/repository"
    DOMAIN_SVC_PKG="${DOMAIN_PKG}/service"
    USECASE_PKG="${MODULE_PREFIX}/usecase"
    INFRA_PKG="${MODULE_PREFIX}/infrastructure"
    DELIVERY_PKG="${MODULE_PREFIX}/delivery"
    
    # Generate mocks based on command line arguments
    if [ -n "$specific_package" ] && [ -n "$specific_interface" ]; then
        # Generate mock for specific interface in specific package
        generate_mock_for_interface "${MODULE_PREFIX}/${specific_package}" "$specific_interface"
        return $?
    elif [ -n "$specific_package" ]; then
        # Generate mocks for all interfaces in specific package
        generate_mocks_for_package "${MODULE_PREFIX}/${specific_package}"
        return $?
    fi
    
    # Generate mocks for domain repositories
    echo "Generating mocks for domain repositories..."
    generate_mocks_for_package $DOMAIN_REPO_PKG
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    # Generate mocks for domain services
    echo "Generating mocks for domain services..."
    generate_mocks_for_package $DOMAIN_SVC_PKG
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    # Generate mocks for use cases
    echo "Generating mocks for use cases..."
    generate_mocks_for_package $USECASE_PKG
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    # Generate mocks for infrastructure interfaces
    echo "Generating mocks for infrastructure interfaces..."
    generate_mocks_for_package $INFRA_PKG
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    # Generate mocks for key interfaces in the Document Management Platform
    echo "Generating mocks for key interfaces..."
    
    # Document domain
    generate_mock_for_interface $DOMAIN_REPO_PKG "DocumentRepository"
    generate_mock_for_interface $DOMAIN_SVC_PKG "DocumentService"
    
    # Storage domain
    generate_mock_for_interface $DOMAIN_SVC_PKG "StorageService"
    
    # Search domain
    generate_mock_for_interface $DOMAIN_SVC_PKG "SearchService"
    
    # Folder domain
    generate_mock_for_interface $DOMAIN_REPO_PKG "FolderRepository"
    generate_mock_for_interface $DOMAIN_SVC_PKG "FolderService"
    
    # Security domain
    generate_mock_for_interface $DOMAIN_SVC_PKG "SecurityService"
    generate_mock_for_interface $DOMAIN_SVC_PKG "VirusScanningService"
    generate_mock_for_interface $DOMAIN_SVC_PKG "AuthenticationService"
    
    # Event domain
    generate_mock_for_interface $DOMAIN_SVC_PKG "EventService"
    
    echo "===================================================="
    echo "  All mocks generated successfully!"
    echo "  Mocks location: $OUTPUT_DIR"
    echo "===================================================="
    
    return 0
}

# Execute the main function
main "$@"
exit $?