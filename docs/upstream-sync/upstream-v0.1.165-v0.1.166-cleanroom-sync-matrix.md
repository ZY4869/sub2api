# Sub2API 上游 v0.1.165-v0.1.166 clean-room 同步矩阵

## 范围与约束

- 真实上游参考：`Wei-Shaw/sub2api`
- 上游发布：`v0.1.166`
- 上游标签提交：`dc893dd`
- 上游对比范围：`v0.1.165...v0.1.166`
- 本地基线：`0.1.411`
- 执行方式：clean-room 本地重写；不执行 `git pull`、`git fetch`、merge、rebase、cherry-pick；只用 release、compare、commit title 做主题审计，不复制上游源码、测试或文档正文。
- 许可证决策：MIT-only；保留本地 `LICENSE` 和 README 许可段，不复制或引入上游 LGPL/GPL/CLA 协议文本。
- 明确排除：上游 LGPL/GPL/CLA 协议文本、赞助/partner 资产、CI 发布脚本、README 许可段、依赖例行升级、版本号、/api-docs/*、/admin/api-docs/*。
- 新功能策略：面板 API 限流融合重写，默认关闭；漏洞修复按本地架构重写、确认本地已覆盖或标记不适用。

## 真实 compare 主题矩阵

| 上游 compare 主题 | 处理结论 | 本地处理与证据 |
|---|---|---|
| 上游许可证切换 | 排除：许可材料 | 保留本地 MIT-only，不改 `LICENSE`、README 许可段、版本源，不引入 LGPL/GPL/CLA 文本。release guard 继续检查许可证、版本号、API docs 禁区和运行代码 copyleft 文本隔离。 |
| `feat(security): add panel API rate limiting to protect DB from high-frequency requests` / panel API rate limit | 已重写引入 | 新增 `panel_rate_limit_settings`，存储在现有 `settings` 表 JSON，无迁移；默认 `{enabled:false,user_rpm:240,heavy_rpm:60,public_ip_rpm:300,exempt_admin:true}`。复用 `backend/internal/middleware.RateLimiter` 的 `Allow(ctx, key, limit, window, opts)`，没有建立第二套限流器。后台接口为 `GET /api/v1/admin/settings/panel-rate-limit` 和 `PUT /api/v1/admin/settings/panel-rate-limit`；前端安全设置卡片支持中英文、加载、保存、开关和数值输入。 |
| 面板限流路由接入 | 已重写引入 | 公开 `/api/v1/settings/public` 与 `/api/v1/pages/:slug` 使用可信真实 IP 的 public 限流，私网、回环、空 IP 跳过；用户面板认证组使用 user 限流；后台 dashboard/channel/account/usage 等高频聚合组使用 heavy 限流；现有 auth 登录、注册、验证码等固定限流保持原策略。Redis 故障在面板限流中 fail-open，超限返回 `429` JSON。 |
| `fix(config): honor explicit CONFIG_FILE path` | 已重写引入 | `backend/internal/config/config_load.go` 加载时优先读取 `CONFIG_FILE` 指定文件；未设置时保持 `DATA_DIR` 与既有搜索路径。`backend/internal/config/config.go` 的 `GetServerAddress()` 同步支持显式配置文件。 |
| `fix(claude): 伪装的 Claude Code CLI 版本号升级到 2.1.220` | 已重写引入 | 默认伪装 UA 为 `claude-cli/2.1.220 (external, cli)`；相关路径保留本地常量和 identity 设置解析，不复制上游实现。 |
| `fix(gateway): detect proxied Claude Code traffic by body to preserve prompt cache` / Claude Code body detection | 已重写引入 | `SetClaudeCodeClientContext` 保留 UA 优先路径；代理改写 UA 时，只在 `messages` 路径、Anthropic 必要头存在、body fingerprint 命中 Claude Code system prompt 且 `metadata.user_id` 可解析时识别为 Claude Code。测试覆盖非 `claude-cli` UA、body fingerprint 命中、prompt `cache_control` 保留、缺少严格头时拒绝。 |
| `fix(deploy): prevent Caddy compression from buffering SSE` | 已重写引入 | `deploy/Caddyfile` 的压缩匹配从宽泛 `text/*` 收窄为具体文本类型，避免 `text/event-stream` 被 Caddy 压缩缓冲。 |
| `fix(ci): make Caddy check portable across awk implementations` | 排除：CI 材料 | 本任务不引入上游 CI 发布脚本或脚本实现；本地只保留 SSE 运行配置修复。 |
| `fix(composite): pass the requested model through when a prefix route leaves upstream_model empty` / composite prefix passthrough | 本地已覆盖并补测试 | 本地 composite route runtime 在 `target_model_id` 为空时使用请求模型作为 `RuntimeModelID` 透传；新增 `Composite` 回归测试锁定 prefix route passthrough。 |
| `fix(settings): keep fields a settings PUT never sent at their stored value` | 本地已覆盖 | 后台设置更新已有 `previousSettings` 保留逻辑，敏感字段空值不会覆盖既有值；本轮保留本地实现。 |
| `test: stop four concurrency tests from failing on a busy machine` | 不适用：上游测试稳定性材料 | 该主题是上游测试耐抖调整，不是运行时行为；本地未复制上游测试。 |
| `fix(admin): filter usage logs by request id` | 本地已覆盖 | 使用记录和运维请求追踪已有 `request_id`、`client_request_id` 过滤/关联字段；本轮不改变公开响应语义。 |
| `fix: show routed user in usage filters` | 本地已覆盖 | 使用记录和管理筛选已有 routed user / user 关联信息，用量视图保留现有本地实现。 |
| `fix(billing): price Antigravity Gemini 3.6 Flash` | 已重写引入 | 官方 Gemini 文档确认 `gemini-3.6-flash` 为 Antigravity agent 新默认模型并给出 1M/64k 上下文与 $1.50/$7.50 per 1M token 定价。本地按既有模型 registry/pricing 流程新增 `gemini-3.6-flash`：模型 seed、catalog seed、pricing JSON、Antigravity 默认映射、前端白名单与 generated registry 同步更新，并由 pricing/modelregistry/domain/frontend 测试锁定。 |
| `fix: show optional affiliate code on registration` | 本地已覆盖 | 注册请求已支持可选 `aff_code`，并传入 `RegisterWithVerification` 与 affiliate 绑定流程；不影响现有邀请码和优惠码流程。 |
| `fix(payment): group dashboard stats by currency` | 本地已覆盖 | 支付和用量统计已有 `cost_by_currency`、`actual_cost_by_currency`、today currency 汇总等多币种字段；本轮不改变支付语义。 |
| `fix(deps): update image and telemetry packages` | 排除：依赖材料 | 普通同步任务不升级依赖、不改锁文件；未引入上游依赖版本变化。 |
| `fix(gemini): 完善gemini号池模式时retryable失效问题` | 已重写引入 | 新增本地 helper `geminiPoolRetryableOnSameAccount`；池模式下尊重账号配置的可重试状态码，仅在默认 failover 路径传入 same-account retry 标记，不覆盖管理员错误策略分支。 |
| `fix(repository): parse nanosecond next_probe_at in due probe scheduling` / nanosecond `next_probe_at` | 不适用但本地同类能力已覆盖 | 本地没有同名 due billing probe `next_probe_at` 字段；同类 `expiry_probe_priority_until` 与 repository schedule priority 解析已支持 `time.RFC3339Nano` 和 `time.RFC3339`，新增 `Probe` 回归测试锁定纳秒时间解析。 |
| `fix(openai): strip foreign reasoning on account failover` / foreign reasoning failover | 已重写引入 | `trimOpenAIEncryptedReasoningItems` 复制 slice/map 后移除外部 provider 专属 `encrypted_content`，避免污染 canonical request；WS/HTTP 恢复路径从 canonical request 派生，并按是否含 `function_call_output` 决定是否保留 `previous_response_id`。新增 `ReasoningFailover` 测试覆盖清洗外部 encrypted reasoning、保留 summary、清理 retry previous_response_id 且不污染原始请求。 |
| `fix(openai): track websocket models per turn` | 本地已覆盖 | OpenAI WS v2 透传路径已有 per-turn `RequestModel` 形态，用于多轮请求的模型和计费记录；本轮只补同步记录与目标测试。 |
| `fix(security-audit): reject unavailable prompt config` | 本地已覆盖 | `securityaudit.Config.Save()` 保存后重新 `Load()`，不可用或配置解析失败会返回错误；运行期不可用返回 `prompt_guard_unavailable`，避免假成功。 |
| `fix(frontend): adapt available channels for mobile` | 本地已覆盖 | 可用渠道/设置等页面已有移动端布局与溢出测试锚点；本轮新增面板限流卡片沿用设置页现有布局风格。 |
| `fix(antigravity):` / `fix(antigravity-openai-compat)` / Antigravity OpenAI-compatible native gateway | 已重写引入 | 新增 `/antigravity/v1beta/openai/...` 兼容路由，保持 `ForcePlatform(antigravity)`，复用 Gemini OpenAI-compatible passthrough dispatcher 和本地鉴权、订阅、分组、审计、用量、错误映射、failover 链路。Antigravity + `EndpointGeminiOpenAICompat` endpoint derivation 不退回 `/v1/messages`；选择账号时强制 Antigravity API key + custom `base_url`，不混用普通 Gemini 或 OpenAI 账号；usage-only non-stream OpenAI-compatible 响应返回受控错误 `GEMINI_OPENAI_COMPAT_USAGE_ONLY_RESPONSE`。 |
| Antigravity OpenAI-compatible chat/completions、responses、embeddings、files、batches、models 等路径 | 已重写引入 | `protocol_capability_registry`、gateway routes、prompt audit 和 passthrough URL 构造均加入 `/antigravity/v1beta/openai/...`；本地 `AntigravityCompat`/`OpenAICompat` 测试覆盖 endpoint normalize、protocol capability、prompt audit、URL 去本地前缀和 base URL 防双前缀。 |
| `fix(grok): pause accounts after manual test payment failure` / Grok 402 | 已重写引入 | Grok official 手动测试真实响应遇到 `>=400` 会调用现有 `HandleUpstreamError` 链路；`Grok 402` 复用本地上游错误策略暂停账号，不新增平行状态处理。 |
| `fix(gemini): preserve Hermes web search functions` | 本地已覆盖 | 本地 apicompat、Antigravity 转换、分组定价和前端配置已有 `web_search` / `web_search_price_per_call` 链路。 |
| `fix(usage): correct mapped model statistics` | 本地已覆盖 | 用量映射支持 requested/display model 与 mapped model 分离统计；本轮保留本地 usage 聚合实现。 |
| `fix(usage): preserve final upstream model` | 本地已覆盖 | 用量记录、DTO、仓储和服务层已有 `UpstreamModel` 元数据，支持 final upstream model 留存和统计。 |
| `Codex++ Responses<->Anthropic compatibility fixes` / Responses/Anthropic compatibility | 已重写引入并补测试 | 本地 `apicompat` 覆盖 Responses/Anthropic 与 Chat bridge：`namespace`、`custom`、`tool_search` proxy，`function_call_output` 数组输入，空 `input_schema` 规范化，Responses Lite `additional_tools` lift/restore/去重。新增 `AnthropicCompat`、`ResponsesCompat`、`AdditionalTools` 回归测试。 |
| `fix(frontend): 修复分组描述换行和下拉框溢出` | 本地已覆盖 | 分组描述、选择器和设置页已有响应式/换行处理；本轮不做无关全站视觉改造。 |
| `fix(frontend): 完善下拉框视口边界处理` | 本地已覆盖 | 现有选择控件保持本地布局与边界处理；本轮不引入上游 UI 代码。 |
| `fix(frontend): 修复渠道监控时间线在窄卡片下溢出` | 本地已覆盖 | 渠道监控和可用渠道视图已有本地窄屏溢出处理；若后续发现具体页面仍溢出，应作为独立前端任务处理。 |
| `fix/issue-4863-turnstile-invite-overlap` | 本地已覆盖 | 注册流程已区分邀请码、Turnstile/验证码和可选 `aff_code`；本轮不改变现有邀请码/优惠码/人机验证流程。 |
| `chore: sync VERSION to 0.1.165 [skip ci]` | 排除：发布材料 | 本地版本保持 `0.1.411`，不回退到 `0.1.166` 或 `0.1.165`。 |
| compare merge commits / PR merge commits | 排除：Git 历史材料 | 不执行 merge/cherry-pick，不复制合并提交元数据；只用标题做主题审计。 |
| `chore: update sponsors` / sponsor partner assets | 排除：资产材料 | 不引入上游赞助/partner 资产变更；release guard 检查矩阵留痕并禁止引入 sponsor/partner 资产主题。 |

## 验收与回滚

- 面板限流默认关闭时，现有面板请求行为不变。
- 开启面板限流后，认证用户按 `user_id`，公开接口按可信公网 IP，heavy 聚合接口按配置限流；Redis 故障不阻断面板请求。
- `CONFIG_FILE` 指向临时配置文件时优先生效，未设置时保持旧搜索路径。
- SSE 响应的 `text/event-stream` 不进入 Caddy 压缩匹配。
- Claude 默认 UA 为 `claude-cli/2.1.220`，并且代理改写 UA 时可由 Claude Code body detection 识别以保留 prompt cache。
- composite prefix passthrough 在 `target_model_id` 为空时透传请求模型。
- Antigravity OpenAI-compatible native gateway 只选择 Antigravity API key 账号，保持 `/openai/...` native passthrough，不混用普通 Gemini/OpenAI。
- release guard 检查 MIT-only、无 `/api-docs/*` 或 `/admin/api-docs/*`、无上游 LGPL/GPL/CLA 协议文本、版本未回退到 `0.1.166`、未引入上游赞助/partner 资产变更，并守卫矩阵关键主题存在性。

## 本地验证命令

```powershell
Push-Location backend
go test ./internal/middleware ./internal/server/middleware ./internal/server/routes ./internal/handler/admin ./internal/service ./internal/config ./internal/repository ./internal/securityaudit ./internal/domain ./internal/modelregistry -run "RateLimiter|PanelRate|AuthRate|Setting|Config|Claude|ClaudeCode|Grok|Gemini|Composite|Probe|ReasoningFailover|AnthropicCompat|AntigravityCompat|OpenAICompat|Usage|Payment|Prompt|ReleaseGuard|Pricing|ModelRegistry" -count=1
Pop-Location
pnpm --dir frontend typecheck
pnpm --dir frontend test:run
git diff --check
```
