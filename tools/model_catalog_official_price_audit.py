#!/usr/bin/env python3
import argparse
import csv
import json
import re
import sys
from pathlib import Path


DATE_SUFFIX = re.compile(r"-(?:\d{8}|\d{4}-\d{2}-\d{2})$")
OPENAI_REASONING = re.compile(r"^o\d")
TOKEN_FIELDS = {
    "input_cost_per_token",
    "input_cost_per_token_priority",
    "input_cost_per_token_above_threshold",
    "input_cost_per_token_priority_above_threshold",
    "output_cost_per_token",
    "output_cost_per_token_priority",
    "output_cost_per_token_above_threshold",
    "output_cost_per_token_priority_above_threshold",
    "cache_creation_input_token_cost",
    "cache_creation_input_token_cost_above_1hr",
    "cache_read_input_token_cost",
    "cache_read_input_token_cost_priority",
}
INT_FIELDS = {"input_token_threshold", "output_token_threshold"}
PRICING_FIELDS = [
    "currency",
    "input_cost_per_token",
    "input_cost_per_token_priority",
    "input_token_threshold",
    "input_cost_per_token_above_threshold",
    "input_cost_per_token_priority_above_threshold",
    "output_cost_per_token",
    "output_cost_per_token_priority",
    "output_token_threshold",
    "output_cost_per_token_above_threshold",
    "output_cost_per_token_priority_above_threshold",
    "cache_creation_input_token_cost",
    "cache_creation_input_token_cost_above_1hr",
    "cache_read_input_token_cost",
    "cache_read_input_token_cost_priority",
    "output_cost_per_image",
]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--snapshot",
        default="backend/resources/model-pricing/model_prices_and_context_window.json",
    )
    parser.add_argument("--overrides-json")
    parser.add_argument("--output", required=True)
    parser.add_argument(
        "--families",
        nargs="+",
        default=["openai", "anthropic", "gemini"],
    )
    return parser.parse_args()


def canonicalize(model: str) -> str:
    value = model.strip().lstrip("/")
    for prefix in ("models/", "publishers/google/models/"):
        if value.startswith(prefix):
            value = value[len(prefix) :]
    if "/publishers/google/models/" in value:
        value = value.split("/publishers/google/models/")[-1]
    if "/models/" in value:
        value = value.split("/models/")[-1]
    return value.lstrip("/").lower()


def format_display_name(model: str) -> str:
    trimmed = DATE_SUFFIX.sub("", canonicalize(model))
    parts = trimmed.split("-")
    if not parts:
        return trimmed
    parts[0] = {
        "claude": "Claude",
        "gpt": "GPT",
        "gemini": "Gemini",
        "sora": "Sora",
        "codex": "Codex",
    }.get(parts[0], parts[0].upper() if OPENAI_REASONING.match(parts[0]) else parts[0].capitalize())
    return "-".join(parts)


def infer_family(model: str, provider: str) -> str:
    canonical = canonicalize(model)
    if canonical.startswith("claude"):
        return "anthropic"
    if canonical.startswith("gemini"):
        return "gemini"
    if canonical.startswith(("gpt", "sora", "codex")) or OPENAI_REASONING.match(canonical):
        return "openai"
    lowered_provider = provider.strip().lower()
    return lowered_provider if lowered_provider in {"openai", "anthropic", "gemini"} else "other"


def load_json(path: str | None) -> dict:
    if not path:
        return {}
    with open(path, "r", encoding="utf-8") as handle:
        payload = json.load(handle)
    if isinstance(payload, str):
        payload = json.loads(payload)
    return payload if isinstance(payload, dict) else {}


def normalize_overrides(raw: dict) -> dict:
    result = {}
    for model, override in raw.items():
        if isinstance(override, dict):
            result[canonicalize(model)] = override
    return result


def format_value(field: str, value) -> str:
    if value is None:
        return ""
    if field == "currency":
        currency = str(value).strip().upper()
        return currency if currency else "USD"
    if field in INT_FIELDS:
        return str(int(value))
    if field in TOKEN_FIELDS:
        return f"{float(value) * 1_000_000:.6f}".rstrip("0").rstrip(".")
    return f"{float(value):.6f}".rstrip("0").rstrip(".")


def merge_pricing(base: dict, override: dict) -> dict:
    merged = {field: base.get(field) for field in PRICING_FIELDS}
    merged["currency"] = merged.get("currency") or "USD"
    for field in PRICING_FIELDS:
        if override.get(field) is not None:
            merged[field] = override[field]
    return merged


def build_headers() -> list[str]:
    headers = ["model", "display_name", "family", "provider", "mode"]
    for prefix in ("upstream", "official_override", "effective_official"):
        headers.extend(f"{prefix}_{field}" for field in PRICING_FIELDS)
    headers.extend(["manual_review_status", "manual_notes"])
    return headers


def build_rows(snapshot: dict, overrides: dict, families: set[str]) -> list[dict[str, str]]:
    rows = []
    for model, pricing in snapshot.items():
        if not isinstance(pricing, dict):
            continue
        family = infer_family(model, str(pricing.get("litellm_provider", "")))
        if family not in families:
            continue
        canonical = canonicalize(model)
        official_override = overrides.get(canonical, {})
        effective = merge_pricing(pricing, official_override)
        row = {
            "model": canonical,
            "display_name": format_display_name(model),
            "family": family,
            "provider": str(pricing.get("litellm_provider", "")),
            "mode": str(pricing.get("mode", "")),
            "manual_review_status": "pending",
            "manual_notes": "",
        }
        for field in PRICING_FIELDS:
            row[f"upstream_{field}"] = format_value(field, pricing.get(field))
            row[f"official_override_{field}"] = format_value(field, official_override.get(field))
            row[f"effective_official_{field}"] = format_value(field, effective.get(field))
        rows.append(row)
    return sorted(rows, key=lambda item: (item["family"], item["display_name"], item["model"]))


def main() -> int:
    args = parse_args()
    snapshot = load_json(args.snapshot)
    overrides = normalize_overrides(load_json(args.overrides_json))
    output_path = Path(args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)

    rows = build_rows(snapshot, overrides, {family.lower() for family in args.families})
    with output_path.open("w", encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=build_headers())
        writer.writeheader()
        writer.writerows(rows)

    print(f"Wrote {len(rows)} audit rows to {output_path}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
