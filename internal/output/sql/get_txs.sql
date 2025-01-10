CREATE OR REPLACE FUNCTION api.get_address_filtered_transactions_and_successful_proposals(address TEXT)
RETURNS TABLE (id VARCHAR(64), data JSONB)
LANGUAGE SQL STABLE
AS $$
WITH
-- 1) Expand transactions => messages containing `address`
base_messages AS (
  SELECT
    t.id,
    t.data,
    msg.value AS message
  FROM api.transactions t
  CROSS JOIN LATERAL jsonb_array_elements(t.data -> 'tx' -> 'body' -> 'messages') AS msg(value)
  WHERE msg.value::text ILIKE '%' || address || '%'
),

-- 2) All successful transactions (code=0)
all_successful_txs AS (
  SELECT DISTINCT
    bm.id,
    bm.data
  FROM base_messages bm
  WHERE COALESCE((bm.data -> 'txResponse' ->> 'code')::INT, 0) = 0
),

-- 3) Errored transactions (code != 0)
--    Only keep if at least one message references `address`
--    in one of the allowed fields.
errored_with_address AS (
  SELECT DISTINCT
    bm.id,
    bm.data
  FROM base_messages bm
  WHERE COALESCE((bm.data -> 'txResponse' ->> 'code')::INT, 0) != 0
    AND (
      bm.message ->> 'sender'       = address
      OR bm.message ->> 'fromAddress' = address
      OR bm.message ->> 'admin'     = address
      OR bm.message ->> 'voter'     = address
      OR bm.message ->> 'address'   = address
      OR (
        bm.message ? 'proposers'
        AND address IN (
          SELECT jsonb_array_elements_text(bm.message -> 'proposers')
        )
      )
    )
),

-- 4) Identify submitted proposals (MsgSubmitProposal), capturing proposal_id.
submit_proposals AS (
  SELECT
    bm.id AS submit_id,
    bm.data AS submit_data,
    proposal_attr.attr ->> 'value' AS proposal_id
  FROM base_messages bm
  CROSS JOIN LATERAL jsonb_array_elements(bm.data -> 'tx' -> 'body' -> 'messages') AS msg(value)
  JOIN LATERAL (
    SELECT attr
    FROM jsonb_array_elements(bm.data -> 'txResponse' -> 'events') AS event,
         jsonb_array_elements(event -> 'attributes') AS attr
    WHERE event ->> 'type' = 'cosmos.group.v1.EventSubmitProposal'
      AND attr ->> 'key' = 'proposal_id'
    LIMIT 1
  ) AS proposal_attr ON TRUE
  WHERE msg.value ->> '@type' = '/cosmos.group.v1.MsgSubmitProposal'
),

-- 5) Find successful executions (EventExec) with a matching proposal_id.
execs AS (
  SELECT
    bm.id AS exec_id,
    bm.data AS exec_data,
    attrs.attr_map ->> 'proposal_id' AS proposal_id,
    attrs.attr_map ->> 'result' AS result
  FROM base_messages bm
  CROSS JOIN LATERAL (
    SELECT event
    FROM jsonb_array_elements(bm.data -> 'txResponse' -> 'events') AS event
    WHERE event ->> 'type' = 'cosmos.group.v1.EventExec'
    LIMIT 1
  ) AS exec_event
  JOIN LATERAL (
    SELECT jsonb_object_agg(attr ->> 'key', attr ->> 'value') AS attr_map
    FROM jsonb_array_elements(exec_event.event -> 'attributes') AS attr
  ) AS attrs(attr_map) ON TRUE
),

-- 6) matching_proposals = transactions that submitted a proposal
--    which was successfully executed.
matching_proposals AS (
  SELECT DISTINCT
    sp.submit_id AS id,
    sp.submit_data AS data
  FROM submit_proposals sp
  JOIN execs e
    ON sp.proposal_id = e.proposal_id
  -- optionally: AND e.result = 'PROPOSAL_EXECUTE_SUCCESS'
)

-- 7) Final SELECT:
--    (a) All successful transactions
--    (b) Errored transactions that reference `address` in allowed fields
--    (c) Successfully executed proposals
SELECT DISTINCT id, data
FROM (
  -- (a) All successful transactions
  SELECT id, data
  FROM all_successful_txs

  UNION

  -- (b) Errored transactions that reference `address` in allowed fields
  SELECT id, data
  FROM errored_with_address

  UNION

  -- (c) Successfully executed proposals (submission must be code=0)
  SELECT id, data
  FROM matching_proposals
  WHERE COALESCE((data -> 'txResponse' ->> 'code')::INT, 0) = 0
) combined;
$$;
