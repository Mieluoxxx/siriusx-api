# Siriusx-API 架构设计文档

> 基于 Astro + Go 技术栈的轻量级 AI 模型聚合网关
> 核心学习自 claude-code-nexus，增强多供应商管理和负载均衡

---

## 1. 技术栈

### 1.1 后端

- **语言**: Go 1.21+
- **Web 框架**: Gin (轻量、高性能、中间件丰富)
- **数据库**: SQLite + GORM (类型安全、轻量级)
- **配置管理**: Viper (支持 YAML、环境变量)
- **日志**: Zap (结构化、高性能)
- **HTTP 客户端**: 标准库 `net/http` + 连接池

### 1.2 前端

- **框架**: Astro 4.x (极致轻量、静态优先)
- **UI 组件**:
  - React 组件（用于交互复杂的页面）
  - Tailwind CSS (快速样式开发)
  - Headless UI (无障碍组件)
- **状态管理**: Zustand (轻量级)
- **图表**: ECharts (监控面板)

### 1.3 部署

- **容器化**: Docker + Docker Compose
- **基础镜像**: Alpine Linux
- **持久化**: Volume 挂载 (数据库 + 配置文件)

---

## 2. 系统架构

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────┐
│                      Claude Code CLI                         │
│  (设置 ANTHROPIC_BASE_URL=http://localhost:8080)            │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                    Siriusx-API Gateway                       │
├─────────────────────────────────────────────────────────────┤
│  Web UI (Astro)                                              │
│  ├── 供应商管理                                              │
│  ├── 模型映射配置                                            │
│  ├── 令牌管理                                                │
│  ├── ClaudeCode 配置 (haiku/sonnet/opus 映射)               │
│  └── 监控面板                                                │
├─────────────────────────────────────────────────────────────┤
│  API Gateway (Go + Gin)                                      │
│  ├── /v1/messages        (Claude 原生格式)                   │
│  ├── /v1/chat/completions (OpenAI 兼容格式)                 │
│  └── /api/*              (管理 API)                          │
├─────────────────────────────────────────────────────────────┤
│  核心模块                                                     │
│  ├── 格式转换引擎 (Claude ↔ OpenAI)                         │
│  ├── 供应商管理 (CRUD + 健康检查)                           │
│  ├── 模型映射路由 (统一命名 → 供应商模型)                   │
│  ├── 负载均衡器 (权重分配)                                   │
│  ├── 故障转移 (智能重试)                                     │
│  └── 令牌管理 (API Key 验证)                                │
├─────────────────────────────────────────────────────────────┤
│  存储层                                                       │
│  └── SQLite (providers, models, mappings, tokens)            │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ↓
┌─────────────────────────────────────────────────────────────┐
│              多个 OpenAI 兼容供应商                          │
├─────────────────────────────────────────────────────────────┤
│  供应商 A: OneAPI                                            │
│  └── gpt-4o-mini, gpt-4o, ...                               │
├─────────────────────────────────────────────────────────────┤
│  供应商 B: Azure OpenAI                                      │
│  └── gpt-4o-deployment, ...                                 │
├─────────────────────────────────────────────────────────────┤
│  供应商 C: 本地 Ollama                                       │
│  └── llama3, qwen, ...                                      │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. 核心模块设计

### 3.1 格式转换引擎

**位置**: `internal/converter/`

**职责**:
- Claude API 请求 → OpenAI API 请求
- OpenAI API 响应 → Claude API 响应
- 流式响应 (SSE) 转换

**核心文件**:
```
internal/converter/
├── claude_to_openai.go      # 请求转换
├── openai_to_claude.go      # 非流式响应转换
├── stream_converter.go      # 流式响应转换（关键！）
├── types.go                 # Claude/OpenAI 结构体定义
└── converter_test.go        # 单元测试
```

**核心接口**:
```go
package converter

// 请求转换
func ConvertClaudeToOpenAI(
    claudeReq *ClaudeRequest,
    targetModel string,
) (*OpenAIRequest, error)

// 响应转换
func ConvertOpenAIToClaude(
    openAIResp *OpenAIResponse,
    originalModel string,
) (*ClaudeResponse, error)

// 流式转换器
type StreamConverter struct {
    MessageID         string
    OriginalModel     string
    ContentBlocks     []ContentBlock
    ToolArgsBuffer    map[string]string
    TotalInputTokens  int
    TotalOutputTokens int
}

func NewStreamConverter(originalModel string) *StreamConverter
func (sc *StreamConverter) GenerateInitialEvents() []SSEEvent
func (sc *StreamConverter) ProcessOpenAIChunk(chunk *OpenAIChunk) []SSEEvent
func (sc *StreamConverter) GenerateFinishEvents(finishReason string) []SSEEvent
```

**参考**: [格式转换核心学习文档](./format-conversion-study.md)

---

### 3.2 供应商管理

**位置**: `internal/provider/`

**职责**:
- 供应商 CRUD
- API Key 加密存储
- 健康检查 (定期 ping `/v1/models`)
- 启用/禁用状态管理

**数据库表设计**:
```sql
CREATE TABLE providers (
    id          TEXT PRIMARY KEY,  -- UUID
    name        TEXT NOT NULL,     -- "我的 OneAPI"
    base_url    TEXT NOT NULL,     -- "https://api.oneapi.com"
    api_key     TEXT NOT NULL,     -- AES 加密
    enabled     BOOLEAN DEFAULT 1, -- 启用状态
    priority    INTEGER DEFAULT 50,-- 优先级 (1-100)
    health_status TEXT DEFAULT 'unknown', -- healthy|unhealthy|unknown
    last_check  DATETIME,          -- 最后健康检查时间
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**核心接口**:
```go
package provider

type Provider struct {
    ID           string    `gorm:"primaryKey"`
    Name         string    `gorm:"not null"`
    BaseURL      string    `gorm:"not null"`
    APIKey       string    `gorm:"not null"` // 加密存储
    Enabled      bool      `gorm:"default:true"`
    Priority     int       `gorm:"default:50"`
    HealthStatus string    `gorm:"default:'unknown'"`
    LastCheck    time.Time
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type Service interface {
    Create(provider *Provider) error
    Update(id string, provider *Provider) error
    Delete(id string) error
    GetByID(id string) (*Provider, error)
    List() ([]*Provider, error)
    HealthCheck(id string) error
    HealthCheckAll() error
}
```

**健康检查逻辑**:
```go
func (s *ProviderService) HealthCheck(providerID string) error {
    provider, err := s.GetByID(providerID)
    if err != nil {
        return err
    }

    // 调用 /v1/models 接口
    url := provider.BaseURL + "/v1/models"
    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+decryptAPIKey(provider.APIKey))

    resp, err := httpClient.Do(req)
    if err != nil || resp.StatusCode != 200 {
        provider.HealthStatus = "unhealthy"
    } else {
        provider.HealthStatus = "healthy"
    }

    provider.LastCheck = time.Now()
    return s.Update(providerID, provider)
}
```

---

### 3.3 模型映射与路由

**位置**: `internal/mapping/`

**职责**:
- 统一模型命名管理
- 模型 → 供应商映射
- 细粒度映射到具体供应商的具体模型
- 权重和优先级配置

**数据库表设计**:
```sql
CREATE TABLE unified_models (
    id          TEXT PRIMARY KEY,  -- UUID
    name        TEXT NOT NULL UNIQUE, -- "claude-sonnet-4"
    description TEXT,              -- "高性能 Sonnet 模型"
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE model_mappings (
    id               TEXT PRIMARY KEY,
    unified_model_id TEXT NOT NULL,  -- 关联 unified_models.id
    provider_id      TEXT NOT NULL,  -- 关联 providers.id
    target_model     TEXT NOT NULL,  -- "gpt-4o"
    weight           INTEGER DEFAULT 50, -- 负载均衡权重 (0-100)
    priority         INTEGER DEFAULT 1,  -- 故障转移优先级 (1, 2, 3...)
    enabled          BOOLEAN DEFAULT 1,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (unified_model_id) REFERENCES unified_models(id),
    FOREIGN KEY (provider_id) REFERENCES providers(id)
);
```

**核心接口**:
```go
package mapping

type UnifiedModel struct {
    ID          string    `gorm:"primaryKey"`
    Name        string    `gorm:"unique;not null"` // "claude-sonnet-4"
    Description string
    CreatedAt   time.Time
    UpdatedAt   time.Time
    Mappings    []ModelMapping `gorm:"foreignKey:UnifiedModelID"`
}

type ModelMapping struct {
    ID             string `gorm:"primaryKey"`
    UnifiedModelID string `gorm:"not null"`
    ProviderID     string `gorm:"not null"`
    TargetModel    string `gorm:"not null"` // "gpt-4o"
    Weight         int    `gorm:"default:50"`
    Priority       int    `gorm:"default:1"`
    Enabled        bool   `gorm:"default:true"`
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type Service interface {
    // 统一模型管理
    CreateUnifiedModel(model *UnifiedModel) error
    UpdateUnifiedModel(id string, model *UnifiedModel) error
    DeleteUnifiedModel(id string) error
    GetUnifiedModel(id string) (*UnifiedModel, error)
    ListUnifiedModels() ([]*UnifiedModel, error)

    // 映射管理
    AddMapping(mapping *ModelMapping) error
    UpdateMapping(id string, mapping *ModelMapping) error
    DeleteMapping(id string) error
    GetMappingsByUnifiedModel(unifiedModelID string) ([]*ModelMapping, error)

    // 路由逻辑
    ResolveModel(unifiedModelName string) ([]*ModelMapping, error)
}
```

**路由逻辑示例**:
```go
func (s *MappingService) ResolveModel(unifiedModelName string) ([]*ModelMapping, error) {
    // 1. 查找统一模型
    unifiedModel, err := s.GetUnifiedModelByName(unifiedModelName)
    if err != nil {
        return nil, err
    }

    // 2. 获取所有启用的映射
    mappings, err := s.GetMappingsByUnifiedModel(unifiedModel.ID)
    if err != nil {
        return nil, err
    }

    // 3. 过滤启用且供应商健康的映射
    validMappings := []ModelMapping{}
    for _, mapping := range mappings {
        if mapping.Enabled {
            provider, _ := providerService.GetByID(mapping.ProviderID)
            if provider != nil && provider.Enabled && provider.HealthStatus == "healthy" {
                validMappings = append(validMappings, mapping)
            }
        }
    }

    // 4. 按优先级排序
    sort.Slice(validMappings, func(i, j int) bool {
        return validMappings[i].Priority < validMappings[j].Priority
    })

    return validMappings, nil
}
```

---

### 3.4 负载均衡与故障转移

**位置**: `internal/balancer/`

**职责**:
- 按权重选择供应商
- 故障检测 (超时、5xx、限流)
- 自动故障转移 (按优先级重试)

**核心接口**:
```go
package balancer

type LoadBalancer interface {
    // 按权重选择供应商
    SelectProvider(mappings []*ModelMapping) (*ModelMapping, error)

    // 故障转移请求
    ExecuteWithFailover(
        ctx context.Context,
        claudeReq *ClaudeRequest,
        mappings []*ModelMapping,
    ) (*ClaudeResponse, error)
}

type WeightedLoadBalancer struct {
    providerService provider.Service
    httpClient      *http.Client
}

func NewWeightedLoadBalancer(providerService provider.Service) *WeightedLoadBalancer
```

**选择逻辑**:
```go
func (lb *WeightedLoadBalancer) SelectProvider(mappings []*ModelMapping) (*ModelMapping, error) {
    if len(mappings) == 0 {
        return nil, errors.New("no available providers")
    }

    // 计算总权重
    totalWeight := 0
    for _, mapping := range mappings {
        totalWeight += mapping.Weight
    }

    // 生成随机数
    rand.Seed(time.Now().UnixNano())
    randomWeight := rand.Intn(totalWeight)

    // 选择供应商
    cumulativeWeight := 0
    for _, mapping := range mappings {
        cumulativeWeight += mapping.Weight
        if randomWeight < cumulativeWeight {
            return mapping, nil
        }
    }

    return mappings[0], nil
}
```

**故障转移逻辑**:
```go
func (lb *WeightedLoadBalancer) ExecuteWithFailover(
    ctx context.Context,
    claudeReq *ClaudeRequest,
    mappings []*ModelMapping,
) (*ClaudeResponse, error) {
    var lastErr error

    // 按优先级尝试每个供应商
    for _, mapping := range mappings {
        provider, err := lb.providerService.GetByID(mapping.ProviderID)
        if err != nil {
            continue
        }

        // 转换请求
        openAIReq, err := converter.ConvertClaudeToOpenAI(claudeReq, mapping.TargetModel)
        if err != nil {
            lastErr = err
            continue
        }

        // 发送请求
        resp, err := lb.sendRequest(ctx, provider, openAIReq)
        if err != nil {
            lastErr = err
            // 检测是否需要故障转移
            if shouldFailover(err, resp) {
                log.Warnf("Provider %s failed, trying next...", provider.Name)
                continue
            }
            return nil, err
        }

        // 成功，转换响应
        claudeResp, err := converter.ConvertOpenAIToClaude(resp, claudeReq.Model)
        if err != nil {
            lastErr = err
            continue
        }

        return claudeResp, nil
    }

    return nil, fmt.Errorf("all providers failed: %w", lastErr)
}

// 判断是否需要故障转移
func shouldFailover(err error, resp *http.Response) bool {
    // 超时
    if err != nil && errors.Is(err, context.DeadlineExceeded) {
        return true
    }

    // 5xx 错误
    if resp != nil && resp.StatusCode >= 500 {
        return true
    }

    // 429 限流
    if resp != nil && resp.StatusCode == 429 {
        return true
    }

    return false
}
```

---

### 3.5 令牌管理

**位置**: `internal/token/`

**职责**:
- API Token 生成
- Token 验证
- 权限管理（可选）

**数据库表设计**:
```sql
CREATE TABLE tokens (
    id          TEXT PRIMARY KEY,  -- UUID
    name        TEXT NOT NULL,     -- "我的开发 Token"
    token       TEXT NOT NULL UNIQUE, -- "sk-xxx" (生成的 API Key)
    enabled     BOOLEAN DEFAULT 1,
    expires_at  DATETIME,          -- 可选的过期时间
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**核心接口**:
```go
package token

type Token struct {
    ID        string    `gorm:"primaryKey"`
    Name      string    `gorm:"not null"`
    Token     string    `gorm:"unique;not null"` // "sk-xxx"
    Enabled   bool      `gorm:"default:true"`
    ExpiresAt *time.Time
    CreatedAt time.Time
    UpdatedAt time.Time
}

type Service interface {
    Create(name string) (*Token, error)
    Validate(token string) (*Token, error)
    Delete(id string) error
    List() ([]*Token, error)
}
```

**生成逻辑**:
```go
func (s *TokenService) Create(name string) (*Token, error) {
    // 生成 32 字节随机字符串
    randomBytes := make([]byte, 32)
    rand.Read(randomBytes)
    tokenStr := "sk-" + base64.URLEncoding.EncodeToString(randomBytes)

    token := &Token{
        ID:        uuid.New().String(),
        Name:      name,
        Token:     tokenStr,
        Enabled:   true,
        CreatedAt: time.Now(),
    }

    return token, s.db.Create(token).Error
}
```

---

### 3.6 ClaudeCode 配置管理

**位置**: `internal/claude/`

**职责**:
- 管理 haiku/sonnet/opus 的模型映射
- 提供快捷配置接口

**预设配置示例**:
```go
var DefaultClaudeCodeMappings = map[string]string{
    "claude-3-haiku":     "claude-haiku",
    "claude-3-5-sonnet":  "claude-sonnet-4",
    "claude-3-opus":      "claude-opus",
}
```

**核心接口**:
```go
package claude

type ClaudeCodeService struct {
    mappingService mapping.Service
}

func (s *ClaudeCodeService) SetupDefaultMappings() error {
    // 创建统一模型
    models := []mapping.UnifiedModel{
        {Name: "claude-haiku", Description: "快速响应的 Haiku 模型"},
        {Name: "claude-sonnet-4", Description: "平衡性能的 Sonnet 模型"},
        {Name: "claude-opus", Description: "最强大的 Opus 模型"},
    }

    for _, model := range models {
        s.mappingService.CreateUnifiedModel(&model)
    }

    return nil
}
```

---

### 3.7 配置导入导出 (FR26)

**位置**: `internal/config/`

**职责**:
- 导出完整系统配置到 YAML/JSON 文件
- 从文件导入配置并恢复系统状态
- 配置文件格式验证
- 支持增量导入（合并而非覆盖）

#### 3.7.1 配置文件格式

**完整配置文件示例** (`siriusx_config.yaml`):
```yaml
version: "1.0"
exported_at: "2025-10-01T12:00:00Z"

providers:
  - id: "550e8400-e29b-41d4-a716-446655440000"
    name: "我的 OneAPI"
    base_url: "https://api.oneapi.com"
    api_key: "sk-xxx"  # 导出时保留加密或明文（用户选择）
    enabled: true
    priority: 50

  - id: "660e8400-e29b-41d4-a716-446655440001"
    name: "Azure OpenAI"
    base_url: "https://myazure.openai.azure.com"
    api_key: "encrypted:AES256:xxxxx"
    enabled: true
    priority: 60

unified_models:
  - id: "model-001"
    name: "claude-sonnet-4"
    description: "高性能 Sonnet 模型"
    mappings:
      - provider_id: "550e8400-e29b-41d4-a716-446655440000"
        target_model: "gpt-4o"
        weight: 70
        priority: 1
        enabled: true

      - provider_id: "660e8400-e29b-41d4-a716-446655440001"
        target_model: "gpt-4-deployment"
        weight: 30
        priority: 2
        enabled: true

  - id: "model-002"
    name: "claude-haiku"
    description: "快速响应模型"
    mappings:
      - provider_id: "550e8400-e29b-41d4-a716-446655440000"
        target_model: "gpt-4o-mini"
        weight: 100
        priority: 1
        enabled: true

tokens:
  - id: "token-001"
    name: "开发环境 Token"
    token: "sk-dev-xxxxxxxxxxxx"
    enabled: true
    expires_at: null

  - id: "token-002"
    name: "生产环境 Token"
    token: "sk-prod-yyyyyyyyy"
    enabled: true
    expires_at: "2026-01-01T00:00:00Z"
```

**JSON 格式支持**:
同样的数据结构也支持 JSON 格式导出。

#### 3.7.2 核心接口设计

```go
package config

type ExportOptions struct {
    Format          string // "yaml" | "json"
    IncludeTokens   bool   // 是否包含 API Tokens
    EncryptAPIKeys  bool   // 是否加密供应商 API Key
    Pretty          bool   // 格式化输出
}

type ImportOptions struct {
    Mode            string // "merge" | "overwrite"
    SkipValidation  bool   // 跳过格式验证
    DryRun          bool   // 仅验证，不实际导入
}

type ConfigService interface {
    // 导出配置
    Export(opts ExportOptions) ([]byte, error)
    ExportToFile(filepath string, opts ExportOptions) error

    // 导入配置
    Import(data []byte, opts ImportOptions) (*ImportResult, error)
    ImportFromFile(filepath string, opts ImportOptions) (*ImportResult, error)

    // 验证配置
    Validate(data []byte) error
}

type ImportResult struct {
    ProvidersCreated  int
    ProvidersUpdated  int
    ModelsCreated     int
    ModelsUpdated     int
    MappingsCreated   int
    TokensCreated     int
    Errors            []string
}
```

#### 3.7.3 导出实现

```go
func (s *ConfigServiceImpl) Export(opts ExportOptions) ([]byte, error) {
    config := &ConfigFile{
        Version:    "1.0",
        ExportedAt: time.Now(),
    }

    // 1. 导出供应商
    providers, err := s.providerService.List()
    if err != nil {
        return nil, err
    }
    for _, p := range providers {
        apiKey := p.APIKey
        if opts.EncryptAPIKeys {
            apiKey = "encrypted:" + apiKey
        } else {
            apiKey, _ = s.crypto.Decrypt(p.APIKey)
        }
        config.Providers = append(config.Providers, ProviderConfig{
            ID:       p.ID,
            Name:     p.Name,
            BaseURL:  p.BaseURL,
            APIKey:   apiKey,
            Enabled:  p.Enabled,
            Priority: p.Priority,
        })
    }

    // 2. 导出统一模型和映射
    models, err := s.mappingService.ListUnifiedModels()
    if err != nil {
        return nil, err
    }
    for _, m := range models {
        mappings, _ := s.mappingService.GetMappingsByUnifiedModel(m.ID)

        modelConfig := UnifiedModelConfig{
            ID:          m.ID,
            Name:        m.Name,
            Description: m.Description,
        }

        for _, mapping := range mappings {
            modelConfig.Mappings = append(modelConfig.Mappings, MappingConfig{
                ProviderID:  mapping.ProviderID,
                TargetModel: mapping.TargetModel,
                Weight:      mapping.Weight,
                Priority:    mapping.Priority,
                Enabled:     mapping.Enabled,
            })
        }

        config.UnifiedModels = append(config.UnifiedModels, modelConfig)
    }

    // 3. 导出 Tokens（可选）
    if opts.IncludeTokens {
        tokens, err := s.tokenService.List()
        if err != nil {
            return nil, err
        }
        for _, t := range tokens {
            config.Tokens = append(config.Tokens, TokenConfig{
                ID:        t.ID,
                Name:      t.Name,
                Token:     t.Token,
                Enabled:   t.Enabled,
                ExpiresAt: t.ExpiresAt,
            })
        }
    }

    // 4. 序列化
    var data []byte
    switch opts.Format {
    case "json":
        if opts.Pretty {
            data, err = json.MarshalIndent(config, "", "  ")
        } else {
            data, err = json.Marshal(config)
        }
    case "yaml":
        data, err = yaml.Marshal(config)
    default:
        return nil, fmt.Errorf("unsupported format: %s", opts.Format)
    }

    return data, err
}
```

#### 3.7.4 导入实现

```go
func (s *ConfigServiceImpl) Import(data []byte, opts ImportOptions) (*ImportResult, error) {
    result := &ImportResult{}

    // 1. 解析配置文件
    var config ConfigFile
    if err := yaml.Unmarshal(data, &config); err != nil {
        // 尝试 JSON 格式
        if err := json.Unmarshal(data, &config); err != nil {
            return nil, fmt.Errorf("failed to parse config: %w", err)
        }
    }

    // 2. 验证配置
    if !opts.SkipValidation {
        if err := s.Validate(data); err != nil {
            return nil, err
        }
    }

    // 3. Dry Run 模式：仅验证
    if opts.DryRun {
        log.Info("Dry run mode: configuration is valid")
        return result, nil
    }

    // 4. 导入供应商
    for _, pc := range config.Providers {
        existing, _ := s.providerService.GetByID(pc.ID)

        // 解密 API Key
        apiKey := pc.APIKey
        if strings.HasPrefix(apiKey, "encrypted:") {
            apiKey = strings.TrimPrefix(apiKey, "encrypted:")
        } else {
            apiKey, _ = s.crypto.Encrypt(apiKey)
        }

        provider := &Provider{
            ID:       pc.ID,
            Name:     pc.Name,
            BaseURL:  pc.BaseURL,
            APIKey:   apiKey,
            Enabled:  pc.Enabled,
            Priority: pc.Priority,
        }

        if existing == nil {
            err := s.providerService.Create(provider)
            if err != nil {
                result.Errors = append(result.Errors, fmt.Sprintf("Provider %s: %v", pc.Name, err))
            } else {
                result.ProvidersCreated++
            }
        } else if opts.Mode == "merge" || opts.Mode == "overwrite" {
            err := s.providerService.Update(pc.ID, provider)
            if err != nil {
                result.Errors = append(result.Errors, fmt.Sprintf("Provider %s: %v", pc.Name, err))
            } else {
                result.ProvidersUpdated++
            }
        }
    }

    // 5. 导入统一模型
    for _, mc := range config.UnifiedModels {
        existing, _ := s.mappingService.GetUnifiedModel(mc.ID)

        model := &UnifiedModel{
            ID:          mc.ID,
            Name:        mc.Name,
            Description: mc.Description,
        }

        if existing == nil {
            err := s.mappingService.CreateUnifiedModel(model)
            if err != nil {
                result.Errors = append(result.Errors, fmt.Sprintf("Model %s: %v", mc.Name, err))
                continue
            }
            result.ModelsCreated++
        } else if opts.Mode == "overwrite" {
            err := s.mappingService.UpdateUnifiedModel(mc.ID, model)
            if err != nil {
                result.Errors = append(result.Errors, fmt.Sprintf("Model %s: %v", mc.Name, err))
                continue
            }
            result.ModelsUpdated++
        }

        // 6. 导入映射
        for _, mappingConfig := range mc.Mappings {
            mapping := &ModelMapping{
                ID:             uuid.New().String(),
                UnifiedModelID: mc.ID,
                ProviderID:     mappingConfig.ProviderID,
                TargetModel:    mappingConfig.TargetModel,
                Weight:         mappingConfig.Weight,
                Priority:       mappingConfig.Priority,
                Enabled:        mappingConfig.Enabled,
            }

            err := s.mappingService.AddMapping(mapping)
            if err != nil {
                result.Errors = append(result.Errors, fmt.Sprintf("Mapping for %s: %v", mc.Name, err))
            } else {
                result.MappingsCreated++
            }
        }
    }

    // 7. 导入 Tokens
    for _, tc := range config.Tokens {
        existing, _ := s.tokenService.GetByID(tc.ID)

        token := &Token{
            ID:        tc.ID,
            Name:      tc.Name,
            Token:     tc.Token,
            Enabled:   tc.Enabled,
            ExpiresAt: tc.ExpiresAt,
        }

        if existing == nil && opts.Mode != "overwrite" {
            // 仅在 merge 模式下创建新 Token
            _, err := s.tokenService.Create(token.Name)
            if err != nil {
                result.Errors = append(result.Errors, fmt.Sprintf("Token %s: %v", tc.Name, err))
            } else {
                result.TokensCreated++
            }
        }
    }

    return result, nil
}
```

#### 3.7.5 配置验证

```go
func (s *ConfigServiceImpl) Validate(data []byte) error {
    var config ConfigFile

    // 解析验证
    if err := yaml.Unmarshal(data, &config); err != nil {
        if err := json.Unmarshal(data, &config); err != nil {
            return fmt.Errorf("invalid format: %w", err)
        }
    }

    // 版本验证
    if config.Version != "1.0" {
        return fmt.Errorf("unsupported version: %s", config.Version)
    }

    // 供应商验证
    providerIDs := make(map[string]bool)
    for _, p := range config.Providers {
        if p.ID == "" || p.Name == "" || p.BaseURL == "" {
            return fmt.Errorf("invalid provider: missing required fields")
        }
        if providerIDs[p.ID] {
            return fmt.Errorf("duplicate provider ID: %s", p.ID)
        }
        providerIDs[p.ID] = true
    }

    // 模型和映射验证
    for _, m := range config.UnifiedModels {
        if m.ID == "" || m.Name == "" {
            return fmt.Errorf("invalid model: missing required fields")
        }

        for _, mapping := range m.Mappings {
            if !providerIDs[mapping.ProviderID] {
                return fmt.Errorf("model %s references unknown provider: %s", m.Name, mapping.ProviderID)
            }
            if mapping.TargetModel == "" {
                return fmt.Errorf("model %s has empty target_model", m.Name)
            }
        }
    }

    return nil
}
```

#### 3.7.6 API 端点

```
# 配置导出
GET  /api/config/export?format=yaml&include_tokens=false&encrypt_keys=true
POST /api/config/export  # Body: ExportOptions, Response: 配置文件下载

# 配置导入
POST /api/config/import?mode=merge&dry_run=false
     # Body: Multipart file upload 或 JSON/YAML content
     # Response: ImportResult

# 配置验证
POST /api/config/validate
     # Body: 配置文件内容
     # Response: {valid: true, errors: []}
```

#### 3.7.7 Web UI 集成

**导出界面**:
- 下拉选择格式（YAML/JSON）
- 复选框：包含 Tokens、加密 API Keys
- 点击"导出"按钮下载文件

**导入界面**:
- 文件上传或文本框粘贴
- 导入模式选择（merge/overwrite）
- Dry Run 预览
- 显示导入结果统计

---

## 4. API 设计

### 4.1 Claude API 端点

**功能**: Claude 原生格式接口（供 Claude Code 使用）

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

**响应**: Claude 格式（流式或非流式）

**处理流程**:
1. 验证 Token (`Authorization: Bearer sk-xxx`)
2. 解析请求 → `ClaudeRequest`
3. 查找模型映射 → `[]*ModelMapping`
4. 负载均衡选择供应商
5. 转换请求 → `OpenAIRequest`
6. 发送到供应商
7. 转换响应 → `ClaudeResponse`
8. 返回给客户端

---

#### `POST /v1/chat/completions`

**功能**: OpenAI 兼容格式接口

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

**处理流程**: 类似 `/v1/messages`，但跳过格式转换步骤

---

### 4.2 管理 API 端点

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

## 5. 目录结构

```
Siriusx-API/
├── cmd/
│   └── server/
│       └── main.go                # 主入口
├── internal/
│   ├── converter/                 # 格式转换引擎
│   │   ├── claude_to_openai.go
│   │   ├── openai_to_claude.go
│   │   ├── stream_converter.go
│   │   └── types.go
│   ├── provider/                  # 供应商管理
│   │   ├── service.go
│   │   ├── repository.go
│   │   └── health_checker.go
│   ├── mapping/                   # 模型映射
│   │   ├── service.go
│   │   ├── repository.go
│   │   └── router.go
│   ├── balancer/                  # 负载均衡
│   │   ├── weighted_balancer.go
│   │   └── failover.go
│   ├── token/                     # 令牌管理
│   │   ├── service.go
│   │   └── repository.go
│   ├── claude/                    # ClaudeCode 配置
│   │   └── service.go
│   ├── api/                       # API 路由
│   │   ├── router.go
│   │   ├── handlers/
│   │   │   ├── claude_handler.go
│   │   │   ├── provider_handler.go
│   │   │   ├── model_handler.go
│   │   │   └── token_handler.go
│   │   └── middleware/
│   │       ├── auth.go
│   │       └── cors.go
│   ├── config/                    # 配置管理
│   │   └── config.go
│   └── models/                    # 数据模型
│       ├── provider.go
│       ├── model.go
│       ├── mapping.go
│       └── token.go
├── web/                           # Astro 前端
│   ├── src/
│   │   ├── pages/
│   │   │   ├── index.astro       # Dashboard
│   │   │   ├── providers.astro   # 供应商管理
│   │   │   ├── models.astro      # 模型管理
│   │   │   ├── tokens.astro      # 令牌管理
│   │   │   └── claude.astro      # ClaudeCode 配置
│   │   ├── components/
│   │   │   ├── ProviderCard.tsx
│   │   │   ├── ModelMapping.tsx
│   │   │   └── StatChart.tsx
│   │   └── layouts/
│   │       └── Layout.astro
│   ├── public/
│   └── astro.config.mjs
├── config/
│   ├── config.example.yaml       # 配置示例
│   └── default_mappings.yaml     # 默认模型映射
├── docs/
│   ├── format-conversion-study.md
│   ├── architecture-design.md
│   ├── api-reference.md          # 手写 API 参考文档
│   └── swagger/                  # Swagger 自动生成的文档
│       ├── docs.go
│       ├── swagger.json
│       └── swagger.yaml
├── Makefile                      # 包含 swagger-gen 等命令
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

---

## 6. 前端设计

### 6.1 技术栈与架构

**前端框架**: Astro 4.x (静态优先、极致轻量)

**UI 框架**:
- **组件库**: shadcn/ui (基于 Radix UI + Tailwind CSS)
- **React**: 用于交互复杂的动态组件
- **Tailwind CSS**: 快速样式开发
- **图表**: ECharts (监控面板和统计图表)

**状态管理**: Zustand (轻量级、TypeScript 友好)

**路由**: Astro 内置文件系统路由

**HTTP 客户端**: Axios (统一的 API 调用)

### 6.2 Web UI 路由设计

#### 6.2.1 路由表

| 路由路径 | 页面组件 | 功能描述 | 权限要求 |
|---------|---------|---------|---------|
| `/` | Dashboard | 仪表板首页,展示系统概览 | 需登录 |
| `/login` | Login | 用户登录页面 | 公开访问 |
| `/providers` | ProviderList | Provider 管理列表页 | 需登录 |
| `/providers/new` | ProviderForm | 新建 Provider | 需登录 |
| `/providers/:id` | ProviderDetail | Provider 详情页 | 需登录 |
| `/providers/:id/edit` | ProviderForm | 编辑 Provider | 需登录 |
| `/models` | ModelList | 模型映射列表页 | 需登录 |
| `/models/new` | ModelForm | 新建模型映射 | 需登录 |
| `/models/:id` | ModelDetail | 模型映射详情页 | 需登录 |
| `/models/:id/edit` | ModelForm | 编辑模型映射 | 需登录 |
| `/tokens` | TokenList | Token 管理列表页 | 需登录 |
| `/tokens/new` | TokenForm | 新建 Token | 需登录 |
| `/tokens/:id` | TokenDetail | Token 详情页 | 需登录 |
| `/logs` | LogViewer | 日志查看器 | 需登录 |
| `/settings` | Settings | 系统设置页面 | 需登录 |

#### 6.2.2 页面设计规范

##### Dashboard (仪表板)
**功能描述**:系统概览页面,展示关键指标和统计信息

**主要内容**:
- 统计卡片区域:
  - Provider 总数及状态分布
  - 模型映射总数
  - Token 总数及使用统计
  - 近期请求统计(成功率、失败率)
- 图表区域:
  - 请求量趋势图(最近 7 天)
  - Provider 可用性状态图
  - 热门模型使用排行
- 快捷操作区域:
  - 新建 Provider 按钮
  - 新建模型映射按钮
  - 新建 Token 按钮

**交互设计**:
- 统计卡片支持点击跳转到对应列表页
- 图表支持日期范围选择
- 实时刷新(轮询或 WebSocket)

##### ProviderList (Provider 列表)
**功能描述**:展示所有 OpenAI 兼容 Provider 配置

**主要内容**:
- 搜索和筛选区:
  - 按名称搜索
  - 按状态筛选(启用/禁用)
  - 按健康状态筛选(健康/不健康)
- 数据表格:
  - 列:名称、Base URL、API Key(脱敏)、优先级、健康状态、操作
  - 支持排序(按优先级、创建时间)
  - 批量操作(启用/禁用、删除)
- 分页组件

**交互设计**:
- 表格行支持点击查看详情
- 状态切换(启用/禁用)支持一键操作
- 删除操作需二次确认
- "新建 Provider" 按钮固定在右上角
- 健康状态实时更新

##### ProviderForm (Provider 表单)
**功能描述**:创建或编辑 Provider 配置

**表单字段**:
1. 基本信息:
   - 名称(必填)
   - Base URL(必填,OpenAI 兼容 API 地址)
   - API Key(必填,加密存储)
2. 高级设置:
   - 优先级(数字,默认 50)
   - 启用状态(开关)

**表单验证**:
- API Key 格式验证
- URL 格式验证
- 优先级范围验证(1-100)

**交互设计**:
- 实时表单验证
- 保存前测试连接(可选)
- 保存成功后跳转到详情页

##### ModelList (模型映射列表)
**功能描述**:展示所有模型映射配置

**主要内容**:
- 搜索和筛选区:
  - 按统一模型名称搜索
  - 按状态筛选
  - 按 Provider 筛选
- 数据表格:
  - 列:统一模型名、目标模型、Provider、权重、优先级、状态、操作
  - 支持排序
  - 批量操作
- 分页组件

**交互设计**:
- 支持拖拽排序(调整优先级)
- 状态切换一键操作
- 快速复制映射配置

##### TokenList (Token 列表)
**功能描述**:展示所有 API Token

**主要内容**:
- 搜索和筛选区:
  - 按名称搜索
  - 按状态筛选
  - 按到期时间筛选
- 数据表格:
  - 列:名称、Token(部分显示)、创建时间、到期时间、使用次数、状态、操作
  - 支持排序
  - 批量操作(启用/禁用、删除)

**交互设计**:
- Token 显示前 8 位和后 4 位,中间用 *** 代替
- 点击"复制"图标可复制完整 Token
- 删除操作需二次确认
- 显示 Token 使用统计图表

##### LogViewer (日志查看器)
**功能描述**:查看系统运行日志和请求日志

**主要内容**:
- 筛选区:
  - 日志级别筛选(INFO/WARN/ERROR)
  - 时间范围选择
  - 关键词搜索
  - 请求 ID 搜索
- 日志列表:
  - 时间戳
  - 日志级别
  - 来源模块
  - 日志内容
  - 详细信息展开

**交互设计**:
- 实时日志推送(SSE)
- 支持暂停/恢复日志流
- 日志内容高亮显示
- 支持导出日志(CSV/JSON)

##### Settings (系统设置)
**功能描述**:系统全局配置

**主要内容**:
1. 基本设置:
   - 系统名称
   - 管理员密码修改
   - 时区设置
2. 负载均衡设置:
   - 策略选择(加权轮询)
   - 重试次数
   - 故障转移开关
3. 安全设置:
   - Token 有效期
   - 速率限制
4. 其他:
   - 日志保留时间
   - 数据备份设置

**交互设计**:
- 分 Tab 显示不同设置类别
- 修改需二次确认
- 保存后显示成功提示

#### 6.2.3 组件设计

##### 通用组件

**Navbar (导航栏)**
- Logo 和系统名称
- 主导航链接(Dashboard/Providers/Models/Tokens/Logs/Settings)
- 用户菜单(右上角):
  - 用户名显示
  - 登出按钮

**Sidebar (侧边栏,可选)**
- 折叠/展开功能
- 导航树结构
- 当前页面高亮

**Card (卡片组件)**
- 标题栏
- 内容区
- 操作按钮区
- 支持 loading 状态

**Table (数据表格)**
- 支持排序
- 支持分页
- 支持行选择
- 支持自定义列渲染
- 空状态展示

**Form (表单组件)**
- 统一的表单验证
- 错误提示样式
- 提交加载状态
- 取消和保存按钮

**Modal (模态框)**
- 确认对话框
- 表单弹窗
- 信息展示弹窗

**Toast (消息提示)**
- 成功/警告/错误/信息 四种类型
- 自动消失
- 可手动关闭

**StatusBadge (状态标签)**
- 不同状态对应不同颜色
- 启用(绿色)、禁用(灰色)、错误(红色)

##### 业务组件

**ProviderCard**
- 展示 Provider 基本信息
- 显示健康状态指示器
- 快捷操作按钮(编辑/删除/测试连接)

**ModelMappingCard**
- 展示模型映射关系
- 显示关联的 Provider
- 快捷切换启用状态

**TokenCard**
- 展示 Token 信息
- 显示使用统计
- 快捷复制功能

**RequestChart**
- 请求量趋势图表
- 成功率统计
- 支持时间范围切换

**LogEntry**
- 日志条目展示
- 支持展开详情
- 高亮关键信息

#### 6.2.4 状态管理

##### 全局状态(Zustand)

**authStore**
```typescript
interface AuthState {
  isAuthenticated: boolean;
  user: User | null;
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
}
```

**providerStore**
```typescript
interface ProviderState {
  providers: Provider[];
  loading: boolean;
  fetchProviders: () => Promise<void>;
  createProvider: (data: CreateProviderDTO) => Promise<void>;
  updateProvider: (id: string, data: UpdateProviderDTO) => Promise<void>;
  deleteProvider: (id: string) => Promise<void>;
  toggleProvider: (id: string, enabled: boolean) => Promise<void>;
}
```

**modelStore**
```typescript
interface ModelState {
  models: ModelMapping[];
  loading: boolean;
  fetchModels: () => Promise<void>;
  createModel: (data: CreateModelDTO) => Promise<void>;
  updateModel: (id: string, data: UpdateModelDTO) => Promise<void>;
  deleteModel: (id: string) => Promise<void>;
}
```

**tokenStore**
```typescript
interface TokenState {
  tokens: Token[];
  loading: boolean;
  fetchTokens: () => Promise<void>;
  createToken: (data: CreateTokenDTO) => Promise<Token>;
  deleteToken: (id: string) => Promise<void>;
}
```

**logStore**
```typescript
interface LogState {
  logs: LogEntry[];
  filters: LogFilters;
  streaming: boolean;
  setFilters: (filters: LogFilters) => void;
  startStreaming: () => void;
  stopStreaming: () => void;
  exportLogs: (format: 'csv' | 'json') => void;
}
```

##### 页面级状态(React Hooks)

使用 `useState`、`useEffect` 等 Hook 管理页面临时状态:
- 表单输入值
- 模态框显示状态
- 加载状态
- 错误信息

#### 6.2.5 API 交互

##### API Client 配置

**基础配置**
```typescript
const apiClient = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器:添加 Token
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('auth_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// 响应拦截器:处理错误
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // 跳转到登录页
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);
```

##### API 端点定义

**Provider API**
- `GET /api/providers` - 获取 Provider 列表
- `POST /api/providers` - 创建 Provider
- `GET /api/providers/:id` - 获取 Provider 详情
- `PUT /api/providers/:id` - 更新 Provider
- `DELETE /api/providers/:id` - 删除 Provider
- `POST /api/providers/:id/health-check` - 测试 Provider 连接

**Model API**
- `GET /api/models` - 获取模型映射列表
- `POST /api/models` - 创建模型映射
- `GET /api/models/:id` - 获取模型映射详情
- `PUT /api/models/:id` - 更新模型映射
- `DELETE /api/models/:id` - 删除模型映射

**Token API**
- `GET /api/tokens` - 获取 Token 列表
- `POST /api/tokens` - 创建 Token
- `GET /api/tokens/:id` - 获取 Token 详情
- `DELETE /api/tokens/:id` - 删除 Token

**Log API**
- `GET /api/logs` - 获取日志列表(分页)
- `GET /api/logs/stream` - 实时日志流(SSE)
- `GET /api/logs/export` - 导出日志

**Stats API**
- `GET /api/stats/dashboard` - 获取仪表板统计数据
- `GET /api/stats/requests` - 获取请求统计数据
- `GET /api/stats/providers` - 获取 Provider 统计数据

#### 6.2.6 路由守卫

**认证守卫**
```typescript
// src/components/ProtectedRoute.tsx
export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuthStore();

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}
```

**路由配置**
```typescript
// src/App.tsx
function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/" element={<ProtectedRoute><Layout /></ProtectedRoute>}>
          <Route index element={<Dashboard />} />
          <Route path="providers" element={<ProviderList />} />
          <Route path="providers/new" element={<ProviderForm />} />
          <Route path="providers/:id" element={<ProviderDetail />} />
          <Route path="providers/:id/edit" element={<ProviderForm />} />
          <Route path="models" element={<ModelList />} />
          <Route path="models/new" element={<ModelForm />} />
          <Route path="models/:id" element={<ModelDetail />} />
          <Route path="models/:id/edit" element={<ModelForm />} />
          <Route path="tokens" element={<TokenList />} />
          <Route path="tokens/new" element={<TokenForm />} />
          <Route path="tokens/:id" element={<TokenDetail />} />
          <Route path="logs" element={<LogViewer />} />
          <Route path="settings" element={<Settings />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
```

#### 6.2.7 性能优化

**代码分割**
- 使用 `React.lazy()` 和 `Suspense` 实现页面级代码分割
- 减小初始加载包体积

**列表虚拟化**
- 使用 `react-window` 或 `react-virtual` 优化长列表渲染
- 适用于日志查看器和大量数据表格

**请求优化**
- 使用 `SWR` 或 `React Query` 实现数据缓存和自动重新验证
- 减少不必要的 API 请求

**图片和资源优化**
- 图片懒加载
- 使用 WebP 格式
- 图标使用 SVG

**防抖和节流**
- 搜索输入使用防抖
- 滚动事件使用节流

#### 6.2.8 错误处理

**全局错误边界**
```typescript
// src/components/ErrorBoundary.tsx
class ErrorBoundary extends React.Component {
  state = { hasError: false, error: null };

  static getDerivedStateFromError(error) {
    return { hasError: true, error };
  }

  componentDidCatch(error, errorInfo) {
    console.error('Error caught by boundary:', error, errorInfo);
    // 可以上报到错误监控服务
  }

  render() {
    if (this.state.hasError) {
      return <ErrorFallback error={this.state.error} />;
    }
    return this.props.children;
  }
}
```

**API 错误处理**
- 统一错误响应格式
- 友好的错误提示
- 支持错误重试

**表单验证错误**
- 实时验证
- 字段级错误提示
- 阻止无效提交

#### 6.2.9 国际化(可选)

如需支持多语言,建议使用 `i18next` 和 `react-i18next`:

```typescript
// src/i18n/zh-CN.ts
export default {
  common: {
    save: '保存',
    cancel: '取消',
    delete: '删除',
    edit: '编辑',
    create: '创建',
  },
  provider: {
    title: 'Provider 管理',
    create: '新建 Provider',
    edit: '编辑 Provider',
    list: 'Provider 列表',
  },
  // ...
};
```

#### 6.2.10 测试策略

**单元测试**
- 使用 `Vitest` 测试工具函数和自定义 Hook
- 测试覆盖率目标:80%

**组件测试**
- 使用 `React Testing Library` 测试组件渲染和交互
- 测试关键用户操作流程

**E2E 测试**
- 使用 `Playwright` 或 `Cypress` 测试完整用户流程
- 覆盖关键业务场景(登录、创建 Provider、发起请求等)

**测试示例**
```typescript
// src/components/ProviderCard.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { ProviderCard } from './ProviderCard';

test('renders provider card with correct data', () => {
  const provider = {
    id: '1',
    name: 'Test Provider',
    status: 'enabled',
  };

  render(<ProviderCard provider={provider} />);

  expect(screen.getByText('Test Provider')).toBeInTheDocument();
  expect(screen.getByText('启用')).toBeInTheDocument();
});

test('calls onDelete when delete button is clicked', () => {
  const handleDelete = vi.fn();
  const provider = { id: '1', name: 'Test Provider' };

  render(<ProviderCard provider={provider} onDelete={handleDelete} />);

  fireEvent.click(screen.getByText('删除'));

  expect(handleDelete).toHaveBeenCalledWith('1');
});
```

---

## 7. 部署架构

### 7.1 Docker Compose

```yaml
version: '3.8'

services:
  siriusx-api:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data          # SQLite 数据库
      - ./config:/app/config      # 配置文件
    environment:
      - GIN_MODE=release
      - DATABASE_PATH=/app/data/siriusx.db
      - CONFIG_PATH=/app/config/config.yaml
      - ENCRYPTION_KEY=${ENCRYPTION_KEY}
    restart: unless-stopped
```

### 7.2 Dockerfile

```dockerfile
# 构建阶段
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o siriusx-api ./cmd/server

# 前端构建
FROM node:20-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm install
COPY web/ .
RUN npm run build

# 运行阶段
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/siriusx-api .
COPY --from=frontend-builder /app/web/dist ./web/dist
COPY config/config.example.yaml ./config/
RUN apk add --no-cache ca-certificates
EXPOSE 8080
CMD ["./siriusx-api"]
```

---

## 8. 核心流程图

### 8.1 请求处理流程

```
┌──────────────┐
│ Claude Code  │
│  发送请求    │
└──────┬───────┘
       │ POST /v1/messages
       │ Authorization: Bearer sk-xxx
       ↓
┌──────────────────────────────────────┐
│ 1. Token 验证中间件                   │
│    - 验证 Token 有效性                │
│    - 检查是否过期                     │
└──────┬───────────────────────────────┘
       ↓
┌──────────────────────────────────────┐
│ 2. 解析 Claude 请求                   │
│    - model: "claude-sonnet-4"         │
│    - messages: [...]                  │
└──────┬───────────────────────────────┘
       ↓
┌──────────────────────────────────────┐
│ 3. 查找模型映射                       │
│    - 查询 unified_models              │
│    - 获取 model_mappings (按优先级)   │
│    - 过滤启用且健康的映射             │
└──────┬───────────────────────────────┘
       ↓
┌──────────────────────────────────────┐
│ 4. 负载均衡选择供应商                 │
│    - 按权重随机选择                   │
│    - 获取供应商配置                   │
└──────┬───────────────────────────────┘
       ↓
┌──────────────────────────────────────┐
│ 5. 格式转换 (Claude → OpenAI)         │
│    - 转换 system prompt               │
│    - 转换 messages                    │
│    - 转换 tools                       │
└──────┬───────────────────────────────┘
       ↓
┌──────────────────────────────────────┐
│ 6. 发送到供应商                       │
│    POST {baseUrl}/v1/chat/completions │
│    Authorization: Bearer {apiKey}     │
└──────┬───────────────────────────────┘
       │
       ├─ 成功 ──────────────────────┐
       │                              │
       ├─ 失败 (超时/5xx/429)        │
       │      ↓                       │
       │ ┌────────────────────┐      │
       │ │ 7. 故障转移        │      │
       │ │  - 尝试下一个供应商│      │
       │ └────────────────────┘      │
       │                              │
       └──────────────────────────────┘
                   ↓
┌──────────────────────────────────────┐
│ 8. 格式转换 (OpenAI → Claude)         │
│    - 转换 content                     │
│    - 转换 tool_calls                  │
│    - 转换 stop_reason                 │
└──────┬───────────────────────────────┘
       ↓
┌──────────────────────────────────────┐
│ 9. 返回 Claude 响应                   │
│    - 流式: SSE 格式                   │
│    - 非流式: JSON 格式                │
└──────────────────────────────────────┘
```

---

## 9. 性能优化

### 9.1 HTTP 客户端连接池

```go
var httpClient = &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
}
```

### 9.2 配置缓存

```go
type ConfigCache struct {
    mu       sync.RWMutex
    mappings map[string][]*ModelMapping
    ttl      time.Duration
    lastLoad time.Time
}

func (c *ConfigCache) Get(unifiedModel string) ([]*ModelMapping, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    if time.Since(c.lastLoad) > c.ttl {
        return nil, false
    }

    mappings, ok := c.mappings[unifiedModel]
    return mappings, ok
}
```

### 9.3 流式响应优化

- 使用 `io.Pipe` 实现零拷贝
- 增量解析 SSE 事件
- 及时刷新缓冲区

---

## 10. API 文档生成方案

### 10.1 Swagger/OpenAPI 集成

**工具**: [swaggo/swag](https://github.com/swaggo/swag)

**安装**:
```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

**集成方式**:

#### 1. 主程序注释

在 `cmd/server/main.go` 中添加 Swagger 元数据：

```go
// @title           Siriusx-API
// @version         2.0
// @description     轻量级 AI 模型聚合网关 - 支持多供应商、负载均衡、故障转移
// @termsOfService  https://github.com/yourusername/siriusx-api

// @contact.name   API Support
// @contact.url    https://github.com/yourusername/siriusx-api/issues
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 使用 Bearer Token 进行认证，格式: "Bearer sk-xxx"

package main

import (
    "github.com/gin-gonic/gin"
    swaggerFiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"
    _ "github.com/yourusername/siriusx-api/docs" // Swagger 生成的文档
)

func main() {
    r := gin.Default()

    // Swagger UI 端点
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

    // ... 其他路由
}
```

#### 2. Handler 注释示例

**Claude Messages API**:
```go
// CreateMessage godoc
// @Summary      发送 Claude 消息
// @Description  接收 Claude Messages API 格式请求，转换后路由到配置的供应商
// @Tags         Claude API
// @Accept       json
// @Produce      json
// @Param        request body ClaudeRequest true "Claude 请求体"
// @Success      200 {object} ClaudeResponse "Claude 响应"
// @Failure      400 {object} ErrorResponse "请求错误"
// @Failure      401 {object} ErrorResponse "认证失败"
// @Failure      500 {object} ErrorResponse "服务器错误"
// @Security     BearerAuth
// @Router       /v1/messages [post]
func (h *ClaudeHandler) CreateMessage(c *gin.Context) {
    // 实现...
}
```

**供应商管理 API**:
```go
// ListProviders godoc
// @Summary      列出所有供应商
// @Description  获取所有配置的 AI 服务供应商列表
// @Tags         Provider Management
// @Produce      json
// @Success      200 {array} Provider "供应商列表"
// @Failure      500 {object} ErrorResponse "服务器错误"
// @Security     BearerAuth
// @Router       /api/providers [get]
func (h *ProviderHandler) ListProviders(c *gin.Context) {
    // 实现...
}

// CreateProvider godoc
// @Summary      创建供应商
// @Description  添加新的 AI 服务供应商配置
// @Tags         Provider Management
// @Accept       json
// @Produce      json
// @Param        request body CreateProviderRequest true "供应商配置"
// @Success      201 {object} Provider "创建的供应商"
// @Failure      400 {object} ErrorResponse "请求错误"
// @Failure      500 {object} ErrorResponse "服务器错误"
// @Security     BearerAuth
// @Router       /api/providers [post]
func (h *ProviderHandler) CreateProvider(c *gin.Context) {
    // 实现...
}
```

**模型管理 API**:
```go
// ListModels godoc
// @Summary      列出所有统一模型
// @Description  获取所有用户定义的统一模型名称
// @Tags         Model Management
// @Produce      json
// @Success      200 {array} UnifiedModel "统一模型列表"
// @Security     BearerAuth
// @Router       /api/models [get]
func (h *ModelHandler) ListModels(c *gin.Context) {
    // 实现...
}
```

#### 3. 数据模型定义

在 `internal/models/` 中定义 Swagger 模型：

```go
package models

// Provider 供应商配置
type Provider struct {
    ID           string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
    Name         string    `json:"name" example:"我的 OneAPI"`
    BaseURL      string    `json:"base_url" example:"https://api.oneapi.com"`
    Enabled      bool      `json:"enabled" example:"true"`
    Priority     int       `json:"priority" example:"50"`
    HealthStatus string    `json:"health_status" example:"healthy" enums:"healthy,unhealthy,unknown"`
    LastCheck    time.Time `json:"last_check,omitempty"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
} // @name Provider

// ErrorResponse 错误响应
type ErrorResponse struct {
    Error ErrorDetail `json:"error"`
} // @name ErrorResponse

// ErrorDetail 错误详情
type ErrorDetail struct {
    Type    string `json:"type" example:"invalid_request_error"`
    Message string `json:"message" example:"Invalid model name"`
} // @name ErrorDetail
```

#### 4. 生成文档

**生成命令**:
```bash
# 在项目根目录执行
swag init -g cmd/server/main.go -o docs/swagger

# 或添加到 Makefile
make swagger-gen
```

**生成文件**:
```
docs/swagger/
├── docs.go          # Go 代码
├── swagger.json     # OpenAPI 3.0 JSON
└── swagger.yaml     # OpenAPI 3.0 YAML
```

#### 5. 访问 Swagger UI

**本地开发**:
```
http://localhost:8080/swagger/index.html
```

**Docker 部署**:
- Swagger 静态文件会包含在 Docker 镜像中
- 生产环境也可通过 `/swagger/index.html` 访问

### 10.2 API 文档结构

**分组标签**:
- `Claude API` - Claude Messages API 端点
- `OpenAI API` - OpenAI Chat Completions API 端点
- `Provider Management` - 供应商管理
- `Model Management` - 模型管理
- `Token Management` - 令牌管理
- `Monitoring` - 监控和健康检查

**认证说明**:
所有管理 API 都需要在请求头中包含 Bearer Token：
```
Authorization: Bearer sk-your-token-here
```

### 10.3 文档维护流程

**开发流程**:
1. 在 Handler 中添加 Swagger 注释
2. 运行 `swag init` 生成文档
3. 访问 Swagger UI 验证
4. 提交代码（包含 `docs/swagger/` 目录）

**CI/CD 集成**:
```yaml
# .github/workflows/api-docs.yml
name: Generate API Docs

on:
  push:
    branches: [ main ]

jobs:
  docs:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Install swag
        run: go install github.com/swaggo/swag/cmd/swag@latest
      - name: Generate docs
        run: swag init -g cmd/server/main.go -o docs/swagger
      - name: Commit docs
        run: |
          git config user.name "GitHub Actions"
          git add docs/swagger/
          git commit -m "docs: update API documentation" || true
          git push
```

### 10.4 额外文档资源

**人工文档**:
- `docs/api-reference.md` - 手写的 API 参考文档（补充 Swagger 未覆盖的内容）
- 包含使用示例、最佳实践、故障排查

**示例请求**:
在 Swagger UI 中提供可执行的示例请求，方便测试。

---

## 11. 测试策略

### 11.1 测试目标

**覆盖率目标**:
- 单元测试：核心业务逻辑覆盖率 ≥ 80%
- 集成测试：关键流程覆盖率 ≥ 70%
- E2E 测试：主要用户场景 100% 覆盖

**测试原则**:
- 测试先行（TDD）：关键模块先写测试
- 快速反馈：单元测试运行时间 < 5s
- 隔离性：每个测试独立运行，不依赖执行顺序

### 11.2 单元测试

**测试范围**: 核心业务逻辑，不依赖外部服务

**测试工具**:
- Go 标准库 `testing`
- 断言库：`github.com/stretchr/testify/assert`
- Mock 工具：`github.com/stretchr/testify/mock`

**关键测试用例**:

#### 1. 格式转换引擎测试
```go
// internal/converter/converter_test.go
func TestConvertClaudeToOpenAI(t *testing.T) {
    tests := []struct {
        name     string
        input    *ClaudeRequest
        expected *OpenAIRequest
    }{
        {
            name: "基本文本消息转换",
            input: &ClaudeRequest{
                Model: "claude-sonnet-4",
                Messages: []Message{
                    {Role: "user", Content: "Hello"},
                },
                MaxTokens: 1024,
            },
            expected: &OpenAIRequest{
                Model: "gpt-4o",
                Messages: []OpenAIMessage{
                    {Role: "user", Content: "Hello"},
                },
                MaxTokens: 1024,
            },
        },
        {
            name: "System prompt 转换",
            input: &ClaudeRequest{
                System: "You are a helpful assistant",
                Messages: []Message{
                    {Role: "user", Content: "Hi"},
                },
            },
            expected: &OpenAIRequest{
                Messages: []OpenAIMessage{
                    {Role: "system", Content: "You are a helpful assistant"},
                    {Role: "user", Content: "Hi"},
                },
            },
        },
        // ... 更多用例
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ConvertClaudeToOpenAI(tt.input, "gpt-4o")
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func TestStreamConverter(t *testing.T) {
    sc := NewStreamConverter("claude-sonnet-4")

    // 测试初始事件生成
    events := sc.GenerateInitialEvents()
    assert.Len(t, events, 2) // message_start + content_block_start

    // 测试 chunk 处理
    chunk := &OpenAIChunk{
        Choices: []struct{
            Delta struct{ Content string }
        }{
            {Delta: struct{ Content string }{Content: "Hello"}},
        },
    }
    events = sc.ProcessOpenAIChunk(chunk)
    assert.Len(t, events, 1)
    assert.Equal(t, "content_block_delta", events[0].Event)
}
```

#### 2. 负载均衡器测试
```go
// internal/balancer/balancer_test.go
func TestWeightedLoadBalancer_SelectProvider(t *testing.T) {
    mappings := []*ModelMapping{
        {ID: "1", Weight: 70, ProviderID: "provider-1"},
        {ID: "2", Weight: 30, ProviderID: "provider-2"},
    }

    lb := NewWeightedLoadBalancer(nil)

    // 运行 1000 次，验证权重分布
    counts := map[string]int{}
    for i := 0; i < 1000; i++ {
        mapping, err := lb.SelectProvider(mappings)
        assert.NoError(t, err)
        counts[mapping.ID]++
    }

    // 验证权重比例（允许 ±10% 误差）
    assert.InDelta(t, 700, counts["1"], 100)
    assert.InDelta(t, 300, counts["2"], 100)
}

func TestExecuteWithFailover(t *testing.T) {
    mockProvider := new(MockProviderService)
    mockProvider.On("GetByID", "provider-1").Return(&Provider{
        ID: "provider-1",
        BaseURL: "https://failing.example.com",
    }, nil)
    mockProvider.On("GetByID", "provider-2").Return(&Provider{
        ID: "provider-2",
        BaseURL: "https://working.example.com",
    }, nil)

    lb := NewWeightedLoadBalancer(mockProvider)

    // 测试故障转移：第一个供应商失败，自动尝试第二个
    mappings := []*ModelMapping{
        {Priority: 1, ProviderID: "provider-1"},
        {Priority: 2, ProviderID: "provider-2"},
    }

    resp, err := lb.ExecuteWithFailover(context.Background(), &ClaudeRequest{}, mappings)
    assert.NoError(t, err)
    assert.NotNil(t, resp)
}
```

#### 3. 供应商健康检查测试
```go
// internal/provider/health_test.go
func TestHealthCheck(t *testing.T) {
    // Mock HTTP 服务器
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
        w.Write([]byte(`{"data":[]}`))
    }))
    defer server.Close()

    service := NewProviderService(db)
    provider := &Provider{
        ID: "test-provider",
        BaseURL: server.URL,
    }

    err := service.HealthCheck(provider.ID)
    assert.NoError(t, err)

    // 验证健康状态已更新
    updated, _ := service.GetByID(provider.ID)
    assert.Equal(t, "healthy", updated.HealthStatus)
}
```

### 11.3 集成测试

**测试范围**: 多个模块协作，使用真实数据库

**测试工具**:
- SQLite 内存数据库（`:memory:`）
- `github.com/gin-gonic/gin` 测试模式

**关键测试用例**:

#### 1. 完整请求流程测试
```go
// internal/api/integration_test.go
func TestCompleteRequestFlow(t *testing.T) {
    // 1. 初始化测试数据库
    db := setupTestDB()
    defer db.Close()

    // 2. 创建测试数据
    provider := &Provider{
        ID: "test-provider",
        BaseURL: mockServerURL,
        Enabled: true,
    }
    providerService.Create(provider)

    unifiedModel := &UnifiedModel{Name: "test-model"}
    mappingService.CreateUnifiedModel(unifiedModel)

    mapping := &ModelMapping{
        UnifiedModelID: unifiedModel.ID,
        ProviderID: provider.ID,
        TargetModel: "gpt-4o",
        Weight: 100,
    }
    mappingService.AddMapping(mapping)

    // 3. 发送测试请求
    req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(`{
        "model": "test-model",
        "messages": [{"role": "user", "content": "Hello"}],
        "max_tokens": 100
    }`))
    req.Header.Set("Authorization", "Bearer test-token")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    // 4. 验证响应
    assert.Equal(t, 200, w.Code)

    var resp ClaudeResponse
    json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, "test-model", resp.Model)
    assert.NotEmpty(t, resp.Content)
}
```

#### 2. 故障转移集成测试
```go
func TestFailoverIntegration(t *testing.T) {
    // 创建两个供应商：一个失败，一个成功
    failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(500)
    }))
    defer failingServer.Close()

    workingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
        w.Write([]byte(`{"choices":[{"message":{"content":"Success"}}]}`))
    }))
    defer workingServer.Close()

    // ... 设置测试数据

    // 发送请求，验证自动故障转移
    req := httptest.NewRequest("POST", "/v1/messages", ...)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, 200, w.Code)
    // 验证日志中记录了故障转移
}
```

### 11.4 E2E 测试

**测试范围**: 完整用户场景，使用真实服务

**测试工具**:
- Docker Compose 启动完整环境
- Go HTTP 客户端发送真实请求

**测试场景**:

#### 1. Claude Code 集成测试
```bash
#!/bin/bash
# tests/e2e/claude_code_test.sh

# 1. 启动服务
docker-compose up -d
sleep 5

# 2. 配置供应商
curl -X POST http://localhost:8080/api/providers \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "Test Provider",
    "base_url": "https://api.example.com",
    "api_key": "test-key"
  }'

# 3. 配置模型映射
curl -X POST http://localhost:8080/api/models \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"name": "claude-sonnet-4"}'

# 4. 使用 Claude Code 发送请求
export ANTHROPIC_BASE_URL=http://localhost:8080
export ANTHROPIC_API_KEY=$TEST_TOKEN

echo "Test message" | claude

# 5. 验证响应
if [ $? -eq 0 ]; then
  echo "✅ E2E test passed"
else
  echo "❌ E2E test failed"
  exit 1
fi
```

#### 2. 负载均衡验证
```go
// tests/e2e/load_balance_test.go
func TestLoadBalancingE2E(t *testing.T) {
    // 配置两个供应商，权重 70:30
    setupProviders()

    // 发送 100 个请求
    providerHits := map[string]int{}
    for i := 0; i < 100; i++ {
        resp := sendRequest("/v1/messages", testPayload)
        providerID := resp.Header.Get("X-Provider-ID")
        providerHits[providerID]++
    }

    // 验证权重分布
    assert.InDelta(t, 70, providerHits["provider-1"], 15)
    assert.InDelta(t, 30, providerHits["provider-2"], 15)
}
```

### 11.5 测试命令与 CI/CD

**Makefile 命令**:
```makefile
# 运行所有单元测试
test:
	go test ./... -v -cover

# 运行单元测试并生成覆盖率报告
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# 运行集成测试
test-integration:
	go test ./... -tags=integration -v

# 运行 E2E 测试
test-e2e:
	./tests/e2e/run_all.sh

# 运行所有测试
test-all: test test-integration test-e2e
```

**GitHub Actions CI**:
```yaml
# .github/workflows/test.yml
name: Test Suite

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run unit tests
        run: make test-coverage
      - name: Upload coverage
        uses: codecov/codecov-action@v3

  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run integration tests
        run: make test-integration

  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Start services
        run: docker-compose up -d
      - name: Run E2E tests
        run: make test-e2e
```

### 11.6 性能测试

**工具**: `k6` 或 `wrk`

**测试场景**:
```javascript
// tests/performance/load_test.js
import http from 'k6/http';
import { check } from 'k6';

export let options = {
  stages: [
    { duration: '30s', target: 50 },  // 爬坡到 50 并发
    { duration: '1m', target: 50 },   // 保持 50 并发
    { duration: '30s', target: 0 },   // 降到 0
  ],
};

export default function() {
  let payload = JSON.stringify({
    model: 'claude-sonnet-4',
    messages: [{ role: 'user', content: 'Hello' }],
    max_tokens: 100,
  });

  let res = http.post('http://localhost:8080/v1/messages', payload, {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer test-token',
    },
  });

  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 2s': (r) => r.timings.duration < 2000,
  });
}
```

**性能目标**:
- P95 响应时间 < 2s（非流式）
- P99 响应时间 < 5s
- 并发处理能力 ≥ 100 请求/秒

---

## 12. 安全设计

### 12.1 API Key 加密

```go
import "crypto/aes"

func encryptAPIKey(plaintext, key string) (string, error) {
    block, err := aes.NewCipher([]byte(key))
    // ... AES-GCM 加密
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptAPIKey(ciphertext, key string) (string, error) {
    // ... AES-GCM 解密
    return string(plaintext), nil
}
```

### 12.2 Token 验证中间件

```go
func AuthMiddleware(tokenService token.Service) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if !strings.HasPrefix(authHeader, "Bearer ") {
            c.JSON(401, gin.H{"error": "Missing or invalid token"})
            c.Abort()
            return
        }

        tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
        token, err := tokenService.Validate(tokenStr)
        if err != nil || !token.Enabled {
            c.JSON(401, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        c.Set("token", token)
        c.Next()
    }
}
```

---

## 13. 监控与日志

### 13.1 请求统计

```go
type RequestStats struct {
    TotalRequests   int64
    SuccessRequests int64
    FailedRequests  int64
    ByProvider      map[string]int64
    ByModel         map[string]int64
}

func (s *StatsCollector) RecordRequest(provider, model string, success bool) {
    atomic.AddInt64(&s.TotalRequests, 1)
    if success {
        atomic.AddInt64(&s.SuccessRequests, 1)
    } else {
        atomic.AddInt64(&s.FailedRequests, 1)
    }
    // ... 记录到数据库或内存
}
```

### 13.2 结构化日志

```go
import "go.uber.org/zap"

logger, _ := zap.NewProduction()
logger.Info("Request processed",
    zap.String("provider", provider.Name),
    zap.String("model", mapping.TargetModel),
    zap.Int("status_code", resp.StatusCode),
    zap.Duration("latency", latency),
)
```

---

## 14. 下一步

- [ ] 完成 Go 格式转换引擎实现
- [ ] 实现供应商管理模块
- [ ] 实现模型映射与路由
- [ ] 实现负载均衡与故障转移
- [ ] 设计 Astro 前端界面
- [ ] Docker 打包与部署
- [ ] 编写 API 文档
- [ ] 编写用户手册

---

**参考文档**:
- [格式转换核心学习](./format-conversion-study.md)
- [claude-code-nexus PRD](../ref_proj/claude-code-nexus/REQUIREMENTS.md)
