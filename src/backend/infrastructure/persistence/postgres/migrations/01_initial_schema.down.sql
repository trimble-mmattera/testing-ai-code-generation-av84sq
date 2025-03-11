-- Drop permissions table
DROP TABLE IF EXISTS permissions CASCADE;

-- Drop document_tags table
DROP TABLE IF EXISTS document_tags CASCADE;

-- Drop tags table
DROP TABLE IF EXISTS tags CASCADE;

-- Drop document_metadata table
DROP TABLE IF EXISTS document_metadata CASCADE;

-- Drop document_versions table
DROP TABLE IF EXISTS document_versions CASCADE;

-- Drop documents table
DROP TABLE IF EXISTS documents CASCADE;

-- Drop folders table
DROP TABLE IF EXISTS folders CASCADE;

-- Drop user_roles table
DROP TABLE IF EXISTS user_roles CASCADE;

-- Drop roles table
DROP TABLE IF EXISTS roles CASCADE;

-- Drop user_settings table
DROP TABLE IF EXISTS user_settings CASCADE;

-- Drop users table
DROP TABLE IF EXISTS users CASCADE;

-- Drop tenant_settings table
DROP TABLE IF EXISTS tenant_settings CASCADE;

-- Drop tenants table
DROP TABLE IF EXISTS tenants CASCADE;

-- Drop UUID extension
DROP EXTENSION IF EXISTS "uuid-ossp";