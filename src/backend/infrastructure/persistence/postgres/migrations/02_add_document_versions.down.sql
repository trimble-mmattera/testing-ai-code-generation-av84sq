-- Drop the trigger that updates version count when a new version is added
DROP TRIGGER update_document_version_count ON document_versions;

-- Drop the function that increments version count and updates current version ID
DROP FUNCTION update_document_version_count_function();

-- Drop the trigger that sets initial version count for new documents
DROP TRIGGER set_initial_version_count ON documents;

-- Drop the function that sets initial version count
DROP FUNCTION set_initial_version_count_function();

-- Drop the index on current_version_id column
DROP INDEX idx_documents_current_version_id;

-- Remove version-related columns from the documents table
ALTER TABLE documents
DROP COLUMN current_version_id,
DROP COLUMN version_count;

-- Reverted document versioning enhancements