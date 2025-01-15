-- Returns all proposals containing the given address and that were successfully executed
CREATE OR REPLACE FUNCTION api.executed_proposals_containing(address TEXT)
RETURNS TABLE (id VARCHAR(64), data JSONB)
LANGUAGE SQL STABLE
AS $$
WITH

-- Get all messages that contain the given address in any of its `messages/**` fields
base_messages AS (
  SELECT
      id,
      data,
      message
  FROM api.txs_containing(address)
),

-- Get all group proposals containing the given address
submit_proposals AS (
  SELECT
    bm.id AS submit_id,
    bm.data AS submit_data,
    proposal_attr.attr ->> 'value' AS proposal_id
  FROM base_messages bm
  JOIN LATERAL (
    SELECT attr
    FROM jsonb_array_elements(bm.data -> 'txResponse' -> 'events') AS event,
         jsonb_array_elements(event -> 'attributes') AS attr
    WHERE event ->> 'type' = 'cosmos.group.v1.EventSubmitProposal'
      AND attr ->> 'key' = 'proposal_id'
    LIMIT 1
  ) AS proposal_attr ON TRUE
  WHERE bm.message ->> '@type' = '/cosmos.group.v1.MsgSubmitProposal'
),

-- Find all proposal execution events and their corresponding proposals
-- The executor can be any address which is why we can't use `base_messages` here
execs AS (
  SELECT
    t.id AS exec_id,
    t.data AS exec_data,
    attrs.attr_map ->> 'proposal_id' AS proposal_id,
    attrs.attr_map ->> 'result' AS result
  FROM api.transactions t
  CROSS JOIN LATERAL (
    SELECT event
    FROM jsonb_array_elements(t.data -> 'txResponse' -> 'events') AS event
    WHERE event ->> 'type' = 'cosmos.group.v1.EventExec'
    LIMIT 1
  ) AS exec_event
  JOIN LATERAL (
    SELECT jsonb_object_agg(attr ->> 'key', attr ->> 'value') AS attr_map
    FROM jsonb_array_elements(exec_event.event -> 'attributes') AS attr
  ) AS attrs(attr_map) ON TRUE
)

-- Find all proposals that were successfully executed
SELECT DISTINCT
  sp.submit_id AS id,
  sp.submit_data AS data
FROM submit_proposals sp
JOIN execs e
  ON sp.proposal_id = e.proposal_id
WHERE e.result = '"PROPOSAL_EXECUTOR_RESULT_SUCCESS"';
$$;
