BEGIN;

DROP INDEX IF EXISTS api.message_main_mentions_idx;
DROP INDEX IF EXISTS api.message_main_sender_idx;

REVOKE EXECUTE ON FUNCTION api.get_messages_for_address(TEXT) FROM web_anon;
REVOKE SELECT ON api.messages_main FROM web_anon;
REVOKE SELECT ON api.messages_raw FROM web_anon;
REVOKE SELECT ON api.transactions_raw FROM web_anon;
REVOKE SELECT ON api.transactions_main FROM web_anon;
REVOKE SELECT ON api.blocks_raw FROM web_anon;

DROP FUNCTION IF EXISTS api.get_messages_for_address(TEXT);

DROP TRIGGER IF EXISTS new_message_update ON api.messages_raw;
DROP TRIGGER IF EXISTS new_transaction_update ON api.transactions_raw;

DROP FUNCTION IF EXISTS update_message_main();
DROP FUNCTION IF EXISTS update_transaction_main();
DROP FUNCTION IF EXISTS extract_proposal_ids(JSONB);
DROP FUNCTION IF EXISTS extract_proposal_failure_logs(json_data JSONB);
DROP FUNCTION IF EXISTS extract_metadata(JSONB);

DROP TABLE api.messages_main;
DROP TABLE api.messages_raw;
DROP TABLE api.transactions_main;

ALTER TABLE api.transactions_raw RENAME TO transactions;
ALTER TABLE api.blocks_raw RENAME TO blocks;

GRANT SELECT ON api.blocks TO web_anon;
GRANT SELECT ON api.transactions TO web_anon;

COMMIT;
