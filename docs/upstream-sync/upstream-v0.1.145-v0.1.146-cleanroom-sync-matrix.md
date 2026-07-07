# Upstream v0.1.145-v0.1.146 Cleanroom Sync Matrix

## Boundary

- Source range: `Wei-Shaw/sub2api` tags `v0.1.145` at `b023f1a507039c620c12a07f22e8d7e2a0f70df5` and `v0.1.146` at `3aee00f59fc2cb6c46e9ee7631b9d799a736b596`.
- Local baseline: `0.1.376`; do not downgrade local version metadata to upstream `0.1.145` or `0.1.146`.
- Cleanroom rule: do not run `git pull`, merge, rebase, cherry-pick, or copy upstream LGPL code/text. Use upstream behavior and bug evidence only.
- Explicit exclusions: upstream LGPL/GPL license text, README license wording changes, sponsor/governance files, release workflow churn, `/api-docs/*`, `/admin/api-docs/*`, and any version rollback.
- License result: MIT-only boundary is preserved. Root `LICENSE` and README license sections remain MIT.
- API contract anchors: `current_concurrency`, `/responses/compact`, `openai_advanced_scheduler_*`, `request_headers`, and `payment_subscription_usd_to_cny_rate`.

## Completion Matrix

| Upstream behavior | Local cleanroom result | Evidence | Status |
| --- | --- | --- | --- |
| Docker/local HTTP should be development-friendly without weakening core defaults | Core config still defaults to HTTPS-only; `docker-compose.local.yml`, `docker-compose.dev.yml`, and `.env.example` explicitly allow HTTP only for local/dev examples and document the production rollback switch. | `backend/internal/config/config_defaults.go`, `deploy/docker-compose.local.yml`, `deploy/docker-compose.dev.yml`, `deploy/.env.example` | Implemented |
| API key current concurrency should be visible without granting extra capability | API key service DTOs include `current_concurrency`; user key table shows realtime concurrency as a statistic only, with missing values rendered as `0`. | `backend/internal/service/api_key*.go`, `backend/internal/handler/dto/mappers_api_key.go`, `frontend/src/views/user/keys/KeysTable.vue`, Keys concurrency Vitest | Implemented/Verified |
| `/responses/compact` needs a normalized independent gateway endpoint | Endpoint normalization routes compact responses consistently with local OpenAI gateway behavior and tests. | `backend/internal/handler/endpoint.go`, `openai_gateway_endpoint_normalization_test.go` | Implemented |
| OpenAI advanced scheduling needs runtime knobs and admin score visibility | Runtime settings cover advanced enablement, sticky weighted scoring, subscription priority, TopK, and score weights; admin account list exposes a default-hidden scheduler score snapshot. | `setting_service_openai_scheduler.go`, `openai_account_scheduler*.go`, `SettingsGatewayExtraTab.vue`, `AccountsViewTable.vue` | Implemented |
| Redis concurrency cleanup should avoid stale slots and legacy wait residue | Active slot indexes and targeted sweeps clean expired account/user slots, wait counters, and stale index members without broad SCAN/N+1 behavior; the account load batch integration test no longer carries a TODO skip. | `backend/internal/repository/concurrency_cache.go`, `user_msg_queue_cache.go`, repository/service tests | Implemented; Docker integration pending |
| User message queue lock/cache boundaries need cleanup hardening | Lock index handling, boundary checks, and anomalous lock rescheduling are rewritten in local cache/service layers. | `backend/internal/repository/user_msg_queue_cache.go`, `user_msg_queue_service.go`, queue tests | Implemented; Docker integration pending |
| Account import should support drag/drop and multi-payload parsing | Existing import modal and credentials builder accept drag/drop JSON payloads, batch parse accounts, and preserve local create/edit flows. | `ImportDataModal.vue`, `credentialsBuilder.ts`, account modal tests | Implemented |
| API-key accounts need safe Anthropic/OpenAI request header overrides | Header override service rejects sensitive/protocol headers such as authorization, API key, cookie, and host; OpenAI/Anthropic/Grok gateway paths apply sanitized overrides. | `account_header_override.go`, gateway service files, account editor UI/tests | Implemented |
| EasyPay/custom payment methods and subscription CNY opt-in should fit local payment architecture | Existing payment settings/workbench carry custom provider visibility, masked/configured secret state, USD-to-CNY opt-in rate, and preview UI without new third-party dependencies. | `payment_settings.go`, `payment_service_*.go`, `PaymentSettingsCard.vue`, `PaymentWorkbench.vue` | Implemented |
| Payment provider response storage must tolerate NUL bytes | Provider payloads are sanitized before persistence/query/refund/webhook flows store or replay metadata. | `payment_service_util.go`, `payment_service_order.go`, `payment_service_refund.go`, payment tests | Implemented |
| OpenAI-compatible model sync must handle non-`/v1` base URLs | Model URL builder appends `/models` to versioned compatible bases such as `/v2`, `/v3`, and `/v4`, and keeps `/v1/models` behavior for host/base paths. | `openai_upstream_target.go`, `openai_upstream_target_test.go` | Implemented |
| Antigravity OAuth 401 should proactively refresh once | 401 invalid markers force the next Antigravity refresh and are cleared after successful or non-retryable refresh outcomes. | `antigravity_force_refresh.go`, `ratelimit_upstream_error.go`, token refresh tests | Implemented |
| Anthropic custom visible model lists should preserve local policy projection | Public model enumeration merges default and mapped Anthropic visibility through local policy and availability snapshots only. | `api_key_public_models_test.go`, model policy services | Implemented |
| OpenAI/Grok pricing and model metadata changes | Added local registry/pricing entries for `gpt-5.6-sol`, `gpt-5.6-terra`, `gpt-5.6-luna`, plus Grok image pricing updates. | `model_catalog_seed.json`, `registry_seed.json`, generated registry/pricing files | Implemented |
| Codex CLI headers and User-Agent compatibility | Local passthrough keeps `OpenAI-Beta: responses=experimental`, Codex originator/default User-Agent behavior, and account header override compatibility. | `openai_gateway_passthrough.go`, `openai_codex_transform.go`, Codex tests | Verified |
| API docs center and upstream release/license metadata | No `/api-docs/*` routes/pages are introduced; upstream LGPL/GPL text, README license changes, workflows, and version bumps are excluded. | `release_guard_test.go`, this matrix | Excluded/Guarded |

## Observability And Security

- Payment, OAuth refresh, import, scheduling, Redis cleanup, and gateway paths keep structured start/success/failure logging without raw secrets, credentials, or provider payload tokens.
- Header override settings are sanitized before routing and never allow overriding authorization, API key, cookie, host, or protocol-critical headers.
- New admin settings remain admin-only; payment creation remains user-authenticated and amount/provider validated.
- API key concurrency is read-only telemetry and does not expand model visibility, policy, or quota.
- Model lists and runtime support checks continue to use local policy projection and local availability snapshots; read paths do not trigger synchronous downstream probing.

## Regression Evidence

- Verified backend default suite: `go test ./...`.
- Verified backend targets: handler endpoint/gateway normalization, OpenAI scheduler/score snapshot, model URL building, Anthropic visible lists, Antigravity refresh, payment settings/orders/refunds, account header overrides, pricing, protocol capability, release guards, and API key public model projection.
- Verified frontend targets: payment settings/workbench, subscription CNY preview, account import/credentials builder, API-key account header settings, `KeysTable` current concurrency display, and `KeysView` current concurrency column wiring.
- Verified frontend full checks: `pnpm --prefix frontend run typecheck`, `pnpm --prefix frontend run build`, and `git diff --check`.
- Pending Docker-only verification in this workspace: `CI=true go test -tags integration ./internal/repository -run "TestConcurrencyCacheSuite|TestUserMsgQueueCacheSuite" -count=1 -v` must run with Docker available; when Docker is unavailable the repository integration harness exits early by design.

## Protected Local State

- Preserve local MIT `LICENSE`, README license sections, and version `0.1.376`.
- Preserve local account status visual changes in account status components and tests.
- Do not add `/api-docs/*` or `/admin/api-docs/*`; public model detail examples remain in `backend/internal/service/model_catalog_public_example_templates.go`.
