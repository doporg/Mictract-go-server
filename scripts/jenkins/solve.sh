#!/bin/bash
function parse_yaml {
   local prefix=$2
   local s='[[:space:]]*' w='[a-zA-Z0-9_]*' fs=$(echo @|tr @ '\034')
   sed -ne "s|^\($s\):|\1|" \
        -e "s|^\($s\)\($w\)$s:$s[\"']\(.*\)[\"']$s\$|\1$fs\2$fs\3|p" \
        -e "s|^\($s\)\($w\)$s:$s\(.*\)$s\$|\1$fs\2$fs\3|p"  $1 |
   awk -F$fs '{
      indent = length($1)/2;
      vname[indent] = $2;
      for (i in vname) {if (i > indent) {delete vname[i]}}
      if (length($3) > 0) {
         vn=""; for (i=0; i<indent; i++) {vn=(vn)(vname[i])("_")}
         printf("%s%s%s=\"%s\"\n", "'$prefix'",vn, $2, $3);
      }
   }'
}

JENKINS_JOB_NAME=$1
MICTRACT_URL=$2

# /var/jenkins_home/workspace/构建任务名
CC_DIR="/var/jenkins_home/workspace/${JENKINS_JOB_NAME}"

SRC_PATH=/tmp/cc/${JENKINS_JOB_NAME}/src
mkdir -p $SRC_PATH

eval $(parse_yaml ${CC_DIR}/ccconfig.yaml)

echo "testing, get out"
echo $CC_DIR
echo $SRC_PATH
echo $chaincodeinfo_nickname
echo $chaincodeinfo_label
echo $chaincodeinfo_version
echo $chaincodeinfo_sequence
echo $chaincodeinfo_policy
echo $chaincodeinfo_initRequired
echo $chaincodeinfo_channelID



cp $CC_DIR/* $SRC_PATH
cd $SRC_PATH/../
tar cfz code.tar.gz src


curl -v -F 'nickname=$chaincodeinfo_nickname' \
-F 'label=$chaincodeinfo_label' \
-F 'policy=$chaincodeinfo_policy' \
-F 'version=$chaincodeinfo_version' \
-F 'sequence=$chaincodeinfo_sequence' \
-F 'initRequired=$chaincodeinfo_initRequired' \
-F 'channelID=${chaincodeinfo_channelID}' \
-F 'file="@code.tar.gz"' $MICTRACT_URL/api/chaincode

echo "done!"