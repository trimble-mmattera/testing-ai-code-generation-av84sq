// Package clamav provides a client implementation for the ClamAV antivirus service.
// It enables virus scanning capabilities for the Document Management Platform by
// communicating with a ClamAV daemon to scan document content for malicious code.
package clamav

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"../../../pkg/errors"
	"../../../pkg/logger"
)

// Default values and constants
const (
	defaultTimeout = 30 * time.Second
	chunkSize      = 8192
)

var (
	okResponse     = []byte("OK")
	foundResponse  = []byte("FOUND")
	errorResponse  = []byte("ERROR")
	inStreamCommand = []byte("zINSTREAM\x00")
)

// clamAVClient is a client for communicating with ClamAV daemon
type clamAVClient struct {
	address string
	timeout time.Duration
	logger  logger.Logger
}

// NewClamAVClient creates a new ClamAV client with the specified address
func NewClamAVClient(address string) (*clamAVClient, error) {
	if address == "" {
		return nil, errors.NewValidationError("ClamAV address cannot be empty")
	}
	
	client := &clamAVClient{
		address: address,
		timeout: defaultTimeout,
	}
	
	return client, nil
}

// ScanStream scans a document stream for viruses
func (c *clamAVClient) ScanStream(ctx context.Context, reader io.Reader) (bool, string, error) {
	log := logger.WithContext(ctx)
	log.Info("Starting virus scan")
	
	if reader == nil {
		return false, "", errors.NewValidationError("Reader cannot be nil")
	}
	
	// Establish connection to ClamAV daemon with timeout
	conn, err := net.DialTimeout("tcp", c.address, c.timeout)
	if err != nil {
		log = logger.WithError(err)
		log.Error("Failed to connect to ClamAV daemon")
		return false, "", errors.NewDependencyError(fmt.Sprintf("Failed to connect to ClamAV: %s", err.Error()))
	}
	defer conn.Close()
	
	// Set deadline for the connection
	if err := conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		log = logger.WithError(err)
		log.Error("Failed to set connection deadline")
		return false, "", errors.NewDependencyError(fmt.Sprintf("Failed to set connection deadline: %s", err.Error()))
	}
	
	// Send INSTREAM command to ClamAV
	if _, err := conn.Write(inStreamCommand); err != nil {
		log = logger.WithError(err)
		log.Error("Failed to send INSTREAM command")
		return false, "", errors.NewDependencyError(fmt.Sprintf("Failed to send INSTREAM command: %s", err.Error()))
	}
	
	// Read document content in chunks and send to ClamAV
	buf := make([]byte, chunkSize)
	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			log.Error("Context canceled during virus scan")
			return false, "", errors.Wrap(ctx.Err(), "Context canceled during virus scan")
		default:
			// Continue processing
		}
		
		n, readErr := reader.Read(buf)
		if n > 0 {
			// Send chunk size as 4-byte integer in network byte order (big-endian)
			sizeBytes := []byte{
				byte(n >> 24),
				byte(n >> 16),
				byte(n >> 8),
				byte(n),
			}
			
			// Send chunk size
			if _, err := conn.Write(sizeBytes); err != nil {
				log = logger.WithError(err)
				log.Error("Failed to send chunk size")
				return false, "", errors.NewDependencyError(fmt.Sprintf("Failed to send chunk size: %s", err.Error()))
			}
			
			// Send chunk data
			if _, err := conn.Write(buf[:n]); err != nil {
				log = logger.WithError(err)
				log.Error("Failed to send chunk data")
				return false, "", errors.NewDependencyError(fmt.Sprintf("Failed to send chunk data: %s", err.Error()))
			}
		}
		
		// If we've reached the end of the file or encountered an error, stop
		if readErr != nil {
			if readErr != io.EOF {
				log = logger.WithError(readErr)
				log.Error("Error reading document content")
				return false, "", errors.Wrap(readErr, "Error reading document content")
			}
			break
		}
	}
	
	// Send zero-length chunk to signal end of stream
	zeroSizeBytes := []byte{0, 0, 0, 0}
	if _, err := conn.Write(zeroSizeBytes); err != nil {
		log = logger.WithError(err)
		log.Error("Failed to send end of stream signal")
		return false, "", errors.NewDependencyError(fmt.Sprintf("Failed to send end of stream signal: %s", err.Error()))
	}
	
	// Read response from ClamAV
	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	response := scanner.Bytes()
	
	if err := scanner.Err(); err != nil {
		log = logger.WithError(err)
		log.Error("Failed to read scan response")
		return false, "", errors.NewDependencyError(fmt.Sprintf("Failed to read scan response: %s", err.Error()))
	}
	
	// Parse response to determine if document is clean
	if bytes.Contains(response, okResponse) {
		log.Info("Document scan completed: clean")
		return true, "", nil
	} else if bytes.Contains(response, foundResponse) {
		// Extract virus name from response
		parts := bytes.SplitN(response, []byte(": "), 2)
		virusName := ""
		if len(parts) > 1 {
			virusName = string(bytes.TrimSpace(parts[1]))
			virusName = string(bytes.TrimSuffix([]byte(virusName), []byte(" FOUND")))
		}
		
		log.Info("Document scan completed: virus found", "virus", virusName)
		return false, virusName, errors.NewSecurityError(fmt.Sprintf("Virus detected: %s", virusName))
	} else if bytes.Contains(response, errorResponse) {
		errorMsg := string(response)
		log.Error("Document scan error", "error", errorMsg)
		return false, "", errors.NewDependencyError(fmt.Sprintf("ClamAV scan error: %s", errorMsg))
	}
	
	// Unknown response
	log.Error("Document scan returned unknown response", "response", string(response))
	return false, "", errors.NewDependencyError(fmt.Sprintf("Unknown ClamAV response: %s", string(response)))
}

// Ping checks if ClamAV daemon is available
func (c *clamAVClient) Ping(ctx context.Context) error {
	log := logger.WithContext(ctx)
	log.Info("Pinging ClamAV service")
	
	// Establish connection to ClamAV daemon
	conn, err := net.DialTimeout("tcp", c.address, c.timeout)
	if err != nil {
		log = logger.WithError(err)
		log.Error("Failed to connect to ClamAV")
		return errors.NewDependencyError(fmt.Sprintf("Failed to connect to ClamAV: %s", err.Error()))
	}
	defer conn.Close()
	
	// Set deadline based on timeout
	if err := conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		log = logger.WithError(err)
		log.Error("Failed to set connection deadline")
		return errors.NewDependencyError(fmt.Sprintf("Failed to set connection deadline: %s", err.Error()))
	}
	
	// Send PING command to ClamAV
	if _, err := conn.Write([]byte("PING\n")); err != nil {
		log = logger.WithError(err)
		log.Error("Failed to send PING command")
		return errors.NewDependencyError(fmt.Sprintf("Failed to send PING command: %s", err.Error()))
	}
	
	// Read response from ClamAV
	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	response := scanner.Bytes()
	
	if err := scanner.Err(); err != nil {
		log = logger.WithError(err)
		log.Error("Failed to read ping response")
		return errors.NewDependencyError(fmt.Sprintf("Failed to read ping response: %s", err.Error()))
	}
	
	// Check if response is PONG
	if !bytes.Equal(response, []byte("PONG")) {
		log.Error("ClamAV ping failed", "response", string(response))
		return errors.NewDependencyError(fmt.Sprintf("Unexpected ClamAV ping response: %s", string(response)))
	}
	
	log.Info("ClamAV ping successful")
	return nil
}

// SetTimeout sets the timeout for ClamAV operations
func (c *clamAVClient) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}