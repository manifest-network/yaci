#!/usr/bin/env bash

# Send some tokens to the POA admin address
manifestd tx bank send $ADDR1 ${POA_ADMIN_ADDRESS} 10000${DENOM} ${COMMON_MANIFESTD_ARGS} --note "tx-01" && sleep ${TIMEOUT_COMMIT}

# Create a new token and modify its metadata
manifestd tx tokenfactory create-denom ufoobar ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-02" && sleep ${TIMEOUT_COMMIT}
manifestd tx tokenfactory modify-metadata factory/$ADDR1/ufoobar FOOBAR "This is the foobar token" 6 ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-03" && sleep ${TIMEOUT_COMMIT}

# Submit, vote and execute a Payout proposal
echo ${PAYOUT_PROPOSAL} > payout_proposal.json && cat payout_proposal.json
manifestd tx group submit-proposal payout_proposal.json ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-04" && sleep ${TIMEOUT_COMMIT}
manifestd tx group vote 1 $ADDR1 VOTE_OPTION_YES '' ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-05" && sleep ${TIMEOUT_COMMIT}
sleep ${VOTING_TIMEOUT} # Wait for the voting period to end
manifestd tx group exec 1 ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-06" && sleep ${TIMEOUT_COMMIT}

# Send some tokens to the POA admin address again, this time using the amino-json sign mode
manifestd tx bank send $ADDR1 ${POA_ADMIN_ADDRESS} 10000${DENOM} ${COMMON_MANIFESTD_ARGS} --sign-mode amino-json --note "tx-07" && sleep ${TIMEOUT_COMMIT}

# Submit, vote and execute a Payout proposal using the amino-json sign mode
# Payout will be made to ADDR2
cat payout_proposal.json | jq '.title="Payout proposal with AMINO"' > other_proposal.json
manifestd tx group submit-proposal other_proposal.json ${COMMON_MANIFESTD_ARGS} --from $KEY --sign-mode amino-json --note "tx-08" && sleep ${TIMEOUT_COMMIT}
manifestd tx group vote 2 $ADDR1 VOTE_OPTION_YES '' ${COMMON_MANIFESTD_ARGS} --from $KEY --sign-mode amino-json --note "tx-09" && sleep ${TIMEOUT_COMMIT}
sleep ${VOTING_TIMEOUT} # Wait for the voting period to end
manifestd tx group exec 2 ${COMMON_MANIFESTD_ARGS} --from $KEY --sign-mode amino-json --note "tx-10" && sleep ${TIMEOUT_COMMIT}

# Create a group with policy.
echo ${GROUP_MEMBERS} > members.json && cat members.json
echo ${DECISION_POLICY} > policy.json && cat policy.json
manifestd tx group create-group-with-policy $ADDR1 "" "" members.json policy.json ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-11" --group-policy-as-admin && sleep ${TIMEOUT_COMMIT}

# Send some tokens to the new group
manifestd tx bank send $ADDR1 ${USER_GROUP_ADDRESS} 10000${DENOM} ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-12" && sleep ${TIMEOUT_COMMIT}

# Submit, vote and execute a CreateDenom proposal
echo ${CREATE_DENOM_PROPOSAL} > create_denom_proposal.json && cat create_denom_proposal.json
manifestd tx group submit-proposal create_denom_proposal.json ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-13" && sleep ${TIMEOUT_COMMIT}
manifestd tx group vote 3 $ADDR1 VOTE_OPTION_YES '' ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-14" && sleep ${TIMEOUT_COMMIT}
sleep ${VOTING_TIMEOUT} # Wait for the voting period to end
manifestd tx group exec 3 ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-15" && sleep ${TIMEOUT_COMMIT}

# Mint some of the new token to the new user group
echo ${MINT_NEW_DENOM_PROPOSAL} > mint_new_denom_proposal.json && cat mint_new_denom_proposal.json
manifestd tx group submit-proposal mint_new_denom_proposal.json ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-16" && sleep ${TIMEOUT_COMMIT}
manifestd tx group vote 4 $ADDR1 VOTE_OPTION_YES '' ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-17" && sleep ${TIMEOUT_COMMIT}
sleep ${VOTING_TIMEOUT} # Wait for the voting period to end
manifestd tx group exec 4 ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-18" && sleep ${TIMEOUT_COMMIT}

# Send some of the new token to ADDR2
echo ${SEND_NEW_DENOM_PROPOSAL} > send_new_denom_proposal.json && cat send_new_denom_proposal.json
manifestd tx group submit-proposal send_new_denom_proposal.json ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-19" && sleep ${TIMEOUT_COMMIT}
manifestd tx group vote 5 $ADDR1 VOTE_OPTION_YES '' ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-20" && sleep ${TIMEOUT_COMMIT}
sleep ${VOTING_TIMEOUT} # Wait for the voting period to end
manifestd tx group exec 5 ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-21" && sleep ${TIMEOUT_COMMIT}

# Update group members
echo ${UPDATE_GROUP_MEMBERS_PROPOSAL} > update_group_members_proposal.json && cat update_group_members_proposal.json
manifestd tx group submit-proposal update_group_members_proposal.json ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-22" && sleep ${TIMEOUT_COMMIT}
manifestd tx group vote 6 $ADDR1 VOTE_OPTION_YES '' ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-23" && sleep ${TIMEOUT_COMMIT}
sleep ${VOTING_TIMEOUT} # Wait for the voting period to end
manifestd tx group exec 6 ${COMMON_MANIFESTD_ARGS} --from $KEY --note "tx-24" && sleep ${TIMEOUT_COMMIT}
