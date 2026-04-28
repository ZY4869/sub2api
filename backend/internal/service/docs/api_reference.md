# API 文档中心

> 仓库默认 API 文档基线已经迁移为多文件结构，服务端运行时会拼装完整 Markdown。

## 当前基线路径

- `backend/internal/service/docs/index.md`
- `backend/internal/service/docs/pages/common.md`
- `backend/internal/service/docs/pages/openai-native.md`
- `backend/internal/service/docs/pages/openai.md`
- `backend/internal/service/docs/pages/anthropic.md`
- `backend/internal/service/docs/pages/gemini.md`
- `backend/internal/service/docs/pages/grok.md`
- `backend/internal/service/docs/pages/deepseek.md`
- `backend/internal/service/docs/pages/antigravity.md`
- `backend/internal/service/docs/pages/vertex-batch.md`
- `backend/internal/service/docs/pages/document-ai.md`

## 维护说明

- 用户侧与管理端接口仍然读取服务端拼装后的完整 Markdown。
- 管理端运行时覆盖继续按 `page_id` 粒度存储，不会回写仓库基线。
- 面向终端用户的“文档同步说明”不要再写回协议页正文，维护约束统一保留在本文件。
- `document-ai` 的路由和 page_id 保持英文 slug，但用户可见标题统一展示为“百度智能文档”。
- `deepseek` 表示 DeepSeek 一级平台的 OpenAI / Anthropic 兼容入口；示例默认使用 `deepseek-v4-flash` / `deepseek-v4-pro`。
- `openai-native` 表示 `/v1/responses` 等 OpenAI 原生入口；`openai` 表示 `chat/completions` 等兼容入口。
- 后续协议说明、认证规则、错误示例和代码样例请直接维护上述多文件基线，并保持代码块的 `focus=` 重点行元数据。
