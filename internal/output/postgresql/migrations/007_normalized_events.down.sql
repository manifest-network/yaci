BEGIN;

DROP TRIGGER IF EXISTS new_event_update ON api.events_raw;
DROP TRIGGER IF EXISTS new_transaction_events_raw ON api.transactions_raw;

DROP FUNCTION IF EXISTS api.update_event_main();
DROP FUNCTION IF EXISTS api.update_events_raw();
DROP FUNCTION IF EXISTS api.extract_event_msg_index(jsonb);

-- Indexes are dropped automatically with the table
DROP TABLE IF EXISTS api.events_main;
DROP TABLE IF EXISTS api.events_raw;

COMMIT;
