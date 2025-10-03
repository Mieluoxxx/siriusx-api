export default function Dashboard() {
  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 py-8 px-4 sm:px-6 lg:px-8">
      <div className="max-w-7xl mx-auto">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900">
            Siriusx-API 管理界面
          </h1>
          <p className="mt-2 text-sm text-gray-600">
            统一 API 转发和模型映射管理平台
          </p>
        </div>

        {/* 快速操作链接 - 管理功能卡片 */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
          <a
            href="/providers"
            className="block bg-white rounded-lg shadow p-6 hover:shadow-xl hover:scale-105 transition-all duration-200 border-t-4 border-blue-500"
          >
            <h3 className="text-lg font-semibold text-gray-900 mb-2">
              📡 供应商管理
            </h3>
            <p className="text-sm text-gray-600">
              管理 API 供应商配置和健康状态
            </p>
          </a>

          <a
            href="/models"
            className="block bg-white rounded-lg shadow p-6 hover:shadow-xl hover:scale-105 transition-all duration-200 border-t-4 border-purple-500"
          >
            <h3 className="text-lg font-semibold text-gray-900 mb-2">
              🤖 模型管理
            </h3>
            <p className="text-sm text-gray-600">
              配置统一模型和映射关系
            </p>
          </a>

          <a
            href="/tokens"
            className="block bg-white rounded-lg shadow p-6 hover:shadow-xl hover:scale-105 transition-all duration-200 border-t-4 border-green-500"
          >
            <h3 className="text-lg font-semibold text-gray-900 mb-2">
              🔑 Token 管理
            </h3>
            <p className="text-sm text-gray-600">
              创建和管理 API 访问令牌
            </p>
          </a>
        </div>


        {/* ClaudeCode 使用指南 */}
        <div className="bg-white rounded-lg shadow">
          <div className="px-6 py-4 border-b border-gray-200">
            <h2 className="text-lg font-semibold text-gray-900">ClaudeCode 使用指南</h2>
            <p className="mt-1 text-sm text-gray-500">
              配置 Claude Code CLI 工具来使用 Siriusx-API
            </p>
          </div>
          <div className="px-6 py-4">
            <div className="prose max-w-none">
              <h3 className="text-md font-semibold text-gray-900 mb-3">环境变量配置</h3>
              <p className="text-sm text-gray-600 mb-3">
                如果需要在终端中自动使用配置,可以将以下内容添加到 <code className="bg-gray-100 px-1 py-0.5 rounded">.bashrc</code> 或 <code className="bg-gray-100 px-1 py-0.5 rounded">.zshrc</code> 中:
              </p>
              <pre className="bg-gray-900 text-gray-100 p-4 rounded-lg overflow-x-auto text-sm">
{`# 设置认证令牌
export ANTHROPIC_AUTH_TOKEN="sk-your-token-here"

# 设置 API 基础 URL
export ANTHROPIC_BASE_URL="https://your-api-domain.com"`}
              </pre>

              <h3 className="text-md font-semibold text-gray-900 mb-3 mt-6">运行 Claude Code</h3>
              <p className="text-sm text-gray-600 mb-3">
                配置完成后,使用以下命令运行 Claude Code:
              </p>
              <pre className="bg-gray-900 text-gray-100 p-4 rounded-lg overflow-x-auto text-sm">
{`# 使用统一模型名称
claude --model [统一模型名]

# 例如:
claude --model claude-sonnet-4-5`}
              </pre>

              <div className="mt-4 bg-blue-50 border-l-4 border-blue-500 p-4 rounded">
                <p className="text-sm text-blue-700">
                  <strong>提示:</strong> 统一模型名称可以在 <a href="/models" className="underline">模型管理</a> 页面查看。Token 可以在 <a href="/tokens" className="underline">Token 管理</a> 页面创建。
                </p>
              </div>
            </div>
          </div>
        </div>

      </div>
    </div>
  );
}
