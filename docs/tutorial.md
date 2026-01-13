# TriggerMesh 详细教程

> **提示**：如果您只需要快速开始，请查看 [README](../README-zh.md) 的快速开始部分。
> 
> 本文档是 TriggerMesh 的详细中文教程，涵盖了安装、配置、使用、故障排查等完整内容。

## 一、项目介绍

### 1. 什么是 TriggerMesh

TriggerMesh 是一个轻量级、可控、可审计的 CI Build 触发中枢服务，主要功能包括：

- 统一 HTTP Trigger API
- API Key 鉴权（最小安全边界）
- 触发请求与结果审计
- 支持 Jenkins Pipeline / Job 触发
- Jenkins Token 内聚，不对外暴露

### 2. 系统架构

```
[ Caller System ]
        |
        v
[ TriggerMesh API ]
        |
        v
[ Jenkins HTTP API ]
        |
        v
[ Jenkins Agents ]
```

### 3. 技术栈

- **语言**：Go 1.21+
- **数据库**：SQLite（可平滑升级 PostgreSQL）
- **认证**：API Key
- **日志**：slog（Go 官方日志库）
- **部署**：Docker

## 二、系统要求

### 1. 硬件要求

- CPU：1 核以上
- 内存：512MB 以上
- 存储：1GB 以上可用空间

### 2. 软件要求

- **Go**：1.21+（如果从源码编译）
- **Docker**：20.10+（如果使用 Docker 部署）
- **Jenkins**：2.0+（需要集成 Jenkins 时）

## 三、安装方法

### 1. 从源码编译

```bash
# 克隆仓库
git clone https://github.com/nesnilnehc/triggermesh.git
cd triggermesh

# 编译
go build -o triggermesh ./cmd/triggermesh

# 运行
./triggermesh --config config.yaml
```

### 2. 使用 Docker 部署

```bash
# 克隆仓库
git clone https://github.com/nesnilnehc/triggermesh.git
cd triggermesh

# 构建镜像
docker build -t triggermesh .

# 运行容器
docker run -d -p 8080:8080 -v ./config.yaml:/app/config.yaml triggermesh
```

### 3. 使用 Docker Compose 部署

```bash
# 克隆仓库
git clone https://github.com/nesnilnehc/triggermesh.git
cd triggermesh

# 创建配置文件
cp config.yaml.example config.yaml

# 编辑配置文件
vi config.yaml

# 启动服务
docker-compose up -d
```

## 四、配置说明

### 1. 配置文件格式

配置文件采用 YAML 格式，示例配置：

```yaml
server:
  port: 8080
  host: "0.0.0.0"

database:
  path: ./triggermesh.db

jenkins:
  url: https://your-jenkins-url
  token: your-jenkins-token

api:
  keys:
    - your-api-key
```

### 2. 配置项说明

#### 2.1 服务器配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `server.port` | int | 8080 | 服务监听端口 |
| `server.host` | string | "0.0.0.0" | 服务监听地址 |

#### 2.2 数据库配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `database.path` | string | ./triggermesh.db | SQLite 数据库文件路径 |

#### 2.3 Jenkins 配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `jenkins.url` | string | - | Jenkins 服务器地址 |
| `jenkins.token` | string | - | Jenkins API Token |

#### 2.4 API 配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `api.keys` | []string | - | 允许访问的 API Key 列表 |

### 3. 环境变量覆盖

所有配置项都可以通过环境变量覆盖，格式为 `TRIGGERMESH_<SECTION>_<KEY>`，例如：

```bash
export TRIGGERMESH_SERVER_PORT=9090
export TRIGGERMESH_JENKINS_TOKEN=secret-token
export TRIGGERMESH_LOG_LEVEL=debug
```

## 五、使用指南

### 1. 启动服务

#### 1.1 直接启动

```bash
# 使用默认配置文件
go run cmd/triggermesh/main.go

# 使用指定配置文件
go run cmd/triggermesh/main.go --config config.yaml

# 使用环境变量指定端口
PORT=8081 go run cmd/triggermesh/main.go --config config.yaml
```

#### 1.2 使用 Makefile

```bash
# 构建并运行
make run

# 仅构建
make build

# 运行测试
make test
```

### 2. 健康检查

```bash
curl http://localhost:8080/health
# 响应：OK
```

### 3. 触发 Jenkins 构建

#### 3.1 基本请求

```bash
curl -X POST \
  http://localhost:8080/api/v1/trigger/jenkins \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{"job": "test-job"}'
```

#### 3.2 带参数的请求

```bash
curl -X POST \
  http://localhost:8080/api/v1/trigger/jenkins \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "job": "test-job",
    "parameters": {
      "branch": "main",
      "environment": "production"
    }
  }'
```

#### 3.3 响应示例

```json
{
  "success": true,
  "build_id": "",
  "build_url": "https://your-jenkins-url/job/test-job/",
  "message": "Successfully triggered Jenkins build for job test-job"
}
```

### 4. 查询审计日志

```bash
curl -H "Authorization: Bearer your-api-key" http://localhost:8080/api/v1/audit
```

响应示例：

```json
[
  {
    "id": 1,
    "timestamp": "2026-01-05T22:30:00Z",
    "api_key": "your-api-key",
    "method": "POST",
    "path": "/api/v1/trigger/jenkins",
    "status": 200,
    "job_name": "test-job",
    "params": "{\"branch\":\"main\",\"environment\":\"production\"}",
    "result": "success"
  }
]
```

## 六、API 文档

### 1. 根路径

```
GET /
```

返回 API 信息和可用端点。

### 2. 健康检查

```
GET /health
```

返回服务健康状态。

### 3. 触发 Jenkins 构建

```
POST /api/v1/trigger/jenkins
```

**请求体**：

```json
{
  "job": "string",
  "parameters": {"key": "value"}
}
```

**响应**：

```json
{
  "success": true,
  "build_id": "string",
  "build_url": "string",
  "message": "string"
}
```

### 4. 查询审计日志

```
GET /api/v1/audit?limit=100&offset=0
```

**参数**：
- `limit`：返回记录数，默认 100
- `offset`：偏移量，默认 0

**响应**：

```json
[
  {
    "id": 1,
    "timestamp": "2026-01-05T22:30:00Z",
    "api_key": "string",
    "method": "string",
    "path": "string",
    "status": 200,
    "job_name": "string",
    "params": "string",
    "result": "string",
    "error": "string"
  }
]
```

## 七、故障排查

### 1. 端口冲突

**问题**：启动服务时提示端口已被占用

**解决方案**：

```bash
# 使用不同端口
PORT=8081 ./triggermesh --config config.yaml

# 或修改配置文件
vi config.yaml
# 修改 server.port 为其他端口
```

### 2. Jenkins 连接失败

**问题**：触发 Jenkins 构建时提示连接失败

**解决方案**：

1. 检查 Jenkins URL 是否正确
2. 检查 Jenkins Token 是否有效
3. 检查 Jenkins 服务是否正常运行
4. 检查网络连接是否正常

### 3. API Key 验证失败

**问题**：请求 API 时返回 401 Unauthorized

**解决方案**：

1. 检查 API Key 是否正确
2. 检查 API Key 是否在配置文件的 `api.keys` 列表中
3. 检查 Authorization 头格式是否正确（应为 `Bearer your-api-key`）

### 4. 数据库初始化失败

**问题**：启动服务时提示数据库初始化失败

**解决方案**：

1. 检查数据库文件路径是否正确
2. 检查是否有写入文件的权限
3. 确保没有其他进程占用数据库文件

## 八、最佳实践

### 1. 安全建议

- 定期更换 API Key 和 Jenkins Token
- 限制 API Key 的使用范围
- 启用日志审计，定期检查审计日志
- 使用 HTTPS 协议（在生产环境）

### 2. 性能优化

- 根据实际负载调整服务器资源
- 定期清理审计日志
- 考虑使用 PostgreSQL 替代 SQLite（在高并发场景）

### 3. 高可用性

- 考虑使用负载均衡器
- 实现数据库备份策略
- 配置监控和告警

## 九、开发指南

### 1. 项目结构

```
triggermesh/
├── cmd/triggermesh/       # 主程序入口
├── docs/                  # 文档
├── internal/              # 内部包
│   ├── api/              # API 相关代码
│   ├── config/           # 配置管理
│   ├── engine/           # CI 引擎抽象层
│   ├── logger/           # 日志系统
│   └── storage/          # 数据库操作
├── tests/                # 测试目录
├── config.yaml.example   # 配置示例
├── Dockerfile            # Docker 配置
├── docker-compose.yml    # Docker Compose 配置
├── Makefile              # 构建脚本
└── README.md             # 项目文档
```

### 2. 开发流程

1. 克隆仓库
2. 安装依赖：`go mod tidy`
3. 运行测试：`go test ./internal/...`
4. 启动开发服务器：`make run`
5. 提交代码前运行：`make fmt vet test`

### 3. 添加新的 CI 引擎

1. 实现 `internal/engine/interface.go` 中的 `CIEngine` 接口
2. 在 `internal/engine/` 目录下创建新的引擎目录
3. 在主程序中注册新的引擎

### 4. 测试

- **单元测试**：`go test ./internal/...`
- **集成测试**：`go test ./tests/integration/...`
- **端到端测试**：`go test ./tests/e2e/...`

## 十、常见问题

### 1. 如何查看日志？

日志默认输出到 stderr，格式为 JSON。可以通过环境变量 `TRIGGERMESH_LOG_LEVEL` 设置日志级别：

```bash
export TRIGGERMESH_LOG_LEVEL=debug
```

### 2. 如何备份数据库？

SQLite 数据库可以直接复制文件备份：

```bash
cp triggermesh.db triggermesh.db.backup
```

### 3. 如何升级到 PostgreSQL？

1. 导出 SQLite 数据
2. 创建 PostgreSQL 数据库和表结构
3. 导入数据到 PostgreSQL
4. 修改配置文件使用 PostgreSQL

### 4. 如何添加新的 API 端点？

1. 在 `internal/api/handlers/` 目录下创建新的处理器
2. 在 `internal/api/router.go` 中注册新的路由
3. 更新 API 文档

## 十一、联系与支持

- **项目地址**：https://github.com/nesnilnehc/triggermesh
- **问题反馈**：请通过 [GitHub Issues](https://github.com/nesnilnehc/triggermesh/issues) 提交

## 十二、许可证

本项目采用 MIT 许可证，详见 LICENSE 文件。
