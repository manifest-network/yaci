-- Parse transaction messages and extract relevant information to be used in the API.
CREATE OR REPLACE FUNCTION api.parse_tx(msg JSONB, data JSONB, proposal_id TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
SELECT
  CASE
    -- 1) If it's a /cosmos.bank.v1beta1.MsgSend
    WHEN (msg->>'@type') = '/cosmos.bank.v1beta1.MsgSend'
    THEN
      jsonb_build_object(
        'from',   msg->>'fromAddress',
        'to',     msg->>'toAddress',
        'amount', msg->'amount'
      )

    -- 2) If it's a /cosmos.group.v1.MsgSubmitProposal
    WHEN (msg->>'@type') = '/cosmos.group.v1.MsgSubmitProposal'
    THEN
      jsonb_build_object(
        'proposal_id', (
          SELECT trim(both '"' from attr->>'value')
          FROM jsonb_array_elements(data->'txResponse'->'events') AS e
          CROSS JOIN LATERAL jsonb_array_elements(e->'attributes') AS attr
          WHERE e->>'type' = 'cosmos.group.v1.EventSubmitProposal'
            AND attr->>'key' = 'proposal_id'
          LIMIT 1
        )
        -- more fields if you want ...
      )

    -- 3) Otherwise, just return the entire message
    ELSE
      msg
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
