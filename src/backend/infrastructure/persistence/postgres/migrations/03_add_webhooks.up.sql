-- Create webhooks table to store webhook subscription information
CREATE TABLE webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    event_types TEXT[] NOT NULL,
    secret_key TEXT NOT NULL,
    description TEXT NULL,
    status VARCHAR(50) NOT NULL,
    failure_count INTEGER NOT NULL DEFAULT 0,
    last_failure_time TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create webhook_deliveries table to store delivery attempts and results
CREATE TABLE webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    webhook_id UUID NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 1,
    response_status INTEGER NULL,
    response_body TEXT NULL,
    error_message TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP NULL
);

-- Create indexes for webhooks table
CREATE INDEX webhooks_tenant_id_idx ON webhooks(tenant_id);
CREATE INDEX webhooks_status_idx ON webhooks(status);
CREATE INDEX webhooks_event_types_idx ON webhooks USING GIN(event_types);

-- Create indexes for webhook_deliveries table
CREATE INDEX webhook_deliveries_webhook_id_idx ON webhook_deliveries(webhook_id);
CREATE INDEX webhook_deliveries_event_id_idx ON webhook_deliveries(event_id);
CREATE INDEX webhook_deliveries_status_idx ON webhook_deliveries(status);
CREATE INDEX webhook_deliveries_created_at_idx ON webhook_deliveries(created_at);

-- Create function and trigger for automatically updating updated_at on webhooks
CREATE OR REPLACE FUNCTION update_webhook_updated_at()
RETURNS trigger AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER webhook_updated_at
BEFORE UPDATE ON webhooks
FOR EACH ROW
EXECUTE FUNCTION update_webhook_updated_at();

-- Create function and trigger for automatically updating updated_at on webhook_deliveries
CREATE OR REPLACE FUNCTION update_webhook_delivery_updated_at()
RETURNS trigger AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER webhook_delivery_updated_at
BEFORE UPDATE ON webhook_deliveries
FOR EACH ROW
EXECUTE FUNCTION update_webhook_delivery_updated_at();

-- Add table comments for documentation
COMMENT ON TABLE webhooks IS 'Stores webhook subscriptions for event notifications';
COMMENT ON TABLE webhook_deliveries IS 'Stores webhook delivery attempts and results';

-- Add column comments for webhooks table
COMMENT ON COLUMN webhooks.id IS 'Unique identifier for the webhook';
COMMENT ON COLUMN webhooks.tenant_id IS 'Reference to the tenant that owns this webhook';
COMMENT ON COLUMN webhooks.url IS 'The URL to which webhook events will be sent';
COMMENT ON COLUMN webhooks.event_types IS 'Array of event types this webhook subscribes to';
COMMENT ON COLUMN webhooks.secret_key IS 'Secret key used for generating webhook payload signatures';
COMMENT ON COLUMN webhooks.description IS 'Optional description of the webhook purpose';
COMMENT ON COLUMN webhooks.status IS 'Current status of the webhook (active, inactive)';
COMMENT ON COLUMN webhooks.failure_count IS 'Count of consecutive delivery failures';
COMMENT ON COLUMN webhooks.last_failure_time IS 'Timestamp of the last delivery failure';

-- Add column comments for webhook_deliveries table
COMMENT ON COLUMN webhook_deliveries.id IS 'Unique identifier for the delivery attempt';
COMMENT ON COLUMN webhook_deliveries.webhook_id IS 'Reference to the webhook being delivered';
COMMENT ON COLUMN webhook_deliveries.event_id IS 'Reference to the event being delivered';
COMMENT ON COLUMN webhook_deliveries.status IS 'Current status of the delivery (pending, success, failed)';
COMMENT ON COLUMN webhook_deliveries.attempt_count IS 'Number of delivery attempts made';
COMMENT ON COLUMN webhook_deliveries.response_status IS 'HTTP status code from the webhook endpoint';
COMMENT ON COLUMN webhook_deliveries.response_body IS 'Response body from the webhook endpoint';
COMMENT ON COLUMN webhook_deliveries.error_message IS 'Error message if delivery failed';
COMMENT ON COLUMN webhook_deliveries.completed_at IS 'Timestamp when delivery was completed (success or final failure)';