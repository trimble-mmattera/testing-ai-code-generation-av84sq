-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tenants table - Stores customer organizations (tenants) using the platform
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX tenants_name_idx ON tenants(name);

-- Tenant settings table - Stores tenant-specific settings as key-value pairs
CREATE TABLE tenant_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX tenant_settings_tenant_key_idx ON tenant_settings(tenant_id, key);

-- Users table - Stores user accounts with authentication information
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    username VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX users_tenant_id_idx ON users(tenant_id);
CREATE INDEX users_email_idx ON users(email);
CREATE UNIQUE INDEX users_tenant_username_idx ON users(tenant_id, username);
CREATE UNIQUE INDEX users_tenant_email_idx ON users(tenant_id, email);

-- User settings table - Stores user-specific settings as key-value pairs
CREATE TABLE user_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX user_settings_user_key_idx ON user_settings(user_id, key);

-- Roles table - Stores role definitions for role-based access control
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX roles_tenant_id_idx ON roles(tenant_id);
CREATE UNIQUE INDEX roles_tenant_name_idx ON roles(tenant_id, name);

-- User roles table - Maps users to roles for role-based access control
CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT user_roles_pkey PRIMARY KEY (user_id, role_id)
);
CREATE INDEX user_roles_role_id_idx ON user_roles(role_id);

-- Folders table - Stores folder hierarchy for document organization
CREATE TABLE folders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    path TEXT NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX folders_tenant_id_idx ON folders(tenant_id);
CREATE INDEX folders_parent_id_idx ON folders(parent_id);
CREATE INDEX folders_path_idx ON folders(path);
CREATE UNIQUE INDEX folders_tenant_parent_name_idx ON folders(tenant_id, parent_id, name);

-- Documents table - Stores document metadata for uploaded files
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    folder_id UUID NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size BIGINT NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id),
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX documents_tenant_id_idx ON documents(tenant_id);
CREATE INDEX documents_folder_id_idx ON documents(folder_id);
CREATE INDEX documents_owner_id_idx ON documents(owner_id);
CREATE INDEX documents_status_idx ON documents(status);
CREATE INDEX documents_content_type_idx ON documents(content_type);
CREATE UNIQUE INDEX documents_tenant_folder_name_idx ON documents(tenant_id, folder_id, name);

-- Document versions table - Stores version information for documents
CREATE TABLE document_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    size BIGINT NOT NULL,
    content_hash VARCHAR(64) NOT NULL,
    status VARCHAR(50) NOT NULL,
    storage_path TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(id)
);
CREATE INDEX document_versions_document_id_idx ON document_versions(document_id);
CREATE INDEX document_versions_status_idx ON document_versions(status);
CREATE UNIQUE INDEX document_versions_document_version_idx ON document_versions(document_id, version_number);

-- Document metadata table - Stores custom metadata for documents as key-value pairs
CREATE TABLE document_metadata (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX document_metadata_document_id_idx ON document_metadata(document_id);
CREATE INDEX document_metadata_key_value_idx ON document_metadata(document_id, key, value);
CREATE UNIQUE INDEX document_metadata_document_key_idx ON document_metadata(document_id, key);

-- Tags table - Stores tags for document categorization
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX tags_tenant_id_idx ON tags(tenant_id);
CREATE UNIQUE INDEX tags_tenant_name_idx ON tags(tenant_id, name);

-- Document tags table - Maps documents to tags (many-to-many relationship)
CREATE TABLE document_tags (
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES users(id),
    CONSTRAINT document_tags_pkey PRIMARY KEY (document_id, tag_id)
);
CREATE INDEX document_tags_tag_id_idx ON document_tags(tag_id);

-- Permissions table - Stores access control permissions for resources
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID NOT NULL,
    permission_type VARCHAR(50) NOT NULL,
    inherited BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX permissions_tenant_id_idx ON permissions(tenant_id);
CREATE INDEX permissions_role_id_idx ON permissions(role_id);
CREATE INDEX permissions_resource_idx ON permissions(resource_type, resource_id);
CREATE INDEX permissions_tenant_resource_idx ON permissions(tenant_id, resource_type, resource_id);
CREATE UNIQUE INDEX permissions_unique_idx ON permissions(tenant_id, role_id, resource_type, resource_id, permission_type);

-- Insert default system roles
INSERT INTO roles (id, tenant_id, name, description, created_at, updated_at)
SELECT uuid_generate_v4(), id, 'reader', 'Can view documents and folders', NOW(), NOW() FROM tenants
UNION ALL
SELECT uuid_generate_v4(), id, 'contributor', 'Can view, upload, and update documents', NOW(), NOW() FROM tenants
UNION ALL
SELECT uuid_generate_v4(), id, 'editor', 'Can view, upload, update, and delete documents', NOW(), NOW() FROM tenants
UNION ALL
SELECT uuid_generate_v4(), id, 'administrator', 'Can perform all operations including folder management', NOW(), NOW() FROM tenants
UNION ALL
SELECT uuid_generate_v4(), id, 'system', 'Special role for system operations', NOW(), NOW() FROM tenants;

-- Add table comments for documentation
COMMENT ON TABLE tenants IS 'Stores customer organizations using the platform';
COMMENT ON TABLE tenant_settings IS 'Stores tenant-specific settings as key-value pairs';
COMMENT ON TABLE users IS 'Stores user accounts with authentication information';
COMMENT ON TABLE user_settings IS 'Stores user-specific settings as key-value pairs';
COMMENT ON TABLE roles IS 'Stores role definitions for role-based access control';
COMMENT ON TABLE user_roles IS 'Maps users to roles for role-based access control';
COMMENT ON TABLE folders IS 'Stores folder hierarchy for document organization';
COMMENT ON TABLE documents IS 'Stores document metadata for uploaded files';
COMMENT ON TABLE document_versions IS 'Stores version information for documents';
COMMENT ON TABLE document_metadata IS 'Stores custom metadata for documents as key-value pairs';
COMMENT ON TABLE tags IS 'Stores tags for document categorization';
COMMENT ON TABLE document_tags IS 'Maps documents to tags (many-to-many relationship)';
COMMENT ON TABLE permissions IS 'Stores access control permissions for resources';