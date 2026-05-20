<!-- SPDX-License-Identifier: Apache-2.0 -->
# How Fabric-X Networks Are Structured

## Overview

A Fabric-X network is a permissioned technical and governance domain where known organizations share identity rules, ordering, validation, and ledger state. Organizations agree on who participates, which certificate authorities and MSPs are trusted, which policies control administration, and which services maintain the network.

Fabric-X mirrors many Fabric concepts: organizations, CAs, MSPs, ordering, validation, policies, and ledgers. The major structural difference is channel topology. Fabric-X does not support channels as separate network partitions. It uses one channel for the network and namespaces for application-level state and policy boundaries.

This topic describes those components at conceptual level. It avoids deployment details and internal service mechanics.

## Organizations

Organizations are administrative trust domains that own identities, infrastructure, and governance responsibilities. An organization might represent a bank, payment operator, asset issuer, regulator, technology provider, or other consortium member.

Each organization typically controls its own private keys, certificates, administrators, operational processes, and service instances. It may run ordering infrastructure, committer services, query services, FSC nodes, custom endorsers, or client applications. A single organization can perform multiple roles, and different organizations can agree to different responsibilities through policy.

Organizations are not just operational units. They are also governance units. Network policies can grant organizations authority to approve configuration updates, endorse transactions for specific namespaces, or administer identities.

## Certificate Authorities and MSPs

Certificate Authorities issue certificates for administrators, services, applications, and other identities. Those certificates bind identities to organizations and allow participants to authenticate connections, sign configuration updates, endorse transaction results, and submit transactions.

Membership Service Providers define how certificates are trusted. An MSP identifies trusted root and intermediate CAs, administrator identities, and role rules such as client, peer-like service, orderer-like service, or admin roles. Policies reference MSP principals rather than raw public keys, so governance can be expressed in organizational terms.

Keeping CA operations and MSP configuration correct is central to network trust. If an MSP trusts a root CA, identities issued under that root can satisfy roles allowed by policy.

## Applications and Endorsement

Applications submit work to Fabric-X using FSC views or custom endorsers instead of Fabric contracts or chaincode as primary application model. Application logic prepares transaction intent, obtains required endorsements, and submits endorsed transactions for ordering.

Endorsement policies define which organizations or identities must sign transaction results before those transactions can be accepted. Different namespaces can use different endorsement policies, letting one network host multiple applications or business domains while keeping validation rules explicit.

Fabric-X applications should be designed around identities, signatures, policies, namespaces, and state updates. They do not rely on Fabric chaincode lifecycle or per-channel contract deployment as main structure.

## Ordering Service

Arma is the ordering service for Fabric-X. It receives endorsed transactions, establishes total order, and emits ordered blocks for validation and commit.

Ordering does not decide whether application updates are valid. It provides shared sequence. Committer services later evaluate signatures, policies, and read versions before applying valid updates to world state.

Organizations govern ordering participation and configuration through network policy. Operational details such as deployment topology and tuning are outside this conceptual topic.

## Committer and Query Services

Committer services receive ordered blocks and validate transactions. They validate signatures, policies, and MVCC read versions, then update world state for valid transactions. Invalid transactions remain part of ordered history but do not update world state.

Query services provide read access to committed state. They allow applications and operators to inspect current world state without submitting transactions. Deployments may separate commit and query responsibilities so read traffic and validation work can be operated independently.

Together, committers and query services provide ledger maintenance and state access for applications that use Fabric-X.

## Single Channel and Namespaces

Fabric-X uses a single channel for network-wide ordering, validation context, configuration, and ledger history. All organizations participating in one Fabric-X network share that channel.

Unlike Hyperledger Fabric, Fabric-X does not support channels as separate network partitions. Instead, Fabric-X uses namespaces to organize application state and attach policies to logical domains. A namespace can represent an asset class, business process, payment rail, or application. Namespaces share ordered history but can have distinct endorsement and validation policy boundaries.

This model avoids channel sprawl while keeping application boundaries visible. When complete isolation is required, use separate Fabric-X networks rather than attempting to create multiple channels in one network.

## Ledger

The ledger records ordered transaction history and current world state. Blocks provide immutable history of submitted transactions. World state stores latest values used by applications and validation.

Committer services apply valid writes to world state after checking signatures, policies, and MVCC read versions. MVCC checks prevent transactions from updating state based on stale reads. Query services read committed state for applications and operators.

Namespaces organize ledger keys and policy domains within shared network history. They do not create separate ledgers.

## Policies and Governance

Policies define how organizations make decisions and how transactions are validated. Administrative policies can control MSP updates, organization membership changes, service configuration, namespace creation, and endorsement policy changes. Endorsement policies control which identities must approve transaction results for specific namespaces or state domains.

Because policies reference MSP identities and roles, governance can evolve without hard-coding individual public keys into every rule. Organizations can rotate certificates, update trusted roots, or change administrators through policy-controlled configuration updates.

Clear policy design is critical. Before launching a network, organizations should agree who can administer the network, who can operate services, who can endorse each namespace, and how disputes or membership changes are handled.

## When to Use Separate Networks

Use separate Fabric-X networks when participants require separate ledgers, separate ordering histories, separate governance, or strong administrative isolation. Separate networks replace multi-channel isolation when separate ledgers or governance domains are required.

Namespaces are appropriate when applications can share network governance and ledger history while keeping state domains and endorsement policies distinct. Separate networks are appropriate when sharing those foundations would violate legal, regulatory, operational, or confidentiality requirements.

## See Also

- [Identity Management](identity.md)
- [Membership Service Provider](msp.md)
- [Endorsement and Other Policies](policies/policies.md)
- [Genesis Block, Block Store, and World State](ledger.md)
- [Namespaces](namespaces.md)
