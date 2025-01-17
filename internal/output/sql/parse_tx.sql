-- Parse transaction messages and extract relevant information to be used in the API.
CREATE OR REPLACE FUNCTION api.parse_tx(msg JSONB, data JSONB, proposal_id TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
SELECT
  CASE
    WHEN (msg->>'@type') LIKE  '/cosmos.bank%'
    THEN
      api.parse_bank_txs(msg, proposal_id)

    WHEN (msg->>'@type') LIKE '/liftedinit.manifest%'
    THEN
      api.parse_manifest_txs(msg, proposal_id)

    WHEN (msg->>'@type') LIKE '/cosmos.group%'
    THEN
      api.parse_group_txs(msg, data, proposal_id)

    WHEN (msg->>'@type') LIKE '/osmosis.tokenfactory%'
    THEN
      api.parse_tokenfactory_txs(msg, proposal_id)

    ELSE
      jsonb_build_object(
        'type', msg->>'@type',
        'error', 'unsupported message type'
      )
  END
  ||
  CASE
    -- If it's a nested message, add the proposal_id
    WHEN proposal_id IS NOT NULL
    THEN jsonb_build_object('proposal_id', proposal_id)
    ELSE '{}'::JSONB
  END
;
$$;
