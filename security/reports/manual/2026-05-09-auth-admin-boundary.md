# 2026-05-09 本地认证与管理员边界证据

## 运行基线

- `server-test-linux` 监听：`127.0.0.1:18081`
- `mock_upstream.py` 监听：`127.0.0.1:19090`
- `GET /health`：`200 {"status":"ok"}`

## 首轮矩阵结果

| 场景 | HTTP | 关键响应 |
|---|---:|---|
| anonymous -> `GET /api/v1/auth/me` | 401 | `Authorization header is required` |
| admin JWT -> `GET /api/v1/auth/me` | 200 | 返回管理员资料 |
| normal user JWT -> `GET /api/v1/auth/me` | 200 | 返回普通用户资料 |
| anonymous -> `GET /api/v1/admin/users?page=1&page_size=1` | 401 | `Authorization required` |
| normal user JWT -> `GET /api/v1/admin/users?page=1&page_size=1` | 403 | `Admin access required` |
| admin JWT -> `GET /api/v1/admin/users?page=1&page_size=1` | 200 | 成功返回分页用户列表 |
| anonymous -> `GET /api/v1/admin/data-management/agent/health` | 401 | `Authorization required` |
| normal user JWT -> `GET /api/v1/admin/data-management/agent/health` | 403 | `Admin access required` |
| admin JWT -> `GET /api/v1/admin/data-management/agent/health` | 200 | `enabled=false`, `reason=DATA_MANAGEMENT_DEPRECATED` |
| admin JWT -> `GET /api/v1/admin/data-management/config` | 503 | `reason=DATA_MANAGEMENT_DEPRECATED` |
| anonymous -> `GET /api/v1/admin/backups/s3-config` | 401 | `Authorization required` |
| normal user JWT -> `GET /api/v1/admin/backups/s3-config` | 403 | `Admin access required` |
| admin JWT -> `GET /api/v1/admin/backups/s3-config` | 200 | 返回空白审计基线配置 |
| admin JWT -> `POST /api/v1/admin/backups/s3-config/test` 缺 `secret_access_key` | 400 | `reason=BACKUP_S3_TEST_SECRET_REQUIRED` |

## 备注

- 本轮仅验证匿名 / 普通用户 / 管理员首轮边界。
- API Key 生命周期、模型权限、额度、限流和受控出站验证待下一步继续。
