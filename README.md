我想要做一个大模型API格式转换项目

现在主要基于上游new-api

1. 收集各家供应商的API，汇总为固定的模型
    - 收集各家API OpenAI兼容的 BASE URL 和 API Key （可以采用多个Key轮询的方式）统一管理，Tab页面为供应商管理
    - 拥有一个模型管理，可以自定义模型的名称，解决明明混乱问题，例如有的叫做claude-4-sonnet，有的叫做claude-sonnet-4，我可以自定义为claude-sonnet-4，然后将其他供应商的模型重定向这个name，
    - 模型管理具有优先级和权重两个属性
2. 可以自定义模型的端点，上游输入new-api（OPENAI兼容格式），下游转发可以使用openai的/v1/chat/completions,也可以选择端点为claude的/v1/messages
3. 轻量级