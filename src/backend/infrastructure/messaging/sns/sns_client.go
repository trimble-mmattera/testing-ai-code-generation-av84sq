// Package sns provides a client for AWS Simple Notification Service (SNS)
// that supports the event-driven architecture of the Document Management Platform.
package sns

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws" // v2.0.0+
	"github.com/aws/aws-sdk-go-v2/config" // v2.0.0+
	"github.com/aws/aws-sdk-go-v2/credentials" // v2.0.0+
	"github.com/aws/aws-sdk-go-v2/service/sns" // v2.0.0+

	"../../../pkg/config"
	"../../../pkg/errors"
	"../../../pkg/logger"
)

// SNSClientInterface defines the contract for SNS operations
type SNSClientInterface interface {
	// Publish publishes a message to an SNS topic
	Publish(ctx context.Context, topicName string, message interface{}) (string, error)
}

// SNSClient implements SNSClientInterface using AWS SDK
type SNSClient struct {
	client           *sns.Client
	logger           *logger.Logger
	topicNameToARNMap map[string]string
}

// NewSNSClient creates a new SNSClient with the provided configuration
func NewSNSClient(cfg *config.SNSConfig) (SNSClientInterface, error) {
	if cfg == nil {
		return nil, errors.NewValidationError("SNS configuration is required")
	}

	// Create AWS configuration options
	options := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	// Add credentials if provided
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		options = append(options, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}

	// Add custom endpoint if provided
	if cfg.Endpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               cfg.Endpoint,
				SigningRegion:     cfg.Region,
				HostnameImmutable: true,
			}, nil
		})
		options = append(options, config.WithEndpointResolverWithOptions(customResolver))
	}

	// Load the AWS configuration
	awsCfg, err := config.LoadDefaultConfig(context.Background(), options...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load AWS configuration")
	}

	// Create SNS client with the AWS configuration
	snsClient := sns.NewFromConfig(awsCfg)

	// Initialize topic mapping
	topicMap := map[string]string{
		"document": cfg.DocumentTopicARN,
		"event":    cfg.EventTopicARN,
	}

	return &SNSClient{
		client:           snsClient,
		logger:           logger.WithField("component", "sns_client"),
		topicNameToARNMap: topicMap,
	}, nil
}

// Publish publishes a message to an SNS topic
func (c *SNSClient) Publish(ctx context.Context, topicName string, message interface{}) (string, error) {
	// Get logger with context
	log := logger.WithContext(ctx)
	log = logger.WithField("topic", topicName)

	// Get topic ARN
	topicARN, err := c.getTopicARN(topicName)
	if err != nil {
		logger.WithError(err).Error("Failed to get topic ARN")
		return "", err
	}

	// Marshal message to JSON
	messageJSON, err := json.Marshal(message)
	if err != nil {
		logger.WithError(err).Error("Failed to marshal message to JSON")
		return "", errors.Wrap(err, "failed to marshal message to JSON")
	}

	// Create publish input
	input := &sns.PublishInput{
		TopicArn: aws.String(topicARN),
		Message:  aws.String(string(messageJSON)),
	}

	// Publish message
	result, err := c.client.Publish(ctx, input)
	if err != nil {
		logger.WithError(err).Error("Failed to publish message to SNS")
		return "", errors.Wrap(err, "failed to publish message to SNS")
	}

	// Log successful publish
	logger.Info("Successfully published message to SNS", "messageId", *result.MessageId)

	return *result.MessageId, nil
}

// getTopicARN gets the ARN for a topic name
func (c *SNSClient) getTopicARN(topicName string) (string, error) {
	arn, ok := c.topicNameToARNMap[topicName]
	if !ok {
		return "", errors.NewValidationError(fmt.Sprintf("invalid topic name: %s", topicName))
	}
	return arn, nil
}