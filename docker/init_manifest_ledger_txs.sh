#!/bin/bash

# Send some tokens to the POA admin address
manifestd tx bank send $ADDR1 ${POA_ADMIN_ADDRESS} 10000${DENOM} ${COMMON_MANIFESTD_ARGS} && sleep ${TIMEOUT_COMMIT}

# Create a new token and modify its metadata
manifestd tx tokenfactory create-denom ufoobar ${COMMON_MANIFESTD_ARGS} --from $KEY && sleep ${TIMEOUT_COMMIT}
manifestd tx tokenfactory modify-metadata factory/$ADDR1/ufoobar FOOBAR "This is the foobar token" 6 ${COMMON_MANIFESTD_ARGS} --from $KEY && sleep ${TIMEOUT_COMMIT}

# Submit, vote and execute a Payout proposal
echo ${PAYOUT_PROPOSAL} > proposal.json && cat proposal.json
manifestd tx group submit-proposal proposal.json ${COMMON_MANIFESTD_ARGS} --from $KEY && sleep ${TIMEOUT_COMMIT}
manifestd tx group vote 1 $ADDR1 VOTE_OPTION_YES '' ${COMMON_MANIFESTD_ARGS} --from $KEY && sleep ${TIMEOUT_COMMIT}
sleep 10 # Wait for the voting period to end
manifestd tx group exec 1 ${COMMON_MANIFESTD_ARGS} --from $KEY && sleep ${TIMEOUT_COMMIT}

# Send some tokens to the POA admin address again, this time using the amino-json sign mode
manifestd tx bank send $ADDR1 ${POA_ADMIN_ADDRESS} 10000${DENOM} ${COMMON_MANIFESTD_ARGS} --sign-mode amino-json && sleep ${TIMEOUT_COMMIT}

# Submit, vote and execute a Payout proposal using the amino-json sign mode
# Payout will be made to ADDR2
cat proposal.json | jq '.title="Payout proposal with AMINO"' > other_proposal.json
manifestd tx group submit-proposal other_proposal.json ${COMMON_MANIFESTD_ARGS} --from $KEY --sign-mode amino-json && sleep ${TIMEOUT_COMMIT}
manifestd tx group vote 2 $ADDR1 VOTE_OPTION_YES '' ${COMMON_MANIFESTD_ARGS} --from $KEY --sign-mode amino-json && sleep ${TIMEOUT_COMMIT}
sleep 10 # Wait for the voting period to end
manifestd tx group exec 2 ${COMMON_MANIFESTD_ARGS} --from $KEY --sign-mode amino-json && sleep ${TIMEOUT_COMMIT}

# Create a group with policy.
echo ${GROUP_MEMBERS} > members.json && cat members.json
echo ${DECISION_POLICY} > policy.json && cat policy.json
manifestd tx group create-group-with-policy $ADDR1 "" "" members.json policy.json ${COMMON_MANIFESTD_ARGS} --from $KEY && sleep ${TIMEOUT_COMMIT}
