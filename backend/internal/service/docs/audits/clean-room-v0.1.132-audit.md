# Clean-Room v0.1.132 收口审计

日期：2026-05-29
分支：`cleanroom/upstream-v0.1.132-cleanroom`
本地版本基线：`v0.1.324-dirty`

## 审计边界

- 本次收口只基于本地实现与行为意图核验，不执行上游拉取、合并或变基。
- 本地 `LICENSE` 首行为 `MIT License`；`README.md` 与 `README_CN.md` 的许可证段落仍为 `MIT License`。
- 项目源码、文档与部署文件按许可证关键词扫描，排除第三方依赖与静态图标包后未发现上游协议替换痕迹。
- 工作区在收口前已存在大量 staged、unstaged 与 untracked 文件；本记录用于固定审计口径，不代表已将既有重构折叠成单一小变更。
- 迁移编号保持本地序列：`backend/migrations/125_add_group_visible_model_patterns.sql`；未使用上游迁移编号。

## 行为核验

- 分组自定义模型列表：`visible_model_patterns` 只在本地公开目录、账号策略、API Key 绑定与运行时可服务候选上取交集，不扩大可见或可调用集合。
- 账号池同账号重试状态码：`credentials.pool_mode_retry_status_codes` 默认兼容 `[401,403,429]`，并与 `pool_mode_retry_count` 保持独立。
- Responses 流式失败：`/v1/responses`、`/responses` 与相关兼容路径在 SSE 已开始或 writer 已写入后发送 `response.failed` 事件。
- 长上下文计费：长上下文倍率覆盖输入、输出、`cache_read` 与 `cache_creation` 相关成本。
- Antigravity usage：流式 `message_start.usage.input_tokens` 会进入 usage 解析与记录链路。
- OpenAI Chat Responses usage：Responses 与 Chat Completions 桥接路径保留 usage 计费信息。
- 模型级冷却：模型运行时 404 / 额度侧限制按账号-模型组合记录，不冷却整号；公开枚举读路径临时隐藏不可服务模型。
- OpenAI WS failover：限额信号触发账号切换，同时保留既有限额持久化与恢复逻辑。
- 重新授权：OAuth 账号刷新保留 `Extra` 配置，并通过 token cache invalidator 清理旧凭据缓存。
- Bedrock 转发：被过滤的 beta 能力会同步清理不适用的 `context_management` 片段。
- Ops SLA：本地策略限制按 business-limited 归类，不计入上游 SLA 错误。

## 文档与契约

- `backend/internal/service/docs/pages/common.md` 已记录 `visible_model_patterns`、`pool_mode_retry_status_codes`、业务限制和 OpenAI Pro 额度冷却语义。
- `backend/internal/service/docs/pages/openai.md` 与 `backend/internal/service/docs/pages/openai-native.md` 已记录 OpenAI Responses / Chat Completions 在额度冷却时返回 `429 rate_limit_error`，以及模型枚举临时隐藏语义。
- `backend/internal/service/docs/pages/openai-native.md` 已记录 Responses SSE 已开始后失败会追加 `response.failed`。

## 测试锚点

- 分组模型收敛：`backend/internal/service/api_key_public_models_test.go`。
- 池模式重试状态码：`backend/internal/service/account_pool_mode_test.go`、`frontend/src/utils/__tests__/accountFormShared.spec.ts`、`frontend/src/utils/__tests__/accountApiKeyAdvancedSettingsForm.spec.ts`。
- Responses SSE 失败事件：`backend/internal/handler/openai_gateway_handler_test.go`、`backend/internal/handler/gateway_handler_error_fallback_test.go`。
- 长上下文 cache 计费：`backend/internal/service/billing_service_test.go`、`backend/internal/service/billing_center_service_test.go`。
- Antigravity usage：`backend/internal/service/antigravity_gateway_service_test.go`。
- OpenAI usage 与 WS failover：`backend/internal/service/openai_gateway_responses_chat_bridge_test.go`、`backend/internal/service/openai_ws_protocol_forward_test.go`、`backend/internal/service/openai_ws_ratelimit_signal_test.go`。
- 模型级冷却：`backend/internal/service/api_key_public_models_test.go`、`backend/internal/service/ratelimit_service_clear_test.go`。
- 重新授权与缓存清理：`backend/internal/service/token_refresh_service_test.go`、`backend/internal/service/token_cache_invalidator_test.go`。
- Bedrock context 清理：`backend/internal/service/gateway_request_test.go`、`backend/internal/service/gateway_service_bedrock_beta_test.go`。
- Ops SLA 分类：`backend/internal/handler/ops_error_classify*.go` 相关测试与 `backend/internal/service/ops_*_test.go`。

## 收口验证命令

```powershell
git status --short --branch
Get-Content LICENSE -TotalCount 5
Select-String -Path README.md,README_CN.md -Pattern '许可证|License|MIT' -Context 1,2
rg -n "许可证关键词模式" LICENSE README.md README_CN.md docs deploy backend frontend -S -g '!frontend/node_modules/**' -g '!backend/tmp/**' -g '!frontend/public/lobehub-icons-static-svg/package/**'
go test ./internal/service ./internal/handler ./internal/repository ./internal/server -count=1
pnpm --dir frontend typecheck
pnpm --dir frontend test:run
```

## 最终验证结果

- `LICENSE`、`README.md`、`README_CN.md` 许可证核验通过，仍为本地 MIT。
- 协议关键词扫描通过：排除第三方依赖与静态图标包后，项目源码、文档与部署文件 0 命中。
- `go test ./internal/service ./internal/handler ./internal/repository ./internal/server -count=1` 通过。
- `pnpm --dir frontend typecheck` 通过。
- `pnpm --dir frontend test:run` 通过，用时约 115 秒。
- 工作区仍保留收口前已有的大量 staged、unstaged 与 untracked 文件，后续审查应继续按功能域拆分查看。
