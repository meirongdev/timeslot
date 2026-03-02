# Deployment Guide

## Docker

### Build

```bash
# From repo root
docker build -t timeslot:latest .
```

### Run

```bash
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/config.json:/config/config.json:ro \
  -v timeslot-data:/data \
  --name timeslot \
  timeslot:latest
```

Set `"db_path": "/data/timeslot.db"` and `"listen_addr": ":8080"` in `config.json`.

---

## Kubernetes

Manifests are in `deploy/k8s/`. SQLite requires `ReadWriteOnce` access — **keep `replicas: 1`**.

### Prerequisites

- A Kubernetes cluster with a `StorageClass` supporting `ReadWriteOnce`
- `kubectl` configured for the cluster
- A container registry to push your image

### 1. Build and push image

```bash
docker build -t ghcr.io/<your-org>/timeslot:latest .
docker push ghcr.io/<your-org>/timeslot:latest
```

Update `deploy/k8s/deployment.yaml` → `image:` with your registry path.

### 2. Configure secrets

Edit `deploy/k8s/secret.yaml` and fill in real values:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: timeslot-secrets
  namespace: timeslot
type: Opaque
stringData:
  admin_password: "your-strong-password"
```

> **Do not commit `secret.yaml` with real secrets.** Use [Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets), [External Secrets Operator](https://external-secrets.io/), or inject from your CI/CD vault.

### 3. Configure application

Edit `deploy/k8s/configmap.yaml`:
- Adjust `timezone`, `slot_duration_min`, etc.
- The `admin_password` is injected automatically via an `initContainer` at startup.

### 4. Deploy

```bash
kubectl apply -f deploy/k8s/namespace.yaml
kubectl apply -f deploy/k8s/secret.yaml
kubectl apply -f deploy/k8s/configmap.yaml
kubectl apply -f deploy/k8s/pvc.yaml
kubectl apply -f deploy/k8s/deployment.yaml
kubectl apply -f deploy/k8s/service.yaml
```

Or apply the whole directory:

```bash
kubectl apply -f deploy/k8s/
```

### 5. Verify

```bash
kubectl -n timeslot get pods
kubectl -n timeslot logs -f deployment/timeslot
```

The readiness probe checks `GET /api/slots`; the pod becomes Ready once the DB is initialized and the server is listening.

### 6. Ingress / TLS

`deploy/k8s/service.yaml` includes an Ingress resource (commented TLS annotations). Edit the `host:` and uncomment the `cert-manager.io/cluster-issuer` annotation if you use cert-manager.

### Upgrading

```bash
docker build -t ghcr.io/<your-org>/timeslot:v1.x .
docker push ghcr.io/<your-org>/timeslot:v1.x

kubectl -n timeslot set image deployment/timeslot timeslot=ghcr.io/<your-org>/timeslot:v1.x
```

SQLite migrations run automatically on startup — no manual migration step needed.

### Backup

Back up the SQLite file from the PVC:

```bash
kubectl -n timeslot exec deployment/timeslot -- \
  sqlite3 /data/timeslot.db ".backup '/data/timeslot-backup.db'"
```

Then copy the backup file off the pod using `kubectl cp`.

---

## ArgoCD Integration

### Helm Chart部署

本项目已提供 Helm Chart（deploy/helm/），推荐通过 ArgoCD 进行 GitOps 持续交付。

### 步骤

1. **确保 Helm Chart 已推送到 Git 仓库**
   - 目录：`deploy/helm/`

2. **在 ArgoCD 创建应用**
   - 方式一：Web UI
     - 新建 Application，选择 Helm 类型。
     - 填写仓库地址、Chart 路径（如 `deploy/helm`）、目标集群和命名空间。
     - 可自定义 values.yaml 参数（如镜像 tag、端口、持久化等）。
   - 方式二：YAML 声明
     ```yaml
     apiVersion: argoproj.io/v1alpha1
     kind: Application
     metadata:
       name: timeslot
       namespace: argocd
     spec:
       project: default
       source:
         repoURL: 'https://your.git.repo'
         targetRevision: main
         path: deploy/helm
         helm:
           valueFiles:
             - values.yaml
       destination:
         server: 'https://kubernetes.default.svc'
         namespace: timeslot
       syncPolicy:
         automated:
           prune: true
           selfHeal: true
     ```

3. **配置镜像仓库和密钥**
   - 在 `values.yaml` 或 ArgoCD UI 中设置 `image` 字段为你的镜像地址。
   - 管理员密码（`adminPassword`）在 `values.yaml` 中设置，建议通过 ArgoCD 的 `helm.parameters` 或 Secrets 管理工具注入。

4. **自动化部署与回滚**
   - 每次推送代码或更新 Helm `values.yaml`，ArgoCD 会自动同步并部署。
   - 可在 ArgoCD UI 一键回滚到历史版本。

### 参考命令

```bash
# 手动同步
argocd app sync timeslot

# 查看状态
argocd app get timeslot
```

---

## Architecture Notes for k8s

| Concern | Approach |
|---|---|
| Persistence | PVC (`ReadWriteOnce`) mounted at `/data` |
| Config | ConfigMap template + initContainer renders `admin_password` into `/config/config.json` |
| Secrets | k8s Secret (`admin_password`) injected via env vars in initContainer |
| Replicas | Must be 1 (SQLite single-writer) |
| Memory | ~32–128 MB typical |
| Health | Liveness: `GET /admin/`; Readiness: `GET /api/slots` |
