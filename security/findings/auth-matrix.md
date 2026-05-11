# Sub2API 步骤 3：认证态矩阵与 API Key 生命周期验证

审计日期：2026-05-09  
执行边界：仅限本地 WSL / 授权远程低风险验证  
当前状态：本地 WSL 验证已完成，远程低风险验证阻塞

## 当前说明

- 本文件用于承接步骤 3 的认证态矩阵、API Key 生命周期、用户状态联动与模型边界运行态验证结果。
- 当前 WSL 主机服务回退路径已稳定运行：`server-test-linux` 监听 `127.0.0.1:18081`，`GET /health` 返回 `200 {"status":"ok"}`。
- 本轮已完成：
  - anonymous / normal user / admin / disabled user 的运行态矩阵
  - API Key 生命周期
  - 新建分组 key 的即时可用性验证
  - 禁用用户后 JWT、API Key、登录链路同时失效验证
  - 分组 key 下的模型别名 / 大小写 / 负向模型边界补证
- 远程低风险验证仍缺测试账号、测试 API Key、测试窗口，因此远程矩阵验证尚未执行。

## 本地基线证据

- 监听状态：`127.0.0.1:18081` 由 `server-test-linux` 监听，`127.0.0.1:19090` 由 `mock_upstream.py` 监听。
- 健康检查：`GET /health` 返回 `200`，响应体为 `{"status":"ok"}`。
- 参考证据文件：
  - `security/reports/manual/2026-05-09-auth-admin-boundary.md`
  - `security/reports/manual/2026-05-09-apikey-lifecycle.md`
  - `security/reports/manual/mock-upstream.log`
  - `security/reports/manual/server-audit.log`

## 本地认证态与管理员边界结果

| 身份 | 路径 | 预期 | 实际 | 结论 |
|---|---|---|---|---|
| anonymous | `GET /api/v1/auth/me` | 未登录拒绝 | `401`，消息 `Authorization header is required` | done |
| admin JWT | `GET /api/v1/auth/me` | 返回管理员资料 | `200`，返回 `admin.audit@example.local` 管理员资料 | done |
| normal user JWT | `GET /api/v1/auth/me` | 返回普通用户资料 | `200`，返回本地临时普通用户资料 | done |
| disabled user JWT | `GET /api/v1/auth/me` | 禁用后立即失效 | `401`，原因 `USER_INACTIVE` | done |
| disabled user credentials | `POST /api/v1/auth/login` | 禁用后拒绝再次登录 | `403`，原因 `USER_NOT_ACTIVE` | done |
| anonymous | `GET /api/v1/admin/users?page=1&page_size=1` | 管理端拒绝 | `401`，消息 `Authorization required` | done |
| normal user JWT | `GET /api/v1/admin/users?page=1&page_size=1` | 非管理员拒绝 | `403`，消息 `Admin access required` | done |
| admin JWT | `GET /api/v1/admin/users?page=1&page_size=1` | 管理员可访问 | `200`，成功返回分页用户列表 | done |
| anonymous | `GET /api/v1/admin/data-management/agent/health` | 管理端拒绝 | `401`，消息 `Authorization required` | done |
| normal user JWT | `GET /api/v1/admin/data-management/agent/health` | 非管理员拒绝 | `403`，消息 `Admin access required` | done |
| admin JWT | `GET /api/v1/admin/data-management/agent/health` | 返回退化健康态 | `200`，`enabled=false`，`reason=DATA_MANAGEMENT_DEPRECATED` | done |
| admin JWT | `GET /api/v1/admin/data-management/config` | 退化链路拒绝 | `503`，`reason=DATA_MANAGEMENT_DEPRECATED` | done |
| anonymous | `GET /api/v1/admin/backups/s3-config` | 管理端拒绝 | `401`，消息 `Authorization required` | done |
| normal user JWT | `GET /api/v1/admin/backups/s3-config` | 非管理员拒绝 | `403`，消息 `Admin access required` | done |
| admin JWT | `GET /api/v1/admin/backups/s3-config` | 管理员可读取配置 | `200`，返回空白审计基线配置 | done |
| admin JWT | `POST /api/v1/admin/backups/s3-config/test` 缺 `secret_access_key` | 输入校验拒绝 | `400`，`reason=BACKUP_S3_TEST_SECRET_REQUIRED` | done |

## 本地 API Key 生命周期与用户状态联动

- 正式生命周期验证入口统一使用 `POST /v1/messages`。
- `GET /v1/usage` 会走 `skipBilling=true`，不适合作为 `expired` / `quota_exhausted` 的正式生命周期证据。

| 场景 | 路径 | 预期 | 实际 | 结论 |
|---|---|---|---|---|
| active API key | `POST /v1/messages` | 生命周期通过，进入后续计费/路由阶段 | `403`，`reason=INSUFFICIENT_BALANCE` | done |
| disabled API key | `POST /v1/messages` | 禁用后立即拒绝 | `401`，`reason=API_KEY_DISABLED` | done |
| expired API key | `POST /v1/messages` | 过期后拒绝 | `403`，`reason=API_KEY_EXPIRED` | done |
| quota exhausted API key | `POST /v1/messages` | 配额耗尽后拒绝 | `429`，`reason=API_KEY_QUOTA_EXHAUSTED` | done |
| deleted API key | `POST /v1/messages` | 删除后拒绝 | `401`，`reason=INVALID_API_KEY` | done |
| groupless API key | `POST /v1/messages` | 未绑定分组时拒绝调度 | `403`，消息 `API Key is not assigned to any group...` | done |
| funded active API key | `POST /v1/messages` | 穿过生命周期与余额校验，进入账号调度 | `503`，`reason=GROUP_EXHAUSTED` | done |
| grouped active API key | `POST /v1/messages` | 新建后可立即命中受控本地 upstream | `200`，命中 `http://127.0.0.1:19090/echo-headers/v1/messages?beta=true` | done |
| disabled user + active API key | `POST /v1/messages` | 禁用用户后 key 同步失效 | `401`，原因 `USER_INACTIVE` | done |

## 新建 key 即时可用性补证

- `user 13` 在本轮新建后，`POST /api/v1/keys` 成功返回 `key 13`，并带 `group_id: 2`。
- 完整 key 只在创建响应中可见一次，符合“一次性展示”的预期。
- 该 grouped key 在创建后立即可用于 `POST /v1/messages`，无需二次刷新或重新登录。

## 分组 key 的模型边界补证

本轮最终模型边界验证在 `group_id: 2`、受控本地 anthropic mock upstream、`policy_projection=account_scope` 且 `count=1` 的路径下执行。

| 模型输入 | 实际 | 关键证据 | 结论 |
|---|---|---|---|
| `claude-sonnet-4.5` | `200` | mock 记录 `x-client-request-id=step3c-allowed-1778328548` | done |
| `claude-sonnet-4-5` | `200` | mock 记录 `x-client-request-id=step3c-alias-1778328548` | done |
| `CLAUDE-SONNET-4-5` | `200` | mock 记录 `x-client-request-id=step3c-case_variant-1778328548` | done |
| `gpt-5.4` | `503`，`GROUP_EXHAUSTED` | 服务日志记录 `user_id=13 api_key_id=13 group_id=2 model=gpt-5.4`，未见对应 mock 命中 | done_with_note |

补充说明：

- 上述负向结果说明在本轮收紧到单模型的 `group 2` 路径下，`gpt-5.4` 未观察到未授权 upstream 命中。
- 但更早一轮在 `account_id=11`、`policy_projection=default_library`、`count=9` 的较宽策略下，`gpt-5.4` 曾返回 `200`；因此本轮不把 `gpt-5.4 -> 503 GROUP_EXHAUSTED` 单独升级为新漏洞，只作为“收紧策略路径下未见绕过”的说明保留。

## 计划覆盖的身份类型

| 身份类型 | 本地 WSL | 远程低风险 |
|---|---|---|
| anonymous | done | blocked |
| normal_user_a | done | blocked |
| normal_user_b | done | blocked |
| disabled_user | done | blocked |
| admin | done | blocked |
| active_api_key | done | blocked |
| grouped_active_api_key | done | blocked |
| disabled_api_key | done | blocked |
| deleted_api_key | done | blocked |
| expired_api_key | done | blocked |
| quota_zero_api_key | done | blocked |

## 阻塞项

- WSL 容器链路仍受 Docker Hub 出站超时影响，当前深测继续使用主机服务回退路径。
- 远程低风险验证缺测试账号、测试 API Key、测试窗口与目标 IP / 授权范围补充信息。

## 当前暂停点

- 步骤 3 的本地认证态矩阵与 API Key 生命周期验证已完成并留痕。
- 后续等待你修复 `security/findings/wsl-deep-test-findings.md` 中的问题，再进入步骤 4 做对照复审。
