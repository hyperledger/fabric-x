<!-- SPDX-License-Identifier: Apache-2.0 -->
# Logging Guide

This document provides guidance for logging in Fabric-X networks.

## Log Configuration

Logging is configured in each component's configuration file via the `LogSpec` field.

### Available Log Levels

| Level | Description |
|-------|-------------|
| `debug` | Detailed debug information |
| `info` | General operational information |
| `warning` | Warning conditions |
| `error` | Error conditions |
| `fatal` | Fatal errors (process exits) |

### Configuration Example

```yaml
General:
  LogSpec: info
```

## Log Output

By default, logs are written to:
- **stdout/stderr** for all deployments
- Log file path if explicitly configured

### Viewing Logs

```bash
# For systemd deployments (if configured)
journalctl -u arma-router -f

# For direct binary execution
tail -f /var/log/arma/router.log

# For Docker
docker logs -f <container-id>

# For Kubernetes
kubectl logs -f -l app=arma-router
kubectl logs -f -l app=committer-coordinator
```

## Log Format

Logs use the fabric-lib-go `flogging` format:
```
2024-01-15 10:30:45.123 INFO [Router1] Router listening on 127.0.0.1:7050, PartyID: 1
2024-01-15 10:30:46.234 INFO [Batcher1Shard1] Batcher listening on 127.0.0.1:7053
2024-01-15 10:30:47.345 INFO [Consensus1] Consensus listening on 127.0.0.1:7051
```

### Component-Specific Log Prefixes

- **Router**: `[Router<PartyID>]` - ROUTER_METRICS messages
- **Batcher**: `[Batcher<PartyID>Shard<ShardID>]` - BATCHER_METRICS messages
- **Consenter**: `[Consensus<PartyID>]` - CONSENSUS_METRICS messages
- **Assembler**: `[Assembler<PartyID>]` - ASSEMBLER_METRICS messages
- **Committer services**: `[coordinator]`, `[verifier]`, `[validator-committer]`, `[sidecar]`, `[query]`

## Troubleshooting with Logs

### Common Log Locations

| Component | Default Location |
|-----------|------------------|
| Arma (all roles) | stdout/stderr |
| Committer (all services) | stdout/stderr |

**Note:** Redirect to files via shell redirection or configure your process manager (systemd, Docker, Kubernetes) to capture logs.

### Debug Logging

Enable debug logs for troubleshooting:

```yaml
# Temporary debug logging
General:
  LogSpec: debug
```

**Warning:** Debug logging significantly impacts performance. Only enable for troubleshooting.

**Note:** JSON/structured logging is **not supported**. The `flogging` library only supports console text format.

## Log Retention

### Configuration

No built-in log rotation - use external tools:

```bash
# logrotate configuration
/var/log/arma/*.log {
    daily
    rotate 7
    compress
    missingok
    notifempty
}
```

### For Kubernetes Deployments

```bash
# View logs
kubectl logs -f -l app=arma-router
kubectl logs -f -l app=committer-coordinator
```

## Audit Logging

Audit logging is not implemented as a separate feature. Security-relevant events are logged at `info` or `warning` level:

- Certificate validation failures
- Authentication errors
- Permission denied events

## Best Practices

1. **Use `info` level in production** - Balances detail and performance
2. **Enable `debug` only for troubleshooting** - Revert after issue resolution
3. **Centralize logs** - Use Fluentd/Fluent Bit to aggregate logs
4. **Monitor error rates** - Alert on increasing error log frequency

---

**See Also:** [Monitoring Guide](./monitoring.md) | [Troubleshooting](./troubleshooting.md)
