package sqs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws" // v2.0.0+
	awsConfig "github.com/aws/aws-sdk-go-v2/config" // v2.0.0+
	"github.com/aws/aws-sdk-go-v2/credentials" // v2.0.0+
	"github.com/aws/aws-sdk-go-v2/service/sqs" // v2.0.0+
	"github.com/aws/aws-sdk-go-v2/service/sqs/types" // v2.0.0+

	"../../../../pkg/config"
	"../../../../pkg/errors"
	"../../../../pkg/logger"
)

// Default constants for SQS operations
const (
	defaultVisibilityTimeout   = 30 * time.Second
	defaultWaitTimeSeconds     = 20
	defaultMaxNumberOfMessages = 10
	defaultRetryAttempts       = 3
)

// SQSClient is a client for interacting with AWS SQS
type SQSClient struct {
	client *sqs.Client
	logger logger.Logger
}

// NewSQSClient creates a new SQS client with the provided configuration
func NewSQSClient(ctx context.Context, cfg config.SQSConfig) (*SQSClient, error) {
	// Create AWS configuration options
	var opts []func(*awsConfig.LoadOptions) error

	// Add custom endpoint if provided
	if cfg.Endpoint != "" {
		opts = append(opts, awsConfig.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:               cfg.Endpoint,
						SigningRegion:     cfg.Region,
						HostnameImmutable: true,
					}, nil
				},
			),
		))
	}

	// Add credentials if provided
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		opts = append(opts, awsConfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}

	// Load AWS configuration
	awsCfg, err := awsConfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, errors.NewDependencyError(fmt.Sprintf("failed to load AWS config: %v", err))
	}

	// Create SQS client
	sqsClient := sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		if cfg.Region != "" {
			o.Region = cfg.Region
		}
	})

	// Return new SQSClient
	return &SQSClient{
		client: sqsClient,
	}, nil
}

// GetQueueURL gets the URL for a queue by name
func GetQueueURL(ctx context.Context, client *SQSClient, queueName string) (string, error) {
	input := &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	}

	result, err := client.client.GetQueueUrl(ctx, input)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("failed to get queue URL for %s", queueName))
	}

	return *result.QueueUrl, nil
}

// SendMessage sends a message to an SQS queue
func (c *SQSClient) SendMessage(ctx context.Context, queueURL string, messageBody string, attributes map[string]string) (string, error) {
	log := logger.WithContext(ctx)

	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(messageBody),
	}

	// Add message attributes if provided
	if len(attributes) > 0 {
		msgAttrs := make(map[string]types.MessageAttributeValue)
		for k, v := range attributes {
			msgAttrs[k] = types.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(v),
			}
		}
		input.MessageAttributes = msgAttrs
	}

	result, err := c.client.SendMessage(ctx, input)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("failed to send message to queue %s", queueURL))
	}

	log.Info("Message sent to SQS queue", 
		"queue_url", queueURL, 
		"message_id", *result.MessageId)
	
	return *result.MessageId, nil
}

// ReceiveMessage receives messages from an SQS queue
func (c *SQSClient) ReceiveMessage(ctx context.Context, queueURL string, maxMessages int32, visibilityTimeout time.Duration) ([]types.Message, error) {
	log := logger.WithContext(ctx)

	// Set default values if not provided
	if maxMessages <= 0 {
		maxMessages = defaultMaxNumberOfMessages
	}
	if visibilityTimeout <= 0 {
		visibilityTimeout = defaultVisibilityTimeout
	}

	input := &sqs.ReceiveMessageInput{
		QueueUrl:              aws.String(queueURL),
		MaxNumberOfMessages:   maxMessages,
		VisibilityTimeout:     int32(visibilityTimeout.Seconds()),
		WaitTimeSeconds:       defaultWaitTimeSeconds,
		MessageAttributeNames: []string{"All"},
		AttributeNames:        []string{"All"},
	}

	result, err := c.client.ReceiveMessage(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to receive messages from queue %s", queueURL))
	}

	log.Info("Received messages from SQS queue", 
		"queue_url", queueURL, 
		"message_count", len(result.Messages))
	
	return result.Messages, nil
}

// DeleteMessage deletes a message from an SQS queue
func (c *SQSClient) DeleteMessage(ctx context.Context, queueURL string, receiptHandle string) error {
	log := logger.WithContext(ctx)

	input := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	}

	_, err := c.client.DeleteMessage(ctx, input)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to delete message from queue %s", queueURL))
	}

	log.Info("Deleted message from SQS queue", 
		"queue_url", queueURL, 
		"receipt_handle", receiptHandle)
	
	return nil
}

// ChangeMessageVisibility changes the visibility timeout of a message
func (c *SQSClient) ChangeMessageVisibility(ctx context.Context, queueURL string, receiptHandle string, visibilityTimeout int32) error {
	log := logger.WithContext(ctx)

	input := &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(queueURL),
		ReceiptHandle:     aws.String(receiptHandle),
		VisibilityTimeout: visibilityTimeout,
	}

	_, err := c.client.ChangeMessageVisibility(ctx, input)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to change message visibility in queue %s", queueURL))
	}

	log.Info("Changed message visibility in SQS queue", 
		"queue_url", queueURL, 
		"receipt_handle", receiptHandle, 
		"visibility_timeout", visibilityTimeout)
	
	return nil
}

// PurgeQueue purges all messages from an SQS queue
func (c *SQSClient) PurgeQueue(ctx context.Context, queueURL string) error {
	log := logger.WithContext(ctx)

	input := &sqs.PurgeQueueInput{
		QueueUrl: aws.String(queueURL),
	}

	_, err := c.client.PurgeQueue(ctx, input)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to purge queue %s", queueURL))
	}

	log.Info("Purged SQS queue", "queue_url", queueURL)
	return nil
}

// GetQueueAttributes gets attributes of an SQS queue
func (c *SQSClient) GetQueueAttributes(ctx context.Context, queueURL string, attributeNames []string) (map[string]string, error) {
	log := logger.WithContext(ctx)

	input := &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(queueURL),
		AttributeNames: attributeNames,
	}

	result, err := c.client.GetQueueAttributes(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to get queue attributes for %s", queueURL))
	}

	log.Info("Retrieved attributes from SQS queue", 
		"queue_url", queueURL, 
		"attribute_count", len(result.Attributes))
	
	return result.Attributes, nil
}