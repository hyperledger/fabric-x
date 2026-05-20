<!-- SPDX-License-Identifier: Apache-2.0 -->
# Glossary

## Terms

### Arma
Horizontally scalable BFT ordering service with microservice architecture.

### BAF (Batch Attestation Fragment)
Arma-internal structure containing batch digest and attestation.

### Batcher
Arma microservice that bundles transactions into batches.

### Committer
Post-ordering service that validates and commits transactions.

### Consenter
Arma microservice that runs SmartBFT consensus.

### Coordinator
Committer service that orchestrates parallel validation.

### Endorser
Service that signs transactions using FSC.

### FSC (Fabric Smart Client)
Endorsement framework replacing chaincodes in Fabric-X.

### Namespace
Logical isolation unit within a single channel.

### Query Service
Read-only service for state queries (not part of validation pipeline).

### Router
Arma microservice that accepts and dispatches transactions.

### Shard
Independent processing unit in Arma.

### Sidecar
Committer service that fetches blocks from ordering service.

### SmartBFT
Byzantine Fault Tolerant consensus protocol.

### Threshold Signature
Combined signature from multiple parties.

### Validator-Committer (VC)
Service that performs MVCC validation and database commits.

### View
FSC interactive protocol defining transaction logic.
