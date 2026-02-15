# sync-node

sync-node 是一个分布式文件同步系统，采用 Central/Node 架构设计，支持海量文件的分发、同步与一致性管理。

## 🌟 核心功能

- **文件分发**: 支持批量文件的中心化分发，自动负载均衡到各子节点
- **断点续传**: 子节点支持断点续传下载，网络中断后可自动恢复
- **节点管理**: 实时监控节点状态，自动检测节点上下线
- **任务调度**: 基于分布式锁的任务分发，支持指定节点和随机负载策略
- **一致性保障**: 文件 MD5 校验确保数据完整性
- **告警通知**: 支持短信告警，节点异常或任务失败时及时通知
- **配置热更新**: 支持通过 Etcd 动态更新配置，无需重启服务
- **高可用**: 基于 Etcd 实现服务发现和分布式协调，支持节点故障自动恢复

## 🏗️ 架构设计

系统采用 Central/Node 架构：

- **Central Node (中心节点)**: 负责任务管理、节点监控、文件存储和 API 服务
- **Worker Nodes (子节点)**: 负责接收任务、下载文件、状态上报
- **Etcd**: 用于服务发现、分布式锁、配置同步和任务队列
- **PostgreSQL**: 存储任务历史、节点元数据等持久化数据
- **Aliyun OSS**: 中心节点文件存储，子节点从 OSS 下载文件

## 🚀 快速开始

### 前置要求

- Docker Engine >= 20.10
- Docker Compose >= 2.0
- Go >= 1.21 (仅本地开发需要)

### 一键部署

```bash
# 克隆项目
git clone <repository-url>
cd sync-node

# 启动所有服务
./deploy.sh start
```

启动成功后，访问以下地址：

- **Central Dashboard**: http://localhost:8088/view/index.html
- **Central API**: http://localhost:8088/api/v1/nodes
- **EtcdKeeper**: http://localhost:8090/etcdkeeper/

### 创建同步任务

```bash
curl -X POST http://localhost:8088/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "file_data": [
      {
        "fileName": "example.txt",
        "sourceUrl": "https://example.com/example.txt",
        "md5": "d41d8cd98f00b204e9800998ecf8427e"
      }
    ]
  }'
```

### 查看节点状态

```bash
curl http://localhost:8088/api/v1/nodes
```

### 停止服务

```bash
./deploy.sh stop
```

## 📁 目录结构

```
sync-node/
├── central/            # 中心节点 (Central Node)
│   ├── config/         # 配置加载与热更新
│   ├── handler/        # HTTP API 处理器 (路由控制)
│   ├── module/         # 业务模块组装 (Fx Modules)
│   ├── router/         # 路由注册
│   ├── service/        # 核心业务逻辑实现
│   │   ├── alert/      # 告警服务
│   │   ├── config/     # 配置服务
│   │   ├── node/       # 节点监控与管理
│   │   ├── oss/        # 对象存储服务
│   │   └── task/       # 任务管理服务
│   ├── util/           # 内部工具函数
│   └── main.go         # 中心节点入口
├── node/               # 子节点 (Worker Node)
│   ├── config/         # 子节点配置
│   ├── module/         # 业务模块 (心跳、任务)
│   ├── service/        # 基础设施服务 (HTTP下载、Etcd交互)
│   ├── util/           # 子节点专用工具 (文件操作等)
│   └── main.go         # 子节点入口
├── common/             # 公共组件库
│   ├── constant/       # 全局常量 (错误码, Etcd前缀)
│   ├── lock/           # 分布式锁
│   ├── middleware/     # 中间件 (认证、限流)
│   ├── model/          # 数据模型 (GORM/JSON)
│   ├── service/        # 公共服务客户端 (Etcd, PG)
│   ├── sms/            # 短信服务
│   └── utils/          # 通用工具 (Logger, JSON, Hash, Retry, Redact)
├── db/                 # 数据库脚本
│   └── schema.sql      # 初始化 SQL
├── docs/               # 项目文档
│   ├── deploy.md       # 部署指南
│   ├── development.md  # 开发文档
│   ├── error_codes.md  # 错误码说明
│   ├── file-data.md    # 文件数据结构
│   ├── openapi.yaml    # API 接口定义
│   └── test-plan.md    # 测试计划
├── docker-compose.yml  # 本地开发基础设施编排
├── deploy.sh           # 部署脚本
└── README.md           # 项目说明
```

## 🛠 技术栈

- **语言**: Go 1.21+
- **Web框架**: Gin
- **数据库**: PostgreSQL 15 (GORM)
- **配置中心/服务发现**: Etcd v3.5
- **对象存储**: MinIO (本地模拟) / Aliyun OSS (生产)
- **配置管理**: Viper (支持环境变量与热更新)
- **依赖注入**: Uber Fx
- **日志**: Logrus

## 📚 API 文档

完整的 API 文档请查看 [docs/openapi.yaml](docs/openapi.yaml)，主要接口包括：

### 任务管理
- `GET /api/v1/tasks` - 获取任务列表
- `POST /api/v1/tasks` - 创建同步任务
- `GET /api/v1/tasks/{id}` - 获取任务详情
- `DELETE /api/v1/tasks/{id}` - 删除任务

### 节点管理
- `GET /api/v1/nodes` - 获取节点列表
- `GET /api/v1/nodes/{id}` - 获取节点详情
- `POST /api/v1/nodes/{id}/reconnect` - 重连节点

### 配置管理
- `GET /api/v1/config` - 获取全局配置
- `PUT /api/v1/config` - 更新全局配置

### 文件管理
- `POST /api/v1/files/upload` - 上传文件
- `GET /api/v1/files/{id}/download` - 下载文件

### API 使用示例

#### 创建批量同步任务

```bash
curl -X POST http://localhost:8088/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "file_data": [
      {
        "fileName": "package-v1.zip",
        "sourceUrl": "https://example.com/package-v1.zip",
        "md5": "a1b2c3d4e5f6...",
        "targetNodeId": "node-1"
      },
      {
        "fileName": "config.json",
        "sourceUrl": "https://example.com/config.json",
        "md5": "f6e5d4c3b2a1..."
      }
    ]
  }'
```

#### 查询任务状态

```bash
curl http://localhost:8088/api/v1/tasks?page=1&pageSize=10
```

#### 查看在线节点

```bash
curl http://localhost:8088/api/v1/nodes?status=online
```

## 🔧 本地开发

### 环境准备

1. 安装 Go 1.21+
2. 安装 PostgreSQL 15
3. 安装 Etcd 3.5+

### 启动依赖服务

```bash
# 使用 Docker Compose 启动依赖
docker-compose up -d etcd postgres

# 或手动启动
docker run -d --name etcd -p 2379:2379 quay.io/coreos/etcd:v3.5.9
docker run -d --name postgres -p 5432:5432 -e POSTGRES_USER=user -e POSTGRES_PASSWORD=password postgres:15
```

### 运行中心节点

```bash
cd central
go mod download
go run main.go
```

### 运行子节点

```bash
cd node
go mod download
# 修改 config.yaml 设置唯一的 NODE_ID
go run main.go
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./central/service/task/...

# 运行测试并显示覆盖率
go test -cover ./...
```

## ⚙️ 配置说明

### 中心节点配置 (central/config.yaml)

```yaml
server:
  port: ":8088"

postgres:
  host: "localhost"
  port: 5432
  user: "user"
  password: "password"
  dbname: "sync_node_db"
  sslmode: "disable"

etcd:
  endpoints:
    - "http://localhost:2379"
  username: ""
  password: ""

oss:
  provider: "aliyun"
  endpoint: "https://oss-cn-beijing.aliyuncs.com"
  accessKeyId: "YOUR_ACCESS_KEY"
  accessKeySecret: "YOUR_SECRET"
  bucketName: "your-bucket"

log:
  level: "info"
  path: "./logs"
```

### 子节点配置 (node/config.yaml)

```yaml
node:
  id: "node-1"
  workDir: "./downloads"
  httpPort: ":8081"

etcd:
  endpoints:
    - "http://localhost:2379"
  username: ""
  password: ""

log:
  level: "info"
  path: "./logs"
```

## 📊 监控与运维

### 查看服务日志

```bash
# 查看所有服务日志
./deploy.sh logs

# 查看特定服务日志
docker-compose logs -f central
docker-compose logs -f node-1
```

### 监控节点状态

- 通过 Central API 获取节点状态
- 通过 EtcdKeeper 查看 Etcd 中的节点信息
- 检查日志文件了解节点运行情况

### 告警配置

在中心节点配置中设置短信告警：

```yaml
sms:
  provider: "aliyun"
  accessKeyId: "YOUR_ACCESS_KEY"
  accessKeySecret: "YOUR_SECRET"
  signName: "your-sign-name"
  templateCode: "your-template-code"
  phones:
    - "13800000000"
    - "13900000000"
```

## 📖 文档

- [部署指南](docs/deploy.md) - 详细的部署步骤和配置说明
- [开发文档](docs/development.md) - 开发规范和模块说明
- [测试计划](docs/test-plan.md) - 测试策略和测试用例
- [API 文档](docs/openapi.yaml) - OpenAPI 3.0 规范的 API 定义
- [错误码说明](docs/error_codes.md) - 错误码列表和处理建议

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License