# WhoAmI

[![Build](https://github.com/ysicing/whoami/actions/workflows/ci.yaml/badge.svg)](https://github.com/ysicing/whoami/actions/workflows/ci.yaml)
[![License](https://img.shields.io/github/license/ysicing/whoami)](LICENSE)

一个用于 Kubernetes 环境测试的 HTTP 服务，用于测试 Service、ConfigMap、环境变量、资源限制、健康检查、升级回滚等特性。

## 功能特性

- ✅ Pod 信息展示（hostname、IP、namespace）
- ✅ 环境变量展示（仅显示以 GAEA 开头的环境变量）
- ✅ ConfigMap 配置读取
- ✅ 资源限制信息展示
- ✅ 健康检查端点（/healthz、/readyz）
- ✅ 版本信息接口（/version）
- ✅ 独立端点（/envs、/cm）
- ✅ 优雅关闭支持

## API 端点

### GET /

返回完整的服务信息，包括版本、Pod 信息、环境变量、ConfigMap 内容和资源限制。

**响应示例：**
```json
{
  "version": {
    "version": "v1.0.0",
    "git_commit": "abc123",
    "build_time": "2024-01-01_12:00:00",
    "go_version": "1.23"
  },
  "pod": {
    "hostname": "whoami-7d8f4c5b6-x9z8y",
    "pod_ip": "10.244.0.5",
    "host_ip": "192.168.1.10",
    "namespace": "default"
  },
  "environment": {
    "GAEA_ENV": "production",
    "GAEA_REGION": "us-west-1"
  },
  "configmaps": {
    "files": {
      "config.yml": "..."
    },
    "count": 1
  },
  "resources": {
    "cpu_request": "100m",
    "cpu_limit": "200m",
    "mem_request": "128Mi",
    "mem_limit": "256Mi"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### GET /version

返回版本信息。

**响应示例：**
```json
{
  "version": "v1.0.0",
  "git_commit": "abc123",
  "build_time": "2024-01-01_12:00:00",
  "go_version": "1.23"
}
```

### GET /envs

仅返回环境变量（只显示以 GAEA 开头的环境变量）。

**响应示例：**
```json
{
  "GAEA_ENV": "production",
  "GAEA_REGION": "us-west-1",
  "GAEA_CLUSTER": "k8s-prod-01"
}
```

### GET /cm

仅返回 ConfigMap 配置信息。

**响应示例：**
```json
{
  "files": {
    "app.conf": "environment=production\ndebug=false",
    "database.conf": "host=localhost\nport=5432"
  },
  "count": 2
}
```

### GET /healthz

健康检查端点，返回 `{"status": "ok"}`。

### GET /readyz

就绪检查端点，返回 `{"status": "ready"}`。

## 构建和运行

### 本地构建

```bash
# 查看所有可用命令
make help

# 构建二进制文件
make build

# 运行应用
make run

# 运行测试
make test
```

### Docker 构建

镜像使用多阶段构建：
- **构建阶段**：基于 `ysicing/god` 镜像
- **运行阶段**：基于 `ysicing/debian` 镜像
- **用户权限**：容器以 root 用户运行

```bash
# 构建 Docker 镜像
make docker-build

# 本地运行 Docker 容器
make docker-run

# 推送镜像到仓库
DOCKER_REGISTRY=your-registry DOCKER_IMAGE=your-image make docker-push
```

### 手动构建

```bash
# 构建
docker build \
  --build-arg VERSION=v1.0.0 \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  --build-arg BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S') \
  -t whoami:latest .

# 运行
docker run -p 8080:8080 whoami:latest
```

## CI/CD

项目使用 GitHub Actions 自动构建 Docker 镜像。

### 触发构建

只在推送 tag 时触发构建：

```bash
# 创建并推送标签
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

GitHub Actions 会自动：
- 构建多平台 Docker 镜像（linux/amd64、linux/arm64）
- 推送到 GitHub Container Registry
- 镜像标签与 Git 标签一致（如：`v1.0.0`）

### 使用预构建镜像

```bash
# 拉取指定版本
docker pull ghcr.io/ysicing/whoami:v1.0.0

# 运行
docker run -p 8080:8080 ghcr.io/ysicing/whoami:v1.0.0
```

## Kubernetes 部署

### 环境变量配置

在 Pod 规范中，建议配置以下环境变量以获取完整信息：

```yaml
env:
  - name: POD_IP
    valueFrom:
      fieldRef:
        fieldPath: status.podIP
  - name: HOST_IP
    valueFrom:
      fieldRef:
        fieldPath: status.hostIP
  - name: POD_NAMESPACE
    valueFrom:
      fieldRef:
        fieldPath: metadata.namespace
  - name: CPU_REQUEST
    valueFrom:
      resourceFieldRef:
        containerName: whoami
        resource: requests.cpu
  - name: CPU_LIMIT
    valueFrom:
      resourceFieldRef:
        containerName: whoami
        resource: limits.cpu
  - name: MEM_REQUEST
    valueFrom:
      resourceFieldRef:
        containerName: whoami
        resource: requests.memory
  - name: MEM_LIMIT
    valueFrom:
      resourceFieldRef:
        containerName: whoami
        resource: limits.memory
```

### ConfigMap 挂载

将 ConfigMap 挂载到 `/etc/config` 目录：

```yaml
volumes:
  - name: config
    configMap:
      name: whoami-config
volumeMounts:
  - name: config
    mountPath: /etc/config
```

### 健康检查配置

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

## 配置

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| PORT | 8080 | HTTP 服务端口 |
| POD_IP | - | Pod IP 地址 |
| HOST_IP | - | 宿主机 IP 地址 |
| POD_NAMESPACE | - | Pod 所在命名空间 |
| CPU_REQUEST | - | CPU 请求值 |
| CPU_LIMIT | - | CPU 限制值 |
| MEM_REQUEST | - | 内存请求值 |
| MEM_LIMIT | - | 内存限制值 |

## 测试场景

### 1. Service 测试
```bash
kubectl run test --rm -it --image=curlimages/curl -- curl http://whoami:8080/
```

### 2. ConfigMap 测试
创建 ConfigMap 并挂载到 Pod，访问 `/` 端点查看配置内容。

### 3. 滚动更新测试
```bash
# 更新镜像版本
kubectl set image deployment/whoami whoami=whoami:v2

# 观察滚动更新过程
kubectl rollout status deployment/whoami

# 查看版本信息
curl http://whoami:8080/version
```

### 4. 回滚测试
```bash
# 回滚到上一个版本
kubectl rollout undo deployment/whoami

# 查看回滚状态
kubectl rollout status deployment/whoami
```

## License

MIT
