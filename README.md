# Siriusx-API

> 轻量级、可自托管的 AI 模型聚合网关

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-In%20Development-yellow)](https://github.com/Mieluoxxx/Siriusx-API)

---

## 📖 项目愿景

Siriusx-API 是一个**轻量级、可自托管的 AI 模型聚合网关**，旨在解决多供应商 API 管理的复杂性问题。

### 核心目标

- 🔌 **多供应商聚合**: 统一管理来自不同 AI 服务提供商的 API
- ⚖️ **智能负载均衡**: 支持优先级和权重配置的请求分发
- 🔄 **格式转换**: 支持 OpenAI 和 Claude 两种 API 格式的无缝转换
- 🛡️ **故障转移**: 自动故障检测和请求重试机制
- 🪶 **轻量部署**: 单一二进制文件，开箱即用，资源占用极低
- 🔧 **易于扩展**: 模块化设计，方便添加新的供应商和功能

---

## ✨ 核心特性

### 1. 供应商管理
- 支持多个 OpenAI 兼容的上游供应商
- 每个供应商支持多 API Key 轮询
- 供应商级别的健康检查和故障转移

### 2. 模型映射
- 自定义模型名称，解决命名混乱问题
- 支持模型别名和重定向
- 优先级和权重配置

### 3. API 格式转换
- **上游**: 接收 OpenAI 兼容格式 (`/v1/chat/completions`)
- **下游**: 支持转发到 OpenAI 或 Claude 格式
- 流式响应 (SSE) 格式自动转换

### 4. Claude Code 支持
- 完整支持 Claude Code CLI 的消息格式
- 支持 Claude 特有的思考模式和工具调用
- 针对 Claude API 的格式优化

---

## 🛠️ 技术栈

### 后端
- **语言**: Go 1.21+
- **Web 框架**: [Gin](https://github.com/gin-gonic/gin) - 轻量、高性能
- **数据库**: SQLite + [GORM](https://gorm.io/) - 类型安全 ORM
- **配置管理**: [Viper](https://github.com/spf13/viper) - 支持 YAML 和环境变量
- **日志**: [Zap](https://github.com/uber-go/zap) - 结构化高性能日志
- **HTTP 客户端**: 标准库 `net/http` + 连接池

### 前端
- **框架**: [Astro 4.x](https://astro.build/) - 极致轻量、静态优先
- **UI 组件**: React + [Tailwind CSS](https://tailwindcss.com/) + [Headless UI](https://headlessui.com/)
- **状态管理**: [Zustand](https://zustand-demo.pmnd.rs/) - 轻量级状态管理
- **图表**: [ECharts](https://echarts.apache.org/) - 监控面板可视化

### 部署
- **容器化**: Docker + Docker Compose
- **基础镜像**: Alpine Linux (极致轻量)
- **持久化**: Volume 挂载（数据库 + 配置文件）

---

## 🚀 快速开始

### 环境要求

- **Go**: 1.21 或更高版本
- **Docker**: 20.10+ (可选，用于容器化部署)
- **Node.js**: 20+ (可选，用于前端开发)

### 安装依赖

```bash
# 克隆项目
git clone https://github.com/Mieluoxxx/Siriusx-API.git
cd Siriusx-API

# 下载 Go 依赖
go mod download
```

### 启动服务

```bash
# 使用 Makefile 启动
make run

# 或直接使用 Go 命令
go run ./cmd/server
```

### 构建项目

```bash
# 构建二进制文件
make build

# 运行编译后的程序
./bin/siriusx-api
```

---

## 🔐 安全配置

### API Key 加密存储

Siriusx-API 支持对存储在数据库中的供应商 API Key 进行加密，以提高安全性。

#### 配置加密密钥

```bash
# 生成 32 字节的随机加密密钥
openssl rand -base64 32

# 或者使用 Go 生成
go run -c 'package main; import ("crypto/rand"; "encoding/base64"; "fmt"); func main() { key := make([]byte, 32); rand.Read(key); fmt.Println(base64.StdEncoding.EncodeToString(key)) }'
```

#### 环境变量配置

```bash
# 在 .env 文件或环境变量中设置
export ENCRYPTION_KEY="your-32-byte-base64-encoded-key"

# 启动服务
go run ./cmd/server
```

#### 生产环境要求

```bash
# 在生产环境中，必须配置加密密钥
export GO_ENV=production
export ENCRYPTION_KEY="your-secure-encryption-key"
```

#### 加密特性

- 🔒 **算法**: AES-256-GCM (认证加密)
- 🔄 **随机性**: 每次加密生成新的 Nonce
- 🛡️ **完整性**: 内置防篡改验证
- 📱 **脱敏显示**: API 响应中显示为 `sk-****1234`
- ⚡ **高性能**: 硬件加速支持，单次操作 < 1ms

#### 安全最佳实践

1. **密钥管理**:
   - 使用环境变量存储密钥，避免硬编码
   - 生产环境使用强随机生成的 32 字节密钥
   - 定期轮换加密密钥（需要重新加密现有数据）

2. **环境隔离**:
   - 开发环境可以不配置加密（会有警告提示）
   - 生产环境必须配置 `GO_ENV=production` 和 `ENCRYPTION_KEY`

3. **备份与恢复**:
   - 备份数据库时，确保同时备份加密密钥
   - 加密密钥丢失将导致无法解密现有 API Key

---

## 📂 项目结构

```
Siriusx-API/
├── cmd/
│   └── server/
│       └── main.go              # 主入口
├── internal/                    # 私有包
│   ├── converter/               # 格式转换引擎 (核心)
│   ├── provider/                # 供应商管理
│   ├── crypto/                  # 加密模块 (AES-256-GCM)
│   ├── mapping/                 # 模型映射
│   ├── balancer/                # 负载均衡
│   ├── token/                   # 令牌管理
│   ├── api/                     # API 路由和中间件
│   ├── config/                  # 配置管理
│   ├── db/                      # 数据库连接与迁移
│   └── models/                  # 数据模型
├── web/                         # Astro 前端项目
├── config/                      # 配置文件模板
│   └── config.example.yaml      # 配置示例
├── docs/                        # 项目文档
│   ├── prd.md                   # 产品需求文档
│   ├── architecture-design.md   # 架构设计
│   └── stories/                 # 开发 Story
├── Dockerfile                   # Docker 镜像构建
├── docker-compose.yml           # Docker Compose 配置
├── Makefile                     # 构建脚本
└── README.md                    # 本文档
```

---

## 📚 文档

- [产品需求文档 (PRD)](docs/prd.md)
- [架构设计文档](docs/architecture-design.md)
- [格式转换研究](docs/format-conversion-study.md)
- [开发 Story](docs/stories/)

---

## 🔧 开发

### 可用的 Makefile 命令

```bash
make build      # 编译项目
make run        # 启动开发服务器
make test       # 运行测试
make clean      # 清理构建产物
make help       # 显示帮助信息
```

### 编码规范

- 遵循 Go 标准库风格指南
- 使用 `gofmt` 格式化代码
- 使用 `golangci-lint` 进行代码检查
- 包命名使用小写单数形式 (如 `provider` 而非 `providers`)

### 测试策略

- **单元测试**: 覆盖率目标 > 80%
- **集成测试**: 覆盖关键业务流程
- **E2E 测试**: 覆盖主要用户场景

---

## 🗺️ 开发路线图

### Epic 1: 项目初始化与基础设施 ✅
- [x] Story 1.1: 创建 Go 项目骨架
- [x] Story 1.2: 引入 SQLite 数据库和 GORM
- [ ] Story 1.3: 配置 Docker 环境

### Epic 2: 核心转换引擎 ✅
- [x] Story 2.1: 实现 Claude → OpenAI 请求转换器
- [x] Story 2.2: 实现 OpenAI → Claude 响应转换器
- [x] Story 2.3: 实现流式响应转换器 (Server-Sent Events)

### Epic 3: 供应商管理 ✅
- [x] Story 3.1: 供应商 CRUD API
- [x] Story 3.2: API Key 加密存储 ✅

### Epic 4-7: 更多功能敬请期待...

详细路线图请查看 [PRD 文档](docs/prd.md)。

---

## 📄 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

---

## 🤝 贡献

欢迎贡献！请先阅读 [贡献指南](CONTRIBUTING.md)（待补充）。

---

## 📧 联系方式

- **项目主页**: [https://github.com/Mieluoxxx/Siriusx-API](https://github.com/Mieluoxxx/Siriusx-API)
- **问题反馈**: [GitHub Issues](https://github.com/Mieluoxxx/Siriusx-API/issues)

---

**当前状态**: 🚧 项目骨架阶段 | **版本**: v0.1.0 | **最后更新**: 2025-10-02
