import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from model_registry_official_audit import (  # noqa: E402
    audit_provider,
    collect_provider_models,
    collect_provider_seed_models,
    generate_audit_report,
    parse_seed_entries,
)


def test_collect_provider_models_and_report_generation(tmp_path: Path) -> None:
    seed_path = tmp_path / "registry_seed.json"
    seed_path.write_text(
        """
[
  {
    "id": "gemini-3-pro-preview",
    "provider": "gemini",
    "protocol_ids": ["models/gemini-3-pro-preview"],
    "aliases": []
  },
  {
    "id": "deepseek-v3",
    "provider": "deepseek",
    "display_name": "Deepseek-v3",
    "protocol_ids": ["deepseek-v3"],
    "aliases": []
  },
  {
    "id": "deepseek-chat",
    "provider": "deepseek",
    "display_name": "DeepSeek Chat",
    "protocol_ids": ["deepseek-chat"],
    "aliases": [],
    "status": "deprecated",
    "replaced_by": "deepseek-v4-flash"
  },
  {
    "id": "kimi-latest",
    "provider": "moonshot",
    "display_name": "Kimi-latest",
    "protocol_ids": [],
    "aliases": []
  },
  {
    "id": "gpt-5.4",
    "provider": "openai",
    "display_name": "GPT-5.4",
    "protocol_ids": ["gpt-5.4"],
    "aliases": []
  }
]
""".strip(),
        encoding="utf-8",
    )

    entries = parse_seed_entries(seed_path)
    provider_models = collect_provider_models(entries)
    assert provider_models["gemini"] == {"gemini-3-pro-preview"}
    assert provider_models["deepseek"] == {"deepseek-v3", "deepseek-chat"}
    assert provider_models["moonshot"] == {"kimi-latest"}
    assert provider_models["openai"] == {"gpt-5.4"}

    provider_seed_models = collect_provider_seed_models(entries)

    gemini_row = audit_provider("gemini", provider_seed_models["gemini"])
    assert gemini_row.hard_remove == ["gemini-3-pro-preview"]
    assert gemini_row.keep_and_normalize == []

    deepseek_row = audit_provider("deepseek", provider_seed_models["deepseek"])
    assert deepseek_row.hard_remove == ["deepseek-chat"]
    assert deepseek_row.keep_and_normalize == ["deepseek-v3"]

    moonshot_row = audit_provider("moonshot", provider_seed_models["moonshot"])
    assert moonshot_row.hard_remove == []
    assert moonshot_row.skip_source_insufficient == ["kimi-latest"]
    assert moonshot_row.naming_only_fix == ["kimi-latest"]

    openai_row = audit_provider("openai", provider_seed_models["openai"])
    assert openai_row.keep_as_is == ["gpt-5.4"]
    assert openai_row.hard_remove == []

    output_path = tmp_path / "model_registry_official_audit_20260512_phase2.md"
    generate_audit_report(seed_path=seed_path, output_path=output_path, audit_date="2026-05-12")

    report = output_path.read_text(encoding="utf-8")
    assert "## gemini" in report
    assert "hard_remove：" in report
    assert "- `gemini-3-pro-preview`" in report
    assert "keep_and_normalize：" in report
    assert "- `deepseek-v3`" in report
    assert "skip_source_insufficient" in report
