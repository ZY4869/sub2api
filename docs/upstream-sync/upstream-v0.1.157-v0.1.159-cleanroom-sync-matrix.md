# Upstream v0.1.157-v0.1.159 Cleanroom Sync Matrix

Local baseline: current local workspace (`0.1.396`). Upstream reference is `Wei-Shaw/sub2api` compare range `v0.1.156...v0.1.159`; local same-name tags, if present, are intentionally ignored.

Cleanroom rules:

- No `git pull`, `fetch`, merge, rebase, or cherry-pick from `Wei-Shaw/sub2api`.
- Do not copy upstream LGPL/GPL source text, license text, README/governance text, sponsor assets, release metadata, generated files, or migration numbering.
- Keep the root `LICENSE` MIT-only, keep README license statements MIT, and keep local version metadata unchanged.
- Do not add `/api-docs/*` or `/admin/api-docs/*` routes, pages, handlers, or runtime overrides.
- Keep async image task semantics inside local `/v1/images/batches`; do not add an upstream-shaped `/images/tasks/:task_id` service tree.

Alpha/search history note: upstream first introduced the standalone `alpha/search`
surface in v0.1.152 (`52071d391`), fixed embedded-frontend bypass handling in
v0.1.153, and added the PAT-to-Responses `web_search` fallback in v0.1.157
(`695665cbc`). The local implementation keeps the cleanroom boundary and adds a
default-on system switch plus an OpenAI-account switch; PAT fallback uses the
existing Responses transport and does not persist a credential error for a
request-scoped 401.

| Upstream behavior anchor | Local cleanroom result | Evidence | Status |
| --- | --- | --- | --- |
| Gateway bugfix sweep | Local gateway tests and services cover WS v2 illegal event rejection, ingress read close handling, Responses Lite context normalization, rejected-field retry, model-scoped transient cooldown, body-limit failover, alpha/search PAT/API key dispatch, OAuth image-tool billing, image intent Responses-capable gating, Claude Haiku mimicry, Grok WS v2/free-cache/endpoint regressions, and trusted client IP. | Gateway service, handler, and package tests across `backend/internal/service`, `backend/internal/handler`, and `backend/internal/pkg` | Guarded by local behavior tests |
| Billing and account capability snapshot | `GET /v1/sub2api/billing` is retained. Admin billing probe settings, single-account probe, and bulk probe persist local snapshots in account extra metadata; scheduling reads local snapshots and does not synchronously probe on read paths. | `backend/internal/handler/gateway_key_billing.go`, `backend/internal/handler/admin/account_billing_probe.go`, `backend/internal/service/setting_service_billing_probe.go`, billing probe tests | Fused |
| Payment and usage accounting | Local `prices_by_currency` remains the plan pricing source. Rebate-on-topup settings are retained. `image_input_price`, `image_input_tokens`, and `image_input_cost` are carried through migrations, ent fields, usage logging, billing center pricing, and frontend usage display. | `backend/migrations/160_add_usage_log_image_input_fields.sql`, usage/billing service files, payment settings files | Fused |
| Security and audit retention | Audit logs use local repository/service/handler layers. Sensitive admin operations record success/failure/denied events with redacted metadata. `audit_log_retention_days` defaults to 180 days, is configurable through admin settings, and cleanup of expired audit logs requires `X-Sub2API-Step-Up-TOTP`. | `backend/internal/service/audit_log.go`, `backend/internal/handler/admin/audit_log_handler.go`, `frontend/src/api/admin/stepUp.ts`, audit retention tests | Cleanroom rewrite |
| Auth session binding | Refresh-token sessions bind to trusted client IP and User-Agent hashes. Legacy unbound refresh tokens remain accepted once, then rotate into bound sessions; mismatches revoke the token family. | `backend/internal/service/auth_session_binding.go`, `backend/internal/server/middleware/auth_session_binding.go`, auth refresh binding tests | Cleanroom rewrite |
| Step-up protected sensitive actions | Existing admin sensitive actions retain step-up behavior: account/proxy export, backup download URL and S3 target changes, data-management S3 profile changes, and administrator create/promote. The shared frontend step-up header wrapper is reused by user admin actions and audit cleanup. | `backend/internal/handler/admin/admin_security.go`, admin handlers, `frontend/src/api/admin/stepUp.ts` | Fused |
| Image async task semantics | Upstream task states map into local batch job statuses: `submitted`, `running`, `completed`, `failed`, and `cancelled`. Provider task IDs remain internal provider metadata and public responses expose local batch IDs only. Polling, cancellation, output deletion, and friendly errors stay on `/v1/images/batches`. | `backend/internal/service/image_batch_types.go`, `backend/internal/service/image_batch_worker.go`, `backend/internal/service/image_batch_service_test.go` | Local architecture preserved |
| Object storage and fallback | Image batch outputs use the local output repository and configured cleanup/download limits. S3-related sensitive target changes remain step-up protected through backup/data-management settings; missing object-store configuration falls back to local safe storage instead of exposing provider internals. | `backend/internal/repository/image_batch_repo_worker.go`, `backend/internal/repository/image_batch_repo_items.go`, backup/data-management handlers | Locally equivalent |
| Admin UX additions | User platform quota bulk update, group copy, channel monitor copy, API key account-name upstream link, Grok upstream endpoint shortcuts, custom request headers, and billing probe settings remain in local admin surfaces. | Admin handlers, routes, and Vue components under `frontend/src/views/admin` and `frontend/src/components/settings` | Already present |
| Chinese account usage window | The account usage-window area uses real zh locale strings for title, snapshot time, used/remaining labels, and OpenAI reset-credit count. Regression tests forbid `Usage Windows`, `Snapshot`, `Used`, and `resets left` in the targeted zh labels. | `frontend/src/i18n/locales/zh/admin/accounts/common/overview.ts`, `frontend/src/i18n/locales/zh/admin/accounts/formAndFilters.ts`, `frontend/src/i18n/__tests__/zhAudit.spec.ts` | Guarded |
| Upstream license/README/version/release metadata | Upstream license and release metadata are excluded. Root `LICENSE`, README license text, and local version remain protected by release guard tests. | `LICENSE`, `README*.md`, `backend/cmd/server/VERSION`, `backend/internal/repository/release_guard_159_test.go` | Guarded |
| Standalone API docs center | Still forbidden; no `/api-docs/*` or `/admin/api-docs/*` routes/pages/handlers are added. | `backend/internal/repository/release_guard_159_test.go`, existing router tests | Guarded |
| OpenAI alpha/search compatibility | `/v1/alpha/search`, `/alpha/search`, and `/backend-api/codex/alpha/search` share OpenAI-only routing, capability scheduling, per-call billing, and a default-on global/account toggle. Alpha/search is selected as the `OpenAIEndpointCapabilityAlphaSearch` protocol capability: direct OpenAI groups bypass ordinary API-key/group model visibility patterns, while composite groups resolve only from the current request `model`. OAuth uses ChatGPT Codex alpha/search; API keys use `/v1/alpha/search`; PATs fall back to Codex Responses `web_search` with metadata validation and SSE-to-JSON conversion. Invalid non-SSE `2xx` responses are treated as upstream protocol failover and are not billed. | `backend/internal/service/gateway_group_selector.go`, `backend/internal/service/gateway_group_selector_alpha_search_test.go`, `backend/internal/service/openai_gateway_alpha_search.go`, `backend/internal/service/openai_account_scheduler_capability_test.go`, `backend/internal/handler/admin/setting_handler_general_test.go`, `backend/internal/server/routes/gateway_test.go`, `frontend/src/utils/__tests__/accountCreateExtras.spec.ts` | 已验证（本地自动化）；真实 OpenAI/ChatGPT 网络请求仍需脱敏账号手工验收 |

Validation commands:

- `go test ./internal/service ./internal/handler ./internal/handler/admin ./internal/repository ./internal/server ./internal/pkg/... -count=1`
- `pnpm --dir frontend typecheck`
- `pnpm --dir frontend test:run`
- `git diff -- LICENSE README.md README_CN.md README_EN.md backend/cmd/server/VERSION --`
- `rg "/api-docs|admin/api-docs" backend frontend`
