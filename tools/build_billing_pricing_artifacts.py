#!/usr/bin/env python3
import argparse
import json
from copy import deepcopy
from datetime import datetime
from pathlib import Path
from typing import Any


ROOT_FIELDS = [
    "input_price",
    "output_price",
    "cache_price",
    "tier_threshold_tokens",
    "input_price_above_threshold",
    "output_price_above_threshold",
    "shared_multiplier",
]

BOOLEAN_FIELDS = [
    "special_enabled",
    "tiered_enabled",
    "multiplier_enabled",
]

SPECIAL_FIELDS = [
    "batch_input_price",
    "batch_output_price",
    "batch_cache_price",
    "grounding_search",
    "grounding_maps",
    "file_search_embedding",
    "file_search_retrieval",
]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Build confirmed billing pricing patch and unresolved list from an issue worklist JSON."
    )
    parser.add_argument("input", help="Path to billing_pricing_patch issue worklist JSON")
    parser.add_argument(
        "--output-dir",
        default=".",
        help="Directory to write generated artifacts into (default: current directory)",
    )
    return parser.parse_args()


def load_json(path: Path) -> dict[str, Any]:
    with path.open("r", encoding="utf-8") as handle:
        payload = json.load(handle)
    if not isinstance(payload, dict):
        raise ValueError("root JSON must be an object")
    return payload


def clone_layer(value: Any) -> dict[str, Any]:
    if isinstance(value, dict):
        cloned = deepcopy(value)
        if not isinstance(cloned.get("special"), dict):
            cloned["special"] = {}
        if "item_multipliers" in cloned and not isinstance(cloned["item_multipliers"], dict):
            cloned["item_multipliers"] = {}
        return cloned
    return {
        "special_enabled": False,
        "special": {},
        "tiered_enabled": False,
        "multiplier_enabled": False,
    }


def has_non_empty_patch(value: Any) -> bool:
    if not isinstance(value, dict):
        return False
    return any(True for _ in value.keys())


def build_confirmed_layer(layer: dict[str, Any]) -> dict[str, Any]:
    patch: dict[str, Any] = {}
    has_confirmed_value = False

    for key in ROOT_FIELDS:
        if key in layer and layer[key] is not None:
            patch[key] = layer[key]
            has_confirmed_value = True

    if "multiplier_mode" in layer and layer["multiplier_mode"] is not None:
        patch["multiplier_mode"] = layer["multiplier_mode"]

    special = layer.get("special")
    if isinstance(special, dict):
        special_patch = {key: special[key] for key in SPECIAL_FIELDS if special.get(key) is not None}
        if special_patch:
            patch["special"] = special_patch
            has_confirmed_value = True

    item_multipliers = layer.get("item_multipliers")
    if isinstance(item_multipliers, dict) and item_multipliers:
        patch["item_multipliers"] = {
            key: value for key, value in item_multipliers.items() if value is not None
        }
        if patch["item_multipliers"]:
            has_confirmed_value = True

    if not has_confirmed_value:
        return {}

    for key in BOOLEAN_FIELDS:
        if key in layer:
            patch[key] = bool(layer[key])
    if "special" not in patch and bool(layer.get("special_enabled")):
        patch["special"] = {}

    return patch


def build_confirmed_entry(entry: dict[str, Any]) -> dict[str, Any] | None:
    current = entry.get("current") if isinstance(entry.get("current"), dict) else {}
    official_layer = clone_layer(current.get("official"))
    sale_layer = clone_layer(current.get("sale"))

    official_patch = build_confirmed_layer(official_layer)
    sale_patch = build_confirmed_layer(sale_layer)

    patch: dict[str, Any] = {}
    if official_patch:
        patch["official"] = official_patch
    if sale_patch:
        patch["sale"] = sale_patch
    if not patch:
        return None

    return {
        "model": entry.get("model"),
        "display_name": entry.get("display_name"),
        "provider": entry.get("provider"),
        "mode": entry.get("mode"),
        "currency": entry.get("currency"),
        "pricing_status": entry.get("pricing_status"),
        "pricing_warnings": entry.get("pricing_warnings") or [],
        "current": {
            "official": official_layer,
            "sale": sale_layer,
        },
        "patch": patch,
        "notes": "Auto-built confirmed patch from current.official/current.sale known fields only.",
    }


def layer_has_confirmed_fields(layer: dict[str, Any]) -> bool:
    return bool(build_confirmed_layer(layer))


def is_unresolved_entry(entry: dict[str, Any]) -> bool:
    patch = entry.get("patch")
    if has_non_empty_patch(patch):
        return False
    current = entry.get("current") if isinstance(entry.get("current"), dict) else {}
    official_layer = clone_layer(current.get("official"))
    sale_layer = clone_layer(current.get("sale"))
    return not layer_has_confirmed_fields(official_layer) and not layer_has_confirmed_fields(sale_layer)


def build_unresolved_markdown(entries: list[dict[str, Any]], source_name: str) -> str:
    lines = [
        "# MODEL_PRICING_UNRESOLVED",
        "",
        f"- Source: `{source_name}`",
        f"- Generated at: `{datetime.utcnow().isoformat(timespec='seconds')}Z`",
        f"- Total unresolved models: `{len(entries)}`",
        "",
    ]
    for entry in entries:
        model = str(entry.get("model") or "").strip()
        display_name = str(entry.get("display_name") or "").strip() or model
        provider = str(entry.get("provider") or "").strip() or "-"
        currency = str(entry.get("currency") or "").strip() or "USD"
        pricing_status = str(entry.get("pricing_status") or "").strip() or "unknown"
        warnings = entry.get("pricing_warnings") if isinstance(entry.get("pricing_warnings"), list) else []

        lines.append(f"## {display_name}")
        lines.append("")
        lines.append(f"- model: `{model}`")
        lines.append(f"- provider: `{provider}`")
        lines.append(f"- currency: `{currency}`")
        lines.append(f"- pricing_status: `{pricing_status}`")
        if warnings:
          for warning in warnings:
              lines.append(f"- warning: {warning}")
        else:
            lines.append("- warning: none")
        lines.append("")
    return "\n".join(lines).rstrip() + "\n"


def main() -> int:
    args = parse_args()
    input_path = Path(args.input).resolve()
    output_dir = Path(args.output_dir).resolve()
    output_dir.mkdir(parents=True, exist_ok=True)

    payload = load_json(input_path)
    models = payload.get("models")
    if not isinstance(models, list):
        raise ValueError("models must be an array")

    confirmed_models = []
    unresolved_models = []
    for raw in models:
        if not isinstance(raw, dict):
            continue
        confirmed_entry = build_confirmed_entry(raw)
        if confirmed_entry is not None:
            confirmed_models.append(confirmed_entry)
        if is_unresolved_entry(raw):
            unresolved_models.append(raw)

    stamp = datetime.utcnow().strftime("%Y%m%d_%H%M%S")
    confirmed_payload = {
        "version": 1,
        "kind": "billing_pricing_patch",
        "generated_at": datetime.utcnow().isoformat(timespec="seconds") + "Z",
        "export_mode": "executable_template",
        "models": confirmed_models,
    }

    confirmed_path = output_dir / f"billing_pricing_patch_confirmed_{stamp}.json"
    unresolved_path = output_dir / f"MODEL_PRICING_UNRESOLVED_{stamp}.md"

    with confirmed_path.open("w", encoding="utf-8") as handle:
        json.dump(confirmed_payload, handle, ensure_ascii=False, indent=2)
        handle.write("\n")
    unresolved_path.write_text(
        build_unresolved_markdown(unresolved_models, input_path.name),
        encoding="utf-8",
    )

    print(f"Wrote confirmed patch: {confirmed_path}")
    print(f"Wrote unresolved list: {unresolved_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
