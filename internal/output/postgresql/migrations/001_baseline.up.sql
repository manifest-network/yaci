BEGIN;

-- Create the schema if it doesn't exist
CREATE SCHEMA IF NOT EXISTS api;

-- Create the tables if they don't exist
CREATE TABLE IF NOT EXISTS api.blocks (
    id SERIAL PRIMARY KEY,
    data JSONB NOT NULL
);
CREATE TABLE IF NOT EXISTS api.transactions (
    id VARCHAR(64) PRIMARY KEY,
    data JSONB NOT NULL
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
GRANT SELECT ON api.blocks TO web_anon;
GRANT SELECT ON api.transactions TO web_anon;

CREATE OR REPLACE FUNCTION api.get_address_filtered_transactions_and_successful_proposals(address TEXT)
RETURNS TABLE (id VARCHAR(64), data JSONB)
AS $$
WITH base_messages AS (
  SELECT
    t.id,
    t.data,
    msg.value AS message
  FROM
    api.transactions t,
    LATERAL jsonb_array_elements(t.data -> 'tx' -> 'body' -> 'messages') AS msg(value)
  WHERE
    -- Exclude messages that are MsgSubmitProposal
    msg.value ->> '@type' != '/cosmos.group.v1.MsgSubmitProposal'
),
filtered_messages AS (
  SELECT
    id,
    data
  FROM
    base_messages
  WHERE
    -- Include only desired message types
    message ->> '@type' IN (
      '/cosmos.bank.v1beta1.MsgSend',
      '/osmosis.tokenfactory.v1beta1.MsgMint',
      '/osmosis.tokenfactory.v1beta1.MsgBurn'
    )
    -- Check if the message contains the given address anywhere in its content
    AND message::text ILIKE '%' || address || '%'
),
submit_proposals AS (
  SELECT
    t.id AS submit_id,
    t.data AS submit_data,
    proposal_attr.attr ->> 'value' AS proposal_id
  FROM
    api.transactions t
    JOIN LATERAL jsonb_array_elements(t.data -> 'tx' -> 'body' -> 'messages') AS msg(value) ON TRUE
    JOIN LATERAL (
      SELECT attr
      FROM jsonb_array_elements(t.data -> 'txResponse' -> 'events') AS event,
           jsonb_array_elements(event -> 'attributes') AS attr
      WHERE event ->> 'type' = 'cosmos.group.v1.EventSubmitProposal'
        AND attr ->> 'key' = 'proposal_id'
    ) AS proposal_attr ON TRUE
  WHERE
    msg.value ->> '@type' = '/cosmos.group.v1.MsgSubmitProposal'
    AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(msg.value -> 'messages') AS nested_msg(value)
      WHERE nested_msg.value::text ILIKE '%' || address || '%'
    )
    AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(msg.value -> 'messages') AS nested_msg(value)
      WHERE nested_msg.value ->> '@type' IN (
        '/cosmos.bank.v1beta1.MsgSend',
        '/osmosis.tokenfactory.v1beta1.MsgMint',
        '/osmosis.tokenfactory.v1beta1.MsgBurn',
        '/liftedinit.manifest.v1.MsgPayout',
        '/liftedinit.manifest.v1.MsgBurnHeldBalance'
      )
    )
),
execs AS (
  SELECT
    t.id AS exec_id,
    t.data AS exec_data,
    attrs.attr_map ->> 'proposal_id' AS proposal_id,
    attrs.attr_map ->> 'result' AS result
  FROM
    api.transactions t
    JOIN LATERAL (
      SELECT event
      FROM jsonb_array_elements(t.data -> 'txResponse' -> 'events') AS event
      WHERE event ->> 'type' = 'cosmos.group.v1.EventExec'
      LIMIT 1
    ) AS exec_event ON TRUE
    JOIN LATERAL (
      SELECT jsonb_object_agg(attr ->> 'key', attr ->> 'value') AS attr_map
      FROM jsonb_array_elements(exec_event.event -> 'attributes') AS attr
    ) AS attrs(attr_map) ON TRUE
),
matching_proposals AS (
  SELECT
    sp.submit_id AS id,
    sp.submit_data AS data
  FROM
    submit_proposals sp
    JOIN execs e ON sp.proposal_id = e.proposal_id
)
SELECT DISTINCT id, data
FROM
(
  SELECT
    id,
    data
  FROM filtered_messages
  WHERE COALESCE((data->'txResponse'->>'code')::int, 0) = 0
  UNION
  SELECT
    id,
    data
  FROM matching_proposals
  WHERE COALESCE((data->'txResponse'->>'code')::int, 0) = 0
) combined;
$$ LANGUAGE SQL STABLE;

COMMIT;
