#!/usr/bin/env bash
# Basic QC script to check if the expected transactions are present in the PostgREST API
# Only the number of transaction, transaction memo and nested value are checked

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
  local addr expected_txs_len tx_checks addr_txs addr_txs_len tx_check memo_and_nested
  addr="$1"
  expected_txs_len="$2"
  tx_checks=("${@:3}")

  addr_txs=$(curl -s "http://${POSTGREST_HOST}:3000/rpc/get_messages_for_address?_address=${addr}")
  memo_and_nested=$(echo "$addr_txs" | jq '[.. | {memo: .memo?, nested: (.message_index? >= 10000)} | select(.memo != null)]')
  addr_txs_len=$(echo "${memo_and_nested}" | jq 'length')

  echo "Address: ${addr}"

  if [ "${addr_txs_len}" -ne "${expected_txs_len}" ]; then
    printf "  \\e[31m\\u2717\\e[0m Invalid number of transactions. Expected: %s, Found: %s\\n" "$expected_txs_len" "$addr_txs_len"
    return 1
  fi

  printf "  \\e[32m\\u2713\\e[0m Transactions found: %s\\n" "$addr_txs_len"

  for tx_check in "${tx_checks[@]}"; do
    IFS=":" read -r memo nested <<< "$tx_check"
    check_tx "$memo_and_nested" "$memo" "$nested" || return 1
  done
}

# Address-specific configurations
declare -A ADDRESS_EXPECTED_TXS=(
  ["${ADDR1}"]="28"
  ["${ADDR2}"]="12"
  ["${USER_GROUP_ADDRESS}"]="15"
  ["${POA_ADMIN_ADDRESS}"]="5"
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
  "tx-create-denom-proposal-submit:false"
  "tx-create-denom-proposal-vote:false"
  "tx-create-denom-proposal-exec:false"
  "tx-mint-new-denom-proposal-submit:false"
  "tx-mint-new-denom-proposal-vote:false"
  "tx-mint-new-denom-proposal-exec:false"
  "tx-send-new-denom-proposal-submit:false"
  "tx-send-new-denom-proposal-vote:false"
  "tx-send-new-denom-proposal-exec:false"
  "tx-update-group-members-proposal-submit:false"
  "tx-update-group-members-proposal-vote:false"
  "tx-update-group-members-proposal-exec:false"
  "tx-send-proposal-error-submit:false"
  "tx-send-proposal-error-vote:false"
  "tx-send-proposal-error-exec:false"
)

declare -a ADDRESS2_TX_CHECKS=(
  "tx-multi-send-to-poa-admin:false"
  "tx-change-admin:false"
  "tx-force-transfer:false"
  "tx-payout-proposal-submit:false"
  "tx-payout-proposal-submit:true"
  "tx-create-group-with-policy:false"
  "tx-send-new-denom-proposal-submit:false"
  "tx-send-new-denom-proposal-submit:true"
  "tx-update-group-members-proposal-submit:false"
  "tx-update-group-members-proposal-submit:true"
  "tx-send-addr2-to-addr1-error:false"
  "tx-send-proposal-error-submit:false"
)

declare -a USER_GROUP_ADDRESS_TX_CHECKS=(
  "tx-send-to-user-group:false"
  "tx-create-denom-proposal-submit:false"
  "tx-create-denom-proposal-submit:true"
  "tx-create-denom-proposal-exec:false"
  "tx-mint-new-denom-proposal-submit:false"
  "tx-mint-new-denom-proposal-submit:true"
  "tx-mint-new-denom-proposal-exec:false"
  "tx-send-new-denom-proposal-submit:false"
  "tx-send-new-denom-proposal-submit:true"
  "tx-send-new-denom-proposal-exec:false"
  "tx-update-group-members-proposal-submit:false"
  "tx-update-group-members-proposal-submit:true"
  "tx-update-group-members-proposal-exec:false"
  "tx-send-proposal-error-submit:false"
  "tx-send-proposal-error-exec:false"
)

declare -a POA_ADMIN_ADDRESS_TX_CHECKS=(
  "tx-send-to-poa-admin:false"
  "tx-multi-send-to-poa-admin:false"
  "tx-payout-proposal-submit:false"
  "tx-payout-proposal-submit:true"
  "tx-payout-proposal-exec:false"
)

declare -A ADDRESS_TX_CHECKS=(
  ["${ADDR1}"]="ADDRESS1_TX_CHECKS[@]"
  ["${ADDR2}"]="ADDRESS2_TX_CHECKS[@]"
  ["${USER_GROUP_ADDRESS}"]="USER_GROUP_ADDRESS_TX_CHECKS[@]"
  ["${POA_ADMIN_ADDRESS}"]="POA_ADMIN_ADDRESS_TX_CHECKS[@]"
)

for addr in "${!ADDRESS_EXPECTED_TXS[@]}"; do
  expected_txs_len="${ADDRESS_EXPECTED_TXS[$addr]}"
  tx_checks=("${!ADDRESS_TX_CHECKS[$addr]}")  # Dynamically fetch the array for the address

  check_addr_txs "$addr" "$expected_txs_len" "${tx_checks[@]}" || exit 1
done
