BEGIN;

---
-- Do not drop the `metadata` key from the JSONB object
---
CREATE OR REPLACE FUNCTION extract_metadata(msg JSONB)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  WITH keys_to_remove AS (
      SELECT ARRAY['@type', 'sender', 'executor', 'admin', 'voter', 'messages', 'proposalId', 'proposers', 'authority', 'fromAddress']::text[] AS keys
  )
  SELECT msg - (SELECT keys FROM keys_to_remove)
$$;

---
-- Convert the existing data to the new schema using a staging table and our update triggers
---
CREATE TABLE IF NOT EXISTS api.transactions_staging (
    id VARCHAR(64) PRIMARY KEY,
    data JSONB NOT NULL
);

CREATE OR REPLACE TRIGGER staging_transaction_update
AFTER INSERT OR UPDATE
ON api.transactions_staging
FOR EACH ROW
EXECUTE FUNCTION update_transaction_main();

INSERT INTO api.transactions_staging(id, data)
SELECT id, data
FROM api.transactions_raw;

DROP TRIGGER staging_transaction_update ON api.transactions_staging;
DROP TABLE api.transactions_staging;

COMMIT;
