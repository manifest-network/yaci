#!/usr/bin/env bash
# This script initializes the manifest ledger with transactions
# The transaction count if tracked in the file specified by TX_COUNT_PATH which is used by the YACI healthcheck script
set -e

TX_COUNT=0
PROPOSAL_ID=1
CONTRACT_VERSION=v0.1.0

# Download the wasm contract
echo "Downloading converter.wasm contract version ${CONTRACT_VERSION}"
curl -sL https://github.com/manifest-network/manifest-contracts/releases/download/${CONTRACT_VERSION}/converter.wasm --output /tmp/converter.wasm

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
  printf "\n"
  TX_COUNT=$((TX_COUNT + 1))
}

# $1 = proposal, $2 = voter address, $3 = note, rest = optional flags
function run_proposal() {
  manifestd tx group submit-proposal "/proposals/$1" ${COMMON_MANIFESTD_ARGS} --note "${3}-submit" "${@:4}" && sleep ${TIMEOUT_COMMIT}
  manifestd tx group vote ${PROPOSAL_ID} "$2" VOTE_OPTION_YES '' ${COMMON_MANIFESTD_ARGS} --note "${3}-vote" "${@:4}" && sleep ${TIMEOUT_COMMIT}
  sleep ${VOTING_TIMEOUT} # Wait for the voting period to end
  manifestd tx group exec ${PROPOSAL_ID} ${COMMON_MANIFESTD_ARGS} --note "${3}-exec" "${@:4}" && sleep ${TIMEOUT_COMMIT}
  printf "\n"
  PROPOSAL_ID=$((PROPOSAL_ID + 1))
  TX_COUNT=$((TX_COUNT + 3))
}

## Wasm module
echo "-> Storing converter.wasm contract"
run_tx tx wasm store /tmp/converter.wasm --from $KEY --note "tx-store-converter"

echo "-> Instantiating converter.wasm contract"
run_tx tx wasm instantiate 1 '{"admin":"'${POA_ADMIN_ADDRESS}'","poa_admin":"'${POA_ADMIN_ADDRESS}'","rate":"2","source_denom":"umfx","target_denom":"factory/'${POA_ADMIN_ADDRESS}'/upwr","paused":false}' --from $KEY --admin $ADDR1 --label "converter" --note "tx-instantiate-converter"

echo "-> Granting converter contract mint/burn permissions"
run_proposal "grant.json" "$ADDR1" "tx-grant-proposal" --from $KEY

# Converter contract address is `manifest14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4zfs7u`
echo "-> Converting 3 MFX to 6 PWR"
run_tx tx wasm execute ${CONVERT_WASM_ADDRESS} '{"convert":{}}' --from $KEY --note "tx-mint-from-converter" --amount 3000000umfx

## Bank module
echo "-> Bank send"
run_tx tx bank send $ADDR1 ${POA_ADMIN_ADDRESS} 10000${DENOM} --from $KEY --note "tx-send-to-poa-admin"

echo "-> Bank multi-send"
run_tx tx bank multi-send $ADDR1 $ADDR2 ${POA_ADMIN_ADDRESS} 10000${DENOM} --from $KEY --note "tx-multi-send-to-poa-admin"

## Vesting
echo "-> Creating periodic vesting account"
run_tx tx vesting create-periodic-vesting-account ${VESTING_ADDR} /generated/vesting_period.json --from $KEY --note "tx-create-periodic-vesting-account"

## Tokenfactory module
echo "-> Create new denom (TF)"
run_tx tx tokenfactory create-denom ufoobar --from $KEY --note "tx-create-denom-ufoobar"

echo "-> Modify new denom metadata(TF)"
run_tx tx tokenfactory modify-metadata factory/$ADDR1/ufoobar FOOBAR "This is the foobar token" 6 --from $KEY --note "tx-modify-metadata-foobar"

echo "-> Mint new denom (TF)"
run_tx tx tokenfactory mint-to $ADDR1 2000000factory/$ADDR1/ufoobar  --from $KEY --note "tx-mint-to"

echo "-> Burn new denom (TF)"
run_tx tx tokenfactory burn-from $ADDR1 1000000factory/$ADDR1/ufoobar  --from $KEY --note "tx-burn-from"

echo "-> Change admin of new denom (TF)"
run_tx tx tokenfactory change-admin factory/$ADDR1/ufoobar $ADDR2 --from $KEY --note "tx-change-admin"

## Manifest module
echo "-> Payout proposal (Manifest)"
run_proposal "payout.json" "$ADDR1" "tx-payout-proposal" --from $KEY

echo "-> BurnHeldBalance proposal (Manifest)"
run_proposal "burn.json" "$ADDR1" "tx-burn-proposal" --from $KEY

# Submit and withdraw payout and burn proposals to make sure collector works properly
echo "-> Submit and withdraw payout proposal (Manifest)"
run_tx tx group submit-proposal "/proposals/payout.json" --note "tx-payout-proposal-submit-2" --from $KEY
run_tx tx group withdraw-proposal $PROPOSAL_ID $ADDR1 --note "tx-payout-proposal-withdraw" --from $KEY
PROPOSAL_ID=$((PROPOSAL_ID + 1))

echo "-> Submit and withdraw burn proposal (Manifest)"
run_tx tx group submit-proposal "/proposals/burn.json" --note "tx-burn-proposal-submit-2" --from $KEY
run_tx tx group withdraw-proposal $PROPOSAL_ID $ADDR1 --note "tx-burn-proposal-withdraw" --from $KEY
PROPOSAL_ID=$((PROPOSAL_ID + 1))

echo "-> Mint UPWR proposal (Manifest)"
run_proposal "mint_upwr.json" "$ADDR1" "tx-mint-upwr-proposal" --from $KEY

echo "-> Burn UPWR proposal (Manifest)"
run_proposal "burn_upwr.json" "$ADDR1" "tx-burn-upwr-proposal" --from $KEY

### Group module
echo ${GROUP_MEMBERS} > members.json && cat members.json
echo ${DECISION_POLICY} > policy.json && cat policy.json
echo "-> Create group with policy"
run_tx tx group create-group-with-policy $ADDR1 "" "" members.json policy.json --from $KEY --group-policy-as-admin --note "tx-create-group-with-policy"

echo "-> Send to user group"
run_tx tx bank send $ADDR1 ${USER_GROUP_ADDRESS} 10000${DENOM} --from $KEY --note "tx-send-to-user-group"

echo "-> Create denom proposal (TF)"
run_proposal "create_denom.json" "$ADDR1" "tx-create-denom-proposal" --from $KEY

echo "-> Mint new denom proposal (TF)"
run_proposal "mint_new_denom.json" "$ADDR1" "tx-mint-new-denom-proposal" --from $KEY

echo "-> Send new denom proposal (TF)"
run_proposal "send_new_denom.json" "$ADDR1" "tx-send-new-denom-proposal" --from $KEY

echo "-> Update group members proposal"
run_proposal "update_group_members.json" "$ADDR1" "tx-update-group-members-proposal" --from $KEY

### Error transactions
echo "-> Send tx that will error (Manifest)"
run_tx tx bank send $ADDR2 $ADDR1 100000000001000000${DENOM} --from $KEY --note "tx-send-addr2-to-addr1-error"

echo "-> Send proposal that will error (Manifest)"
run_proposal "send_error.json" "$ADDR1" "tx-send-proposal-error" --from $KEY

echo "Total transactions: $TX_COUNT"
echo "${TX_COUNT}" > "${TX_COUNT_PATH}"
