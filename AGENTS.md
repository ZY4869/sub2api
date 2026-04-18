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
