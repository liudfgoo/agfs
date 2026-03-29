# AGFS 代码库详细分析报告

## 目录

1. [项目概述](#项目概述)
2. [系统架构](#系统架构)
3. [核心组件](#核心组件)
4. [插件系统](#插件系统)
5. [API 参考](#api-参考)
6. [客户端 SDK](#客户端-sdk)
7. [开发环境配置](#开发环境配置)
8. [数据流](#数据流)
9. [测试](#测试)
10. [部署](#部署)

---

## 项目概述

**AGFS (Aggregated File System)** 是一个基于插件的 RESTful 文件系统服务器，将各种后端服务（消息队列、数据库、对象存储、KV 存储等）统一为文件系统操作。受 Plan 9 启发，它提供了一个通用文件接口，AI 代理和应用程序可以通过简单的文件操作（如 `cat`、`echo`、`ls`）与其交互。

### 项目愿景

创建一个���一的接口，**万物皆文件**——从数据库到消息队列再到云存储，使后端服务通过直观的文件操作即可访问。

### 目标用户

- **AI 代理**：LLM 无需学习复杂 API 即可使用 AGFS
- **开发者**：简化的后端服务集成
- **系统管理员**：统一的管理界面
- **数据工程师**：简化的 ETL 和数据管道操作

### 核心特性

- **插件架构**：模块化文件系统后端
- **RESTful API**：基于 HTTP 的文件操作
- **多语言 SDK**：Go、Python 客户端
- **Shell 集成**：支持完整脚本编写的 Unix 风格 shell
- **FUSE 支持**：Linux 上的原生文件系统挂载
- **MCP 集成**：面向 AI 助手的模型上下文协议
- **动态挂载**：运行时插件管理

### 技术栈

- **服务器**：Go 1.21+ 配合插件系统
- **Shell**：Python 3.10+ 配合 Rich 终端
- **存储**：SQLite、TiDB、S3、内存
- **协议**：HTTP/REST、FUSE、MCP
- **部署**：Docker、systemd

---

## 系统架构

### 高层结构

```
┌─────────────────────────────────────────────────────────┐
│                      客户端层                            │
├───────────┬───────────┬───────────┬───────────┬─────────┤
│agfs-shell │ Python SDK│  Go SDK   │ MCP Server│ FUSE    │
└───────────┴───────────┴───────────┴───────────┴─────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                    AGFS 服务器                           │
├───────────┬───────────┬───────────┬─────────────────────┤
│ HTTP 处理器│ 挂载管理器 │ 插件加载器 │ 文件系统接口        │
└───────────┴───────────┴───────────┴─────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                      插件层                              │
├───────────┬───────────┬───────────┬───────────┬─────────┤
│  MemFS   │  QueueFS │   S3FS    │   SQLFS   │VectorFS │
│  LocalFS │   KVFS   │ StreamFS  │HeartbeatFS │  ...    │
└───────────┴───────────┴───────────┴───────────┴─────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                      后端层                              │
├───────────┬───────────┬───────────┬───────────┬─────────┤
│  内存     │SQLite/TiDB│  AWS S3   │ 本地磁盘  │HTTP端点 │
└───────────┴───────────┴───────────┴───────────┴─────────┘
```

### 设计模式

1. **插件模式**：动态文件系统加载
2. **适配器模式**：为不同后端提供统一接口
3. **策略模式**：每个插件采用不同的存储策略
4. **外观模式**：简单的文件操作隐藏复杂性
5. **观察者模式**：QueueFS 和 StreamFS 的事件处理

---

## 核心组件

### 目录结构

```
agfs/
├── agfs-server/          # Go 主服务器
│   ├── cmd/server/main.go         # 服务器入口点
│   ├── pkg/
│   │   ├── config/                # 配置管理
│   │   ├── filesystem/            # 核心文件系统接口
│   │   ├── handlers/              # HTTP 处理器
│   │   ├── mountablefs/           # 挂载管理
│   │   ├── plugin/                # 插件加载系统
│   │   └── plugins/               # 内置插件 (20+)
│   ├── config.example.yaml        # 配置模板
│   └── Dockerfile                 # 容器部署
│
├── agfs-shell/           # Python shell 客户端
│   ├── agfs_shell/
│   │   ├── commands/              # 57 个命令模块
│   │   ├── streams.py             # 流 I/O 处理
│   │   ├── process.py             # 进程执行
│   │   ├── pipeline.py            # 管道编排
│   │   ├── parser.py              # 命令解析
│   │   ├── filesystem.py          # AGFS 抽象
│   │   └── shell.py               # REPL 实现
│   └── tests/                     # 测试套件 (92 个测试)
│
├── agfs-fuse/            # FUSE 文件系统 (Linux)
│   ├── cmd/agfs-fuse/main.go      # FUSE 入口点
│   └── pkg/
│       ├── fusefs/                # FUSE 实现
│       └── cache/                 # 缓存层
│
├── agfs-sdk/             # 客户端 SDK
│   ├── go/
│   │   ├── client.go              # Go 客户端
│   │   └── types.go               # 类型定义
│   └── python/
│       └── pyagfs/
│           ├── client.py          # Python 客户端
│           ├── helpers.py         # 工具函数
│           └── exceptions.py      # 错误处理
│
└── agfs-mcp/             # MCP 服务器 (AI)
    └── src/agfs_mcp/
        └── server.py              # MCP 实现
```

### 服务器架构 (Go)

**核心接口**：`pkg/filesystem/filesystem.go`

```go
// 每个插件必须实现此接口
type Filesystem interface {
    // 文件操作
    Read(path string, offset, size int64) ([]byte, error)
    Write(path string, data []byte, offset int64) (int, error)
    Create(path string) error
    Remove(path string, recursive bool) error

    // 目录操作
    Mkdir(path string, mode uint32) error
    Readdir(path string) ([]FileInfo, error)

    // 元数据
    Stat(path string) (*FileInfo, error)
    Rename(oldPath, newPath string) error

    // 能力声明
    Capabilities() *Capabilities
}
```

### Shell 架构 (Python)

**核心组件**：

1. **Lexer** (`lexer.py`)：词法分析
2. **Parser** (`parser.py`)：构建 AST
3. **Executor** (`executor.py`)：执行 AST
4. **Process** (`process.py`)：命令执行上下文
5. **Pipeline** (`pipeline.py`)：连接进程流

**命令注册**：`commands/__init__.py`

```python
# 57 个内置命令，按类别组织
commands = {
    # 文件操作
    'cat': cat.CatCommand(),
    'ls': ls.LsCommand(),
    'mkdir': mkdir.MkdirCommand(),

    # 文本处理
    'grep': grep.GrepCommand(),
    'sort': sort.SortCommand(),
    'jq': jq.JqCommand(),

    # 系统控制
    'export': export.ExportCommand(),
    'cd': cd.CdCommand(),
    'pwd': pwd.PwdCommand(),

    # ... 50+ 更多命令
}
```

---

## 插件系统

### 内置插件 (20+)

#### 存储插件

| 插件 | 路径 | 描述 | 后端 |
|------|------|------|------|
| **MemFS** | `/memfs` | 内存文件系统 | RAM |
| **LocalFS** | `/local` | 本地目录挂载 | 磁盘 |
| **S3FS** | `/s3fs` | AWS S3 集成 | S3 API |
| **SQLFS** | `/sqlfs` | 数据库文件存储 | SQLite/TiDB/MySQL |

#### 应用插件

| 插件 | 路径 | 描述 | 用例 |
|------|------|------|------|
| **QueueFS** | `/queuefs` | 消息队列系统 | 任务分发 |
| **KVFS** | `/kvfs` | 键值存储 | 简单数据存储 |
| **StreamFS** | `/streamfs` | 流式数据 | 实时数据流 |
| **StreamRotateFS** | `/streamrotate` | 轮转流 | 日志轮转 |
| **HeartbeatFS** | `/heartbeatfs` | 进程监控 | 健康检查 |

#### 网络和工具插件

| 插件 | 路径 | 描述 | 用例 |
|------|------|------|------|
| **ProxyFS** | `/proxyfs` | AGFS 联邦 | 远程挂载 |
| **HTTPFS** | `/httagfs` | HTTP 文件服务器 | Web 访问 |
| **ServerInfoFS** | `/serverinfo` | 服务器元数据 | 监控 |
| **VectorFS** | `/vectorfs` | 语义搜索 | AI 搜索 |
| **GPTFS** | `/gptfs` | LLM 集成 | AI 文本生成 |

### 插件配置

插件在 `config.yaml` 中配置：

```yaml
plugins:
  # 单实例配置
  memfs:
    enabled: true
    path: /memfs
    config:
      init_dirs: ["/tmp", "/home"]

  # 多实例配置
  sqlfs:
    - name: local
      enabled: true
      path: /sqlfs
      config:
        backend: sqlite
        db_path: sqlfs.db

    - name: production
      enabled: true
      path: /sqlfs_prod
      config:
        backend: tidb
        dsn: "user:pass@tcp(host:4000)/db"
```

### 插件能力

每个插件声明其能力：

```go
type Capabilities struct {
    SupportsRandomWrite   bool  // 支持 pwrite
    SupportsTruncate      bool  // 支持 truncate()
    SupportsSync          bool  // 支持 fsync()
    SupportsTouch         bool  // 支持 touch()
    SupportsFileHandle    bool  // 支持有状态文件句柄
    IsAppendOnly          bool  // 仅追加文件
    IsReadDestructive     bool  // 读取有副作用
    IsObjectStore         bool  // S3 类语义
    IsBroadcast           bool  // 广播读取
    SupportsStreamRead    bool  // 流式读取
}
```

### 外部插件

AGFS 支持加载外部插件：

- **原生库** (.so, .dylib, .dll)
- **WebAssembly 模块** (.wasm)
- **远程插件** (通过 HTTP URL)

---

## API 参考

### 基础 URL

所有端点前缀为 `/api/v1`：

```
http://localhost:8080/api/v1/{endpoint}
```

### 文件操作

#### 读取文件

```bash
GET /api/v1/files?path=/memfs/data.txt&offset=0&size=1024
```

**响应**：二进制内容 (`application/octet-stream`)

#### 写入文件

```bash
PUT /api/v1/files?path=/memfs/data.txt&flags=append
Content-Type: application/octet-stream

Hello World
```

**标志**：`append`（追加）、`create`（创建）、`exclusive`（排他）、`truncate`（截断）、`sync`（同步）

**响应**：
```json
{
  "message": "write successful",
  "written": 11
}
```

#### 删除文件

```bash
DELETE /api/v1/files?path=/memfs/data.txt&recursive=false
```

#### 获取文件信息

```bash
GET /api/v1/stat?path=/memfs/data.txt
```

**响应**：
```json
{
  "name": "data.txt",
  "size": 1024,
  "mode": 420,
  "modTime": "2024-01-01T12:00:00Z",
  "isDir": false
}
```

### 目录操作

#### 列出目录

```bash
GET /api/v1/directories?path=/memfs
```

**响应**：
```json
{
  "files": [
    {"name": "file1.txt", "size": 100, "isDir": false},
    {"name": "dir1", "size": 0, "isDir": true}
  ]
}
```

#### 创建目录

```bash
POST /api/v1/directories?path=/memfs/newdir&mode=755
```

### 插件管理

#### 列出挂载

```bash
GET /api/v1/mounts
```

**响应**：
```json
{
  "mounts": [
    {
      "path": "/memfs",
      "pluginName": "memfs",
      "config": {}
    }
  ]
}
```

#### 挂载插件

```bash
POST /api/v1/mount
Content-Type: application/json

{
  "fstype": "memfs",
  "path": "/my_memfs",
  "config": {
    "init_dirs": ["/tmp"]
  }
}
```

### 高级操作

#### 搜索 (Grep)

```bash
POST /api/v1/grep
Content-Type: application/json

{
  "path": "/memfs/logs",
  "pattern": "error|warning",
  "recursive": true,
  "case_insensitive": true
}
```

**响应**：
```json
{
  "matches": [
    {
      "file": "/memfs/logs/app.log",
      "line": 42,
      "content": "ERROR: Connection failed"
    }
  ],
  "count": 1
}
```

#### 文件句柄（有状态操作）

```bash
# 打开文件
POST /api/v1/handles/open?path=/memfs/file.txt&flags=readwrite&lease=60

# 通过句柄读取
GET /api/v1/handles/{handle_id}/read?offset=0&size=1024

# 通过句柄写入
PUT /api/v1/handles/{handle_id}/write?offset=0
Content-Type: application/octet-stream

data

# 关闭句柄
DELETE /api/v1/handles/{handle_id}
```

---

## 客户端 SDK

### Go SDK

**安装**：
```bash
go get github.com/c4pt0r/agfs/agfs-sdk/go
```

**使用示例**：
```go
package main

import (
    "github.com/c4pt0r/agfs/agfs-sdk/go"
)

func main() {
    client := agfs.NewClient("http://localhost:8080")

    // 写入文件
    client.Write("/memfs/hello.txt", []byte("Hello, AGFS!"))

    // 读取文件
    data, _ := client.Read("/memfs/hello.txt", 0, -1)

    // 列出目录
    files, _ := client.ReadDir("/memfs")

    // 搜索
    results, _ := client.Grep("/memfs/logs", "error", true, true)
}
```

### Python SDK

**安装**：
```bash
pip install pyagfs
```

**使用示例**：
```python
from pyagfs import AGFSClient, cp, upload, download

client = AGFSClient("http://localhost:8080")

# 基本操作
client.write("/memfs/data.txt", b"Hello, AGFS!")
content = client.cat("/memfs/data.txt")

# 高级辅助函数
upload(client, "./local_dir", "/remote_dir", recursive=True)
download(client, "/remote_file.txt", "./local_file.txt")
cp(client, "/remote/src.txt", "/remote/dest.txt")

# 流式处理
for chunk in client.cat("/large/file.log", stream=True):
    process(chunk)

# 搜索
results = client.grep("/logs", "error", recursive=True)
```

### Shell 客户端

**启动 Shell**：
```bash
agfs-shell
```

**交互式使用**：
```bash
agfs:/> ls /memfs
agfs:/> echo "Hello" > /memfs/test.txt
agfs:/> cat /memfs/test.txt
agfs:/> mkdir /queuefs/tasks
agfs:/> echo "job1" > /queuefs/tasks/enqueue
```

**脚本编写** (`.as` 文件)：
```bash
#!/usr/bin/env agfs-shell

QUEUE_PATH=/queuefs/tasks
mkdir $QUEUE_PATH

while true; do
    task=$(cat $QUEUE_PATH/dequeue)
    echo "处理任务: $task"
    # 处理任务...
done
```

### MCP 服务器

**配置** (`~/.config/claude/claude_desktop_config.json`)：
```json
{
  "mcpServers": {
    "agfs": {
      "command": "agfs-mcp",
      "env": {
        "AGFS_SERVER_URL": "http://localhost:8080"
      }
    }
  }
}
```

**可用工具**：
- `agfs_cat`、`agfs_write`、`agfs_rm`
- `agfs_ls`、`agfs_mkdir`
- `agfs_upload`、`agfs_download`
- `agfs_grep`、`agfs_mount`、`agfs_unmount`
- `agfs_health`、`agfs_notify`

---

## 开发环境配置

### 前置要求

- **Go**：1.21+
- **Python**：3.10+
- **Docker**（可选）
- **FUSE 库**（Linux 上支持 FUSE）

### 服务器开发

**构建**：
```bash
cd agfs-server
make build
```

**运行**：
```bash
# 默认配置
./build/agfs-server

# 自定义配置
./build/agfs-server -c config.yaml

# 自定义端口
./build/agfs-server -addr :9000
```

**测试**：
```bash
make test
```

### Shell 开发

**设置**：
```bash
cd agfs-shell
uv sync
```

**运行**：
```bash
# 交互模式
uv run agfs-shell

# 执行命令
uv run agfs-shell -c "ls /memfs"

# 执行脚本
uv run agfs-shell script.as
```

**测试**：
```bash
# 运行所有测试
uv run pytest

# 运行特定测试
uv run pytest tests/test_builtins.py -v
```

### 插件开发

**创建原生插件** (C)：

```c
// myplugin.c
#include "fsi.h"

typedef struct {
    char* root;
} MyFS;

FSI* NewFSI(ConfigMap* config) {
    MyFS* fs = calloc(1, sizeof(MyFS));
    fs->root = getConfigString(config, "root");
    return (FSI*)fs;
}

int FSI_Write(FSI* fsi, const char* path,
              const char* data, size_t size) {
    MyFS* fs = (MyFS*)fsi;
    // 实现写入逻辑
    return size;
}
```

**编译**：
```bash
gcc -shared -fPIC -o myplugin.so myplugin.c
```

**加载**：
```yaml
# config.yaml
external_plugins:
  enabled: true
  plugin_paths:
    - "./myplugin.so"
```

---

## 数据流

### 文件写入操作

```
客户端 → HTTP 处理器 → 挂载管理器 → 插件 → 后端存储
        ↓               ↓            ↓        ↓
    PUT 请求     解析路径      转发请求   存储数据
        ↓               ↓            ↓        ↓
       响应         返回结果     返回写入量   确认
```

### 插件挂载操作

```
客户端 → 挂载处理器 → 插件加载器 → 插件注册表 → 插件实例
  ↓          ↓           ↓            ↓           ↓
POST 请求  解析配置    查找插件    获取工厂     创建实例
  ↓          ↓           ↓            ↓           ↓
  响应      注册插件    返回结果    更新状态     完成挂载
```

### QueueFS 任务分发

```
Agent1 → QueueFS → 后端队列
  ↓         ↓          ↓
写入任务   存储消息   消息ID
  ↓         ↓          ↓
 确认ID   返回结果    等待

Agent2 → QueueFS → 后端队列
  ↓         ↓          ↓
读取任务   获取删除   消息数据
  ↓         ↓          ↓
 处理任务   返回数据    完成
```

---

## 测试

### 服务器测试

**单元测试**：
```bash
cd agfs-server
go test ./pkg/... -v
```

**集成测试**：
```bash
# 启动服务器
./build/agfs-server -c config.test.yaml

# 运行测试
go test ./tests/integration/... -v
```

### Shell 测试

**测试覆盖率**：25% (92 个测试用例)

```bash
cd agfs-shell

# 运行所有测试
uv run pytest

# 运行覆盖率测试
uv run pytest --cov=agfs_shell --cov-report=html

# 运行特定测试类别
uv run pytest tests/test_builtins.py
uv run pytest tests/test_parser.py
uv run pytest tests/test_pipeline.py
```

**测试类别**：
- `test_builtins.py`：命令实现
- `test_parser.py`：命令解析
- `test_pipeline.py`：管道执行
- `test_process.py`：进程管理
- `test_shell_core.py`：Shell REPL

### 手动测试

**服务器健康检查**：
```bash
curl http://localhost:8080/api/v1/health
```

**文件操作**：
```bash
# 写入
curl -X PUT "http://localhost:8080/api/v1/files?path=/memfs/test.txt" \
  -d "Hello, AGFS!"

# 读取
curl "http://localhost:8080/api/v1/files?path=/memfs/test.txt"

# 列出
curl "http://localhost:8080/api/v1/directories?path=/"
```

**Shell 操作**：
```bash
agfs-shell
agfs:/> ls /
agfs:/> echo "test" > /memfs/hello.txt
agfs:/> cat /memfs/hello.txt
```

---

## 部署

### Docker 部署

**拉取镜像**：
```bash
docker pull c4pt0r/agfs:latest
```

**运行服务器**：
```bash
# 基本运行
docker run -d -p 8080:8080 --name agfs-server c4pt0r/agfs

# 数据持久化
docker run -d -p 8080:8080 \
  -v $(pwd)/data:/data \
  -v $(pwd)/config.yaml:/config.yaml \
  --name agfs-server c4pt0r/agfs

# FUSE 支持（仅 Linux）
docker run -d -p 8080:8080 \
  --device /dev/fuse \
  --cap-add SYS_ADMIN \
  --security-opt apparmor:unconfined \
  c4pt0r/agfs
```

### Systemd 服务

**创建服务** (`/etc/systemd/system/agfs.service`)：

```ini
[Unit]
Description=AGFS Server
After=network.target

[Service]
Type=simple
User=agfs
Group=agfs
WorkingDirectory=/opt/agfs
ExecStart=/opt/agfs/agfs-server -c /etc/agfs/config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

**启用服务**：
```bash
sudo systemctl enable agfs
sudo systemctl start agfs
sudo systemctl status agfs
```

### Kubernetes 部署

**部署清单** (`agfs-deployment.yaml`)：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agfs-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: agfs
  template:
    metadata:
      labels:
        app: agfs
    spec:
      containers:
      - name: agfs
        image: c4pt0r/agfs:latest
        ports:
        - containerPort: 8080
        env:
        - name: AGFS_CONFIG
          value: "/config/config.yaml"
        volumeMounts:
        - name: config
          mountPath: /config
        - name: data
          mountPath: /data
      volumes:
      - name: config
        configMap:
          name: agfs-config
      - name: data
        persistentVolumeClaim:
          claimName: agfs-data
---
apiVersion: v1
kind: Service
metadata:
  name: agfs-service
spec:
  selector:
    app: agfs
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

### 监控

**健康检查**：
```bash
# 服务器健康
curl http://localhost:8080/api/v1/health

# 插件状态
curl http://localhost:8080/api/v1/mounts

# 服务器信息
curl http://localhost:8080/api/v1/stat?path=/serverinfo
```

**日志记录**：
```yaml
# config.yaml
server:
  log_level: debug  # debug, info, warn, error
```

**指标**（通过 ServerInfoFS）：
```bash
agfs:/> cat /serverinfo/uptime
agfs:/> cat /serverinfo/stats
```

---

## 附录

### 文件系统路径约定

| 路径类型 | 示例 | 描述 |
|----------|------|------|
| **绝对路径** | `/memfs/data.txt` | 根挂载文件系统 |
| **相对路径** | `data.txt` | 相对于当前目录 |
| **本地 FS** | `local:~/file.txt` | 本地文件系统引用 |
| **AGFS** | `agfs://remote/path` | 远程 AGFS 引用 |

### 插件功能矩阵

| 插件 | 写入 | 读取 | 列表 | 删除 | 流式 | 特殊功能 |
|------|------|------|------|------|------|----------|
| MemFS | ✓ | ✓ | ✓ | ✓ | ✗ | - |
| LocalFS | ✓ | ✓ | ✓ | ✓ | ✗ | - |
| S3FS | ✓ | ✓ | ✓ | ✓ | ✗ | 对象存储 |
| SQLFS | ✓ | ✓ | ✓ | ✓ | ✗ | 数据库 |
| QueueFS | enqueue | dequeue | status | clear | ✗ | 消息队列 |
| KVFS | ✓ | ✓ | ✓ | ✓ | ✗ | 键值存储 |
| StreamFS | ✓ | ✓ | ✗ | ✗ | ✓ | 广播 |
| VectorFS | ✓ | ✓ | ✓ | ✓ | ✗ | 语义搜索 |

### 错误代码

| 代码 | 描述 |
|------|------|
| 400 | 错误请求（参数无效） |
| 404 | 未找到（文件/目录不存在） |
| 409 | 冲突（文件已存在） |
| 500 | 内部服务器错误 |
| 503 | 服务不可用（插件错误） |

### 性能考虑

- **MemFS**：最快，仅内存
- **LocalFS**：直接磁盘访问，适合大文件
- **S3FS**：网络延迟，最适合对象存储
- **SQLFS**：数据库开销，适合元数据
- **QueueFS**：无锁队列，高吞吐量
- **StreamFS**：零拷贝流式传输，低延迟

### 安全注意事项

1. **身份验证**：未实现（添加反向代理）
2. **授权**：仅有文件权限
3. **TLS**：使用反向代理（nginx、traefik）
4. **输入验证**：文件路径已清理
5. **资源限制**：按插件配置

### 贡献指南

查看各组件的 README：
- [服务器](./agfs-server/README.md)
- [Shell](./agfs-shell/README.md)
- [FUSE](./agfs-fuse/README.md)
- [SDK](./agfs-sdk/)

---

**文档版本**：1.0
**最后更新**：2026-03-27
**项目**：AGFS (Aggregated File System)
**许可证**：Apache 2.0
