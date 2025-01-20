#!/usr/bin/env bash
# This script checks if the total number of transactions in the PostgREST API is equal to the total number of
# transactions tracked in the file specified by TX_COUNT_PATH.
#
# The transactions are created by the init_manifest_ledger_txs.sh script and the total number of transaction created
# by the script is outputted in TX_COUNT_PATH.

# Make sure the file tracking the transaction count exists
if [ -f "${TX_COUNT_PATH}" ]; then
  total_tx_count=$(cat "${TX_COUNT_PATH}")
  echo "Total transactions: $total_tx_count"
else
  echo "${TX_COUNT_PATH} does not exist. Aborting..."
  exit 1
fi

# Query the PostgREST API to get the total number of transactions
output=$(curl http://postgrest:3000/transactions -I \
  -H "Range-Unit: items" \
  -H "Range: 0-0" \
  -H "Prefer: count=exact")

if [ $? -eq 0 ]; then
  # Retrieve the total number of transactions from the Content-Range header
  total_items=$(echo "$output" | grep -i "Content-Range" | awk -F'/' '{print $2}' | tr -d '\r')

  echo "Total items: $total_items"

  # Compare the total number of transactions from the transaction count with the total number of items from the header
  if [ "$total_items" -eq "${total_tx_count}" ]; then
    echo "Total items ($total_items) equal to number of transactions (${total_tx_count})."
    exit 0
  else
    echo "Total items ($total_items) are not equal to number of transactions (${total_tx_count})."
    exit 1
  fi
else
  exit 1
fi