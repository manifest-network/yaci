-- Returns relevant transactions for a given user address
--   Transactions initiated by the user
--   Executed proposals containing the user address
--   Transfer transactions containing the user address
CREATE OR REPLACE FUNCTION api.user_txs(address TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
WITH

-- Get all executed proposals containing the user address
executed_proposals AS (
  SELECT
    id,
    data
  FROM api.executed_proposals_containing(address)
),

-- Get all transactions containing the user address
txs_containing AS (
  SELECT
    id,
    data
  FROM api.txs_containing(address)
),

-- Combine all found transactions
combined AS (
  SELECT id, data FROM executed_proposals
  UNION
  SELECT id, data FROM txs_containing
),

-- Expand all top-level messages in the transactions
expanded AS (
  SELECT
    c.id AS tx_hash,
    c.data AS tx_data,
    msg.value AS message,
    FALSE AS is_nested,
    NULL::TEXT AS proposal_id,
    NULL::TEXT AS group_policy_address
  FROM combined c
  CROSS JOIN LATERAL jsonb_array_elements(
    c.data -> 'tx' -> 'body' -> 'messages'
  ) AS msg(value)
),

-- Expand all nested messages in the proposal
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
    ) AS proposal_id,
    top_msg.value->>'groupPolicyAddress' AS group_policy_address
  FROM executed_proposals c
  CROSS JOIN LATERAL jsonb_array_elements(c.data->'tx'->'body'->'messages') AS top_msg(value)
  CROSS JOIN LATERAL jsonb_array_elements(top_msg.value->'messages') AS nested(value)
  WHERE top_msg.value->>'@type' = '/cosmos.group.v1.MsgSubmitProposal'
),

-- Combine all top-level messages with nested messages
all_msgs AS (
  SELECT tx_hash, tx_data, message, is_nested, proposal_id, group_policy_address FROM expanded
  UNION ALL
  SELECT tx_hash, tx_data, message, is_nested, proposal_id, group_policy_address FROM nested_expanded
),

-- Parse all messages and extract relevant information
json_row AS (
  SELECT
    jsonb_build_object(
      'is_nested', is_nested,
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
        ),
        group_policy_address
      ),
      'error', all_msgs.tx_data->'txResponse'->>'rawLog',
      'fee', all_msgs.tx_data->'tx'->'authInfo'->'fee',
      'memo', all_msgs.tx_data->'tx'->'body'->>'memo',
      'tx_type', all_msgs.message->>'@type',
      'timestamp', all_msgs.tx_data->'txResponse'->>'timestamp',
      'tx_hash', all_msgs.tx_hash,
      'height', (all_msgs.tx_data->'txResponse'->>'height')::BIGINT,
      'metadata', api.parse_tx(all_msgs.message, all_msgs.tx_data, all_msgs.proposal_id)
    ) AS row_obj
  FROM all_msgs
)

-- Return all transactions containing the user address in any of the relevant fields
SELECT jsonb_agg(json_row.row_obj) AS user_txs
FROM json_row
WHERE jsonb_path_exists(
        json_row.row_obj,
        '$.** ? (@.type() == "string" && @ == $addr || @.type() == "array" && @[*] == $addr)',
        jsonb_build_object('addr', address)
  )
$$;
