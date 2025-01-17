-- Return all transactions containing a specific address anywhere in the transaction and transaction response.
-- The address matching works with both quoted and non-quoted strings.
-- Other substrings will NOT get matched.
--   E.g., given an address of `manifest1abcd`, the string `factory/manifest1abcd/uabc` will not match.
-- This function will return failed transactions, i.e., transactions with an error code != 0.
CREATE OR REPLACE FUNCTION api.txs_containing(address TEXT)
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
    t.data,
    format('$.** ? (@ like_regex "^\"*%s\"*$")', address)::JSONPATH
);
$$;
