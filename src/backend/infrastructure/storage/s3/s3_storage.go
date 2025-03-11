// Package s3 implements the StorageService interface using AWS S3 for document storage.
package s3

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws" // v1.44.0+
	"github.com/aws/aws-sdk-go/aws/credentials" // v1.44.0+
	"github.com/aws/aws-sdk-go/aws/session" // v1.44.0+
	"github.com/aws/aws-sdk-go/service/s3" // v1.44.0+
	"github.com/aws/aws-sdk-go/service/s3/s3manager" // v1.44.0+

	"../../../domain/services"
	"../../../pkg/config"
	"../../../pkg/logger"
	"../../../pkg/utils"
)

// s3Storage implements the StorageService interface using AWS S3
type s3Storage struct {
	client     *s3.S3
	uploader   *s3manager.Uploader
	downloader *s3manager.Downloader
	config     config.S3Config
}

// NewS3Storage creates a new S3 storage service with the provided configuration
func NewS3Storage(config config.S3Config) services.StorageService {
	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(config.Region),
		Endpoint:         aws.String(config.Endpoint),
		Credentials:      credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, ""),
		S3ForcePathStyle: aws.Bool(config.ForcePathStyle),
		DisableSSL:       aws.Bool(!config.UseSSL),
	})

	if err != nil {
		logger.Error("Failed to create AWS session", "error", err.Error())
		return nil
	}

	// Create S3 client, uploader, and downloader
	s3Client := s3.New(sess)
	uploader := s3manager.NewUploader(sess)
	downloader := s3manager.NewDownloader(sess)

	return &s3Storage{
		client:     s3Client,
		uploader:   uploader,
		downloader: downloader,
		config:     config,
	}
}

// StoreTemporary stores a document in temporary storage during processing.
// It ensures tenant isolation by using tenantID in the storage path.
func (s *s3Storage) StoreTemporary(ctx context.Context, tenantID string, documentID string, content io.Reader, size int64, contentType string) (string, error) {
	// Validate inputs
	if tenantID == "" {
		return "", errors.New("tenant ID cannot be empty")
	}
	if documentID == "" {
		return "", errors.New("document ID cannot be empty")
	}
	if content == nil {
		return "", errors.New("content cannot be nil")
	}

	// Generate temporary storage path with tenant isolation
	storagePath := fmt.Sprintf("temp/%s/%s", tenantID, documentID)

	// Log the upload operation
	logger.InfoContext(ctx, "Storing document in temporary storage",
		"tenant_id", tenantID,
		"document_id", documentID,
		"size", size,
		"content_type", contentType,
		"storage_path", storagePath)

	// Prepare upload input
	uploadInput := &s3manager.UploadInput{
		Bucket:               aws.String(s.config.TempBucket),
		Key:                  aws.String(storagePath),
		Body:                 content,
		ContentType:          aws.String(contentType),
		ContentLength:        aws.Int64(size),
		ServerSideEncryption: aws.String("AES256"), // Enable server-side encryption
	}

	// Upload to S3
	_, err := s.uploader.UploadWithContext(ctx, uploadInput)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to upload document to temporary storage",
			"tenant_id", tenantID,
			"document_id", documentID,
			"error", err.Error())
		return "", err
	}

	// Log successful upload
	logger.InfoContext(ctx, "Document stored in temporary storage",
		"tenant_id", tenantID,
		"document_id", documentID,
		"storage_path", storagePath)

	return storagePath, nil
}

// StorePermanent moves a document from temporary to permanent storage after processing.
// It ensures tenant isolation by using tenantID in the storage path.
func (s *s3Storage) StorePermanent(ctx context.Context, tenantID string, documentID string, versionID string, folderID string, tempPath string) (string, error) {
	// Validate inputs
	if tenantID == "" {
		return "", errors.New("tenant ID cannot be empty")
	}
	if documentID == "" {
		return "", errors.New("document ID cannot be empty")
	}
	if versionID == "" {
		return "", errors.New("version ID cannot be empty")
	}
	if tempPath == "" {
		return "", errors.New("temporary path cannot be empty")
	}

	// Generate permanent storage path with tenant isolation
	permanentPath := fmt.Sprintf("%s/%s/%s/%s", tenantID, folderID, documentID, versionID)

	// Log the move operation
	logger.InfoContext(ctx, "Moving document from temporary to permanent storage",
		"tenant_id", tenantID,
		"document_id", documentID,
		"version_id", versionID,
		"folder_id", folderID,
		"temp_path", tempPath,
		"permanent_path", permanentPath)

	// Copy object from temporary to permanent storage
	_, err := s.client.CopyObjectWithContext(ctx, &s3.CopyObjectInput{
		Bucket:               aws.String(s.config.Bucket),
		CopySource:           aws.String(fmt.Sprintf("%s/%s", s.config.TempBucket, tempPath)),
		Key:                  aws.String(permanentPath),
		ServerSideEncryption: aws.String("AES256"), // Enable server-side encryption
	})

	if err != nil {
		logger.ErrorContext(ctx, "Failed to copy document from temporary to permanent storage",
			"tenant_id", tenantID,
			"document_id", documentID,
			"error", err.Error())
		return "", err
	}

	// Delete the temporary object
	_, err = s.client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.config.TempBucket),
		Key:    aws.String(tempPath),
	})

	if err != nil {
		logger.WarnContext(ctx, "Failed to delete document from temporary storage",
			"tenant_id", tenantID,
			"document_id", documentID,
			"error", err.Error())
		// We continue even if deletion fails - the temp bucket should have lifecycle policies
	}

	// Log successful move
	logger.InfoContext(ctx, "Document moved to permanent storage",
		"tenant_id", tenantID,
		"document_id", documentID,
		"permanent_path", permanentPath)

	return permanentPath, nil
}

// MoveToQuarantine moves a document from temporary to quarantine storage when a virus is detected.
// It ensures tenant isolation by using tenantID in the storage path.
func (s *s3Storage) MoveToQuarantine(ctx context.Context, tenantID string, documentID string, tempPath string) (string, error) {
	// Validate inputs
	if tenantID == "" {
		return "", errors.New("tenant ID cannot be empty")
	}
	if documentID == "" {
		return "", errors.New("document ID cannot be empty")
	}
	if tempPath == "" {
		return "", errors.New("temporary path cannot be empty")
	}

	// Generate quarantine storage path with tenant isolation
	quarantinePath := fmt.Sprintf("quarantine/%s/%s", tenantID, documentID)

	// Log the quarantine operation
	logger.InfoContext(ctx, "Moving document from temporary to quarantine storage",
		"tenant_id", tenantID,
		"document_id", documentID,
		"temp_path", tempPath,
		"quarantine_path", quarantinePath)

	// Copy object from temporary to quarantine storage
	_, err := s.client.CopyObjectWithContext(ctx, &s3.CopyObjectInput{
		Bucket:               aws.String(s.config.QuarantineBucket),
		CopySource:           aws.String(fmt.Sprintf("%s/%s", s.config.TempBucket, tempPath)),
		Key:                  aws.String(quarantinePath),
		ServerSideEncryption: aws.String("AES256"), // Enable server-side encryption
	})

	if err != nil {
		logger.ErrorContext(ctx, "Failed to copy document from temporary to quarantine storage",
			"tenant_id", tenantID,
			"document_id", documentID,
			"error", err.Error())
		return "", err
	}

	// Delete the temporary object
	_, err = s.client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.config.TempBucket),
		Key:    aws.String(tempPath),
	})

	if err != nil {
		logger.WarnContext(ctx, "Failed to delete document from temporary storage",
			"tenant_id", tenantID,
			"document_id", documentID,
			"error", err.Error())
		// We continue even if deletion fails - the temp bucket should have lifecycle policies
	}

	// Log successful quarantine
	logger.InfoContext(ctx, "Document moved to quarantine storage",
		"tenant_id", tenantID,
		"document_id", documentID,
		"quarantine_path", quarantinePath)

	return quarantinePath, nil
}

// GetDocument retrieves a document from storage.
func (s *s3Storage) GetDocument(ctx context.Context, storagePath string) (io.ReadCloser, error) {
	// Validate storage path
	if storagePath == "" {
		return nil, errors.New("storage path cannot be empty")
	}

	// Determine the bucket based on the storage path
	bucket, key, err := s.parseBucketAndKey(storagePath)
	if err != nil {
		return nil, err
	}

	// Log the download operation
	logger.InfoContext(ctx, "Retrieving document from storage",
		"storage_path", storagePath,
		"bucket", bucket,
		"key", key)

	// Get object from S3
	result, err := s.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		logger.ErrorContext(ctx, "Failed to retrieve document from storage",
			"storage_path", storagePath,
			"error", err.Error())
		return nil, err
	}

	return result.Body, nil
}

// GetPresignedURL generates a presigned URL for direct document download.
func (s *s3Storage) GetPresignedURL(ctx context.Context, storagePath string, fileName string, expirationSeconds int) (string, error) {
	// Validate inputs
	if storagePath == "" {
		return "", errors.New("storage path cannot be empty")
	}
	if fileName == "" {
		return "", errors.New("file name cannot be empty")
	}
	if expirationSeconds <= 0 {
		return "", errors.New("expiration seconds must be positive")
	}

	// Determine the bucket based on the storage path
	bucket, key, err := s.parseBucketAndKey(storagePath)
	if err != nil {
		return "", err
	}

	// Log the presigned URL generation
	logger.InfoContext(ctx, "Generating presigned URL for document download",
		"storage_path", storagePath,
		"file_name", fileName,
		"expiration_seconds", expirationSeconds)

	// Create request for the GetObject operation
	req, _ := s.client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		ResponseContentDisposition: aws.String(fmt.Sprintf("attachment; filename=%s", fileName)),
	})

	// Generate presigned URL with expiration time
	url, err := req.Presign(time.Duration(expirationSeconds) * time.Second)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to generate presigned URL",
			"storage_path", storagePath,
			"error", err.Error())
		return "", err
	}

	logger.InfoContext(ctx, "Presigned URL generated successfully",
		"storage_path", storagePath,
		"expiration_seconds", expirationSeconds)

	return url, nil
}

// DeleteDocument deletes a document from storage.
func (s *s3Storage) DeleteDocument(ctx context.Context, storagePath string) error {
	// Validate storage path
	if storagePath == "" {
		return errors.New("storage path cannot be empty")
	}

	// Determine the bucket based on the storage path
	bucket, key, err := s.parseBucketAndKey(storagePath)
	if err != nil {
		return err
	}

	// Log the delete operation
	logger.InfoContext(ctx, "Deleting document from storage",
		"storage_path", storagePath,
		"bucket", bucket,
		"key", key)

	// Delete object from S3
	_, err = s.client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		logger.ErrorContext(ctx, "Failed to delete document from storage",
			"storage_path", storagePath,
			"error", err.Error())
		return err
	}

	logger.InfoContext(ctx, "Document deleted from storage",
		"storage_path", storagePath)

	return nil
}

// CreateBatchArchive creates a compressed archive of multiple documents.
func (s *s3Storage) CreateBatchArchive(ctx context.Context, storagePaths []string, filenames []string) (io.ReadCloser, error) {
	// Validate inputs
	if len(storagePaths) == 0 {
		return nil, errors.New("storage paths cannot be empty")
	}

	if len(storagePaths) != len(filenames) {
		return nil, errors.New("number of storage paths must match number of filenames")
	}

	// Log the batch archive creation
	logger.InfoContext(ctx, "Creating batch archive",
		"document_count", len(storagePaths))

	// Create a buffer to write the ZIP archive to
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Add each document to the archive
	for i, storagePath := range storagePaths {
		// Get the document content
		reader, err := s.GetDocument(ctx, storagePath)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to retrieve document for batch archive",
				"storage_path", storagePath,
				"error", err.Error())
			zipWriter.Close()
			return nil, err
		}

		// Create a new file in the ZIP archive
		filename := filenames[i]
		writer, err := zipWriter.Create(filename)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to create file in ZIP archive",
				"filename", filename,
				"error", err.Error())
			reader.Close()
			zipWriter.Close()
			return nil, err
		}

		// Copy the document content to the ZIP file
		_, err = utils.CopyReader(reader, writer)
		reader.Close()
		if err != nil {
			logger.ErrorContext(ctx, "Failed to copy document content to ZIP archive",
				"storage_path", storagePath,
				"error", err.Error())
			zipWriter.Close()
			return nil, err
		}
	}

	// Close the ZIP writer
	err := zipWriter.Close()
	if err != nil {
		logger.ErrorContext(ctx, "Failed to close ZIP writer",
			"error", err.Error())
		return nil, err
	}

	// Create a ReadCloser from the buffer
	readCloser := io.NopCloser(bytes.NewReader(buf.Bytes()))

	logger.InfoContext(ctx, "Batch archive created successfully",
		"document_count", len(storagePaths),
		"archive_size", buf.Len())

	return readCloser, nil
}

// parseBucketAndKey parses a storage path into bucket and key components
func (s *s3Storage) parseBucketAndKey(storagePath string) (string, string, error) {
	var bucket string

	// Determine which bucket to use based on the path prefix
	if strings.HasPrefix(storagePath, "temp/") {
		bucket = s.config.TempBucket
	} else if strings.HasPrefix(storagePath, "quarantine/") {
		bucket = s.config.QuarantineBucket
	} else {
		bucket = s.config.Bucket
	}

	return bucket, storagePath, nil
}