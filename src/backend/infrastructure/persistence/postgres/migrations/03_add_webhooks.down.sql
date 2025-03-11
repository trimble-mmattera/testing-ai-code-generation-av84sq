-- Drop triggers that update timestamps
DROP TRIGGER webhook_updated_at ON webhooks;
DROP TRIGGER webhook_delivery_updated_at ON webhook_deliveries;

-- Drop functions that were used by the triggers
DROP FUNCTION update_webhook_updated_at();
DROP FUNCTION update_webhook_delivery_updated_at();

-- Drop indexes from webhook_deliveries table
DROP INDEX webhook_deliveries_created_at_idx;
DROP INDEX webhook_deliveries_status_idx;
DROP INDEX webhook_deliveries_event_id_idx;
DROP INDEX webhook_deliveries_webhook_id_idx;

-- Drop indexes from webhooks table
DROP INDEX webhooks_event_types_idx;
DROP INDEX webhooks_status_idx;
DROP INDEX webhooks_tenant_id_idx;

-- Drop webhook_deliveries table first (contains foreign key to webhooks)
DROP TABLE webhook_deliveries;

-- Drop webhooks table
DROP TABLE webhooks;