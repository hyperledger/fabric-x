<!-- SPDX-License-Identifier: Apache-2.0 -->
# Fabric-X Ledger

## Overview

A Fabric-X ledger records the history and current state of a permissioned network. Like Hyperledger Fabric, it has two related views: an immutable block history and a current world state.

Fabric-X adapts this ledger model to its single-channel architecture. There is one shared channel history for the network, and namespaces provide logical separation for application state and endorsement policy.

## Ledger Components

Fabric-X ledger storage is composed of:

| Component | Purpose |
| --- | --- |
| Block history | Immutable ordered record of transactions and configuration changes. |
| World state | Current key/value state after applying valid transactions. |
| Transaction status | Final valid or invalid result for every ordered transaction. |
| Configuration data | Genesis and subsequent configuration updates that define membership, MSPs, policies, ordering settings, and namespace metadata. |

The block history explains how the current state was reached. The world state lets applications read current values without replaying every block.

## Block History

Arma orders endorsed transaction bytes into blocks before validation. Each block has a deterministic position in the shared history and links to previous blocks through hashes and metadata.

Ordering a transaction does not mean the transaction is valid. It means the network has agreed where that transaction appears in the log. Committer services later verify policy and state consistency at that position.

Fabric-X has one shared channel history, not separate per-channel ledgers. If participants need separate histories, separate governance domains, or complete ledger isolation, they should use separate Fabric-X networks.

## World State

World state stores the latest committed value for each key. Fabric-X uses PostgreSQL/YugabyteDB-backed committer storage for world state and related metadata.

Only valid transactions update world state. Invalid transactions remain in block history and have recorded status, but their writes are not applied.

World state is optimized for current reads. Block history is optimized for audit, replay, recovery, and proof of ordering.

## Namespaces

Namespaces provide logical state separation inside the single channel. A namespace groups application keys and associates them with endorsement and governance rules.

Different namespaces can represent different assets, applications, business domains, or participant workflows. They share the same ordered block history while keeping state organization and policy evaluation explicit.

Namespaces are not separate ledgers. They are logical boundaries within the shared Fabric-X ledger model.

## Read/Write Sets

Endorsement logic produces read/write sets before ordering. A read set records keys and versions observed during execution. A write set records proposed creates, updates, or deletes.

Read/write sets make validation deterministic. After Arma orders a transaction, the committer can check whether the versions read during endorsement are still current at that transaction's block position.

## Validation and MVCC

Validation combines policy checks and state checks:

1. The verifier checks signatures and namespace endorsement policy.
2. The validator-committer checks read versions against world state.
3. Valid writes update namespace state.
4. Final status is recorded for every transaction.

MVCC validation prevents stale updates. A transaction can have correct endorsements and still fail if a previous valid transaction changed a key it read.

## Valid and Invalid Transactions

Both valid and invalid transactions remain in block history. This preserves auditability and allows all participants to inspect what was ordered.

Valid transactions update world state. Invalid transactions do not update world state, but Fabric-X records their validation status so clients and operators can understand the final outcome.

## Query Path

The Query Service provides read-only access to committed state, policies, configuration, and transaction status. Applications, FSC views, and custom endorsers use the query path to read committed data without coupling to validation workers.

Because query traffic is separate from commit processing, operators can scale read capacity and commit capacity independently.

## Genesis and Configuration Blocks

The genesis block is block zero for the Fabric-X network. It carries initial configuration that all participants must share: organizations, MSP definitions, policies, ordering configuration, and initial namespace metadata.

Configuration changes are recorded through governed configuration updates. These updates change the trust and governance model, so they must satisfy the relevant configuration policies and remain auditable in ledger history.

## See Also

- [Fabric-X Network](network.md)
- [Namespaces](namespaces.md)
- [Endorsement and Other Policies](policies/policies.md)
- [Transaction Flow](transaction-flow.md)
- [Query Service](query-service.md)
- [Validator-Committer](../committer/docs/validator-committer.md)
