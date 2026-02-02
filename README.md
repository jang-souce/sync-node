# sync-node

sync-node 是一个分布式文件同步系统，采用 Central/Node 架构设计，支持海量文件的分发、同步与一致性管理。

## 📁 目录结构

```
sync-node/
├── central/            # 中心节点 (Central Node)
│   ├── config/         # 配置加载与热更新
│   ├── handler/        # HTTP API 处理器
│   ├── module/         # 业务逻辑模块 (任务、节点、文件管理)
│   ├── service/        # 基础设施服务 (Etcd, OSS, SMS)
│   └── main.go         # 入口文件
├── node/               # 子节点 (Worker Node) - 施工中
│   ├── module/         # 核心业务 (心跳上报、任务执行)
│   └── main.go         # 入口文件
├── common/             # 公共组件库
│   ├── constant/       # 全局常量 (错误码, Etcd前缀, 状态枚举)
│   ├── model/          # 数据库模型 (GORM)
│   ├── service/        # 公共客户端 (Etcd Client, PG Client)
│   └── utils/          # 工具函数 (Logger, JSON, Hash)
├── db/                 # 数据库脚本
│   └── schema.sql      # 初始化 SQL
├── docs/               # 项目文档
│   └── openapi.yaml    # API 接口定义
└── docker-compose.yml  # 本地开发基础设施编排
```

## 🛠 技术栈

- **语言**: Go 1.24
- **Web框架**: Gin
- **数据库**: PostgreSQL 15 (GORM)
- **配置中心/服务发现**: Etcd v3.5
- **对象存储**: MinIO (本地模拟) / Aliyun OSS (生产)
- **配置管理**: Viper (支持环境变量与热更新)
- **日志**: Logrus