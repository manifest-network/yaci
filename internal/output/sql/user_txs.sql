-- Returns relevant transactions for a given user address
--   Transactions initiated by the user
--   Executed proposals containing the user address
CREATE OR REPLACE FUNCTION api.user_txs(address TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
WITH

initiated_by AS (
  SELECT
    id,
    data
  FROM api.txs_initiated_by(address)
),

executed_proposals AS (
  SELECT
    id,
    data
  FROM api.executed_proposals_containing(address)
),

transfer_txs AS (
  SELECT
    id,
    data
  FROM api.txs_transfer_containing(address)
),

combined AS (
  SELECT id, data FROM initiated_by
  UNION
  SELECT id, data FROM executed_proposals
  UNION
  SELECT id, data FROM transfer_txs
),

expanded AS (
  SELECT
    c.id AS tx_hash,
    c.data AS tx_data,
    msg.value AS message,
    FALSE AS is_nested,
    NULL::TEXT       AS proposal_id
  FROM combined c
  CROSS JOIN LATERAL jsonb_array_elements(
    c.data -> 'tx' -> 'body' -> 'messages'
  ) AS msg(value)
),

nested_expanded AS (
  SELECT
    c.id        AS tx_hash,
    c.data      AS tx_data,
    nested.value AS message,
    TRUE        AS is_nested,
    (
      SELECT trim(both '"' from attr->>'value')
      FROM jsonb_array_elements(c.data->'txResponse'->'events') AS e
      CROSS JOIN LATERAL jsonb_array_elements(e->'attributes') AS attr
      WHERE e->>'type' = 'cosmos.group.v1.EventSubmitProposal'
        AND attr->>'key' = 'proposal_id'
      LIMIT 1
    ) AS proposal_id
  FROM executed_proposals c
  CROSS JOIN LATERAL jsonb_array_elements(c.data->'tx'->'body'->'messages') AS top_msg(value)
  CROSS JOIN LATERAL jsonb_array_elements(top_msg.value->'messages') AS nested(value)
  WHERE top_msg.value->>'@type' = '/cosmos.group.v1.MsgSubmitProposal'
),

all_msgs AS (
  SELECT tx_hash, tx_data, message, is_nested, proposal_id FROM expanded
  UNION ALL
  SELECT tx_hash, tx_data, message, is_nested, proposal_id FROM nested_expanded
)

SELECT jsonb_agg(
  jsonb_build_object(
    'is_nested', is_nested,
    -- 5a) sender: match the address with any of the possible sender fields
    'sender',
    COALESCE(
      NULLIF(all_msgs.message->>'sender', ''),
      NULLIF(all_msgs.message->>'fromAddress', ''),
      NULLIF(all_msgs.message->>'admin', ''),
      NULLIF(all_msgs.message->>'voter', ''),
      NULLIF(all_msgs.message->>'address', ''),
      NULLIF(all_msgs.message->>'executor', ''),
      (
        -- If `proposers` exists, grab its first element
        SELECT jsonb_array_elements_text(all_msgs.message->'proposers')
        LIMIT 1
      )
    ),
    'error', all_msgs.tx_data->'txResponse'->>'rawLog',
    'fee', all_msgs.tx_data->'tx'->'authInfo'->'fee',
    'tx_type', all_msgs.message->>'@type',
    'timestamp', all_msgs.tx_data->'txResponse'->>'timestamp',
    'tx_hash', all_msgs.tx_hash,
    'height', (all_msgs.tx_data->'txResponse'->>'height')::BIGINT,
    'metadata', api.parse_tx(all_msgs.message, all_msgs.tx_data, all_msgs.proposal_id)
  )
) AS user_txs
FROM all_msgs;
$$;
