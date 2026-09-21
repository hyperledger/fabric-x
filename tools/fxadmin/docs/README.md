<!--
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
-->

# fxadmin — Fabric-X Reconfiguration Admin CLI

`fxadmin` is a command-line tool for **administering the configuration of
Fabric-X**. It lets administrators pull the
current configuration from the running network, edit it, collect the required
admin endorsements, wrap the result into a submittable transaction, and
broadcast that transaction to the ordering service.

---

## The reconfiguration flow
A configuration change is produced by the following steps:

**Manual mode**
1. Get the last config block from the orderer.
2. Decode the block to json.
3. Edit the json manually.
4. Compute the config update from the current config json and the modified json.
5. Sign a config-update as admin endorser, typically more than one endorser.
6. Merge admin signatures.
7. Prepare the TX proposal for submission by wrapping it in an envelope and signing it with a Channel Writer principal (can be one of the admins).
8. Submit the config-update to all routers.
9. Follow the assembler ledger until the new config is committed in all the parties.

NOTE: Steps 2–4 describe the manual process for building a configuration change.
For the most common changes - such as endpoints, node certificates, CA lists, 
and adding or removing organizations and parties - use the [Modify Commands](#modify-commands-automation-of-the-manual-step) instead. 
These commands edit the config block directly and allow multiple changes to be accumulated into a single update.
The manual process remains fully supported; the modify commands simply automate the common cases.

**Auto mode**
1. Get the last config block from the orderer.
2. Copy the last config block to "next" block.
3. Apply one or more modifications to next block.
4. Compute the config update from the current config block and the next block.
5. Sign a config-update as admin endorser, typically more than one endorser.
6. Merge admin signatures.
7. Prepare the TX proposal for submission by wrapping it in an envelope and signing it with a Channel Writer principal (can be one of the admins).
8. Submit the config-update to all routers.
9. Follow the assembler ledger until the new config is committed in all the parties.

---

## Prerequisites

- An **admin identity** for at least one organization (MSP directory).
- The admin's TLS client certificate/key and whether TLS is enabled.
- The orderer TLS root CAs and the orderer endpoints for the assemblers and routers (retrieved from the **current config block** of the channel).    
The first time you run `fxadmin`, use the channel **genesis block**. After each committed reconfiguration, use the newly produced config block.



## Build

`fxadmin` is built from this repository:

```bash
go build -o bin/fxadmin ./tools/fxadmin
```

Verify the build:

```bash
./bin/fxadmin --help
```

---

## Configuration file (`admin.yaml`)

The file describes the admin's **MSP identity**, and its
**TLS** certificate and key.

**Example**
```yaml
# admin configuration for orderer org1

msp:
  localMspID: org1
  localMspDir: ARTIFACTS_DIR/ordererOrganizations/org1/users/Admin@org1/msp

tls:
  enabled: true
  clientCert: ARTIFACTS_DIR/ordererOrganizations/org1/users/Admin@org1/tls/client.crt
  clientKey:  ARTIFACTS_DIR/ordererOrganizations/org1/users/Admin@org1/tls/client.key
```

**Fields**

| Section   | Field          | Description                                                                                                                              |
| --------- | -------------- |------------------------------------------------------------------------------------------------------------------------------------------|
| `msp`     | `localMspID`   | MSP ID of the admin's organization (e.g. `org1`). Used to build the signer's identity.                                                   |
| `msp`     | `localMspDir`  | Path to the admin's MSP directory (signing key + certificates).                                                                          |
| `tls`     | `enabled`      | Enable TLS. `false` (the default) = no TLS; `true` without `clientCert`/`clientKey` = server-side TLS (the server is verified using the orderer root CAs taken from the current config block); `true` with `clientCert`+`clientKey` = mutual TLS. |
| `tls`     | `clientCert`   | Admin's TLS **client certificate** (required for mutual TLS).                                                                            |
| `tls`     | `clientKey`    | Admin's TLS **client private key** (required for mutual TLS).                                                                            |

> **Note** — `clientCert` and `clientKey` must be provided together. If only one
> is set, mutual TLS is disabled and a warning is logged.

---

## Commands

### General flags

The following flags apply to all commands:

| Flag           | Description                       |
| -------------- | --------------------------------- |
| `-h`, `--help` | Show help for the command.        |
| `--version`    | Print the `fxadmin` version.      |

---

### Ledger Commands

```bash
fxadmin ledger [ledger flags] <command> [command flags]
```
#### Flags

These flags are common to all ledger commands:

| Flag              | Required | Description                                              |
| ----------------- | :------: |----------------------------------------------------------|
| `--current-block` |   yes    | Current config block containing the assembler's endpoints |
| `--config`        |   yes    | Admin configuration file (identity, TLS)                 |

#### Commands
```bash
# Ask for the ledger height
fxadmin ledger height

# Pull the latest block
fxadmin ledger block latest --output block.pb

# Pull block number 123
fxadmin ledger block 123 --output block.pb

# Pull the latest ledger configuration
fxadmin ledger config latest --output config.pb
```

---

#### Get the ledger height
Print the **height** of the ledger.

```bash
fxadmin ledger \
  --config admin.yaml \
  --current-block current_block.pb \
  height
```
---

#### Get the latest block
Fetch the latest committed block from an assembler and write it to a file.

```bash
fxadmin ledger \
  --config admin.yaml \
  --current-block current_block.pb \
  block latest \
  --output latest_block.pb
```
---

#### Get a specific block
Fetch a specific block by number and write it to a file.

```bash
fxadmin ledger \
  --config admin.yaml \
  --current-block current_block.pb \
  block 123 \
  --output block_123.pb
```
---

#### Get the last config block
Fetch the latest committed **config** block from an assembler and write it to a file.

```bash
fxadmin ledger \
  --config admin.yaml \
  --current-block current_block.pb \
  config latest \
  --output last_config.pb
```
---

> **Planned (not yet implemented)** — support environment variables as defaults
> for the common flags, to reduce repetitive typing and ease automation:
>
> ```bash
> export FXADMIN_CONFIG=admin.yaml
> export FXADMIN_CURRENT_BLOCK=current_block.pb
> ```

---
### Decode Command

The decode command extracts the `common.Config` embedded in a binary config block
(.pb) and writes it as a human-readable JSON file for editing.
This command is typically used after retrieving the latest configuration from the ledger.

```bash
fxadmin decode <config_block.pb> --output <current_config.json>
```

Example:
```bash
fxadmin decode last_config.pb --output current_config.json
```

**Arguments**

| Argument            | Required | Description                                      |
|---------------------| :------: |--------------------------------------------------|
| `<config_block.pb>` |   yes    | Path to the protobuf config block file to decode |

**Flags**

| Flag       | Required | Description                                      |
|------------| :------: |--------------------------------------------------|
| `--output` |   yes    | Path to the output `common.Config` JSON file, derived from the block |

---
### Manual step
After decoding, copy `current_config.json` to `modified_config.json` and edit `modified_config.json`
manually to describe the desired configuration (for example: rotate a
certificate, add or remove an organization/party, change an endpoint, or adjust
a batching/consensus parameter). Leave `current_config.json` untouched.

---

### Modify Commands (Automation of the Manual step)
The `modify` commands automate the **manual** change (decode → hand-edit) for the **most popular** configuration changes. 
They edit a config **block file** directly. Several changes can accumulate into the same block and
commit as a single config update. The manual step remains available for any change the modify commands do not cover.

**The flow**

```bash
# 1. Get the latest config block.
fxadmin ledger --config admin.yaml --current-block current_block.pb config latest --output current.pb

# 2. Copy it to a "next" block.
cp current.pb next.pb

# 3. Apply one or more modify commands to next.pb (changes accumulate).
# For example: change the port of the batcher of party1, shard1 
fxadmin modify party node --party 1 --role batcher --shard 1 --port 9014 --block next.pb
#   … more modify commands may be run against the same next.pb …

# 4. Compute the config update from the two blocks (block mode).
fxadmin compute-update --pb current.pb next.pb --output config_update.pb

# 5. Endorse → merge → send → follow  (unchanged - see the Transaction Command).
```

`modify` edits the block on disk and never contacts the network. 
Keep `current.pb` untouched for the `compute-update --pb` step in #4.

**Commands**

```bash
# Application
fxadmin modify app add                --org=<org.yaml>            --block <next.pb>
fxadmin modify app remove             <org>                       --block <next.pb>
fxadmin modify app known-certs add    --org <org> --cert <pem>…   --block <next.pb>
fxadmin modify app known-certs remove --org <org> --cert <pem>…   --block <next.pb>

# Orderer (ARMA parties)
fxadmin modify party add              --party=<party.yaml>        --block <next.pb>
fxadmin modify party remove           <party-id>                  --block <next.pb>

# Party's node (any subset of its fields)
fxadmin modify party node   --party <N> --role <role> [--shard <S>] \
    [--host <H>] [--port <P>] [--tls-cert <pem>] [--sign-cert <pem>]  --block <next.pb>

# A party's CA / TLS-CA certificate lists
fxadmin modify party ca  add|remove|set  --party <N> \
    [--sign-cert <pem>…] [--tls-cert <pem>…]                         --block <next.pb>
```

**Common flags on every command**

| Flag | Required | Description                                               |
|------|:--------:|-----------------------------------------------------------|
| `--block` | yes | Config block file to edit **in place** (the "next" block) |

Certificate flags always take a PEM **file path**, never inline certificate
bytes; the tool reads and parses the file.

For modify party node and modify party ca, omitted flags leave the corresponding fields unchanged, 
while supplied flags update them. Multiple flags can be provided in a single command to update several fields at once.

---

**modify app add**

Add an application organization to the `Application` group.

```bash
fxadmin modify app add --org=peer2.yaml --block next.pb
```

`peer2.yaml` is the organization definition:

```yaml
Name:   peer2                                # group name under Application
ID:     peer2                                # MSP ID
MSPDir: crypto/peerOrganizations/peer2/msp   # CA / TLS-CA / admin certs, node OUs
Policies:                                    # optional; default = standard org policies
  Readers:     { Type: Signature, Rule: "OR('peer2.member')" }
  Writers:     { Type: Signature, Rule: "OR('peer2.member')" }
  Admins:      { Type: Signature, Rule: "OR('peer2.admin')" }
  Endorsement: { Type: Signature, Rule: "OR('peer2.member')" }
AnchorPeers:                                 # optional
  - { Host: peer0.peer2.example.com, Port: 7051 }
```

**Flags**

| Flag | Required | Description                               |
|------|:--------:|-------------------------------------------|
| `--org` | yes | Path to the organization definition YAML  |
| `--block` | yes | Config block to edit in place             |

---
**modify app remove**

Remove an application organization.

```bash
fxadmin modify app remove peer2 --block next.pb
```

**Arguments**

| Argument | Required | Description                             |
|----------| :------: |-----------------------------------------|
| `<org>` | yes | Application organization name to remove |

---

**modify app known-certs add / remove**

Add or remove entries in an application org's MSP `known_certs` list.

```bash
fxadmin modify app known-certs add    --org peer1 --cert known1.pem --block next.pb
fxadmin modify app known-certs remove --org peer1 --cert known1.pem --block next.pb
```

**Flags**

| Flag | Required | Description                                       |
|------|:--------:|---------------------------------------------------|
| `--org` | yes | Application organization whose known-certs change |
| `--cert` | yes (≥1) | PEM path to add / remove             |
| `--block` | yes | Config block to edit in place                     |

`add` is idempotent (skips certs already present with a message to the admin). 
`remove` errors if a named cert is absent.

---

**modify party add**

Add a new ARMA party and its orderer organization, if it does not already exist.

```bash
fxadmin modify party add --party=party5.yaml --block next.pb
```

`party.yaml` is the orderer's shared-config party entry, with
certificate paths:

```yaml
# party.yaml
PartyID: 5
CACerts:    [crypto/.../org5/ca/ca-cert.pem]
TLSCACerts: [crypto/.../org5/tlsca/tlsca-cert.pem]
RouterConfig:    { Host: 127.0.0.1, Port: 8053, TLSCert: .../router/tls.crt }
BatchersConfig:
  - { ShardID: 1, Host: 127.0.0.1, Port: 8054, SignCert: .../b1/sign.crt, TLSCert: .../b1/tls.crt }
  - { ShardID: 2, Host: 127.0.0.1, Port: 8055, SignCert: .../b2/sign.crt, TLSCert: .../b2/tls.crt }
ConsenterConfig: { Host: 127.0.0.1, Port: 8052, SignCert: .../c/sign.crt, TLSCert: .../c/tls.crt }
AssemblerConfig: { Host: 127.0.0.1, Port: 8051, TLSCert: .../a/tls.crt }
```

**Flags**

| Flag | Required | Description |
|------|:--------:|-------------|
| `--party` | yes | Path to the party definition YAML |
| `--block` | yes | Config block to edit in place |

---

**modify party remove**

Remove an ARMA party and its orderer organization, unless the organization is still associated with another party.

```bash
fxadmin modify party remove 5 --block next.pb
```

**Arguments**

| Argument | Required | Description |
|----------| :------: |-------------|
| `<party-id>` | yes | Numeric `PartyID` to remove |

---

**modify party node**

Change one party node's endpoint and/or certificates. The node is selected by
`--party` + `--role` (+ `--shard` for a batcher); the optional field flags say
what to change. Any subset of fields can be changed in one command.

```bash
# move an endpoint
fxadmin modify party node --party 1 --role batcher --shard 1 --port 9014 --block next.pb

# rotate a cert
fxadmin modify party node --party 1 --role router --tls-cert new-router-tls.crt --block next.pb

# several fields of one node at once
fxadmin modify party node --party 1 --role consenter \
  --sign-cert new-c-sign.crt --tls-cert new-c-tls.crt --port 8052 --block next.pb
```

**Flags**

| Flag | Required | Description |
|------|:--------:|-------------|
| `--party` | yes | Party ID |
| `--role` | yes | `router \| batcher \| consenter \| assembler` |
| `--shard` | if batcher | Batcher shard ID |
| `--host` | no | New host (unchanged if omitted) |
| `--port` | no | New port (unchanged if omitted) |
| `--tls-cert` | no | Path to new TLS certificate (unchanged if omitted) |
| `--sign-cert` | no | Path to new signing certificate (unchanged if omitted) |
| `--block` | yes | Config block to edit in place |

Only `batcher` and `consenter` nodes have a signing certificate. Using
`--sign-cert` with `--role router` or `--role assembler` is an error. Changing a
`router` or `assembler` endpoint also updates that party's orderer-organization
`Endpoints` value (the router is advertised as the `broadcast` address, the
assembler as the `deliver` address).

___

**modify party ca add | remove | set**

Change a party's sign-CA (`--sign-cert`) and/or TLS-CA (`--tls-cert`) certificate
**lists**.

```bash
# add a new CA alongside the current ones
fxadmin modify party ca add    --party 1 --sign-cert new-ca.crt --block next.pb

# remove a retired TLS-CA
fxadmin modify party ca remove --party 1 --tls-cert old-tlsca.crt --block next.pb

# replace both lists
fxadmin modify party ca set --party 1 --sign-cert new-ca.crt --tls-cert new-tlsca.crt --block next.pb
```

**Subcommands**

| Subcommand | Effect on the named list(s) |
|------------|-----------------------------|
| `add`      | Append the given cert(s) (idempotent; keeps existing, so old and new coexist). |
| `remove`   | Drop the given cert(s) (errors if absent; refuses to empty a list). |
| `set`      | Replace the list with exactly the given cert(s). |

**Flags**

| Flag | Required | Description |
|------|:--------:|-------------|
| `--party` | yes | Party whose CA list(s) change |
| `--sign-cert` | ≥1 of the two | PEM path(s) for the signing-CA list (repeatable) |
| `--tls-cert` | ≥1 of the two | PEM path(s) for the TLS-CA list (repeatable) |
| `--block` | yes | Config block to edit in place |

Pass only `--sign-cert` to change the CA list, only `--tls-cert` for the TLS-CA
list, or both to change both. CA changes are applied to both the party and its orderer org MSP.

---

### Compute a config update Command

Compute the **`ConfigUpdate`** which is the delta between the original configuration and the
modified configuration.

`compute-update` has two modes for supplying that pair:

- **JSON mode** (default) - compute the update from two config **JSON** files. Goes with the [manual step](#manual-step).
- **Block mode** (`--pb`) - compute the update from two config **block** files. Goes with the
  [Modify Commands](#modify-commands-automation-of-the-manual-step), which edit a
  block directly.

Both modes produce the same kind of `ConfigUpdate`. If the two inputs are identical the command
produces an empty update and reports that there is nothing to do.

<br><br>
#### JSON mode (default)

The original is the config JSON decoded from `--current-block`; the modified is its edited copy.
The channel ID is taken from the current config block supplied with `--current-block`
and stamped onto the resulting `ConfigUpdate`. All three inputs must belong to the
**same channel**.

```bash
fxadmin compute-update <current.json> <modified.json> --current-block <current_block.pb> --output <config_update.pb>
```

Example:
```bash
fxadmin compute-update \
   current_config.json \
   modified_config.json \
   --current-block last_config.pb \
   --output config_update.pb
```

**Arguments** (order matters: original first, modified second)

| Argument          | Required | Description                                            |
|-------------------| :------: |--------------------------------------------------------|
| `<current.json>`  |   yes    | Original configuration, as decoded from `--current-block` |
| `<modified.json>` |   yes    | Modified configuration (edited copy of `current.json`) |

**Flags**

| Flag              | Required | Description                                                          |
|-------------------| :------: |----------------------------------------------------------------------|
| `--current-block` |   yes    | Path to the current config block whose channel ID the update targets |
| `--output`        |   yes    | Path to the output `ConfigUpdate` protobuf file                      |


<br><br>
#### Block mode (`--pb`)

Computes the delta directly from two config **block** files.
This command is for use with the [Modify Commands](#modify-commands-automation-of-the-manual-step): 
the `modify` commands edit a block file directly, so the delta is computed from the
original block and the edited ("next") block.


```bash
fxadmin compute-update --pb <current.pb> <next.pb> --output <config_update.pb>
```

Example:
```bash
fxadmin compute-update --pb \
   current.pb \
   next.pb \
   --output config_update.pb
```

**Arguments** (order matters: original first, modified second)

| Argument       | Required | Description                                                          |
|----------------| :------: |----------------------------------------------------------------------|
| `<current.pb>` |   yes    | **Original** config block - the one fetched from the ledger          |
| `<next.pb>`    |   yes    | **Modified** config block - the copy edited by the `modify` commands |

**Flags**

| Flag       | Required | Description                                                          |
|------------| :------: |----------------------------------------------------------------------|
| `--pb`     |   yes    | Select block mode (diff two config blocks instead of two JSON files) |
| `--output` |   yes    | Path to the output `ConfigUpdate` protobuf file                      |

---
### Transaction Command

The tx command endorses, merges, prepares, and submits configuration update transactions.

```bash
fxadmin tx <command> [command arguments] [command flags]
```

#### Commands

```bash
# Endorse a configuration update
fxadmin tx endorse config_update.pb \
  --config admin.yaml \
  --output endorsed_config_update1.pb

# Merge multiple endorsements
fxadmin tx merge \
  endorsed_config_update1.pb \
  endorsed_config_update2.pb \
  --output endorsed_config_update.pb

# Prepare the configuration transaction
fxadmin tx prepare endorsed_config_update.pb \
  --config admin.yaml \
  --output config_tx.pb

# Submit the configuration transaction
fxadmin tx submit config_tx.pb \
  --config admin.yaml \
  --current-block current_block.pb
  
# Prepare + submit the configuration transaction
fxadmin tx send endorsed_config_update.pb \
   --config admin.yaml \
   --current-block current_block.pb \
   --output config_tx.pb
```
---

#### Endorse a configuration update

The endorse command signs a configuration update using the administrator identity defined in the admin configuration YAML file.  
The admin signs on `ConfigUpdate`, producing an endorsed update that
carries that admin's `ConfigSignature`.

```bash
fxadmin tx endorse \
  <config_update.pb> \
  --config <admin.yaml> \
  --output <endorsement.pb>
```
**Arguments**

| Argument             | Required | Description                                    |
|----------------------| :------: | ---------------------------------------------- |
| `<config_update.pb>` |   yes    | Path to the configuration update protobuf file to endorse |

**Flags**

| Flag         | Required | Description                                                       |
| ------------ | :------: | ---------------------------------------------------------------- |
| `--config`   |   yes    | Path to the admin configuration YAML file containing the administrator signing identity |
| `--output`   |   yes    | Path to the generated endorsement protobuf file    |

**Example**

```bash
# org1 admin endorses
fxadmin tx endorse config_update.pb \
  --config admin_org1.yaml \
  --output endorsed_config_update1.pb

# org2 admin endorses
fxadmin tx endorse config_update.pb \
  --config admin_org2.yaml \
  --output endorsed_config_update2.pb
```

---

#### Merge Endorsements
The merge command combines multiple endorsements into a single endorsed configuration update envelope (`ConfigUpdateEnvelope`).
Use this to gather the endorsements needed to satisfy the channel's admin policy (e.g. a majority of orderer orgs). 
Duplicate signers are de-duplicated.

```bash
fxadmin tx merge \
  <endorsement.pb>... \
  --output <endorsed_config_update.pb>
```
**Arguments**

| Argument              | Required | Description                                          |
| --------------------- | :------: | ---------------------------------------------------- |
| `<endorsement.pb>...`  |   yes    | Paths to one or more endorsement protobuf files to merge   |

**Flags**

| Flag         | Required | Description                                                  |
| ------------ | :------: | ------------------------------------------------------------ |
| `--output`   |   yes    | Path to the merged configuration update envelope |

**Example**

```bash
fxadmin tx merge \
  endorsed_config_update1.pb \
  endorsed_config_update2.pb \
  --output endorsed_config_update.pb
```

> **Note:** The `merge` command is only required when the configuration update has been endorsed by multiple administrators. If a single administrator endorsement satisfies the endorsement policy, the output of `tx endorse` can be used directly as input to `tx prepare` or `tx send`.

---

#### Prepare a Configuration Transaction

The prepare command creates a ready to submit configuration transaction envelope from the endorsed configuration update.
The resulting transaction is signed by the submitting client which is a channel writer (it can be the admin).

```bash
fxadmin tx prepare \
  <endorsed_config_update.pb> \
  --config <admin.yaml> \
  --output <config_tx.pb>
```

**Arguments**

| Arguments                                  | Required | Description                                                            |
|---------------------------------------| :------: | --------------------------------------------------------------------- |
| `<endorsed_config_update.pb>` |   yes    | Path to the endorsed configuration update protobuf file  |


**Flags**

| Flag         | Required | Description                                                             |
| ------------ | :------: | ---------------------------------------------------------------------- |
| `--config`   |   yes    | Path to the admin configuration YAML file containing the submitting client identity  |
| `--output`   |   yes    | Path to the generated configuration transaction protobuf file                   |

**Example**
```bash
fxadmin tx prepare endorsed_config_update.pb \
  --config admin.yaml \
  --output config_tx.pb
```

---

#### Submit a Configuration Transaction

The submit command submits a prepared configuration transaction to all routers via the
Broadcast API. A router forwards it into the ordering pipeline; once ordered and
committed, the new configuration takes effect across all parties.

The command broadcasts to every router and collects each router's acknowledgement, logging
the per-router outcome. It **succeeds** only when a BFT quorum of the routers acknowledges 
the transaction: with `n` parties (read from the`--current-block`) the quorum is `2f+1`, 
where `f = (n-1)/3` is the number of faulty parties the network tolerates. 
If fewer than a quorum acknowledge — because routers rejected the
transaction or were unreachable — the command fails, so a partial or failed delivery is
reported rather than silently succeeding.

```bash
fxadmin tx submit \
  <config_tx.pb> \
  --config <admin.yaml> \
  --current-block <current_block.pb>
```

**Arguments**

| Argument       | Required | Description                                     |
|----------------| :------: | ----------------------------------------------- |
| `<config_tx.pb>` |   yes    | Path to the prepared configuration transaction protobuf file       |

**Flags**

| Flag              | Required | Description                                                               |
| ----------------- | :------: |---------------------------------------------------------------------------|
| `--current-block` |   yes    | Path to the current block protobuf file containing the router's endpoints |
| `--config`        |   yes    | Path to the admin configuration YAML file                                 |

**Example**

```bash
fxadmin tx submit config_tx.pb \
  --config admin.yaml \
  --current-block current_block.pb
```

After submitting the transaction, confirm that the configuration update has been committed by reading the latest configuration block until the updated configuration block is available.

---
#### Send a Configuration Update
The send command prepares and submits an endorsed configuration update in a single step.

It creates a configuration transaction from the endorsed configuration update, signs it using the submitting client identity defined in the admin configuration YAML file, and submits the transaction to all configured routers.

The send command is equivalent to running `prepare` followed by `submit`. As with `submit`, it
succeeds only when a BFT quorum (`2f+1`) of the routers acknowledged the transaction, and fails
otherwise.

```bash
fxadmin tx send \
 <endorsed_config_update.pb> \
 --config <admin.yaml> \
 --current-block <current_block.pb> \
 --output <config_tx.pb>
```
**Arguments**

| Arguments                                  | Required | Description                                                            |
|---------------------------------------| :------: | --------------------------------------------------------------------- |
| `<endorsed_config_update.pb>` |   yes    | Path to the endorsed configuration update protobuf file  |


**Flags**

| Flag         | Required | Description                                                             |
| ------------ | :------: | ---------------------------------------------------------------------- |
| `--current-block` |   yes    | Path to the current block protobuf file containing the router's endpoints |
| `--config`        |   yes    | Path to the admin configuration YAML file                                 |
| `--output`        |   yes    | Path to write the prepared configuration transaction to, for record keeping |

The prepared configuration transaction is written to `--output` before it is broadcast, so a record of what was submitted is kept even if the broadcast fails.

**Example**
```bash
fxadmin tx send \
   endorsed_config_update.pb \
   --config admin.yaml \
   --current-block current_block.pb \
   --output config_tx.pb
```
---
### Follow the assembler ledger
The follow command waits for the next config block to commit across all assemblers.

It reads the current config block to learn the current config sequence `S` and the assembler
endpoints, and waits for the block a pending update will produce once committed:
`expected = S + 1`. A configuration update changes at most one assembler, so the endpoints in the
current block still reach the rest; assemblers that cannot be reached are reported as unreachable.

For each assembler, follow pulls blocks until it observes a config block whose sequence is
`expected` (or higher — a later update may also have committed), which means the next config is
committed on that assembler. An assembler whose last config sequence is still below `expected` is
behind (the config is not committed there yet). Polling continues per assembler until it commits
or the timeout expires. When done, the command prints, for each assembler, its last block number,
the last config sequence in its ledger, and whether it committed.

The `--timeout` is the hard upper bound on the whole command, so an unresponsive assembler cannot block
past it. An assembler that never answers within the window is reported as `unreachable`.

Once at least `f+1` assemblers report the **same** next config block (`f = (n-1)/3`
is the number of faulty parties the network of `n` parties tolerates), that block is written
to `--output`. The written block is ready to serve as the `--current-block` of the next
reconfiguration round. If a quorum of `f+1` matching blocks is not reached before the timeout,
follow prints the report, writes no output file, and exits with an error.

Blocks are compared by their **header and data only**, ignoring the block metadata, which
carries the orderer signatures over the block. In ARMA, block metadata may contain different signer sets 
or signature ordering, so the raw block bytes may differ.

```bash
fxadmin follow \
   --config <admin.yaml> \
   --current-block <current_block.pb> \
   --timeout <duration> \
   --output <next_config.pb>
```

**Flags**

| Flag              | Required | Description                                                                             |
|-------------------| :------: |-----------------------------------------------------------------------------------------|
| `--current-block` |   yes    | Path to the current block protobuf file containing the assembler's endpoints            |
| `--config`        |   yes    | Path to the admin configuration YAML file                                               |
| `--timeout`       |   yes    | Maximum amount of time to pull blocks from the assemblers before reporting the results  |
| `--output`        |   yes    | Path to write the next committed config block to, for use as the next `--current-block` |

**Example**
```bash
fxadmin follow \
 --config admin.yaml \
 --current-block last_config.pb \
 --timeout 30s \
 --output next_config.pb
```
**Output**

The command logs a one-line summary, for example:

```
expected last config sequence: 5, 3 out of 4 assemblers committed a block with last config sequence 5
```

If any assembler has not committed when the timeout elapses, it also logs a warning, for example:

```
timeout of 30s elapsed with 1 of 4 assemblers not yet committed to last config sequence 5
```

It then prints the polling timeout and a per-assembler table:

```
polling timeout: 30s
ASSEMBLER        LAST BLOCK  LAST CONFIG SEQUENCE  STATUS
assembler1:7051  104         5                     committed
assembler2:7053  104         5                     committed
assembler3:7055  104         5                     committed
assembler4:7057  103         4                     behind
```

An assembler that could not be reached during the whole window is shown with `-` for both its last
block and last config sequence, and an `unreachable` status.

When at least f+1 assemblers agrees on the next config block, follow writes it to `--output` and logs, for example:

```
config block at last config sequence 5 agreed by 3 assemblers (quorum 3), written to next_config.pb
```

If less than f+1 agrees before the timeout, no output file is written and the command fails with an error
such as `no config block at last config sequence 5 was agreed by a quorum of 3 assemblers`.

 ---

## End-to-end walkthrough (with Manual step)

### Single organization

When one organization's admin signature satisfies the config policy, the whole
flow can be run by that admin:

```bash
# 1. Read the current configuration.
fxadmin ledger --config=admin.yaml --current-block=current_block.pb config latest --output last_config.pb

# 2. Decode it to JSON.
fxadmin decode last_config.pb --output=current_config.json

# 3. Edit manually: copy the current_config.json to modified_config.json and edit.

# 4. Compute the update.
fxadmin compute-update current_config.json modified_config.json --current-block=last_config.pb --output=config_update.pb

# 5. Endorse, prepare, submit.
fxadmin tx endorse config_update.pb --config=admin.yaml --output=endorsed_config_update.pb
fxadmin tx prepare endorsed_config_update.pb --config=admin.yaml --output=config_tx.pb
fxadmin tx submit  config_tx.pb --config=admin.yaml --current-block=last_config.pb

# 6. Follow the assembler ledger to make sure the config tx was committed.
#    The new config block is written to next_config.pb, ready to be
#    the --current-block of the next reconfiguration round.
fxadmin follow --config=admin.yaml --current-block=last_config.pb --timeout=60s --output=next_config.pb

```

### Multiple organizations

When the config policy requires signatures from more than one organization, each admin
endorses the same `ConfigUpdate` independently, the endorsements are merged, and a single
submitting client (any channel writer, e.g. the org1 admin) prepares and submits the
transaction:

```bash
# 1. Read the current configuration.
fxadmin ledger --config=admin_org1.yaml --current-block=current_block.pb config latest --output last_config.pb

# 2. Decode it to JSON.
fxadmin decode last_config.pb --output=current_config.json

# 3. Edit manually: copy the current_config.json to modified_config.json and edit.

# 4. Compute the update.
fxadmin compute-update current_config.json modified_config.json --current-block=last_config.pb --output=config_update.pb

# 5. Each org endorses the same update independently.
fxadmin tx endorse config_update.pb --config=admin_org1.yaml --output=endorsed_config_update1.pb
fxadmin tx endorse config_update.pb --config=admin_org2.yaml --output=endorsed_config_update2.pb

# 6. Merge the endorsements into a single endorsed config update.
fxadmin tx merge endorsed_config_update1.pb endorsed_config_update2.pb --output=endorsed_config_update.pb

# 7. Prepare and submit in one step with send (prepare + submit).
fxadmin tx send endorsed_config_update.pb --config=admin_org1.yaml --current-block=last_config.pb --output=config_tx.pb

# 8. Follow the assembler ledger to make sure the config tx was committed.
#    The new config block is written to next_config.pb, ready to be
#    the --current-block of the next reconfiguration round.
fxadmin follow --config=admin_org1.yaml --current-block=last_config.pb --timeout=60s --output=next_config.pb
```

## End-to-end walkthrough (with Modify commands)

### Multiple organizations
This example changes party 1's batcher (shard 1) endpoint and rotates the router's
TLS certificate - two changes accumulated into one config update:

```bash
# 1. Read the current config block, and copy it to a "next" block.
fxadmin ledger --config=admin_org1.yaml --current-block=current_block.pb config latest --output current.pb
cp current.pb next.pb

# 2. Apply the changes to next.pb (each edits the block in place; changes accumulate).
fxadmin modify party node --party 1 --role batcher --shard 1 --port 9014 --block next.pb
fxadmin modify party node --party 1 --role router --tls-cert new-router-tls.crt --block next.pb

# 3. Compute the update from the two blocks (block mode).
fxadmin compute-update --pb current.pb next.pb --output=config_update.pb

# 4. Each org endorses the same update independently.
fxadmin tx endorse config_update.pb --config=admin_org1.yaml --output=endorsed_config_update1.pb
fxadmin tx endorse config_update.pb --config=admin_org2.yaml --output=endorsed_config_update2.pb
fxadmin tx endorse config_update.pb --config=admin_org3.yaml --output=endorsed_config_update3.pb

# 5. Merge the endorsements into a single endorsed config update.
fxadmin tx merge \
  endorsed_config_update1.pb endorsed_config_update2.pb endorsed_config_update3.pb \
  --output=endorsed_config_update.pb

# 6. Prepare and submit in one step with send (prepare + submit).
fxadmin tx send endorsed_config_update.pb --config=admin_org1.yaml --current-block=current.pb --output=config_tx.pb

# 7. Follow the assembler ledger to make sure the config tx was committed.
fxadmin follow --config=admin_org1.yaml --current-block=current.pb --timeout=60s --output=next_config.pb
```

Only steps 1–3 differ from the manual walkthrough; endorse / merge / send / follow
are identical. For the full command reference see
[Modify Commands](#modify-commands-automation-of-the-manual-step).