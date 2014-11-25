#!/bin/bash

export GOPATH=`pwd`
PACKAGES_DIR=packages/

rm -rf $PACKAGES_DIR/*

for OS in linux; do
    for ARCH in amd64 386; do
        BIN_PATH="$PACKAGES_DIR/${OS}_${ARCH}/scalarm_simulation_manager"
        echo "Building: $OS $ARCH in ${BIN_PATH}..."
        GOOS=$OS GOARCH=$ARCH CGO_ENABLED=0 go build -o $PACKAGES_DIR/${OS}_${ARCH}/scalarm_simulation_manager scalarm_simulation_manager
        strip $BIN_PATH
        xz $BIN_PATH
    done
done
