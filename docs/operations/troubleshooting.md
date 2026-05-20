<!-- SPDX-License-Identifier: Apache-2.0 -->
# Fabric-X Troubleshooting Guide

This guide provides diagnostic approaches for common issues in Fabric-X deployments.

## Quick Diagnostics

### Check Component Health

Use gRPC health check protocol (no HTTP health endpoints):

```bash
# Arma components (replace ports with your configured values)
grpcurl -plaintext localhost:7052 grpc.health.v1.Health/Check  # Router
grpcurl -plaintext localhost:7053 grpc.health.v1.Health/Check  # Batcher
grpcurl -plaintext localhost:7051 grpc.health.v1.Health/Check  # Consenter
grpcurl -plaintext localhost:7050 grpc.health.v1.Health/Check  # Assembler

# Committer components
grpcurl -plaintext localhost:4001 grpc.health.v1.Health/Check  # Sidecar
grpcurl -plaintext localhost:9001 grpc.health.v1.Health/Check  # Coordinator
grpcurl -plaintext localhost:5001 grpc.health.v1.Health/Check  # Verifier
grpcurl -plaintext localhost:6001 grpc.health.v1.Health/Check  # VC
grpcurl -plaintext localhost:7001 grpc.health.v1.Health/Check  # Query
```

**Note:** Ports are configurable. Check your configuration files for actual values.

### Check Logs

```bash
# Direct execution (if logs redirected to files)
tail -f /var/log/arma/router.log
tail -f /var/log/committer/coordinator.log

# Kubernetes
kubectl logs -f deployment/arma-router
kubectl logs -f deployment/committer-coordinator

# Docker
docker logs -f <container-id>

# Note: systemd units not provided in repository
# If you create custom systemd units, use:
# journalctl -u arma-router -f --since "10 minutes ago"
```

### Network Connectivity

```bash
# Test Arma ports (replace with your configured ports)
nc -zv localhost 8050    # Router (RouterListenPort)
nc -zv localhost 6050    # Batcher (BatcherListenPort)
nc -zv localhost 7050    # Consenter (ConsenterListenPort)
nc -zv localhost 9050    # Assembler (AssemblerListenPort)

# Test Committer ports (replace with your configured ports)
nc -zv localhost 4001    # Sidecar (ServerListenPort)
nc -zv localhost 9001    # Coordinator (ListenPort)
nc -zv localhost 5001    # Verifier (ListenPort)
nc -zv localhost 6001    # VC (ListenPort)
nc -zv localhost 7001    # Query (ListenPort)
```

### Default Port Reference Table

**Arma Ordering Service** (configurable via `local_config.yaml`):

| Component | Service Port (Default) | Config Field | Monitoring Port (Default) | Config Field |
|-----------|----------------------|--------------|--------------------------|--------------|
| Router | 8050 | `RouterListenPort` | 9090 | `MonitoringListenPort` |
| Batcher | 6050 | `BatcherListenPort` | 9091 | `MonitoringListenPort` |
| Consenter | 7050 | `ConsenterListenPort` | 9092 | `MonitoringListenPort` |
| Assembler | 9050 | `AssemblerListenPort` | 9093 | `MonitoringListenPort` |

**Committer Pipeline** (configurable via service YAML configs):

| Component | Service Port (Default) | Config Field | Monitoring Port (Default) | Config Field |
|-----------|----------------------|--------------|--------------------------|--------------|
| Sidecar | 4001 | `server.listenPort` | 2114 | `metricsPort` |
| Coordinator | 9001 | `server.listenPort` | 2119 | `metricsPort` |
| Verifier | 5001 | `server.listenPort` | 2115 | `metricsPort` |
| VC | 6001 | `server.listenPort` | 2116 | `metricsPort` |
| Query | 7001 | `server.listenPort` | 2117 | `metricsPort` |

**Database**: 5433 (PostgreSQL/YugabyteDB, configurable via DB connection string)

> **Important:** All ports are **configurable**. Always verify actual ports in your configuration files before troubleshooting.

## Common Issues

### 1. Arma Components Won't Start

**Symptoms:** Process exits immediately, no logs

**Diagnostics:**
```bash
# Check configuration (arma binary location may vary)
./bin/arma router --config=config.yaml 2>&1
./bin/arma batcher --config=config.yaml 2>&1
./bin/arma consensus --config=config.yaml 2>&1
./bin/arma assembler --config=config.yaml 2>&1

# Verify crypto files exist
ls -la msp/keystore/
ls -la tls/server.key

# Check file permissions
chmod 600 tls/server.key
```

**Solutions:**
- Ensure `msp/` directory exists with proper structure
- Verify TLS certificates are readable
- Check `config.yaml` syntax with YAML validator
- Ensure config file path is correct

### 2. Consenter Cluster Won't Form

**Symptoms:** Consenters stuck waiting for leader

**Diagnostics:**
```bash
# Check logs for connection errors
grep -i "connection\|dial" /var/log/arma/consenter.log

# Verify all 4 nodes have unique PartyID
grep PartyID *.yaml

# Check network connectivity between nodes
ping consenter2.example.com
nc -zv consenter2.example.com 7051
```

**Solutions:**
- Ensure shared_config.yaml is identical on all nodes
- Verify TLS certificates allow mutual authentication
- Check firewall rules allow inter-node communication

### 3. Transaction Submission Fails

**Symptoms:** Client receives errors sending to Router

**Diagnostics:**
```bash
# Test with armageddon (binary in bin/ directory)
./bin/armageddon submit \
  --config=user_config.yaml \
  --transactions=1 \
  --rate=1

# Check Router logs
grep -i "submit\|error" /var/log/arma/router.log
```

**Common causes:**
- TLS certificate mismatch
- Client signature verification enabled but client lacks certs
- Batcher shard not available for routing
- Router cannot connect to batchers

### 4. Committer Validation Failures

**Symptoms:** Transactions fail during validation phase

**Diagnostics:**
```bash
# Check Verifier logs (location depends on deployment)
grep -i "validation\|error" /var/log/committer/verifier.log

# Monitor Coordinator metrics (replace with your port)
curl http://localhost:2119/metrics | grep coordinator

# Test database connectivity (YugabyteDB default port 5433, PostgreSQL varies)
psql -h localhost -p 5433 -U fabricx -d committer -c "SELECT 1;"
```

**Solutions:**
- Ensure Coordinator started before other services
- Verify database schema initialized (VC service auto-creates schema on startup)
- Check service addresses match configuration
- Verify database is running and accessible

### 5. High Latency in Transaction Processing

**Diagnostics:**
```bash
# Check metrics (replace ports with your configured values)
curl http://localhost:9090/metrics | grep consensus
curl http://localhost:9091/metrics | grep batcher

# Monitor resource usage
top -p $(pgrep arma)
iostat -x 1
```

**Solutions:**
- Scale Batcher shards horizontally
- Increase Consenter resources (CPU-bound)
- Check disk I/O on state database
- Adjust BatchTimeout in config
- Check network latency between nodes

### 6. Database Connection Errors

**Symptoms:** Committer services cannot connect to database

**Diagnostics:**
```bash
# Test connection (port depends on your DB - YugabyteDB default 5433, PostgreSQL varies)
psql -h localhost -p 5433 -U fabricx -d committer -c "SELECT version();"

# Check connection limit
psql -c "SELECT count(*) FROM pg_stat_activity;"

# Review slow queries (requires pg_stat_statements extension)
psql -c "SELECT query, mean_exec_time FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;"
```

**Solutions:**
- Verify database is running
- Check connection pool settings in service configs
- Increase `max_connections` in postgresql.conf (PostgreSQL) or tablet config (YugabyteDB)
- Ensure VC service has initialized the schema

## Recovery Procedures

### Restart Single Component

```bash
# For direct execution (send SIGTERM, then restart)
pkill -f "arma router"
./bin/arma router --config=config.yaml &

# Docker Compose
docker-compose restart router

# Kubernetes
kubectl rollout restart deployment/arma-router

# Note: systemd units not provided in repository
# If you created custom systemd units, use:
# systemctl restart arma-router
```

### Restart Committer Pipeline

**Stop order** (reverse dependencies):
```bash
# Sidecar depends on Coordinator
pkill -f "committer.*sidecar"

# Query depends on VC
pkill -f "committer.*query"

# Verifier depends on Coordinator
pkill -f "committer.*verifier"

# VC depends on Coordinator
pkill -f "committer.*vc"

# Coordinator stops last
pkill -f "committer.*coordinator"
```

**Start order** (respect dependencies):
```bash
# 1. Coordinator starts first
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
```

### Database Recovery

```bash
# VC service auto-initializes schema on startup
# To force reinit, drop and recreate database (use with caution)
dropdb -U fabricx committer
createdb -U fabricx committer
# Restart VC service - it will recreate schema

# Restore from backup
pg_restore -U fabricx -d committer backup.dump
```

## Debugging Tools

### Armageddon Test Commands

```bash
# Test config generation
./bin/armageddon showtemplate

# Test transaction flow
./bin/armageddon submit --config=user.yaml --transactions=100 --rate=10
./bin/armageddon load --config=user.yaml --transactions=1000 --rate="100 200 500"
./bin/armageddon receive --config=user.yaml --expectedTxs=100 --pullFromPartyId=1
```

### pprof Profiling

Components expose pprof endpoints on their monitoring port under `/debug/pprof/`:

```bash
# CPU profile (replace 9090 with your component's monitoring port)
curl http://localhost:9090/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Memory profile
curl http://localhost:9090/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Other profiles
curl http://localhost:9090/debug/pprof/cmdline
curl http://localhost:9090/debug/pprof/trace?seconds=30 > trace.out
```

## Support Resources

- Documentation: https://github.com/hyperledger/fabric-x-docs
- Issues: https://github.com/hyperledger/fabric-x/issues
- Mailing List: fabric-x@lists.hyperledger.org
