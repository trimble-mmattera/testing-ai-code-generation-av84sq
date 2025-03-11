// Package sqs provides AWS SQS implementations for queue interfaces in the Document Management Platform.
package sqs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types" // v2.0.0+

	"../../../../domain/services"
	"../../../../pkg/config"
	"../../../../pkg/errors"
	"../../../../pkg/logger"
)

const queueNameSuffix = "-document-scan-tasks"
const dlqNameSuffix = "-document-scan-tasks-dlq"
const maxBatchSize = 10

// DocumentScanQueue implements the services.ScanQueue interface using AWS SQS
type DocumentScanQueue struct {
	sqsClient *SQSClient
	queueURL  string
	dlqURL    string
	logger    logger.Logger
}

// NewDocumentScanQueue creates a new DocumentScanQueue instance that implements the ScanQueue interface
func NewDocumentScanQueue(ctx context.Context, sqsClient *SQSClient, cfg config.Config) (services.ScanQueue, error) {
	// Validate that sqsClient is not nil
	if sqsClient == nil {
		return nil, errors.NewValidationError("sqsClient cannot be nil")
	}

	// Get the queue name using tenant prefix from config
	tenantPrefix := cfg.Env // Using env as tenant prefix
	queueName := tenantPrefix + queueNameSuffix

	// Get the DLQ name using tenant prefix from config
	dlqName := tenantPrefix + dlqNameSuffix

	// Get queue URL using GetQueueURL function
	queueURL, err := sqsClient.GetQueueURL(ctx, queueName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get queue URL")
	}

	// Get DLQ URL using GetQueueURL function
	dlqURL, err := sqsClient.GetQueueURL(ctx, dlqName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get DLQ URL")
	}

	// Initialize and return new DocumentScanQueue with the SQS client and queue URLs
	return &DocumentScanQueue{
		sqsClient: sqsClient,
		queueURL:  queueURL,
		dlqURL:    dlqURL,
		logger:    logger.WithField("component", "DocumentScanQueue"),
	}, nil
}

// Enqueue adds a document to the scanning queue
func (q *DocumentScanQueue) Enqueue(ctx context.Context, task services.ScanTask) error {
	log := logger.WithContext(ctx)
	
	// Marshal the scan task to JSON
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return errors.Wrap(err, "failed to marshal scan task to JSON")
	}
	
	// Send the JSON message to the SQS queue using sqsClient.SendMessage
	err = q.sqsClient.SendMessage(ctx, q.queueURL, string(taskJSON))
	if err != nil {
		return errors.NewDependencyError(fmt.Sprintf("failed to enqueue scan task: %v", err))
	}
	
	log.Info("Document scan task enqueued successfully", 
		"document_id", task.DocumentID,
		"tenant_id", task.TenantID)
	
	return nil
}

// Dequeue retrieves the next document to scan from the queue
func (q *DocumentScanQueue) Dequeue(ctx context.Context) (*services.ScanTask, error) {
	log := logger.WithContext(ctx)
	
	// Receive a single message from the SQS queue using sqsClient.ReceiveMessage
	messages, err := q.sqsClient.ReceiveMessage(ctx, q.queueURL, 1)
	if err != nil {
		return nil, errors.NewDependencyError(fmt.Sprintf("failed to dequeue scan task: %v", err))
	}
	
	// If no messages are received, return nil, nil
	if len(messages) == 0 {
		return nil, nil
	}
	
	// Unmarshal the message body to a ScanTask
	var task services.ScanTask
	err = json.Unmarshal([]byte(*messages[0].Body), &task)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal scan task from JSON")
	}
	
	// Delete the message from the queue using sqsClient.DeleteMessage
	err = q.sqsClient.DeleteMessage(ctx, q.queueURL, *messages[0].ReceiptHandle)
	if err != nil {
		return nil, errors.NewDependencyError(fmt.Sprintf("failed to delete message from queue: %v", err))
	}
	
	log.Info("Document scan task dequeued successfully", 
		"document_id", task.DocumentID,
		"tenant_id", task.TenantID)
	
	return &task, nil
}

// DequeueBatch retrieves multiple documents to scan from the queue
func (q *DocumentScanQueue) DequeueBatch(ctx context.Context, batchSize int) ([]services.ScanTask, error) {
	log := logger.WithContext(ctx)
	
	// If batchSize > maxBatchSize, limit to maxBatchSize
	if batchSize > maxBatchSize {
		batchSize = maxBatchSize
	}
	
	// Receive messages from the SQS queue using sqsClient.ReceiveMessage
	messages, err := q.sqsClient.ReceiveMessage(ctx, q.queueURL, batchSize)
	if err != nil {
		return nil, errors.NewDependencyError(fmt.Sprintf("failed to dequeue scan tasks batch: %v", err))
	}
	
	// If no messages are received, return empty slice, nil
	if len(messages) == 0 {
		return []services.ScanTask{}, nil
	}
	
	// Initialize slice to hold scan tasks
	tasks := make([]services.ScanTask, 0, len(messages))
	
	// For each message:
	for _, message := range messages {
		// Unmarshal the message body to a ScanTask
		var task services.ScanTask
		err = json.Unmarshal([]byte(*message.Body), &task)
		if err != nil {
			log.Error("Failed to unmarshal scan task from JSON", 
				"error", err,
				"message_body", *message.Body)
			continue
		}
		
		// Delete the message from the queue using sqsClient.DeleteMessage
		err = q.sqsClient.DeleteMessage(ctx, q.queueURL, *message.ReceiptHandle)
		if err != nil {
			log.Error("Failed to delete message from queue", 
				"error", err,
				"receipt_handle", *message.ReceiptHandle)
			continue
		}
		
		// Add the task to the slice
		tasks = append(tasks, task)
	}
	
	log.Info("Document scan tasks batch dequeued successfully", 
		"count", len(tasks))
	
	return tasks, nil
}

// Complete marks a scan task as completed and removes it from the queue
func (q *DocumentScanQueue) Complete(ctx context.Context, task services.ScanTask) error {
	// This is a no-op in the SQS implementation as messages are deleted when dequeued
	return nil
}

// Retry requeues a scan task for retry after a failure
func (q *DocumentScanQueue) Retry(ctx context.Context, task services.ScanTask) error {
	log := logger.WithContext(ctx)
	
	// Increment the RetryCount of the task
	task.RetryCount++
	
	// Marshal the updated task to JSON
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return errors.Wrap(err, "failed to marshal scan task to JSON")
	}
	
	// Send the JSON message to the SQS queue using sqsClient.SendMessage
	err = q.sqsClient.SendMessage(ctx, q.queueURL, string(taskJSON))
	if err != nil {
		return errors.NewDependencyError(fmt.Sprintf("failed to requeue scan task for retry: %v", err))
	}
	
	log.Info("Document scan task requeued for retry", 
		"document_id", task.DocumentID,
		"tenant_id", task.TenantID,
		"retry_count", task.RetryCount)
	
	return nil
}

// DeadLetter moves a scan task to the dead letter queue after maximum retries
func (q *DocumentScanQueue) DeadLetter(ctx context.Context, task services.ScanTask, reason string) error {
	log := logger.WithContext(ctx)
	
	// Create a message with the task and failure reason
	message := struct {
		Task   services.ScanTask `json:"task"`
		Reason string            `json:"reason"`
	}{
		Task:   task,
		Reason: reason,
	}
	
	// Marshal the message to JSON
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "failed to marshal dead letter message to JSON")
	}
	
	// Send the JSON message to the DLQ using sqsClient.SendMessage
	err = q.sqsClient.SendMessage(ctx, q.dlqURL, string(messageJSON))
	if err != nil {
		return errors.NewDependencyError(fmt.Sprintf("failed to move scan task to dead letter queue: %v", err))
	}
	
	log.Info("Document scan task moved to dead letter queue", 
		"document_id", task.DocumentID,
		"tenant_id", task.TenantID,
		"reason", reason)
	
	return nil
}

// MoveToDeadLetterQueue is a helper method to move a failed task to the dead letter queue
func (q *DocumentScanQueue) MoveToDeadLetterQueue(ctx context.Context, task services.ScanTask, err error) error {
	log := logger.WithContext(ctx).WithError(err)
	
	// Create a message with the task and error details
	message := struct {
		Task  services.ScanTask `json:"task"`
		Error string            `json:"error"`
	}{
		Task:  task,
		Error: err.Error(),
	}
	
	// Marshal the message to JSON
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "failed to marshal dead letter message to JSON")
	}
	
	// Send the JSON message to the DLQ using sqsClient.SendMessage
	err = q.sqsClient.SendMessage(ctx, q.dlqURL, string(messageJSON))
	if err != nil {
		return errors.NewDependencyError(fmt.Sprintf("failed to move scan task to dead letter queue: %v", err))
	}
	
	log.Info("Document scan task moved to dead letter queue due to error", 
		"document_id", task.DocumentID,
		"tenant_id", task.TenantID)
	
	return nil
}