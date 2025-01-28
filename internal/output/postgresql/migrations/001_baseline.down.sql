BEGIN;

DROP FUNCTION IF EXISTS api.get_address_filtered_transactions_and_successful_proposals(TEXT);

REVOKE SELECT ON api.transactions FROM web_anon;
REVOKE SELECT ON api.blocks FROM web_anon;
REVOKE USAGE ON SCHEMA api FROM web_anon;

DROP TABLE IF EXISTS api.transactions;
DROP TABLE IF EXISTS api.blocks;

DROP SCHEMA IF EXISTS api CASCADE;

DROP ROLE IF EXISTS web_anon;

COMMIT;
