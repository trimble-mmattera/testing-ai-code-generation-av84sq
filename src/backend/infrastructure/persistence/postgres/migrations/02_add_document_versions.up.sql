-- Add version tracking columns to the documents table
ALTER TABLE documents
ADD COLUMN current_version_id UUID REFERENCES document_versions(id) NULL,
ADD COLUMN version_count INTEGER NOT NULL DEFAULT 0;

-- Create an index on the current_version_id column for faster lookups
CREATE INDEX idx_documents_current_version_id ON documents(current_version_id);

-- Create a function to set the initial version count for new documents
CREATE OR REPLACE FUNCTION set_initial_version_count_function()
RETURNS trigger AS $$
BEGIN
    NEW.version_count := 0;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create a trigger to set the initial version count when a document is created
CREATE TRIGGER set_initial_version_count
BEFORE INSERT ON documents
FOR EACH ROW
EXECUTE FUNCTION set_initial_version_count_function();

-- Create a function to increment version count and update current version ID when a new version is added
CREATE OR REPLACE FUNCTION update_document_version_count_function()
RETURNS trigger AS $$
BEGIN
    UPDATE documents
    SET version_count = version_count + 1,
        current_version_id = NEW.id
    WHERE id = NEW.document_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create a trigger to update version count and current version ID when a new version is added
CREATE TRIGGER update_document_version_count
AFTER INSERT ON document_versions
FOR EACH ROW
EXECUTE FUNCTION update_document_version_count_function();

-- Add comments to the new columns for documentation
COMMENT ON COLUMN documents.current_version_id IS 'Reference to the current version of the document';
COMMENT ON COLUMN documents.version_count IS 'Count of versions for this document';