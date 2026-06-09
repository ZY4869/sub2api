---
name: sub2api-admin
description: Use this skill when administering a Sub2API instance through its authenticated HTTP admin API, including inspecting accounts, proxies, usage, and ops status with dry-run-first safety.
---

# Sub2API Admin

Use this skill for operational admin tasks against a running Sub2API deployment.

## Safety Rules

- Use only the HTTP admin API. Do not connect to the database, edit runtime config files, or bypass authentication.
- Read `SUB2API_ADMIN_BASE_URL` and `SUB2API_ADMIN_TOKEN` from the environment.
- Default to dry-run behavior for write-like operations. Require an explicit dry-run override before making changes.
- Never print admin tokens, account credentials, proxy passwords, API keys, or full upstream error bodies.
- Prefer resource IDs and redacted names in logs.

## Workflow

1. Read [references/admin-api.md](references/admin-api.md) when you need endpoint shapes or script examples.
2. Use `scripts/sub2api-admin.js` for repeatable checks and safe admin actions.
3. For changes, first run with default dry-run mode and inspect the planned request.
4. Only run with `SUB2API_ADMIN_DRY_RUN=0` when the user has explicitly approved the exact action.

## Script Quick Start

```bash
SUB2API_ADMIN_BASE_URL=http://localhost:8080 \
SUB2API_ADMIN_TOKEN=... \
node skills/sub2api-admin/scripts/sub2api-admin.js health
```

Common commands:

- `health`: check admin API reachability.
- `list-proxies`: list proxies without credentials.
- `list-accounts`: list accounts without credentials.
- `restore-original-proxy --account-id <id>`: plan or execute the account proxy restore action.
