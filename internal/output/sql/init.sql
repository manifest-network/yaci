CREATE SCHEMA IF NOT EXISTS api;

CREATE TABLE IF NOT EXISTS api.blocks_raw (
    id SERIAL PRIMARY KEY,
    data JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS api.transactions_raw (
    id VARCHAR(64) PRIMARY KEY,
    data JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS api.transactions_main (
    id VARCHAR(64) PRIMARY KEY REFERENCES api.transactions_raw(id),
    fee JSONB,
    memo TEXT,
    error TEXT,
    height TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS api.messages_raw(
    id VARCHAR(64) REFERENCES api.transactions_raw(id),
    message_index BIGINT,
    data JSONB,
    PRIMARY KEY (id, message_index)
);

CREATE TABLE IF NOT EXISTS api.messages_main(
    id VARCHAR(64),
    message_index BIGINT,
    type TEXT,
    sender TEXT,
    PRIMARY KEY (id, message_index),
    FOREIGN KEY (id, message_index) REFERENCES api.messages_raw(id, message_index)
);

-- Create a role for anonymous web access if it doesn't exist
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'web_anon') THEN
    CREATE ROLE web_anon NOLOGIN;
  END IF;
END
$$;

-- Grant access to the web_anon role. Will succeed even if the role already has access.
GRANT USAGE ON SCHEMA api TO web_anon;
GRANT SELECT ON api.blocks_raw TO web_anon;
GRANT SELECT ON api.transactions_main TO web_anon;
GRANT SELECT ON api.transactions_raw TO web_anon;
GRANT SELECT ON api.messages_raw TO web_anon;
GRANT SELECT ON api.messages_main TO web_anon;

CREATE OR REPLACE FUNCTION extract_proposal_failure_logs(json_data JSONB)
RETURNS TEXT
LANGUAGE sql
AS $$
WITH
  events AS (
    SELECT jsonb_array_elements(json_data->'txResponse'->'events') AS event
  ),

  typed_attributes AS (
    SELECT
      event->>'type' AS event_type,
      jsonb_array_elements(event->'attributes') AS attribute
    FROM events
  )

  SELECT
    TRIM(BOTH '"' FROM typed_attributes.attribute->>'value') AS logs
  FROM typed_attributes
  WHERE
    typed_attributes.event_type = 'cosmos.group.v1.EventExec'
    AND typed_attributes.attribute->>'key' = 'logs'
    AND EXISTS (
      SELECT 1
      FROM typed_attributes t2
      WHERE t2.event_type = typed_attributes.event_type
        AND t2.attribute->>'key' = 'result'
        AND t2.attribute->>'value' = '"PROPOSAL_EXECUTOR_RESULT_FAILURE"'
    )
  LIMIT 1;
$$;

CREATE OR REPLACE FUNCTION api.update_transaction_main()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
  error_text TEXT;
BEGIN
  error_text := NEW.data->'txResponse'->>'rawLog';

  IF error_text IS NULL THEN
    error_text := extract_proposal_failure_logs(NEW.data);
  END IF;

  INSERT INTO api.transactions_main (id, fee, memo, error, height)
  VALUES (
            NEW.id,
            NEW.data->'tx'->'authInfo'->'fee',
            NEW.data->'tx'->'body'->>'memo',
            error_text,
            NEW.data->'txResponse'->>'height'
         )
  ON CONFLICT (id) DO UPDATE
  SET fee = EXCLUDED.fee,
      memo = EXCLUDED.memo,
      error = EXCLUDED.error,
      height = EXCLUDED.height;

  INSERT INTO api.messages_raw (id, message_index, data)
  SELECT NEW.id, message_index - 1, message
  FROM jsonb_array_elements(NEW.data->'tx'->'body'->'messages') WITH ORDINALITY AS message(message, message_index);

  RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION api.update_message_main()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
  sender TEXT;
BEGIN
  sender := COALESCE(
    NULLIF(NEW.data->>'sender', ''),
    NULLIF(NEW.data->>'fromAddress', ''),
    NULLIF(NEW.data->>'admin', ''),
    NULLIF(NEW.data->>'voter', ''),
    NULLIF(NEW.data->>'address', ''),
    NULLIF(NEW.data->>'executor', ''),
    (
      SELECT jsonb_array_elements_text(NEW.data->'proposers')
      LIMIT 1
    ),
    (
      CASE
        WHEN jsonb_typeof(NEW.data->'inputs') = 'array'
             AND jsonb_array_length(NEW.data->'inputs') > 0
        THEN NEW.data->'inputs'->0->>'address'
        ELSE NULL
      END
    )
  );

  INSERT INTO api.messages_main (id, message_index, type, sender)
  VALUES (
           NEW.id,
           NEW.message_index,
           NEW.data->>'@type',
           sender
         )
  ON CONFLICT (id, message_index) DO NOTHING;

  RETURN NEW;
END;
$$;

CREATE OR REPLACE TRIGGER new_transaction_update
AFTER INSERT OR UPDATE
ON api.transactions_raw
FOR EACH ROW
EXECUTE FUNCTION api.update_transaction_main();

CREATE OR REPLACE TRIGGER new_message_update
AFTER INSERT OR UPDATE
ON api.messages_raw
FOR EACH ROW
EXECUTE FUNCTION api.update_message_main();
