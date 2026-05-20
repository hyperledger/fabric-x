<!-- SPDX-License-Identifier: Apache-2.0 -->
# Hyperledger Fabric-X Model

## Overview

Fabric-X keeps Fabric's execute-order-validate-commit model while adapting it to Fabric-X services. Application execution and endorsement happen before ordering. Arma orders endorsed transaction bytes into blocks. Committer services validate signatures, policies, and read versions, then commit valid writes and record final status for every transaction.

This model separates application logic, ordering, validation, and commit. FSC views or custom endorsers produce transaction read/write sets and signatures. Arma gives each transaction a deterministic position in the ordered block stream. The committer decides whether the transaction is authorized and still consistent with committed state.

## Assets

Assets are represented as namespace key/value state. A namespace is the logical scope for application state and policy. State changes are recorded as transactions, so creates, updates, and deletes become part of the ordered ledger history.

Applications choose how to encode asset values. The Fabric-X model treats them as bytes associated with keys in namespaces, with transaction read/write sets describing proposed state transitions.

## Endorsement Logic

Fabric-X does not define built-in chaincode execution on this model page. Application logic runs in FSC views or custom endorsers.

Endorsement logic reads committed state, applies application rules, and produces:

- namespace read sets with versions;
- namespace write sets with proposed changes;
- transaction metadata;
- signatures or endorsements required by namespace policy.

The endorsed transaction is not final state. It is input to ordering and later committer validation.

## Ledger Features

The ledger contains immutable ordered blocks plus current world state. Blocks record ordered transactions and validation results. World state stores latest committed values for namespace keys.

Fabric-X uses one shared channel. Namespaces separate application state and policy within that channel instead of creating separate channel ledgers per application. This gives applications logical separation while preserving one shared ordered history.

Committer services apply only valid writes to world state. Invalid transactions remain recorded with final status for auditability, but do not change current state.

## Privacy and Confidentiality

Fabric-X uses one shared channel, so it does not use multi-channel privacy as a primary privacy model. Fabric-X also does not use private data collections in this conceptual model.

Use namespaces for logical separation of application state and policy. When confidentiality is required, applications should use app-level encryption or other application-controlled data protection before transaction data is ordered and committed.

## Security and Membership Services

Fabric-X uses Fabric-compatible membership concepts. MSPs define trusted identities and organizations. X.509 certificates identify users, applications, and services. Policies express which identities or organizations may submit, endorse, administer, or access resources. TLS protects service communication.

These security and membership services make transactions attributable and policy-enforced across the permissioned network.

## Consensus

In Fabric-X, consensus covers the full transaction path: endorsement, ordering, validation, and commit. Agreement is not only transaction order; it also depends on required signatures, policy satisfaction, and state-version validity.

Arma provides BFT ordering for endorsed transaction bytes. Committer services enforce namespace policy and MVCC validity after ordering. A transaction reaches final outcome when the committer records its valid or invalid status and commits any valid writes.

## Execute-Order-Validate-Commit Flow

### Execute / Endorse

FSC views or custom endorsers run application logic against committed state. They produce read/write sets, metadata, and signatures that can satisfy namespace endorsement policy.

### Order

Arma orders endorsed transaction bytes into blocks. Ordering assigns deterministic position, but does not by itself make a transaction valid.

### Validate

Committer services verify signatures and namespace policies, then perform MVCC validation against committed versions. Transactions with stale reads or insufficient endorsements are marked invalid.

### Commit

Committer services apply valid writes to world state and persist final status for every transaction. Invalid transactions remain in the immutable block history without updating application state.

## See Also

- [Transaction Flow](transaction-flow.md)
- [Ledger](ledger.md)
- [Namespaces](namespaces.md)
- [Policies](policies/policies.md)
- [Identity](identity.md)
- [MSP](msp.md)
