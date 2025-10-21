# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

FROM golang:1.24.5-bookworm AS builder

ENV GOCACHE=/go-cache \
    GOMODCACHE=/gomod-cache \
    CGO_ENABLED=0

# Change the hash to checkout a different commit / version tag. It should be the same as in app/go.mod.
RUN git clone https://github.com/hyperledger-labs/fabric-token-sdk.git && \
    cd fabric-token-sdk && \
    git checkout 300aea42b033

RUN --mount=type=cache,target=/gomod-cache \
    --mount=type=cache,target=/go-cache \
    cd fabric-token-sdk; go build -buildvcs=false -o /tcc token/services/network/fabric/tcc/main/main.go

# Final image
FROM scratch
ENV CHAINCODE_TLS=false
COPY --from=builder /tcc /usr/bin/tcc
EXPOSE 9999

ENV PUBLIC_PARAMS_FILE_PATH=/zkatdlognoghv1_pp.json
ADD zkatdlognoghv1_pp.json /zkatdlognoghv1_pp.json

CMD ["tcc"]
