# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# syntax=docker/dockerfile:1

###########################################
# Stage 1: Build image
###########################################
FROM public.ecr.aws/docker/library/golang:1.24 AS builder

# Build environment variables
ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE"
ENV GOBIN=/tmp/bin

ARG FABRIC_X_VERSION=latest
ARG ARMAGEDDON_VERSION=latest

WORKDIR /go/src/github.com/hyperledger/fabric-x-tools

# Install Fabric-X tooling
RUN go install github.com/hyperledger/fabric-x/tools/configtxgen@${FABRIC_X_VERSION} && \
    go install github.com/hyperledger/fabric-x/tools/cryptogen@${FABRIC_X_VERSION} && \
    go install github.com/hyperledger/fabric-x/tools/configtxlator@${FABRIC_X_VERSION} && \
    go install github.com/hyperledger/fabric-x/tools/fxconfig@${FABRIC_X_VERSION} && \
    go install github.com/hyperledger/fabric-x-orderer/cmd/armageddon@${ARMAGEDDON_VERSION}


###########################################
# Stage 2: Production runtime image
###########################################
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.6 AS prod

ARG VERSION=1.0

# Add non-root user (UID 10001) without installing extra packages
RUN /usr/sbin/useradd -u 10001 -r -g root -s /sbin/nologin \
        -c "Fabric-X tools user" fabricx && \
    mkdir -p /home/fabricx && \
    chown -R 10001:0 /home/fabricx && \
    chmod 0755 /home/fabricx

# Copy only the built tools
COPY --from=builder /tmp/bin/* /usr/local/bin/

# OCI metadata labels
LABEL name="fabric-x-tools" \
    maintainer="IBM Research ZRL Decentralized Trust Group" \
    version="${VERSION}" \
    description="Fabric-X CLI tools (configtxgen, cryptogen, configtxlator, fxconfig, armageddon) packaged in a UBI image" \
    license="Apache-2.0" \
    vendor="IBM Research"

# Use non-root user
USER 10001
WORKDIR /home/fabricx

# Default Entrypoint
ENTRYPOINT ["/bin/sh"]
