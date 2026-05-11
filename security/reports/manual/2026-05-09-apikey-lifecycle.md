# 2026-05-09 API Key 生命周期证据

## 运行基线

- `server-test-linux` 监听：`127.0.0.1:18081`
- `mock_upstream.py` 监听：`127.0.0.1:19090`
- `GET /health`：`200 {"status":"ok"}`
- 本轮不记录明文 API Key，仅保留状态、HTTP 返回码与关键 reason。

## 验证入口选择

- `GET /v1/usage` 只适合验证 key 存在性、disabled、deleted 等基础鉴权差异，不适合作为 `expired` / `quota_exhausted` 的正式证据入口。
- 原因是 `backend/internal/server/middleware/api_key_auth.go` 对 `/v1/usage` 走 `skipBilling=true` 分支，会跳过生命周期后半段的过期与配额拦截。
- 因此本轮正式生命周期矩阵统一以 `POST /v1/messages` 为准，并携带 `anthropic-version: 2023-06-01`。

## `/v1/messages` 生命周期矩阵

| 场景 | HTTP | reason / code | 说明 |
|---|---:|---|---|
| active key | 403 | `INSUFFICIENT_BALANCE` | 已通过 key 生命周期检查，但因用户钱包余额不足被后续计费逻辑拒绝 |
| disabled key | 401 | `API_KEY_DISABLED` | key 被显式禁用后立即拒绝 |
| expired key | 403 | `API_KEY_EXPIRED` | 过期态由运行时生命周期检查拦截 |
| quota exhausted key | 429 | `API_KEY_QUOTA_EXHAUSTED` | 配额耗尽态由运行时生命周期检查拦截 |
| deleted key | 401 | `INVALID_API_KEY` | 删除后回退为无效 key |

## funded active key 补证

| 场景 | HTTP | reason / code | 说明 |
|---|---:|---|---|
| active key + 用户有余额 | 503 | `GROUP_EXHAUSTED` | 说明请求已通过 key 生命周期与余额校验，继续进入分组/账号选择阶段；当前默认组内没有可用上游账号 |

## mock upstream 接入后复测

| 场景 | HTTP | 关键证据 | 说明 |
|---|---:|---|---|
| active key + 用户有余额 + 默认组已挂本地 anthropic mock 账号 | 200 | 命中 `http://127.0.0.1:19090/v1/messages?beta=true` | 说明默认组已具备可调度上游账号，请求可穿过生命周期、余额与分组选择并到达受控本地 upstream |

## 新增日志锚点

- `2026-05-09 17:32:34 +08:00`：管理端 `POST /api/v1/admin/accounts/1/test` 命中 `anthropic->anthropic:native_passthrough`
- `2026-05-09 17:34:06 +08:00`：有余额用户 API Key 访问 `POST /v1/messages` 返回 `200`，服务日志记录 `account_id=1`

## 日志锚点

- `security/reports/manual/server-audit.log`
- `2026-05-09 17:14:16 +08:00` 到 `17:14:17 +08:00`：`/v1/messages` 生命周期矩阵
- `2026-05-09 17:15:21 +08:00`：funded active key 返回 `GROUP_EXHAUSTED`
