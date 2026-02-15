# Sync-Node 部署指南

本文档详细介绍了 Sync-Node 系统的部署流程，支持 Docker Compose 一键部署和手动部署两种方式。

## 1. 环境准备 (Prerequisites)

在开始部署之前，请确保您的环境满足以下要求：

- **操作系统**: Linux / macOS / Windows (WSL2)
- **软件依赖**:
  - Docker Engine >= 20.10
  - Docker Compose >= 2.0
  - Go >= 1.20 (仅在进行源码编译或本地开发时需要)
- **硬件要求**:
  - CPU: 2 Core+
  - Memory: 4GB+ (运行所有组件建议配置)

## 2. 架构概览 (Architecture)

部署包含以下核心组件：

- **Central Service**: 中心管控服务，负责任务分发、节点监控和 API 接口。
- **Node Service**: 边缘节点服务，负责执行文件下载任务。
- **Etcd**: 分布式键值存储，用于节点注册、配置下发和任务队列。
- **PostgreSQL**: 关系型数据库，存储任务历史、节点元数据等持久化数据。
- **EtcdKeeper**: Etcd 可视化管理界面 (便于调试)。

## 3. 配置说明 (Configuration)

### 3.1 环境变量 (Environment Variables)

系统优先使用环境变量进行配置，这在 Docker 部署中尤为重要。主要变量如下：

| 服务 | 变量名 | 说明 | 默认值/示例 |
| :--- | :--- | :--- | :--- |
| **Global** | `ETCD_ENDPOINTS` | Etcd 连接地址 | `http://etcd:2379` |
| **Central** | `POSTGRES_HOST` | 数据库主机 | `postgres` |
| | `POSTGRES_USER` | 数据库用户 | `user` |
| | `POSTGRES_PASSWORD` | 数据库密码 | `password` |
| | `OSS_ACCESSKEYID` | OSS AccessKey | (需自行配置) |
| | `OSS_ACCESSKEYSECRET` | OSS Secret | (需自行配置) |
| **Node** | `NODE_ID` | 节点唯一标识 | `node-1` |
| | `NODE_WORKDIR` | 下载工作目录 | `/root/downloads` |

### 3.2 配置文件

如果不使用环境变量，也可以通过修改配置文件进行设置：

- **Central**: `central/config.yaml`
- **Node**: `node/config.yaml`

**注意**: Docker 部署模式下，`docker-compose.yml` 中的环境变量设置会覆盖配置文件中的同名配置。

#### 3.2.1 中心节点配置 (central/config.yaml)

```yaml
server:
  port: ":8088"                    # HTTP 服务端口

etcd:
  endpoints:
    - "localhost:2379"             # Etcd 集群地址
  username: ""                      # Etcd 用户名（可选）
  password: ""                      # Etcd 密码（可选）

postgres:
  host: "localhost"                 # 数据库主机
  port: 5432                        # 数据库端口
  user: "user"                      # 数据库用户
  password: "password"             # 数据库密码
  dbname: "sync_node_db"           # 数据库名
  sslmode: "disable"               # SSL 模式 (disable/require/verify-ca)

oss:
  provider: "aliyun"               # 存储提供商 (aliyun/minio)
  endpoint: "oss-cn-beijing.aliyuncs.com"  # OSS 端点
  accessKeyId: "your-access-key"    # OSS AccessKey ID
  accessKeySecret: "your-secret"   # OSS AccessKey Secret
  bucketName: "your-bucket"        # OSS Bucket 名称

sms:
  accessKeyId: "your-access-key"   # 阿里云 AccessKey
  accessKeySecret: "your-secret"   # 阿里云 Secret
  signName: "SyncNode"            # 短信签名
  templateCode: "SMS_123456789"   # 短信模板代码
  phoneNumbers:                    # 接收告警的手机号
    - "13800000000"

log:
  level: "info"                     # 日志级别 (debug/info/warn/error)
  path: "./logs/central.log"       # 日志文件路径
```

#### 3.2.2 子节点配置 (node/config.yaml)

```yaml
node:
  id: "node-001"                    # 节点唯一标识
  workDir: "./downloads"           # 下载工作目录
  hostname: ""                     # 节点主机名（自动获取）
  httpPort: ":8081"                # HTTP 服务端口
  heartbeatInterval: 15            # 心跳间隔（秒）

etcd:
  endpoints:
    - "localhost:2379"             # Etcd 集群地址
  username: ""                      # Etcd 用户名（可选）
  password: ""                      # Etcd 密码（可选）

log:
  level: "info"                     # 日志级别
  path: "./logs/node.log"          # 日志文件路径
```

#### 3.2.3 配置优先级

配置项的优先级从高到低为：
1. 环境变量（Docker 部署推荐）
2. 配置文件
3. 代码默认值

## 4. 快速部署 (Docker Compose)

推荐使用 Docker Compose 进行一键部署，脚本位于项目根目录。

### 4.1 启动服务

使用 `deploy.sh` 脚本可以快速启动所有服务：

```bash
# 赋予脚本执行权限
chmod +x deploy.sh

# 启动所有服务 (后台运行)
./deploy.sh start
```

或者直接使用 docker-compose 命令：

```bash
docker-compose up -d --build
```

### 4.2 验证部署

启动成功后，各服务访问地址如下：

- **Central Dashboard**: [http://localhost:8088/view/index.html](http://localhost:8088/view/index.html)
- **Central API**: [http://localhost:8088/api/v1/nodes](http://localhost:8088/api/v1/nodes)
- **EtcdKeeper (Etcd UI)**: [http://localhost:8090/etcdkeeper/](http://localhost:8090/etcdkeeper/)
- **PostgreSQL**: 端口 `5432`

### 4.3 查看日志

查看所有服务日志：

```bash
./deploy.sh logs
```

或者查看特定服务日志：

```bash
docker-compose logs -f central
docker-compose logs -f node-1
```

### 4.4 停止服务

```bash
./deploy.sh stop
```

## 5. 手动部署 (Manual Deployment)

适用于开发调试或非容器化环境。

### 5.1 启动依赖组件

首先需要启动 Etcd 和 PostgreSQL。可以使用 Docker 单独启动：

```bash
# 启动 Etcd
docker run -d --name etcd -p 2379:2379 -e ALLOW_NONE_AUTHENTICATION=yes quay.io/coreos/etcd:v3.5.9 etcd --advertise-client-urls http://0.0.0.0:2379 --listen-client-urls http://0.0.0.0:2379

# 启动 PostgreSQL
docker run -d --name postgres -p 5432:5432 -e POSTGRES_USER=user -e POSTGRES_PASSWORD=password -e POSTGRES_DB=sync_node_db postgres:15
```

### 5.2 运行 Central Service

1. 修改 `central/config.yaml` 确保连接地址指向本地 (localhost)。
2. 运行服务：

```bash
cd central
go run main.go
```

### 5.3 运行 Node Service

1. 修改 `node/config.yaml`，设置唯一的 `id` (如 `node-local-1`)。
2. 运行服务：

```bash
cd node
go run main.go
```

## 6. 常见问题 (Troubleshooting)

### Q1: 节点无法注册到 Central
- **检查 Etcd**: 确认 `ETCD_ENDPOINTS` 配置正确，且 Central 和 Node 都能连接到同一个 Etcd 集群。
- **检查网络**: 如果是 Docker 部署，确保所有容器在同一个网络 (`sync-net`) 下。

### Q2: 任务下载失败
- **检查 OSS 配置**: 确认 Central 的 OSS AccessKey/Secret 配置正确且有权限读取 Bucket。
- **检查网络**: 确认 Node 容器可以访问外网（下载 OSS 文件）。

### Q3: 数据库连接错误
- **检查 PostgreSQL**: 确认数据库容器已启动，且用户名密码与配置一致。
- **首次启动**: 系统会自动执行 `db/schema.sql` 初始化表结构（由 GORM 或手动迁移脚本处理，需确认代码实现）。

### Q4: 端口冲突
- 如果本地 8088 (Central) 或 2379 (Etcd) 端口被占用，请修改 `docker-compose.yml` 中的映射端口。
