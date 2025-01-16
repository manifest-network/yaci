-- Retrieve messages where `address` is the user that initiated the transaction.
-- This function will also return failed transactions, i.e., transactions with an error code != 0.
-- You can use this function to track fees associated with transactions.
CREATE OR REPLACE FUNCTION api.txs_initiated_by(address TEXT)
RETURNS TABLE (id VARCHAR(64), data JSONB)
LANGUAGE SQL STABLE
AS $$
SELECT
    t.id,
    t.data
FROM api.transactions t
CROSS JOIN LATERAL jsonb_array_elements(t.data -> 'tx' -> 'body' -> 'messages') AS msg(value)
WHERE
    -- If `address` matches any of these fields...
    address = ANY(
      ARRAY[
        msg.value->>'sender',
        msg.value->>'fromAddress',
        msg.value->>'admin',
        msg.value->>'voter',
        msg.value->>'address',
        msg.value->>'executor'
      ]
    )
    OR (
      -- ...or if `proposers` array exists and contains `address`.
      msg.value ? 'proposers'
      AND address IN (
        SELECT jsonb_array_elements_text(msg.value -> 'proposers')
      )
    );
$$;
