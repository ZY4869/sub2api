# 2026-05-09 Base URL / SSRF / 出站验证证据

## 运行基线

- `server-test-linux` 监听：`127.0.0.1:18081`
- `mock_upstream.py` 监听：`127.0.0.1:19090`
- 当前仅访问本地 mock upstream，不访问真实第三方、云 metadata 或未授权目标

## 源码反向定位结论

- 默认安全基线未收紧上游地址：
  - `backend/internal/config/config_defaults.go`
  - `security.url_allowlist.enabled=false`
  - `security.url_allowlist.allow_private_hosts=true`
  - `security.url_allowlist.allow_insecure_http=true`
- 管理端账号创建/更新路径本身不预校验 `credentials.base_url`：
  - `backend/internal/handler/admin/account_handler_crud.go`
  - `backend/internal/service/admin_service_accounts.go`
- 保存账号后会自动调度模型探测：
  - `backend/internal/handler/admin/account_handler_model_probe_refresh.go`
  - `backend/internal/service/account_model_import_service.go`
- 运行/测试阶段的 `base_url` 校验发生在更后面：
  - `backend/internal/service/gateway_service.go`
  - `backend/internal/service/openai_gateway_upstream_request.go`
  - `backend/internal/service/account_test_service.go`
  - `backend/internal/util/urlvalidator/validator.go`

## 本地运行态验证矩阵

| account_id | 账号名 | `base_url` | 创建结果 | 自动 probe 证据 | `real_forward` 结果 | 结论 |
|---|---|---|---|---|---|---|
| 2 | `audit-http-ok` | `http://127.0.0.1:19090` | `200` | mock 收到 `GET /v1/models` | `success`，mock 收到 `POST /v1/messages?beta=true` | 允许本地 HTTP + 私网地址，且保存后立即真实出站 |
| 3 | `audit-http-path` | `http://127.0.0.1:19090/echo-headers` | `200` | mock 收到 `GET /echo-headers/v1/models` | `success`，mock 收到 `POST /echo-headers/v1/messages?beta=true` | 路径前缀可控，运行态会拼接到用户提供路径后 |
| 4 | `audit-file-scheme` | `file:///etc/passwd` | `200` | 服务日志出现 `Get "file:///etc/passwd/v1/models"` | `failed`，`invalid base_url: invalid url: file:///etc/passwd` | 写入前未阻断，保存后已触发后台真实探测尝试 |
| 5 | `audit-gopher` | `gopher://127.0.0.1:70` | `200` | 服务日志出现 `Get "gopher://127.0.0.1:70/v1/models"` | `failed`，`invalid base_url: invalid url scheme: gopher` | 写入前未阻断，保存后已触发后台真实探测尝试 |
| 6 | `audit-bad-port` | `http://127.0.0.1:99999` | `200` | 服务日志出现 `Get "http://127.0.0.1:99999/v1/models"` | `failed`，`invalid base_url: invalid port: 99999` | 写入前未阻断，保存后已触发后台真实探测尝试 |
| 7 | `audit-empty-host` | `http:///missing-host` | `200` | 服务日志出现 `Get "http:///missing-host/v1/models"` | `failed`，`invalid base_url: invalid url: http:///missing-host` | 写入前未阻断，保存后已触发后台真实探测尝试 |
| 8 | `audit-redirect-local` | `http://127.0.0.1:19090/redirect-local` | `200` | mock 收到 `GET /redirect-local/v1/models` | `success`，mock 先收 `POST /redirect-local/v1/messages?beta=true`，随后收 `GET /final/v1/messages?beta=true&redirected=1` | `real_forward` 会跟随 `302`，且 `POST` 被降级为 `GET` |

## 关键观测

- 管理端保存账号时允许写入明显异常的 `base_url`，没有在写入前返回 `4xx`。
- 保存后后台自动模型探测会立即对该 `base_url` 发起真实出站请求，即使后续 `real_forward` 阶段会报 `invalid base_url`。
- `real_forward` 对重定向路径会跟随 `302`，且从 `POST` 跟到了 `GET`。
- 本轮重定向观测来自 `real_forward`，不是 `GET /v1/models` 的模型 probe。

## 关键日志锚点

- `security/reports/manual/mock-upstream.log`
  - `2026-05-09T18:02:31+0800`：`/v1/models` 与 `/echo-headers/v1/models`
  - `2026-05-09T18:07:46+0800`：`/redirect-local/v1/messages?beta=true`
  - `2026-05-09T18:07:46+0800`：`/final/v1/messages?beta=true&redirected=1`
- `security/reports/manual/server-audit.log`
  - `account_id=4`：`invalid url: file:///etc/passwd`
  - `account_id=5`：`invalid url scheme: gopher`
  - `account_id=6`：`invalid port: 99999`
  - `account_id=7`：`invalid url: http:///missing-host`
  - `account_id=8`：`account_test_complete status=success`

## 当前结论

- 当前 `base_url` 安全边界并不在管理端写入前生效，而是在保存后后台 probe 或运行态测试阶段才部分生效。
- 这使得“管理员保存上游账号”本身就成为可触发真实出站访问的动作。
- 现有 `302` 跟随行为说明后续仍需继续验证：
  - 是否存在跨主机重定向时的敏感头保留
  - 是否会在重定向后重新校验目标地址
  - 用户态网关请求是否允许覆盖内部 `Authorization` / `Cookie` / `x-api-key`
