<!-- SPDX-License-Identifier: Apache-2.0 -->
# Monitoring and Metrics Guide

This guide covers monitoring Fabric-X components using Prometheus.

## Prometheus Setup

All Arma and Committer components expose Prometheus metrics endpoints.

### Endpoint Configuration

Monitoring ports are **configurable** via `MonitoringListenPort` in Arma component configs and `metricsPort` in Committer service configs. **The ports shown below are DEFAULTS only** - always verify your actual configuration.

| Component | Default Port | Config Field | Endpoint |
|-----------|--------------|--------------|----------|
| Arma Router | 9090 | `MonitoringListenPort` | `http://localhost:9090/metrics` |
| Arma Batcher | 9091 | `MonitoringListenPort` | `http://localhost:9091/metrics` |
| Arma Consenter | 9092 | `MonitoringListenPort` | `http://localhost:9092/metrics` |
| Arma Assembler | 9093 | `MonitoringListenPort` | `http://localhost:9093/metrics` |
| Committer Sidecar | 2114 | `metricsPort` | `http://localhost:2114/metrics` |
| Committer Verifier | 2115 | `metricsPort` | `http://localhost:2115/metrics` |
| Committer VC | 2116 | `metricsPort` | `http://localhost:2116/metrics` |
| Committer Query | 2117 | `metricsPort` | `http://localhost:2117/metrics` |
| Committer Coordinator | 2119 | `metricsPort` | `http://localhost:2119/metrics` |

> **⚠️ Important:** All monitoring ports are **CONFIGURABLE**. The values above are factory defaults. Always check your component configuration files (`local_config.yaml` for Arma, service YAMLs for Committer) for actual port assignments.

### Basic Prometheus Configuration

```yaml
scrape_configs:
  - job_name: 'arma'
    static_configs:
      - targets:
        - 'localhost:9090'  # Router (check config)
        - 'localhost:9091'  # Batcher (check config)
        - 'localhost:9092'  # Consenter (check config)
        - 'localhost:9093'  # Assembler (check config)

  - job_name: 'committer'
    static_configs:
      - targets:
        - 'localhost:2114'  # Sidecar
        - 'localhost:2115'  # Verifier
        - 'localhost:2116'  # VC
        - 'localhost:2117'  # Query
        - 'localhost:2119'  # Coordinator
```

## Arma Metrics

Metrics are exposed via the fabric-lib-go metrics library and Prometheus.

### Standard Metrics

Common metrics available on all components:

- `go_gc_duration_seconds` - GC duration
- `go_goroutines` - Number of goroutines
- `go_memstats_*` - Memory statistics
- `process_cpu_seconds_total` - CPU usage
- `process_open_fds` - Open file descriptors

### Consenter-Specific Metrics

Consensus metrics from Consenter (`consensus_` prefix):

- `consensus_decisions_count` - Total decisions made
- `consensus_blocks_count` - Total blocks ordered
- `consensus_bafs_count` - Batch attestation fragments received
- `consensus_complaints_count` - Complaints received
- `consensus_txs_count` - Total transactions ordered

### Batcher-Specific Metrics

Batcher metrics (`batcher_` prefix):

- `batcher_current_role` - Current role (1=primary, 2=secondary)
- `batcher_mempool_size` - Current mempool size
- `batcher_role_changes_total` - Total role changes
- `batcher_batches_created_total` - Total batches created
- `batcher_batches_pulled_total` - Total batches pulled
- `batcher_batched_txs_total` - Total transactions batched
- `batcher_router_txs_total` - Total transactions from router
- `batcher_complaints_total` - Total complaints sent
- `batcher_first_resends_total` - Total first resends

### From Integration Tests

The integration tests verify metrics capture patterns like:

```go
// From testutil/network_utils.go
FetchPrometheusMetricValue(t, re, url)  // Fetches metric from /metrics endpoint
CaptureArmaNodePrometheusServiceURL(t, node)  // Gets metrics URL from log output
```

## Health Checks

### gRPC Health Checks

For gRPC services, use the standard gRPC health check protocol:

```bash
grpcurl -plaintext localhost:5001 grpc.health.v1.Health/Check
```

## Log Monitoring

Configure `LogSpec` in component configs to control log verbosity:

```yaml
General:
  LogSpec: info  # Options: debug, info, warning, error, fatal
```

## Key Alerts

| Alert | Query | Severity |
|-------|-------|----------|
| Component Down | `up{job="arma"} == 0` | Critical |
| High Latency | `histogram_quantile(0.95, rate(request_duration_seconds_bucket[5m])) > 5` | Warning |
| Memory Usage | `process_resident_memory_bytes / 1024 / 1024 > 4096` | Warning |
| Consensus Stalled | `rate(consensus_decisions_count[1m]) == 0` | Critical |

## Troubleshooting Metrics

### No metrics showing

1. Check monitoring port is configured in component config:
   ```yaml
   General:
     MonitoringListenPort: 9090  # Your configured port
     MonitoringListenAddress: 0.0.0.0
   ```

2. Verify firewall allows access to monitoring port

3. Check component logs for errors:
   ```bash
   # Check logs (location depends on deployment)
   tail -f /var/log/arma/router.log
   # or for Docker
   docker logs <container-id>
   # or for Kubernetes
   kubectl logs -l app=arma-router
   ```

### High metric cardinality

- Limit label values (avoid unbounded values like transaction IDs)
- Use aggregation for high-cardinality histograms

---

**See Also:** [Logging Guide](./logging.md) | [Troubleshooting](./troubleshooting.md)
