#!/usr/bin/env bash
# This script initializes the manifest ledger genesis file and starts a ledger node

set -e

update_test_genesis () {
  cat $HOME_DIR/config/genesis.json | jq "$1" > $HOME_DIR/config/tmp_genesis.json && mv $HOME_DIR/config/tmp_genesis.json $HOME_DIR/config/genesis.json
}

echo "$MNEMO1" | $BINARY keys add "$KEY" --home="$HOME_DIR" --keyring-backend "$KEYRING" --recover
echo "$MNEMO2" | $BINARY keys add "$KEY2" --home="$HOME_DIR" --keyring-backend "$KEYRING" --recover
$BINARY init $MONIKER --home=$HOME_DIR --chain-id $CHAIN_ID
update_test_genesis '.consensus["params"]["block"]["max_gas"]="1000000000"'
update_test_genesis '.app_state["bank"]["denom_metadata"]=[{"base":"umfx","denom_units":[{"aliases":[],"denom":"umfx","exponent":0},{"aliases":[],"denom":"MFX","exponent":6}],"description":"MFX","display":"MFX","name":"MFX","symbol":"MFX","uri":"","uri_hash":""}]'
update_test_genesis '.app_state["gov"]["params"]["min_deposit"]=[{"denom":"'$DENOM'","amount":"1000000"}]'
update_test_genesis '.app_state["gov"]["params"]["voting_period"]="15s"'
update_test_genesis '.app_state["gov"]["params"]["expedited_voting_period"]="10s"'
update_test_genesis '.app_state["staking"]["params"]["bond_denom"]="'${BOND_DENOM}'"'
update_test_genesis '.app_state["staking"]["params"]["min_commission_rate"]="0.000000000000000000"'
update_test_genesis '.app_state["mint"]["params"]["mint_denom"]="'$DENOM'"'
update_test_genesis '.app_state["mint"]["params"]["blocks_per_year"]="6311520"'
update_test_genesis '.app_state["tokenfactory"]["params"]["denom_creation_fee"]=[]'
update_test_genesis '.app_state["tokenfactory"]["params"]["denom_creation_gas_consume"]=0'
update_test_genesis '.app_state["feegrant"]["allowances"]=[{"granter":"'${GAS_STATION_ADDR}'","grantee":"'${BANK_ADDR}'","allowance":{"@type":"/cosmos.feegrant.v1beta1.AllowedMsgAllowance","allowance":{"@type":"/cosmos.feegrant.v1beta1.BasicAllowance","spend_limit":[],"expiration":null},"allowed_messages":["/cosmos.bank.v1beta1.MsgSend"]}}]'
update_test_genesis '.app_state["group"]["group_seq"]="1"'
update_test_genesis '.app_state["group"]["groups"]=[{"id":"1","admin":"'${POA_ADMIN_ADDRESS}'","metadata":"AQ==","version":"2","total_weight":"2","created_at":"2024-05-16T15:10:54.372190727Z"}]'
update_test_genesis '.app_state["group"]["group_members"]=[{"group_id":"1","member":{"address":"'${ADDR1}'","weight":"1","metadata":"user1","added_at":"2024-05-16T15:10:54.372190727Z"}},{"group_id":"1","member":{"address":"'${ADDR2}'","weight":"1","metadata":"user2","added_at":"2024-05-16T15:10:54.372190727Z"}}]'
update_test_genesis '.app_state["group"]["group_policy_seq"]="1"'
update_test_genesis '.app_state["group"]["group_policies"]=[{"address":"'${POA_ADMIN_ADDRESS}'","group_id":"1","admin":"'${POA_ADMIN_ADDRESS}'","metadata":"AQ==","version":"2","decision_policy":{"@type":"/cosmos.group.v1.ThresholdDecisionPolicy","threshold":"1","windows":{"voting_period":"'${VOTING_TIMEOUT}'","min_execution_period":"0s"}},"created_at":"2024-05-16T15:10:54.372190727Z"}]'
update_test_genesis '.app_state["tokenfactory"]["factory_denoms"]=[{"denom":"factory/manifest1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsfmy9qj/upwr","authority_metadata":{"admin":"manifest1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsfmy9qj"}}]'
update_test_genesis '.app_state["bank"]["denom_metadata"] |= . + [{"description":"PWR","denom_units":[{"denom":"factory/manifest1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsfmy9qj/upwr","exponent":0,"aliases":["PWR"]},{"denom":"PWR","exponent":6,"aliases":["factory/manifest1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsfmy9qj/upwr"]}],"base":"factory/manifest1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsfmy9qj/upwr","display":"PWR","name":"POWER","symbol":"PWR","uri":"","uri_hash":""}]'
update_test_genesis '.app_state["wasm"]["params"]["code_upload_access"]["permission"]="Everybody"'
update_test_genesis '.app_state["wasm"]["params"]["instantiate_default_permission"]="Everybody"'
$BINARY genesis add-genesis-account $KEY 100000000000000000${BOND_DENOM},100000000000000000000000000000${DENOM} --keyring-backend $KEYRING --home=$HOME_DIR
$BINARY genesis add-genesis-account $KEY2 100000000000000000${DENOM} --keyring-backend $KEYRING --home=$HOME_DIR
$BINARY genesis gentx $KEY 1000000${BOND_DENOM} --keyring-backend $KEYRING --home=$HOME_DIR --chain-id $CHAIN_ID --commission-rate=0.0 --commission-max-rate=1.0 --commission-max-change-rate=0.1
$BINARY genesis collect-gentxs --home=$HOME_DIR
$BINARY genesis validate-genesis --home=$HOME_DIR
sed -i 's/laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:'$RPC'"/g' $HOME_DIR/config/config.toml
sed -i 's/cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' $HOME_DIR/config/config.toml
sed -i 's/address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:'$REST'"/g' $HOME_DIR/config/app.toml
sed -i 's/enable = false/enable = true/g' $HOME_DIR/config/app.toml
sed -i 's/pprof_laddr = "localhost:6060"/pprof_laddr = "localhost:'$PROFF'"/g' $HOME_DIR/config/config.toml
sed -i 's/laddr = "tcp:\/\/0.0.0.0:26656"/laddr = "tcp:\/\/0.0.0.0:'$P2P'"/g' $HOME_DIR/config/config.toml
sed -i 's/address = "localhost:9090"/address = "0.0.0.0:'$GRPC'"/g' $HOME_DIR/config/app.toml
sed -i 's/address = "localhost:9091"/address = "0.0.0.0:'$GRPC_WEB'"/g' $HOME_DIR/config/app.toml
sed -i 's/address = ":8080"/address = "0.0.0.0:'$ROSETTA'"/g' $HOME_DIR/config/app.toml
sed -i 's/timeout_commit = "5s"/timeout_commit = "'$TIMEOUT_COMMIT'"/g' $HOME_DIR/config/config.toml
POA_ADMIN_ADDRESS=${POA_ADMIN_ADDRESS} $BINARY start --home=${HOME_DIR} --pruning=nothing --minimum-gas-prices=0.0011${DENOM} --rpc.laddr="tcp://0.0.0.0:$RPC"
