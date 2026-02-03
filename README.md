# sync-node

sync-node 是一个分布式文件同步系统，采用 Central/Node 架构设计，支持海量文件的分发、同步与一致性管理。

## 📁 目录结构

```
sync-node/
├── central/            # 中心节点 (Central Node)
│   ├── cmd/            # 命令行工具
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
│   ├── service/        # 基础设施服务 (HTTP下载等)
│   ├── util/           # 子节点专用工具 (文件操作等)
│   └── main.go         # 子节点入口
├── common/             # 公共组件库
│   ├── constant/       # 全局常量 (错误码, Etcd前缀)
│   ├── model/          # 数据模型 (GORM/JSON)
│   ├── service/        # 公共服务客户端 (Etcd, PG)
│   ├── sms/            # 短信服务
│   └── utils/          # 通用工具 (Logger, JSON, Hash)
├── db/                 # 数据库脚本
│   └── schema.sql      # 初始化 SQL
├── docs/               # 项目文档
│   ├── openapi.yaml    # API 接口定义
└── docker-compose.yml  # 本地开发基础设施编排
```

## 🛠 技术栈

- **语言**: Go 1.24
- **Web框架**: Gin
- **数据库**: PostgreSQL 15 (GORM)
- **配置中心/服务发现**: Etcd v3.5
- **对象存储**: MinIO (本地模拟) / Aliyun OSS (生产)
- **配置管理**: Viper (支持环境变量与热更新)
- **依赖注入**: Uber Fx
- **日志**: Logrus