# Project AGENTS

## UI Identity Rules

- When the UI needs to show vendor identity, prefer the vendor SVG icon instead of vendor text.
- When the UI needs to show a model display name or model ID, prepend the model icon.
- Only fall back to plain text when the icon cannot be resolved.
- Apply this rule first to probe cards, model lists, and account-related UI touched by new work. Converge incrementally instead of doing a whole-site replacement at once.

## Model Example Template Sync Rules

- The repository no longer ships a standalone user/admin API documentation center. Do not add `/api-docs/*` or `/admin/api-docs/*` routes, pages, handlers, or runtime overrides.
- Public model detail protocol examples live in `backend/internal/service/model_catalog_public_example_templates.go`.
- Any change to model request paths, alias paths, authentication rules, protocol compatibility surface, error response shape, or example request shape must update the matching public model example template and related tests in the same change.
- Keep templates concise and runnable, with stable page IDs selected by `selectPublicModelCatalogExampleSpec`; they are model-detail examples, not a replacement for a full documentation center.

## Model Policy Rules

- `extra.model_scope_v2.entries[]` is the single source of truth for account model policy. New writes must normalize into `policy_mode + entries[]`; legacy fields are compatibility inputs only.
- The only allowed exposure priority is `explicit whitelist / alias mapping > default library`. `manual_models`, `model_probe_snapshot`, `openai_known_models`, and other probe outputs may annotate availability, but must never expand the visible or callable model set.
- External model list/detail responses and external model selection must use `display_model_id` only. `target_model_id` is internal routing and diagnostics metadata and must not be exposed as another callable public ID.
- Read paths for model lists, model detail, runtime support checks, and test-model selection must use local policy projection plus local availability snapshot only. Do not trigger synchronous downstream model probing on read paths.
- Any model-policy change must regression-cover admin available models, public `/models` or `/v1/models` style enumeration, runtime routing/support checks, snapshot refresh behavior, and legacy compatibility in the same change.
- Any change to model request paths, response semantics, alias visibility, compatibility fallbacks, or model enumeration behavior must update the public model example templates when examples are affected, plus related tests in the same change.

## Local Deployment Rules

- When the request is about local deployment, default the target environment to WSL even if WSL is not mentioned explicitly.
- Default local deployment to Docker Compose containers. Do not switch to native Windows processes, Windows services, or other non-container local deployment modes unless the user explicitly asks for them.
- When the user asks to deploy locally and there is no existing local Docker Compose deployment, treat it as a first-time deployment and default to starting the containers.
- When the user asks to update a local deployment, or an existing local Docker Compose deployment is already present, treat it as a container update and default to updating the image and recreating the containers instead of only restarting them.
- The automatic choice between starting containers and updating containers must be based on whether an existing local Docker Compose deployment already exists.
- Only override these defaults when the user explicitly requests a non-WSL environment, a non-Docker workflow, or another deployment method.
