# 分布式文件同步系统 - 测试计划 (Test Plan)

本文档详细描述了分布式文件同步系统的测试策略、范围、方法以及详细的测试用例，旨在确保系统在功能、性能和稳定性方面均达到预期标准。

## 1. 测试策略 (Test Strategy)

采用金字塔测试模型，自底向上进行验证：

*   **单元测试 (Unit Testing)**: 占比 70%。针对各个核心模块（Utility, Service, Module）进行独立测试，使用 Mock 技术隔离外部依赖（Etcd, OSS, DB）。
*   **集成测试 (Integration Testing)**: 占比 20%。重点验证组件间的交互，如 API 接口、数据库操作、Etcd 事件监听。
*   **系统/端到端测试 (System/E2E Testing)**: 占比 10%。模拟真实用户场景，验证全链路流程（从任务创建到节点文件同步完成）。

## 2. 测试范围 (Test Scope)

### 2.1 单元测试 (Unit Tests)

#### 公共模块 (Common)
*   **Utils**: `common/utils`
    *   MD5 计算准确性 (包括空文件、大文件)。
    *   Retry 机制 (重试次数、退避策略)。
    *   File 操作 (原子写入、存在性检查)。
*   **Etcd Client**: `common/service/etcd`
    *   Mock Etcd 交互，验证 Put/Get/Watch 逻辑封装。

#### 中心节点 (Central Node)
*   **Task Service**: `central/service/task`
    *   任务创建逻辑 (参数校验、DB 插入)。
    *   任务分发逻辑 (Etcd Put)。
    *   任务状态更新 (DB Update)。
    *   **重点**: `RepublishPendingTasks` (启动恢复机制)。
*   **Node Service**: `central/service/node`
    *   节点状态解析。
    *   Watch 事件处理 (节点上线/下线)。
*   **OSS Service**: `central/service/oss`
    *   URL 解析与签名生成。
    *   Mock OSS SDK 调用。
*   **Config Service**: `central/service/config`
    *   配置加载与热更新。

#### 子节点 (Child Node)
*   **HTTP Service**: `node/service/http`
    *   **重点**: 断点续传 (Range Header)。
    *   下载进度回调。
*   **Task Module**: `node/module/task`
    *   任务接收处理。
    *   任务状态上报 (Etcd Put)。
*   **Heartbeat Module**: `node/module/heartbeat`
    *   租约续期逻辑。

### 2.2 集成测试 (Integration Tests)

*   **API 接口测试**: 使用 `httptest` 验证 Gin Handler 的请求/响应格式。
*   **数据库集成**: 验证 GORM 模型与 PostgreSQL 的实际交互 (CRUD)。
*   **Etcd 集成**: 验证真实的 Watch/Put 流程，确保事件能被正确触发。
*   **OSS 集成**: 验证实际的文件上传/下载链路 (需配置测试 Bucket)。

### 2.3 系统测试 (System Tests)

*   **全链路同步**: Central 创建任务 -> Etcd 分发 -> Node 监听到 -> Node 下载 -> 校验 -> 上报完成。
*   **异常恢复**: Node 宕机重启后继续下载；Central 重启后恢复未完成任务。

## 3. 详细测试用例 (Detailed Test Cases)

### 3.1 节点生命周期 (Node Lifecycle)

| ID | 模块 | 用例名称 | 前置条件 | 测试步骤 | 预期结果 | 优先级 |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **NL-01** | Node | 正常注册 | Etcd 正常 | 启动子节点 | Etcd 中生成 `/file_sync/node/{id}`，状态 `online` | P0 |
| **NL-02** | Node | 心跳保活 | 子节点运行中 | 持续运行 2 分钟 | Etcd 中 Key 租约不断续期，Key 不消失 | P0 |
| **NL-03** | Node | 正常下线 | 子节点运行中 | 发送 SIGTERM 信号停止子节点 | Etcd 中 Key 被删除或状态变为 `offline` | P1 |
| **NL-04** | Node | 异常掉线 | 子节点运行中 | `kill -9` 强制杀掉子节点 | 租约过期后 (约 45s)，Etcd 中 Key 自动消失 | P1 |
| **NL-05** | Central | 感知节点上线 | Central 运行中 | 启动一个新的子节点 | Central 日志打印 "Node detected: online"，DB 更新节点状态 | P0 |
| **NL-06** | Central | 感知节点离线 | 子节点运行中 | 强制停止子节点 | Central 监听到 Delete 事件，DB 更新节点状态为 `offline` | P0 |

### 3.2 任务管理 (Task Management)

| ID | 模块 | 用例名称 | 前置条件 | 测试步骤 | 预期结果 | 优先级 |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **TM-01** | Task | 创建任务-单文件 | 节点在线 | POST `/api/v1/tasks` (1个文件) | 1. DB 插入 MainTask/SubTask<br>2. Etcd 写入 `/file_sync/task/{node}/{id}` | P0 |
| **TM-02** | Task | 创建任务-多文件 | 节点在线 | POST `/api/v1/tasks` (5个文件) | 1. DB 插入 1 MainTask, 5 SubTasks<br>2. Etcd 写入 5 条记录 | P1 |
| **TM-03** | Task | 任务分发-指定节点 | Node-1 在线 | 创建任务指定 `targetNodeId=Node-1` | 只有 Node-1 收到任务，其他节点无反应 | P1 |
| **TM-04** | Task | 任务分发-随机负载 | Node-1, Node-2 在线 | 连续创建 10 个任务 (不指定节点) | 任务大致均匀分布在 Node-1 和 Node-2 | P2 |
| **TM-05** | Task | 任务去重 | 任务已存在 | 再次创建完全相同的任务 | 返回成功，但 DB/Etcd 不应产生重复数据 (取决于业务逻辑，目前是生成新 ID) | P2 |
| **TM-06** | Task | 任务恢复 (Republish) | 任务 Pending | 1. 停止 Central<br>2. 手动删除 Etcd 中的任务 Key<br>3. 启动 Central | Central 启动时扫描 DB Pending 任务，重新写入 Etcd | P1 |

### 3.3 文件同步与传输 (File Sync)

| ID | 模块 | 用例名称 | 前置条件 | 测试步骤 | 预期结果 | 优先级 |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **FS-01** | Sync | 正常下载 | 任务已下发 | 子节点监听到任务 | 1. 文件下载至 `downloads/` 目录<br>2. 文件 MD5 与预期一致<br>3. 任务状态变更为 `completed` | P0 |
| **FS-02** | Sync | 断点续传 | 任务下载中 | 1. 下载过程中停止 Node<br>2. 重启 Node | 1. Node 检测到临时文件<br>2. 发送 Range 请求继续下载<br>3. 文件最终完整 | P0 |
| **FS-03** | Sync | MD5 校验失败 | OSS 文件被篡改 | 下发任务 (预期 MD5 与实际不符) | 1. 下载完成后校验失败<br>2. 删除错误文件<br>3. 任务状态置为 `failed` | P1 |
| **FS-04** | Sync | 磁盘空间不足 | 磁盘已满 | 下发大文件任务 | 任务报错 `disk full`，状态置为 `failed` | P2 |
| **FS-05** | Sync | OSS 源文件删除 | 任务Type=2 | 任务同步完成 | Central 收到完成事件后，调用 OSS 接口删除源文件 | P1 |

### 3.4 异常处理 (Error Handling)

| ID | 模块 | 用例名称 | 前置条件 | 测试步骤 | 预期结果 | 优先级 |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **EH-01** | Common | Etcd 连接断开 | 系统运行中 | 断开网络或停止 Etcd | 1. 组件报错但进程不退<br>2. 自动重试连接<br>3. 恢复后自动重连 | P0 |
| **EH-02** | Common | DB 连接断开 | Central 运行中 | 停止 PostgreSQL | 1. API 返回 500<br>2. 恢复后 API 正常 | P1 |
| **EH-03** | Task | 下载超时 | 网络极慢 | 限制网速，下发大任务 | 超过 `GlobalConfig.TaskTimeout` 后，任务标记为超时/失败 | P2 |

## 4. 自动化测试计划 (Automation Plan)

### 4.1 工具链
*   **Test Framework**: Go native `testing`
*   **Assertion**: `github.com/stretchr/testify/assert`
*   **Mocking**: `github.com/stretchr/testify/mock` (用于 mock Etcd, OSS)
*   **CI/CD**: GitHub Actions (待配置)

### 4.2 执行命令
```bash
# 运行所有单元测试
go test ./... -v

# 运行特定模块测试 (如 Task Service)
go test ./central/service/task/... -v

# 运行集成测试 (需要本地启动 Etcd/PG/MinIO)
# 可以在测试代码中通过 build tag 区分
go test -tags=integration ./... -v
```

### 4.3 覆盖率目标
*   核心业务逻辑 (Task/Node Service): > 80%
*   工具类 (Utils): > 90%
*   整体项目: > 70%

## 5. 执行检查清单 (Execution Checklist)

### 5.1 文件传输场景 (File Transfer)
- [x] **小文件传输 (< 1MB)**:
    - [x] 上传一个小文件 (如 1KB 文本) 到 OSS。
    - [x] 下发任务到节点。
    - [x] 验证节点 `downloads` 目录文件存在且 MD5 正确。
- [x] **大文件传输 (> 100MB)**:
    - [x] 上传一个大文件 (如 200MB 视频/压缩包) 到 OSS。
    - [x] 下发任务。
    - [x] 验证下载过程日志是否有进度输出。
    - [x] 验证下载后文件完整性。
- [x] **多文件批量传输**:
    - [x] 创建一个包含 5 个不同大小文件 (混合小/中/大) 的任务。
    - [x] 验证所有 5 个文件均被下载。
    - [x] 验证数据库中 5 个子任务状态均为 `completed`。
- [x] **断点续传 (Resumable Download)**:
    - [x] 启动一个大文件下载任务。
    - [x] 在下载进度约 50% 时强制停止 Node 进程 (Ctrl+C)。
    - [x] 重启 Node。
    - [x] 观察日志，应显示 `Range: bytes=xxxxx-` 请求。
    - [x] 验证文件最终下载完成且 Hash 正确。

### 5.2 状态与监控场景 (Status & Monitoring)
- [x] **节点在线状态更新**:
    - [x] 启动 Node，Etcd 查看 `/file_sync/node/{id}` 值为 JSON 数据，状态 `online`。
    - [x] 停止 Node，等待约 45s (或租约过期)，验证 Central 日志报错或数据库状态变更为 `offline`。
- [x] **短信报警触发 (SMS Alert)**:
    - [x] **场景 A**: 节点异常离线 (非正常退出)。
        - [x] `kill -9` 杀掉 Node 进程 (模拟 Docker Stop)。
        - [x] 等待 Central 检测到 Watch Delete 事件。
        - [x] 验证 Central 日志中是否调用 `SendSMS` (日志显示 `failed to send sms`，证明已调用)。
    - [x] **场景 B**: 任务多次重试失败。
        - [x] 构造一个 OSS URL 损坏的任务 (手动注入 Etcd，OSS URL 签名无效)。
        - [x] Node 重试 3 次后失败 (日志验证：Operation failed (attempt 3/3) 及 Task ... failed to download)。
        - [x] 验证 Central/Etcd 收到失败状态 (Etcd 中 status 变为 failed)。

### 5.3 异常容错场景 (Fault Tolerance)
- [x] **Etcd 临时故障**:
    - [x] 停止 Etcd 容器 (`docker stop etcd`)。
    - [x] 观察 Central/Node 日志，应报错但进程不退 (Central 返回 500 error，context deadline exceeded)。
    - [x] 启动 Etcd 容器。
    - [x] 验证连接自动恢复，任务分发恢复正常。
- [x] **重复任务幂等性**:
    - [x] 发送两个完全相同的任务（TaskID 不同，但 FileHash/Url/Name 相同）。
    - [x] 验证节点不会重复下载（如果文件已存在且 Hash 一致），或者能正确覆盖且不报错。
        - 结果: 已优化逻辑，检测到本地文件存在且 Hash 匹配时，直接跳过下载并返回成功。

## 6. 测试场景组合矩阵 (Scenario Matrix)

以下场景通过 cURL 命令模拟 API 请求触发，覆盖了文件数量、文件大小及任务类型的不同组合。

| 优先级 | 组合编号 | 文件数量 | 文件大小 | 任务类型 | 测试重点与预期 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **P0** | **SC-01** | **单文件** | **小文件** (KB级) | **只下载** (Type 1) | **[冒烟测试]** 验证链路通畅。文件应秒传，OSS 文件**保留**。 |
| **P0** | **SC-02** | **单文件** | **小文件** (KB级) | **删除源** (Type 2) | **[业务逻辑]** 验证核心业务。下载完成后，Central 应**自动删除** OSS 上的源文件。 |
| **P1** | **SC-03** | **单文件** | **大文件** (>100MB) | **只下载** (Type 1) | **[稳定性]** 验证进度条、断点续传。观察日志中 `syncedSize` 是否持续增长。 |
| **P1** | **SC-04** | **单文件** | **大文件** (>100MB) | **删除源** (Type 2) | **[完整流程]** 验证大文件传输耗时较长情况下，任务完成后的清理逻辑是否依然可靠。 |
| **P2** | **SC-05** | **多文件** | **小文件** (批量) | **只下载** (Type 1) | **[并发/聚合]** 验证主任务进度聚合。如 5 个文件，主任务进度应从 0% -> 20% -> ... -> 100%。 |
| **P2** | **SC-06** | **多文件** | **小文件** (批量) | **删除源** (Type 2) | **[批量清理]** 验证所有子任务完成后，OSS 上对应的多个源文件是否都被删除。 |
| **P3** | **SC-07** | **多文件** | **大文件** (批量) | **只下载** (Type 1) | **[压力测试]** 验证带宽和磁盘压力。多个大文件同时下载时节点是否稳定，是否有超时。 |
| **P3** | **SC-08** | **多文件** | **混合大小** | **混合类型** | **[真实场景]** 模拟生产环境：有的文件大、有的文件小；有的需保留、有的需删除。 |

## 7. 测试请求命令 (cURL Commands)

### SC-01: 单文件 / 小文件 / 仅下载
**目标**: 验证最基础的下载流程。
**文件**: OSS 测试小文件 (small.txt)

```bash
curl -X POST http://localhost:8088/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "file_data": [
        {
            "fileName": "small.txt",
            "sourceUrl": "https://yang-sync.oss-cn-beijing.aliyuncs.com/test-data%2Fsmall.txt?Expires=1801891908&OSSAccessKeyId=your-access-key-id&Signature=5AuQXPv8JjennQ30%2BT14spK78ro%3D",
            "taskType": 1,
            "targetNodeId": "node-1"
        }
    ]
}'
```

### SC-02: 单文件 / 小文件 / 下载后删除 OSS
**目标**: 验证任务完成后的清理逻辑。
**文件**: OSS 测试删除文件 1 (delete_me_1.txt)
> **注意**: 此测试执行后，OSS 上的 `test-data/delete_me_1.txt` 将被删除。如需再次测试需重新上传。

```bash
curl -X POST http://localhost:8088/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "file_data": [
        {
            "fileName": "delete_me_1.txt",
            "sourceUrl": "https://yang-sync.oss-cn-beijing.aliyuncs.com/test-data%2Fdelete_me_1.txt?Expires=1801891913&OSSAccessKeyId=your-access-key-id&Signature=NEA8SYsXYMpi8OGI1cpXmBiyzo8%3D",
            "taskType": 2,
            "targetNodeId": "node-1"
        }
    ]
}'
```

### SC-03: 单文件 / 大文件 / 仅下载 (断点续传)
**目标**: 验证大文件下载及断点续传能力。
**文件**: OSS 测试大文件 (large.bin, 20MB)

```bash
curl -X POST http://localhost:8088/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "file_data": [
        {
            "fileName": "large.bin",
            "sourceUrl": "https://yang-sync.oss-cn-beijing.aliyuncs.com/test-data%2Flarge.bin?Expires=1801891913&OSSAccessKeyId=your-access-key-id&Signature=4sB5m8i90ICNbYus2t4UoLuy8V4%3D",
            "taskType": 1,
            "targetNodeId": "node-1"
        }
    ]
}'
```

### SC-05: 多文件 / 小文件 / 仅下载
**目标**: 验证并发处理和主任务状态聚合。

```bash
curl -X POST http://localhost:8088/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "file_data": [
        {
            "fileName": "small_1.txt",
            "sourceUrl": "https://yang-sync.oss-cn-beijing.aliyuncs.com/test-data%2Fsmall.txt?Expires=1801891908&OSSAccessKeyId=your-access-key-id&Signature=5AuQXPv8JjennQ30%2BT14spK78ro%3D",
            "taskType": 1,
            "targetNodeId": ""
        },
        {
            "fileName": "small_2.txt",
            "sourceUrl": "https://yang-sync.oss-cn-beijing.aliyuncs.com/test-data%2Fsmall.txt?Expires=1801891908&OSSAccessKeyId=your-access-key-id&Signature=5AuQXPv8JjennQ30%2BT14spK78ro%3D",
            "taskType": 1,
            "targetNodeId": ""
        }
    ]
}'
```

### SC-08: 混合场景 (多文件 / 混合大小 / 混合类型)
**目标**: 模拟真实生产环境负载。
**文件**: small.txt (Type 1) + delete_me_2.txt (Type 2)

```bash
curl -X POST http://localhost:8088/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "file_data": [
        {
            "fileName": "small.txt",
            "sourceUrl": "https://yang-sync.oss-cn-beijing.aliyuncs.com/test-data%2Fsmall.txt?Expires=1801892665&OSSAccessKeyId=your-access-key-id&Signature=XI3cN%2FQtcsHUTYaB9RgTMxYDqpc%3D",
            "taskType": 1,
            "targetNodeId": ""
        }
    ]
}'
```

### SC-09: 复杂混合场景 (High Load / Mixed Types / Random Distribution)
**目标**: 压力测试，覆盖单下载、下载后删除、指定节点、随机分发，文件数量 > 6。

```bash
curl -X POST http://localhost:8088/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "file_data": [
        {
            "fileName": "small_spec_1.txt",
            "sourceUrl": "https://yang-sync.oss-cn-beijing.aliyuncs.com/test-data%2Fsmall.txt?Expires=1801892665&OSSAccessKeyId=your-access-key-id&Signature=XI3cN%2FQtcsHUTYaB9RgTMxYDqpc%3D",
            "taskType": 1,
            "targetNodeId": "node-1"
        },
        {
            "fileName": "small_random_1.txt",
            "sourceUrl": "https://yang-sync.oss-cn-beijing.aliyuncs.com/test-data%2Fsmall.txt?Expires=1801892665&OSSAccessKeyId=your-access-key-id&Signature=XI3cN%2FQtcsHUTYaB9RgTMxYDqpc%3D",
            "taskType": 1,
            "targetNodeId": ""
        },
        {
            "fileName": "large_spec_1.bin",
            "sourceUrl": "https://yang-sync.oss-cn-beijing.aliyuncs.com/test-data%2Flarge.bin?Expires=1801892673&OSSAccessKeyId=your-access-key-id&Signature=X7Yo5OxEVOO0g%2BkA0fz4ir0WLB4%3D",
            "taskType": 1,
            "targetNodeId": "node-2"
        },
        {
            "fileName": "large_random_1.bin",
            "sourceUrl": "https://yang-sync.oss-cn-beijing.aliyuncs.com/test-data%2Flarge.bin?Expires=1801892673&OSSAccessKeyId=your-access-key-id&Signature=X7Yo5OxEVOO0g%2BkA0fz4ir0WLB4%3D",
            "taskType": 1,
            "targetNodeId": ""
        },
        {
            "fileName": "delete_spec_1.txt",
            "sourceUrl": "https://yang-sync.oss-cn-beijing.aliyuncs.com/test-data%2Fdelete_me_3.txt?Expires=1801892673&OSSAccessKeyId=your-access-key-id&Signature=udYtNUZk6dFLFYASMV7WboLzLHo%3D",
            "taskType": 2,
            "targetNodeId": "node-3"
        },
        {
            "fileName": "delete_random_1.txt",
            "sourceUrl": "https://yang-sync.oss-cn-beijing.aliyuncs.com/test-data%2Fdelete_me_4.txt?Expires=1801892673&OSSAccessKeyId=your-access-key-id&Signature=VDpaj7XbPPEDLrkTsxAtGrjVdN8%3D",
            "taskType": 2,
            "targetNodeId": ""
        },
        {
            "fileName": "small_batch_extra.txt",
            "sourceUrl": "https://yang-sync.oss-cn-beijing.aliyuncs.com/test-data%2Fsmall.txt?Expires=1801892665&OSSAccessKeyId=your-access-key-id&Signature=XI3cN%2FQtcsHUTYaB9RgTMxYDqpc%3D",
            "taskType": 1,
            "targetNodeId": ""
        }
    ]
}'
```
