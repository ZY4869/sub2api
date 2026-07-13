# Upstream v0.1.151-v0.1.152 Cleanroom Sync Matrix

Local baseline: `v0.1.386`. Upstream reference is the official `Wei-Shaw/sub2api` remote tag `v0.1.152` at `553ab6f911247963eb368fcf6ac1dcb65d5495b1`. This local sync is behavior-only.

Cleanroom rules:

- No `git pull`, merge, rebase, or cherry-pick from `Wei-Shaw/sub2api`.
- Do not copy upstream LGPL/GPL source text, license text, README/governance text, release metadata, route files, tests, or migration numbering.
- Keep the root `LICENSE` MIT-only and keep local version metadata at `0.1.386`.
- Do not add `/api-docs/*` or `/admin/api-docs/*` routes, pages, handlers, or runtime overrides.
- Use local migration `158_add_group_web_search_price_per_call.sql`; do not adopt upstream migration numbering.

| Upstream behavior anchor | Local cleanroom result | Evidence | Status |
| --- | --- | --- | --- |
| OpenAI-compatible `alpha/search` endpoint | Added native OpenAI gateway dispatch for `POST /v1/alpha/search` and `/alpha/search`, using existing authentication, account selection, upstream forwarding, request tracing, and error passthrough. Non-OpenAI runtimes are rejected through the protocol capability matrix. | `backend/internal/server/routes/gateway.go`, `gateway_route_dispatchers.go`, `backend/internal/handler/openai_gateway_handler_search.go`, `backend/internal/service/openai_gateway_alpha_search.go`, `protocol_capability_registry.go`, `protocol_capability_matrix_test.go` | Closed |
| Per-call web search billing | Successful strict `2xx` alpha search responses create a per-request usage log with model `alpha/search`; upstream failure, forwarding failure, missing account, and non-`2xx` responses do not bill. Default price is `0.01` USD per call; nil and negative group price use the default, `0` is free, and positive values override. Basic group/user multipliers apply without peak multiplier. | `backend/internal/service/openai_gateway_search_usage.go`, `openai_gateway_search_usage_test.go`, `backend/internal/handler/openai_gateway_handler_search.go` | Closed |
| Group-scoped search price contract | `groups.web_search_price_per_call` stores the nullable `web_search_price_per_call` field and flows through ent, repository, service models, admin DTOs, API key auth snapshots, and frontend group create/edit forms. Update requests distinguish omitted fields from explicit `null`, so clearing the override is preserved. | `backend/migrations/158_add_group_web_search_price_per_call.sql`, `backend/ent/schema/group.go`, `backend/internal/repository/group_repo.go`, `backend/internal/service/admin_service_groups.go`, `backend/internal/service/api_key_auth_cache.go`, `backend/internal/handler/admin/group_handler.go`, `frontend/src/views/admin/GroupsView.vue`, `frontend/src/views/admin/groups/GroupCreateDialog.vue`, `GroupEditDialog.vue` | Closed |
| Responses tool compatibility fixes | Responses tools support `custom`, `tool_search`, namespace tools, and string shorthand. Namespace flattening detects name conflicts, `tool_choice` is forwarded only when the chosen target converts successfully, `tool_search` params objects decode correctly, and chat-completions response conversion restores Responses-style output items. | `backend/internal/pkg/apicompat/responses_tool_proxy.go`, `responses_to_chatcompletions_request.go`, `chatcompletions_to_responses_response.go`, `types.go`, `chatcompletions_responses_test.go` | Closed |
| Responses to Anthropic streaming cache token passthrough | Anthropic streaming conversion preserves `cache_creation_input_tokens` when present in Responses usage payloads. | `backend/internal/pkg/apicompat/responses_to_anthropic.go`, `anthropic_responses_test.go` | Closed |
| Grok xAI API key and OAuth routing | Existing Grok integration remains fused instead of duplicated: API key and OAuth accounts use official xAI-style `Authorization: Bearer` requests and `/v1/responses` endpoints, while SSO stays on its separate reverse route. Prompt-cache unsupported fields remain stripped for API key forwarding, and supported reasoning fields remain intact. | `backend/internal/service/grok_gateway_service_apikey.go`, `grok_gateway_service.go`, `grok_gateway_service_sso.go`, `grok_gateway_responses_test.go`, `account_test_service_grok.go`, `account_model_import_probe_grok.go` | Closed |
| Fast/Flex user-scoped policy UI | The settings card keeps manual `user_ids` entry and adds a searchable user selector backed by the existing admin user search API. Save behavior merges, deduplicates, and validates positive user IDs, preserving old configurations. | `frontend/src/components/settings/OpenAIFastPolicySettingsCard.vue`, `OpenAIFastPolicyUserSelector.vue`, `frontend/src/components/settings/__tests__/OpenAIFastPolicySettingsCard.spec.ts`, `frontend/src/i18n/locales/en/admin/settings/gateway.ts`, `frontend/src/i18n/locales/zh/admin/settings/gateway.ts` | Closed |
| Ops capture writer release safety | Error and request-trace capture writers use nil-safe delegate helpers for `Status`, `Size`, `Written`, `WriteHeader`, `Header`, `Flush`, `Hijack`, `CloseNotify`, `Pusher`, `Write`, and `WriteString`, keeping original writer restoration behavior while preventing post-release panics. | `backend/internal/handler/ops_capture_writer_delegate.go`, `ops_error_capture_writer.go`, `ops_request_trace_capture_writer.go`, `ops_error_logger_test.go` | Closed |
| Release, license, and docs-center exclusions | Upstream license/README/governance/release metadata are excluded. Root `LICENSE` remains MIT, local version remains `0.1.386`, and `/api-docs/*` plus `/admin/api-docs/*` stay forbidden. | `backend/internal/repository/release_guard_test.go`, this matrix | Guarded |

Validation commands:

- `go test ./internal/service ./internal/server/routes ./internal/handler ./internal/pkg/apicompat ./internal/repository`
- `pnpm --dir frontend test:run OpenAIFastPolicySettingsCard GroupsView`
- `pnpm --dir frontend typecheck`
- `rg "/api-docs|admin/api-docs" .`
- `git diff -- LICENSE README*`
