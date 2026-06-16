# MicroK8s Certificate Exporter
MicroK8s-focused Prometheus exporter for monitoring control-plane certificate expiration.
![Go](https://img.shields.io/badge/Go-1.26-blue?logo=go)
![Docker](https://img.shields.io/badge/Docker-Multi--Arch-2496ED?logo=docker&logoColor=white)
![Prometheus](https://img.shields.io/badge/Prometheus-Metrics-E6522C?logo=prometheus&logoColor=white)
![Platform](https://img.shields.io/badge/Platform-MicroK8s-orange)
![Architecture](https://img.shields.io/badge/Architecture-amd64%20%7C%20arm64-blue)
![License](https://img.shields.io/badge/License-MIT-green)

MicroK8s-focused Prometheus exporter for monitoring control-plane certificate expiration.

This exporter runs as a DaemonSet on MicroK8s control-plane nodes, reads certificate files directly from the host filesystem, and exposes certificate expiration metrics for Prometheus and Grafana, reads MicroK8s certificate files directly from the host, and exposes certificate expiration metrics for Prometheus and Grafana.

Lightweight Prometheus exporter for monitoring MicroK8s certificate expiration.

## Features

* Monitor MicroK8s certificates
* Expose certificate expiration metrics to Prometheus
* Alert before certificates expire
* Lightweight single binary written in Go
* DaemonSet deployment (one exporter per node)
* Multi-architecture support (amd64 / arm64)
* Hardened security configuration
* Configurable certificates via environment variables
* Runs only on MicroK8s control-plane nodes
* No Kubernetes API permissions required

## Monitored Certificates

By default:

* `server.crt`
* `front-proxy-client.crt`

Example:

```yaml
CERT_FILES=server.crt,front-proxy-client.crt
```

Additional certificates can be added if required.

## Metrics

### Days Remaining

```promql
microk8s_cert_days_remaining
```

Days remaining before certificate expiration.

Example:

```text
microk8s_cert_days_remaining{node="master1",cert="server"} 18
```

---

### Expiration Timestamp

```promql
microk8s_cert_not_after_timestamp
```

Certificate expiration time as Unix timestamp.

Example:

```text
microk8s_cert_not_after_timestamp{node="master1",cert="server"} 1785678900
```

---

### Certificate Expired

```promql
microk8s_cert_expired
```

Returns:

* `0` = valid
* `1` = expired

Example:

```text
microk8s_cert_expired{node="master1",cert="server"} 0
```

---

### Exporter Health

```promql
microk8s_cert_exporter_last_scrape_success
```

Returns:

* `1` = successful scan
* `0` = failed scan

---

### Configured Certificates

```promql
microk8s_cert_exporter_certs_total
```

Number of configured certificates.

---

### Failed Certificate Reads

```promql
microk8s_cert_exporter_certs_failed
```

Number of certificate files that could not be read.

## Architecture

```text
+-------------------+
| Prometheus        |
+---------+---------+
          |
          v
+-------------------+
| Service           |
+---------+---------+
          |
          v
+-------------------+
| DaemonSet         |
| Control Plane     |
| Nodes Only        |
+---------+---------+
          |
          v
+-------------------+
| MicroK8s Certs    |
| server.crt        |
| front-proxy-...   |
+-------------------+
```



## Security

The exporter follows a hardened deployment model:

* Runs as root only for certificate access
* `allowPrivilegeEscalation=false`
* `readOnlyRootFilesystem=true`
* All Linux capabilities dropped
* RuntimeDefault seccomp profile
* Read-only hostPath mount
* No Kubernetes API access required
* `automountServiceAccountToken=false`

Example:

```yaml
securityContext:
  runAsUser: 0
  runAsGroup: 0
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true

  capabilities:
    drop:
      - ALL

  seccompProfile:
    type: RuntimeDefault
```

## Installation

### Control Plane Scheduling

The exporter is intended to run only on MicroK8s control-plane nodes.

Example:

```yaml
nodeSelector:
  node-role.kubernetes.io/control-plane: ""
```

Deploy all Kubernetes resources:

```bash
kubectl apply -f kubernetes/
```

Or:

```bash
kubectl apply -f kubernetes/daemonset.yaml
kubectl apply -f kubernetes/service.yaml
kubectl apply -f kubernetes/servicemonitor.yaml
kubectl apply -f kubernetes/alertrule.yaml
```

Verify:

```bash
kubectl get daemonset -n monitoring
kubectl get pods -n monitoring
kubectl get servicemonitor -n monitoring
kubectl get prometheusrule -n monitoring
```

## Configuration

Environment variables:

| Variable        | Default                           | Description             |
| --------------- | --------------------------------- | ----------------------- |
| CERT_DIR        | /host-certs                       | Certificate directory   |
| CERT_FILES      | server.crt,front-proxy-client.crt | Certificates to monitor |
| SCRAPE_INTERVAL | 5m                                | Refresh interval        |
| LISTEN_ADDRESS  | :9101                             | Metrics listen address  |
| NODE_NAME       | Kubernetes node name              | Node label              |

Example:

```yaml
env:
  - name: CERT_FILES
    value: server.crt,front-proxy-client.crt

  - name: SCRAPE_INTERVAL
    value: 5m
```

## Alerts

Included Prometheus rules:

### Certificate Expired

Severity: Critical

Triggers when:

```promql
microk8s_cert_expired == 1
```

---

### Certificate Expiring Soon

Severity: Warning

Triggers when:

```promql
microk8s_cert_days_remaining < 30
```

---

### Certificate Expiring Critical

Severity: Critical

Triggers when:

```promql
microk8s_cert_days_remaining < 14
```

---

### Certificate Read Failure

Severity: Warning

Triggers when:

```promql
microk8s_cert_exporter_certs_failed > 0
```

## Building

### Docker

```bash
docker build -t microk8s-cert-exporter .
```

### Multi-Architecture

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t yourrepo/microk8s-cert-exporter:latest \
  --push .
```

## Grafana Examples

Certificates expiring within 30 days:

```promql
microk8s_cert_days_remaining < 30
```

Expired certificates:

```promql
microk8s_cert_expired == 1
```

Certificate expiration date:

```promql
microk8s_cert_not_after_timestamp
```

## Compatibility

Tested with:

* MicroK8s

Designed specifically for monitoring MicroK8s control-plane certificates.

Default certificates:

* server.crt
* front-proxy-client.crt

Additional certificates can be configured using the CERT_FILES environment variable.

## License

MIT License
