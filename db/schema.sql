-- MainTask（主任务表）: 记录一次批量任务提交的整体信息
CREATE TABLE IF NOT EXISTS main_tasks (
    id VARCHAR(36) PRIMARY KEY, -- 主任务 ID (UUID)
    total_count INT NOT NULL, -- 子任务总数
    created_at BIGINT, -- 创建时间 (Unix 时间戳)
    deleted_at TIMESTAMPTZ -- 软删除时间
);

-- SubTask（子任务表）: 具体的文件分发任务，一对一绑定节点
CREATE TABLE IF NOT EXISTS sub_tasks (
    id VARCHAR(36) PRIMARY KEY, -- 子任务 ID (UUID)
    main_task_id VARCHAR(36) NOT NULL, -- 关联主任务 ID
    node_id VARCHAR(36) NOT NULL, -- 目标节点 ID
    
    -- 文件信息
    file_name VARCHAR(255) NOT NULL, -- 文件名
    file_size BIGINT NOT NULL, -- 文件大小
    file_hash VARCHAR(64) NOT NULL, -- 文件哈希
    oss_url TEXT NOT NULL, -- OSS 下载链接
    source_url TEXT, -- 原始文件来源 URL
    tag VARCHAR(255), -- 标签
    task_type INT DEFAULT 1, -- 任务类型: 1-只下载, 2-下载后同步删除OSS资源

    -- 任务状态
    status VARCHAR(20) DEFAULT 'PENDING', -- 状态: PENDING, DOWNLOADING, COMPLETED, FAILED
    error_msg TEXT, -- 错误信息
    retry_count INT DEFAULT 0, -- 重试次数
    synced_size BIGINT DEFAULT 0, -- 已同步字节数
    updated_at BIGINT, -- 最后更新时间 (Unix 时间戳)
    created_at BIGINT, -- 创建时间 (Unix 时间戳)
    deleted_at TIMESTAMPTZ, -- 软删除时间

    CONSTRAINT fk_sub_tasks_main_task FOREIGN KEY (main_task_id) REFERENCES main_tasks(id) ON DELETE CASCADE
);

CREATE INDEX idx_sub_tasks_main_task_id ON sub_tasks(main_task_id);
CREATE INDEX idx_sub_tasks_node_id ON sub_tasks(node_id);

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

-- GlobalConfig（配置表）: 系统全局配置
CREATE TABLE IF NOT EXISTS global_configs (
    id BIGSERIAL PRIMARY KEY,
    heartbeat_interval INT, -- 心跳间隔 (秒)
    task_timeout INT, -- 任务超时时间 (秒)
    updated_at BIGINT -- 更新时间 (Unix 时间戳)
);
