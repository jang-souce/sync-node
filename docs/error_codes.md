# 错误码说明 (Error Codes)

本文档列出了系统 API 接口返回的错误码及其含义。

## HTTP 状态码映射

| 错误码 (Code) | 说明 (Description) | HTTP 状态码 |
| :--- | :--- | :--- |
| 200 | 成功 (Success) | 200 |
| 400 | 参数错误 (Invalid Parameter) | 400 |
| 401 | 未授权 (Unauthorized) | 401 |
| 404 | 资源不存在 (Not Found) | 404 |
| 500 | 内部错误 (Internal Error) | 500 |
| 503 | 服务器繁忙 (Server Busy) | 503 |

## 业务错误码 (Business Error Codes)

| 错误码 (Code) | 标识符 (Identifier) | 说明 (Description) | 建议处理 (Action) |
| :--- | :--- | :--- | :--- |
| 1001 | CodeTaskNotFound | 任务不存在 | 检查任务 ID 是否正确 |
| 1002 | CodeNodeNotFound | 节点不存在 | 检查节点 ID 是否正确或节点是否在线 |
| 1003 | CodeEtcdError | Etcd 操作失败 | 检查 Etcd 连接或稍后重试 |
| 1004 | CodeDBError | 数据库操作失败 | 检查数据库连接或稍后重试 |
| 1005 | CodeOSSError | OSS 操作失败 | 检查 OSS 配置或网络连接 |
| 1006 | CodeTaskDuplicated | 任务重复 | 请勿重复提交相同的任务 |

## 异常处理与重试策略

系统内部对瞬时错误（如数据库锁竞争、网络波动）实现了自动重试机制。
对于 API 调用方，建议对 `503` 和 `1xxx` 系列错误（除 `1001`, `1002`, `1006` 外）实施指数退避重试策略。
