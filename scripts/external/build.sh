#!/bin/bash

OUTPUT=$1
CCPATH=$2

cd $CCPATH
GOPROXY="https://goproxy.cn" GO111MODULE=on go build -o $OUTPUT -v .
