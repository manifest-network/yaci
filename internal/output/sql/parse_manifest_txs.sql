CREATE OR REPLACE FUNCTION api.parse_msg_payout(msg JSONB, proposal_id TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  SELECT jsonb_build_object(
    'proposal_id', proposal_id,
    'payout_pairs', (
      SELECT jsonb_agg(payout_pairs)
      FROM jsonb_array_elements(msg->'payoutPairs') AS payout_pairs
    )
  )
$$;

CREATE OR REPLACE FUNCTION api.parse_msg_burn_held_balance(msg JSONB, proposal_id TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  SELECT jsonb_build_object(
    'proposal_id', proposal_id,
    'burn_coins', (
      SELECT jsonb_agg(burn_coins)
      FROM jsonb_array_elements(msg->'burnCoins') AS burn_coins
    )
  )
$$;

CREATE OR REPLACE FUNCTION api.parse_manifest_txs(msg JSONB, proposal_id TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
SELECT
  CASE
    WHEN (msg->>'@type') = '/liftedinit.manifest.v1.MsgPayout'
    THEN
      api.parse_msg_payout(msg, proposal_id)

    WHEN (msg->>'@type') = '/liftedinit.manifest.v1.MsgBurnHeldBalance'
    THEN
      api.parse_msg_burn_held_balance(msg, proposal_id)
    ELSE
      jsonb_build_object(
        'type', msg->>'@type',
        'error', 'unsupported manifest message type'
      )
  END
$$;
