#!/bin/bash
set -e  # Exit immediately if a command exits with a non-zero status

# Default environment variables
CONFIG_FILE=${CONFIG_FILE:-config/development.yml}
AWS_ENDPOINT=${AWS_ENDPOINT:-http://localstack:4566}
AWS_REGION=${AWS_REGION:-us-east-1}
AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID:-test}
AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY:-test}

# Log messages with timestamp and level
log() {
  local level=$1
  local message=$2
  local timestamp=$(date +"%Y-%m-%d %H:%M:%S")
  
  # Format the message with timestamp and log level
  local formatted_message="[$timestamp] [$level] $message"
  
  # Print to stdout or stderr based on level
  if [ "$level" == "error" ]; then
    echo "$formatted_message" >&2
  else
    echo "$formatted_message"
  fi
}

# Wait for LocalStack to be ready
wait_for_localstack() {
  log "info" "Waiting for LocalStack to be ready..."
  local max_retries=30
  local retry_count=0
  
  while [ $retry_count -lt $max_retries ]; do
    if aws --endpoint-url=$AWS_ENDPOINT \
           --region=$AWS_REGION \
           --no-verify-ssl \
           --no-sign-request \
           s3 ls &> /dev/null; then
      log "info" "LocalStack is ready"
      return 0
    fi
    
    retry_count=$((retry_count + 1))
    log "info" "Waiting for LocalStack... ($retry_count/$max_retries)"
    sleep 2
  done
  
  log "error" "LocalStack failed to become ready after $max_retries attempts"
  return 1
}

# Wait for PostgreSQL to be ready
wait_for_postgres() {
  log "info" "Waiting for PostgreSQL to be ready..."
  
  # Extract database connection details from config file
  local db_host=$(yq e '.database.host' $CONFIG_FILE)
  local db_port=$(yq e '.database.port' $CONFIG_FILE)
  local db_user=$(yq e '.database.user' $CONFIG_FILE)
  local db_name=$(yq e '.database.dbname' $CONFIG_FILE)
  
  local max_retries=30
  local retry_count=0
  
  while [ $retry_count -lt $max_retries ]; do
    if pg_isready -h $db_host -p $db_port -U $db_user &> /dev/null; then
      log "info" "PostgreSQL is ready"
      return 0
    fi
    
    retry_count=$((retry_count + 1))
    log "info" "Waiting for PostgreSQL... ($retry_count/$max_retries)"
    sleep 2
  done
  
  log "error" "PostgreSQL failed to become ready after $max_retries attempts"
  return 1
}

# Setup S3 buckets
setup_s3_buckets() {
  log "info" "Setting up S3 buckets..."
  
  # Extract bucket names from config file
  local bucket=$(yq e '.s3.bucket' $CONFIG_FILE)
  local temp_bucket=$(yq e '.s3.temp_bucket' $CONFIG_FILE)
  local quarantine_bucket=$(yq e '.s3.quarantine_bucket' $CONFIG_FILE)
  
  # Create buckets if they don't exist
  aws --endpoint-url=$AWS_ENDPOINT \
      --region=$AWS_REGION \
      s3 mb s3://$bucket --no-verify-ssl 2>/dev/null || true
  
  aws --endpoint-url=$AWS_ENDPOINT \
      --region=$AWS_REGION \
      s3 mb s3://$temp_bucket --no-verify-ssl 2>/dev/null || true
  
  aws --endpoint-url=$AWS_ENDPOINT \
      --region=$AWS_REGION \
      s3 mb s3://$quarantine_bucket --no-verify-ssl 2>/dev/null || true
  
  log "info" "S3 buckets created successfully"
  return 0
}

# Setup SQS queues
setup_sqs_queues() {
  log "info" "Setting up SQS queues..."
  
  # Extract queue URLs from config file
  local document_queue_url=$(yq e '.sqs.document_queue_url' $CONFIG_FILE)
  local scan_queue_url=$(yq e '.sqs.scan_queue_url' $CONFIG_FILE)
  local index_queue_url=$(yq e '.sqs.index_queue_url' $CONFIG_FILE)
  
  # Extract queue names from URLs
  local document_queue_name=$(basename $document_queue_url)
  local scan_queue_name=$(basename $scan_queue_url)
  local index_queue_name=$(basename $index_queue_url)
  
  # Create queues
  aws --endpoint-url=$AWS_ENDPOINT \
      --region=$AWS_REGION \
      sqs create-queue --queue-name $document_queue_name --no-verify-ssl 2>/dev/null || true
  
  aws --endpoint-url=$AWS_ENDPOINT \
      --region=$AWS_REGION \
      sqs create-queue --queue-name $scan_queue_name --no-verify-ssl 2>/dev/null || true
  
  aws --endpoint-url=$AWS_ENDPOINT \
      --region=$AWS_REGION \
      sqs create-queue --queue-name $index_queue_name --no-verify-ssl 2>/dev/null || true
  
  log "info" "SQS queues created successfully"
  return 0
}

# Setup SNS topics
setup_sns_topics() {
  log "info" "Setting up SNS topics..."
  
  # Extract topic ARNs from config file
  local document_topic_arn=$(yq e '.sns.document_topic_arn' $CONFIG_FILE)
  local event_topic_arn=$(yq e '.sns.event_topic_arn' $CONFIG_FILE)
  
  # Extract topic names from ARNs
  local document_topic_name=$(basename $document_topic_arn)
  local event_topic_name=$(basename $event_topic_arn)
  
  # Create topics
  aws --endpoint-url=$AWS_ENDPOINT \
      --region=$AWS_REGION \
      sns create-topic --name $document_topic_name --no-verify-ssl 2>/dev/null || true
  
  aws --endpoint-url=$AWS_ENDPOINT \
      --region=$AWS_REGION \
      sns create-topic --name $event_topic_name --no-verify-ssl 2>/dev/null || true
  
  log "info" "SNS topics created successfully"
  return 0
}

# Setup database schema and initial data
setup_database() {
  log "info" "Setting up database schema and initial data..."
  
  # Run database migrations
  log "info" "Running database migrations..."
  bash ./scripts/migration.sh -c $CONFIG_FILE up
  if [ $? -ne 0 ]; then
    log "error" "Failed to run database migrations"
    return 1
  fi
  
  # Create default tenant, user, and root folder
  local tenant_id=$(create_default_tenant)
  local user_id=$(create_default_user "$tenant_id")
  create_root_folder "$tenant_id" "$user_id"
  
  log "info" "Database setup completed successfully"
  return 0
}

# Create default tenant for development
create_default_tenant() {
  log "info" "Creating default tenant..."
  
  # Extract database connection details from config file
  local db_host=$(yq e '.database.host' $CONFIG_FILE)
  local db_port=$(yq e '.database.port' $CONFIG_FILE)
  local db_user=$(yq e '.database.user' $CONFIG_FILE)
  local db_password=$(yq e '.database.password' $CONFIG_FILE)
  local db_name=$(yq e '.database.dbname' $CONFIG_FILE)
  
  # Check if default tenant exists
  local tenant_id=$(PGPASSWORD=$db_password psql -h $db_host -p $db_port -U $db_user -d $db_name -t -c "SELECT id FROM tenants WHERE name = 'Development Tenant'")
  
  if [ -z "$tenant_id" ]; then
    # Generate a UUID for the tenant
    local new_tenant_id=$(uuidgen | tr '[:upper:]' '[:lower:]')
    
    # Insert default tenant
    PGPASSWORD=$db_password psql -h $db_host -p $db_port -U $db_user -d $db_name -c "
      INSERT INTO tenants (id, name, status, created_at, updated_at)
      VALUES ('$new_tenant_id', 'Development Tenant', 'active', NOW(), NOW());
    "
    
    log "info" "Created default tenant with ID: $new_tenant_id"
    echo $new_tenant_id
  else
    tenant_id=$(echo $tenant_id | xargs)  # Trim whitespace
    log "info" "Default tenant already exists with ID: $tenant_id"
    echo $tenant_id
  fi
}

# Create default admin user for development
create_default_user() {
  local tenant_id=$1
  log "info" "Creating default admin user for tenant: $tenant_id..."
  
  # Extract database connection details from config file
  local db_host=$(yq e '.database.host' $CONFIG_FILE)
  local db_port=$(yq e '.database.port' $CONFIG_FILE)
  local db_user=$(yq e '.database.user' $CONFIG_FILE)
  local db_password=$(yq e '.database.password' $CONFIG_FILE)
  local db_name=$(yq e '.database.dbname' $CONFIG_FILE)
  
  # Check if default admin user exists
  local user_id=$(PGPASSWORD=$db_password psql -h $db_host -p $db_port -U $db_user -d $db_name -t -c "SELECT id FROM users WHERE tenant_id = '$tenant_id' AND username = 'admin'")
  
  if [ -z "$user_id" ]; then
    # Generate a UUID for the user
    local new_user_id=$(uuidgen | tr '[:upper:]' '[:lower:]')
    
    # Insert default admin user (password is 'password')
    PGPASSWORD=$db_password psql -h $db_host -p $db_port -U $db_user -d $db_name -c "
      INSERT INTO users (id, tenant_id, username, email, password_hash, status, created_at, updated_at)
      VALUES (
        '$new_user_id', 
        '$tenant_id', 
        'admin', 
        'admin@example.com', 
        '\$2a\$10\$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',  -- hash for 'password'
        'active', 
        NOW(), 
        NOW()
      );
    "
    
    # Assign administrator role to the user
    local admin_role_id=$(PGPASSWORD=$db_password psql -h $db_host -p $db_port -U $db_user -d $db_name -t -c "SELECT id FROM roles WHERE name = 'administrator' AND tenant_id = '$tenant_id'")
    
    if [ -z "$admin_role_id" ]; then
      # Create administrator role if it doesn't exist
      local new_role_id=$(uuidgen | tr '[:upper:]' '[:lower:]')
      PGPASSWORD=$db_password psql -h $db_host -p $db_port -U $db_user -d $db_name -c "
        INSERT INTO roles (id, tenant_id, name, description, created_at, updated_at)
        VALUES ('$new_role_id', '$tenant_id', 'administrator', 'Full administrative access', NOW(), NOW());
      "
      admin_role_id=$new_role_id
    else
      admin_role_id=$(echo $admin_role_id | xargs)  # Trim whitespace
    fi
    
    # Assign role to user
    PGPASSWORD=$db_password psql -h $db_host -p $db_port -U $db_user -d $db_name -c "
      INSERT INTO user_roles (user_id, role_id, created_at)
      VALUES ('$new_user_id', '$admin_role_id', NOW());
    "
    
    log "info" "Created default admin user with ID: $new_user_id"
    echo $new_user_id
  else
    user_id=$(echo $user_id | xargs)  # Trim whitespace
    log "info" "Default admin user already exists with ID: $user_id"
    echo $user_id
  fi
}

# Create root folder for the tenant
create_root_folder() {
  local tenant_id=$1
  local user_id=$2
  log "info" "Creating root folder for tenant: $tenant_id..."
  
  # Extract database connection details from config file
  local db_host=$(yq e '.database.host' $CONFIG_FILE)
  local db_port=$(yq e '.database.port' $CONFIG_FILE)
  local db_user=$(yq e '.database.user' $CONFIG_FILE)
  local db_password=$(yq e '.database.password' $CONFIG_FILE)
  local db_name=$(yq e '.database.dbname' $CONFIG_FILE)
  
  # Check if root folder exists
  local folder_id=$(PGPASSWORD=$db_password psql -h $db_host -p $db_port -U $db_user -d $db_name -t -c "SELECT id FROM folders WHERE tenant_id = '$tenant_id' AND parent_id IS NULL")
  
  if [ -z "$folder_id" ]; then
    # Generate a UUID for the folder
    local new_folder_id=$(uuidgen | tr '[:upper:]' '[:lower:]')
    
    # Insert root folder
    PGPASSWORD=$db_password psql -h $db_host -p $db_port -U $db_user -d $db_name -c "
      INSERT INTO folders (id, tenant_id, parent_id, name, path, owner_id, created_at, updated_at)
      VALUES ('$new_folder_id', '$tenant_id', NULL, 'Root', '/', '$user_id', NOW(), NOW());
    "
    
    log "info" "Created root folder with ID: $new_folder_id"
    echo $new_folder_id
  else
    folder_id=$(echo $folder_id | xargs)  # Trim whitespace
    log "info" "Root folder already exists with ID: $folder_id"
    echo $folder_id
  fi
}

# Main script execution
log "info" "Starting development environment setup..."

# Wait for required services
wait_for_localstack
if [ $? -ne 0 ]; then
  log "error" "Failed to connect to LocalStack"
  exit 1
fi

wait_for_postgres
if [ $? -ne 0 ]; then
  log "error" "Failed to connect to PostgreSQL"
  exit 1
fi

# Setup infrastructure
setup_s3_buckets
setup_sqs_queues
setup_sns_topics
setup_database

log "info" "Development environment setup completed successfully"
exit 0