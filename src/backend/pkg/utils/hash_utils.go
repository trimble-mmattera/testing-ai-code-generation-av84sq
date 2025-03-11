// Package utils provides hash utility functions for the Document Management Platform.
// This file contains utilities for calculating and verifying cryptographic hashes of
// document content, supporting document integrity verification and content identification.
package utils

import (
	"crypto/md5"    // standard library
	"crypto/sha1"   // standard library
	"crypto/sha256" // standard library
	"crypto/sha512" // standard library
	"encoding/hex"  // standard library
	"hash"          // standard library
	"io"            // standard library
	"strings"       // standard library

	"../errors" // For standardized error handling
)

// Hash algorithm constants
const (
	HashAlgorithmMD5    = "md5"
	HashAlgorithmSHA1   = "sha1"
	HashAlgorithmSHA256 = "sha256"
	HashAlgorithmSHA512 = "sha512"
	
	// Default hash algorithm used when none is specified
	DefaultHashAlgorithm = HashAlgorithmSHA256
)

// HashBytes calculates a hash of a byte slice using the specified algorithm.
func HashBytes(data []byte, algorithm string) (string, error) {
	hasher, err := GetHasher(algorithm)
	if err != nil {
		return "", err
	}
	
	hasher.Write(data)
	hashSum := hasher.Sum(nil)
	return hex.EncodeToString(hashSum), nil
}

// HashString calculates a hash of a string using the specified algorithm.
func HashString(data string, algorithm string) (string, error) {
	return HashBytes([]byte(data), algorithm)
}

// HashReader calculates a hash of data from a reader using the specified algorithm.
func HashReader(reader io.Reader, algorithm string) (string, error) {
	hasher, err := GetHasher(algorithm)
	if err != nil {
		return "", err
	}
	
	if _, err := io.Copy(hasher, reader); err != nil {
		return "", err
	}
	
	hashSum := hasher.Sum(nil)
	return hex.EncodeToString(hashSum), nil
}

// CalculateFileHash calculates a hash of a file's content from a reader.
func CalculateFileHash(r io.Reader, algorithm string) (string, error) {
	if algorithm == "" {
		algorithm = DefaultHashAlgorithm
	}
	
	return HashReader(r, algorithm)
}

// VerifyHash verifies that a hash matches the expected value.
func VerifyHash(expectedHash string, actualHash string) (bool, error) {
	// Case-insensitive comparison
	expectedHash = strings.ToLower(expectedHash)
	actualHash = strings.ToLower(actualHash)
	
	if expectedHash == actualHash {
		return true, nil
	}
	
	return false, errors.NewValidationError("hash mismatch")
}

// GetHasher returns a hash.Hash instance for the specified algorithm.
func GetHasher(algorithm string) (hash.Hash, error) {
	algorithm = strings.ToLower(algorithm)
	
	switch algorithm {
	case HashAlgorithmMD5:
		return md5.New(), nil
	case HashAlgorithmSHA1:
		return sha1.New(), nil
	case HashAlgorithmSHA256:
		return sha256.New(), nil
	case HashAlgorithmSHA512:
		return sha512.New(), nil
	default:
		return nil, errors.NewValidationError("unsupported hash algorithm: " + algorithm)
	}
}

// IsValidHashAlgorithm checks if the specified algorithm is valid.
func IsValidHashAlgorithm(algorithm string) bool {
	algorithm = strings.ToLower(algorithm)
	
	switch algorithm {
	case HashAlgorithmMD5, HashAlgorithmSHA1, HashAlgorithmSHA256, HashAlgorithmSHA512:
		return true
	default:
		return false
	}
}