# Project AGENTS

## UI Identity Rules

- When the UI needs to show vendor identity, prefer the vendor SVG icon instead of vendor text.
- When the UI needs to show a model display name or model ID, prepend the model icon.
- Only fall back to plain text when the icon cannot be resolved.
- Apply this rule first to probe cards, model lists, and account-related UI touched by new work. Converge incrementally instead of doing a whole-site replacement at once.
