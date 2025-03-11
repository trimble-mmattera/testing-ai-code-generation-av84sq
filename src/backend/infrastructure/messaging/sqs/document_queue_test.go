package sqs

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types" // v2.0.0+
	"github.com/stretchr/testify/assert" // v1.8.0+
	"github.com/stretchr/testify/mock" // v1.8.0+
	"github.com/stretchr/testify/require" // v1.8.0+

	"../../../../domain/services"
	"../../../../pkg/config"
	pkgErrors "../../../../pkg/errors"
)

// mockSQSClient is a mock implementation of the SQSClient for testing
type mockSQSClient struct {
	mock.Mock
}

// SendMessage mock implementation of SendMessage
func (m *mockSQSClient) SendMessage(ctx context.Context, queueURL string, messageBody string, attributes map[string]string) (string, error) {
	return m.Called(ctx, queueURL, messageBody, attributes).Get(0).(string), m.Called(ctx, queueURL, messageBody, attributes).Error(1)
}

// ReceiveMessage mock implementation of ReceiveMessage
func (m *mockSQSClient) ReceiveMessage(ctx context.Context, queueURL string, maxMessages int32, visibilityTimeout time.Duration) ([]types.Message, error) {
	return m.Called(ctx, queueURL, maxMessages, visibilityTimeout).Get(0).([]types.Message), m.Called(ctx, queueURL, maxMessages, visibilityTimeout).Error(1)
}

// DeleteMessage mock implementation of DeleteMessage
func (m *mockSQSClient) DeleteMessage(ctx context.Context, queueURL string, receiptHandle string) error {
	return m.Called(ctx, queueURL, receiptHandle).Error(0)
}

const testQueueURL = "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue"
const testDLQURL = "https://sqs.us-east-1.amazonaws.com/123456789012/test-dlq"

// TestNewDocumentScanQueue tests the creation of a new DocumentScanQueue instance
func TestNewDocumentScanQueue(t *testing.T) {
	// Create a mock SQS client
	mockClient := new(mockSQSClient)
	
	// Create a test configuration
	cfg := &config.Config{
		SQS: config.SQSConfig{
			ScanQueueURL: testQueueURL,
		},
	}
	
	// Call NewDocumentScanQueue with the mock client and config
	queue, err := NewDocumentScanQueue(mockClient, cfg)
	
	// Assert that the returned queue is not nil
	assert.NotNil(t, queue)
	// Assert that no error is returned
	assert.NoError(t, err)
	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}

// TestNewDocumentScanQueue_Error tests error handling when creating a DocumentScanQueue fails
func TestNewDocumentScanQueue_Error(t *testing.T) {
	// Create a mock SQS client
	mockClient := new(mockSQSClient)
	
	// Create a test configuration with invalid queue URL
	cfg := &config.Config{
		SQS: config.SQSConfig{
			ScanQueueURL: "", // Empty URL to trigger error
		},
	}
	
	// Call NewDocumentScanQueue with the mock client and config
	queue, err := NewDocumentScanQueue(mockClient, cfg)
	
	// Assert that the returned queue is nil
	assert.Nil(t, queue)
	// Assert that an error is returned
	assert.Error(t, err)
	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}

// TestDocumentScanQueue_Enqueue tests enqueueing a document scan task
func TestDocumentScanQueue_Enqueue(t *testing.T) {
	// Create a mock SQS client
	mockClient := new(mockSQSClient)
	
	// Set up expectations for SendMessage
	mockClient.On("SendMessage", mock.Anything, testQueueURL, mock.Anything, mock.Anything).Return("message-id", nil)
	
	// Create a DocumentScanQueue with the mock client
	queue := &DocumentScanQueue{
		client:   mockClient,
		queueURL: testQueueURL,
		dlqURL:   testDLQURL,
	}
	
	// Create a test ScanTask
	task := services.ScanTask{
		DocumentID:  "doc-123",
		VersionID:   "ver-123",
		TenantID:    "tenant-123",
		StoragePath: "path/to/document",
		RetryCount:  0,
	}
	
	// Call Enqueue with the test task
	err := queue.Enqueue(context.Background(), task)
	
	// Assert that no error is returned
	assert.NoError(t, err)
	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}

// TestDocumentScanQueue_Enqueue_Error tests error handling when enqueueing a document scan task fails
func TestDocumentScanQueue_Enqueue_Error(t *testing.T) {
	// Create a mock SQS client
	mockClient := new(mockSQSClient)
	
	// Set up expectations for SendMessage to return an error
	mockClient.On("SendMessage", mock.Anything, testQueueURL, mock.Anything, mock.Anything).Return("", errors.New("send message error"))
	
	// Create a DocumentScanQueue with the mock client
	queue := &DocumentScanQueue{
		client:   mockClient,
		queueURL: testQueueURL,
		dlqURL:   testDLQURL,
	}
	
	// Create a test ScanTask
	task := services.ScanTask{
		DocumentID:  "doc-123",
		VersionID:   "ver-123",
		TenantID:    "tenant-123",
		StoragePath: "path/to/document",
		RetryCount:  0,
	}
	
	// Call Enqueue with the test task
	err := queue.Enqueue(context.Background(), task)
	
	// Assert that an error is returned
	assert.Error(t, err)
	// Assert that the error is a dependency error
	assert.True(t, pkgErrors.IsDependencyError(err))
	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}

// TestDocumentScanQueue_Dequeue tests dequeueing a document scan task
func TestDocumentScanQueue_Dequeue(t *testing.T) {
	// Create a mock SQS client
	mockClient := new(mockSQSClient)
	
	// Create a test ScanTask and marshal it to JSON
	testTask := services.ScanTask{
		DocumentID:  "doc-123",
		VersionID:   "ver-123",
		TenantID:    "tenant-123",
		StoragePath: "path/to/document",
		RetryCount:  0,
	}
	
	taskJSON, err := json.Marshal(testTask)
	require.NoError(t, err)
	
	// Create message with task
	bodyStr := string(taskJSON)
	receiptStr := "receipt-handle-123"
	
	// Set up expectations for ReceiveMessage to return a message with the task
	mockClient.On("ReceiveMessage", mock.Anything, testQueueURL, int32(1), mock.Anything).Return([]types.Message{
		{
			Body:          &bodyStr,
			ReceiptHandle: &receiptStr,
		},
	}, nil)
	
	// Set up expectations for DeleteMessage
	mockClient.On("DeleteMessage", mock.Anything, testQueueURL, receiptStr).Return(nil)
	
	// Create a DocumentScanQueue with the mock client
	queue := &DocumentScanQueue{
		client:   mockClient,
		queueURL: testQueueURL,
		dlqURL:   testDLQURL,
	}
	
	// Call Dequeue
	task, err := queue.Dequeue(context.Background())
	
	// Assert that the returned task is not nil
	assert.NotNil(t, task)
	// Assert that the task properties match the test task
	assert.Equal(t, testTask.DocumentID, task.DocumentID)
	assert.Equal(t, testTask.VersionID, task.VersionID)
	assert.Equal(t, testTask.TenantID, task.TenantID)
	assert.Equal(t, testTask.StoragePath, task.StoragePath)
	assert.Equal(t, testTask.RetryCount, task.RetryCount)
	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}

// TestDocumentScanQueue_Dequeue_NoMessages tests dequeueing when no messages are available
func TestDocumentScanQueue_Dequeue_NoMessages(t *testing.T) {
	// Create a mock SQS client
	mockClient := new(mockSQSClient)
	
	// Set up expectations for ReceiveMessage to return empty messages
	mockClient.On("ReceiveMessage", mock.Anything, testQueueURL, int32(1), mock.Anything).Return([]types.Message{}, nil)
	
	// Create a DocumentScanQueue with the mock client
	queue := &DocumentScanQueue{
		client:   mockClient,
		queueURL: testQueueURL,
		dlqURL:   testDLQURL,
	}
	
	// Call Dequeue
	task, err := queue.Dequeue(context.Background())
	
	// Assert that the returned task is nil
	assert.Nil(t, task)
	// Assert that no error is returned
	assert.NoError(t, err)
	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}

// TestDocumentScanQueue_Dequeue_ReceiveError tests error handling when receiving messages fails
func TestDocumentScanQueue_Dequeue_ReceiveError(t *testing.T) {
	// Create a mock SQS client
	mockClient := new(mockSQSClient)
	
	// Set up expectations for ReceiveMessage to return an error
	mockClient.On("ReceiveMessage", mock.Anything, testQueueURL, int32(1), mock.Anything).Return([]types.Message{}, errors.New("receive error"))
	
	// Create a DocumentScanQueue with the mock client
	queue := &DocumentScanQueue{
		client:   mockClient,
		queueURL: testQueueURL,
		dlqURL:   testDLQURL,
	}
	
	// Call Dequeue
	task, err := queue.Dequeue(context.Background())
	
	// Assert that the returned task is nil
	assert.Nil(t, task)
	// Assert that an error is returned
	assert.Error(t, err)
	// Assert that the error is a dependency error
	assert.True(t, pkgErrors.IsDependencyError(err))
	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}

// TestDocumentScanQueue_Dequeue_UnmarshalError tests error handling when unmarshaling a message fails
func TestDocumentScanQueue_Dequeue_UnmarshalError(t *testing.T) {
	// Create a mock SQS client
	mockClient := new(mockSQSClient)
	
	// Set up expectations for ReceiveMessage to return a message with invalid JSON
	bodyStr := "invalid-json"
	receiptStr := "receipt-handle-123"
	mockClient.On("ReceiveMessage", mock.Anything, testQueueURL, int32(1), mock.Anything).Return([]types.Message{
		{
			Body:          &bodyStr,
			ReceiptHandle: &receiptStr,
		},
	}, nil)
	
	// Set up expectations for DeleteMessage
	mockClient.On("DeleteMessage", mock.Anything, testQueueURL, receiptStr).Return(nil)
	
	// Create a DocumentScanQueue with the mock client
	queue := &DocumentScanQueue{
		client:   mockClient,
		queueURL: testQueueURL,
		dlqURL:   testDLQURL,
	}
	
	// Call Dequeue
	task, err := queue.Dequeue(context.Background())
	
	// Assert that the returned task is nil
	assert.Nil(t, task)
	// Assert that an error is returned
	assert.Error(t, err)
	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}

// TestDocumentScanQueue_Dequeue_DeleteError tests error handling when deleting a message fails
func TestDocumentScanQueue_Dequeue_DeleteError(t *testing.T) {
	// Create a mock SQS client
	mockClient := new(mockSQSClient)
	
	// Create a test ScanTask and marshal it to JSON
	testTask := services.ScanTask{
		DocumentID:  "doc-123",
		VersionID:   "ver-123",
		TenantID:    "tenant-123",
		StoragePath: "path/to/document",
		RetryCount:  0,
	}
	
	taskJSON, err := json.Marshal(testTask)
	require.NoError(t, err)
	
	// Set up expectations for ReceiveMessage to return a message with the task
	bodyStr := string(taskJSON)
	receiptStr := "receipt-handle-123"
	mockClient.On("ReceiveMessage", mock.Anything, testQueueURL, int32(1), mock.Anything).Return([]types.Message{
		{
			Body:          &bodyStr,
			ReceiptHandle: &receiptStr,
		},
	}, nil)
	
	// Set up expectations for DeleteMessage to return an error
	mockClient.On("DeleteMessage", mock.Anything, testQueueURL, receiptStr).Return(errors.New("delete error"))
	
	// Create a DocumentScanQueue with the mock client
	queue := &DocumentScanQueue{
		client:   mockClient,
		queueURL: testQueueURL,
		dlqURL:   testDLQURL,
	}
	
	// Call Dequeue
	task, err := queue.Dequeue(context.Background())
	
	// Assert that the returned task is nil
	assert.Nil(t, task)
	// Assert that an error is returned
	assert.Error(t, err)
	// Assert that the error is a dependency error
	assert.True(t, pkgErrors.IsDependencyError(err))
	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}

// TestDocumentScanQueue_DequeueBatch tests dequeueing a batch of document scan tasks
func TestDocumentScanQueue_DequeueBatch(t *testing.T) {
	// Create a mock SQS client
	mockClient := new(mockSQSClient)
	
	// Create multiple test ScanTasks and marshal them to JSON
	testTasks := []services.ScanTask{
		{
			DocumentID:  "doc-123",
			VersionID:   "ver-123",
			TenantID:    "tenant-123",
			StoragePath: "path/to/document1",
			RetryCount:  0,
		},
		{
			DocumentID:  "doc-456",
			VersionID:   "ver-456",
			TenantID:    "tenant-123",
			StoragePath: "path/to/document2",
			RetryCount:  1,
		},
	}
	
	// Create messages for the batch
	messages := make([]types.Message, len(testTasks))
	for i, task := range testTasks {
		taskJSON, err := json.Marshal(task)
		require.NoError(t, err)
		
		bodyStr := string(taskJSON)
		receiptStr := "receipt-handle-" + task.DocumentID
		
		messages[i] = types.Message{
			Body:          &bodyStr,
			ReceiptHandle: &receiptStr,
		}
		
		// Set up expectations for DeleteMessage for each message
		mockClient.On("DeleteMessage", mock.Anything, testQueueURL, receiptStr).Return(nil)
	}
	
	// Set up expectations for ReceiveMessage to return messages with the tasks
	mockClient.On("ReceiveMessage", mock.Anything, testQueueURL, int32(2), mock.Anything).Return(messages, nil)
	
	// Create a DocumentScanQueue with the mock client
	queue := &DocumentScanQueue{
		client:   mockClient,
		queueURL: testQueueURL,
		dlqURL:   testDLQURL,
	}
	
	// Call DequeueBatch with a batch size
	tasks, err := queue.DequeueBatch(context.Background(), 2)
	
	// Assert that the returned tasks slice is not nil
	assert.NotNil(t, tasks)
	// Assert that the number of tasks matches the expected count
	assert.Equal(t, len(testTasks), len(tasks))
	// Assert that each task's properties match the test tasks
	for i, task := range tasks {
		assert.Equal(t, testTasks[i].DocumentID, task.DocumentID)
		assert.Equal(t, testTasks[i].VersionID, task.VersionID)
		assert.Equal(t, testTasks[i].TenantID, task.TenantID)
		assert.Equal(t, testTasks[i].StoragePath, task.StoragePath)
		assert.Equal(t, testTasks[i].RetryCount, task.RetryCount)
	}
	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}

// TestDocumentScanQueue_DequeueBatch_MaxBatchSize tests that DequeueBatch respects the maximum batch size
func TestDocumentScanQueue_DequeueBatch_MaxBatchSize(t *testing.T) {
	// Create a mock SQS client
	mockClient := new(mockSQSClient)
	
	// Set up expectations for ReceiveMessage with maxBatchSize
	const maxBatchSize = 10
	mockClient.On("ReceiveMessage", mock.Anything, testQueueURL, int32(maxBatchSize), mock.Anything).Return([]types.Message{}, nil)
	
	// Create a DocumentScanQueue with the mock client
	queue := &DocumentScanQueue{
		client:   mockClient,
		queueURL: testQueueURL,
		dlqURL:   testDLQURL,
		maxBatchSize: maxBatchSize,
	}
	
	// Call DequeueBatch with a batch size larger than maxBatchSize
	_, err := queue.DequeueBatch(context.Background(), 20)
	
	// Assert that no error is returned
	assert.NoError(t, err)
	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}

// TestDocumentScanQueue_Complete tests completing a document scan task
func TestDocumentScanQueue_Complete(t *testing.T) {
	// Create a mock SQS client
	mockClient := new(mockSQSClient)
	
	// Create a DocumentScanQueue with the mock client
	queue := &DocumentScanQueue{
		client:   mockClient,
		queueURL: testQueueURL,
		dlqURL:   testDLQURL,
	}
	
	// Create a test ScanTask
	task := services.ScanTask{
		DocumentID:  "doc-123",
		VersionID:   "ver-123",
		TenantID:    "tenant-123",
		StoragePath: "path/to/document",
		RetryCount:  0,
	}
	
	// Call Complete with the test task
	err := queue.Complete(context.Background(), task)
	
	// Assert that no error is returned (Complete is a no-op in SQS implementation)
	assert.NoError(t, err)
	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}

// TestDocumentScanQueue_Retry tests retrying a document scan task
func TestDocumentScanQueue_Retry(t *testing.T) {
	// Create a mock SQS client
	mockClient := new(mockSQSClient)
	
	// Set up expectations for SendMessage
	mockClient.On("SendMessage", mock.Anything, testQueueURL, mock.Anything, mock.Anything).Return("message-id", nil)
	
	// Create a DocumentScanQueue with the mock client
	queue := &DocumentScanQueue{
		client:   mockClient,
		queueURL: testQueueURL,
		dlqURL:   testDLQURL,
	}
	
	// Create a test ScanTask with RetryCount = 0
	task := services.ScanTask{
		DocumentID:  "doc-123",
		VersionID:   "ver-123",
		TenantID:    "tenant-123",
		StoragePath: "path/to/document",
		RetryCount:  0,
	}
	
	// Call Retry with the test task
	err := queue.Retry(context.Background(), task)
	
	// Assert that no error is returned
	assert.NoError(t, err)
	// Assert that the task's RetryCount was incremented
	assert.Equal(t, 1, task.RetryCount)
	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}

// TestDocumentScanQueue_Retry_Error tests error handling when retrying a document scan task fails
func TestDocumentScanQueue_Retry_Error(t *testing.T) {
	// Create a mock SQS client
	mockClient := new(mockSQSClient)
	
	// Set up expectations for SendMessage to return an error
	mockClient.On("SendMessage", mock.Anything, testQueueURL, mock.Anything, mock.Anything).Return("", errors.New("send message error"))
	
	// Create a DocumentScanQueue with the mock client
	queue := &DocumentScanQueue{
		client:   mockClient,
		queueURL: testQueueURL,
		dlqURL:   testDLQURL,
	}
	
	// Create a test ScanTask
	task := services.ScanTask{
		DocumentID:  "doc-123",
		VersionID:   "ver-123",
		TenantID:    "tenant-123",
		StoragePath: "path/to/document",
		RetryCount:  0,
	}
	
	// Call Retry with the test task
	err := queue.Retry(context.Background(), task)
	
	// Assert that an error is returned
	assert.Error(t, err)
	// Assert that the error is a dependency error
	assert.True(t, pkgErrors.IsDependencyError(err))
	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}

// TestDocumentScanQueue_DeadLetter tests moving a document scan task to the dead letter queue
func TestDocumentScanQueue_DeadLetter(t *testing.T) {
	// Create a mock SQS client
	mockClient := new(mockSQSClient)
	
	// Set up expectations for SendMessage to the DLQ
	mockClient.On("SendMessage", mock.Anything, testDLQURL, mock.Anything, mock.Anything).Return("message-id", nil)
	
	// Create a DocumentScanQueue with the mock client
	queue := &DocumentScanQueue{
		client:   mockClient,
		queueURL: testQueueURL,
		dlqURL:   testDLQURL,
	}
	
	// Create a test ScanTask
	task := services.ScanTask{
		DocumentID:  "doc-123",
		VersionID:   "ver-123",
		TenantID:    "tenant-123",
		StoragePath: "path/to/document",
		RetryCount:  3,
	}
	
	// Call DeadLetter with the test task and a reason
	err := queue.DeadLetter(context.Background(), task, "Maximum retries exceeded")
	
	// Assert that no error is returned
	assert.NoError(t, err)
	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}

// TestDocumentScanQueue_DeadLetter_Error tests error handling when moving a task to the dead letter queue fails
func TestDocumentScanQueue_DeadLetter_Error(t *testing.T) {
	// Create a mock SQS client
	mockClient := new(mockSQSClient)
	
	// Set up expectations for SendMessage to return an error
	mockClient.On("SendMessage", mock.Anything, testDLQURL, mock.Anything, mock.Anything).Return("", errors.New("send message error"))
	
	// Create a DocumentScanQueue with the mock client
	queue := &DocumentScanQueue{
		client:   mockClient,
		queueURL: testQueueURL,
		dlqURL:   testDLQURL,
	}
	
	// Create a test ScanTask
	task := services.ScanTask{
		DocumentID:  "doc-123",
		VersionID:   "ver-123",
		TenantID:    "tenant-123",
		StoragePath: "path/to/document",
		RetryCount:  3,
	}
	
	// Call DeadLetter with the test task and a reason
	err := queue.DeadLetter(context.Background(), task, "Maximum retries exceeded")
	
	// Assert that an error is returned
	assert.Error(t, err)
	// Assert that the error is a dependency error
	assert.True(t, pkgErrors.IsDependencyError(err))
	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}

// TestDocumentScanQueue_MoveToDeadLetterQueue tests the helper method for moving a failed task to the dead letter queue
func TestDocumentScanQueue_MoveToDeadLetterQueue(t *testing.T) {
	// Create a mock SQS client
	mockClient := new(mockSQSClient)
	
	// Set up expectations for SendMessage to the DLQ
	mockClient.On("SendMessage", mock.Anything, testDLQURL, mock.Anything, mock.Anything).Return("message-id", nil)
	
	// Create a DocumentScanQueue with the mock client
	queue := &DocumentScanQueue{
		client:   mockClient,
		queueURL: testQueueURL,
		dlqURL:   testDLQURL,
	}
	
	// Create a test ScanTask
	task := services.ScanTask{
		DocumentID:  "doc-123",
		VersionID:   "ver-123",
		TenantID:    "tenant-123",
		StoragePath: "path/to/document",
		RetryCount:  3,
	}
	
	// Create a test error
	testErr := errors.New("processing error")
	
	// Call MoveToDeadLetterQueue with the test task and error
	err := queue.MoveToDeadLetterQueue(context.Background(), task, testErr)
	
	// Assert that no error is returned
	assert.NoError(t, err)
	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}