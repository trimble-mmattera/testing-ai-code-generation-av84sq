// Package thumbnails implements the ThumbnailService interface for generating
// and managing document thumbnails in the Document Management Platform.
package thumbnails

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"

	"github.com/nfnt/resize" // v0.0.0-20180221191011-83c6a9932646
	"github.com/pdfcpu/pdfcpu/pkg/api" // v0.4.0

	"../../../domain/services"
	"../../../pkg/config"
	"../../../pkg/logger"
	"../../../pkg/utils"
)

// thumbnailPathPrefix defines the prefix for thumbnail storage paths
const thumbnailPathPrefix = "thumbnails"

// thumbnailGenerator implements the ThumbnailService interface
type thumbnailGenerator struct {
	storageService services.StorageService
	config         config.S3Config
}

// NewThumbnailGenerator creates a new thumbnail generator service with the provided storage service and configuration
func NewThumbnailGenerator(storageService services.StorageService, config config.S3Config) services.ThumbnailService {
	if storageService == nil {
		panic("storageService is required")
	}
	return &thumbnailGenerator{
		storageService: storageService,
		config:         config,
	}
}

// GenerateThumbnail generates a thumbnail for a document
func (t *thumbnailGenerator) GenerateThumbnail(ctx context.Context, documentID, versionID, tenantID, storagePath string) (string, error) {
	// Validate input parameters
	if documentID == "" || versionID == "" || tenantID == "" || storagePath == "" {
		return "", errors.New("missing required parameters for thumbnail generation")
	}

	logger.InfoContext(ctx, "Generating thumbnail", 
		"documentID", documentID, 
		"versionID", versionID, 
		"tenantID", tenantID,
		"storagePath", storagePath)

	// Generate thumbnail path with tenant isolation
	thumbnailPath := t.generateThumbnailPath(tenantID, documentID, versionID)

	// Get document content from storage service
	documentContent, err := t.storageService.GetDocument(ctx, storagePath)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to get document content", 
			"error", err.Error(),
			"documentID", documentID,
			"storagePath", storagePath)
		return "", fmt.Errorf("failed to get document content: %w", err)
	}
	defer documentContent.Close()

	// Determine document type from storage path extension
	fileExt := utils.GetFileExtension(storagePath)
	fileExt = strings.ToLower(fileExt)

	var thumbnailData []byte

	// Generate thumbnail based on document type
	switch fileExt {
	case ".pdf":
		thumbnailData, err = t.generatePDFThumbnail(ctx, documentContent)
	case ".jpg", ".jpeg":
		thumbnailData, err = t.generateImageThumbnail(ctx, documentContent, "image/jpeg")
	case ".png":
		thumbnailData, err = t.generateImageThumbnail(ctx, documentContent, "image/png")
	case ".gif":
		thumbnailData, err = t.generateImageThumbnail(ctx, documentContent, "image/gif")
	default:
		// For other document types, generate a generic icon
		thumbnailData, err = t.generateGenericThumbnail(ctx, fileExt)
	}

	if err != nil {
		logger.ErrorContext(ctx, "Failed to generate thumbnail", 
			"error", err.Error(),
			"documentID", documentID,
			"fileType", fileExt)
		return "", fmt.Errorf("failed to generate thumbnail: %w", err)
	}

	// Store thumbnail in S3 using storage service
	err = t.storageService.StoreFile(ctx, thumbnailPath, bytes.NewReader(thumbnailData), int64(len(thumbnailData)), "image/png")
	if err != nil {
		logger.ErrorContext(ctx, "Failed to store thumbnail", 
			"error", err.Error(),
			"documentID", documentID,
			"thumbnailPath", thumbnailPath)
		return "", fmt.Errorf("failed to store thumbnail: %w", err)
	}

	logger.InfoContext(ctx, "Thumbnail generated successfully", 
		"documentID", documentID, 
		"thumbnailPath", thumbnailPath)

	return thumbnailPath, nil
}

// GetThumbnail retrieves a document thumbnail
func (t *thumbnailGenerator) GetThumbnail(ctx context.Context, documentID, versionID, tenantID string) (io.ReadCloser, error) {
	// Validate input parameters
	if documentID == "" || versionID == "" || tenantID == "" {
		return nil, errors.New("missing required parameters for thumbnail retrieval")
	}

	logger.InfoContext(ctx, "Retrieving thumbnail", 
		"documentID", documentID, 
		"versionID", versionID, 
		"tenantID", tenantID)

	// Generate thumbnail path with tenant isolation
	thumbnailPath := t.generateThumbnailPath(tenantID, documentID, versionID)

	// Get thumbnail content from storage service
	thumbnailContent, err := t.storageService.GetDocument(ctx, thumbnailPath)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to retrieve thumbnail", 
			"error", err.Error(),
			"documentID", documentID,
			"thumbnailPath", thumbnailPath)
		return nil, fmt.Errorf("failed to retrieve thumbnail: %w", err)
	}

	logger.InfoContext(ctx, "Thumbnail retrieved successfully", 
		"documentID", documentID, 
		"thumbnailPath", thumbnailPath)

	return thumbnailContent, nil
}

// GetThumbnailURL generates a URL for accessing a document thumbnail
func (t *thumbnailGenerator) GetThumbnailURL(ctx context.Context, documentID, versionID, tenantID string, expirationSeconds int) (string, error) {
	// Validate input parameters
	if documentID == "" || versionID == "" || tenantID == "" || expirationSeconds <= 0 {
		return "", errors.New("missing required parameters for thumbnail URL generation")
	}

	logger.InfoContext(ctx, "Generating thumbnail URL", 
		"documentID", documentID, 
		"versionID", versionID, 
		"tenantID", tenantID,
		"expirationSeconds", expirationSeconds)

	// Generate thumbnail path with tenant isolation
	thumbnailPath := t.generateThumbnailPath(tenantID, documentID, versionID)

	// Generate presigned URL for thumbnail using storage service
	url, err := t.storageService.GetPresignedURL(ctx, thumbnailPath, expirationSeconds)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to generate thumbnail URL", 
			"error", err.Error(),
			"documentID", documentID,
			"thumbnailPath", thumbnailPath)
		return "", fmt.Errorf("failed to generate thumbnail URL: %w", err)
	}

	logger.InfoContext(ctx, "Thumbnail URL generated successfully", 
		"documentID", documentID, 
		"thumbnailPath", thumbnailPath)

	return url, nil
}

// DeleteThumbnail deletes a document thumbnail
func (t *thumbnailGenerator) DeleteThumbnail(ctx context.Context, documentID, versionID, tenantID string) error {
	// Validate input parameters
	if documentID == "" || versionID == "" || tenantID == "" {
		return errors.New("missing required parameters for thumbnail deletion")
	}

	logger.InfoContext(ctx, "Deleting thumbnail", 
		"documentID", documentID, 
		"versionID", versionID, 
		"tenantID", tenantID)

	// Generate thumbnail path with tenant isolation
	thumbnailPath := t.generateThumbnailPath(tenantID, documentID, versionID)

	// Delete thumbnail using storage service
	err := t.storageService.DeleteFile(ctx, thumbnailPath)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to delete thumbnail", 
			"error", err.Error(),
			"documentID", documentID,
			"thumbnailPath", thumbnailPath)
		return fmt.Errorf("failed to delete thumbnail: %w", err)
	}

	logger.InfoContext(ctx, "Thumbnail deleted successfully", 
		"documentID", documentID, 
		"thumbnailPath", thumbnailPath)

	return nil
}

// generateThumbnailPath generates a storage path for a thumbnail with tenant isolation
func (t *thumbnailGenerator) generateThumbnailPath(tenantID, documentID, versionID string) string {
	// Validate input parameters
	if tenantID == "" || documentID == "" || versionID == "" {
		return ""
	}

	// Construct path using format: thumbnails/{tenantID}/{documentID}/{versionID}
	return fmt.Sprintf("%s/%s/%s/%s", thumbnailPathPrefix, tenantID, documentID, versionID)
}

// generatePDFThumbnail generates a thumbnail from a PDF document
func (t *thumbnailGenerator) generatePDFThumbnail(ctx context.Context, content io.Reader) ([]byte, error) {
	// Create temporary buffer for PDF content
	var buf bytes.Buffer
	_, err := utils.CopyReader(content, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to copy PDF content: %w", err)
	}

	// Use pdfcpu to extract first page as image
	var imgBuf bytes.Buffer
	err = api.ExtractFirstPage(bytes.NewReader(buf.Bytes()), &imgBuf, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to extract first page from PDF: %w", err)
	}

	// Decode extracted image
	img, _, err := image.Decode(&imgBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PDF page: %w", err)
	}

	// Resize extracted image to thumbnail dimensions
	thumbnailImg := resize.Thumbnail(
		services.DefaultThumbnailWidth, 
		services.DefaultThumbnailHeight, 
		img, 
		resize.Lanczos3,
	)

	// Encode resized image as PNG
	var thumbnailBuf bytes.Buffer
	err = png.Encode(&thumbnailBuf, thumbnailImg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return thumbnailBuf.Bytes(), nil
}

// generateImageThumbnail generates a thumbnail from an image
func (t *thumbnailGenerator) generateImageThumbnail(ctx context.Context, content io.Reader, contentType string) ([]byte, error) {
	// Decode image based on content type
	var img image.Image
	var err error

	switch contentType {
	case "image/jpeg":
		img, err = jpeg.Decode(content)
	case "image/png":
		img, err = png.Decode(content)
	case "image/gif":
		img, err = gif.Decode(content)
	default:
		return nil, fmt.Errorf("unsupported image content type: %s", contentType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize image to thumbnail dimensions using resize package
	thumbnailImg := resize.Thumbnail(
		services.DefaultThumbnailWidth, 
		services.DefaultThumbnailHeight, 
		img, 
		resize.Lanczos3,
	)

	// Encode resized image as PNG
	var thumbnailBuf bytes.Buffer
	err = png.Encode(&thumbnailBuf, thumbnailImg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return thumbnailBuf.Bytes(), nil
}

// generateGenericThumbnail generates a generic thumbnail for unsupported document types
func (t *thumbnailGenerator) generateGenericThumbnail(ctx context.Context, contentType string) ([]byte, error) {
	// Select appropriate generic icon based on content type
	// In a real implementation, this would return predefined icons for different file types
	// For example, a document icon for .docx, a spreadsheet icon for .xlsx, etc.
	
	// As a simplified implementation, we'll just return a placeholder error
	return nil, fmt.Errorf("generation of generic thumbnails not yet implemented for: %s", contentType)
}