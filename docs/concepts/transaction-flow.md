<!-- SPDX-License-Identifier: Apache-2.0 -->
# End-to-End Transaction Flow in Fabric-X

## Overview

Fabric-X follows execute-order-validate-commit. Applications create endorsed transactions, Arma orders transaction bytes into blocks, and the committer pipeline validates authorization and state before committing results.

The flow has two important read/status loops. Before submission, FSC nodes or custom endorsers may read committed state and policies through the Query Service to build endorsed transaction data. After ordering and commit processing, clients learn final status from the committer and can query committed state through the same database-backed read path.

## Flow Diagram

At a high level, Fabric-X starts with application execution outside the orderer. A client invokes an FSC view or custom endorser, which reads committed public state, policies, and configuration through the Query Service. Endorsers may also read local private databases when endorsement logic depends on privacy-preserving data that should not be exposed through the shared state path.

The endorsed transaction is then submitted to the Arma orderer nodes. Arma orders transaction bytes into blocks; it does not evaluate endorsement policies, run MVCC checks, or decide whether a transaction changes application state. Ordering gives the transaction a deterministic block position and makes the block available to committers.

Each committer validates and commits independently against its colocated database. The committer verifies signatures and namespace policies, checks read versions, applies valid writes, records final status, and returns that status to clients. Clients should use committer status and Query Service reads after commit to observe final state.

```mermaid
flowchart LR
    Client[Client App] --> Endorser[FSC Node / Custom Endorser]
    Endorser -->|read state / policies| Query[Query Service]
    Query --> DB[(State Database)]
    Endorser --> Tx[Signed transaction]
    Tx --> Router[Arma Router]
    Router --> Batcher[Arma Batcher]
    Batcher --> Consenter[Arma Consenter]
    Consenter --> Assembler[Arma Assembler]
    Assembler --> Block[Ordered block]
    Block --> Sidecar
    Sidecar --> Coordinator
    Coordinator --> Verifier
    Verifier --> Coordinator
    Coordinator --> VC[Validator-Committer]
    VC -->|valid writes + statuses| DB
    VC --> Coordinator
    Coordinator --> Sidecar
    Sidecar -->|status / notification| Client
    Client -->|post-commit query| Query
```

## Full Transaction Sequence

```mermaid
sequenceDiagram
    title Full Fabric-X Transaction Flow
    participant Client as Client App
    participant FSC as FSC View / Custom Endorser
    participant QS as Query Service
    participant O1 as Orderer 1
    participant O2 as Orderer 2
    participant O3 as Orderer 3
    participant O4 as Orderer 4
    participant CM1 as Committer 1
    participant DB1 as Database 1
    participant CM2 as Committer 2
    participant DB2 as Database 2
    participant CM3 as Committer 3
    participant DB3 as Database 3
    participant CM4 as Committer 4
    participant DB4 as Database 4

    Client->>FSC: Invoke application workflow
    FSC->>QS: Read committed state, config, and namespace policies
    QS->>DB1: Open consistent read-only view
    DB1-->>QS: State rows, versions, policies
    QS-->>FSC: Read response
    FSC->>FSC: Execute application logic and assemble proposal
    FSC-->>Client: Endorsed transaction bytes

    Note over Client,O4: Client submits the endorsed transaction to all orderers
    Client->>O1: Submit endorsed transaction
    Client->>O2: Submit endorsed transaction
    Client->>O3: Submit endorsed transaction
    Client->>O4: Submit endorsed transaction
    Note over O1,O4: Consensus and block creation across orderer nodes
    O1-->>CM1: Deliver ordered block
    O2-->>CM2: Deliver ordered block
    O3-->>CM3: Deliver ordered block
    O4-->>CM4: Deliver ordered block

    Note over CM1,CM4: Independent committers validate and commit without communicating with each other
    CM1->>DB1: Validate transaction, apply valid writes, record status
    CM2->>DB2: Validate transaction, apply valid writes, record status
    CM3->>DB3: Validate transaction, apply valid writes, record status
    CM4->>DB4: Validate transaction, apply valid writes, record status
    Note over CM1,CM4: Independent committers return status
    CM1-->>Client: Commit status / notification
    CM2-->>Client: Commit status / notification
    CM3-->>Client: Commit status / notification
    CM4-->>Client: Commit status / notification
    Client->>QS: Query committed state or status context
    QS->>DB1: Read committed view
    DB1-->>QS: Current state
    QS-->>Client: Query response
```

The full flow starts with execution, not ordering. Client code invokes an FSC view or custom endorser, and endorsement logic reads committed state through the Query Service. Those reads return database versions and policy/configuration data that become inputs to the transaction read set, write set, and signatures.

After endorsement, the client submits immutable transaction bytes to Arma. The overview treats each orderer node as one box and leaves router, batcher, consenter, and assembler details to the dedicated orderer sequence below. At this point the transaction is ordered, but it is not yet valid or committed.

The committer pipeline gives each ordered transaction a final outcome. In this overview, each committer is independent and colocated with its own database: each receives a block from its paired orderer, validates and commits locally, and returns status without communicating with other committers. The dedicated committer sequence below expands the internals of one committer pipeline. Clients should treat committer status, not orderer acceptance, as transaction finality.

## Detailed Signing Sequence

```mermaid
sequenceDiagram
    title Multi-Organization Signing with FSC Views and Custom Endorsers
    participant Client as Client App
    participant Initiator as Org1 FSC Initiator View
    participant O1FSC as Org1 FSC Endorser View
    participant O1LocalDB as Org1 Local Private DB
    participant O2FSC as Org2 FSC Endorser View
    participant O2LocalDB as Org2 Local Private DB
    participant O3Custom as Org3 Custom Endorser
    participant O3LocalDB as Org3 Local Private DB
    participant O4Custom as Org4 Custom Endorser
    participant O4LocalDB as Org4 Local Private DB
    participant QS as Query Service
    participant DB as State Database

    Client->>Initiator: Start transaction intent
    Initiator->>QS: Read namespace state and policy
    QS->>DB: Read committed versions and policy rows
    DB-->>QS: Versions, values, policy
    QS-->>Initiator: Proposal inputs
    Initiator->>Initiator: Build proposal and expected RW set

    par Org1 local FSC signing
        Initiator->>O1FSC: Execute FSC responder view
        O1FSC->>O1LocalDB: Read private data
        O1LocalDB-->>O1FSC: Private inputs
        O1FSC->>QS: Read state/policy
        QS-->>O1FSC: Read response
        O1FSC-->>Initiator: Org1 RW set, signature, and certificate
    and Org2 remote FSC signing
        Initiator->>O2FSC: Check Org1 RW set
        O2FSC->>QS: Read state/policy for RW-set check
        QS-->>O2FSC: State and policy response
        O2FSC->>O2LocalDB: Read private data
        O2LocalDB-->>O2FSC: Private inputs
        O2FSC-->>Initiator: Org2 signature and certificate
    and Org3 custom signing
        Initiator->>O3Custom: Check Org1 RW set
        O3Custom->>QS: Read state/policy for RW-set check
        QS-->>O3Custom: State and policy response
        O3Custom->>O3LocalDB: Read private data
        O3LocalDB-->>O3Custom: Private inputs
        O3Custom-->>Initiator: Org3 signature and certificate
    and Org4 custom signing
        Initiator->>O4Custom: Check Org1 RW set
        O4Custom->>QS: Read state/policy for RW-set check
        QS-->>O4Custom: State and policy response
        O4Custom->>O4LocalDB: Read private data
        O4LocalDB-->>O4Custom: Private inputs
        O4Custom-->>Initiator: Org4 signature and certificate
    end

    Initiator->>Initiator: Check policy satisfaction and assemble transaction
    Initiator-->>Client: Endorsed transaction envelope
```

**Phase 1: Execute / Endorse.** Application logic runs before ordering. In Fabric-X this is commonly implemented with Fabric Smart Client views or another application-level endorsement flow.

The signing sequence shows four organizations participating before the transaction reaches the orderer. Org1 drives the FSC initiator view, Org1 and Org2 can endorse through FSC responder views, and Org3 and Org4 can represent custom endorser services. The resulting transaction can mix endorsement mechanisms as long as final signatures satisfy the namespace policy checked later by the committer.

FSC views support interactive protocols. The initiator can prepare proposal inputs, contact other FSC nodes, exchange application messages, and collect signatures over the proposed read/write effects. Org1 generates the initial read/write set. Other endorsers verify that read/write set by reading committed state and policy through the Query Service, and each organization may also read a local private database for privacy-preserving endorsement inputs.

Custom endorsers model endorsement logic as external services. They can implement domain-specific validation or cryptographic signing outside FSC while still returning Fabric-X-compatible endorsement material. The diagram labels these calls as gRPC because custom endorsers are service-style participants, not local view invocations.

During this phase, an FSC node or custom endorser executes business logic and may read current committed state through the Query Service. The read results become part of the transaction's read set or otherwise influence the proposed write set. The result is a transaction with an identifier, namespace read/write information, endorsements or signatures for affected namespaces, and data needed by the committer to verify policies and validate reads.

Endorsement does not mutate the database. It produces signed evidence that organizations accepted proposed effects under current inputs. The committer later re-checks signatures, policies, and read versions, so endorsement success alone does not guarantee commit success.

## Detailed Orderer Sequence

```mermaid
sequenceDiagram
    title Arma Ordering with Four Orderer Nodes on Shard 0
    participant Client as Client App
    participant R1 as Org1 Router
    participant R2 as Org2 Router
    participant R3 as Org3 Router
    participant R4 as Org4 Router
    participant B1 as Org1 Batcher S0
    participant B2 as Org2 Batcher S0
    participant B3 as Org3 Batcher S0
    participant B4 as Org4 Batcher S0
    participant C1 as Org1 Consenter
    participant C2 as Org2 Consenter
    participant C3 as Org3 Consenter
    participant C4 as Org4 Consenter
    participant A1 as Org1 Assembler
    participant A2 as Org2 Assembler
    participant A3 as Org3 Assembler
    participant A4 as Org4 Assembler
    participant Sidecar as Committer Sidecar

    Client->>R1: Submit endorsed transaction tx
    Client->>R2: Submit endorsed transaction tx
    Client->>R3: Submit endorsed transaction tx
    Client->>R4: Submit endorsed transaction tx
    R1->>R1: Check envelope and route shard 0
    R2->>R2: Check envelope and route shard 0
    R3->>R3: Check envelope and route shard 0
    R4->>R4: Check envelope and route shard 0

    R1->>B1: Forward tx to primary batcher for shard 0
    R2->>B2: Forward tx to secondary batcher for shard 0
    R3->>B3: Forward tx to secondary batcher for shard 0
    R4->>B4: Forward tx to secondary batcher for shard 0

    B1->>B1: Create shard 0 batch and BAF
    B1->>B2: Forward shard 0 batch
    B1->>B3: Forward shard 0 batch
    B1->>B4: Forward shard 0 batch
    B1->>C1: Submit BAF for shard 0
    B2->>C2: Submit BAF for shard 0
    B3->>C3: Submit BAF for shard 0
    B4->>C4: Submit BAF for shard 0

    C1->>C2: SmartBFT pre-prepare / prepare / commit
    C2->>C3: SmartBFT prepare / commit messages
    C3->>C4: SmartBFT prepare / commit messages
    C4-->>C1: Quorum messages
    Note over C1,C4: Four organizations provide n=4 consensus replicas and quorum orders BAFs

    C1-->>A1: Ordered BAFs
    C2-->>A2: Ordered BAFs
    C3-->>A3: Ordered BAFs
    C4-->>A4: Ordered BAFs
    A1->>B1: Pull complete batch payload represented by BAF
    B1-->>A1: Complete shard 0 batch payload
    A1->>B2: Pull complete batch payload represented by BAF
    B2-->>A1: Complete shard 0 batch payload
    A1->>B3: Pull complete batch payload represented by BAF
    B3-->>A1: Complete shard 0 batch payload
    A1->>B4: Pull complete batch payload represented by BAF
    B4-->>A1: Complete shard 0 batch payload
    A1->>A1: Fuse payloads, attestations, and block metadata
    A1-->>Sidecar: Deliver ordered block
```

**Phase 2: Order with Arma.** Arma provides the ordering service. Routers accept client submissions and forward requests, batchers group requests into batches and produce batch attestation material, consenters run consensus over ordering metadata, and assemblers gather ordered batches and produce blocks.

The orderer sequence separates submission, batching, consensus, and assembly. Four orderer nodes are shown, each with a router, a shard-0 batcher, a consenter, and an assembler. Routers are the client-facing entry point and can exist in multiple organizations. They do not perform endorsement-policy or MVCC validation; they route accepted transaction bytes to the batcher shard responsible for that traffic.

Shard 0 receives the same example transaction through all routers. The primary batcher creates the shard batch and BAF, forwards the batch to secondary batchers, and each batcher submits BAF material for consensus.

The four consenters represent four organizations running the BFT ordering protocol. Consensus is over batch attestation fragments and ordering metadata, not over database state. With four replicas, the protocol can tolerate one Byzantine fault under the usual n=3f+1 model, while still producing a deterministic order for batches.

Assemblers complete the orderer output. They observe ordered attestations, pull the complete batch payload represented by the BAF from shard-0 batchers, and fuse the data into blocks. Delivery to the committer sidecar transfers ordered blocks into validation and commit processing. Ordering creates a deterministic block position for transaction bytes, but it does not mean the transaction will update world state.

## Detailed Committer Sequence

```mermaid
sequenceDiagram
    title Committer Pipeline with Parallel Batch Validation and Database Cluster
    participant Sidecar as Sidecar
    participant Coord as Coordinator
    participant V1 as Verifier 1
    participant V2 as Verifier 2
    participant VC1 as Validator-Committer 1
    participant VC2 as Validator-Committer 2
    participant DB1 as DB Node 1
    participant DB2 as DB Node 2
    participant DB3 as DB Node 3
    participant QS as Query Service
    participant Client as Client App

    Sidecar->>Sidecar: Fetch ordered block from assembler/orderer
    Sidecar->>Coord: Send block and metadata
    Coord->>Coord: Decode transactions and build dependency graph
    Coord->>Coord: Select dependency-free work

    par Signature checks for batch A
        Coord->>V1: Verify batch of transactions signatures against the namespace policy
        V1-->>Coord: Valid / invalid batch results
    and Signature checks for another batch
        Coord->>V2: Verify different batch of transactions signatures against the namespace policy
        V2-->>Coord: Valid / invalid batch results
    end

    Coord->>Coord: Mark failed-policy transactions invalid

    par MVCC validation and commit for batch A
        Coord->>VC1: Validate and commit batch of transactions
        VC1->>DB1: Read current versions and apply valid writes
        DB1->>DB2: Replicate or coordinate database transaction
        DB2-->>VC1: Batch commit response
        VC1-->>Coord: Commit statuses for batch
    and MVCC validation and commit for another batch
        Coord->>VC2: Validate and commit different batch of transactions
        VC2->>DB2: Read current versions and apply valid writes
        DB2->>DB3: Replicate or coordinate database transaction
        DB3-->>VC2: Batch commit response
        VC2-->>Coord: Commit statuses for different batch
    end

    Coord->>Coord: Merge statuses and advance block watermark
    Coord-->>Sidecar: Final status batch
    Sidecar-->>Client: Transaction status notification
    Client->>QS: Query committed state
    QS->>DB1: Read stable committed view
    DB1-->>QS: Current values and versions
    QS-->>Client: Query response
```

**Phase 3: Validate and Commit.** The committer pipeline processes each ordered block, turns ordered transactions into durable outcomes, and records final status for every transaction.

The committer sequence starts after ordering. The sidecar fetches or receives ordered blocks and streams them to the coordinator. The coordinator decodes transactions, extracts read/write sets, and builds a dependency graph so independent work can run in parallel without violating read/write ordering constraints.

Verifiers handle authorization checks. They validate signatures against namespace policies for assigned batches of transactions. Transactions that fail these checks receive final invalid statuses and do not move to MVCC validation, but the block pipeline continues processing other transactions.

Validator-committers perform state validation and persistence. They compare transaction read versions against committed database versions, reject MVCC conflicts, apply valid writes, and record statuses. Multiple validator-committers can process batches of transactions in parallel while the database cluster coordinates durability and consistency.

The database cluster is the source of committed truth for both write and read paths. Once validator-committers finish, the coordinator merges statuses and advances block progress, then the sidecar returns final status notifications. Valid transactions update namespace state. Invalid transactions do not update application state, but they still receive final statuses so clients can distinguish policy failures, MVCC conflicts, malformed transactions, and successful commits. Clients and endorsers use the Query Service after finality to observe stable committed state rather than reading from in-flight pipeline state.

## Read Path

The Query Service is not part of the commit path. It serves read-only access to committed state, namespace policies, and configuration through database views.

This read path is used both before and after commit processing. Before submission, endorsers read state and policies to build transaction proposals. After finality, clients and endorsers read the updated committed state and use status/notification results to decide follow-up work.

## Status Lifecycle

Transaction status is finalized by the committer. Common outcomes include committed, signature or policy invalid, MVCC conflict, and other validation failures defined by committer protobuf status values.

A transaction can therefore be ordered but not committed. Applications should wait for committer status before treating a transaction as final, then use the Query Service to observe committed state.

## See Also

- [Fabric-X Model](fabric-x-model.md)
- [Arma Ordering](../orderer/docs/architecture.md)
- [Committer Pipeline](../committer/docs/architecture.md)
- [Dependency Graph](../committer/docs/coordinator.md)
- [Fabric-X Committer Architecture](https://hyperledger.github.io/fabric-x-committer/architecture/)
- [Sidecar](https://hyperledger.github.io/fabric-x-committer/sidecar/)
- [Coordinator](https://hyperledger.github.io/fabric-x-committer/coordinator/)
- [Verification Service](https://hyperledger.github.io/fabric-x-committer/verification-service/)
- [Validator-Committer](https://hyperledger.github.io/fabric-x-committer/validator-committer/)
- [Query Service](https://hyperledger.github.io/fabric-x-committer/query-service/)
