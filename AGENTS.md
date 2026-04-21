# Project AGENTS

## UI Identity Rules

- When the UI needs to show vendor identity, prefer the vendor SVG icon instead of vendor text.
- When the UI needs to show a model display name or model ID, prepend the model icon.
- Only fall back to plain text when the icon cannot be resolved.
- Apply this rule first to probe cards, model lists, and account-related UI touched by new work. Converge incrementally instead of doing a whole-site replacement at once.

## API Docs Sync Rules

- The repository default API documentation baseline lives at `backend/internal/service/docs/index.md` plus `backend/internal/service/docs/pages/*.md`.
- Any change to request paths, alias paths, authentication rules, authentication priority, deprecated parameters, protocol compatibility surface, error response shape, example requests, or onboarding constraints must update the matching page file in `backend/internal/service/docs/pages/` in the same change, and keep `backend/internal/service/docs/index.md` as the shared document title source.
- Keep the API docs baseline in the agreed multi-file virtual-page format: one `#` document title in `docs/index.md`, fixed `##` page IDs (`common`, `openai`, `anthropic`, `gemini`, `grok`, `antigravity`, `vertex-batch`) in `docs/pages/*.md`, `###` section headings, and `#### Python` / `#### JavaScript` / `#### REST` example tabs followed by fenced code blocks.
- When protocol behavior changes, update both the narrative rules and the matching code examples for the affected virtual page; do not leave a section without a current runnable example unless the section explicitly documents that the action is unsupported.
- Runtime overrides saved from `/admin/api-docs` do not replace the repository baseline. Keep the code-tracked template accurate even when production content has been customized.

## Model Policy Rules

- `extra.model_scope_v2.entries[]` is the single source of truth for account model policy. New writes must normalize into `policy_mode + entries[]`; legacy fields are compatibility inputs only.
- The only allowed exposure priority is `explicit whitelist / alias mapping > default library`. `manual_models`, `model_probe_snapshot`, `openai_known_models`, and other probe outputs may annotate availability, but must never expand the visible or callable model set.
- External model list/detail responses and external model selection must use `display_model_id` only. `target_model_id` is internal routing and diagnostics metadata and must not be exposed as another callable public ID.
- Read paths for model lists, model detail, runtime support checks, and test-model selection must use local policy projection plus local availability snapshot only. Do not trigger synchronous downstream model probing on read paths.
- Any model-policy change must regression-cover admin available models, public `/models` or `/v1/models` style enumeration, runtime routing/support checks, snapshot refresh behavior, and legacy compatibility in the same change.
- Any change to model request paths, response semantics, alias visibility, compatibility fallbacks, or model enumeration behavior must update `backend/internal/service/docs/pages/` and the related tests in the same change.
