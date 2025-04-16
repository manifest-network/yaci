BEGIN;

CREATE INDEX IF NOT EXISTS idx_messages_main_type ON api.messages_main (type);

COMMIT;