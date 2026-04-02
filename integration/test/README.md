<!--
SPDX-License-Identifier: Apache-2.0
-->
# E2E Integration Test

End-to-end integration test for the Fabric-X transaction pipeline. Exercises the full lifecycle:

```
Loadgen → Arma Routers (BFT broadcast)
    → Arma Batchers (4 parties, 1 shard)
    → Arma Consenters (4 parties, SmartBFT)
    → Arma Assemblers → Committer Sidecar (BFT block delivery)
    → Coordinator → Verifier → Validator-Committer (DB commit)
```

## Prerequisites

- Docker (with compose plugin)
- `cryptogen`, `configtxgen`, and `fxconfig` built from this repo (`make tools` from fabric-x root)
- `curl` and `nc` (netcat) on the host
- Docker images: `arma-4p1s`, `committer-test-node`, `fabric-x-loadgen`

## Quick Start

```bash
# From fabric-x root: build host binaries (cryptogen, configtxgen, fxconfig)
make tools

cd integration/test

# Build/resolve orderer + committer images and tools
./build-e2e.sh

# Run the test
./run-e2e.sh
```

## Scripts

### `build-e2e.sh`

Builds the orderer (`arma-4p1s`) and committer (`committer-test-node`) Docker images needed by `run-e2e.sh`. Also clones fabric-x at a specific ref to build host tools (`cryptogen`, `configtxgen`, `fxconfig`).

Image resolution strategy per component:

1. **Pre-built pull** — for release tags (`vX.Y.Z`), tries `docker.io/hyperledger/` first.
2. **Local build** — clones the component repo at the specified ref and builds locally.

All refs and image names default to values in `refs.conf` and can be overridden via CLI flags.

```bash
# Build using refs.conf defaults
./build-e2e.sh

# Build committer from a specific tag (pulls from registry if available)
./build-e2e.sh --committer-ref=v1.2.3

# Build orderer from a specific commit hash
./build-e2e.sh --orderer-ref=abc123

# Build both from refs
./build-e2e.sh --orderer-ref=main --committer-ref=my-feature-branch

# Use a custom repo URL (e.g., a fork)
./build-e2e.sh --committer-repo=https://github.com/myorg/fabric-x-committer.git --committer-ref=my-branch

# Then run the test with the built images
export ORDERER_IMAGE=localhost/arma-4p1s
export COMMITTER_IMAGE=docker.io/hyperledger/committer-test-node
./run-e2e.sh
```

**Options:**

| Flag | Description |
|---|---|
| `--fabric-x-ref=REF` | Tag, branch, or commit for fabric-x tools |
| `--committer-ref=REF` | Tag, branch, or commit for fabric-x-committer |
| `--orderer-ref=REF` | Tag, branch, or commit for fabric-x-orderer |
| `--fabric-x-repo=URL` | Override default fabric-x GitHub repo URL |
| `--committer-repo=URL` | Override default committer GitHub repo URL |
| `--orderer-repo=URL` | Override default orderer GitHub repo URL |

On completion, the script prints `export` commands for `ORDERER_IMAGE` and `COMMITTER_IMAGE` that you can pass to `run-e2e.sh`. In GitHub Actions, the resolved image names are also written to `GITHUB_OUTPUT`.

### `run-e2e.sh`

Runs the full E2E test. Generates all artifacts on the host, starts containers, runs the loadgen, and verifies that >= 5000 transactions were committed.

```bash
# Use default images
./run-e2e.sh

# Override images
ORDERER_IMAGE=localhost/arma-4p1s:mybranch ./run-e2e.sh
COMMITTER_IMAGE=docker.io/hyperledger/committer-test-node:v1.0.0 ./run-e2e.sh

# Override all
ORDERER_IMAGE=... COMMITTER_IMAGE=... LOADGEN_IMAGE=... ./run-e2e.sh
```

**Environment variables:**

| Variable | Default | Description |
|---|---|---|
| `ORDERER_IMAGE` | `localhost/arma-4p1s` | Arma 4-party-1-shard Docker image |
| `COMMITTER_IMAGE` | `docker.io/hyperledger/committer-test-node` | Committer test node image |
| `LOADGEN_IMAGE` | `docker.io/hyperledger/fabric-x-loadgen` | Load generator image |
| `FABRIC_X_BIN` | `../../bin` | Path to `cryptogen`, `configtxgen`, and `fxconfig` |

**Steps performed:**

1. Generate crypto material (`cryptogen`)
2. Generate Arma shared config protobuf (`armageddon`, runs inside orderer image)
3. Generate orderer local configs from templates
4. Generate channel genesis block (`configtxgen`)
5. Start Arma orderer and committer containers
6. Wait for health (router port 6022, batcher port 6024, sidecar port 4001)
7. Create namespace using `fxconfig` with multi-org endorsement (both peer-org-0 and peer-org-1 sign)
8. Run loadgen (submits ~10,000 TXs)
9. Verify >= 5000 committed transactions via VC Prometheus metrics
10. Cleanup

### `clean.sh`

Removes all containers, volumes, networks, and generated artifacts. Resolves symlinks (e.g., `/tmp` → `/private/tmp` on macOS) to ensure the actual directories are cleaned. Safe to run multiple times.

```bash
./clean.sh
```

### `refs.conf`

Centralized configuration file for default component refs and Docker image names. Sourced by `build-e2e.sh` and the GitHub Actions workflow. All values can be overridden via CLI arguments or environment variables.

## Running `.github/workflows/e2e.yml` manually

You can run the GitHub Actions workflow from github.com or with the GitHub CLI.

### From github.com

1. Open: `https://github.com/hyperledger/fabric-x/actions/workflows/e2e.yml`
2. Click **Run workflow**.
3. Select the branch that contains your workflow/code changes.
4. Enter `fabric-x-ref`, `orderer-ref`, and `committer-ref` (or leave empty for refs.conf defaults).
5. Click **Run workflow**.

### Workflow inputs

| Input | Meaning |
|---|---|
| `fabric-x-ref` | Ref for `fabric-x` tools (cryptogen, configtxgen, fxconfig). `""` = refs.conf default |
| `orderer-ref` | Ref for `fabric-x-orderer` (`""` = refs.conf default, commit hash = build from source, tag = pull from registry) |
| `committer-ref` | Ref for `fabric-x-committer` (`""` = refs.conf default, commit hash = build from source, tag = pull from registry) |

Notes:

- The workflow delegates orderer + committer image resolution/build and tools build to `integration/test/build-e2e.sh`.
- `build-e2e.sh` writes resolved image names to `GITHUB_OUTPUT` for downstream workflow steps.
- The loadgen image tag matches the committer-ref (explicit or from refs.conf).
- The fabric-x tools (cryptogen, configtxgen, fxconfig) are built at the specified `fabric-x-ref` and placed in `integration/test/.build/fabric-x/bin`.

### GitHub CLI examples

```bash
# 1) Default (refs.conf defaults for all components)
gh workflow run e2e.yml

# 2) Build all components from commit hashes
gh workflow run e2e.yml \
  -f fabric-x-ref=abc123 \
  -f orderer-ref=f1dfdd7b4e3c5d9ff6c1843da0bf78155262d4f2 \
  -f committer-ref=d35655cc

# 3) Pull all by release tags
gh workflow run e2e.yml \
  -f fabric-x-ref=v1.0.0 \
  -f orderer-ref=v1.2.3 \
  -f committer-ref=v1.2.3

# 4) Pull all by branch names
gh workflow run e2e.yml \
  -f fabric-x-ref=main \
  -f orderer-ref=main \
  -f committer-ref=feature/e2e-fix

# 5) Mixed: tools from tag, orderer from commit (build), committer from tag (pull)
gh workflow run e2e.yml \
  -f fabric-x-ref=v1.0.0 \
  -f orderer-ref=f1dfdd7b4e3c5d9ff6c1843da0bf78155262d4f2 \
  -f committer-ref=v1.2.3

# 6) Test with specific fabric-x tools version only
gh workflow run e2e.yml \
  -f fabric-x-ref=v1.1.0
```

To inspect runs:

```bash
gh run list --workflow e2e.yml
gh run watch
```

## Directory Structure

```
integration/test/
├── run-e2e.sh                  # Main test script
├── build-e2e.sh                # Image build helper
├── clean.sh                    # Cleanup script
├── refs.conf                   # Default refs and image names
├── docker-compose.yaml         # Container orchestration (arma + committer + loadgen)
├── fxconfig-peer-org-0.yaml    # fxconfig config for peer-org-0 (namespace endorsement)
├── fxconfig-peer-org-1.yaml    # fxconfig config for peer-org-1 (namespace endorsement)
├── loadgen.yaml                # Load generator configuration
├── .gitignore                  # Ignores generated directories
├── networkconfig/              # Channel and network definitions
│   ├── configtx.yaml           #   Channel config (orgs, policies, capabilities, consenter mapping)
│   ├── arma_config.yaml        #   Arma topology (parties, roles, endpoints, certs)
│   └── crypto-config.yaml      #   Cryptogen input (org structure, node types)
├── ordererconfig/              # Orderer local config templates
│   ├── base.yaml.tpl           #   Shared config (TLS, MSP, bootstrap, storage)
│   ├── role_router.yaml        #   Router-specific settings
│   ├── role_assembler.yaml     #   Assembler-specific settings
│   ├── role_batcher.yaml       #   Batcher-specific settings
│   └── role_consenter.yaml     #   Consenter-specific settings
└── committerconfig/            # Committer service configs (mounted as-is)
    ├── sidecar.yaml            #   Block delivery from Arma, BFT verification
    ├── coordinator.yaml        #   Validation pipeline orchestration
    ├── verifier.yaml           #   Transaction signature verification
    ├── vc.yaml                 #   Read-set validation + DB commit
    └── query.yaml              #   Read-only state access
```

### Generated at runtime (git-ignored)

```
/tmp/fabric-x-test/
├── artifacts/                  # Crypto material, configs, genesis block
│   ├── ordererOrganizations/   # 4 orderer org certs/keys
│   ├── peerOrganizations/      # 2 peer org certs/keys
│   ├── config/                 # Generated orderer local configs (party1-4)
│   ├── bootstrap/              # Arma shared config protobuf
│   ├── fxconfig-tx/            # fxconfig transaction files (endorsement pipeline)
│   └── config-block.pb.bin     # Channel genesis block
└── arma-storage/               # Orderer ledger data (party1-4 x 4 roles)
```

## Network Topology

The test runs a 4-party Arma orderer with 1 shard. All 16 orderer processes (4 parties x 4 roles) run in a single container.

**Port scheme** (per party, offset = (partyID - 1) * 100):

| Role | Party 1 | Party 2 | Party 3 | Party 4 | Purpose |
|---|---|---|---|---|---|
| Router | 6022 | 6122 | 6222 | 6322 | Client broadcast (loadgen → orderer) |
| Assembler | 6023 | 6123 | 6223 | 6323 | Block delivery (orderer → sidecar) |
| Batcher | 6024 | 6124 | 6224 | 6324 | Internal: TX batching |
| Consenter | 6025 | 6125 | 6225 | 6325 | Internal: SmartBFT consensus |

**Committer ports:**

| Port | Service | Purpose |
|---|---|---|
| 4001 | Sidecar | Block delivery to loadgen |
| 7001 | Coordinator | gRPC API |
| 2114 | Sidecar | Prometheus metrics |
| 2116 | VC | Prometheus metrics (used for verification) |
