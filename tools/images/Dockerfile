
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# syntax=docker/dockerfile:1
FROM golang:1.24 AS builder

ARG FABRIC_X_VERSION=latest

ENV GOBIN=/tmp/bin

# install tools
RUN go install github.com/hyperledger/fabric-x/tools/configtxgen@$FABRIC_X_VERSION
RUN go install github.com/hyperledger/fabric-x/tools/cryptogen@$FABRIC_X_VERSION
RUN go install github.com/hyperledger/fabric-x/tools/configtxlator@$FABRIC_X_VERSION

FROM registry.access.redhat.com/ubi9/ubi-micro:9.5 AS prod

WORKDIR /
# copy them all
COPY --from=builder /tmp/bin/* /usr/local/bin/

CMD ["bash"]