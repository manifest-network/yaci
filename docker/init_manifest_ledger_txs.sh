#!/usr/bin/env bash
# This script initializes the manifest ledger with transactions
# The transaction count if tracked in the file specified by TX_COUNT_PATH which is used by the YACI healthcheck script

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

# $1 = proposal, $2 = voter address, rest = optional flags
function run_proposal() {
  echo "$1" >proposal.json && cat proposal.json
  manifestd tx group submit-proposal proposal.json ${COMMON_MANIFESTD_ARGS} "${@:3}" && sleep ${TIMEOUT_COMMIT}
  manifestd tx group vote ${PROPOSAL_ID} "$2" VOTE_OPTION_YES '' ${COMMON_MANIFESTD_ARGS} "${@:3}" && sleep ${TIMEOUT_COMMIT}
  sleep ${VOTING_TIMEOUT} # Wait for the voting period to end
  manifestd tx group exec ${PROPOSAL_ID} ${COMMON_MANIFESTD_ARGS} "${@:3}" && sleep ${TIMEOUT_COMMIT}
  PROPOSAL_ID=$((PROPOSAL_ID + 1))
  TX_COUNT=$((TX_COUNT + 3))
}

# Send some tokens to the POA admin address
run_tx tx bank send $ADDR1 ${POA_ADMIN_ADDRESS} 10000${DENOM} --from $KEY --note "tx-send-to-poa-admin"

# Create a new token and modify its metadata
run_tx tx tokenfactory create-denom ufoobar --from $KEY --note "tx-create-denom-ufoobar"
run_tx tx tokenfactory modify-metadata factory/$ADDR1/ufoobar FOOBAR "This is the foobar token" 6 --from $KEY --note "tx-modify-metadata-foobar"

# Submit, vote and execute a Payout proposal
run_proposal "${PAYOUT_PROPOSAL}" "$ADDR1" --from $KEY --note "tx-payout-proposal"

# Create a group with policy.
echo ${GROUP_MEMBERS} > members.json && cat members.json
echo ${DECISION_POLICY} > policy.json && cat policy.json
run_tx tx group create-group-with-policy $ADDR1 "" "" members.json policy.json --from $KEY --note "tx-create-group-with-policy"

# Send some tokens to the new group
run_tx tx bank send $ADDR1 ${USER_GROUP_ADDRESS} 10000${DENOM} --from $KEY --note "tx-send-to-user-group"

# Submit, vote and execute a CreateDenom proposal
run_proposal "${CREATE_DENOM_PROPOSAL}" "$ADDR1" --from $KEY --note "tx-create-denom-proposal"

# Mint some of the new token to the new user group
run_proposal "${MINT_NEW_DENOM_PROPOSAL}" "$ADDR1" --from $KEY --note "tx-mint-new-denom-proposal"

# Send some of the new token to ADDR2
run_proposal "${SEND_NEW_DENOM_PROPOSAL}" "$ADDR1" --from $KEY --note "tx-send-new-denom-proposal"

# Update group members
run_proposal "${UPDATE_GROUP_MEMBERS_PROPOSAL}" "$ADDR1" --from $KEY --note "tx-update-group-members-proposal"

echo "Total transactions: $TX_COUNT"
echo "${TX_COUNT}" > "${TX_COUNT_PATH}"
