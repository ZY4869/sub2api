# Upstream v0.1.152-v0.1.153 Cleanroom Sync Matrix

Local baseline: `v0.1.386`. Upstream reference is the official `Wei-Shaw/sub2api` release `v0.1.153` at `a2bc133` on 2026-07-13. This sync is behavior-only and ignores any local tag with the same version number.

Cleanroom rules:

- No `git pull`, merge, rebase, or cherry-pick from `Wei-Shaw/sub2api`.
- Do not copy upstream LGPL/GPL source text, license text, README/governance text, release metadata, route files, tests, or migration numbering.
- Keep the root `LICENSE` MIT-only and keep local version metadata at `0.1.386`.
- Do not add `/api-docs/*` or `/admin/api-docs/*` routes, pages, handlers, or runtime overrides.
- Use local migration `159_add_usage_logs_api_key_recent_ip_index_notx.sql`; do not adopt upstream migration `174`.
- Optional Apple Container support must default to local `ZY4869/sub2api` / `ghcr.io/zy4869/sub2api` namespace, not the upstream namespace or image.

| Upstream behavior anchor | Local cleanroom result | Evidence | Status |
| --- | --- | --- | --- |
| Grok video edits and extensions | Fused into the existing Grok gateway video workflow as native endpoints: `/v1/videos/edits`, `/v1/videos/extensions`, root aliases, and `/grok/v1` aliases. They reuse local API key auth, request tracing, group routing, media forwarding, content audit, concurrency, billing, failover, and usage recording. Non-Grok protocols remain rejected by the capability matrix. | `backend/internal/service/grok_gateway_video_request.go`, `grok_gateway_video_workflow.go`, `grok_gateway_service_apikey.go`, `grok_gateway_service_sso.go`, `backend/internal/handler/endpoint.go`, `backend/internal/server/routes/gateway.go`, `protocol_capability_registry.go`, `protocol_capability_matrix_test.go`, `grok_gateway_video_workflow_test.go` | Fused |
| Public model examples for new video paths | Model-detail examples were extended in the local template source, not by adding a docs center. | `backend/internal/service/model_catalog_public_endpoints.go`, `backend/internal/service/model_catalog_public_example_templates.go` | Fused |
| Apple Container deployment | Added an optional local script and document. Defaults point to `ZY4869/sub2api` and `ghcr.io/zy4869/sub2api:latest`; secrets are accepted only through environment variables. Default deployment guidance remains WSL + Docker Compose. | `deploy/apple-container.sh`, `deploy/APPLE_CONTAINER.md`, `backend/internal/repository/release_guard_test.go` | Cleanroom rewrite |
| OpenAI OAuth `plan_type` manual override | Edit modal now shows the override only for OpenAI OAuth accounts, loads `credentials.plan_type`, trims it on save, and deletes it when cleared. API key accounts do not show the field. | `frontend/src/components/account/EditAccountModal.vue`, `editAccountModal/useEditAccountModal.ts`, `watchers.ts`, `submit.ts`, `credentialsBuilder.ts`, `EditAccountModal.spec.ts`, `credentialsBuilder.spec.ts`, zh/en form locale files | Cleanroom rewrite |
| API key recent IP query index | Added local notx migration `159` using the actual local `usage_logs.ip_address` column and concurrent partial index for `(api_key_id, created_at DESC, ip_address)`. | `backend/migrations/159_add_usage_logs_api_key_recent_ip_index_notx.sql`, `backend/internal/repository/release_guard_test.go` | Cleanroom rewrite |
| Embedded static resource caching and SPA bypass | `index.html` and SPA fallback stay `no-cache`; hashed assets receive long immutable cache headers; other static assets receive short cache headers; `/alpha/search`, `/v1/*`, and `/grok/*` bypass SPA fallback. | `backend/internal/web/embed_on.go`, `embed_test.go` | Cleanroom rewrite |
| Scheduler bad `last_used_at` handling | Future or otherwise abnormal `last_used_at` values are normalized before LRU sorting/grouping so bad data cannot block account selection indefinitely. | `backend/internal/service/gateway_account_selection_sort.go`, `scheduler_layered_filter_test.go`, `scheduler_shuffle_test.go` | Cleanroom rewrite |
| Pool-mode same-account retry coverage | Existing local retry machinery remains fused across Anthropic/Gemini/general paths; matrix notes the behavior is covered by local scheduler/retry tests and not reimplemented from upstream. | `backend/internal/service/antigravity_single_account_retry_test.go`, `gateway_anthropic_apikey_passthrough_test.go`, `account_test_service_*` | Already present / fused |
| OpenAI WebSocket inbound lifetime cap | Added configurable max session lifetime with default `14400` seconds and relay watchdog close path. | `backend/internal/config/config_types_gateway.go`, `config_defaults.go`, `config_validate.go`, `config_test.go`, `backend/internal/service/openai_ws_v2/passthrough_relay.go`, `passthrough_relay_test.go`, `openai_ws_metrics.go`, `openai_ws_v2_passthrough_adapter.go` | Cleanroom rewrite |
| Grok API Key third-party base URL and model sync | Local Grok model import uses the account `base_url` after local URL policy validation and calls `/v1/models` with bearer auth. | `backend/internal/service/account_model_import_probe_grok.go`, `account_model_import_service_test.go` | Guarded / cleanroom test |
| `/alpha/search` embedded frontend bypass | Included with static fallback bypass so gateway endpoints are not swallowed by the SPA. | `backend/internal/web/embed_on.go`, `embed_test.go` | Cleanroom rewrite |
| Codex additional/custom tools and namespace tools | Already covered by the previous v0.1.152 sync; no duplicate implementation was added. | `backend/internal/pkg/apicompat/responses_tool_proxy.go`, `responses_tool_proxy_chat.go`, `responses_tool_proxy_output.go`, `chatcompletions_responses_test.go` | Already present |
| Anthropic streaming stop reason compatibility | Local conversion tests cover Responses to Anthropic streaming behavior, including content-filter/cache-token compatibility from the prior sync line. | `backend/internal/pkg/apicompat/responses_to_anthropic.go`, `anthropic_responses_test.go` | Already present / guarded |
| Read tool parameter streaming passthrough | Responses tool proxy keeps tool params in the local conversion layer rather than introducing a separate upstream adapter. | `backend/internal/pkg/apicompat/responses_tool_proxy*.go` | Already present |
| Usage date timezone consistency | Local usage/stat paths already route through configured `TZ` / date formatting utilities and existing usage tests; no new schema change required. | `backend/internal/handler/admin/usage_handler.go`, frontend KeyUsage date formatting tests where present | Already present |
| Deprecated payment API removal | No deprecated payment endpoint that leaks internal provider/channel configuration is exposed. Formal payment routes remain: order create/get/resume/cancel, webhook, admin order list, admin refund, and ops metrics. | `backend/internal/server/routes/user.go`, `backend/internal/server/routes/admin.go`, `frontend/src/api/payment.ts`, `backend/internal/repository/release_guard_test.go` | Excluded / guarded |
| Missing zh/en locale parity | New user-visible `plan_type` labels were added to both zh and en locale files in the same change. | `frontend/src/i18n/locales/zh/admin/accounts/common/form.ts`, `frontend/src/i18n/locales/en/admin/accounts/common/form.ts` | Cleanroom rewrite |
| DataTable small data jitter | Small desktop datasets render directly instead of virtualizing; virtualized row height estimates use stable row keys with measured height cache. | `frontend/src/components/common/DataTable.vue`, `DataTable.spec.ts` | Cleanroom rewrite |
| Upstream license and version metadata | Upstream LGPL/COPYING/CLA/README/release metadata are excluded. Root `LICENSE` remains MIT; local version remains `0.1.386`; no `/api-docs/*` or `/admin/api-docs/*` was introduced. | `LICENSE`, `backend/cmd/server/VERSION`, `backend/internal/repository/release_guard_test.go`, this matrix | Guarded |

Validation commands:

- `go test ./internal/service -run "TestImportAccountModels_GrokAPIKeyUsesConfiguredBaseURL|TestGrokBuildVideo|TestProtocol|TestOpenAIWS|TestRelay|TestSort|TestSelectByLRU"`
- `go test ./internal/handler -run "TestNormalizeInboundEndpoint|TestDeriveUpstreamEndpoint"`
- `go test ./internal/server/routes -run TestGatewayRoutes`
- `go test ./internal/repository -run "TestUpstream152To153CleanroomMatrixGuards|TestMigration"`
- `go test -tags embed ./internal/web`
- `pnpm --dir frontend test:run DataTable EditAccountModal credentialsBuilder`
- `git diff -- LICENSE README.md README_CN.md README_EN.md backend/cmd/server/VERSION`
- `rg "LGPL|GNU LESSER|COPYING|CLA.md|Wei-Shaw/sub2api" .`
