# 上游 v0.1.143 洁净重写同步矩阵

## 合规边界

- 参考对象：仅参考 `Wei-Shaw/sub2api` 上游 v0.1.143 的行为语义和修复目标。
- 操作边界：未执行 `git pull`、merge、cherry-pick；未复制上游 LGPL 协议文本、文档、资产或代码片段。
- 本地保护：根目录 `LICENSE` 保持 MIT；本地版本线保持 `0.1.374`；本仓库同名早期版本不作为参考来源。
- 冲突策略：本地功能优先；重复能力按本地架构融合；公开模型示例仅在路径、认证、alias 或错误响应示例变化时更新。

## 同步矩阵

| 上游语义 | 本地洁净重写点 | 测试/审计状态 | 协议保护结论 |
| --- | --- | --- | --- |
| 分组高峰倍率 | 新增 `peak_rate_enabled`、`peak_start`、`peak_end`、`peak_rate_multiplier`，订阅分组生效，标准分组保存时清空；DTO 返回服务器时区与 UTC offset。 | 已补后端 normalization/active window/token multiplier 测试；已补前端列设置、标准分组清空和非法时间拦截测试；目标回归通过。 | 自主实现字段、校验和 UI，未复制上游实现。 |
| token 计费倍率 | 新增统一 token 倍率入口；高峰倍率只影响 token 计费和按 token 计费图片，不影响按张计费图片。 | 已修复未显式传倍率时 `ActualCost=0` 的回归；已补倍率默认值和 token/flat unit scope 测试；目标回归通过。 | 保持本地计费服务结构。 |
| OpenAI WS `http_bridge` | 融入账号级 WS v2 模式；显式选择时走 HTTP/SSE 桥接；默认模式不变；配置加入启用开关和大首包阈值。 | 已通过后端 service 目标回归和前端 `openaiWsMode` Vitest。 | 本地协议解析与桥接实现，未导入上游代码。 |
| 订阅恢复 | 新增管理员恢复接口与前端确认动作；仅恢复 revoked/soft-deleted 订阅，恢复后按过期时间得到 active/expired。 | 已补 service active/expired/未撤销/冲突测试和前端成功/失败确认测试；handler 目标回归通过。 | 复用本地软删除、缓存刷新和订阅生命周期。 |
| IP 归属 | 新增后端代理开关、批量查询接口、缓存和后台用量页展示；默认关闭；前端不直连 GeoJS。 | 已补 disabled/private/invalid/too many/provider ok/not_found/error/cache 测试；失败路径记录脱敏结构化日志；前端唯一 IP 查询测试通过。 | 仅保存服务商 URL 配置，未复制外部数据或上游资产。 |
| Anthropic Bearer | 账号 extra 增加 `anthropic_apikey_auth_scheme`，默认 `x-api-key`；选择 Bearer 时仅发送 `Authorization: Bearer`。 | 已通过后端 Anthropic passthrough/header 目标回归；已补前端 extra 生成测试。 | 改的是上游账号到供应商鉴权，不改变站内公开 API 认证示例。 |
| 后台分组列设置 | 分组列表增加本地持久化列显示设置，名称和操作列固定显示。 | 已补前端列持久化与固定列测试；`pnpm typecheck` 通过。 | 新增本地 UI 状态，无上游代码复制。 |
| `count_tokens` fallback | 洁净重写 OpenAI/Anthropic 兼容 fallback，失败时结构化记录。 | 已通过后端 service/handler 目标回归。 | 保持本地 gateway/service 封装。 |
| Claude Code stream keepalive | 对流式响应补齐 keepalive 行为，降低空闲断连。 | 已通过后端 service 目标回归。 | 本地流处理逻辑内实现。 |
| `/responses/compact` 图片桥绕过 | compact 请求跳过图片桥注入，避免错误重写。 | 已通过后端 service/handler 目标回归。 | 本地请求规范化路径内实现。 |
| Gemini/Antigravity 参数清理 | 清理无效参数，避免上游拒绝。 | 已通过后端 service 目标回归。 | 本地请求转换层实现。 |
| 用户模型统计按 requested model 分组 | 统计按用户请求模型聚合，避免被上游映射污染。 | 已通过 repository 目标回归。 | 本地统计 SQL/Repo 实现。 |
| Claude OAuth token exchange | 移除多余 `expires_in` 参数。 | 已通过 repository 目标回归。 | 本地 OAuth service 实现。 |
| OpenAI inactive/expired plan 覆盖 | 防御 inactive workspace plan 与过期订阅覆盖问题。 | 已通过后端 service 目标回归。 | 本地账号计划解析实现。 |
| Grok 媒体默认与图片 alias | 默认启用 Grok 媒体能力并归一化图片 alias。 | 已通过后端 service 目标回归。 | 本地模型身份映射实现。 |
| 模型示例模板审计 | 审计 `backend/internal/service/model_catalog_public_example_templates.go`、后端示例选择测试和前端示例构造测试。 | 本次无需改模板：公开路径、站内 Bearer 认证、alias 示例和错误响应示例未因本次改动变化。 | 遵守 AGENTS 规则，未新增 `/api-docs/*` 或 `/admin/api-docs/*`。 |

## 本轮回归

- 后端定向：`go test ./internal/service -run "TestBillingCenterService_SimulateAndRuntimeShareLegacyFallback|TestNormalizeExplicitRateMultiplierDefaultsNonPositiveValues|TestBillingLineActualMultiplierScopesTokenAndFlatUnits|TestNormalizeGroupPeakRateConfig|TestGroupEffectiveTokenRateMultiplierAt|TestAdminServiceLookupUsageIPGeo|TestSubscriptionServiceRestoreSubscription" -count=1 -v` 通过。
- 后端目标包：`go test ./internal/service ./internal/handler ./internal/repository` 通过。
- 前端定向：`pnpm test:run src/views/admin/__tests__/SubscriptionsView.spec.ts src/views/admin/__tests__/UsageView.spec.ts src/views/admin/__tests__/GroupsView.spec.ts src/utils/__tests__/accountCreateExtras.spec.ts src/utils/__tests__/openaiWsMode.spec.ts` 通过。
- 前端类型：`pnpm typecheck` 通过。
- 合规复核：根 `LICENSE` 保持 MIT；后端/前端版本保持 `0.1.374`；未引入上游 LGPL 协议文本、文档、资产或代码片段。

## 发布兼容性

- 新增能力均向后兼容。
- 高峰倍率默认关闭；标准分组保存会清空高峰配置。
- IP 归属默认关闭，且仅管理员后台接口可用。
- Anthropic API Key 账号默认仍使用 `x-api-key`。
- OpenAI WS 默认模式不变，`http_bridge` 只在显式选择或本地阈值保护命中时生效。
