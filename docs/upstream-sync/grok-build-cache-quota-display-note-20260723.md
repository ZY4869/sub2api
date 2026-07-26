# Grok Build Cache Quota Display Note 2026-07-23

## Summary

- Local Grok Build already had short-TTL tool prompt caching; this change does not add a second cache layer.
- The update keeps `accounts.extra.grok_billing_snapshot` and `accounts.extra.grok_usage_snapshot` as the quota snapshot source.
- The main fix is to align xAI request sanitization, latest quota observation timestamps, and remaining-oriented quota display.

## Local Result

- Grok Responses requests now normalize xAI-only incompatible tool shapes before prompt-cache key generation and upstream forwarding.
- Empty/filtered `tools` removes orphaned `tool_choice` and `parallel_tool_calls`; explicit `prompt_cache_key` remains preserved.
- Grok quota usage info reports `grok_last_quota_probe_at`, `grok_last_headers_seen_at`, and `grok_quota_snapshot_state` from the latest effective billing/header observation.
- Grok request/token quota rows display remaining percentage semantics while exposing raw `limit` and `remaining` values in the row tooltip.
- Stale Grok quota snapshots surface a soft note instead of silently looking fresh.

## Validation

- `cd backend && go test -tags unit ./internal/service -run "TestSanitizeGrokOpenAICompatibleRequestBody|TestBuildGrokMessagesCompatResponsesBody|TestGrokResponsesToolPromptCache" -count=1`
- `cd backend && go test -tags unit ./internal/service -run "TestGrokQuotaFetcherBuildUsageInfo|TestGrokQuotaService_QueryQuota" -count=1`
- `cd frontend && pnpm exec vitest run src/components/account/__tests__/UsageProgressBar.spec.ts src/components/account/__tests__/AccountUsageCell.spec.ts src/components/admin/account/__tests__/AccountUsageVisualCell.spec.ts`
