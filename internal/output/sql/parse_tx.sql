-- Parse transaction messages and extract relevant information to be used in the API.
CREATE OR REPLACE FUNCTION api.parse_tx(msg JSONB)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  WITH keys_to_remove AS (
      SELECT ARRAY['@type', 'sender', 'executor', 'voter', 'messages', 'proposalId', 'proposers', 'authority', 'fromAddress', 'metadata']::text[] AS keys
  )
  SELECT
    CASE
      -- If 'metadata' key exists and is a JSON object, merge its contents into the top-level JSON
      WHEN msg ? 'metadata' AND jsonb_typeof(msg->'metadata') = 'object' THEN
        (msg - (SELECT keys FROM keys_to_remove)) || (msg->'metadata')
      ELSE
        msg - (SELECT keys FROM keys_to_remove)
    END
$$;
