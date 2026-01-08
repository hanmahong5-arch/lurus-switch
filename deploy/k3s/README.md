# Lurus Switch K3S Deployment Guide

## Prerequisites

- Single-node server (Alibaba Cloud ECS recommended)
- Minimum specs: 4 vCPU, 8GB RAM, 100GB SSD
- Ubuntu 22.04 or Windows Server 2019+
- Domain names: `api.lurus.cn`, `ai.lurus.cn`
- DNS A records pointing to server IP

## Quick Start

### 1. Install K3S

```bash
# Linux
curl -sfL https://get.k3s.io | sh -

# Verify installation
kubectl get nodes
```

For Windows (experimental):
```powershell
# Use WSL2 or run K3S in a Linux VM
```

### 2. Install cert-manager

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml

# Wait for cert-manager to be ready
kubectl wait --for=condition=available --timeout=300s deployment/cert-manager -n cert-manager
```

### 3. Deploy Lurus Switch

```bash
# Clone the repository
git clone https://github.com/pocketzworld/lurus-switch.git
cd lurus-switch

# Update secrets (IMPORTANT: edit before applying)
vim deploy/k3s/secrets/lurus-secrets.yaml

# Apply all resources
kubectl apply -k deploy/k3s/

# Check deployment status
kubectl get pods -n lurus-system -w
```

### 4. Verify Deployment

```bash
# Check all pods are running
kubectl get pods -n lurus-system

# Check services
kubectl get svc -n lurus-system

# Check ingress
kubectl get ingress -n lurus-system

# Test health endpoint
curl https://ai.lurus.cn/health
```

## Directory Structure

```
deploy/k3s/
├── namespace.yaml              # lurus-system namespace
├── kustomization.yaml          # Kustomize configuration
├── ingress.yaml                # Traefik Ingress + cert-manager
├── configmaps/
│   └── gateway-config.yaml     # Gateway service configuration
├── secrets/
│   └── lurus-secrets.yaml      # Database credentials, API tokens
├── deployments/
│   ├── gateway-service.yaml    # AI API gateway (2 replicas)
│   └── new-api.yaml            # Management console (2 replicas)
├── statefulsets/
│   ├── postgres.yaml           # PostgreSQL database
│   ├── redis.yaml              # Redis cache
│   └── nats.yaml               # NATS message bus
└── hpa/
    └── gateway-hpa.yaml        # Horizontal Pod Autoscaler
```

## Services

| Service | Port | Replicas | Description |
|---------|------|----------|-------------|
| gateway-service | 18100 | 2-10 (HPA) | AI API proxy |
| new-api | 3000 | 2-5 (HPA) | Management console |
| postgres | 5432 | 1 | PostgreSQL database |
| redis | 6379 | 1 | Redis cache |
| nats | 4222 | 1 | NATS message bus |

## Configuration

### Update Secrets

Before deploying, update `secrets/lurus-secrets.yaml`:

```yaml
stringData:
  POSTGRES_PASSWORD: "your-strong-password"
  NEW_API_TOKEN: "your-new-api-token"
  JWT_SECRET: "your-jwt-secret"
```

### Update ConfigMaps

Modify `configmaps/gateway-config.yaml` for environment-specific settings.

### Image Tags

Update `kustomization.yaml` to use specific image tags:

```yaml
images:
  - name: ghcr.io/pocketzworld/gateway-service
    newTag: v1.0.0
  - name: ghcr.io/pocketzworld/new-api
    newTag: v1.0.0
```

## Operations

### Scale Deployments

```bash
# Manual scaling
kubectl scale deployment/gateway-service --replicas=5 -n lurus-system

# Check HPA status
kubectl get hpa -n lurus-system
```

### View Logs

```bash
# Gateway service logs
kubectl logs -f deployment/gateway-service -n lurus-system

# All pods logs
kubectl logs -f -l app=gateway-service -n lurus-system --all-containers
```

### Rolling Update

```bash
# Update image
kubectl set image deployment/gateway-service \
  gateway=ghcr.io/pocketzworld/gateway-service:v1.1.0 \
  -n lurus-system

# Watch rollout
kubectl rollout status deployment/gateway-service -n lurus-system
```

### Rollback

```bash
# View rollout history
kubectl rollout history deployment/gateway-service -n lurus-system

# Rollback to previous version
kubectl rollout undo deployment/gateway-service -n lurus-system

# Rollback to specific revision
kubectl rollout undo deployment/gateway-service --to-revision=2 -n lurus-system
```

### Database Backup

```bash
# PostgreSQL backup
kubectl exec -n lurus-system postgres-0 -- pg_dump -U lurus lurus > backup.sql

# Restore
kubectl exec -i -n lurus-system postgres-0 -- psql -U lurus lurus < backup.sql
```

## Troubleshooting

### Pods not starting

```bash
# Check pod events
kubectl describe pod <pod-name> -n lurus-system

# Check container logs
kubectl logs <pod-name> -n lurus-system --previous
```

### TLS certificate issues

```bash
# Check certificate status
kubectl get certificate -n lurus-system
kubectl describe certificate lurus-tls -n lurus-system

# Check cert-manager logs
kubectl logs -n cert-manager deployment/cert-manager
```

### Ingress not working

```bash
# Check Traefik logs
kubectl logs -n kube-system deployment/traefik

# Verify ingress configuration
kubectl describe ingress lurus-ingress -n lurus-system
```

### Storage issues

```bash
# Check PVC status
kubectl get pvc -n lurus-system

# Check local-path-provisioner
kubectl logs -n local-path-storage deployment/local-path-provisioner
```

## Monitoring

### View Metrics

```bash
# Port forward Prometheus
kubectl port-forward svc/prometheus 9090:9090 -n lurus-system

# Port forward Grafana
kubectl port-forward svc/grafana 3001:3000 -n lurus-system
```

### Health Checks

```bash
# Gateway health
curl http://localhost:18100/health

# NEW-API status
curl http://localhost:3000/api/status

# All services
kubectl get pods -n lurus-system -o wide
```

## CI/CD Integration

The deployment is automated via GitHub Actions:

1. **CI Pipeline** (`.github/workflows/ci.yml`)
   - Runs on push to main/develop
   - Runs tests, linting, security scans
   - Builds Docker images (without pushing)

2. **CD Pipeline** (`.github/workflows/deploy.yml`)
   - Triggered by version tags (v*)
   - Builds and pushes to ghcr.io
   - Deploys to K3S via SSH
   - Creates GitHub Release

### Required GitHub Secrets

| Secret | Description |
|--------|-------------|
| `K3S_HOST` | K3S server IP/hostname |
| `K3S_USER` | SSH username |
| `K3S_SSH_KEY` | SSH private key |

### Deployment Process

```bash
# Tag and push to trigger deployment
git tag v1.0.0
git push origin v1.0.0
```
