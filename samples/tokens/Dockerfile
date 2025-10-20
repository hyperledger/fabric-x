# build stage
FROM golang:1.25.2 AS builder
WORKDIR /go/src/app

ARG NODE_TYPE="owner"
ARG PLATFORM="fabricx"

ENV GOCACHE=/go-cache \
    GOMODCACHE=/gomod-cache \
    CGO_ENABLED=0

COPY . .

RUN --mount=type=cache,target=/go-cache \
    --mount=type=cache,target=/gomod-cache \
    cd "${NODE_TYPE}" && go build -tags "${PLATFORM}" -o /app

FROM busybox

# API port. The FSC port must be exposed explicitly (configured in core.yaml).
EXPOSE 9000

HEALTHCHECK --start-interval=2s --start-period=10s --interval=5s --timeout=1s --retries=5 CMD nc -z 127.0.0.1 9000

WORKDIR /conf
COPY --from=builder /app /usr/bin/app

CMD ["app", "--conf", "/conf"]
