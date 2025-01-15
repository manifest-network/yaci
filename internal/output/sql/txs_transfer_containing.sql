-- Return all successful direct transfer (send, ...) transactions containing a specific address anywhere in the transaction
-- The address matching is exact, i.e., it will not match substrings.
--   E.g., given an address of `manifest1abcd`, the string `factory/manifest1abcd/uabc` will not match.
CREATE OR REPLACE FUNCTION api.txs_transfer_containing(address TEXT)
RETURNS TABLE (id VARCHAR(64), data JSONB, message JSONB)
LANGUAGE SQL STABLE
AS $$
SELECT
    t.id,
    t.data,
    msg.value AS message
FROM api.transactions t
CROSS JOIN LATERAL jsonb_array_elements(t.data -> 'tx' -> 'body' -> 'messages') AS msg(value)
WHERE jsonb_path_exists(
    msg.value,
    '$.** ? (@.type() == "string" && @ == $addr)',
    jsonb_build_object('addr', address)
)
AND
    msg.value->>'@type' IN (
        '/cosmos.bank.v1beta1.MsgSend',
        '/ibc.applications.transfer.v1.MsgTransfer',
        '/osmosis.tokenfactory.v1beta1.MsgMint',
        '/osmosis.tokenfactory.v1beta1.MsgBurn'
    );
$$;
