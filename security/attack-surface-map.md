# Sub2API 攻击面地图

## 1. 项目模块清单

- 后端 HTTP 服务：`backend/internal/server/`
- 鉴权与会话：`backend/internal/server/middleware/`、`backend/internal/service/auth_*`
- API 网关与上游转发：`backend/internal/server/routes/gateway.go`、`backend/internal/handler/gateway_*`、`backend/internal/service/openai_*`
- 管理后台 API：`backend/internal/server/routes/admin.go`
- 用户与 API Key：`backend/internal/server/routes/user.go`、`backend/internal/server/middleware/api_key_auth.go`
- 公共设置与首页：`backend/internal/service/setting_service_public.go`、`frontend/src/views/HomeView.vue`
- 前端管理台与用户台：`frontend/src/views/**`
- 部署与初始化：`deploy/`、`backend/internal/setup/setup.go`
- 备份/恢复/datamanagement：`backend/internal/service/backup_service.go`、`backend/internal/server/routes/admin.go`

## 2. 主要路由面

| 路由前缀 | 认证方式 | 主要用途 | 主要风险 |
|---|---|---|---|
| `/health` | 无 | 健康检查 | 版本/状态暴露 |
| `/setup/status` | 无 | 设置向导状态 | 初始化流程探测 |
| `/api/v1/auth/**` | 无 / JWT | 注册、登录、刷新、OAuth | 暴力尝试、令牌处理、重置链路 |
| `/api/v1/settings/public` | 无 | 公共设置下发 | HTML/XSS、URL 外链 |
| `/api/v1/pages/:slug` | 可选 JWT | 公共页面 | 自定义页面内容与权限边界 |
| `/api/v1/user/**` | JWT | 用户资料、TOTP、关联身份 | 会话与越权 |
| `/api/v1/keys/**` | JWT | 用户 API Key 生命周期 | 凭据生命周期与 IDOR |
| `/api/v1/admin/**` | Admin JWT / admin API key / 部分 WS 子协议 JWT | 后台配置、账号、日志、备份、升级 | 高权限越权、审计缺失、SSR F、供应链 |
| `/v1/**` | API Key | Anthropic/OpenAI 兼容网关 | 模型权限、计费、限流、日志泄露 |
| `/v1beta/**`、`/v1alpha/**` | API Key | Gemini 原生兼容层 | Query key 兼容、模型权限、上游调度 |
| `/grok/v1/**`、`/deepseek/v1/**` | API Key | 平台兼容路由 | 平台隔离、错误处理 |
| `/document-ai/v1/**` | API Key | 文档 AI | 模型与文件链路 |

## 3. 鉴权链路

- JWT 用户鉴权：
  - 中间件：`backend/internal/server/middleware/jwt_auth.go`
  - 校验内容：Bearer token、用户状态、`TokenVersion`
- 管理员鉴权：
  - 中间件：`backend/internal/server/middleware/admin_auth.go`
  - 允许：admin JWT、admin API key、WebSocket `Sec-WebSocket-Protocol: jwt.<token>`
  - 风险点：高权限入口密集，部分 WS 链路对浏览器子协议传递 JWT 敏感
- API Key 鉴权：
  - 中间件：`backend/internal/server/middleware/api_key_auth.go`
  - 支持头：`Authorization: Bearer`、`x-api-key`、`x-goog-api-key`
  - Google 兼容路径允许 `?key=`
  - 风险点：IP 白名单走 `GetTrustedClientIP`，但大量后续业务链路仍读取 `GetClientIP`

## 4. 公开入口与未登录可访问面

- `GET /health`
- `GET /setup/status`
- `POST /api/event_logging/batch`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/login/2fa`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `POST /api/v1/auth/forgot-password`
- `POST /api/v1/auth/reset-password`
- `GET /api/v1/auth/oauth/**`
- `GET /api/v1/settings/public`
- `GET /api/v1/pages/:slug`
- 前端首页 `/` 和公开文档入口

## 5. 前端与 iframe 集成点

- 首页自定义内容：
  - `frontend/src/views/HomeView.vue`
  - `backend/internal/service/setting_service_public.go`
- 外部 iframe 嵌入：
  - `frontend/src/utils/embedded-url.ts`
  - `frontend/src/views/user/PurchaseSubscriptionView.vue`
  - `frontend/src/views/user/CustomPageView.vue`
- OAuth 回调页：
  - `frontend/src/views/auth/LinuxDoCallbackView.vue`
  - `frontend/src/views/auth/SocialOAuthCallbackView.vue`
- 浏览器令牌持久化：
  - `frontend/src/api/auth.ts`
  - `frontend/src/api/client.ts`
  - `frontend/src/stores/auth.ts`

## 6. 敏感数据清单

| 数据 | 风险 | 当前落点 |
|---|---|---|
| 管理员密码 | 后台接管 | `backend/internal/setup/setup.go`、部署环境变量、README 示例 |
| JWT/Refresh Token | 用户/管理员会话接管 | 前端 `localStorage`、OAuth fragment、嵌入 query |
| admin API key | 后台接管 | `settingService.GetAdminAPIKey()` |
| 用户 API Key | 额度盗用 | 用户台与网关鉴权 |
| 上游 access/refresh token | 上游账户接管、成本损失 | 账号凭据存储与刷新逻辑 |
| S3 凭据 | 备份数据泄露、SSRF 扩权 | `BackupS3Config` |
| 备份文件 | 全量敏感数据泄露 | 管理端备份/恢复链路 |
| 日志与错误样本 | 凭据、请求内容、用户数据泄露 | `ops`、失败请求日志 |

## 7. 信任边界

- 浏览器前端 → 后端公开 API
- JWT 用户 → 用户资源
- 管理员 → `/api/v1/admin/**`
- API Key → 网关转发层与计费
- 网关 → 上游 AI 服务 / OAuth 提供商 / S3
- 后端 → PostgreSQL / Redis
- 后端 → datamanagementd / 备份存储
- Docker 容器 → 宿主机 / 反向代理
- 边缘代理头部 → 后端客户端 IP 判断

## 8. 高风险优先审查区域

- P0：
  - 公共首页 `home_content` HTML 渲染
  - iframe 外链 token 传播
  - 管理员初始化与默认部署暴露面
  - 安全敏感链路的客户端 IP 解析
  - 备份 S3 测试与出站访问
- P1：
  - OAuth 回调令牌交付链
  - 管理后台升级、恢复、data-management
  - 失败请求日志与凭据脱敏
  - API Key 生命周期、模型权限、计费与并发
- P2：
  - 远程基线头部、TLS、CORS、source map、文档与静态反向

## 9. 当前已确认的高风险入口

- `GET /api/v1/settings/public` → `home_content` 公开下发
- `frontend/src/views/HomeView.vue` → `v-html="homeContent"`
- `frontend/src/utils/embedded-url.ts` → `token`、`src_host`、`src_url`
- `backend/internal/pkg/ip/ip.go` → `GetClientIP`
- `backend/internal/service/backup_service.go` → `TestS3Connection`
- `deploy/docker-compose*.yml` → `AUTO_SETUP=true`
- `deploy/docker-compose.yml` → `0.0.0.0:8080:8080`

## 10. 后续深测输入

- WSL staging-like 环境
- `.env.audit`
- mock upstream
- `security/reports/*`
- 生产低风险检查目标：
  - `https://api.zyxai.de`
  - `https://demo.sub2api.org/`
