# Upstream v0.1.146-v0.1.150 Cleanroom Sync Matrix

- Local baseline: `v0.1.379` / `e57f84e54`.
- Trusted upstream tag commits verified via peeled remote tags:
  - `v0.1.146` -> `d7a6a4513a58b082922dfb8bd80f36cbe6b8a4c4`
  - `v0.1.147` -> `ba1f130d283daae4de70322cc17f01673808e986`
  - `v0.1.149` -> `dd1a116f4879992f04bb6ddbc77069bd503c3f4a`
  - `v0.1.150` -> `0dec1ad2922ff8c9d27b67f8a31dfb35bce1902b`
- Local same-name tags are intentionally ignored for diff decisions.
- No `git pull`, `fetch`, merge, rebase, or cherry-pick was used.
- MIT-only policy remains mandatory: do not copy upstream LGPL/GPL/CLA/COPYING text, upstream release scripts, or source snippets.

## Matrix

| Area | Status | Local cleanroom evidence | Validation evidence | Exclusions |
|---|---|---|---|---|
| Release/license guard | Closed | Root `LICENSE` still starts with MIT; README license sections remain MIT; local version stays `0.1.379`; `/api-docs/*` and `/admin/api-docs/*` routes remain forbidden by repository guard. | `go test ./internal/repository -run TestUpstream146To150CleanroomMatrixAndRollbackGuards`; final `git diff --check`. | Upstream LGPL/GPL/CLA/COPYING text, upstream release scripts, and upstream docs center are excluded. |
| Online rollback source trust | Closed | `check-updates` exposes up to three older trusted local release candidates; `rollback.target_version` uses the local trusted release policy for `ZY4869/sub2api` release metadata only. | `go test ./internal/repository -run TestUpstream146To150CleanroomMatrixAndRollbackGuards`; `go test ./internal/service -run TestUpdate`. | `Wei-Shaw/sub2api`, Docker Hub upstream namespace, raw GitHub asset CDN URLs, and unsigned/untrusted assets are rejected. |
| Rollback signature safety | Closed | Target-version rollback refuses online binary replacement unless a signature verifier is configured; legacy local-backup rollback remains available. Docker/Compose deployments remain operator image-tag rollback flows. | `go test ./internal/service -run TestUpdate`; `go test ./internal/repository -run TestUpstream146To150CleanroomMatrixAndRollbackGuards`. | No in-container binary replacement for Docker/Compose mode; no fallback to unsigned upstream binaries. |
| Batch image generation | Closed | Local `/v1/images/batches` API, migrations `154-156`, repository/service/worker state machine, billing hold/idempotency, Admin global/group switches, runtime metrics, output cleanup, single image and ZIP download, user `/image-batches` UI, and English/Chinese locale keys are implemented. Public DTOs expose `display_model_id`; provider batch names, `target_model_id`, account IDs, and storage paths remain internal. | `go test ./internal/service -run TestImageBatch`; `go test ./internal/handler/admin -run TestSettingHandlerImageBatchSettingsRoundTrip`; `go test ./internal/repository ./internal/service ./internal/handler/admin ./internal/handler ./internal/server ./internal/server/middleware`; `pnpm test:run src/views/user/__tests__/ImageBatchesView.spec.ts`; `pnpm typecheck`. | Non-Gemini group enablement is rejected by policy; no public exposure of provider internals; no second balance-freeze model beyond existing billing hold. |
| Chat/Responses request compatibility | Closed | `parallel_tool_calls` survives Chat/Responses conversion in both directions, including explicit `false`; Chat `response_format` maps to Responses `text.format`; invalid shapes return compat errors. | `go test ./internal/pkg/apicompat`; gateway hot-path tests listed below. | No full API docs center; public examples remain model-detail templates only when affected. |
| Gateway/model/billing fixes | Closed | Request-field compatibility, compact JSON/SSE handling, heartbeat/error stream paths, Windows WebSocket reset classification, Messages fallback, GPT-5.6 effort/alias handling, model policy read-path projection, static fallback/cache-write pricing, and payment/balance race fixes are preserved. | Gateway, WS, effort, pricing, billing, and payment targeted tests in the validation snapshot; final `go test ./...`. | Public model reads never trigger synchronous downstream probing; probe outputs never expand visible/callable model sets. |
| Admin user role management | Closed | Admin create/update accepts optional `role`; service validates enum values, invalidates auth cache on role changes, audits role changes/rejections, and blocks disabling/demoting/deleting the last active admin. Frontend create/edit forms expose role selection. | `go test -tags unit ./internal/service -run "TestAdminService_(CreateUser|UpdateUser|DeleteUser)"`; `go test ./internal/handler/admin -run TestUserHandlerEndpoints`; `pnpm test:run src/components/admin/user/__tests__/UserModelBindingModeModals.spec.ts`. | No alternate role model beyond `user`/`admin`; no PII in audit messages. |
| Admin users/usage | Closed | Admin usage ranking and user breakdown support `request_type` filters plus token/request/actual-cost metrics. Frontend usage keeps the existing tab structure, request-type filtering, ranking drilldown, latency health display, readable durations, and matched i18n keys. | `go test ./internal/handler/admin -run "TestDashboardHandler"`; `go test ./internal/repository -run "TestUsageLogRepo"`; `pnpm test:run src/views/admin/__tests__/UsageView.spec.ts`. | No duplicate aggregation API; existing dashboard/user-breakdown contract remains the integration point. |

## Guarded Decisions

- Public model examples remain in `backend/internal/service/model_catalog_public_example_templates.go`; do not add an API documentation center.
- Public model enumeration continues to use `display_model_id`; `target_model_id` remains internal diagnostics metadata.
- Update and rollback code must use the local trusted release source and must not download or install upstream LGPL artifacts.
- Docker/Compose rollback remains an operator image-tag action, not in-container binary replacement.

## Validation Snapshot

- `pnpm test:run src/views/user/__tests__/ImageBatchesView.spec.ts`
- `go test -tags unit ./internal/service -run "TestAdminService_(CreateUser|UpdateUser|DeleteUser)"`
- `go test ./internal/handler/admin -run TestUserHandlerEndpoints`
- `npm run test -- UserModelBindingModeModals.spec.ts`
- `go test ./internal/pkg/apicompat`
- `go test ./internal/service -run "Test(ConvertChatCompletionsToResponsesRuntimeUsesRegistry|ForwardAsChatCompletions_ProtocolGatewayChatPreferenceUsesNativeChatUpstream|ForwardResponsesAsChatCompletions_UsesChatUpstreamAndReturnsResponses|ForwardAsChatCompletions_StripsClaudeMillionContextSuffixBeforeResponsesUpstream)"`
- `go test ./internal/service -run "TestForwardAsChatCompletions_WritesLocalizedCompatError|TestForwardResponsesAsChatCompletions_TransportErrorReturnsFailover"`
- `go test ./internal/service -run "TestIsOpenAIWSClientDisconnectError"`
- `go test ./internal/service/openai_ws_v2 -run "TestRelayUtilityCoverageBranches|TestIsDisconnectErrorCoverage"`
- `go test ./internal/service -run "TestGatewayEffortResolutionDefaultSuite|TestExtractOpenAIReasoningEffortFromBody|TestNormalizeOpenAIRequestBodyEffort_UsesMappedGpt56Candidate|TestNormalizeOpenAIRequestBodyEffort_DerivesMaxFromMappedGpt56Suffix"`
- `go test ./internal/service -run "TestGatewayEffortResolutionDefaultSuite|TestExtractOpenAIReasoningEffortFromBody|TestExtractOpenAIReasoningEffortFromBody_DerivesFromMappedCandidate|TestNormalizeOpenAIRequestBodyEffort_UsesMappedGpt56Candidate|TestNormalizeOpenAIRequestBodyEffort_DerivesMaxFromMappedGpt56Suffix"`
- `go test ./internal/service -run "TestNormalizeCodexModel_Gpt53|TestGetModelPricing_Gpt56UsesStaticFallbackWhenRemoteMissing|TestBillingFallbackPricingOpenAIGPT56Families|TestCalculateCost_Gpt56FallbackUsesCacheWritePrice"`
- `go test ./internal/pkg/apicompat`
- `go test ./internal/service -run "TestImageBatch"`
- `go test ./internal/handler/admin -run "TestSettingHandlerImageBatchSettingsRoundTrip"`
- `go test ./internal/repository ./internal/service ./internal/handler/admin ./internal/handler ./internal/server ./internal/server/middleware`
- `pnpm typecheck`
- `pnpm test:run src/components/charts/__tests__/ModelDistributionChart.spec.ts src/components/charts/__tests__/GroupDistributionChart.spec.ts src/views/admin/__tests__/UsageView.spec.ts`
- Pending final sweep before merge: `go test ./...`, `pnpm test:run`, `pnpm build`, and `git diff --check`.
