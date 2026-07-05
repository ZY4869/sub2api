# Upstream v0.1.144 Cleanroom Sync Matrix

## Boundary

- Source range: `Wei-Shaw/sub2api` tag `v0.1.144^{commit}` at `a10dc955189e9c7d70dcbb0f0a6334b2932bbde0`, plus the user-approved follow-up main version-sync commit when it is metadata-only.
- Local baseline: `0.1.375`; do not downgrade local version metadata to upstream `0.1.144`.
- Explicit exclusions: `v0.1.145+`, upstream LGPL license text, README license wording changes, sponsor assets, workflow/governance files, and any direct pull/merge/cherry-pick.
- License result: MIT-only boundary is preserved. Root `LICENSE` and README license sections remain MIT.
- Project rules: no `/api-docs/*` or `/admin/api-docs/*`; model policy remains `extra.model_scope_v2.entries[]`; external model enumeration must not expose internal `target_model_id`.

## Completion Matrix

| Upstream behavior | Local cleanroom result | Evidence | Status |
| --- | --- | --- | --- |
| Concurrent usage log overflow must not silently lose records | Default overflow behavior uses `UsageRecordOverflowPolicySync`, preserving legacy drop/sample config compatibility. | `backend/internal/config/config_defaults.go`, `backend/internal/service/usage_record_worker_pool.go` | Implemented |
| Anthropic Fable `7d_oi` quota and usage | Passive/active usage records `seven_day_fable`; model-scoped rate limit uses the Anthropic `7d_oi` headers. | `account_usage_*`, `ratelimit_anthropic.go`, `ratelimit_session_window.go` | Implemented |
| Admin/user failed request list improvements | Admin ops and user usage error views support sorting, status/category filters, compatible old parameters, and configurable columns. | `ops_handler.go`, `usage_handler.go`, `OpsErrorLogTable.vue`, `FailedRequestsPanel.vue` | Implemented |
| Codex image tool policy | OpenAI OAuth accounts support `codex_image_tool_policy`: follow channel, force inject, no inject, block all. | `openai_codex_transform.go`, `AccountGatewaySettingsEditor.vue`, `accountCreateExtras.ts` | Implemented |
| OpenAI billing mapped-model fix | Local default remains request-model billing; mapped billing model is set only where mapping semantics require it. | OpenAI gateway forward/service tests | Implemented |
| Codex session import | Admin endpoint `codex-sessions/import` creates/updates OpenAI OAuth accounts, dedupes identity keys, and preserves existing refresh credentials on access-token-only import. | `account_codex_import_*.go`, `CodexSessionImportModal.vue` | Implemented |
| Setup migration timeout | Migration timeout has default/configurable behavior and tests. | `backend/internal/setup/setup.go`, `setup_test.go` | Implemented |
| Concurrency cleanup and ops summaries | Concurrency slot cleanup and group/account capacity summaries are available for ops surfaces. | `concurrency_*`, `group_capacity_*`, `ops_*` | Implemented |
| Token/OAuth and platform fixes | `token_expired` non-retry, Antigravity OAuth 401 recovery, Gemini 3.1 Pro mapping, Grok mapping update, and invitation redeem error UX are covered by targeted tests. | service tests and domain constants tests | Implemented |
| Later upstream tags | `v0.1.145+` behavior is intentionally not evaluated or absorbed in this round. | Scope rule above | Excluded |

## Observability And Security

- Usage worker exposes dropped/sync fallback counters and structured warning logs without request bodies or credentials.
- Codex image policy blocks record an ops upstream error event with policy metadata only.
- Codex session import runs through admin auth and write idempotency; result summaries expose counts and item statuses, not raw tokens.
- Fable rate-limit handling records model-scope limit state and does not disable the entire account.
- OAuth recovery and token refresh paths must not log access tokens, refresh tokens, session IDs, or PII.

## Regression Evidence

- Backend targeted service tests: usage, rate limit, OpenAI gateway, token refresh, Antigravity, Grok, redeem, concurrency, group capacity.
- Backend targeted handler/repository/setup tests: usage failed requests, Codex import, ops error query, setup migration timeout.
- Frontend targeted Vitest: account usage cell, Codex image policy extras, Codex import dialog host/toolbar wiring, user usage failed requests.
- Frontend typecheck: `npm --prefix frontend run typecheck`.
- Backend full regression: `go test ./...`.
- Frontend production build: `npm --prefix frontend run build`.
- Whitespace guard: `git diff --check`.
- License/version guard: `git diff -- LICENSE README.md README_CN.md README_EN.md backend/cmd/server/VERSION` is empty.
- Docker Compose config check: WSL `docker compose config -q` under `deploy/` succeeds.
- License guard: root `LICENSE`, README license sections, and version files remain unchanged and MIT-only.

## Open Release Tasks

- Docker Compose manual smoke remains blocked: an existing local `deploy` compose project is present, but the `sub2api` container is restarting and was not recreated from this worktree during this cleanroom sync.
- Keep this matrix as the audit source for v0.1.144; create a separate matrix for any future `v0.1.145+` absorption.
