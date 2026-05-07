import json
import subprocess
import sys
from pathlib import Path


def test_build_billing_pricing_artifacts(tmp_path: Path) -> None:
    payload = {
        "version": 1,
        "kind": "billing_pricing_patch",
        "generated_at": "2026-05-05T12:17:43Z",
        "export_mode": "issue_worklist",
        "models": [
            {
                "model": "gpt-5.4",
                "display_name": "GPT-5.4",
                "provider": "openai",
                "mode": "chat",
                "currency": "USD",
                "pricing_status": "fallback",
                "pricing_warnings": ["sale falls back to official"],
                "current": {
                    "official": {
                        "input_price": 1.5e-6,
                        "tier_threshold_tokens": 200000,
                        "special_enabled": False,
                        "special": {},
                        "tiered_enabled": False,
                        "multiplier_enabled": False,
                    },
                    "sale": {
                        "output_price": 2.5e-6,
                        "shared_multiplier": 1.2,
                        "special_enabled": False,
                        "special": {},
                        "tiered_enabled": False,
                        "multiplier_enabled": False,
                    },
                },
                "patch": {},
                "notes": "",
            },
            {
                "model": "ernie-3.5-8k",
                "display_name": "Ernie-3.5-8k",
                "provider": "baidu",
                "mode": "chat",
                "currency": "CNY",
                "pricing_status": "missing",
                "pricing_warnings": [],
                "current": {
                    "official": {
                        "input_price": 8e-7,
                        "output_price": 2e-6,
                    },
                    "sale": {},
                },
                "patch": {},
                "notes": "",
            },
            {
                "model": "ernie-4.0-8k",
                "display_name": "Ernie-4.0-8k",
                "provider": "baidu",
                "mode": "chat",
                "currency": "USD",
                "pricing_status": "missing",
                "pricing_warnings": ["No stable upstream pricing source found."],
                "current": {
                    "official": {
                        "special_enabled": False,
                        "special": {},
                        "tiered_enabled": False,
                        "multiplier_enabled": False,
                    },
                    "sale": {
                        "special_enabled": False,
                        "special": {},
                        "tiered_enabled": False,
                        "multiplier_enabled": False,
                    },
                },
                "patch": {},
                "notes": "",
            },
        ],
    }
    input_path = tmp_path / "billing_pricing_patch_issue.json"
    input_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")

    script_path = Path(__file__).resolve().parent / "build_billing_pricing_artifacts.py"
    subprocess.run(
        [sys.executable, str(script_path), str(input_path), "--output-dir", str(tmp_path)],
        check=True,
    )

    confirmed_files = sorted(tmp_path.glob("billing_pricing_patch_confirmed_*.json"))
    unresolved_files = sorted(tmp_path.glob("MODEL_PRICING_UNRESOLVED_*.md"))
    assert len(confirmed_files) == 1
    assert len(unresolved_files) == 1

    confirmed_payload = json.loads(confirmed_files[0].read_text(encoding="utf-8"))
    assert confirmed_payload["export_mode"] == "executable_template"
    assert len(confirmed_payload["models"]) == 2

    confirmed_by_model = {
        item["model"]: item for item in confirmed_payload["models"]
    }

    usd_model = confirmed_by_model["gpt-5.4"]
    assert usd_model["currency"] == "USD"
    assert usd_model["patch"]["official"] == {"input_price": 1.5e-6}
    assert usd_model["patch"]["sale"] == {"output_price": 2.5e-6}

    cny_model = confirmed_by_model["ernie-3.5-8k"]
    assert cny_model["currency"] == "CNY"
    assert cny_model["patch"]["official"] == {
        "input_price": 8e-7,
        "output_price": 2e-6,
    }
    assert "sale" not in cny_model["patch"]

    unresolved_text = unresolved_files[0].read_text(encoding="utf-8")
    assert "ernie-4.0-8k" in unresolved_text
    assert "pricing_status: `missing`" in unresolved_text
    assert "No stable upstream pricing source found." in unresolved_text
