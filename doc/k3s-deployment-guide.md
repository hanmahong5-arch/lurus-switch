# Lurus Switch 多节点 K3s 集群部署指南

> **文档版本**: v1.0
> **更新日期**: 2026-01-11
> **适用范围**: 3节点 K3s 高可用集群部署

---

## 目录 / Table of Contents

1. [概述](#一概述)
2. [集群架构](#二集群架构)
3. [服务器准备](#三服务器准备)
4. [K3s 集群安装](#四k3s-集群安装)
5. [基础组件安装](#五基础组件安装)
6. [部署 Lurus Switch](#六部署-lurus-switch)
7. [ArgoCD GitOps 配置](#七argocd-gitops-配置)
8. [Claude Code 集群管理](#八claude-code-集群管理)
9. [添加新应用流程](#九添加新应用流程)
10. [监控与告警](#十监控与告警)
11. [备份与恢复](#十一备份与恢复)
12. [故障排除](#十二故障排除)
13. [扩展路径](#十三扩展路径)

---

## 一、概述

### 1.1 部署目标

- **集群规模**: 3 节点 K3s 高可用集群 (etcd 内置)
- **GitOps**: ArgoCD 自动化部署，Git push 即部署
- **管理方式**: Claude Code + kubectl + ArgoCD UI
- **高可用**: 控制平面 3 副本，应用服务多副本

### 1.2 技术选型

| 组件 | 技术 | 说明 |
|------|------|------|
| 容器编排 | K3s | 轻量级 Kubernetes，适合边缘和小规模集群 |
| 存储 | Longhorn | 分布式块存储，跨节点数据复制 |
| GitOps | ArgoCD | 声明式持续部署 |
| 证书 | cert-manager | 自动 Let's Encrypt 证书 |
| Ingress | Traefik | K3s 内置，支持 HTTP/2 和 SSE |
| 监控 | Prometheus + Grafana | 指标采集和可视化 |
| 追踪 | Jaeger | 分布式链路追踪 |

---

## 二、集群架构

### 2.1 节点拓扑

```
                        ┌─────────────────────────────────────┐
                        │         Load Balancer / DNS         │
                        │   k3s.lurus.cn → Round Robin        │
                        └─────────────────┬───────────────────┘
                                          │
          ┌───────────────────────────────┼───────────────────────────────┐
          │                               │                               │
          ▼                               ▼                               ▼
┌─────────────────────┐     ┌─────────────────────┐     ┌─────────────────────┐
│   Node 1 (Master)   │     │   Node 2 (Master)   │     │   Node 3 (Master)   │
│   Ubuntu 22.04      │     │   Ubuntu 22.04      │     │   Ubuntu 22.04      │
├─────────────────────┤     ├─────────────────────┤     ├─────────────────────┤
│ Label:              │     │ Label:              │     │ Label:              │
│ lurus.cn/role=data  │     │ lurus.cn/role=svc   │     │ lurus.cn/role=obs   │
├─────────────────────┤     ├─────────────────────┤     ├─────────────────────┤
│ K3s Server          │     │ K3s Server          │     │ K3s Server          │
│ etcd (embedded)     │     │ etcd (embedded)     │     │ etcd (embedded)     │
├─────────────────────┤     ├─────────────────────┤     ├─────────────────────┤
│ PostgreSQL          │     │ Gateway Service x2  │     │ Prometheus          │
│ Redis               │     │ NEW-API x2          │     │ Grafana             │
│ NATS                │     │ Provider Service    │     │ Jaeger              │
│ ClickHouse          │     │ Billing Service     │     │ Alertmanager        │
│                     │     │ Log Service         │     │ ArgoCD              │
│                     │     │ Sync Service        │     │ lurus-portal        │
├─────────────────────┤     ├─────────────────────┤     ├─────────────────────┤
│ 最低: 4C 8G 100G    │     │ 最低: 4C 8G 80G     │     │ 最低: 4C 8G 80G     │
│ 推荐: 8C 16G 200G   │     │ 推荐: 8C 16G 100G   │     │ 推荐: 4C 8G 100G    │
└─────────────────────┘     └─────────────────────┘     └─────────────────────┘
```

### 2.2 网络规划

| 端口 | 协议 | 用途 |
|------|------|------|
| 6443 | TCP | K3s API Server |
| 2379-2380 | TCP | etcd 客户端和对等通信 |
| 10250 | TCP | Kubelet |
| 8472 | UDP | Flannel VXLAN |
| 80/443 | TCP | Ingress HTTP/HTTPS |
| 6222 | TCP | NATS 集群通信 |

### 2.3 服务端口映射

| 服务 | 内部端口 | 外部访问 |
|------|----------|----------|
| Gateway Service | 18100 | ai.lurus.cn/v1/* |
| NEW-API | 3000 | api.lurus.cn |
| Provider Service | 18101 | 内部 |
| Billing Service | 18103 | 内部 |
| Log Service | 18102 | 内部 |
| Sync Service | 8081 | 内部 |
| PostgreSQL | 5432 | 内部 |
| Redis | 6379 | 内部 |
| NATS | 4222/8222 | 内部 |
| ClickHouse | 8123/9000 | 内部 |
| Prometheus | 9090 | 内部 |
| Grafana | 3000 | grafana.lurus.cn |
| ArgoCD | 443 | argocd.lurus.cn |

---

## 三、服务器准备

### 3.1 硬件要求

| 角色 | CPU | 内存 | 磁盘 | 网络 |
|------|-----|------|------|------|
| Node 1 (Data) | 4+ vCPU | 8+ GB | 100+ GB SSD | 100 Mbps |
| Node 2 (Services) | 4+ vCPU | 8+ GB | 80+ GB SSD | 100 Mbps |
| Node 3 (Observability) | 4+ vCPU | 8+ GB | 80+ GB SSD | 100 Mbps |

### 3.2 操作系统准备

```bash
#!/bin/bash
# prepare-node.sh - Run on each server

# 1. Update system
sudo apt update && sudo apt upgrade -y

# 2. Install required tools
sudo apt install -y \
    curl \
    wget \
    git \
    htop \
    iotop \
    net-tools \
    jq \
    open-iscsi  # Required for Longhorn

# 3. Disable swap (K3s requirement)
sudo swapoff -a
sudo sed -i '/swap/d' /etc/fstab

# 4. Load required kernel modules
cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF

sudo modprobe overlay
sudo modprobe br_netfilter

# 5. Configure kernel parameters
cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF

sudo sysctl --system

# 6. Configure firewall (if using ufw)
sudo ufw allow 6443/tcp   # K3s API
sudo ufw allow 2379:2380/tcp  # etcd
sudo ufw allow 10250/tcp  # Kubelet
sudo ufw allow 8472/udp   # Flannel VXLAN
sudo ufw allow 80/tcp     # HTTP
sudo ufw allow 443/tcp    # HTTPS

echo "Node preparation completed!"
```

### 3.3 时间同步

```bash
# Install and configure NTP
sudo apt install -y chrony
sudo systemctl enable chrony
sudo systemctl start chrony

# Verify time sync
chronyc tracking
```

---

## 四、K3s 集群安装

### 4.1 安装 Node 1 (初始化集群)

```bash
#!/bin/bash
# install-k3s-node1.sh

export K3S_NODE_NAME="k3s-node1"
export PUBLIC_IP="<node1-public-ip>"

# Install K3s Server (initialize etcd cluster)
curl -sfL https://get.k3s.io | sh -s - server \
    --cluster-init \
    --node-name=${K3S_NODE_NAME} \
    --tls-san=${PUBLIC_IP} \
    --tls-san=k3s.lurus.cn \
    --write-kubeconfig-mode=644 \
    --disable=servicelb

# Wait for K3s to start
sleep 30

# Verify installation
sudo k3s kubectl get nodes

# Get join token
echo "Join Token:"
sudo cat /var/lib/rancher/k3s/server/node-token

# Set node label
sudo k3s kubectl label node ${K3S_NODE_NAME} lurus.cn/role=data
```

### 4.2 安装 Node 2 & 3 (加入集群)

```bash
#!/bin/bash
# install-k3s-node2.sh (Node 3 similar, change NODE_NAME and ROLE)

export K3S_TOKEN="<token-from-node1>"
export K3S_URL="https://<node1-ip>:6443"
export K3S_NODE_NAME="k3s-node2"  # Node 3: k3s-node3
export NODE_ROLE="services"        # Node 3: observability
export PUBLIC_IP="<node2-public-ip>"

# Install K3s Server (join existing cluster)
curl -sfL https://get.k3s.io | sh -s - server \
    --server=${K3S_URL} \
    --token=${K3S_TOKEN} \
    --node-name=${K3S_NODE_NAME} \
    --tls-san=${PUBLIC_IP} \
    --tls-san=k3s.lurus.cn \
    --write-kubeconfig-mode=644 \
    --disable=servicelb

# Wait for join
sleep 30

# Verify
sudo k3s kubectl get nodes

# Set node label
sudo k3s kubectl label node ${K3S_NODE_NAME} lurus.cn/role=${NODE_ROLE}
```

### 4.3 验证集群状态

```bash
# Run on any node
kubectl get nodes -o wide

# Expected output:
# NAME        STATUS   ROLES                       AGE   VERSION
# k3s-node1   Ready    control-plane,etcd,master   10m   v1.29.0+k3s1
# k3s-node2   Ready    control-plane,etcd,master   5m    v1.29.0+k3s1
# k3s-node3   Ready    control-plane,etcd,master   3m    v1.29.0+k3s1

# Verify etcd health
kubectl get endpoints -n kube-system

# Verify node labels
kubectl get nodes --show-labels
```

### 4.4 配置 kubectl 远程访问

```bash
# On your local development machine

# 1. Get kubeconfig from K3s master
scp root@<node1-ip>:/etc/rancher/k3s/k3s.yaml ~/.kube/config

# 2. Update server address to public IP or domain
sed -i 's/127.0.0.1/<node1-public-ip>/g' ~/.kube/config

# 3. Verify connection
kubectl cluster-info
kubectl get nodes
```

---

## 五、基础组件安装

### 5.1 安装 cert-manager

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml

# Wait for ready
kubectl wait --for=condition=available deployment/cert-manager -n cert-manager --timeout=300s

# Create ClusterIssuer (Let's Encrypt)
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@lurus.cn
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
      - http01:
          ingress:
            class: traefik
EOF
```

### 5.2 安装 Longhorn 分布式存储

```bash
# Install Longhorn
kubectl apply -f https://raw.githubusercontent.com/longhorn/longhorn/v1.6.0/deploy/longhorn.yaml

# Wait for ready
kubectl -n longhorn-system wait --for=condition=ready pod -l app=longhorn-manager --timeout=600s

# Set as default StorageClass
kubectl patch storageclass longhorn -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
kubectl patch storageclass local-path -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"false"}}}'
```

---

## 六、部署 Lurus Switch

### 6.1 一键部署

```bash
# Apply all configurations using Kustomize
kubectl apply -k deploy/k3s/

# Monitor deployment progress
kubectl get pods -n lurus-system -w

# Verify all services
kubectl get all -n lurus-system
```

---

## 七、ArgoCD GitOps 配置

### 7.1 安装 ArgoCD

```bash
# Create namespace
kubectl create namespace argocd

# Install ArgoCD
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for ready
kubectl wait --for=condition=available deployment/argocd-server -n argocd --timeout=300s

# Get initial password
echo "ArgoCD Password:"
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
echo
```

### 7.2 配置应用自动同步

```bash
kubectl apply -f deploy/argocd/lurus-application.yaml
```

**ArgoCD 工作流**:
1. Edit `deploy/k3s/*.yaml` files
2. `git commit && git push`
3. ArgoCD automatically syncs changes
4. View status at argocd.lurus.cn

---

## 八、Claude Code 集群管理

### 8.1 配置 kubectl

```bash
# Windows PowerShell
$env:KUBECONFIG = "$HOME\.kube\config"

# Verify connection
kubectl cluster-info
kubectl get nodes
```

### 8.2 常用管理命令

```bash
# Cluster status
kubectl get nodes -o wide
kubectl get pods -n lurus-system -o wide
kubectl top nodes

# View logs
kubectl logs -f deployment/gateway-service -n lurus-system --tail=100

# Rolling restart
kubectl rollout restart deployment/gateway-service -n lurus-system

# Scale
kubectl scale deployment/gateway-service --replicas=5 -n lurus-system

# Apply configuration
kubectl apply -k deploy/k3s/

# Database backup
kubectl exec postgres-0 -n lurus-system -- pg_dump -U lurus lurus > backup.sql
```

---

## 九、添加新应用流程

1. Create Dockerfile
2. Build and push image to ghcr.io
3. Create Deployment YAML in `deploy/k3s/deployments/`
4. Update `deploy/k3s/kustomization.yaml`
5. `git commit && git push`
6. ArgoCD auto-syncs

---

## 十、扩展路径

### 添加 Worker 节点

```bash
# On new server
curl -sfL https://get.k3s.io | K3S_URL=https://<master-ip>:6443 \
  K3S_TOKEN=<token> sh -s - agent \
  --node-label="lurus.cn/role=services"

# Verify
kubectl get nodes
```

---

## 附录

### A. 域名配置

| 域名 | 用途 |
|------|------|
| k3s.lurus.cn | K3s API |
| ai.lurus.cn | AI API Gateway |
| api.lurus.cn | NEW-API Management |
| argocd.lurus.cn | ArgoCD UI |
| grafana.lurus.cn | Grafana Monitoring |

### B. 参考文档

- [K3s Documentation](https://docs.k3s.io/)
- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [Longhorn Documentation](https://longhorn.io/docs/)

---

> **Document Maintenance**: Auto-generated by Claude Code
> **Last Updated**: 2026-01-11
