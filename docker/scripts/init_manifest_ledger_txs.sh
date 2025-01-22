#!/usr/bin/env bash
# This script initializes the manifest ledger with transactions
# The transaction count if tracked in the file specified by TX_COUNT_PATH which is used by the YACI healthcheck script
set -e

TX_COUNT=0
PROPOSAL_ID=1

# We want to keep track of the transaction count in a file; check if the directory containing the file exists
if [ -d "${TX_COUNT_DIR}" ]; then
  echo "${TX_COUNT_DIR} exists"
else
  echo "${TX_COUNT_DIR} does not exist. Aborting..."
  exit 1
fi

# $1 = command, rest = optional flags
function run_tx() {
  manifestd "$@" ${COMMON_MANIFESTD_ARGS} && sleep ${TIMEOUT_COMMIT}
  TX_COUNT=$((TX_COUNT + 1))
}

# $1 = proposal, $2 = voter address, $3 = note, rest = optional flags
function run_proposal() {
  manifestd tx group submit-proposal "/proposals/$1" ${COMMON_MANIFESTD_ARGS} --note "${3}-submit" "${@:4}" && sleep ${TIMEOUT_COMMIT}
  manifestd tx group vote ${PROPOSAL_ID} "$2" VOTE_OPTION_YES '' ${COMMON_MANIFESTD_ARGS} --note "${3}-vote" "${@:4}" && sleep ${TIMEOUT_COMMIT}
  sleep ${VOTING_TIMEOUT} # Wait for the voting period to end
  manifestd tx group exec ${PROPOSAL_ID} ${COMMON_MANIFESTD_ARGS} --note "${3}-exec" "${@:4}" && sleep ${TIMEOUT_COMMIT}
  PROPOSAL_ID=$((PROPOSAL_ID + 1))
  TX_COUNT=$((TX_COUNT + 3))
}

## Bank module
run_tx tx bank send $ADDR1 ${POA_ADMIN_ADDRESS} 10000${DENOM} --from $KEY --note "tx-send-to-poa-admin"
run_tx tx bank multi-send $ADDR1 $ADDR2 ${POA_ADMIN_ADDRESS} 10000${DENOM} --from $KEY --note "tx-multi-send-to-poa-admin"

## Tokenfactory module
run_tx tx tokenfactory create-denom ufoobar --from $KEY --note "tx-create-denom-ufoobar"
run_tx tx tokenfactory modify-metadata factory/$ADDR1/ufoobar FOOBAR "This is the foobar token" 6 --from $KEY --note "tx-modify-metadata-foobar"

#run_tx tx tokenfactory mint-to $ADDR1 2000000factory/$ADDR1/ufoobar  --from $KEY --note "tx-mint-to"
#run_tx tx tokenfactory burn-from $ADDR1 1000000factory/$ADDR1/ufoobar  --from $KEY --note "tx-burn-from"
#run_tx tx tokenfactory change-admin factory/$ADDR1/ufoobar $ADDR2 --from $KEY --note "tx-change-admin"
#run_tx tx tokenfactory force-transfer 1000factory/$ADDR1/ufoobar $ADDR1 $ADDR2 --from $KEY2 --note "tx-force-transfer"
#
### Manifest module
#run_proposal "payout.json" "$ADDR1" "tx-payout-proposal" --from $KEY
#
### Group module
echo ${GROUP_MEMBERS} > members.json && cat members.json
echo ${DECISION_POLICY} > policy.json && cat policy.json
run_tx tx group create-group-with-policy $ADDR1 "" "" members.json policy.json --from $KEY --note "tx-create-group-with-policy"
#run_tx tx group create-group-with-policy $ADDR1 "" "" members.json policy.json --from $KEY --group-policy-as-admin --note "tx-create-group-with-policy"
run_tx tx bank send $ADDR1 ${USER_GROUP_ADDRESS} 10000${DENOM} --from $KEY --note "tx-send-to-user-group"
#
run_proposal "create_denom.json" "$ADDR1" "tx-create-denom-proposal" --from $KEY
#run_proposal "mint_new_denom.json" "$ADDR1" "tx-mint-new-denom-proposal" --from $KEY
#run_proposal "send_new_denom.json" "$ADDR1" "tx-send-new-denom-proposal" --from $KEY
run_proposal "update_group_members.json" "$ADDR1" "tx-update-group-members-proposal" --from $KEY
#
echo "Total transactions: $TX_COUNT"
echo "${TX_COUNT}" > "${TX_COUNT_PATH}"
