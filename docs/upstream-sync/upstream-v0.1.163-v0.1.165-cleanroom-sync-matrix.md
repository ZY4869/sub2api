# 上游 v0.1.163-v0.1.165 Clean-room 同步矩阵

## 范围与基线

- 真实上游参考：`Wei-Shaw/sub2api`
- 上游基线：`v0.1.162=27f094e0960ebd8e52de7ff7e763c6fec2ff4057`
- 上游增量：`v0.1.163=d0bdd7e771636a8d315f542cafd39484f39bd60c`、`v0.1.164=cd8bb98c44303b2c8f04c0da340447c992f0cb7d`、`v0.1.165=e9a58c1cb8b5ef626a75c93b4d953fde5e67aa29`
- 本地基线：`0.1.407`
- 执行方式：clean-room 本地重写；不执行 `git pull`、`git fetch`、merge、rebase、cherry-pick；不复制上游 LGPL/GPL/CLA 协议文本、源码片段、测试、README 许可文案、workflow、发布脚本或上游迁移编号。
- 许可证决策：`LICENSE` 与 README 许可段继续保持 MIT-only。

## 吸收矩阵

| 上游项 | 本地状态 | 本地处理 |
|---|---|---|
| 上游许可协议变化 | 明确排除 | 保留本地 MIT License 与 README MIT-only 许可段；`backend/internal/repository/release_guard_test.go` 校验 `0.1.407`、禁用 API docs 路由和运行代码无 copyleft/CLA 文本。 |
| 分组 `platform=composite` 与聚合路由 | 融合重写 | `backend/ent/schema/composite_model_route.go`、`backend/internal/server/routes/admin.go`、`backend/internal/service/openai_gateway_composite_runtime.go` 与 `backend/internal/service/openai_gateway_composite_runtime_test.go` 覆盖后台 GET/PUT/preview 与运行时解析；外部只用 `display_model_id`，`target_model_id` 仅内部转发。 |
| 分组级 Live 开关 | 融合重写 | `backend/ent/schema/group.go` 的 `groups.allow_live` 默认关闭；`backend/internal/server/routes/gateway.go`、`backend/internal/handler/openai_gateway_handler_websocket.go` 和 `backend/internal/service/protocol_capability_matrix_test.go` 覆盖 `/v1/live`、`/backend-api/codex/realtime/calls`、组级允许检查与 request type。 |
| OpenAI reasoning effort 分组策略 | 融合重写 | `backend/internal/service/group_reasoning_policy.go` 与 `backend/internal/service/group_reasoning_policy_test.go` 覆盖 `max_reasoning_effort`、`reasoning_effort_mappings`、别名、模型匹配和 `none < low < medium < high < xhigh < max` effective 约束；HTTP/WS 调用点在 OpenAI gateway 与 WS forwarder 中复用 `ApplyContextOpenAIReasoningPolicy`。 |
| `usage_logs.session_id` | 本地重写 | 本地迁移 `167_add_usage_log_session_id.sql`、`backend/ent/schema/usage_log.go`、`backend/internal/repository/usage_log_repo_request_type_test.go` 和 `backend/internal/service/openai_gateway_record_usage_test.go` 覆盖 Live/WS `session_id` 持久化。 |
| 注册邮箱别名去重 | 本地重写 | 本地迁移 `168_add_user_email_alias.sql`、`backend/ent/schema/user.go`、`backend/internal/repository/user_repo.go` 和 `backend/internal/service/user_email_alias_test.go` 覆盖 `users.email_alias`、Gmail/Googlemail 别名归一、唯一索引与创建/查询写路径。 |
| Ollama Cloud 用量请求驱动刷新与 15 分钟下限 | 不适用 | 本地仓库没有 Ollama 平台、账号或用量刷新子系统；本轮不新增平行 Ollama 实现，也不把该项标为已覆盖。 |
| Alipay 移动端 deep link | 明确排除 | 本地支付栈是 Airwallex，锚点为 `backend/migrations/118_add_airwallex_payments.sql`、`frontend/src/composables/usePaymentWorkbench.ts` 与 `payment_mobile_force_qrcode_enabled`；没有 Alipay provider，不引入支付宝 deep link。 |
| 公告预览与共享富文本样式 | 已覆盖 | 本地已有公告后台编辑与用户端富文本展示：`frontend/src/views/admin/AnnouncementsView.vue`、`frontend/src/components/common/AnnouncementPopup.vue`、`frontend/src/components/common/AnnouncementBell.vue` 统一用 markdown + DOMPurify 渲染。 |
| `claude-opus-5` 模型/定价/Bedrock 映射/前端预设/限流 scope | 已覆盖 | 用户确认该模型已可用后，本地纳入 `backend/internal/modelregistry/registry_seed.json`、`backend/internal/service/model_catalog_seed.json`、`backend/resources/model-pricing/model_prices_and_context_window.json`、`backend/internal/domain/constants.go` Bedrock 默认映射、Antigravity 默认映射/预设与前端模型白名单；默认可用 bootstrap 使用本地 marker `model_registry_available_models_bootstrap_v20260726`。 |
| 图像请求日志记录 `quality`/`size` | 已覆盖 | 本地迁移 `169_add_usage_log_image_quality.sql` 新增 `usage_logs.image_quality`；`backend/ent/schema/usage_log.go`、`usage_log_repo_write.go`、`usage_log_repo_scan.go`、`openai_gateway_usage.go`、`usage_log_failure.go`、DTO mapper 与后台/用户 UsageTable 同步记录和展示。`quality` 来源为现有 OpenAI image normalized request，只做日志诊断，不改变公开计费语义。 |
| 推广复制按钮移动端适配 | 已覆盖 | 本地推广/affiliate 管理与注册入口保留，复制能力统一走 `frontend/src/composables/useClipboard.ts`；未发现本轮需改的独立移动端推广按钮。 |
| Grok compact/client tools/video content/402/5xx/tool_choice/策略 403/缓存会话 | 已覆盖 | `backend/internal/service/grok_gateway_messages_compat_test.go`、`grok_quota_fetcher_test.go` 与 `docs/upstream-sync/grok-build-cache-quota-display-note-20260723.md` 锚定本地 Grok build/cache/quota 兼容与回归，本轮不回退。 |
| OpenAI input namespace、item ID、OAuth passthrough input、same-account retry、stream quarantine、remote pricing 空 URL | 融合重写 | 复用本地 `backend/internal/service/openai_codex_transform_test.go`、`openai_oauth_passthrough_test.go`、`openai_invalid_encrypted_content_test.go`、`pricing_service_openai_fallback.go` 等修复，不引入上游源码片段。 |
| Gemini chat completions 图像输出 | 已覆盖 | 保留本地 Gemini mixed/image/batch 兼容路径，锚点为 `backend/internal/service/protocol_capability_matrix_test.go`、`backend/internal/service/vertex_upstream_catalog_service_test.go` 与 `backend/internal/handler/gemini_v1beta_surface_test_helper_test.go`。 |
| Ollama PG<=16 due 判定、会话脱敏、刷新饥饿 | 不适用 | 本地没有 Ollama PG due 判定或 Ollama quota fetcher；会话脱敏已有 OpenAI/Grok/OAuth 本地路径，但不把 Ollama 专属项标为已覆盖。 |
| 优雅关停 cleanup、scheduler quota metadata、LastUsedAt 隔离、渠道监控解密失败停调度、后台使用记录筛选、成本倍率小数 | 已覆盖 | 现有 scheduler/usage/account/channel 相关测试继续守护本地行为；`release_guard_test.go` 只把许可证、版本、禁用路由和本轮矩阵锚点作为收口 guard。 |

## 本地新增迁移

- `backend/migrations/165_add_group_live_reasoning_fields.sql`
- `backend/migrations/166_create_composite_model_routes.sql`
- `backend/migrations/167_add_usage_log_session_id.sql`
- `backend/migrations/168_add_user_email_alias.sql`
- `backend/migrations/169_add_usage_log_image_quality.sql`

这些是本地迁移编号，不采用上游迁移文件或上游迁移编号文本。

## 验证要点

- `go generate ./ent` 只由 Ent schema 生成，不手工编辑 Ent 生成物。
- release guard 校验 `MIT License`、`0.1.407`、无 `/api-docs/*` 或 `/admin/api-docs/*`、无 LGPL/GPL/CLA 文本进入运行代码、矩阵关键项存在。
- release guard 明确禁止把 Ollama、Alipay 等无本地子系统项继续写成 `已覆盖`；`claude-opus-5` 必须有 registry、pricing、Bedrock、前端白名单和生成快照证据；图像 `quality` 必须有 `169_add_usage_log_image_quality.sql`、`usage_logs.image_quality`、DTO 和 UI 展示证据。
- 前端后台分组 UI 只把 `composite` 暴露给分组管理；账号平台、渠道平台和通用筛选不扩展到 composite。
- 公开模型枚举仍以 `display_model_id` 为外部 ID，`target_model_id` 仅用于后台配置和内部转发。
