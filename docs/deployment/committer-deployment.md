<!-- SPDX-License-Identifier: Apache-2.0 -->
# Committer Pipeline Deployment

This guide covers deployment of the Fabric-X Committer pipeline components.

> **NOTE:** This document reflects the actual implementation. Previous versions contained significant inaccuracies including wrong ports and fictional features.

## Architecture Overview

The Committer pipeline consists of five microservices:

| Service | Port | Monitoring Port | Role |
|---------|------|-----------------|------|
| Sidecar | 4001 | 2114 | Fetches blocks from orderer, relays to coordinator, aggregates tx status |
| Coordinator | 9001 | 2119 | Coordinates signature verification, dependency tracking, validation/commit |
| Verifier | 5001 | 2115 | Parallel signature verification service |
| VC | 6001 | 2116 | Prepares, validates, and commits transactions to database |
| Query | 7001 | 2117 | State query service with batching and view aggregation |

## Build

```bash
git clone https://github.com/hyperledger/fabric-x-committer.git
cd fabric-x-committer
make build

# Binary at: ./bin/committer
```

## Database Setup

```bash
docker network create fabric-x-net

docker run -d \
  --name fabric-x-db \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=postgres \
  -p 5433:5433 \
  postgres:18.3-alpine3.23 -p 5433
```

## Startup Order

```bash
# 1. Verifier
./bin/committer start verifier -c verifier.yaml &

# 2. VC
./bin/committer start vc -c vc.yaml &

# 3. Query
./bin/committer start query -c query.yaml &

# 4. Coordinator
./bin/committer start coordinator -c coordinator.yaml &

# 5. Sidecar (last, connects to Coordinator)
./bin/committer start sidecar -c sidecar.yaml &
```

## Health Checks

All services expose `/health` on monitoring port:

```bash
# Sidecar
curl http://localhost:2114/health

# Coordinator
curl http://localhost:2119/health

# Verifier
curl http://localhost:2115/health

# VC
curl http://localhost:2116/health

# Query
curl http://localhost:2117/health
```

## Environment Variables

### Database Configuration
- `SC_VC_DATABASE_ENDPOINTS` - Database endpoints (default: localhost:5433)
- `SC_VC_DATABASE_USERNAME` - Database username (default: postgres)
- `SC_VC_DATABASE_PASSWORD` - Database password (default: postgres)
- `SC_VC_DATABASE_DATABASE` - Database name (default: postgres)
- `SC_VC_DATABASE_TLS_MODE` - TLS mode (default: none)
- `SC_QUERY_DATABASE_ENDPOINTS` - Query service database endpoints (default: localhost:5433)
- `SC_QUERY_DATABASE_USERNAME` - Query database username (default: postgres)
- `SC_QUERY_DATABASE_PASSWORD` - Query database password (default: postgres)
- `SC_QUERY_DATABASE_DATABASE` - Query database name (default: postgres)
- `SC_QUERY_DATABASE_TLS_MODE` - Query TLS mode (default: none)

### Service Endpoints
- `SC_COORDINATOR_VERIFIER_ENDPOINTS` - Verifier endpoints (default: localhost:5001)
- `SC_COORDINATOR_VALIDATOR_COMMITTER_ENDPOINTS` - VC endpoints (default: localhost:6001)
- `SC_SIDECAR_COMMITTER_ENDPOINT` - Coordinator endpoint (default: localhost:9001)
- `SC_SIDECAR_ORDERER_ORGANIZATIONS_ORG0_ENDPOINTS` - Orderer endpoints

### TLS Configuration
- `SC_COORDINATOR_SERVER_TLS_MODE` - Server TLS mode (mtls/none)
- `SC_COORDINATOR_VERIFIER_TLS_MODE` - Verifier connection TLS mode
- `SC_COORDINATOR_VALIDATOR_COMMITTER_TLS_MODE` - VC connection TLS mode
- `SC_COORDINATOR_MONITORING_TLS_MODE` - Monitoring TLS mode
- `SC_QUERY_SERVER_TLS_MODE` - Query server TLS mode
- `SC_QUERY_MONITORING_TLS_MODE` - Query monitoring TLS mode
- `SC_SIDECAR_SERVER_TLS_MODE` - Sidecar server TLS mode
- `SC_SIDECAR_MONITORING_TLS_MODE` - Sidecar monitoring TLS mode
- `SC_SIDECAR_COMMITTER_TLS_MODE` - Sidecar-Coordinator TLS mode
- `SC_SIDECAR_ORDERER_TLS_MODE` - Sidecar-Orderer TLS mode
- `SC_VC_SERVER_TLS_MODE` - VC server TLS mode
- `SC_VC_MONITORING_TLS_MODE` - VC monitoring TLS mode
- `SC_VERIFIER_SERVER_TLS_MODE` - Verifier server TLS mode
- `SC_VERIFIER_MONITORING_TLS_MODE` - Verifier monitoring TLS mode

### Logging
- Logging configured via `logging.logSpec` in YAML config files (e.g., `info:grpc=error`)

## Not Implemented

The following features do not exist:
- `committer [service] stop` command
- `committer [service] status` command
- Zero-knowledge proofs
- GPU acceleration
- Redis caching
- NATS Streaming

---

**See Also:** [Orderer Deployment](./orderer-deployment.md) | [CA Setup](./ca-setup.md)
