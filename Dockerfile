# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# syntax=docker/dockerfile:1

###########################################
# Stage 1: Build image
###########################################
FROM golang:1.25 AS builder

# List of CLI tools to build
ARG FABRICX_TOOLS="configtxgen cryptogen configtxlator fxconfig"

# Build environment variables
ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE"

WORKDIR /go/src/github.com/hyperledger/fabric-x

# Copy dependency files first (cache optimization)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binaries
RUN mkdir -p /tmp/bin && \
    for tool in $FABRICX_TOOLS; do \
    go build -o /tmp/bin/$tool ./tools/$tool; \
    done

###########################################
# Stage 2: Production runtime image
###########################################
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.6 AS prod

ARG VERSION=1.0
ARG CREATED
ARG REVISION=1.0

# Add non-root user (UID 10001) without installing extra packages
RUN /usr/sbin/useradd -u 10001 -r -g root -s /sbin/nologin \
    -c "Fabric-X tools user" fabricx && \
    mkdir -p /home/fabricx && \
    chown -R 10001:0 /home/fabricx && \
    chmod 0755 /home/fabricx

# Copy only the built tools
COPY --from=builder /tmp/bin/* /usr/local/bin/

# OCI metadata labels
LABEL org.opencontainers.image.created="${CREATED}" \
    org.opencontainers.image.description="Fabric-X CLI tools (configtxgen, cryptogen, configtxlator, fxconfig) packaged in a UBI image." \
    org.opencontainers.image.licenses="Apache-2.0" \
    org.opencontainers.image.ref.name="ubi9/ubi-minimal" \
    org.opencontainers.image.revision="${REVISION}" \
    org.opencontainers.image.source="https://github.com/hyperledger/fabric-x" \
    org.opencontainers.image.title="fabric-x" \
    org.opencontainers.image.url="https://github.com/hyperledger/fabric-x" \
    org.opencontainers.image.version="${VERSION}"

# Use non-root user
USER 10001
WORKDIR /home/fabricx

# Define default CMD
CMD ["/bin/sh"]
