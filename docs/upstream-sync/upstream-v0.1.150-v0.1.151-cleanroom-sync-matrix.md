# Upstream v0.1.150-v0.1.151 Cleanroom Sync Matrix

Local baseline: `v0.1.379`. Upstream references are the official v0.1.151 release notes and the v0.1.149...v0.1.150 / v0.1.150...v0.1.151 compare behavior. This local sync is behavior-only.

Cleanroom rules:

- No `git pull`, `fetch`, merge, rebase, or cherry-pick from `Wei-Shaw/sub2api`.
- Do not copy upstream LGPL/GPL source text, license text, route files, documentation center text, or migration numbering.
- Keep the root `LICENSE` MIT-only and do not add `/api-docs/*` or `/admin/api-docs/*` routes.
- Use local migration `157_allow_cyber_blocked_usage_request_type.sql`; do not add upstream migration `173`.

| Upstream behavior anchor | Local cleanroom result | Evidence | Status |
| --- | --- | --- | --- |
| Fast/Flex supports user-scoped rules | `openai_fast_policy_settings.rules[].user_ids` is normalized as positive unique IDs; authenticated `ctxkey.UserID` rules win over global rules; default local policy still filters `priority/fast` and passes `flex`. | `backend/internal/service/openai_fast_policy_settings.go`, `openai_fast_policy_enforcer.go`, `frontend/src/components/settings/OpenAIFastPolicySettingsCard.vue` | Closed |
| Codex identity headers pair final User-Agent with originator and minimum version | Shared local helper normalizes OpenAI OAuth Codex outbound headers across normal Responses forwarding, passthrough, account tests, Codex usage probes, WS ingress, and WS v2 passthrough header building. | `backend/internal/service/openai_codex_identity.go`, `openai_gateway_upstream_request.go`, `openai_gateway_passthrough.go`, `openai_ws_headers.go`, `account_test_openai_health.go`, `account_usage_openai_codex_http_probe.go`, `openai_codex_ws_probe.go` | Closed |
| Compact SSE keepalive must not look like a real upstream response | `/responses/compact` appends `response.failed` only after `OpenAIRealSSEStarted` metadata is set; keepalive/comment bytes alone are not treated as real response bytes. Compact timeout still emits its own failed event. | `backend/internal/handler/openai_response_failed_event.go`, `gateway_handler_error_fallback_test.go`, `backend/internal/service/openai_gateway_service_test.go` | Closed |
| GPT-5.6 alias, max effort, pricing, and usage candidates | Backend aliases include `gpt-5.6-{sol,terra,luna}` plus variants including `max`; fallback pricing and `max` effort resolution cover GPT-5.6; frontend/OpenCode capabilities expose `max`. | `backend/internal/service/openai_codex_transform.go`, `pricing_service_openai_fallback.go`, `billing_fallback_pricing.go`, `effort_level.go`, `frontend/src/composables/useModelWhitelist.ts`, `frontend/src/components/keys/UseKeyModal.vue` | Closed |
| Grok Responses keeps supported reasoning effort parameters | Grok OpenAI-compatible sanitizer removes only explicitly unsupported fields and preserves `reasoning` / `reasoning_effort`, including direct `/grok/v1/responses` POST forwarding. | `backend/internal/service/grok_gateway_service_apikey.go`, `grok_gateway_responses_test.go`, `grok_gateway_messages_compat_test.go` | Closed |
| Codex `image_gen` namespace is stripped or normalized without bypassing local policy | Local Codex image tool policy recognizes `image_gen` namespace declarations and keeps group/image permissions enforced. | `backend/internal/service/openai_codex_transform.go`, `openai_codex_transform_test.go` | Closed |
| Anthropic setup-token background refresh | setup-token accounts participate in refresh scheduling based on `expires_at` and distributed locking, without token logging. | `backend/internal/service/token_refresher.go`, `token_refresher_test.go` | Closed |
| Auth regression remains private by default | `/v1/models` and `/v1/chat/completions` without key return 401. | `backend/internal/server/routes/gateway_test.go` | Closed |

Validation commands:

- `go test ./internal/service -run "Test(EnforceCodexIdentityHeaders|OpenAIFastPolicy|OpenAIStreamingKeepalive|OpenAICompactStreaming|SanitizeGrok|GrokForwardResponses|ApplyCodexImageToolPolicy|.*GPT.*56|.*Token.*Refresh)" -count=1`
- `go test ./internal/handler ./internal/server ./internal/server/middleware ./internal/repository -run "Test(Gateway|APIKey|Cleanroom|Upstream|Migration)" -count=1`
- `pnpm test:run -- OpenAIFastPolicySettingsCard UseKeyModal useModelWhitelist`
- `pnpm typecheck`
- `git diff --check`
