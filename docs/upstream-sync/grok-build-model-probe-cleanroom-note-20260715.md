# Grok Build Model Probe Cleanroom Note 2026-07-15

## Summary

- `upstream/main` has related Grok Build built-in model and mapping behavior, but this branch already carries its local catalog through `GrokBuildTextModelIDs()`.
- This change does not cherry-pick upstream implementation. It only completes the local architecture gaps for Grok Build API Key/OAuth model probing and OAuth account default model scope.
- Grok SSO reverse runtime, public routes, and model forwarding paths are intentionally unchanged.

## Local Result

- Grok API Key/OAuth `/v1/models` probing still uses the upstream list when the endpoint returns a valid model payload.
- When the Grok model listing endpoint returns 404, 405, or 410, probing falls back to the local `grok_build_builtin_catalog` source.
- Credential, permission, rate-limit, and upstream failure statuses such as 401, 403, 429, and 5xx continue to fail instead of falling back.
- New Grok OAuth accounts persist a default whitelist `extra.model_scope_v2` containing all `GrokBuildTextModelIDs()` entries.
- Grok OAuth reauthorization fills the default scope only when the existing account lacks `model_scope_v2`; user-edited scopes are preserved.
- Fallback logging records only account/platform/type/base host/fallback source and optional upstream status, without token, response body, or raw upstream error text.

## Validation

- `go test ./internal/service -run "Test.*Grok.*Model|TestImportAccountModels_Grok|TestProbeAccountModels_Grok|TestGrokOAuth" -count=1`
- `go test -tags unit ./internal/service ./internal/handler/admin`
