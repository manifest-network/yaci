CREATE OR REPLACE FUNCTION api.parse_msg_send(msg JSONB, proposal_id TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  SELECT jsonb_build_object(
    'proposal_id', proposal_id,
    'from',   msg->>'fromAddress',
    'to',     msg->>'toAddress',
    'amount', msg->'amount'
  )
$$;

CREATE OR REPLACE FUNCTION api.parse_bank_txs(msg JSONB, proposal_id TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  SELECT
    CASE
      WHEN (msg->>'@type') = '/cosmos.bank.v1beta1.MsgSend'
      THEN
        api.parse_msg_send(msg, proposal_id)
    ELSE
      jsonb_build_object(
        'type', msg->>'@type',
        'error', 'unsupported bank message type'
      )
    END
$$;
