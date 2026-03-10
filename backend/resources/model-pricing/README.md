# Model Pricing Data

This directory contains a local copy of the mirrored model pricing data as a fallback mechanism.

## Source
The original file is maintained by the LiteLLM project and mirrored into the `price-mirror` branch of this repository via GitHub Actions:
- Mirror branch (configurable via `PRICE_MIRROR_REPO`): https://raw.githubusercontent.com/<your-repo>/price-mirror/model_prices_and_context_window.json
- Upstream source: https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json

## Purpose
This local copy serves as a fallback when the remote file cannot be downloaded due to:
- Network restrictions
- Firewall rules
- DNS resolution issues
- GitHub being blocked in certain regions
- Docker container network limitations

## Update Process
The pricingService will:
1. First attempt to download the latest version from GitHub
2. If download fails, use this local copy as fallback
3. Log a warning when using the fallback file

## Manual Update
To manually update this file with the latest pricing data (if automation is unavailable):
```bash
curl -s https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json -o model_prices_and_context_window.json
```

## Official Pricing Audit Support

The model catalog keeps a separate official pricing override layer on top of this upstream snapshot.

- Baseline snapshot: this file
- Official pricing patch layer: `settings.model_official_price_overrides`
- Sale pricing patch layer: `settings.model_price_overrides`

For manual audit / review, use:

```bash
python tools/model_catalog_official_price_audit.py \
  --output docs/model_catalog_official_pricing_audit.csv
```

If you have exported the current official override JSON, you can include it with:

```bash
python tools/model_catalog_official_price_audit.py \
  --overrides-json tmp/model_official_price_overrides.json \
  --output docs/model_catalog_official_pricing_audit.csv
```

See `docs/MODEL_CATALOG_PRICING_CN.md` for the full maintenance workflow.

## File Format
The file contains JSON data with model pricing information including:
- Model names and identifiers
- Input/output token costs
- Context window sizes
- Model capabilities

Last updated: 2025-08-10
