# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# syntax=docker/dockerfile:1
FROM golang:1.24 AS builder


ARG FABRIC_X_VERSION=latest
ARG REPO_PATH=github.com/hyperledger/fabric-x

# Install binaries directly from GitHub into /tmp/bin
ENV GOBIN=/tmp/bin

RUN go install ${REPO_PATH}/tools/configtxgen@$FABRIC_X_VERSION
RUN go install ${REPO_PATH}/tools/cryptogen@$FABRIC_X_VERSION
RUN go install ${REPO_PATH}/tools/configtxlator@$FABRIC_X_VERSION

FROM registry.access.redhat.com/ubi9/ubi-micro:9.5 AS prod
WORKDIR /

# Copy only the built tools
COPY --from=builder /tmp/bin/* /usr/local/bin/

CMD ["bash"]
