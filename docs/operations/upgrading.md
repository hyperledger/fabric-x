<!-- SPDX-License-Identifier: Apache-2.0 -->
# Upgrade Procedures

This guide covers upgrading Fabric-X components.

## Pre-Upgrade Checklist

1. **Backup database:**
   ```bash
   pg_dump -U fabricx committer > backup-$(date +%Y%m%d).sql
   ```

2. **Backup Arma block storage:**
   ```bash
   # Use your actual FileStore.Location from local_config.yaml
   tar czf arma-blocks-backup-$(date +%Y%m%d).tar.gz /var/lib/arma
   ```

3. **Verify current versions:**
   ```bash
   # Check fabric-x-orderer version
   ./bin/arma version
   # or from source
   git describe --tags
   
   # Check fabric-x-committer version
   ./bin/committer version
   # or from source
   git describe --tags
   ```

## Upgrade Order

```
1. Arma binaries (Router, Batcher, Consenter, Assembler)
2. Committer binaries (Coordinator, VC, Verifier, Query, Sidecar)
3. Database schema (auto-migrated by VC service on startup)
```

> **Note on Database Schema Migration:** The VC (Validator-Committer) service automatically creates and migrates the database schema on startup. Schema changes are detected and applied during VC initialization. Manual migration is only needed for troubleshooting or forced resets.

## Arma Upgrade

### Single Node

```bash
# Stop services (method depends on your deployment)
# For direct execution
pkill -f "arma router"
pkill -f "arma batcher"
pkill -f "arma consensus"
pkill -f "arma assembler"

# For systemd (if you created service units)
systemctl stop arma-router arma-batcher arma-consenter arma-assembler

# Backup binary
cp arma arma.backup

# Deploy new version
cp arma-new ./bin/arma

# Restart (order matters for clean startup)
./bin/arma consensus --config=consenter_config.yaml &
sleep 5
./bin/arma batcher --config=batcher_config.yaml &
./bin/arma assembler --config=assembler_config.yaml &
sleep 5
./bin/arma router --config=router_config.yaml &

# Verify (use gRPC health check, not HTTP)
grpcurl -plaintext localhost:7051 grpc.health.v1.Health/Check  # Consenter
grpcurl -plaintext localhost:9092/metrics  # Check metrics endpoint
```

### Rolling Upgrade (Cluster)

```bash
# Upgrade one node at a time, maintaining quorum (3/4 consenters needed)
# Start with consenters, then batchers, then assemblers, then routers
for node in consenter1 consenter2 consenter3 consenter4; do
  # Stop consenter on this node
  ssh $node "pkill -f 'arma consensus'"
  
  # Deploy new binary
  scp arma-new $node:/usr/local/bin/arma
  
  # Restart consenter
  ssh $node "./bin/arma consensus --config=consenter_config.yaml &"
  
  sleep 10  # Wait for node to rejoin
  
  # Verify node is healthy (use gRPC health check)
  ssh $node "grpcurl -plaintext localhost:7051 grpc.health.v1.Health/Check"
  
  # Wait before next node
  sleep 30
done
```

## Committer Upgrade

### Rolling Restart

**Shutdown Order** (reverse dependencies):

```bash
# 1. Stop sidecar first (depends on Coordinator)
pkill -f "committer.*sidecar"

# 2. Stop query (depends on VC)
pkill -f "committer.*query"

# 3. Stop verifier (depends on Coordinator)
pkill -f "committer.*verifier"

# 4. Stop VC (depends on Coordinator)
pkill -f "committer.*vc"

# 5. Stop coordinator last
pkill -f "committer.*coordinator"
```

**Startup Order** (respect dependencies):

```bash
# Deploy new binary
cp committer-new ./bin/committer

# 1. Coordinator must start first
./bin/committer start coordinator -c coordinator.yaml &
sleep 5

# 2. VC starts second (initializes database schema)
./bin/committer start vc -c vc.yaml &
sleep 3

# 3. Verifier and Query can start in parallel
./bin/committer start verifier -c verifier.yaml &
./bin/committer start query -c query.yaml &
sleep 3

# 4. Sidecar starts last (connects to Coordinator)
./bin/committer start sidecar -c sidecar.yaml &

# Verify all services are healthy
grpcurl -plaintext localhost:9001 grpc.health.v1.Health/Check  # Coordinator
grpcurl -plaintext localhost:6001 grpc.health.v1.Health/Check  # VC
```

## Database Schema Migration

Schema is **automatically created/updated** by the VC service on startup. Manual migration is rarely needed.

To force schema recreation (use with caution - data loss):

```bash
# 1. Backup first
pg_dump -U fabricx committer > committer-backup.sql

# 2. Stop all committer services
pkill -f "committer.*sidecar"
pkill -f "committer.*query"
pkill -f "committer.*verifier"
pkill -f "committer.*vc"
pkill -f "committer.*coordinator"

# 3. Drop database
dropdb -U fabricx committer

# 4. Create new database
createdb -U fabricx committer

# 5. Start VC service (it will initialize schema)
./bin/committer start vc -c vc.yaml &
sleep 5

# 6. Start other committer services
./bin/committer start coordinator -c coordinator.yaml &
./bin/committer start verifier -c verifier.yaml &
./bin/committer start query -c query.yaml &
./bin/committer start sidecar -c sidecar.yaml &
```

## Rollback

If issues occur after upgrade:

```bash
# 1. Backup current state before rollback (critical step)
# Stop services first
pkill -f "arma"
pkill -f "committer"

# Backup current database
pg_dump -U fabricx committer > rollback-backup-$(date +%Y%m%d).sql

# Backup current block storage
tar czf arma-blocks-rollback-$(date +%Y%m%d).tar.gz /var/lib/arma

# 2. Restore old Arma binary
cp arma.backup ./bin/arma

# 3. Restore old committer binary
cp committer.backup ./bin/committer

# 4. If database schema changed, restore from pre-upgrade backup
# (Use the backup from BEFORE the upgrade, not the one just created)
psql -U fabricx -d committer < backup-$(date +%Y%m%d).sql

# 5. Restart Arma services with old binary
./bin/arma consensus --config=consenter_config.yaml &
sleep 5
./bin/arma batcher --config=batcher_config.yaml &
./bin/arma assembler --config=assembler_config.yaml &
sleep 5
./bin/arma router --config=router_config.yaml &

# 6. Restart committer services in correct order
./bin/committer start coordinator -c coordinator.yaml &
sleep 5
./bin/committer start vc -c vc.yaml &
sleep 3
./bin/committer start verifier -c verifier.yaml &
./bin/committer start query -c query.yaml &
sleep 3
./bin/committer start sidecar -c sidecar.yaml &

# 7. Verify all services are healthy
grpcurl -plaintext localhost:7051 grpc.health.v1.Health/Check  # Consenter
grpcurl -plaintext localhost:9001 grpc.health.v1.Health/Check  # Coordinator
```
./bin/committer start vc -c vc.yaml &
# ... etc
```

---

**See Also:** [Troubleshooting](./troubleshooting.md) | [Production Guide](../deployment/overview.md)
