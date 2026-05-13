import argparse
import json
from dataclasses import dataclass
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parent.parent
DEFAULT_SEED_PATH = ROOT / "backend" / "internal" / "modelregistry" / "registry_seed.json"
DEFAULT_OUTPUT_PATH = ROOT / "docs" / "model_registry_official_audit_20260512_phase2.md"
DEFAULT_AUDIT_DATE = "2026-05-12"

OFFICIAL_HARD_REMOVE_CANDIDATES = {
    "gemini": {"gemini-3-pro-preview", "gemini-2.5-flash-image-preview", "unknown"},
}

SEED_DEPRECATED_HARD_REMOVE_CANDIDATES = {
    "anthropic": {
        "claude-haiku-4-5-20251001",
        "claude-opus-4-5-20251101",
        "claude-opus-4-5-thinking",
        "claude-sonnet-4-5",
        "claude-sonnet-4-5-20250929",
        "claude-sonnet-4-5-thinking",
    },
    "deepseek": {
        "deepseek-chat",
        "deepseek-reasoner",
    },
    "grok": {
        "grok-2",
        "grok-2-image",
        "grok-2-vision",
        "grok-3-beta",
        "grok-3-fast-beta",
        "grok-3-mini-beta",
        "grok-4",
        "grok-4-0709",
        "grok-beta",
        "grok-imagine-image",
        "grok-imagine-video",
        "grok-vision-beta",
    },
}

OFFICIAL_SOURCE_MATRIX: dict[str, dict[str, Any]] = {
    "openai": {
        "urls": [
            "https://platform.openai.com/docs/models",
            "https://developers.openai.com/api/docs/models/gpt-image-2",
        ],
        "mode": "keep_all",
        "notes": [
            "OpenAI 当前官方模型页仍可查询 gpt-image-2，本轮不做删除。",
            "OpenAI 侧这轮以命名规范收口为主，不自动扩大 hard-remove 范围。",
        ],
    },
    "anthropic": {
        "urls": [
            "https://docs.anthropic.com/en/docs/about-claude/models/all-models",
            "https://docs.anthropic.com/en/docs/about-claude/model-deprecations",
        ],
        "mode": "deprecated_seed_cleanup",
        "notes": [
            "Anthropic dated upstream ID 仍可能是官方可调用 target；本轮只删除仓库里已标记 deprecated + replaced_by 的旧兼容壳。",
        ],
    },
    "gemini": {
        "urls": [
            "https://ai.google.dev/gemini-api/docs/models",
        ],
        "mode": "hybrid_cleanup",
        "notes": [
            "Gemini 本轮同时纳入官方确认删除项与仓库内已落地的非法/脏 preview 壳。",
        ],
    },
    "grok": {
        "urls": [
            "https://docs.x.ai/docs/models",
            "https://docs.x.ai/developers/rest-api-reference/inference/models",
        ],
        "mode": "deprecated_seed_cleanup",
        "notes": [
            "xAI 官方页仍可见部分历史 Grok 名称；本轮删除依据是仓库内已标记 deprecated + replaced_by 的旧兼容壳，而不是声称官方已下线。",
        ],
    },
    "deepseek": {
        "urls": [
            "https://api-docs.deepseek.com/",
        ],
        "mode": "deprecated_seed_cleanup",
        "notes": [
            "DeepSeek 本轮按仓库现有 deprecated + replaced_by 收口旧兼容壳。",
        ],
    },
    "moonshot": {
        "urls": [
            "https://platform.moonshot.ai/docs/overview",
        ],
        "mode": "skip_source_insufficient",
        "notes": [
            "缺少足够稳定的官方当前模型全集主源，本轮只记录命名与来源不足，不自动删除。",
        ],
    },
}

PROVIDER_ALIASES = {
    "xai": "grok",
}


@dataclass(frozen=True)
class SeedModel:
    provider: str
    id: str
    protocol_ids: list[str]
    aliases: list[str]
    display_name: str
    status: str
    replaced_by: str
    exposed_in: list[str]


@dataclass(frozen=True)
class AuditRow:
    provider: str
    urls: list[str]
    hard_remove: list[str]
    keep_and_normalize: list[str]
    keep_as_is: list[str]
    naming_only_fix: list[str]
    skip_source_insufficient: list[str]
    remarks: list[str]


def normalize_id(value: str) -> str:
    normalized = str(value or "").strip().lower()
    if normalized.startswith("models/"):
        normalized = normalized[len("models/") :]
    return normalized


def parse_seed_entries(seed_path: Path) -> list[dict[str, Any]]:
    payload = json.loads(seed_path.read_text(encoding="utf-8"))
    if not isinstance(payload, list):
        raise ValueError("registry seed must be a JSON array")
    return payload


def to_seed_model(entry: dict[str, Any]) -> SeedModel:
    return SeedModel(
        provider=PROVIDER_ALIASES.get(str(entry.get("provider") or "").strip().lower(), str(entry.get("provider") or "").strip().lower()),
        id=normalize_id(entry.get("id") or ""),
        protocol_ids=[normalize_id(value) for value in entry.get("protocol_ids") or [] if normalize_id(value)],
        aliases=[normalize_id(value) for value in entry.get("aliases") or [] if normalize_id(value)],
        display_name=str(entry.get("display_name") or "").strip(),
        status=str(entry.get("status") or "").strip().lower(),
        replaced_by=normalize_id(entry.get("replaced_by") or ""),
        exposed_in=[str(value).strip().lower() for value in entry.get("exposed_in") or [] if str(value).strip()],
    )


def collect_provider_models(entries: list[dict[str, Any]]) -> dict[str, set[str]]:
    provider_models: dict[str, set[str]] = {}
    for raw_entry in entries:
        entry = to_seed_model(raw_entry)
        if not entry.provider or not entry.id:
            continue
        ids = {entry.id}
        ids.update(entry.protocol_ids)
        ids.update(entry.aliases)
        provider_models.setdefault(entry.provider, set()).update(ids)
    return provider_models


def collect_provider_seed_models(entries: list[dict[str, Any]]) -> dict[str, list[SeedModel]]:
    provider_models: dict[str, list[SeedModel]] = {}
    for raw_entry in entries:
        entry = to_seed_model(raw_entry)
        if not entry.provider or not entry.id:
            continue
        provider_models.setdefault(entry.provider, []).append(entry)
    for models in provider_models.values():
        models.sort(key=lambda item: item.id)
    return provider_models


def infer_normalized_models(models: list[SeedModel], hard_remove_ids: set[str]) -> list[str]:
    items: list[str] = []
    for model in models:
        if model.id in hard_remove_ids:
            continue
        if requires_naming_fix(model.display_name, model.id):
            items.append(model.id)
    return sorted(set(items))


def requires_naming_fix(display_name: str, model_id: str) -> bool:
    display_name = (display_name or "").strip()
    if not display_name:
        return True
    if "_" in display_name:
        return True
    if display_name.lower() == display_name:
        return True
    lowered = display_name.lower()
    if "deepseek-" in lowered or lowered.startswith("deepseek"):
        return True
    if lowered.startswith("chatgpt-") or lowered.startswith("chatglm_"):
        return True
    if lowered.startswith("doubao-") or lowered.startswith("mistral-"):
        return True
    if lowered.startswith("kimi-") or lowered.startswith("moonshot-"):
        return True
    return False


def audit_provider(provider: str, models: list[SeedModel]) -> AuditRow:
    config = OFFICIAL_SOURCE_MATRIX.get(provider)
    if config is None:
        skipped = sorted(model.id for model in models)
        return AuditRow(
            provider=provider,
            urls=[],
            hard_remove=[],
            keep_and_normalize=[],
            keep_as_is=[],
            naming_only_fix=[],
            skip_source_insufficient=skipped,
            remarks=["skip_source_insufficient: provider 未配置官方主源映射"],
        )

    urls = list(config["urls"])
    notes = list(config["notes"])
    mode = config["mode"]

    official_hard_remove = OFFICIAL_HARD_REMOVE_CANDIDATES.get(provider, set())
    seed_hard_remove = SEED_DEPRECATED_HARD_REMOVE_CANDIDATES.get(provider, set())

    hard_remove_ids: set[str] = set()
    if mode in {"hybrid_cleanup", "deprecated_seed_cleanup"}:
        for model in models:
            if model.id in seed_hard_remove:
                hard_remove_ids.add(model.id)
            if model.status == "deprecated" and model.replaced_by and model.id in seed_hard_remove:
                hard_remove_ids.add(model.id)
    if mode in {"hybrid_cleanup", "hard_remove_list"}:
        for model in models:
            if model.id in official_hard_remove:
                hard_remove_ids.add(model.id)

    normalized_ids = set(infer_normalized_models(models, hard_remove_ids))

    if mode == "skip_source_insufficient":
        return AuditRow(
            provider=provider,
            urls=urls,
            hard_remove=[],
            keep_and_normalize=[],
            keep_as_is=[],
            naming_only_fix=sorted(normalized_ids),
            skip_source_insufficient=sorted(model.id for model in models),
            remarks=notes + ["skip_source_insufficient"],
        )

    keep_and_normalize = sorted(normalized_ids)
    keep_as_is = sorted(
        model.id for model in models
        if model.id not in hard_remove_ids and model.id not in normalized_ids
    )

    return AuditRow(
        provider=provider,
        urls=urls,
        hard_remove=sorted(hard_remove_ids),
        keep_and_normalize=keep_and_normalize,
        keep_as_is=keep_as_is,
        naming_only_fix=keep_and_normalize,
        skip_source_insufficient=[],
        remarks=notes,
    )


def render_markdown(rows: list[AuditRow], audit_date: str) -> str:
    lines = [
        f"# 模型注册表二次深清理审计（{audit_date}）",
        "",
        f"核对日期：{audit_date}",
        "",
        "审计规则：",
        "- provider 范围从 `backend/internal/modelregistry/registry_seed.json` 动态派生。",
        "- 本轮二次清理同时参考 provider 官方主源，以及仓库内已明确 `deprecated + replaced_by` 的旧兼容壳。",
        "- 只做保守归类：`hard_remove`、`keep_and_normalize`、`keep_as_is`、`skip_source_insufficient`、`naming_only_fix`。",
        "- `preview/latest/dated` 不是删除条件；只有命中官方确认删除或仓库已明确 `deprecated + replaced_by` 才进入 `hard_remove`。",
        "",
    ]

    for row in rows:
        lines.append(f"## {row.provider}")
        lines.append("")
        lines.append("官方来源：")
        if row.urls:
            for url in row.urls:
                lines.append(f"- {url}")
        else:
            lines.append("- 无")
        lines.append("")
        lines.append("核对结论：")
        for remark in row.remarks:
            lines.append(f"- {remark}")
        lines.append("")
        lines.append("hard_remove：")
        lines.extend(render_model_lines(row.hard_remove))
        lines.append("")
        lines.append("keep_and_normalize：")
        lines.extend(render_model_lines(row.keep_and_normalize))
        lines.append("")
        lines.append("keep_as_is：")
        lines.extend(render_model_lines(row.keep_as_is))
        lines.append("")
        lines.append("naming_only_fix：")
        lines.extend(render_model_lines(row.naming_only_fix))
        lines.append("")
        lines.append("skip_source_insufficient：")
        lines.extend(render_model_lines(row.skip_source_insufficient))
        lines.append("")
        lines.append("备注：")
        lines.append(f"- provider: `{row.provider}`")
        lines.append(f"- audit_date: `{audit_date}`")
        lines.append("")
    return "\n".join(lines).rstrip() + "\n"


def render_model_lines(items: list[str]) -> list[str]:
    if not items:
        return ["- 无"]
    return [f"- `{item}`" for item in items]


def generate_audit_report(
    seed_path: Path = DEFAULT_SEED_PATH,
    output_path: Path = DEFAULT_OUTPUT_PATH,
    audit_date: str = DEFAULT_AUDIT_DATE,
) -> list[AuditRow]:
    entries = parse_seed_entries(seed_path)
    provider_seed_models = collect_provider_seed_models(entries)
    rows = [audit_provider(provider, models) for provider, models in sorted(provider_seed_models.items())]
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(render_markdown(rows, audit_date), encoding="utf-8")
    return rows


def main() -> None:
    parser = argparse.ArgumentParser(description="Generate the 2026-05-12 phase2 model registry audit report.")
    parser.add_argument("--seed", type=Path, default=DEFAULT_SEED_PATH)
    parser.add_argument("--output", type=Path, default=DEFAULT_OUTPUT_PATH)
    parser.add_argument("--audit-date", default=DEFAULT_AUDIT_DATE)
    args = parser.parse_args()
    generate_audit_report(seed_path=args.seed, output_path=args.output, audit_date=args.audit_date)


if __name__ == "__main__":
    main()
