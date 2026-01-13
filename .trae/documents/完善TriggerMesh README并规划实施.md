# TriggerMesh 项目实施计划

## 一、项目初始化

### 1. 创建项目结构
```bash
mkdir -p cmd/triggermesh internal/api/handlers internal/api/middleware internal/config internal/engine/jenkins internal/logger internal/storage/models tests/unit tests/integration tests/e2e
```

### 2. 初始化Go模块
```bash
go mod init triggermesh
```

### 3. 创建基础配置文件
- `config.yaml.example`：配置示例
- `.gitignore`：Git忽略文件

## 二、核心功能实现

### 1. 日志系统
- **实现文件**：`internal/logger/logger.go`
- **功能**：基于slog的结构化日志
- **特性**：支持日志级别、JSON格式输出

### 2. 配置管理
- **实现文件**：`internal/config/config.go`
- **功能**：YAML配置加载与环境变量覆盖
- **特性**：配置验证、默认值设置

### 3. 数据库系统
- **实现文件**：`internal/storage/sqlite.go`, `internal/storage/models/audit.go`
- **功能**：SQLite数据库操作
- **特性**：自动迁移、审计日志存储

### 4. 认证系统
- **实现文件**：`internal/api/middleware/auth.go`
- **功能**：API Key验证
- **特性**：支持多个API Key、中间件机制

### 5. CI引擎抽象层
- **实现文件**：`internal/engine/interface.go`
- **功能**：定义CI引擎通用接口
- **特性**：支持未来扩展其他CI引擎

### 6. Jenkins集成
- **实现文件**：`internal/engine/jenkins/client.go`, `internal/engine/jenkins/trigger.go`
- **功能**：Jenkins API客户端与触发逻辑
- **特性**：参数化触发、Token安全管理

### 7. HTTP API
- **实现文件**：`internal/api/router.go`, `internal/api/handlers/jenkins.go`, `internal/api/handlers/audit.go`
- **功能**：HTTP服务器与路由配置
- **特性**：RESTful API设计、统一错误处理

### 8. 主程序入口
- **实现文件**：`cmd/triggermesh/main.go`
- **功能**：程序初始化与启动
- **特性**：命令行参数解析、优雅关闭

## 三、测试实现

### 1. 单元测试
- **实现文件**：`tests/unit/auth_test.go`, `tests/unit/config_test.go`, `tests/unit/engine_test.go`
- **测试框架**：Go标准库`testing` + `testify/assert`
- **覆盖率目标**：核心功能≥80%

### 2. 集成测试
- **实现文件**：`tests/integration/api_test.go`, `tests/integration/jenkins_test.go`
- **测试框架**：Go标准库`testing` + `testify/assert`
- **环境**：测试SQLite数据库 + Jenkins测试实例

### 3. 端到端测试
- **实现文件**：`tests/e2e/trigger_test.go`
- **测试工具**：Go HTTP客户端
- **环境**：完整的TriggerMesh服务实例

## 四、部署配置

### 1. Docker配置
- **实现文件**：`Dockerfile`
- **特性**：最小化镜像、多阶段构建

### 2. Docker Compose配置
- **实现文件**：`docker-compose.yml`
- **特性**：一键部署、环境隔离

### 3. 构建脚本
- **实现文件**：`Makefile`
- **功能**：构建、测试、部署自动化

## 五、开发工作流

### 1. 代码风格
- 使用`gofmt`格式化代码
- 使用`golint`进行代码检查
- 使用`go vet`进行静态分析

### 2. 提交规范
- 提交信息格式：`[类型] 描述`
- 类型包括：feat(新功能)、fix(修复)、docs(文档)、style(样式)、refactor(重构)、test(测试)、chore(构建)

### 3. 测试流程
- 本地开发：运行单元测试
- 提交代码：CI自动运行所有测试
- 发布前：运行完整测试套件

## 六、预期成果

### 1. 核心功能
- ✅ 统一HTTP Trigger API
- ✅ API Key鉴权
- ✅ 触发请求与结果审计
- ✅ 参数化触发Jenkins Pipeline/Job
- ✅ Jenkins Token内聚，不对外暴露

### 2. 技术实现
- ✅ 基于Go 1.21+开发
- ✅ 使用SQLite数据库
- ✅ 基于slog的结构化日志
- ✅ 支持Docker容器化部署

### 3. 测试覆盖
- ✅ 单元测试覆盖率≥80%
- ✅ 集成测试覆盖关键集成点
- ✅ 端到端测试覆盖主要场景

## 七、实施时间表

| 阶段 | 预计时间 | 主要工作 |
|------|----------|----------|
| 项目初始化 | 1天 | 创建目录结构、初始化Go模块 |
| 核心功能实现 | 5天 | 日志、配置、数据库、认证、CI引擎、HTTP API |
| 测试实现 | 3天 | 单元测试、集成测试、端到端测试 |
| 部署配置 | 2天 | Docker配置、Docker Compose配置、构建脚本 |
| 文档完善 | 1天 | README.md、API文档、开发指南 |
| 测试与优化 | 2天 | 性能测试、安全检查、bug修复 |

## 八、未来扩展方向

1. 支持更多CI引擎（GitLab CI、GitHub Actions等）
2. 实现Web UI管理界面
3. 支持更复杂的触发规则
4. 实现告警功能
5. 支持分布式部署
6. 升级到PostgreSQL数据库
7. 实现API Key权限精细化管理
8. 支持多种认证方式
9. 实现流量控制
10. 支持灰度发布