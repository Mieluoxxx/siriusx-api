# Siriusx-API Web 管理界面

基于 Astro + React + Tailwind CSS 的现代化 Web 管理界面。

## 功能特性

- ✅ 系统概览仪表板 - 实时监控系统状态
- ✅ 供应商管理 - CRUD 操作,健康检查,状态切换
- ✅ 模型管理 - 统一模型配置查看
- ✅ Token 管理 - 安全的令牌创建和管理
- ✅ 响应式设计 - 适配各种屏幕尺寸
- ✅ 自动刷新 - Dashboard 每 5 秒自动更新

## 技术栈

- **Astro** 4.x - 静态站点生成
- **React** 18.x - 交互组件
- **TypeScript** - 类型安全
- **Tailwind CSS** 3.x - 实用优先的 CSS 框架

## 快速开始

### 安装依赖

```bash
pnpm install
```

### 开发模式

```bash
pnpm dev
```

访问 http://localhost:4321

### 构建生产版本

```bash
pnpm build
```

### 预览构建结果

```bash
pnpm preview
```

## 环境变量

创建 `.env` 文件:

```env
PUBLIC_API_URL=http://localhost:8080
```

## 项目结构

```
web/
├── src/
│   ├── components/        # React 组件
│   │   ├── Dashboard.tsx
│   │   ├── ProviderManagement.tsx
│   │   ├── ModelManagement.tsx
│   │   └── TokenManagement.tsx
│   ├── layouts/           # Astro 布局
│   │   └── Layout.astro
│   ├── pages/             # 页面路由
│   │   ├── index.astro
│   │   ├── providers.astro
│   │   ├── models.astro
│   │   └── tokens.astro
│   └── lib/
│       └── api.ts         # API 客户端
├── public/                # 静态资源
├── astro.config.mjs
├── tailwind.config.mjs
└── package.json
```

## API 接口

所有 API 请求通过 `src/lib/api.ts` 统一管理:

- **GET** `/api/stats` - 获取系统统计
- **GET** `/api/providers` - 获取供应商列表
- **POST** `/api/providers` - 创建供应商
- **PUT** `/api/providers/:id` - 更新供应商
- **DELETE** `/api/providers/:id` - 删除供应商
- **PATCH** `/api/providers/:id/enabled` - 切换启用状态
- **POST** `/api/providers/:id/health-check` - 健康检查
- **GET** `/api/models` - 获取模型列表
- **GET** `/api/tokens` - 获取 Token 列表
- **POST** `/api/tokens` - 创建 Token
- **DELETE** `/api/tokens/:id` - 删除 Token

## 设计原则

- **KISS** - 简洁的组件设计和清晰的代码结构
- **DRY** - 复用 Tailwind 样式类和通用组件
- **响应式** - 移动优先的设计方法
- **类型安全** - 完整的 TypeScript 类型定义

## 贡献指南

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 提交 Pull Request

## 许可证

MIT License
