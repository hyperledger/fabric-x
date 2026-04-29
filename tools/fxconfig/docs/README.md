<!--
SPDX-License-Identifier: Apache-2.0
-->

# fxconfig - Fabric-X Namespace Management CLI

`fxconfig` is a command-line tool for managing Fabric-X namespaces. It enables creating, updating, and querying namespaces with flexible endorsement policies.

## Quick Start

```bash
# Show version
fxconfig version

# Display effective configuration
fxconfig info

# List all namespaces
fxconfig namespace list

# Create a namespace (single command with endorse and submit)
fxconfig namespace create payments \
  --policy="AND('Org1MSP.member', 'Org2MSP.member')" \
  --endorse --submit --wait
```

## Commands

### Namespace Management

```bash
# Create namespace
fxconfig namespace create <name> [flags]

# Update namespace
fxconfig namespace update <name> [flags]

# List namespaces
fxconfig namespace list [flags]
```

**Common Flags:**
- `--policy=<DSL>` - Endorsement policy DSL string
- `--policy=threshold:<path>` - Threshold ECDSA policy from PEM file
- `--version=<int>` - Version number (update only; create defaults to 0)
- `--output=<path>` - Save transaction to file (`.json` extension)
- `--endorse` - Sign transaction with local MSP identity
- `--submit` - Submit endorsed transaction to ordering service
- `--wait` - Block until transaction commits (requires notification service)

**Policy Examples:**
- Single org: `--policy="OR('Org1MSP.member')"`
- Multi org: `--policy="AND('Org1MSP.member', 'Org2MSP.member')"`
- Complex: `--policy="OutOf(1, 'Org1MSP.member', 'Org2MSP.member')"`
- Threshold ECDSA: `--policy="threshold:/path/to/policy.pem"`

### Transaction Operations

```bash
# Endorse a transaction
fxconfig tx endorse <path> --output=<path>

# Merge multiple endorsed transactions
fxconfig tx merge <path1> <path2> ... --output=<path>

# Submit transaction to ordering service
fxconfig tx submit <path> [--wait]
```

### Utility Commands

```bash
# Show version information
fxconfig version

# Display effective configuration
fxconfig info
```

## Configuration

Configuration is loaded from multiple sources with the following precedence (highest to lowest):

1. **Environment variables** (e.g., `FXCONFIG_ORDERER_ADDRESS=localhost:7050`)
2. **Config file via --config flag** (e.g., `--config=/path/to/config.yaml`)
3. **Project config** (`.fxconfig/config.yaml`)
4. **User config** (`~/.fxconfig/config.yaml`)

### Configuration File Example

```yaml
# MSP identity for endorsing transactions
msp:
  localMspID: Org1MSP
  configPath: /path/to/msp

# Logging configuration
logging:
  level: ERROR  # Default level for all loggers
  format: "%{color}%{time:2006-01-02 15:04:05.000 MST} [%{module}] %{shortfunc} -> %{level:.4s} %{id:03x}%{color:reset} %{message}"

# Parent TLS configuration (can be overridden per service)
tls:
  enabled: true
  clientKey: /path/to/client.key
  clientCert: /path/to/client.crt
  rootCerts:
    - /path/to/ca.crt
  serverNameOverride: ""  # Optional: Override SNI hostname for IP-based connections

# Ordering service configuration
orderer:
  address: localhost:7050
  channel: mychannel  # Channel name (default: mychannel)
  connectionTimeout: 30s
  # Optional: Override parent TLS settings
  tls:
    enabled: true
    rootCerts:
      - /path/to/orderer-ca.crt
    clientCert: /path/to/orderer-client.crt
    clientKey: /path/to/orderer-client.key

# Query service configuration
queries:
  address: localhost:7001
  connectionTimeout: 30s
  # Optional: Override parent TLS settings
  tls:
    enabled: true
    rootCerts:
      - /path/to/peer-ca.crt

# Notification service configuration
notifications:
  address: localhost:7001
  connectionTimeout: 30s
  waitingTimeout: 30s
  # Optional: Override parent TLS settings
  tls:
    enabled: false
```

> **TLS is enabled by default (secure-by-default).** Each service (`orderer`, `queries`, `notifications`) must therefore either:
> - provide at least `tls.rootCerts` (server-side TLS) — and additionally `tls.clientCert` / `tls.clientKey` for mutual TLS; or
> - explicitly opt out with `tls.enabled: false`.
>
> A service with `tls.enabled: true` but no `rootCerts` will fail to load with `rootCertPaths must not be empty`.
>
> **Breaking change (v0.4.0+):** prior versions defaulted to `tls.enabled: false`. Configs upgraded from older versions that omit the `tls` section will now attempt TLS — set `tls.enabled: false` per service to keep the previous plaintext behavior.

### TLS Configuration

- **No TLS**: `enabled: false` or all TLS fields empty
- **Server TLS**: `enabled: true` with only `rootCerts` set (server authentication only)
- **Mutual TLS**: `enabled: true` with `clientKey`, `clientCert`, and `rootCerts` all set (mutual authentication)
- **Service-specific TLS**: Each service (orderer, queries, notifications) can override the parent `tls` section
- **SNI Override**: Use `serverNameOverride` for IP-based connections or custom hostname verification

### Naming Conventions

- **Environment Variables**: SCREAMING_SNAKE_CASE with `FXCONFIG_` prefix (e.g., `FXCONFIG_MSP_CONFIGPATH`, `FXCONFIG_ORDERER_ADDRESS`)
- **Config File Fields**: camelCase (e.g., `configPath`, `connectionTimeout`)
- **Array Parameters**: Comma-separated values in environment variables (e.g., `FXCONFIG_TLS_ROOTCERTS="/path1/cert.pem,/path2/cert.pem"`)

### Environment Variables

Override any configuration parameter using environment variables:

```bash
# Format: FXCONFIG_<SECTION>_<PARAMETER>
export FXCONFIG_MSP_LOCALMSPID=Org1MSP
export FXCONFIG_MSP_CONFIGPATH=/opt/msp
export FXCONFIG_ORDERER_ADDRESS=orderer.example.com:7050
export FXCONFIG_QUERIES_ADDRESS=query.example.com:7001
export FXCONFIG_TLS_ROOTCERTS="/path1/cert.pem,/path2/cert.pem"
```

## Usage Examples

### Single-Org Simple Flow

```bash
# Create, endorse, submit, and wait in one command
fxconfig namespace create hello \
  --policy="AND('Org1MSP.member')" \
  --endorse --submit --wait
```

### Multi-Org Local Development

```bash
# Create transaction
fxconfig namespace create hello \
  --policy="AND('Org1MSP.member', 'Org2MSP.member')" \
  --output=hello.json

# Endorse with Org1
fxconfig tx endorse hello.json \
  --config=org1_config.yaml \
  --output=hello_org1.json

# Endorse with Org2 (appends to existing endorsements)
fxconfig tx endorse hello_org1.json \
  --config=org2_config.yaml \
  --output=hello_endorsed.json

# Submit to network
fxconfig tx submit hello_endorsed.json
```

### Multi-Org Distributed Flow

```bash
# Query current state
fxconfig namespace list
# Output:
# Installed namespaces (1 total):
# 0) hello: version 0 policy: <hex-encoded-policy>

# Org1: Create transaction
fxconfig namespace create hello \
  --policy="AND('Org1MSP.member', 'Org2MSP.member')" \
  --output=hello.json

# Send hello.json to Org1 and Org2 via external secure channel

# Org1: Endorse
fxconfig tx endorse hello.json \
  --config=org1_config.yaml \
  --output=hello_org1.json

# Org2: Endorse (receives hello.json via external channel)
fxconfig tx endorse hello.json \
  --config=org2_config.yaml \
  --output=hello_org2.json

# Collect endorsed transactions from org1 and org2

# Either org: Merge endorsements
fxconfig tx merge \
  hello_org1.json hello_org2.json \
  --output=hello_endorsed.json

# Either org: Submit to network
fxconfig tx submit hello_endorsed.json
```

### Update Namespace

```bash
# First, list to get current version
fxconfig namespace list

# Update with new version
fxconfig namespace update payments \
  --policy="AND('Org1MSP.member', 'Org2MSP.member')" \
  --version 1 \
  --endorse --submit --wait
```

### Configuration Management

```bash
# Use specific config file
fxconfig namespace create hello \
  --config=/path/to/org1-config.yaml \
  --policy="OR('Org1MSP.member')" \
  --endorse --submit

# Check effective configuration
fxconfig info

# Override via environment variables
export FXCONFIG_ORDERER_ADDRESS=orderer.example.com:7050
export FXCONFIG_MSP_LOCALMSPID=Org1MSP
fxconfig namespace create hello --policy="OR('Org1MSP.member')" --endorse --submit
```

### Multi-Organization Setup

Each organization can use its own configuration:

```bash
# Org1 configuration
cat > org1-config.yaml <<EOF
msp:
  localMspID: Org1MSP
  configPath: /opt/org1/msp
orderer:
  address: orderer.example.com:7050
  tls:
    clientKey: /opt/org1/tls/client.key
    clientCert: /opt/org1/tls/client.crt
    rootCerts:
      - /opt/tls/ca.crt
EOF

# Org2 configuration
cat > org2-config.yaml <<EOF
msp:
  localMspID: Org2MSP
  configPath: /opt/org2/msp
orderer:
  address: orderer.example.com:7050
  tls:
    clientKey: /opt/org2/tls/client.key
    clientCert: /opt/org2/tls/client.crt
    rootCerts:
      - /opt/tls/ca.crt
EOF

# Each org can now use their own config
fxconfig namespace create payments \
  --config org1-config.yaml \
  --policy="AND('Org1MSP.member', 'Org2MSP.member')" \
  --output=payments.json

# Org1 endorses
fxconfig tx endorse payments.json \
  --config org1-config.yaml \
  --output=payments_org1.json

# Org2 endorses
fxconfig tx endorse payments.json \
  --config org2-config.yaml \
  --output=payments_org2.json

# Merge and submit
fxconfig tx merge \
  payments_org1.json payments_org2.json \
  --output=payments_endorsed.json

fxconfig tx submit \
  payments_endorsed.json \
  --config org1-config.yaml
```

## Exit Codes

- `0` - Success
- `1` - Error (validation failure, connection error, etc.)

## Error Handling

All errors are written to stderr with descriptive messages:

```bash
# Example error output
Error: namespace name must be specified
Error: policy must be specified
Error: invalid namespace name
Error: connection timeout
```

## Help

Get help for any command:

```bash
fxconfig --help                    # Global help
fxconfig namespace --help          # Namespace commands help
fxconfig namespace create --help   # Create command help
fxconfig namespace update --help   # Update command help
fxconfig tx --help                 # Transaction commands help
fxconfig tx endorse --help         # Endorse command help
fxconfig tx merge --help           # Merge command help
fxconfig tx submit --help          # Submit command help
```

## Troubleshooting

### Check Configuration

```bash
# Display effective configuration
fxconfig info

# Test connectivity
fxconfig namespace list
```

### Common Issues

**Connection timeout:**

```bash
# Increase timeout
FXCONFIG_QUERIES_CONNECTIONTIMEOUT=60s fxconfig namespace list
```

**TLS errors:**

```bash
# Verify TLS configuration
fxconfig info | grep -i tls

# Check certificate paths exist
ls -la /path/to/tls/
```

**MSP errors:**

```bash
# Verify MSP configuration
ls -la /path/to/msp/
fxconfig info | grep -i msp
```

**Policy validation errors:**

```bash
# Ensure policy syntax is correct
# Single org: --policy="OR('Org1MSP.member')"
# Multi org: --policy="AND('Org1MSP.member', 'Org2MSP.member')"
# Complex: --policy="OutOf(2, 'Org1MSP.member', 'Org2MSP.member', 'Org3MSP.member')"
```

## Additional Resources

- [Fabric-X Documentation](https://github.com/hyperledger/fabric-x)