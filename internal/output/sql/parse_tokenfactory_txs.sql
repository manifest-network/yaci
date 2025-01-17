CREATE OR REPLACE FUNCTION api.parse_msg_create_denom(msg JSONB, proposal_id TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  SELECT jsonb_build_object(
    'subdenom', msg->>'subdenom'
  )
$$;

CREATE OR REPLACE FUNCTION api.parse_set_denom_metadata(msg JSONB, proposal_id TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  SELECT jsonb_object_agg(key, value) FROM jsonb_each_text(msg->'metadata')
$$;

CREATE OR REPLACE FUNCTION api.parse_tokenfactory_txs(msg JSONB, proposal_id TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
SELECT
  CASE
    WHEN (msg->>'@type') = '/osmosis.tokenfactory.v1.MsgCreateDenom'
    THEN
      api.parse_msg_create_denom(msg, proposal_id)
    WHEN (msg->>'@type') = '/osmosis.tokenfactory.v1.MsgSetDenomMetadata'
    THEN
      api.parse_set_denom_metadata(msg, proposal_id)
    ELSE
      jsonb_build_object(
        'type', msg->>'@type',
        'error', 'unsupported tokenfactory message type'
      )
  END
$$;
