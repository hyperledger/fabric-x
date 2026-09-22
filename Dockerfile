# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# syntax=docker/dockerfile:1

###########################################
# Stage 1: Build image
###########################################
FROM golang:1.27 AS builder

# No tool here links against C, so keep the builds static and cgo-free.
ENV CGO_ENABLED=0

# Args
ARG IDEMIX_VERSION=v0.0.2
ARG VERSION=1.0
ARG REVISION=1.0

WORKDIR /go/src/github.com/hyperledger/fabric-x

# Copy dependency files first (cache optimization)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Stamp the same version/commit metadata into the binaries as `make release-bins`.
ENV GO_LDFLAGS="-X github.com/hyperledger/fabric-x-common/common/metadata.Version=${VERSION} -X github.com/hyperledger/fabric-x-common/common/metadata.CommitSHA=${REVISION}"

# Build the binaries
RUN go build -ldflags "${GO_LDFLAGS}" -o /tmp/bin/configtxgen ./tools/configtxgen
RUN go build -ldflags "${GO_LDFLAGS}" -o /tmp/bin/cryptogen ./tools/cryptogen
RUN go build -ldflags "${GO_LDFLAGS}" -o /tmp/bin/configtxlator ./tools/configtxlator
RUN go build -ldflags "${GO_LDFLAGS}" -o /tmp/bin/fxconfig ./tools/fxconfig
RUN go build -ldflags "${GO_LDFLAGS}" -o /tmp/bin/fxadmin ./tools/fxadmin
RUN GOBIN=/tmp/bin go install github.com/IBM/idemix/tools/idemixgen@$IDEMIX_VERSION

###########################################
# Stage 2: Production runtime image
###########################################
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8 AS prod

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
    org.opencontainers.image.description="Fabric-X CLI tools (configtxgen, cryptogen, configtxlator, fxconfig, fxadmin) packaged in a UBI image." \
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
