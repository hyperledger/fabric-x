<!-- SPDX-License-Identifier: Apache-2.0 -->
# Frequently Asked Questions

## General

### What is Fabric-X?
Fabric-X is the next-generation Hyperledger Fabric with horizontally scalable ordering and parallel validation.

### How does Fabric-X differ from Fabric v2?
- **Ordering:** Arma BFT (200,000+ TPS) vs Raft/Kafka
- **Validation:** Parallel wave-based vs sequential
- **Endorsement:** FSC Views vs Chaincodes
- **Architecture:** Single channel with namespaces vs multi-channel

## Architecture

### How many channels does Fabric-X support?
Single channel only. Use namespaces for isolation.

### Are private data collections supported?
No. Use namespaces with appropriate access policies.

### What consensus does Arma use?
SmartBFT, a Byzantine Fault Tolerant protocol.

### How does sharding work?
Namespace-based sharding. Each shard handles specific namespaces.

## Performance

### What throughput can I expect?
200,000+ TPS with proper sharding and hardware.

### What is the latency?
50-100ms on the optimistic path.

### How do I scale?
Add more shards (horizontal scaling).

## Development

### Do I need to write chaincodes?
No. Use FSC Views for business logic.

### What languages are supported?
Go is primary. FSC SDK supports multiple languages.

### How do I implement custom endorsement?
See [Custom Endorser](../apps/custom-endorser.md).

## Deployment

### What are the minimum requirements?
- 4 nodes for BFT (3f+1, f=1)
- 8GB RAM per node
- SSD storage recommended

### Is TLS required?
Yes for production. Optional for development.

### Can I run on Kubernetes?
Yes. See deployment guides.

## Troubleshooting

### Where do I find logs?
Check `/var/log/fabric-x/` or container logs.

### How do I monitor the network?
See [Monitoring](../operations/monitoring.md).

### What if consensus fails?
Check [Troubleshooting](../operations/troubleshooting.md).
