#!/bin/bash
set -e  # Exit immediately if a command exits with a non-zero status

# Constants
MIGRATIONS_DIR="src/backend/infrastructure/persistence/postgres/migrations"
CONFIG_FILE="${CONFIG_FILE:-config/default.yml}"
ENV="${ENV:-development}"
VERBOSE="${VERBOSE:-false}"

# Print usage information
print_usage() {
    echo "Database Migration Script for Document Management Platform"
    echo ""
    echo "Usage: $(basename "$0") [options] COMMAND [args]"
    echo ""
    echo "Commands:"
    echo "  up [steps]        Apply all or a specific number of pending migrations"
    echo "  down <steps>      Rollback a specific number of migrations"
    echo "  create <name>     Create a new migration with the given name"
    echo ""
    echo "Options:"
    echo "  -h, --help        Show this help message"
    echo "  -c, --config      Specify the config file (default: $CONFIG_FILE)"
    echo "  -e, --env         Specify the environment (default: $ENV)"
    echo "  -v, --verbose     Enable verbose output"
    echo ""
    echo "Environment Variables:"
    echo "  CONFIG_FILE       Config file path"
    echo "  ENV               Environment (development, staging, production)"
    echo "  VERBOSE           Enable verbose output (true/false)"
    echo ""
    echo "Examples:"
    echo "  $(basename "$0") up                 # Apply all pending migrations"
    echo "  $(basename "$0") up 1               # Apply the next pending migration"
    echo "  $(basename "$0") down 1             # Rollback the most recent migration"
    echo "  $(basename "$0") create add_users   # Create migration files named add_users"
}

# Parse arguments
parse_args() {
    local TEMP
    TEMP=$(getopt -o "hc:e:v" --long "help,config:,env:,verbose" -n "$(basename "$0")" -- "$@")
    
    if [ $? -ne 0 ]; then
        echo "Error: Invalid arguments" >&2
        print_usage
        exit 1
    fi
    
    eval set -- "$TEMP"
    
    while true; do
        case "$1" in
            -h|--help)
                print_usage
                exit 0
                ;;
            -c|--config)
                CONFIG_FILE="$2"
                shift 2
                ;;
            -e|--env)
                ENV="$2"
                shift 2
                ;;
            -v|--verbose)
                VERBOSE="true"
                shift
                ;;
            --)
                shift
                break
                ;;
            *)
                echo "Error: Internal error!" >&2
                exit 1
                ;;
        esac
    done
    
    # Return the remaining arguments (command and command args)
    echo "$@"
}

# Load database configuration from config file
load_config() {
    if [ ! -f "$CONFIG_FILE" ]; then
        log "error" "Config file not found: $CONFIG_FILE"
        exit 1
    fi
    
    # Extract database configuration using a simple grep and awk approach
    # In a production environment, consider using a proper YAML parser
    local host=$(grep -A 10 "Database:" "$CONFIG_FILE" | grep "Host:" | awk '{print $2}')
    local port=$(grep -A 10 "Database:" "$CONFIG_FILE" | grep "Port:" | awk '{print $2}')
    local user=$(grep -A 10 "Database:" "$CONFIG_FILE" | grep "User:" | awk '{print $2}')
    local password=$(grep -A 10 "Database:" "$CONFIG_FILE" | grep "Password:" | awk '{print $2}')
    local dbname=$(grep -A 10 "Database:" "$CONFIG_FILE" | grep "DBName:" | awk '{print $2}')
    local sslmode=$(grep -A 10 "Database:" "$CONFIG_FILE" | grep "SSLMode:" | awk '{print $2}')
    
    # Set default values if not found
    host=${host:-localhost}
    port=${port:-5432}
    user=${user:-postgres}
    password=${password:-postgres}
    dbname=${dbname:-documentdb}
    sslmode=${sslmode:-disable}
    
    # Return database configuration as a space-separated string
    echo "$host $port $user $password $dbname $sslmode"
}

# Build PostgreSQL connection string
build_dsn() {
    local config=($1)
    local host=${config[0]}
    local port=${config[1]}
    local user=${config[2]}
    local password=${config[3]}
    local dbname=${config[4]}
    local sslmode=${config[5]}
    
    # Format the connection string
    echo "postgres://$user:$password@$host:$port/$dbname?sslmode=$sslmode"
}

# Check if migrate is installed
check_migrate() {
    if ! command -v migrate &> /dev/null; then
        log "error" "migrate command not found. Please install golang-migrate:"
        log "error" "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
        return 1
    fi
    return 0
}

# Apply pending migrations
migrate_up() {
    local steps=$1
    
    # Check if migrate is installed
    check_migrate || return 1
    
    log "info" "Loading database configuration..."
    local db_config=$(load_config)
    local dsn=$(build_dsn "$db_config")
    
    log "info" "Applying migrations..."
    
    if [ -n "$steps" ]; then
        log "info" "Applying $steps migration(s)..."
        migrate -path "$MIGRATIONS_DIR" -database "$dsn" up "$steps"
    else
        log "info" "Applying all pending migrations..."
        migrate -path "$MIGRATIONS_DIR" -database "$dsn" up
    fi
    
    local status=$?
    if [ $status -eq 0 ]; then
        log "info" "Migrations applied successfully"
        return 0
    elif [ $status -eq 1 ]; then
        # No change (which migrate treats as an error, but we don't)
        log "info" "No migrations to apply"
        return 0
    else
        log "error" "Failed to apply migrations (status: $status)"
        return 1
    fi
}

# Rollback applied migrations
migrate_down() {
    local steps=$1
    
    if [ -z "$steps" ]; then
        log "error" "Number of migrations to rollback is required"
        return 1
    fi
    
    # Check if migrate is installed
    check_migrate || return 1
    
    log "info" "Loading database configuration..."
    local db_config=$(load_config)
    local dsn=$(build_dsn "$db_config")
    
    log "info" "Rolling back $steps migration(s)..."
    migrate -path "$MIGRATIONS_DIR" -database "$dsn" down "$steps"
    
    local status=$?
    if [ $status -eq 0 ]; then
        log "info" "Migrations rolled back successfully"
        return 0
    elif [ $status -eq 1 ]; then
        # No change (which migrate treats as an error, but we don't)
        log "info" "No migrations to roll back"
        return 0
    else
        log "error" "Failed to roll back migrations (status: $status)"
        return 1
    fi
}

# Create new migration files
create_migration() {
    local name=$1
    
    if [ -z "$name" ]; then
        log "error" "Migration name is required"
        return 1
    fi
    
    # Validate migration name (alphanumeric with underscores)
    if ! [[ $name =~ ^[a-zA-Z0-9_]+$ ]]; then
        log "error" "Migration name must contain only alphanumeric characters and underscores"
        return 1
    fi
    
    # Check if migrations directory exists
    check_migrations_dir
    
    # Generate timestamp for version
    local timestamp=$(date +%Y%m%d%H%M%S)
    local up_file="${MIGRATIONS_DIR}/${timestamp}_${name}.up.sql"
    local down_file="${MIGRATIONS_DIR}/${timestamp}_${name}.down.sql"
    
    # Create empty migration files
    touch "$up_file"
    touch "$down_file"
    
    log "info" "Created migration files:"
    log "info" "  $up_file"
    log "info" "  $down_file"
    
    return 0
}

# Ensure migrations directory exists
check_migrations_dir() {
    if [ ! -d "$MIGRATIONS_DIR" ]; then
        log "info" "Creating migrations directory: $MIGRATIONS_DIR"
        mkdir -p "$MIGRATIONS_DIR"
    fi
    
    return 0
}

# Log messages with timestamp
log() {
    local level=$1
    local message=$2
    local timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    
    # Only print debug messages if verbose mode is enabled
    if [ "$level" == "debug" ] && [ "$VERBOSE" != "true" ]; then
        return
    fi
    
    # Format the log message
    local formatted_message="[$timestamp] [$level] $message"
    
    # Print to stdout or stderr based on level
    if [ "$level" == "error" ]; then
        echo "$formatted_message" >&2
    else
        echo "$formatted_message"
    fi
}

# Main script execution
main() {
    # Parse arguments
    args=($(parse_args "$@"))
    
    # Check if command is provided
    if [ ${#args[@]} -eq 0 ]; then
        log "error" "No command specified"
        print_usage
        exit 1
    fi
    
    # Extract command and command arguments
    cmd=${args[0]}
    cmd_args=("${args[@]:1}")
    
    # Execute the appropriate command
    case "$cmd" in
        up)
            migrate_up "${cmd_args[0]}"
            exit $?
            ;;
        down)
            if [ ${#cmd_args[@]} -eq 0 ]; then
                log "error" "Number of migrations to rollback is required"
                print_usage
                exit 1
            fi
            migrate_down "${cmd_args[0]}"
            exit $?
            ;;
        create)
            if [ ${#cmd_args[@]} -eq 0 ]; then
                log "error" "Migration name is required"
                print_usage
                exit 1
            fi
            create_migration "${cmd_args[0]}"
            exit $?
            ;;
        *)
            log "error" "Unknown command: $cmd"
            print_usage
            exit 1
            ;;
    esac
}

# Run the main function with all args
main "$@"