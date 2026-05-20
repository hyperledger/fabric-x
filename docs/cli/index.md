<!-- SPDX-License-Identifier: Apache-2.0 -->
# Fabric-X CLI Reference

The Fabric-X Command-Line Interface (CLI) provides tools for managing, deploying, and operating Fabric-X networks.

## Overview

Fabric-X includes the following CLI tools:

| Tool | Purpose | Primary Users |
|------|---------|---------------|
| **fxconfig** | Namespace management and transaction submission | Network operators, application developers |
| **arma** | Launch Arma ordering service nodes | Network operators, ordering service admins |
| **armageddon** | Configuration generation and testing for Arma | Network operators, testers |
| **committer** | Launch committer pipeline services | Peer operators, network admins |
| **configtxgen** | Generate configuration transactions | Network architects, deployment engineers |
| **configtxlator** | Convert between config formats | Network administrators |
| **cryptogen** | Generate cryptographic material | Security teams, PKI administrators |

### Go Version Requirements

| Component | Go Version |
|-----------|------------|
| fabric-x | 1.26+ |
| fabric-x-committer | 1.26+ |
| fabric-x-orderer | 1.25.5+ |
| fabric-x-common | 1.25.5+ |

## Command Structure

### Simple Execution Commands

Tools like `arma` and `committer` launch services directly:

```bash
# Arma - launch ordering service nodes
arma router --config router.yaml
arma batcher --config batcher.yaml

# Committer - launch pipeline services
committer start sidecar --config sidecar.yaml
committer start coordinator --config coordinator.yaml
```

### Hierarchical Commands

Tools like `fxconfig` use subcommand structure:

```bash
fxconfig <command> [subcommand] [flags] [arguments]
```

**Example:**

```bash
fxconfig namespace create mynamespace --channel mychannel --mspConfigPath ./msp
```

## Tool Reference

### fxconfig

Namespace and transaction management tool.

**Commands:**

- `namespace create` - Create new namespace
- `namespace list` - List installed namespaces
- `namespace update` - Update namespace
- `tx endorse` - Endorse transaction
- `tx merge` - Merge endorsed transactions
- `tx submit` - Submit transaction to orderer
- `version` - Show version

**Example:**

```bash
# Create namespace
fxconfig namespace create myns --channel mychannel --mspConfigPath ./msp

# Endorse and submit transaction
fxconfig namespace create myns --channel mychannel --output tx.json
fxconfig tx endorse tx.json --output endorsed.json
fxconfig tx submit endorsed.json --wait
```

See [fxconfig](fxconfig.md) for detailed reference.

### arma

Launch Arma ordering service nodes. Each command starts a specific node type.

**Commands:**

- `router` - Launch Router node
- `batcher` - Launch Batcher node
- `consensus` - Launch Consensus node
- `assembler` - Launch Assembler node

**Common Flags:**

| Flag | Description | Required |
|------|-------------|----------|
| `--config` | Path to configuration file | Yes |

**Example:**

```bash
# Terminal 1: Start Router
arma router --config router.yaml

# Terminal 2: Start Batcher
arma batcher --config batcher.yaml

# Terminal 3: Start Consensus
arma consensus --config consensus.yaml

# Terminal 4: Start Assembler
arma assembler --config assembler.yaml
```

See [arma](../orderer/docs/cli/arma.md) for detailed reference.

### armageddon

Configuration generation and testing utility for Arma.

**Commands:**

- `generate` - Generate crypto and config material
- `showtemplate` - Display default configuration template
- `submit` - Submit transactions and verify
- `load` - Load test transactions
- `receive` - Pull blocks and collect statistics
- `createSharedConfigProto` - Create shared config binary
- `createBlock` - Create genesis block
- `version` - Show version

**Example:**

```bash
# Generate configuration
armageddon generate --output ./my-network

# Submit test transactions
armageddon submit --config user_config.yaml --transactions 10000 --rate 1000

# Show default template
armageddon showtemplate > network-template.yaml
```

See [armageddon](../orderer/docs/cli/armageddon.md) for detailed reference.

### committer

Launch committer pipeline services.

**Commands:**

- `start <service>` - Start a service
- `healthcheck <service>` - Check service health
- `version` - Show version

**Services:**

- `sidecar` - Transaction ingress and ledger commit
- `coordinator` - Pipeline orchestration
- `vc` - Validator-Committer (VSCC execution)
- `verifier` - Signature and syntax validation
- `query` - State database queries

**Common Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `-c, --config` | Config file path | `config.yaml` |
| `-p, --pprof-address` | pprof server address | "" (disabled) |

**Example:**

```bash
# Start coordinator first
committer start coordinator -c coordinator.yaml

# Start other services
committer start vc -c vc.yaml
committer start verifier -c verifier.yaml
committer start query -c query.yaml
committer start sidecar -c sidecar.yaml

# Check health
committer healthcheck coordinator
```

See [committer](../committer/docs/cli/committer.md) for detailed reference.

### configtxgen

Generate channel configuration transactions and genesis blocks.

**Key Flags:**

| Flag | Description |
|------|-------------|
| `-profile` | Profile from configtx.yaml |
| `-channelID` | Channel identifier |
| `-outputBlock` | Output path for genesis block |
| `-outputCreateChannelTx` | Output path for channel tx |
| `-configPath` | Path to configtx.yaml directory |

**Example:**

```bash
export FABRIC_CFG_PATH=$PWD

# Generate genesis block
configtxgen -profile GenesisProfile -channelID system-channel -outputBlock genesis.block

# Generate channel creation tx
configtxgen -profile ChannelProfile -channelID mychannel -outputCreateChannelTx channel.tx
```

See [configtxgen](configtxgen.md) for detailed reference.

### configtxlator

Convert between configuration protobuf and JSON/YAML formats.

**Note:** Documentation coming soon. For now, run `configtxlator --help` for available commands.

### cryptogen

Generate cryptographic material for Fabric-X networks.

**Commands:**

- `generate` - Generate crypto material
- `extend` - Extend existing crypto material
- `showtemplate` - Display default template
- `verify` - Verify generated material

**Example:**

```bash
# Generate from template
cryptogen generate --config crypto-config.yaml --output crypto-config

# Show template
cryptogen showtemplate

# Extend with new organization
cryptogen extend --input crypto-config --config new-org.yaml
```

See [cryptogen](cryptogen.md) for detailed reference.

## Getting Help

Each tool supports `--help` flag:

```bash
fxconfig --help
fxconfig namespace create --help
arma --help
committer --help
armageddon --help
configtxgen --help
cryptogen --help
```

## Environment Variables

| Variable | Tool | Description |
|----------|------|-------------|
| `FABRIC_CFG_PATH` | configtxgen, cryptogen | Path to configuration directory |
| `FABRIC_LOGGING_SPEC` | All | Logging level specification |

## Next Steps

- **[fxconfig](fxconfig.md)** - Namespace and transaction management
- **[arma](../orderer/docs/cli/arma.md)** - Arma ordering service launcher
- **[armageddon](../orderer/docs/cli/armageddon.md)** - Configuration generation and testing
- **[committer](../committer/docs/cli/committer.md)** - Committer pipeline launcher
- **[configtxgen](configtxgen.md)** - Configuration transaction generation
- **[cryptogen](cryptogen.md)** - Cryptographic material generation

---

**Version:** 1.0  
**Last Updated:** 2026-04-22
