<!-- SPDX-License-Identifier: Apache-2.0 -->
# Fabric-X Use Cases

Fabric-X is purpose-built for high-assurance, regulated financial infrastructure. Its Byzantine Fault Tolerant (BFT) consensus, horizontal scalability, and 100K+ TPS throughput make it ideal for mission-critical payment and settlement systems.

## CBDC (Central Bank Digital Currency)

Central banks require infrastructure that guarantees finality and safety through BFT consensus, which ensures transaction finality even when malicious actors are present. The system tolerates up to one-third Byzantine nodes while maintaining integrity. Sovereign control is achieved through a permissioned architecture that enables central bank governance over all network participants, ensuring that monetary authority remains with the issuing institution.

Scale is critical for nationwide retail CBDC deployments, and Fabric-X handles volumes exceeding 100K TPS through horizontal sharding. This allows the system to grow alongside adoption without compromising performance. Resilience is built into the architecture with no single point of failure, as distributed validation occurs across trusted institutions, ensuring continuous operation even when individual nodes experience issues.

Fabric-X enables two-tier CBDC architectures where central banks operate core settlement while commercial banks run edge nodes for customer onboarding and redemption.

## Regulated Liability Networks

Financial institutions need shared infrastructure for interbank settlements, syndicated loans, and trade finance that maintains regulatory compliance. Auditability is achieved through complete transaction provenance with cryptographic guarantees, providing an immutable record of all financial activities. Privacy is maintained through application-level encryption within transactions, ensuring confidential data between counterparties is not exposed to the broader network.

Compliance requirements are met through permissioned access control with KYC/AML enforcement at onboarding, ensuring that only verified participants can join the network. Interoperability is supported through atomic cross-namespace transactions with separate endorsement policies, enabling complex multi-institution workflows without sacrificing security or consistency.

Fabric-X supports liability network topologies where each institution operates nodes while sharing a common settlement layer.

## Tokenization Platforms

Real-world asset (RWA) tokenization demands institutional-grade infrastructure capable of high throughput. The platform supports millions of token transfers daily across various asset classes including equity, bonds, funds, and commodities. Regulatory compliance is enforced through endorsement logic that implements transfer restrictions, investor accreditation requirements, and jurisdictional rules automatically.

Settlement finality is critical for tokenized assets, with Fabric-X providing DvP (Delivery vs Payment) and PvP (Payment vs Payment) capabilities featuring instant finality. Asset servicing is automated through endorsers, handling corporate actions, dividend distributions, and interest payments without manual intervention.

Fabric-X enables multi-asset platforms where issuers, custodians, and brokers share infrastructure while maintaining data segregation.

## High-Throughput Payment Systems

Payment system operators such as RTGS, ACH, and card networks require infrastructure capable of peak load handling. The system must process over 100K TPS during peak periods with linear horizontal scaling to accommodate growing transaction volumes. Low latency is essential for time-sensitive payments, with sub-second confirmation times ensuring that funds are available when needed.

Fault tolerance is built into the architecture, allowing the system to continue operations despite node failures or network partitions. An upgrade path is provided for seamless protocol upgrades without network downtime, ensuring that the system can evolve without disrupting critical payment infrastructure.

Fabric-X sharding architecture allows payment systems to scale by adding shards as volume grows, with cross-shard atomic transactions for end-to-end payment flows.

## Why Fabric-X?

Fabric-X provides Byzantine Fault Tolerance through BFT consensus that tolerates up to one-third malicious nodes while maintaining system integrity. Horizontal scalability is achieved through a sharded architecture where capacity can be added simply by adding more shards. The platform delivers sustained throughput of over 100K TPS that scales linearly with the number of shards deployed.

Transaction finality is instant, eliminating the need for probabilistic confirmation mechanisms. The permissioned nature of Fabric-X provides access control at the network and namespace levels, where each namespace defines its own endorsement policy. Privacy between parties must be implemented at the application layer through transaction payload encryption.

Endorsement logic is implemented through FSC views or custom endorsers with deterministic execution and upgrade governance, providing flexibility while maintaining security. Each namespace has its own endorsement policy, enabling different validation rules per use case. Interoperability is supported through cross-chain bridges and atomic cross-shard transactions, enabling complex multi-system workflows.

Fabric-X is designed for financial market infrastructure where safety, scalability, and regulatory compliance are non-negotiable.
