# DEPLOYMENT_MAP.md

## 服务映射

| 服务 | 源码/配置 | 环境 | 域名/路径 | 端口 | 是否公网暴露 | 备注 |
|---|---|---|---|---|---|---|
| Web 前端 | `frontend/` | production | `https://api.zyxai.de/`、`https://demo.sub2api.org/` | 443 | 是 | 前端与 API 由同一网关站点对外提供。 |
| API 后端 | `backend/` | production | `/api/v1`、`/v1`、`/v1beta`、`/document-ai/v1` | 443 / 内部 8080 | 是 | 默认 Compose 会直接映射 `8080`。 |
| 管理后台 | `frontend/` 路由 + `backend/internal/server/routes/admin.go` | production | `/admin`、`/api/v1/admin/**` | 443 | 是 | 由管理员 JWT 或 admin API key 保护。 |
| PostgreSQL | `deploy/docker-compose*.yml` | production / WSL | 无 | 5432 | 否 | Compose 默认未映射到公网。 |
| Redis | `deploy/docker-compose*.yml` | production / WSL | 无 | 6379 | 否 | Compose 默认未映射到公网。 |
| datamanagementd | `deploy/install-datamanagementd.sh`、`backend/internal/server/routes/admin.go` | production / WSL | 管理端数据管理入口 | 本地 socket / 管理 API | 否 | 主动验证放步骤 3。 |
| 备份与恢复 | `backend/internal/server/routes/admin.go`、`backend/internal/service/backup_service.go` | production / WSL | `/api/v1/admin/backups/**`、`/api/v1/admin/data-management/**` | 443 | 是 | 高权限管理链路。 |
| WSL staging-like | 待步骤 3 生成 `docker-compose.audit.yml` | WSL | `http://localhost:18080`（计划） | 18080 | 否 | 用于主动 DAST、mock upstream、并发与 SSRF 验证。 |
| mock upstream | `security/mock/mock_upstream.py`（待步骤 3 创建） | WSL | `http://localhost:19090`（计划） | 19090 | 否 | 所有主动上游测试仅指向自建 mock。 |

## 关键配置路径

- Nginx / 边缘代理：仓库未直接提供生产 Nginx，当前仅有 `deploy/Caddyfile` 与 README 反代注意事项
- Docker Compose：
  - `deploy/docker-compose.yml`
  - `deploy/docker-compose.local.yml`
  - `deploy/docker-compose.standalone.yml`
  - `deploy/docker-compose.dev.yml`
- 环境变量：
  - `deploy/.env.example`
  - `deploy/config.example.yaml`
- systemd：
  - `deploy/sub2api.service`
  - `deploy/sub2api-datamanagementd.service`
- 备份与升级：
  - `backend/internal/service/backup_service.go`
  - `backend/internal/server/routes/admin.go`
  - `backend/internal/setup/setup.go`
- 日志与审计：
  - `backend/internal/handler/ops_error_logger.go`
  - `backend/internal/service/ops_service.go`
