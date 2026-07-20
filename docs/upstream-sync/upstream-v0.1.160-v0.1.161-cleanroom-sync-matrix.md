# Upstream v0.1.160-v0.1.161 Cleanroom Sync Matrix

Local baseline: current workspace version `0.1.404`. Upstream reference is `Wei-Shaw/sub2api` compare range `v0.1.160...v0.1.161`; the reference anchors are `8bfbc5ca99bf2c0ac96e0f29ffd35eb6aca27e62` and `19149ca196eeae4a4482e5299dc6fa4ba0b06c8c`. Any same-name local tags are intentionally ignored.

Cleanroom rules:

- No `git pull`, `fetch`, merge, rebase, or cherry-pick from `Wei-Shaw/sub2api`.
- Do not copy upstream LGPL/GPL source text, license text, README/governance text, sponsor assets, generated files, or migration numbering.
- Keep the root `LICENSE` MIT-only, keep README license statements MIT, and keep local version metadata unchanged.
- Do not add `/api-docs/*` or `/admin/api-docs/*` routes, pages, handlers, or runtime overrides.
- Preserve local architecture and local security defaults; any overlap with existing behavior is fused into the local implementation unless explicitly noted otherwise.

| Upstream behavior anchor | Local cleanroom result | Evidence | Status |
| --- | --- | --- | --- |
| Responses compatibility stream repair | Clean-room bridge now emits `response.content_part.added`, `response.output_text.done`, and `response.content_part.done`, and `response.completed.response.output` is assembled from the accumulated text stream. | `backend/internal/pkg/apicompat/chatcompletions_to_responses_response.go`, `backend/internal/pkg/apicompat/chatcompletions_responses_test.go`, service bridge tests | Cleanroom rewrite |
| SSE frame boundary failure handling | Streaming failures are isolated behind a stable SSE frame boundary, use sanitized `response.failed` error frames, and route already-started failures into Ops logging without leaking partial upstream frames. | `backend/internal/service/openai_gateway_streaming.go`, `backend/internal/service/openai_gateway_service_test.go` | Cleanroom rewrite |
| Pool mode and temp rule handling | `temp_unschedulable_rules` still works in pool mode when custom error codes are disabled, and model-scoped cooldown remains isolated to the matching model. | `backend/internal/service/ratelimit_error_policy.go`, `backend/internal/service/ratelimit_upstream_error.go`, `backend/internal/service/error_policy_test.go` | Cleanroom rewrite |
| Durable API key auth cache invalidation | Auth cache invalidation now has a durable outbox fallback, with hashed invalid-credential rate limiting and redacted `Authorization: Bearer` failure logging to avoid raw key leakage. | `backend/internal/service/api_key_auth_cache_outbox.go`, `backend/internal/repository/api_key_auth_cache_invalidation_outbox_repo.go`, `backend/internal/service/api_key_auth_failure_rate_limit.go`, middleware tests | Cleanroom rewrite |
| Grok media and model capability | Grok media stays on the local implementation. The positive eligibility path is guarded, image/video/status entrypoints remain wired, Responses Lite keeps client function/custom/tool_search tools without native search injection, and media URLs are validated through the local URL policy helper. | `backend/internal/service/account_openai_capability.go`, `backend/internal/server/routes/gateway.go`, `backend/internal/server/routes/gateway_test.go`, `backend/internal/service/openai_responses_lite_tools.go`, `backend/internal/service/grok_gateway_video_workflow.go` | Fused / guarded |
| Docker build portability | Backend Docker build is kept local but updated for BuildKit cross-compile arguments `TARGETOS`, `TARGETARCH`, and `TARGETVARIANT`, plus cached Go module/build directories. | `backend/Dockerfile` | Cleanroom rewrite |
| Governance and release guard | Local sync audit records the clean-room boundary, preserves MIT-only licensing, keeps version `0.1.404`, and rejects upstream `183/184` migration adoption. | `backend/internal/repository/release_guard_161_test.go`, `LICENSE`, `backend/cmd/server/VERSION`, migration guards | Guarded |

Deferred or explicitly excluded:

- Upstream license changes, README/license prose rewrites, and any migration numbering that would replace the local `164_api_key_auth_cache_invalidation_outbox.sql` are excluded.
- No standalone docs center is added; `/api-docs/*` and `/admin/api-docs/*` stay forbidden.
- Any local overlap with upstream feature shape is fused into the current architecture rather than copied as a parallel stack.

Local follow-up:

- Anthropic API Key pool-mode support is exercised through the shared API Key/Bedrock pool-mode path, with local backend/frontend tests and copy updates covering Anthropic creation, editing, and passthrough retryability.

Validation commands:

- `go test ./internal/pkg/apicompat ./internal/service ./internal/handler ./internal/handler/admin ./internal/server/middleware ./internal/repository -count=1`
- `go test -tags unit ./internal/service -run "TestCheckErrorPolicy|TestGrokMediaGenerationEligibility|TestNormalizeOpenAIResponsesLiteToolsDoesNotInjectNativeSearchForClientTools|TestGrokValidateVideoWorkflowRequestRejectsPrivateMediaURLs|TestGrokValidateVideoWorkflowRequestAllowsConfiguredPrivateMediaURLs" -count=1`
- `go test ./internal/repository -run "TestUpstream160To161CleanroomMatrixGuards|TestSelectiveUpstreamAbsorptionReleaseGuards" -count=1`
- `git status --short`
