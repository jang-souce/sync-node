-- TaskMeta（任务表）: 存储文件同步任务元数据
CREATE TABLE IF NOT EXISTS task_metas (
    id VARCHAR(36) PRIMARY KEY, -- 唯一任务 ID (UUID)
    file_name VARCHAR(255) NOT NULL, -- 文件名
    file_size BIGINT NOT NULL, -- 文件大小
    file_hash VARCHAR(64) NOT NULL, -- 文件哈希 (MD5/SHA256)
    oss_url TEXT NOT NULL, -- OSS 下载链接
    source_url TEXT, -- 原始文件来源 URL (第三方拉取)
    tag VARCHAR(255), -- 任务标签
    task_type INT DEFAULT 1, -- 任务类型: 1-只下载, 2-下载后同步删除oss端资源
    status VARCHAR(20) DEFAULT 'pending', -- 任务状态: pending(等待中), processing(进行中), completed(已完成), failed(失败)
    deleted_at TIMESTAMPTZ, -- 软删除时间
    created_at BIGINT -- 创建时间 (Unix 时间戳)
);

-- NodeState（子节点表）: 跟踪子节点的运行时状态
CREATE TABLE IF NOT EXISTS node_states (
    node_id VARCHAR(36) PRIMARY KEY,
    hostname VARCHAR(255), -- 主机名
    ip VARCHAR(50), -- IP 地址
    status VARCHAR(20), -- 状态: online(在线), offline(离线)
    disk_usage DECIMAL(5,2), -- 磁盘使用率 (%)
    total_disk BIGINT, -- 总磁盘空间 (字节)
    available_disk BIGINT, -- 可用磁盘空间 (字节)
    last_heartbeat BIGINT -- 最后心跳时间 (Unix 时间戳)
);

CREATE INDEX idx_node_states_last_heartbeat ON node_states(last_heartbeat);

-- TaskState（任务状态）: 跟踪特定节点上的任务状态
CREATE TABLE IF NOT EXISTS task_states (
    id BIGSERIAL PRIMARY KEY,
    task_id VARCHAR(36) NOT NULL,
    node_id VARCHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL, -- 状态: pending(等待中), downloading(下载中), completed(已完成), failed(失败)
    error_msg TEXT, -- 错误信息
    retry_count INT DEFAULT 0, -- 重试次数
    synced_size BIGINT DEFAULT 0, -- 已同步大小 (字节，用于断点续传)
    updated_at BIGINT, -- 更新时间 (Unix 时间戳)
    CONSTRAINT fk_task_states_task FOREIGN KEY (task_id) REFERENCES task_metas(id) ON DELETE CASCADE
);

CREATE INDEX idx_task_states_task_id ON task_states(task_id);
CREATE INDEX idx_task_states_node_id ON task_states(node_id);

-- GlobalConfig（配置表）: 系统全局配置 (可选，主要存储在 Etcd)
CREATE TABLE IF NOT EXISTS global_configs (
    id BIGSERIAL PRIMARY KEY,
    heartbeat_interval INT, -- 心跳间隔 (秒)
    task_timeout INT, -- 任务超时时间 (秒)
    updated_at BIGINT -- 更新时间 (Unix 时间戳)
);
