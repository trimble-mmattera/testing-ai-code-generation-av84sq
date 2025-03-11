// Package utils provides file utility functions for the Document Management Platform.
// This file contains utilities for file operations, content type detection, file validation,
// and other file-related helper functions used across the system.
package utils

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"../errors" // For standardized error handling
)

// MaxFileSize defines the maximum allowed file size (100MB)
const MaxFileSize int64 = 100 * 1024 * 1024

// AllowedFileTypes defines a map of content types that are allowed for upload
var AllowedFileTypes = map[string]bool{
	"application/pdf":                                             true,
	"application/msword":                                          true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel":                                    true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"application/vnd.ms-powerpoint":                               true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"image/jpeg":                                                  true,
	"image/png":                                                   true,
	"image/gif":                                                   true,
	"image/tiff":                                                  true,
	"text/plain":                                                  true,
	"text/csv":                                                    true,
	"application/json":                                            true,
	"application/xml":                                             true,
	"application/zip":                                             true,
	"application/x-rar-compressed":                                true,
	"application/x-tar":                                           true,
	"application/gzip":                                            true,
}

// DetectContentType detects the content type of a file based on its content
func DetectContentType(r io.Reader) (string, error) {
	// Create a buffer to read the first 512 bytes of the file
	buffer := make([]byte, 512)
	
	// Read up to 512 bytes from the reader
	n, err := r.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}
	
	// Use http.DetectContentType to determine the content type
	contentType := http.DetectContentType(buffer[:n])
	
	return contentType, nil
}

// ValidateFileSize validates that a file size is within the allowed limit
func ValidateFileSize(size int64) error {
	// Check if the size is greater than MaxFileSize
	if size > MaxFileSize {
		return errors.NewValidationError(
			fmt.Sprintf("File size exceeds the maximum allowed size of %d bytes", MaxFileSize),
		)
	}
	
	return nil
}

// ValidateContentType validates that a content type is in the allowed list
func ValidateContentType(contentType string) error {
	// Check if the content type is in the AllowedFileTypes map
	if !AllowedFileTypes[contentType] {
		return errors.NewValidationError(
			fmt.Sprintf("Content type '%s' is not allowed", contentType),
		)
	}
	
	return nil
}

// GetFileExtension gets the file extension from a filename
func GetFileExtension(filename string) string {
	// Use filepath.Ext to extract the extension
	return filepath.Ext(filename)
}

// GetFileNameWithoutExtension gets the filename without its extension
func GetFileNameWithoutExtension(filename string) string {
	// Get the extension using GetFileExtension
	extension := GetFileExtension(filename)
	
	// Remove the extension from the filename
	return filename[:len(filename)-len(extension)]
}

// SanitizeFileName sanitizes a filename to remove invalid characters
func SanitizeFileName(filename string) string {
	// Replace invalid characters with underscores
	// Common invalid characters for cross-platform compatibility
	invalid := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	result := filename
	
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	
	// Trim spaces from the beginning and end
	return strings.TrimSpace(result)
}

// CreateTempFile creates a temporary file with the given content
func CreateTempFile(content io.Reader, prefix string) (*os.File, error) {
	// Create a temporary file with the given prefix
	tempFile, err := os.CreateTemp("", prefix)
	if err != nil {
		return nil, err
	}
	
	// Copy the content to the temporary file
	_, err = io.Copy(tempFile, content)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, err
	}
	
	// Seek to the beginning of the file
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, err
	}
	
	return tempFile, nil
}

// CopyFile copies a file from source to destination
func CopyFile(src, dst string) (int64, error) {
	// Open the source file
	srcFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()
	
	// Create the destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer dstFile.Close()
	
	// Copy the content from source to destination
	return io.Copy(dstFile, srcFile)
}

// CopyReader copies data from a reader to a writer with buffer
func CopyReader(src io.Reader, dst io.Writer) (int64, error) {
	// Create a buffered reader for the source
	bufferedSrc := bufio.NewReader(src)
	
	// Copy data from the buffered reader to the destination
	return io.Copy(dst, bufferedSrc)
}

// ReadFileToString reads a file and returns its content as a string
func ReadFileToString(filePath string) (string, error) {
	// Read the file content using os.ReadFile
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	
	// Convert the byte slice to a string
	return string(bytes), nil
}

// ReadFileToBytes reads a file and returns its content as a byte slice
func ReadFileToBytes(filePath string) ([]byte, error) {
	// Read the file content using os.ReadFile
	return os.ReadFile(filePath)
}

// WriteStringToFile writes a string to a file
func WriteStringToFile(filePath, content string) error {
	// Convert the string to a byte slice
	return os.WriteFile(filePath, []byte(content), 0644)
}

// WriteBytesToFile writes a byte slice to a file
func WriteBytesToFile(filePath string, content []byte) error {
	// Write the byte slice to the file using os.WriteFile
	return os.WriteFile(filePath, content, 0644)
}

// EnsureDirectoryExists ensures that a directory exists, creating it if necessary
func EnsureDirectoryExists(dirPath string) error {
	// Check if the directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// If not, create it with appropriate permissions
		return os.MkdirAll(dirPath, 0755)
	}
	
	return nil
}

// GetFileSize gets the size of a file in bytes
func GetFileSize(filePath string) (int64, error) {
	// Get file information using os.Stat
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	
	// Return the file size
	return fileInfo.Size(), nil
}

// GetReaderSize attempts to determine the size of a reader's content
func GetReaderSize(r io.Reader) (int64, error) {
	// Check if the reader implements io.Seeker
	seeker, ok := r.(io.Seeker)
	if !ok {
		return 0, errors.NewValidationError("Cannot determine size of reader: not seekable")
	}
	
	// Get the current position
	currentPos, err := seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	
	// Seek to the end to determine size
	size, err := seeker.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	
	// Seek back to the original position
	_, err = seeker.Seek(currentPos, io.SeekStart)
	if err != nil {
		return 0, err
	}
	
	return size, nil
}

// LimitReader creates a reader that will read at most n bytes from the source reader
func LimitReader(r io.Reader, n int64) io.Reader {
	// Use io.LimitReader to create a limited reader
	return io.LimitReader(r, n)
}

// IsDirectory checks if a path is a directory
func IsDirectory(path string) (bool, error) {
	// Get file information using os.Stat
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	
	// Check if the file mode indicates a directory
	return fileInfo.IsDir(), nil
}

// IsFile checks if a path is a regular file
func IsFile(path string) (bool, error) {
	// Get file information using os.Stat
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	
	// Check if the file mode indicates a regular file
	return fileInfo.Mode().IsRegular(), nil
}

// CalculateFileHash calculates a hash of a file's content
func CalculateFileHash(r io.Reader, algorithm string) (string, error) {
	var h hash.Hash
	
	// Select the hash algorithm
	switch strings.ToLower(algorithm) {
	case "md5":
		h = md5.New()
	case "sha1":
		h = sha1.New()
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	default:
		return "", errors.NewValidationError(fmt.Sprintf("Unsupported hash algorithm: %s", algorithm))
	}
	
	// Copy the reader to the hash
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	
	// Convert the hash to a hex string
	return hex.EncodeToString(h.Sum(nil)), nil
}