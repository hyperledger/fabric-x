<!-- SPDX-License-Identifier: Apache-2.0 -->
# Fabric-X Architecture Reference

## Overview

Fabric-X represents a fundamental re-architecture of Hyperledger Fabric, designed from the ground up for **mission-critical enterprise workloads** requiring Byzantine Fault Tolerance, horizontal scalability, and high throughput. This document provides a comprehensive architectural reference covering system design, major components, interaction patterns, and guiding principles.

## Architecture Topics

- [Detailed Transaction Architecture](transaction-flow.md) — end-to-end transaction-flow diagrams and service interactions.
- [Arma Ordering](../orderer/docs/architecture.md) — ordering-service architecture.
- Router — client request admission and routing.
- Batcher — transaction batching and batch data service.
- Consenter — BFT ordering decisions.
- Assembler — block assembly and delivery.
- [Committer Pipeline](../committer/docs/architecture.md) — validation and commit architecture.
- [Sidecar](../committer/docs/sidecar.md) — block ingestion and delivery.
- [Coordinator](../committer/docs/coordinator.md) — dependency analysis and orchestration.
- [Verifier](../committer/docs/verification-service.md) — signature and namespace-policy verification.
- [Validator-Committer](../committer/docs/validator-committer.md) — MVCC validation and database commit.

!!! info "Performance Note"
    Fabric-X is designed for high-throughput deployments. Actual throughput depends on workload, hardware, shard count, database, and network conditions. Run benchmarks for deployment-specific numbers.

### Key Architectural Innovations

| Innovation | Fabric Classic | Fabric-X | Improvement |
|------------|---------------|----------|-------------|
| **Ordering Service** | Monolithic orderer | **Arma**: 4 microservices (Router, Batcher, Consenter, Assembler) | Horizontal scalability, BFT consensus |
| **Commitment** | Monolithic peer | **Pipeline services**: Sidecar, Coordinator, Verifier, Validator-Committer | Parallel validation and independent scaling |
| **Endorsement** | Fabric peer execution path | **FSC views** or custom endorsers (native processes) | Native-process endorsement path |
| **Consensus** | Raft (Crash Fault Tolerant) | **SmartBFT** (Byzantine Fault Tolerant) | Tolerates malicious nodes |
| **Network Model** | Multi-channel complexity | **Single channel + namespaces** | Simplified operations, logical isolation |
| **Storage** | Peer-local state database | **Sharded PostgreSQL/YugabyteDB** | Horizontal scalability, rich queries |

---

## Design Constraints

| Constraint | Status | Rationale |
|------------|--------|-----------|
| **mTLS** | Mandatory | All inter-node communication requires mutual TLS. External TLS (`TLS.Enabled`) affects only external APIs, but internal mTLS is always enforced. |
| **Single Channel** | Architectural choice | Namespaces replace channels for logical isolation. Multi-channel not supported. |
| **No Private Data Collections** | Not implemented | Use namespace isolation + application-level encryption instead. |

---

## System Architecture

### High-Level Architecture

```mermaid
flowchart TB
    subgraph network["FABRIC-X NETWORK (Single Channel + Namespaces)"]
        direction TB
        subgraph client["CLIENT LAYER"]
            FSC["FSC App<br/>(FSC View)"]
            Custom["Custom App<br/>(REST/gRPC)"]
            SDK["SDK Client<br/>(Go/Node)"]
        end

        submit["Transaction Submit"]

        subgraph arma["ARMA / ORDERERX (one party view)"]
            R1["R1<br/>Router"]
            B1S1["B1s1<br/>Batcher Shard 1"]
            B1S2["B1s2<br/>Batcher Shard 2"]
            C1["C1<br/>Consenter"]
            OtherC["Other consenters<br/>(SmartBFT cluster)"]
            A1["A1<br/>Assembler"]
            R1 -->|"Hr(tx)"| B1S1
            R1 -->|"Hr(tx)"| B1S2
            B1S1 -->|BAF digest| C1
            B1S2 -->|BAF digest| C1
            C1 <-->|consensus on digests| OtherC
            C1 -->|ordered BA list| A1
            B1S1 -. batches .-> A1
            B1S2 -. batches .-> A1
        end

        subgraph committer["COMMITTER PIPELINE"]
            Sidecar["Sidecar<br/>(Delivery)"]
            Coordinator["Coordinator<br/>(Dependency Graph)"]
            Verifier["Verifier<br/>(Signatures/Policy)"]
            VC["VC<br/>(Validator/Committer)"]
            Sidecar --> Coordinator
            Coordinator --> Verifier
            Verifier --> VC
        end

        DB[("PostgreSQL / YugabyteDB")]
        QueryService["Query Service"]
    end

    FSC --> submit
    Custom --> submit
    SDK --> submit
    submit --> R1
    A1 -->|HLF blocks| Sidecar
    VC --> DB
    QueryService -. reads .-> DB

    style network fill:#f5f5f5,stroke:#333
    style client fill:#e3f2fd,stroke:#333
    style arma fill:#fff3e0,stroke:#333
    style committer fill:#e8f5e9,stroke:#333
```

### Architectural Layers

| Layer | Components | Responsibility |
|-------|------------|----------------|
| **Client** | FSC Apps, Custom Apps, SDKs | Transaction submission, state queries |
| **Ordering** | Arma (Router, Batcher, Consenter, Assembler) | Transaction ordering, BFT consensus, block assembly |
| **Commitment** | Pipeline (Sidecar, Coordinator, Verifier, Validator-Committer (VC)) | Parallel validation, policy enforcement, state updates |
| **Query** | Query Service | Client state queries (NOT in validation pipeline) |
| **Storage** | Sharded PostgreSQL/YugabyteDB | World state persistence, transaction log |

---

## Major Components

### 1. Arma Ordering Service

**Purpose:** Byzantine Fault Tolerant ordering service that receives transactions, reaches consensus on their order, and produces blocks.

**Architecture:** each Arma party replaces one traditional OS node and contains a router, one batcher per shard, a consenter, and an assembler. Parties run `3f + 1` SmartBFT consenters to tolerate `f` Byzantine parties. Arma separates full transaction dissemination from consensus by ordering compact batch digests.

```mermaid
flowchart TB
    E["Client / Endorser"] -->|tx| R1
    E -->|tx| R2
    E -->|tx| R3

    subgraph P1["Party 1"]
        R1["R1<br/>Router"]
        B1S1["B1s1<br/>Batcher S1"]
        B1S2["B1s2<br/>Batcher S2"]
        C1["C1<br/>Consenter"]
        A1["A1<br/>Assembler"]
        R1 -->|"Hr(tx)"| B1S1
        R1 -->|"Hr(tx)"| B1S2
        B1S1 -->|BAF| C1
        B1S2 -->|BAF| C1
        C1 -->|ordered BA list| A1
    end

    subgraph P2["Party 2"]
        R2["R2<br/>Router"]
        B2S1["B2s1<br/>Batcher S1"]
        B2S2["B2s2<br/>Batcher S2"]
        C2["C2<br/>Consenter"]
        A2["A2<br/>Assembler"]
        R2 -->|"Hr(tx)"| B2S1
        R2 -->|"Hr(tx)"| B2S2
        B2S1 -->|BAF| C2
        B2S2 -->|BAF| C2
        C2 -->|ordered BA list| A2
    end

    subgraph P3["Party 3"]
        R3["R3<br/>Router"]
        B3S1["B3s1<br/>Batcher S1"]
        B3S2["B3s2<br/>Batcher S2"]
        C3["C3<br/>Consenter"]
        A3["A3<br/>Assembler"]
        R3 -->|"Hr(tx)"| B3S1
        R3 -->|"Hr(tx)"| B3S2
        B3S1 -->|BAF| C3
        B3S2 -->|BAF| C3
        C3 -->|ordered BA list| A3
    end

    B1S1 <-->|shard 1 batcher comms| B2S1
    B2S1 <-->|shard 1 batcher comms| B3S1
    B1S2 <-->|shard 2 batcher comms| B2S2
    B2S2 <-->|shard 2 batcher comms| B3S2

    C1 <-->|SmartBFT| C2
    C2 <-->|SmartBFT| C3
    C1 <-->|SmartBFT| C3

    B1S1 -. batches .-> A1
    B1S2 -. batches .-> A1
    B2S1 -. batches .-> A1
    B2S2 -. batches .-> A1
    B3S1 -. batches .-> A1
    B3S2 -. batches .-> A1

    B1S1 -. batches .-> A2
    B1S2 -. batches .-> A2
    B2S1 -. batches .-> A2
    B2S2 -. batches .-> A2
    B3S1 -. batches .-> A2
    B3S2 -. batches .-> A2

    B1S1 -. batches .-> A3
    B1S2 -. batches .-> A3
    B2S1 -. batches .-> A3
    B2S2 -. batches .-> A3
    B3S1 -. batches .-> A3
    B3S2 -. batches .-> A3

    A1 -->|HLF blocks| SC["Sidecar / Committer"]
    A2 -->|HLF blocks| SC
    A3 -->|HLF blocks| SC

    style P1 fill:#e3f2fd,stroke:#333
    style P2 fill:#e8f5e9,stroke:#333
    style P3 fill:#fff3e0,stroke:#333
```

#### Component Responsibilities

| Component | Role | Key Responsibilities | Performance Target |
|-----------|------|---------------------|-------------------|
| **Router** | Entry point | Validate transactions, authenticate clients, compute deterministic `Hr(tx)` shard mapping | Horizontally scalable |
| **Batcher** | Transaction grouping | Batch per shard, persist batches, disseminate primary batches to secondaries, generate BAFs | Scales by shard count |
| **Consenter** | Consensus | Run SmartBFT on BAF digests, order Batch Attestations, order complaint votes | BFT finality |
| **Assembler** | Block assembly | Pull ordered metadata from own consenter, fetch batches from shard batchers, construct Fabric blocks | Parallel block assembly |

> **Note:** BAF (Batch Attestation Fragment) is an internal Arma structure used for consensus. It is not exposed to the Committer pipeline.

#### VC Internal Structure

The Validator-Committer (VC) service consists of three internal components:

| Component | Responsibility |
|-----------|----------------|
| **Preparer** | Prepares transactions for validation, checks namespace policies |
| **Validator** | Performs MVCC conflict detection, validates read-write sets |
| **Committer** | Commits validated transactions to database, updates world state |
