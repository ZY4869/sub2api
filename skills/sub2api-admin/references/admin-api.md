# Sub2API Admin API Reference

This reference is intentionally narrow. It documents only the authenticated HTTP admin surface used by the bundled script.

## Environment

- `SUB2API_ADMIN_BASE_URL`: Base URL for the Sub2API server, for example `http://localhost:8080`.
- `SUB2API_ADMIN_TOKEN`: Admin bearer token.
- `SUB2API_ADMIN_DRY_RUN`: Optional. Defaults to dry-run mode. Set to `0`, `false`, or `no` only after explicit approval.

## Authentication

Send the admin token as:

```http
Authorization: Bearer <token>
```

Do not log the token.

## Supported Endpoints

```http
GET /api/v1/admin/health
GET /api/v1/admin/proxies
GET /api/v1/admin/accounts
POST /api/v1/admin/accounts/{id}/restore-original-proxy
```

If a deployment does not provide `/api/v1/admin/health`, use a read-only list endpoint to confirm connectivity.

## Response Handling

- Treat non-2xx responses as failures.
- Print status code and a sanitized response summary only.
- Redact keys named `token`, `api_key`, `access_token`, `refresh_token`, `password`, `secret`, `credentials`, and `authorization`.
- Avoid dumping large arrays by default; summarize counts and IDs.

## Restore Original Proxy

The restore action asks the backend to move an account from its expiry fallback proxy back to the recorded original proxy.

Dry-run request preview:

```bash
node skills/sub2api-admin/scripts/sub2api-admin.js restore-original-proxy --account-id 123
```

Execution after approval:

```bash
SUB2API_ADMIN_DRY_RUN=0 node skills/sub2api-admin/scripts/sub2api-admin.js restore-original-proxy --account-id 123
```
