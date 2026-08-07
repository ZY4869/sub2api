# 上游 v0.1.168-v0.1.171 Clean-room 同步矩阵

## 范围与约束

- 上游参考：`Wei-Shaw/sub2api` releases `v0.1.168=99c8e4b`、`v0.1.169=26d894e`、`v0.1.170=c043c24`、`v0.1.171=f0e7a9c`。
- 本地基线：`0.1.412`，保留本地 MIT License 与本地版本线，不采用上游新增许可证文本。
- 禁止操作：不执行 `git pull`、`git fetch`、merge、rebase、cherry-pick；不复制上游 LGPL/GPL/CLA 文本、赞助/partner 资产、README 许可段、发布脚本或无关 CI。
- 项目规则：不新增 `/api-docs/*` 或 `/admin/api-docs/*`；公开模型列表、模型广场和模型详情只暴露 `display_model_id`，`target_model_id` 仅内部诊断/管理用途；读路径不触发同步 probe。
- 迁移编号：本地新增 migration 使用连续编号 `170_add_passkey_credentials.sql`、`171_add_group_profit_control.sql`，自动忽略上游同名或上游编号。

## 同步矩阵

| 上游项 | 本地状态 | 本地处理 |
|---|---|---|
| Passkey 注册、登录、重命名、删除、最近使用时间 | 已重写融合/已补测 | 通过本地 ent schema、repository、service、handler、路由和前端 Profile/Login 接入；管理员开关 `passkey_enabled` 已接入后端设置键、保存链路、公开设置输出和用户入口 fail-closed gate；有效条件为 `webauthn.enabled && passkey_enabled`。回归：`TestPasskeyRepositoryUserHandleIsIdempotent`、`TestPasskeyRepositoryListRenameAndDeleteEnforceOwnership`、`TestPasskeyRepositoryUpdateCredentialStoresLastUsedAt`、`TestPasskeySettingsRequireWebAuthnConfigAndStoredSetting`、`TestPasskeySettingsInjectionIncludesEffectiveFlag`、`TestSettingHandlerUpdateSettings_PasskeyEnabledRoundTrip`、`TestPasskeyHandlerDisabledGateRejectsBeginLogin`、`ProfilePasskeyCard.spec.ts`。 |
| 模型广场 `/model-plaza` | 已重写融合/已补测 | 后端使用本地 policy projection 与可用快照；匿名仅看非独享组，登录用户可看授权独享组；过滤非 active channel，按 group 平台隔离模型并去重；专用公开 DTO 不暴露 `target_model_id`。回归：`TestModelPlazaBuildGroupsVisibilityFilteringAndDTOPrivacy`、`ModelPlazaContent.spec.ts`。 |
| Kimi K3、GPT-5.6 Luna/Terra、GLM-5.2、Sonnet 5 状态别名、图片价格展示 | 已重写融合 | GPT-5.6 已在本地存在；新增 `kimi-k3`、`glm-5.2`、`claude-sonnet-5` 到本地 registry/catalog/pricing，价格标记为 clean-room local fallback；模型广场补齐图片价与 web search 价展示；GLM-5.2 精确 pricing 与子串定价风险纳入回归检查。 |
| 上游 URL 路径片段闭集校验（GHSA-vrxq-qm4h-6hgg） | 重写融合 | 按本地网关路径拼接边界保留闭集校验，不接受客户端可控片段直接进入上游 URL 路径。 |
| 容器 no-new-privileges | 已重写融合 | `deploy/docker-compose*.yml` 的 `sub2api` 服务加入 `security_opt: no-new-privileges:true`。 |
| 腾讯天御验证码 | 已重写融合/已补测 | 作为统一验证码 provider 接入登录、注册、找回密码、邮箱验证、OAuth start 与 Passkey 登录 proof；缺 app/cloud 密钥或 proof 时 fail-closed。OAuth start 同时支持 GET query 兼容与 POST body proof，POST body 的 `mode/redirect/aff_code` 优先于 query，资料页身份绑定已切到带 Authorization 的 POST start。回归：`TestCaptchaServiceVerifyTencentFailClosedAndPassesProof`、`TestDingTalkOAuthStartBuildsAuthorizeURLAndCookies`、`TestSocialOAuthStartPostBodyDrivesBindRedirectAndAffCode`、`TestSocialOAuthStartBindPostRequiresAuthorization`、`TestLinuxDoOAuthStartPostBodyReturnsAuthorizeURL`、`SocialOAuthSection.spec.ts`、`AuthIdentitiesCard.spec.ts`。 |
| 阿里云验证码 2.0 | 已重写融合/已补测 | 与 Turnstile/Tencent 互斥；设置页合并为单卡 provider 选择，后端保存互斥校验并 fail-closed。回归：`TestCaptchaServiceVerifyAliyunFailClosedAndPassesProof`、`TestCaptchaProviderMutualExclusionAndSelection`。 |
| 设置页单卡互斥验证码 | 已重写融合 | `SettingsSecurityAuthTab.vue` 按 `none/turnstile/tencent/aliyun` 单选切换，保存时关闭其它 provider。 |
| 前端 refresh-token 并发竞争 | 已重写融合/已补测 | 集中刷新队列与 Web Locks 行为收口到 `tokenRefresh.ts`、`client.ts`、`auth.ts`、store；覆盖同标签 in-flight 去重、Web Locks、跨标签新 token 复用和失败清理。回归：`tokenRefresh.spec.ts`。 |
| 分组利润控制 | 已重写融合，默认关闭 | `groups.profit_control_enabled/min_margin/safety_buffer` 默认关闭；创建/编辑 UI 与 preview API 已接入；不开启不改变现有计费。 |
| profit-preview | 已重写融合 | 新增管理员 `POST /admin/groups/profit-preview`，返回后端规范化后的倍率预览。 |
| 上游计费倍率探测与自动写回 | 已覆盖/保守融合，默认关闭 | 本地保留 existing upstream billing probe 设置；自动写回仍默认关闭，避免未经管理员显式开启改变账号倍率。 |
| 退款 `require_force` | 已重写融合 | 余额不足时返回 `PAYMENT_REFUND_REQUIRES_FORCE` 与 `require_force=true` metadata；前端二次确认后带 `force:true`，不再静默部分扣减。 |
| Usage log retained on billing failure | 已重写融合 | `usage_billing_apply.go` 写入 `billing_failed` 失败码；OpenAI/WebSearch/通用 gateway 计费失败时保留 `failed` 且 `actual_cost=0` 的 usage log。回归：`TestOpenAIGatewayServiceRecordUsage_BillingErrorRetainsFailedZeroChargeUsageLog`、`TestOpenAIGatewayServiceRecordWebSearchUsage_BillingErrorRetainsFailedZeroChargeUsageLog`、`TestGatewayServiceRecordUsage_BillingErrorRetainsFailedZeroChargeUsageLog`。 |
| Subscription renewal lock / quota window 修复 | 已重写融合/已覆盖 | `GetByUserIDAndGroupIDForUpdate` 与 `assignOrExtendSubscriptionWithRowLock` 在 ent 事务中锁定续期行；无 ent client 时保留本地 fallback。回归：`TestAssignOrExtendSubscriptionExistingFallbackExtendsFromCurrentExpiry`；quota window 已由 reset credit cache refresh/recovery 相关测试覆盖。 |
| Anthropic interrupted stream partial usage billing | 已覆盖 | 本地流式 drain/partial usage 行为已覆盖：`TestGatewayService_AnthropicAPIKeyPassthrough_StreamingStillCollectsUsageAfterClientDisconnect`、`TestOpenAIStreamingClientDisconnectDrainsUpstreamUsage`、`TestOpenAIGatewayService_OAuthPassthrough_StreamClientDisconnectStillCollectsUsage`。 |
| `count_tokens` 参数清理 | 已重写融合 | `stripCountTokensGenerationFields` 清理 `stream`、`temperature`、`top_p`、`tools`、`tool_choice`、`max_tokens` 等生成参数，避免 Anthropic `count_tokens` 携带生成字段。回归：`TestGatewayService_AnthropicAPIKeyPassthrough_CountTokensFiltersGenerationFields`。 |
| OpenAI WS close frame / SSE 429 / pool capacity retry | 已覆盖/已补测 | WS close frame 保留本地 `closeOpenAIClientWS`、`OpenAIWSClientCloseError` 契约；SSE/JSON failover exhausted 429 已补 `TestGatewayHandleFailoverExhaustedMaps429ForJSONAndSSE`；pool capacity retry 由 `FailoverState.HandleFailoverError`、`HandleSelectionExhausted` 与 `failover_loop_test.go` 现有容量类测试覆盖。 |
| Messages temporary account failover | 已覆盖 | `gateway_messages_claude.go`、`gateway_messages_gemini.go` 继续通过 `TempUnscheduleRetryableError` 与 failover loop 切换临时不可调度账号；本轮保留本地调度模型。 |
| Codex namespace tools / instructions / web search manifest | 已覆盖 | `openai_responses_lite_tools.go/test` 覆盖 `tool_search`、namespace tools、`additional_tools`；`openai_codex_transform.go/test` 覆盖 instructions；`openai_oauth_passthrough_test.go` 覆盖 web search manifest 透传。 |
| Codex originator normalization 与版本同步 | 本地等价覆盖 | 不新增独立 version-sync API，也不新增文档中心；归并到现有 Settings API 的 `min_claude_code_version`、`max_claude_code_version`、Codex OAuth UA policy 与 `gateway.disable_codex_originator_normalization` 配置行为，底层开关键名为 `disable_codex_originator_normalization`。默认 `false` 保持 originator normalization，开启时保留调用方 originator 但仍补齐最低 `version`。回归：`TestSettingHandlerUpdateSettings_CodexVersionAndUserAgentPolicyRoundTrip`、`TestSettingHandlerUpdateSettings_OpenAIClaudeCodeCodexPluginRoundTrip`、`TestEnforceCodexIdentityHeadersWithConfig_CanDisableOriginatorNormalization`。 |
| OpenAI reset credit cache refresh/recovery | 已覆盖 | `openai_quota_service.go` 保持 reset credit 快照、兑换后清本地 quota/runtime state、上游可用时恢复缓存。回归：`TestOpenAIQuotaService_ReadResetCreditsPersistsWhamSnapshot`、`TestAccountHandlerGetUsageResetCreditsReadFailureDoesNotReturnStaleCount`、`TestOpenAIQuotaService_ResetCreditPostsRedeemRequestIDAndClearsLocalQuotaAndRuntimeState`、`TestOpenAIQuotaService_QueryUsageClearsLocalRuntimeStateWhenUpstreamAvailable`。 |
| 模型复制按钮、筛选结果全选账号、批量删除限并发、compact home preset | 已重写融合/已覆盖 | 模型广场保留 `copyModel(model.display_model_id)` 与 `useClipboard`；账号页新增 `select-filtered` 动作，按当前筛选分页拉取账号 ID 并 `setSelectedIds`；批量删除通过 `runWithConcurrency` 限制并发 4；首页 compact preset 不引入上游同名配置，本地已有 `visual_preset_default`、`account_airy_white_surface_enabled` 与 `home_content` 自定义入口作为等价本地能力。回归：`useAccountsBulkActions.spec.ts`、`AccountBulkActionsBar.spec.ts`。 |
| 上游新许可、README、赞助/合作、发布脚本、无关 CI | 明确排除 | 不引入，release guard 继续校验 MIT-only 与无 `/api-docs/*`。 |

## 验证要求

- 后端：`go test` 覆盖 auth/passkey/captcha/settings/model plaza/group profit/payment/gateway/release guard；本轮新增/修正 `TestPasskeyRepositoryUserHandleIsIdempotent`、`TestPasskeyRepositoryListRenameAndDeleteEnforceOwnership`、`TestPasskeyRepositoryUpdateCredentialStoresLastUsedAt`、`TestPasskeySettingsRequireWebAuthnConfigAndStoredSetting`、`TestPasskeySettingsInjectionIncludesEffectiveFlag`、`TestSettingHandlerUpdateSettings_PasskeyEnabledRoundTrip`、`TestPasskeyHandlerDisabledGateRejectsBeginLogin`、`TestCaptchaServiceVerifyTencentFailClosedAndPassesProof`、`TestCaptchaServiceVerifyAliyunFailClosedAndPassesProof`、`TestModelPlazaBuildGroupsVisibilityFilteringAndDTOPrivacy`。
- 前端：`pnpm typecheck` 与 targeted vitest 覆盖认证页、设置页、模型广场、支付退款、分组利润控制和 token refresh；本轮新增 `tokenRefresh.spec.ts`、`ProfilePasskeyCard.spec.ts` 与 `ModelPlazaContent.spec.ts`。
- 收尾：`git diff --check`；搜索 `/api-docs`、`LGPL|GPL|CLA`、公开 `target_model_id`；确认 `LICENSE`、README 许可段和 `backend/cmd/server/VERSION` 仍为本地版本线。
