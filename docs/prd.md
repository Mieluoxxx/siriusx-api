# Siriusx-API Product Requirements Document (PRD)

**Version:** 2.0
**Date:** 2025-10-01
**Status:** In Progress
**Author:** Product Team
**Project:** Siriusx-API - Lightweight AI Model Aggregation Gateway

---

## 文档说明

本文档基于 [claude-code-nexus](../ref_proj/claude-code-nexus) 的优秀实践，结合 Siriusx-API 的独特需求重新规划。

**核心差异化**:
- **技术栈**: Astro + Go (轻量级、高性能、本地可部署)
- **部署方式**: Docker 单容器本地/自托管 (vs Nexus 的 Cloudflare SaaS)
- **核心能力**: 多供应商聚合 + 负载均衡 + 故障转移 (vs Nexus 的单供应商)

---

## 变更记录

| Date | Version | Description | Author |
|------|---------|-------------|--------|
| 2025-09-30 | 1.0 | 初始版本，基于项目简报创建 | Product Manager - John |
| 2025-10-01 | 2.0 | 基于 nexus 学习，重新规划技术栈和功能模块 | Product Team |

---

## 1. 项目愿景与目标

### 1.1 核心愿景

Siriusx-API 是一个**轻量级、可自托管的 AI 模型聚合网关**，专为个人开发者和小团队设计。通过统一的 API 接口，连接多个 OpenAI 兼容的 AI 服务供应商，提供智能负载均衡、自动故障转移和完美的 Claude Code CLI 支持。

### 1.2 解决的核心问题

#### 问题 1: 模型命名混乱

**现状**: 同一模型在不同供应商有不同命名
- 供应商 A: `claude-4`
- 供应商 B: `claude-4-sonnet`
- 供应商 C: `claude-sonnet-4`

**解决方案**: 统一模型命名系统
- 用户定义: `claude-sonnet-4`
- 映射到多个供应商的具体模型
- Claude Code 直接使用统一名称

#### 问题 2: Claude Code 格式不兼容

**现状**: Claude Code CLI 需要原生 `/v1/messages` 格式，但大多数服务只提供 OpenAI 格式

**解决方案**: 完整的格式转换引擎
- 学习自 [claude-code-nexus](../ref_proj/claude-code-nexus/src/utils/claudeConverter.ts)
- 支持双向转换: Claude ↔ OpenAI
- 完美支持流式响应 (SSE)、Tool Use、多模态

#### 问题 3: 单点故障风险

**现状**: 依赖单一供应商，限流/宕机直接导致服务中断

**解决方案**: 多供应商聚合 + 智能故障转移
- 配置多个备用供应商
- 自动检测故障 (超时、5xx、429)
- 按优先级自动切换

#### 问题 4: 缺乏细粒度控制

**现状**: 无法针对特定模型/供应商进行流量分配

**解决方案**: 细粒度负载均衡
- 统一模型 → 多个供应商-模型组合
- 每个组合独立配置权重和优先级
- 示例: `claude-sonnet-4` → 70% OneAPI/gpt-4o + 30% Azure/gpt-4o-deployment

### 1.3 目标用户

- **个人开发者**: 使用 Claude Code CLI 进行日常开发
- **小型团队**: 需要可靠的 AI 基础设施，但不想依赖云服务
- **隐私敏感用户**: 希望在本地/私有服务器部署
- **成本敏感用户**: 通过多供应商组合优化成本

### 1.4 MVP 成功标准

- **部署成功率**: 90%+ 用户能在 10 分钟内完成首次部署
- **系统可用性**: 99.5%+ (不含上游供应商故障)
- **性能**: 支持 100+ QPS，P95 延迟 < 200ms
- **用户满意度**: GitHub Star > 100，社区活跃反馈

---

## 2. 技术架构假设

### 2.1 技术栈

#### 后端

- **语言**: Go 1.21+
  - *理由*: 高性能、低内存、丰富的 HTTP 生态、静态编译
- **Web 框架**: [Gin](https://gin-gonic.com/)
  - *理由*: 轻量级、中间件丰富、性能优秀
- **数据库**: SQLite + [GORM](https://gorm.io/)
  - *理由*: 无需单独部署、文件即数据库、类型安全
- **配置管理**: [Viper](https://github.com/spf13/viper)
  - *理由*: 支持 YAML、环境变量、热重载
- **日志**: [Zap](https://github.com/uber-go/zap)
  - *理由*: 结构化日志、高性能

#### 前端

- **框架**: [Astro 4.x](https://astro.build/)
  - *理由*: 极致轻量、静态优先、Islands 架构、零 JS 默认
- **UI 组件**:
  - React 组件 (交互复杂的页面)
  - [Tailwind CSS](https://tailwindcss.com/) (快速样式开发)
  - [Headless UI](https://headlessui.com/) (无障碍组件)
- **状态管理**: [Zustand](https://zustand-demo.pmnd.rs/) (轻量级)
- **图表**: [ECharts](https://echarts.apache.org/) (监控面板)

#### 部署

- **容器化**: Docker + Docker Compose
- **基础镜像**: Alpine Linux (< 20MB)
- **持久化**: Volume 挂载 (数据库 + 配置文件)

### 2.2 项目结构

```
Siriusx-API/
├── cmd/
│   └── server/
│       └── main.go                # 主入口
├── internal/
│   ├── converter/                 # 格式转换引擎 (核心)
│   │   ├── claude_to_openai.go
│   │   ├── openai_to_claude.go
│   │   ├── stream_converter.go
│   │   └── types.go
│   ├── provider/                  # 供应商管理
│   ├── mapping/                   # 模型映射
│   ├── balancer/                  # 负载均衡
│   ├── token/                     # 令牌管理
│   └── api/                       # API 路由
├── web/                           # Astro 前端
│   ├── src/
│   │   ├── pages/
│   │   ├── components/
│   │   └── layouts/
│   └── astro.config.mjs
├── config/
│   ├── config.example.yaml
│   └── default_mappings.yaml
├── docs/
│   ├── prd.md                     # 本文档
│   ├── architecture-design.md     # 架构设计
│   └── format-conversion-study.md # 格式转换学习
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── README.md
```

**参考**: [架构设计文档](./architecture-design.md)

---

## 3. 核心功能需求

### 3.1 供应商管理

#### FR1: 供应商 CRUD

- **描述**: 支持添加、编辑、删除、查看供应商配置
- **字段**:
  - 名称: "我的 OneAPI"
  - Base URL: "https://api.oneapi.com"
  - API Key: (AES-256 加密存储)
  - 启用状态: true/false
  - 优先级: 1-100 (默认 50)
- **验证**:
  - Base URL 必须是有效的 HTTPS URL
  - API Key 非空
- **存储**: SQLite `providers` 表

#### FR2: 自动获取模型列表

- **描述**: 调用供应商的 `GET /v1/models` 接口，自动发现支持的模型
- **用途**:
  - 验证供应商配置是否正确
  - 辅助用户配置模型映射
- **错误处理**: 请求失败时显示友好错误信息

#### FR3: 健康检查

- **描述**: 定期检测供应商可用性
- **检测方式**:
  - 调用 `/v1/models` 接口
  - 超时时间: 5 秒
- **检测频率**: 每 5 分钟 (可配置)
- **状态**: healthy | unhealthy | unknown
- **展示**: 在 Dashboard 和供应商列表中显示状态图标

#### FR4: API Key 加密

- **算法**: AES-256-GCM
- **密钥来源**: 环境变量 `ENCRYPTION_KEY` (启动时必须提供)
- **存储**: 仅存储加密后的密文
- **使用**: 运行时解密后发送给上游

---

### 3.2 模型管理

#### FR5: 统一模型命名

- **描述**: 创建用户自定义的统一模型名称
- **示例**:
  - `claude-sonnet-4`
  - `claude-haiku-fast`
  - `gpt4-turbo`
- **字段**:
  - 名称: 唯一，不可重复
  - 描述: "平衡性能的 Sonnet 模型"
- **用途**: 作为 Claude Code CLI 请求中的 `model` 字段

#### FR6: 模型映射配置

- **描述**: 将统一模型映射到多个供应商-模型组合
- **配置维度**:
  ```
  统一模型: claude-sonnet-4
    ├── 供应商 A → gpt-4o (权重: 70%, 优先级: 1)
    ├── 供应商 B → gpt-4o-deployment (权重: 30%, 优先级: 2)
    └── 供应商 C → gpt-4 (权重: 0%, 优先级: 3, 备用)
  ```
- **权重 (Weight)**: 0-100，用于负载均衡，总和可以不等于 100
- **优先级 (Priority)**: 1, 2, 3...，数字越小优先级越高，用于故障转移

#### FR7: ClaudeCode 快捷配置

- **描述**: 一键设置 haiku/sonnet/opus 的默认映射
- **预设映射**:
  ```yaml
  claude-3-haiku → claude-haiku (映射到用户配置的供应商)
  claude-3-5-sonnet → claude-sonnet-4
  claude-3-opus → claude-opus
  ```
- **用途**: 简化 Claude Code CLI 初始配置

---

### 3.3 格式转换引擎

#### FR8: Claude → OpenAI 请求转换

**描述**: 将 Claude Messages API 请求转换为 OpenAI Chat Completions API 请求

**转换规则**:

| Claude 字段 | OpenAI 字段 | 转换规则 |
|------------|------------|----------|
| `system` | `messages[0]` | 转为 `{"role": "system", "content": "..."}` 并放在首位 |
| `messages` | `messages` | 逐条转换（见下表） |
| `tools` | `tools` | 转为 `[{"type": "function", "function": {...}}]` |
| `tool_choice` | `tool_choice` | `{"type": "auto"}` → `"auto"`<br>`{"type": "any"}` → `"required"` |
| `max_tokens` | `max_tokens` | 直接映射 |
| `temperature` | `temperature` | 直接映射 |
| `stream` | `stream` | 直接映射 |

**Messages 转换规则**:

| Claude Message | OpenAI Message | 说明 |
|----------------|----------------|------|
| `{"role": "user", "content": "text"}` | `{"role": "user", "content": "text"}` | 文本消息 |
| `{"type": "image", "source": {...}}` | `{"type": "image_url", "image_url": {"url": "data:..."}}` | 图片消息 |
| `{"type": "tool_use", ...}` | `{"role": "assistant", "tool_calls": [...]}` | 工具调用 |
| `{"type": "tool_result", ...}` | `{"role": "tool", "tool_call_id": "...", "content": "..."}` | 工具结果 |

**参考**: [格式转换学习文档](./format-conversion-study.md#11-请求转换claude--openai)

#### FR9: OpenAI → Claude 响应转换

**描述**: 将 OpenAI API 响应转换为 Claude API 响应

**非流式转换**:

| OpenAI | Claude | 说明 |
|--------|--------|------|
| `choices[0].message.content` | `content[0]` | 转为 `{"type": "text", "text": "..."}` |
| `choices[0].message.tool_calls` | `content[1...]` | 每个 tool_call 转为 `{"type": "tool_use", ...}` |
| `choices[0].finish_reason` | `stop_reason` | `"stop"` → `"end_turn"`<br>`"length"` → `"max_tokens"`<br>`"tool_calls"` → `"tool_use"` |
| `usage.prompt_tokens` | `usage.input_tokens` | 直接映射 |
| `usage.completion_tokens` | `usage.output_tokens` | 直接映射 |

**流式转换 (SSE)** - **核心技术难点**:

**事件序列**:
```
1. message_start        # 开始消息
2. content_block_start  # 开始内容块 (文本或工具)
3. content_block_delta  # 内容增量 (text_delta 或 input_json_delta)
4. content_block_stop   # 结束内容块
5. message_delta        # 消息元数据 (stop_reason, usage)
6. message_stop         # 结束消息
```

**状态管理**:
- 跟踪当前 content block 索引
- 累积工具调用参数
- 正确处理文本块和工具块的交替

**参考**: [格式转换学习文档](./format-conversion-study.md#12-流式响应sse---核心难点)

#### FR10: 流式响应支持

- **协议**: Server-Sent Events (SSE)
- **格式**:
  ```
  event: content_block_delta
  data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

  ```
- **实现**: Go `StreamConverter` 类
- **性能**: 零拷贝、增量解析、及时刷新缓冲区

---

### 3.4 负载均衡与故障转移

#### FR11: 权重负载均衡

**描述**: 根据配置的权重，按比例分配请求到不同供应商

**算法**: 加权随机选择
```
权重: [A: 70, B: 30]
总权重: 100
随机数: 0-99
0-69 → 选择 A
70-99 → 选择 B
```

**示例**:
```
统一模型: claude-sonnet-4
├── OneAPI/gpt-4o (权重: 70) → 70% 流量
└── Azure/gpt-4o (权重: 30) → 30% 流量
```

#### FR12: 智能故障检测

**故障条件**:
- 请求超时 (默认 30 秒，可配置)
- HTTP 5xx 错误
- HTTP 429 Too Many Requests (限流)
- 连接失败

**检测方式**: 在发送请求后立即判断响应

#### FR13: 自动故障转移

**流程**:
1. 按优先级排序所有映射
2. 尝试优先级最高的供应商
3. 如果失败且满足故障条件 → 尝试下一个
4. 最多重试 N 次 (默认 3 次，可配置)
5. 所有供应商都失败 → 返回最后一个错误

**示例**:
```
统一模型: claude-sonnet-4
├── 供应商 A (优先级: 1) → 超时 → 故障转移
├── 供应商 B (优先级: 2) → 成功 ✓
└── 供应商 C (优先级: 3) → (未使用)
```

#### FR14: 故障恢复

**机制**:
- 通过健康检查恢复标记为"不可用"的供应商
- 或设置超时 (5 分钟后自动恢复)

---

### 3.5 令牌管理

#### FR15: API Token 生成

- **格式**: `sk-` + 32 字节 base64 编码随机字符串
- **用途**: 用户在 Claude Code CLI 中设置 `ANTHROPIC_API_KEY`
- **字段**:
  - 名称: "我的开发 Token"
  - Token: "sk-xxx"
  - 启用状态: true/false
  - 过期时间: 可选

#### FR16: Token 验证

- **中间件**: 在 `/v1/messages` 和 `/v1/chat/completions` 端点验证
- **验证逻辑**:
  1. 从 `Authorization: Bearer <token>` 提取 Token
  2. 查询数据库验证有效性
  3. 检查是否启用且未过期
- **失败响应**: `401 Unauthorized`

---

### 3.6 API 端点

#### FR17: Claude Messages API

**端点**: `POST /v1/messages`

**功能**: 提供 Claude 原生格式接口，供 Claude Code CLI 使用

**请求示例**:
```json
{
  "model": "claude-sonnet-4",
  "max_tokens": 1024,
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "stream": true
}
```

**响应**: Claude 格式 (流式或非流式)

**处理流程**:
1. Token 验证
2. 解析 Claude 请求
3. 查找模型映射
4. 负载均衡选择供应商
5. 转换为 OpenAI 请求
6. 发送到供应商 (支持故障转移)
7. 转换响应为 Claude 格式
8. 返回给客户端

#### FR18: OpenAI Chat Completions API

**端点**: `POST /v1/chat/completions`

**功能**: 提供 OpenAI 兼容格式接口

**请求示例**:
```json
{
  "model": "claude-sonnet-4",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "stream": false
}
```

**响应**: OpenAI 格式

**处理流程**: 类似 `/v1/messages`，但跳过格式转换（直接透传）

#### FR19: 管理 API

```
# 供应商管理
GET    /api/providers          # 列出所有供应商
POST   /api/providers          # 创建供应商
GET    /api/providers/:id      # 获取供应商详情
PUT    /api/providers/:id      # 更新供应商
DELETE /api/providers/:id      # 删除供应商
POST   /api/providers/:id/health-check  # 触发健康检查

# 模型管理
GET    /api/models             # 列出所有统一模型
POST   /api/models             # 创建统一模型
GET    /api/models/:id         # 获取模型详情
PUT    /api/models/:id         # 更新模型
DELETE /api/models/:id         # 删除模型

# 模型映射管理
GET    /api/models/:id/mappings      # 获取模型的所有映射
POST   /api/models/:id/mappings      # 添加映射
PUT    /api/mappings/:id             # 更新映射
DELETE /api/mappings/:id             # 删除映射

# 令牌管理
GET    /api/tokens             # 列出所有 Token
POST   /api/tokens             # 创建 Token
DELETE /api/tokens/:id         # 删除 Token

# ClaudeCode 配置
GET    /api/claude/config      # 获取 ClaudeCode 配置
POST   /api/claude/setup-defaults # 一键设置默认配置

# 监控
GET    /api/stats              # 获取请求统计
GET    /api/health             # 健康检查
```

---

### 3.7 Web 管理界面

#### FR20: Dashboard (仪表盘)

**功能**:
- 供应商健康状态卡片 (绿色=健康、红色=故障)
- 请求统计图表 (总数、成功数、失败数)
- 最近事件日志 (配置变更、故障转移)

**技术**: Astro + React + ECharts

#### FR21: 供应商管理页面

**功能**:
- 供应商列表 (表格或卡片)
- 添加/编辑供应商对话框
- 删除确认对话框
- 健康检查按钮
- 启用/禁用开关

**交互**:
- 添加后自动测试连接
- 显示实时健康状态

#### FR22: 模型配置页面

**功能**:
- 统一模型列表
- 创建/编辑统一模型
- 为每个模型配置映射:
  - 选择供应商
  - 选择目标模型 (从供应商模型列表)
  - 配置权重 (滑块或输入框)
  - 配置优先级 (拖拽排序)
- 映射可视化 (流程图或树形结构)

**辅助功能**:
- 权重总和提示
- 优先级冲突检测

#### FR23: 令牌管理页面

**功能**:
- Token 列表
- 创建 Token (显示一次，提示保存)
- 删除 Token
- 启用/禁用开关

#### FR24: ClaudeCode 配置页面

**功能**:
- 显示环境变量配置示例:
  ```bash
  export ANTHROPIC_API_KEY="sk-xxx"
  export ANTHROPIC_BASE_URL="http://localhost:8080"
  ```
- 一键设置 haiku/sonnet/opus 默认映射
- 显示当前映射状态

---

### 3.8 配置管理

#### FR25: YAML 配置文件

**格式**:
```yaml
server:
  port: 8080
  log_level: info

database:
  path: /app/data/siriusx.db

encryption:
  key: ${ENCRYPTION_KEY}  # 从环境变量读取

health_check:
  interval: 5m
  timeout: 5s

load_balancer:
  max_retries: 3
  request_timeout: 30s
```

**位置**: `config/config.yaml`

**加载**: 支持环境变量覆盖

#### FR26: 配置导入/导出

**导出**:
- 导出所有供应商、模型、映射配置为 YAML
- 不导出敏感信息 (API Key 仅导出加密后的占位符)

**导入**:
- 从 YAML 文件导入配置
- 合并或覆盖现有配置 (用户选择)

---

### 3.9 部署与运维

#### FR27: Docker 容器化

**镜像**:
- 基础镜像: Alpine Linux
- 大小目标: < 100MB
- 包含: Go 二进制 + Astro 静态文件

**Dockerfile**:
```dockerfile
# 构建阶段
FROM golang:1.21-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o siriusx-api ./cmd/server

# 前端构建
FROM node:20-alpine AS frontend-builder
WORKDIR /app/web
COPY web/ .
RUN npm install && npm run build

# 运行阶段
FROM alpine:latest
WORKDIR /app
COPY --from=go-builder /app/siriusx-api .
COPY --from=frontend-builder /app/web/dist ./web/dist
COPY config/config.example.yaml ./config/
EXPOSE 8080
CMD ["./siriusx-api"]
```

#### FR28: Docker Compose

**配置**:
```yaml
version: '3.8'

services:
  siriusx-api:
    image: siriusx-api:latest
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
      - ./config:/app/config
    environment:
      - ENCRYPTION_KEY=${ENCRYPTION_KEY}
      - GIN_MODE=release
    restart: unless-stopped
```

#### FR29: 健康检查端点

**端点**: `GET /health`

**响应**:
```json
{
  "status": "healthy",
  "version": "2.0.0",
  "uptime": "24h30m",
  "database": "connected"
}
```

**用途**: Docker 健康检查、监控系统

#### FR30: 持久化存储

**挂载点**:
- `/app/data`: SQLite 数据库文件
- `/app/config`: 配置文件

**备份建议**: 定期备份 `/app/data` 目录

---

## 4. 非功能性需求

### 4.1 性能

**NFR1**: 系统单实例必须支持至少 **100 QPS** 的请求吞吐量

**NFR2**: 请求转发延迟 (P95) 必须低于 **200ms** (不含上游 AI 服务)

**NFR3**: 系统启动时间必须少于 **5 秒**

**NFR4**: Docker 镜像大小必须小于 **100MB**

**NFR5**: 最低资源需求: **1 核 CPU + 512MB 内存**

**优化措施**:
- HTTP 客户端连接池
- 配置缓存 (避免每次请求查询数据库)
- 流式响应零拷贝

### 4.2 可用性

**NFR6**: 系统自身可用性必须达到 **99.5%+** (不含上游供应商故障)

**NFR7**: 故障转移成功率必须达到 **95%+**

**NFR8**: 支持跨平台部署: **Linux / macOS / Windows** (通过 Docker)

### 4.3 可维护性

**NFR9**: 代码必须遵循 **Go 最佳实践** (gofmt, golint, go vet)

**NFR10**: 核心模块测试覆盖率 > **80%**

**NFR11**: 提供完整的 **API 文档** (Swagger/OpenAPI)

**NFR12**: 提供详细的 **用户手册** 和 **故障排查指南**

### 4.4 安全性

**NFR13**: API Key 必须使用 **AES-256-GCM** 加密存储

**NFR14**: 所有管理 API 必须经过 **Token 验证**

**NFR15**: 支持 **HTTPS** 部署 (通过反向代理或内置支持)

**NFR16**: 敏感日志 (API Key) 必须脱敏

### 4.5 用户体验

**NFR17**: 首次部署时间 < **10 分钟** (从下载到完成首次 API 调用)

**NFR18**: Web 界面必须支持主流浏览器 (Chrome、Firefox、Safari、Edge 最新版)

**NFR19**: Web 界面必须响应式设计 (桌面优先，支持平板)

**NFR20**: 文档覆盖率 100% 核心功能

---

## 5. UI/UX 设计目标

### 5.1 整体 UX 愿景

**设计理念**: "极简优雅" - 3 分钟内理解如何配置

**设计风格**:
- 清晰、直观、现代
- 扁平化信息架构
- 充足的留白
- 友好的错误提示

**色彩方案**:
- 主色: 蓝色系 (技术、可靠)
- 辅色: 绿色系 (成功、健康)
- 警告: 黄色系
- 错误: 红色系

### 5.2 核心交互范式

1. **渐进式披露**: 默认显示常用配置，高级选项折叠
2. **即时反馈**: 配置变更后立即保存并显示成功/失败提示
3. **自动发现**: 添加供应商后自动调用 `/v1/models` 获取模型列表
4. **拖拽排序**: 优先级配置支持拖拽调整
5. **状态可视化**: 供应商健康状态通过颜色标识

### 5.3 核心页面

#### 1. Dashboard (仪表盘)

**布局**:
```
┌────────────────────────────────────────┐
│  Siriusx-API Dashboard                 │
├────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌────────┐│
│  │供应商健康│  │请求统计  │  │快速   ││
│  │ A ●green │  │总数: 1000│  │操作   ││
│  │ B ●red   │  │成功: 950 │  │       ││
│  │ C ●green │  │失败: 50  │  │[+供应]││
│  └──────────┘  └──────────┘  └────────┘│
├────────────────────────────────────────┤
│  最近事件                              │
│  • 10:30 供应商 B 故障转移到 供应商 A │
│  • 10:25 添加供应商 C                 │
└────────────────────────────────────────┘
```

#### 2. 供应商管理

**布局**:
```
┌────────────────────────────────────────┐
│  供应商管理                [+ 添加供应商]│
├────────────────────────────────────────┤
│  ┌──────────────────────────────────┐  │
│  │ 我的 OneAPI          ● 健康      │  │
│  │ https://api.oneapi.com           │  │
│  │ [编辑] [删除] [健康检查] [禁用]  │  │
│  └──────────────────────────────────┘  │
│  ┌──────────────────────────────────┐  │
│  │ Azure OpenAI         ● 故障      │  │
│  │ https://azure.openai.com         │  │
│  │ [编辑] [删除] [健康检查] [启用]  │  │
│  └──────────────────────────────────┘  │
└────────────────────────────────────────┘
```

#### 3. 模型配置

**布局**:
```
┌────────────────────────────────────────┐
│  模型配置                  [+ 创建模型] │
├────────────────────────────────────────┤
│  统一模型: claude-sonnet-4             │
│  描述: 平衡性能的 Sonnet 模型          │
│  ┌────────────────────────────────────┐│
│  │ 映射配置:                          ││
│  │ ─ OneAPI → gpt-4o                  ││
│  │   权重: ███████░░░ 70%             ││
│  │   优先级: 1 [↑↓]                   ││
│  │                                    ││
│  │ ─ Azure → gpt-4o-deployment        ││
│  │   权重: ███░░░░░░░ 30%             ││
│  │   优先级: 2 [↑↓]                   ││
│  │                                    ││
│  │ [+ 添加映射]                       ││
│  └────────────────────────────────────┘│
└────────────────────────────────────────┘
```

### 5.4 无障碍性

**WCAG AA 标准**:
- 颜色对比度 > 4.5:1
- 键盘可访问 (Tab 键导航)
- 明确的 label 和 aria-label
- 错误提示使用文字 + 图标

---

## 6. MVP 范围与迭代计划

### 6.1 MVP (v1.0)

**目标**: 提供基础但完整的 AI 聚合网关功能

**功能清单**:
- ✅ 格式转换引擎 (Claude ↔ OpenAI)
- ✅ 流式响应支持 (SSE)
- ✅ 供应商 CRUD + 健康检查
- ✅ 统一模型命名 + 模型映射
- ✅ 权重负载均衡
- ✅ 智能故障转移
- ✅ 令牌管理
- ✅ `/v1/messages` 端点 (Claude 格式)
- ✅ `/v1/chat/completions` 端点 (OpenAI 格式)
- ✅ Web UI (Dashboard + 供应商管理 + 模型配置)
- ✅ Docker 部署

**不包含** (后续版本):
- ❌ 工具调用 (Tool Use) 转换
- ❌ 图片消息转换
- ❌ 配置导入/导出
- ❌ 高级监控 (详细日志、Prometheus 指标)

**时间**: 2-3 周

### 6.2 v1.1 - 工具调用支持

**新增**:
- ✅ Tool Use 请求转换
- ✅ Tool Use 响应转换
- ✅ 流式工具调用

**时间**: 1 周

### 6.3 v1.2 - 多模态支持

**新增**:
- ✅ 图片消息转换
- ✅ Tool result 转换

**时间**: 1 周

### 6.4 v2.0 - 企业级功能

**新增**:
- ✅ 配置导入/导出 (YAML)
- ✅ 详细请求日志
- ✅ Prometheus 指标
- ✅ 多用户支持 (可选)
- ✅ RBAC 权限管理 (可选)

**时间**: 2-3 周

---

## 7. 测试策略

### 7.1 单元测试

**覆盖模块**:
- 格式转换引擎 (converter/)
- 供应商管理 (provider/)
- 模型映射 (mapping/)
- 负载均衡 (balancer/)

**目标覆盖率**: > 80%

**工具**: Go 标准测试库 + testify

### 7.2 集成测试

**测试场景**:
1. 添加供应商 → 获取模型列表 → 创建映射
2. 发送 Claude 请求 → 负载均衡 → 格式转换 → 返回响应
3. 供应商故障 → 故障转移 → 成功响应
4. 流式响应完整性测试

### 7.3 手动测试

**测试项**:
- Web UI 可用性测试
- 不同浏览器兼容性
- Docker 部署端到端测试
- Claude Code CLI 集成测试

---

## 8. 风险与挑战

### 8.1 技术风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 流式响应转换实现复杂 | 高 | 学习 nexus 实现，充分测试 |
| 故障转移可能增加延迟 | 中 | 优化重试策略，使用连接池 |
| SQLite 并发性能瓶颈 | 中 | 使用配置缓存，考虑迁移到 PostgreSQL |

### 8.2 产品风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 用户配置复杂度高 | 高 | 提供快捷配置、详细文档、视频教程 |
| 与 Claude Code 版本兼容性 | 中 | 跟踪 Claude Code 更新，及时适配 |
| 社区采用率低 | 中 | 积极推广、收集反馈、快速迭代 |

---

## 9. 成功指标

### 9.1 技术指标

- **性能**: 100+ QPS, P95 延迟 < 200ms
- **可用性**: 99.5%+
- **故障转移成功率**: 95%+
- **测试覆盖率**: 80%+

### 9.2 用户指标

- **部署成功率**: 90%+
- **首次部署时间**: < 10 分钟
- **GitHub Star**: 100+ (3 个月内)
- **活跃用户**: 50+ (3 个月内)

### 9.3 质量指标

- **Bug 率**: < 5 个严重 Bug/月
- **文档覆盖率**: 100% 核心功能
- **社区满意度**: 4.5+ / 5.0

---

## 10. 参考资料

### 10.1 内部文档

- [架构设计文档](./architecture-design.md)
- [格式转换学习文档](./format-conversion-study.md)

### 10.2 外部参考

- [claude-code-nexus PRD](../ref_proj/claude-code-nexus/REQUIREMENTS.md)
- [claude-code-nexus 转换实现](../ref_proj/claude-code-nexus/src/utils/claudeConverter.ts)
- [Claude API 文档](https://docs.anthropic.com/claude/reference)
- [OpenAI API 文档](https://platform.openai.com/docs/api-reference)

---

## 11. 下一步行动

### 11.1 文档完善

- [ ] 执行 PM Checklist 验证 PRD 完整性
- [ ] 创建 API 文档 (Swagger/OpenAPI)
- [ ] 编写用户手册

### 11.2 技术实现

- [ ] 实现格式转换引擎 (Go)
- [ ] 实现供应商管理模块
- [ ] 实现模型映射与路由
- [ ] 实现负载均衡与故障转移
- [ ] 开发 Astro 前端界面

### 11.3 测试与部署

- [ ] 编写单元测试
- [ ] 编写集成测试
- [ ] Docker 打包与部署
- [ ] 编写部署文档

---

## 12. Epic 和 Story 分解

> **说明**: 以下 Epic 和 Story 基于 PRD 中的功能需求 (FR1-FR30) 组织，每个 Story 包含详细的任务清单、验收标准和工时估算。

### 总览

| Epic | 目标 | Story 数量 | 估算时间 |
|------|------|-----------|---------|
| Epic 1: 项目初始化与基础设施 | 完成项目基础框架搭建 | 3 个 Story | 2-3 天 |
| Epic 2: 格式转换引擎 (FR8-FR10) | 实现 Claude ↔ OpenAI 格式双向转换 | 3 个 Story | 4-5 天 |
| Epic 3: 供应商管理 (FR1-FR4) | 实现供应商的 CRUD、健康检查和加密 | 3 个 Story | 3-4 天 |
| Epic 4: 模型映射与路由 (FR5-FR7) | 实现统一模型命名和路由解析 | 3 个 Story | 2-3 天 |
| Epic 5: 负载均衡与故障转移 (FR11-FR14) | 实现智能负载均衡和自动故障转移 | 3 个 Story | 3-4 天 |
| Epic 6: 令牌管理 (FR15-FR16) | 实现 API Token 的生成、验证和管理 | 2 个 Story | 2 天 |
| Epic 7: Web 管理界面 (FR20-FR24, FR6) | 开发 Astro + React 的 Web UI | 6 个 Story | 5-7 天 |
| **总计** | **MVP 完整功能** | **23 个 Story** | **21-30 天** |

---

### Epic 1: 项目初始化与基础设施

**目标**: 完成项目基础框架搭建和开发环境配置
**优先级**: P0 (MUST HAVE)
**估算**: 2-3 天

#### Story 1.1: 创建 Go 项目骨架

**描述**: 初始化 Go 项目结构和依赖管理

**任务**:
- 初始化 Go Module (`go mod init github.com/yourusername/siriusx-api`)
- 创建目录结构 (`cmd/`, `internal/`, `web/`, `config/`, `docs/`)
- 配置 `.gitignore` 文件 (Go + Node.js + IDE)
- 编写基础 `README.md` (项目介绍、快速开始)
- 创建 `Makefile` (常用命令: build, test, run)

**验收标准**:
- [ ] `go build ./cmd/server` 成功编译
- [ ] 目录结构符合 PRD 第 2.2 节定义
- [ ] README 包含项目愿景和技术栈说明
- [ ] Git 仓库初始化完成

**估算**: 0.5 天

---

#### Story 1.2: 引入 SQLite 数据库和 GORM

**描述**: 集成 SQLite 数据库和 GORM ORM 框架

**任务**:
- 安装依赖: `gorm.io/gorm`, `gorm.io/driver/sqlite`
- 创建数据库连接管理器 (`internal/db/database.go`)
- 定义核心数据模型:
  - `Provider` (供应商)
  - `UnifiedModel` (统一模型)
  - `ModelMapping` (模型映射)
  - `Token` (API 令牌)
- 编写自动 Migration 脚本
- 添加数据库初始化逻辑 (启动时自动创建表)

**验收标准**:
- [ ] 数据库文件 `siriusx.db` 自动创建
- [ ] 4 张表成功创建 (providers, unified_models, model_mappings, tokens)
- [ ] GORM 连接池配置正确 (最大连接数 10)
- [ ] 单元测试: 数据库 CRUD 操作正常

**估算**: 1 天

---

#### Story 1.3: 配置 Docker 环境

**描述**: 创建 Docker 镜像和 Docker Compose 配置

**任务**:
- 编写 `Dockerfile` (多阶段构建: Go builder + Node builder + Alpine runtime)
- 编写 `docker-compose.yml` (端口映射 8080, 数据卷挂载)
- 配置环境变量管理 (`.env.example`)
- 编写 Docker 快速启动脚本 (`scripts/docker-start.sh`)

**验收标准**:
- [ ] `docker-compose up` 成功启动服务
- [ ] 访问 `http://localhost:8080/health` 返回 200
- [ ] 数据持久化正常 (重启容器后数据不丢失)
- [ ] Docker 镜像大小 < 100MB

**估算**: 1 天

---

### Epic 2: 格式转换引擎 (FR8-FR10)

**目标**: 实现 Claude ↔ OpenAI 格式双向转换
**优先级**: P0 (MUST HAVE)
**估算**: 4-5 天
**参考**: `ref_proj/claude-code-nexus/src/utils/claudeConverter.ts`

#### Story 2.1: 请求转换器 (Claude → OpenAI)

**描述**: 将 Claude Messages API 请求转换为 OpenAI Chat Completions API 格式

**任务**:
- 创建 `internal/converter/types.go` (定义 Claude/OpenAI 结构体)
- 实现 `ConvertClaudeToOpenAI()` 函数
- 参数映射逻辑:
  - `system` → `messages[0]` (role: "system")
  - `messages` → `messages` (逐条转换)
  - `max_tokens` → `max_tokens`
  - `temperature` → `temperature`
  - `stream` → `stream`
- 处理不支持的参数 (返回警告日志)

**验收标准**:
- [ ] 单元测试覆盖率 > 80%
- [ ] 转换正确率 100% (基于 10+ 测试用例)
- [ ] 支持 `system` 消息转换
- [ ] 性能: 转换延迟 < 1ms

**估算**: 1.5 天

---

#### Story 2.2: 响应转换器 (OpenAI → Claude)

**描述**: 将 OpenAI API 响应转换为 Claude API 格式

**任务**:
- 实现 `ConvertOpenAIToClaude()` 函数
- 字段映射逻辑:
  - `choices[0].message.content` → `content[0]` (type: "text")
  - `id` → `id`
  - `model` → `model`
  - `finish_reason` → `stop_reason` (映射规则: "stop"→"end_turn", "length"→"max_tokens")
  - `usage.prompt_tokens` → `usage.input_tokens`
  - `usage.completion_tokens` → `usage.output_tokens`
- 处理空响应和错误响应

**验收标准**:
- [ ] 单元测试覆盖率 > 80%
- [ ] 转换正确率 100%
- [ ] 支持 `finish_reason` 正确映射
- [ ] 错误响应格式符合 Claude API 规范

**估算**: 1 天

---

#### Story 2.3: 流式响应转换器 (SSE)

**描述**: 实现 Server-Sent Events (SSE) 格式的流式响应转换

**任务**:
- 创建 `StreamConverter` 结构体 (维护状态)
- 实现 SSE 事件解析器
- 转换 OpenAI SSE → Claude SSE:
  - OpenAI `data: {delta: {content: "..."}}` → Claude `event: content_block_delta`
  - 生成 Claude 事件序列: `message_start` → `content_block_start` → `content_block_delta` → `content_block_stop` → `message_delta` → `message_stop`
- 处理 `[DONE]` 事件
- 实现零拷贝流式传输 (使用 `io.Pipe`)

**验收标准**:
- [ ] 流式响应正确率 > 95% (对比 10+ 真实响应)
- [ ] 事件顺序正确 (符合 Claude API 规范)
- [ ] 性能: 零拷贝传输，延迟 < 10ms
- [ ] 集成测试: 与真实 OpenAI API 联调成功

**估算**: 2 天

---

### Epic 3: 供应商管理 (FR1-FR4)

**目标**: 实现供应商的 CRUD、健康检查和 API Key 加密
**优先级**: P0 (MUST HAVE)
**估算**: 3-4 天

#### Story 3.1: 供应商 CRUD API

**描述**: 实现供应商的创建、查询、更新、删除功能

**任务**:
- 创建 `internal/provider/service.go` (业务逻辑层)
- 创建 `internal/provider/repository.go` (数据访问层)
- 实现 API 端点 (`internal/api/handlers/provider_handler.go`):
  - `POST /api/providers` (创建供应商)
  - `GET /api/providers` (查询供应商列表，支持分页)
  - `GET /api/providers/:id` (查询单个供应商)
  - `PUT /api/providers/:id` (更新供应商)
  - `DELETE /api/providers/:id` (软删除，标记为 disabled)
- 添加参数验证 (名称、Base URL、API Key 非空)

**验收标准**:
- [ ] 所有 API 端点返回正确的 HTTP 状态码
- [ ] 支持分页查询 (page, page_size)
- [ ] 软删除不物理删除数据
- [ ] 单元测试覆盖率 > 80%
- [ ] API 响应延迟 < 100ms

**估算**: 1.5 天

---

#### Story 3.2: API Key 加密存储

**描述**: 使用 AES-256-GCM 加密存储供应商 API Key

**任务**:
- 创建 `internal/crypto/encryption.go` (加密工具)
- 实现 AES-256-GCM 加密/解密函数
- 从环境变量 `ENCRYPTION_KEY` 读取加密密钥 (32 字节)
- 在创建/更新供应商时自动加密 API Key
- 在使用 API Key 时自动解密
- API 响应中脱敏显示 API Key (如 `sk-****1234`)

**验收标准**:
- [ ] 加密算法为 AES-256-GCM
- [ ] 加密后的 API Key 不可逆向破解
- [ ] 缺少 `ENCRYPTION_KEY` 时启动失败并提示错误
- [ ] API 响应中 API Key 脱敏显示
- [ ] 单元测试: 加密/解密往返成功

**估算**: 1 天

---

#### Story 3.3: 供应商健康检查

**描述**: 定期检测供应商的健康状态

**任务**:
- 创建 `internal/provider/health_checker.go`
- 实现健康检查逻辑:
  - 调用供应商的 `/v1/models` 端点
  - 超时时间: 5 秒
  - 成功 (200) → 标记为 `healthy`
  - 失败 (超时/5xx) → 标记为 `unhealthy`
- 实现定时任务 (每 5 分钟执行一次)
- 添加手动触发健康检查的 API: `POST /api/providers/:id/health-check`
- 记录健康检查历史 (最近 10 次)

**验收标准**:
- [ ] 定时任务正常运行 (每 5 分钟)
- [ ] 手动触发 API 返回最新健康状态
- [ ] 健康状态更新延迟 < 10s
- [ ] 健康检查不阻塞主服务
- [ ] 集成测试: 模拟供应商故障场景

**估算**: 1 天

---

### Epic 4: 模型映射与路由 (FR5-FR7)

**目标**: 实现统一模型命名和路由解析
**优先级**: P0 (MUST HAVE)
**估算**: 2-3 天

#### Story 4.1: 统一模型管理

**描述**: 创建和管理用户自定义的统一模型名称

**任务**:
- 创建 `internal/mapping/service.go` (业务逻辑层)
- 创建 `internal/mapping/repository.go` (数据访问层)
- 实现 API 端点:
  - `POST /api/models` (创建统一模型)
  - `GET /api/models` (查询模型列表)
  - `GET /api/models/:id` (查询单个模型)
  - `PUT /api/models/:id` (更新模型)
  - `DELETE /api/models/:id` (删除模型)
- 添加名称唯一性校验

**验收标准**:
- [ ] 模型名称唯一性校验正常
- [ ] 支持自定义描述字段
- [ ] API 响应延迟 < 50ms
- [ ] 单元测试覆盖率 > 80%

**估算**: 1 天

---

#### Story 4.2: 模型映射配置

**描述**: 将统一模型映射到具体供应商的模型

**任务**:
- 实现映射 CRUD API:
  - `POST /api/models/:id/mappings` (为模型添加映射)
  - `GET /api/models/:id/mappings` (查询模型的所有映射)
  - `PUT /api/mappings/:id` (更新映射)
  - `DELETE /api/mappings/:id` (删除映射)
- 映射字段:
  - `provider_id` (供应商 ID)
  - `target_model` (目标模型名称)
  - `weight` (权重 1-100)
  - `priority` (优先级 1, 2, 3...)
  - `enabled` (是否启用)
- 添加外键约束校验

**验收标准**:
- [ ] 支持一个统一模型映射到多个供应商
- [ ] 权重范围校验 (1-100)
- [ ] 优先级不重复校验
- [ ] 单元测试覆盖率 > 80%

**估算**: 1 天

---

#### Story 4.3: 路由解析逻辑

**描述**: 根据请求中的模型名称解析到具体供应商

**任务**:
- 创建 `internal/mapping/router.go`
- 实现 `ResolveModel(modelName string)` 函数:
  - 根据统一模型名称查询映射关系
  - 过滤启用且健康的供应商
  - 按优先级排序映射列表
  - 返回可用的映射列表
- 添加缓存机制 (TTL: 5 分钟)
- 处理模型不存在的情况 (返回 404)

**验收标准**:
- [ ] 路由解析延迟 < 10ms (含缓存)
- [ ] 未命中缓存时延迟 < 50ms
- [ ] 模型不存在时返回友好错误消息
- [ ] 单元测试: 测试多种路由场景

**估算**: 1 天

---

### Epic 5: 负载均衡与故障转移 (FR11-FR14)

**目标**: 实现智能负载均衡和自动故障转移
**优先级**: P0 (MUST HAVE)
**估算**: 3-4 天

#### Story 5.1: 加权随机负载均衡器

**描述**: 根据供应商权重分配请求

**任务**:
- 创建 `internal/balancer/weighted_balancer.go`
- 实现加权随机算法:
  - 计算总权重
  - 生成随机数 (0 ~ 总权重-1)
  - 根据累积权重选择供应商
- 实现 `SelectProvider(mappings []ModelMapping)` 函数
- 添加单元测试 (验证权重分布)

**验收标准**:
- [ ] 权重分配误差 < 5% (运行 1000 次)
- [ ] 选择算法延迟 < 1ms
- [ ] 单元测试覆盖率 > 90%
- [ ] 支持动态权重调整

**估算**: 1 天

---

#### Story 5.2: 故障检测逻辑

**描述**: 检测供应商故障并标记为不可用

**任务**:
- 创建 `internal/balancer/failure_detector.go`
- 实现故障检测函数 `IsFailure(err error, resp *http.Response) bool`:
  - 请求超时 (30 秒) → 故障
  - HTTP 5xx 错误 → 故障
  - HTTP 429 限流 → 故障
  - 连接失败 → 故障
- 实现故障计数器 (连续 3 次故障 → 冷却 5 分钟)
- 添加故障恢复机制 (冷却期后自动恢复)

**验收标准**:
- [ ] 故障检测准确率 > 95%
- [ ] 冷却期机制正常工作
- [ ] 单元测试: 模拟各种故障场景
- [ ] 集成测试: 真实供应商故障模拟

**估算**: 1.5 天

---

#### Story 5.3: 智能故障转移

**描述**: 自动切换到备用供应商

**任务**:
- 创建 `internal/balancer/failover.go`
- 实现 `ExecuteWithFailover(ctx context.Context, request *Request, mappings []ModelMapping)` 函数:
  - 按优先级尝试每个供应商
  - 检测故障 → 自动切换到下一个
  - 最多重试 3 次
  - 所有供应商都失败 → 返回 503
- 记录故障转移日志
- 添加 Prometheus 指标 (故障转移次数)

**验收标准**:
- [ ] 故障转移成功率 > 95%
- [ ] 重试次数不超过 3 次
- [ ] 故障转移延迟 < 5s
- [ ] 集成测试: 主供应商故障场景
- [ ] 日志记录完整 (包含故障原因)

**估算**: 1.5 天

---

### Epic 6: 令牌管理 (FR15-FR16)

**目标**: 实现 API Token 的生成、验证和管理
**优先级**: P0 (MUST HAVE)
**估算**: 2 天

#### Story 6.1: Token 生成和存储

**描述**: 生成唯一的 API Token 并存储到数据库

**任务**:
- 创建 `internal/token/service.go`
- 实现 `GenerateToken(name string, expiresAt *time.Time)` 函数:
  - 生成格式: `sk-` + 32 字节 base64 随机字符串
  - 确保唯一性 (查询数据库校验)
  - 存储到数据库 (tokens 表)
- 实现 Token CRUD API:
  - `POST /api/tokens` (创建 Token)
  - `GET /api/tokens` (查询 Token 列表)
  - `DELETE /api/tokens/:id` (删除 Token)
- Token 响应中完整显示一次,后续脱敏

**验收标准**:
- [ ] Token 唯一性 100% (无重复)
- [ ] Token 格式符合 `sk-[a-zA-Z0-9]{43}` 正则
- [ ] 创建 Token 后仅显示一次完整内容
- [ ] 后续查询自动脱敏 (如 `sk-****abc123`)
- [ ] 单元测试覆盖率 > 80%

**估算**: 1 天

---

#### Story 6.2: Token 验证中间件

**描述**: 实现 HTTP 中间件验证 API Token

**任务**:
- 创建 `internal/api/middleware/auth.go`
- 实现 `TokenAuthMiddleware()` 函数:
  - 从请求头 `Authorization: Bearer sk-xxx` 提取 Token
  - 查询数据库验证 Token 是否存在
  - 检查 Token 是否启用 (`enabled = true`)
  - 检查 Token 是否过期 (`expires_at > NOW()`)
  - 验证失败 → 返回 `401 Unauthorized`
  - 验证成功 → 将 Token 信息存入 Context
- 应用到所有需要认证的端点

**验收标准**:
- [ ] Token 验证延迟 < 5ms
- [ ] 无效 Token 返回 401 错误
- [ ] 过期 Token 返回 401 错误
- [ ] 缺少 Authorization 头返回 401 错误
- [ ] 单元测试: 覆盖所有验证场景
- [ ] 集成测试: 真实 HTTP 请求验证

**估算**: 1 天

---

### Epic 7: Web 管理界面 (FR20-FR24, FR6)

**目标**: 开发 Astro + React 的 Web UI
**优先级**: P1 (SHOULD HAVE)
**估算**: 5-7 天

#### Story 7.1: 搭建 Astro + React 项目

**描述**: 初始化前端项目框架

**任务**:
- 创建 Astro 项目 (`web/` 目录)
- 安装依赖: React, Tailwind CSS, shadcn/ui, Zustand, ECharts
- 配置 `astro.config.mjs` (React 集成、代理配置)
- 创建基础布局 (`src/layouts/Layout.astro`)
- 配置 Tailwind CSS 主题
- 集成 shadcn/ui 组件库

**验收标准**:
- [ ] `npm run dev` 成功启动开发服务器
- [ ] Tailwind CSS 样式正常工作
- [ ] shadcn/ui 组件可正常使用
- [ ] 代理配置正确 (转发 API 请求到 Go 后端)

**估算**: 0.5 天

---

#### Story 7.2: Dashboard (仪表盘) 页面

**描述**: 开发系统概览页面

**任务**:
- 创建 `src/pages/index.astro` (Dashboard 页面)
- 实现功能组件:
  - 系统概览卡片 (总供应商、健康/不健康供应商、总请求数)
  - 请求监控图表 (ECharts 折线图,显示 QPS)
  - 最近事件日志 (故障转移、配置变更)
- 实现数据获取逻辑:
  - 调用 `GET /api/stats` 获取统计数据
  - 使用 Zustand 管理状态
  - 自动刷新 (1 秒间隔)
- 实现响应式布局 (桌面优先)

**验收标准**:
- [ ] 卡片数据实时更新 (1 秒刷新)
- [ ] 图表展示最近 1 小时的 QPS
- [ ] 最近事件日志显示最新 10 条
- [ ] 响应式布局在 1920x1080 和 1366x768 下正常显示
- [ ] 页面加载时间 < 1 秒

**估算**: 1.5 天

---

#### Story 7.3: 供应商管理页面

**描述**: 开发供应商 CRUD 界面

**任务**:
- 创建 `src/pages/providers.astro`
- 实现功能组件:
  - 供应商列表表格 (shadcn/ui Table)
  - 添加/编辑供应商对话框 (shadcn/ui Dialog + Form)
  - 删除确认对话框
  - 健康检查按钮
  - 启用/禁用开关 (shadcn/ui Switch)
- 实现交互逻辑:
  - 创建供应商 → 调用 `POST /api/providers`
  - 编辑供应商 → 调用 `PUT /api/providers/:id`
  - 删除供应商 → 调用 `DELETE /api/providers/:id`
  - 手动健康检查 → 调用 `POST /api/providers/:id/health-check`
- 实现表单验证 (名称、Base URL、API Key 非空)

**验收标准**:
- [ ] 所有 CRUD 操作响应 < 500ms
- [ ] 表单验证正确 (显示错误消息)
- [ ] 健康状态实时更新 (绿色/红色图标)
- [ ] API Key 脱敏显示 (`sk-****1234`)
- [ ] 操作成功/失败有 Toast 提示

**估算**: 2 天

---

#### Story 7.4: 模型配置页面

**描述**: 开发模型映射管理界面

**任务**:
- 创建 `src/pages/models.astro`
- 实现功能组件:
  - 统一模型列表
  - 创建/编辑统一模型对话框
  - 映射配置区域 (嵌套表格或卡片)
  - 添加映射对话框 (选择供应商、目标模型、权重、优先级)
  - 映射可视化 (Mermaid 流程图或树形结构)
- 实现交互逻辑:
  - 创建统一模型 → `POST /api/models`
  - 添加映射 → `POST /api/models/:id/mappings`
  - 更新映射 → `PUT /api/mappings/:id`
  - 删除映射 → `DELETE /api/mappings/:id`
- 辅助功能:
  - 权重总和提示 (实时计算)
  - 优先级冲突检测 (高亮显示)

**验收标准**:
- [ ] 权重滑块实时更新 (1-100)
- [ ] 优先级拖拽排序正常工作
- [ ] 映射可视化直观展示路由关系
- [ ] 权重总和提示准确
- [ ] 优先级冲突时显示警告

**估算**: 2 天

---

#### Story 7.5: 令牌管理页面

**描述**: 开发 API Token 管理界面

**任务**:
- 创建 `src/pages/tokens.astro`
- 实现功能组件:
  - Token 列表表格 (名称、Token 脱敏、有效期、操作)
  - 创建 Token 对话框 (输入名称、有效期)
  - Token 显示对话框 (仅显示一次完整 Token,提示保存)
  - 复制 Token 按钮
  - 删除确认对话框
- 实现交互逻辑:
  - 创建 Token → `POST /api/tokens` → 显示完整 Token
  - 复制 Token → 使用 Clipboard API
  - 删除 Token → `DELETE /api/tokens/:id`

**验收标准**:
- [ ] 创建 Token 后显示完整内容 (仅一次)
- [ ] 复制 Token 功能正常 (显示 Toast 提示)
- [ ] Token 列表中 Token 脱敏显示
- [ ] 删除 Token 需要确认

**估算**: 1 天

---

#### Story 7.6: ClaudeCode 配置页面

**描述**: 开发 ClaudeCode 配置生成工具

**任务**:
- 创建 `src/pages/claude-config.astro`
- 实现功能组件:
  - 配置信息展示 (API Base URL、可用模型列表)
  - Token 选择器 (下拉框)
  - 生成配置按钮
  - 配置文件内容展示 (JSON 格式,语法高亮)
  - 复制到剪贴板按钮
  - 手动配置步骤说明 (有序列表)
- 实现交互逻辑:
  - 查询可用模型 → `GET /api/models`
  - 查询 Token 列表 → `GET /api/tokens`
  - 生成配置 JSON
  - 复制配置 → Clipboard API

**验收标准**:
- [ ] 生成的 JSON 格式正确
- [ ] 复制功能正常工作
- [ ] 手动步骤说明清晰易懂
- [ ] 选择不同 Token 时配置自动更新

**估算**: 1 天

---

## 附录: Epic 和 Story 使用指南

### 如何使用这些 Epic 和 Story

1. **选择 Epic**: 按照优先级 (Epic 1 → Epic 7) 顺序开发
2. **选择 Story**: 每个 Epic 内的 Story 按顺序执行 (如 1.1 → 1.2 → 1.3)
3. **创建 Story 文件**: 使用 BMad 命令 `*task create-next-story` 创建详细的 Story 文件
4. **跟踪进度**: 使用 Story 文件中的验收标准跟踪完成情况

### Story 命名规范

- Story 文件命名: `{epicNum}.{storyNum}.story.md`
- 示例: `1.1.story.md`, `2.3.story.md`
- 存储位置: `docs/stories/` 目录

### 估算说明

- **工时单位**: 1 天 = 8 小时工作时间
- **总估算**: 21-30 天 (约 3-4 周，包含测试和调试)
- **建议节奏**: 每周完成 2-3 个 Epic

---

**文档状态**: ✅ Epic 分解已完成 - 可以开始开发
**下一步**: 运行 `*task create-next-story` 创建 Story 1.1
