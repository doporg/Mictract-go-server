#!/bin/bash
# input config_block.pb configtx.yaml||anchors.json
# output org_update_in_envelope.pb
WORKDIR=/mictract/scripts/addorg

createConfigUpdate() {
  CHANNEL=$1
  ORIGINAL=$2
  MODIFIED=$3
  OUTPUT=$4

  set -x
  configtxlator proto_encode --input "${ORIGINAL}" --type common.Config >$WORKDIR/gen/original_config.pb
  configtxlator proto_encode --input "${MODIFIED}" --type common.Config >$WORKDIR/gen/modified_config.pb
  configtxlator compute_update --channel_id "${CHANNEL}" --original $WORKDIR/gen/original_config.pb --updated $WORKDIR/gen/modified_config.pb >$WORKDIR/gen/config_update.pb
  configtxlator proto_decode --input $WORKDIR/gen/config_update.pb --type common.ConfigUpdate >$WORKDIR/gen/config_update.json
  echo '{"payload":{"header":{"channel_header":{"channel_id":"'$CHANNEL'", "type":2}},"data":{"config_update":'$(cat $WORKDIR/gen/config_update.json)'}}}' | jq . >$WORKDIR/gen/config_update_in_envelope.json
  configtxlator proto_encode --input $WORKDIR/gen/config_update_in_envelope.json --type common.Envelope >"${OUTPUT}"
  set +x
}

updateAnchors() {
	jq -s '.[0] * {"channel_group":{"groups":{"Application":{"groups":{"'${MSPID}'":{"values":{"AnchorPeers":.[1]}}}}}}}' $WORKDIR/gen/config.json anchors.json > $WORKDIR/gen/modified_config.json
	createConfigUpdate ${CHANNEL_NAME} $WORKDIR/gen/config.json $WORKDIR/gen/modified_config.json $WORKDIR/org_update_in_envelope.pb
}

addOrg() {
	configtxgen -configPath $WORKDIR -printOrg ${MSPID} >$WORKDIR/gen/org.json
	jq -s '.[0] * {"channel_group":{"groups":{"Application":{"groups": {"'${MSPID}'":.[1]}}}}}' $WORKDIR/gen/config.json $WORKDIR/gen/org.json > $WORKDIR/gen/modified_config.json
	createConfigUpdate ${CHANNEL_NAME} $WORKDIR/gen/config.json $WORKDIR/gen/modified_config.json $WORKDIR/org_update_in_envelope.pb
}

addOrderers() {
	jq -s '.[2] * {"channel_group":{"values":{"OrdererAddresses":{"value":{"addresses":.[0]}}}}} * {"channel_group":{"groups":{"Orderer":{"values":{"ConsensusType":{"value":{"metadata":{"consenters":.[1]}}}}}}}}' $WORKDIR/ord2.json $WORKDIR/ord1.json $WORKDIR/gen/config.json > $WORKDIR/gen/modified_config.json
	createConfigUpdate ${CHANNEL_NAME} $WORKDIR/gen/config.json $WORKDIR/gen/modified_config.json $WORKDIR/org_update_in_envelope.pb
}


mkdir -p $WORKDIR/gen

# 此处".data.data[0].payload.data.config"不加引号zsh报错，bash则都行
configtxlator proto_decode --input $WORKDIR/config_block.pb --type common.Block | jq ".data.data[0].payload.data.config" >$WORKDIR/gen/config.json


MODE=$1
CHANNEL_NAME=$2
MSPID=$3

if [ "${MODE}" == "addOrg" ]; then
	addOrg
elif [ "${MODE}" == "updateAnchors" ]; then
	updateAnchors
elif [ "${MODE}" == "addOrderers" ]; then
	addOrderers
else
	echo "check your args"
fi

# rm -rf gen
exit 0
