<!-- SPDX-License-Identifier: Apache-2.0 -->
# Introduction

This Concepts section introduces the ideas behind Fabric-X before you deploy a network, write applications, or operate services. It starts with blockchain basics, then explains how Fabric-X adapts familiar Hyperledger Fabric concepts for high-throughput ordering, validation, and commit.

Fabric-X remains a permissioned blockchain system. Organizations use MSP-backed identities, policies govern who can endorse or administer resources, transactions produce read/write sets, and the ledger records ordered blocks plus current world state. Fabric-X changes where these responsibilities run: applications endorse transactions through FSC views or custom endorsement logic, Arma orders transactions, and committer services validate and commit them.

## Learning Path

1. **[Blockchain Basics](blockchain.md)** — Start here for distributed ledgers, permissioned networks, consensus, and why enterprise blockchains need shared trust.
2. **[Fabric-X Network](network.md)** — Learn the major network roles: organizations, CAs, MSPs, applications, ordering services, committers, query services, the single channel model, and namespaces.
3. **[Identity Management](identity.md)** — Understand X.509 certificates, certificate authorities, signing identities, TLS identities, and cached identities used by Fabric-X clients and services.
4. **[Membership Service Provider](msp.md)** — See how MSP configuration defines trusted roots, admins, Node OUs, local MSPs, and organizational MSPs.
5. **[Endorsement and Other Policies](policies/policies.md)** — Learn how namespace endorsement policies, MSP signature rules, threshold rules, access policies, and configuration policies authorize actions.
6. **[Ledger](ledger.md)** — Understand the genesis/configuration block, ordered block history, world state, read/write sets, MVCC validation, and query path.

## How These Concepts Fit Together

A Fabric-X transaction begins with an application identity. Endorsement logic checks that identity and produces a signed transaction proposal response. Policies define which signatures are sufficient. Arma orders accepted transactions into blocks. Committer services verify signatures and policies, validate read/write sets against the current world state, and commit valid updates to the ledger.

Fabric-X uses a single channel. Namespaces provide logical separation for state and policies, so applications can isolate data and authorization rules without creating separate channels.

## Related Concept Pages

- [Fabric-X Model](fabric-x-model.md) — execute-order-validate-commit adapted to FSC/custom endorsement, Arma ordering, and committer validation.
- [Transaction Flow](transaction-flow.md) — end-to-end path from endorsed transaction to committed ledger update.
- [Namespaces](namespaces.md) — logical state and policy separation within the single channel.
- [Query Service](query-service.md) — read-only state access with view-based consistency.
- [Threshold Signatures](threshold-signatures.md) — compact multi-party approval for endorsement and policy use cases.

## Fabric Documentation Sources

Several concepts are adapted from Hyperledger Fabric documentation because Fabric-X retains Fabric-compatible identity, MSP, policy, ledger, and execute-order-validate-commit concepts. Fabric-X-specific text updates the parts that differ: no multi-channel model, no contract/chaincode hot path, Arma ordering, and committer validation and commit.
