#!/usr/bin/env bash

# Function to check if a transaction with a specific memo and nested value exists
function check_tx() {
  local transactions memo nested result
  transactions="$1"
  memo="$2"
  nested="$3"

  result=$(echo "$transactions" | jq --arg memo "$memo" --argjson nested "$nested" 'any(.[]; .memo == $memo and .nested == $nested)')

  if [ "$result" == "true" ]; then
    printf "  \\e[32m\\u2713\\e[0m %s:%s\\n" "$memo" "$nested"
    return 0
  else
    printf "  \\e[31m\\u2717\\e[0m %s:%s\\n" "$memo" "$nested"
    return 1
  fi
}

# Function to validate transactions for a specific address
function check_addr_txs() {
  local addr expected_txs_len tx_checks addr_txs addr_txs_len tx_check
  addr="$1"
  expected_txs_len="$2"
  tx_checks=("${@:3}")

  addr_txs=$(curl -s "http://localhost:3000/rpc/user_txs?address=${addr}" |
    jq '[.. | {memo: .memo?, nested: .is_nested?} | select(.memo != null or .nested != null)]')
  addr_txs_len=$(echo "$addr_txs" | jq 'length')

  echo "Address: ${addr}"

  if [ "$addr_txs_len" -ne "$expected_txs_len" ]; then
    printf "  \\e[31m\\u2717\\e[0m Invalid number of transactions. Expected: %s, Found: %s\\n" "$expected_txs_len" "$addr_txs_len"
    return 1
  fi

  printf "  \\e[32m\\u2713\\e[0m Transactions found: %s\\n" "$addr_txs_len"

  for tx_check in "${tx_checks[@]}"; do
    IFS=":" read -r memo nested <<< "$tx_check"
    check_tx "$addr_txs" "$memo" "$nested" || return 1
  done
}

# Address-specific configurations
declare -A ADDRESS_EXPECTED_TXS=(
  ["${ADDR1}"]="12"
)

declare -a ADDRESS1_TX_CHECKS=(
  "tx-send-to-poa-admin:false"
  "tx-multi-send-to-poa-admin:false"
  "tx-create-denom-ufoobar:false"
  "tx-modify-metadata-foobar:false"
  "tx-mint-to:false"
  "tx-burn-from:false"
  "tx-change-admin:false"
  "tx-force-transfer:false"
  "tx-payout-proposal-submit:false"
  "tx-payout-proposal-vote:false"
  "tx-payout-proposal-exec:false"
  "tx-create-group-with-policy:false"
  "tx-send-to-user-group:false"
)

declare -A ADDRESS_TX_CHECKS=(
  ["${ADDR1}"]="ADDRESS1_TX_CHECKS[@]"
)

for addr in "${!ADDRESS_EXPECTED_TXS[@]}"; do
  expected_txs_len="${ADDRESS_EXPECTED_TXS[$addr]}"
  tx_checks=("${!ADDRESS_TX_CHECKS[$addr]}")  # Dynamically fetch the array for the address

  check_addr_txs "$addr" "$expected_txs_len" "${tx_checks[@]}" || exit 1
done
