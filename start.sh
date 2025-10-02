#!/bin/bash
# Siriusx-API 快速启动脚本

set -e

echo "🚀 Siriusx-API 快速启动脚本"
echo "================================"
echo ""

# 检查是否在项目根目录
if [ ! -f "go.mod" ]; then
    echo "❌ 错误: 请在项目根目录运行此脚本"
    exit 1
fi

# 检查 Go 环境
if ! command -v go &> /dev/null; then
    echo "❌ 错误: 未找到 Go 环境，请先安装 Go 1.21+"
    exit 1
fi

# 检查 pnpm 环境
if ! command -v pnpm &> /dev/null; then
    echo "⚠️  警告: 未找到 pnpm，尝试使用 npm..."
    if ! command -v npm &> /dev/null; then
        echo "❌ 错误: 未找到 npm，请先安装 Node.js 20+"
        exit 1
    fi
    PKG_MANAGER="npm"
else
    PKG_MANAGER="pnpm"
fi

echo "✅ 环境检查通过"
echo ""

# 安装后端依赖
echo "📦 安装后端依赖..."
go mod download
echo "✅ 后端依赖安装完成"
echo ""

# 安装前端依赖
echo "📦 安装前端依赖..."
cd web
if [ "$PKG_MANAGER" = "pnpm" ]; then
    pnpm install
else
    npm install
fi
cd ..
echo "✅ 前端依赖安装完成"
echo ""

# 创建日志目录
mkdir -p logs

echo "================================"
echo "📝 启动说明:"
echo ""
echo "1️⃣  后端服务 (端口 8080):"
echo "   cd $(pwd)"
echo "   go run ./cmd/server"
echo ""
echo "2️⃣  前端服务 (端口 4321):"
echo "   cd $(pwd)/web"
echo "   $PKG_MANAGER dev"
echo ""
echo "3️⃣  访问管理界面:"
echo "   http://localhost:4321"
echo ""
echo "================================"
echo ""
echo "🎯 启动后端服务..."

# 后台启动后端
nohup go run ./cmd/server > logs/backend.log 2>&1 &
BACKEND_PID=$!
echo "✅ 后端服务已启动 (PID: $BACKEND_PID)"
echo "   日志文件: logs/backend.log"

# 等待后端启动
sleep 3

echo ""
echo "🎨 启动前端服务..."

# 前台启动前端 (用户可以 Ctrl+C 停止)
cd web
if [ "$PKG_MANAGER" = "pnpm" ]; then
    pnpm dev
else
    npm run dev
fi

# 用户按 Ctrl+C 后停止后端
echo ""
echo "🛑 停止后端服务..."
kill $BACKEND_PID 2>/dev/null || true
echo "✅ 所有服务已停止"
