# Sub2API 步骤 1：本地源码深度审查问题单

审查日期：2026-05-09  
审查范围：本地源码、部署配置、历史审计报告对照  
执行边界：仅源码与配置深审，不直接修改业务代码，不执行生产主动攻击  

## 结论摘要

- 本轮已完成步骤 1 的本地源码深度审查，并与 `docs/security_audit_20260324.md`、`docs/security_audit_20260508.md` 进行了现状对照。
- 目前确认 6 条需要优先处理的问题，其中 4 条为高风险，2 条为中风险偏高。
- 按总控要求，本步骤结束后暂停，等待用户修复后进入步骤 2。

## Findings

### SR-001

- 风险等级：High
- 问题标题：公共首页 `home_content` 仍以未净化 HTML 直接渲染，形成真实存储型 XSS 落点
- 影响模块：公共设置、首页前端、浏览器会话
- 具体文件/函数/路由/配置：
  - `backend/internal/service/setting_service_public.go`
  - `frontend/src/views/HomeView.vue`
  - `frontend/src/stores/auth.ts`
  - `frontend/src/api/client.ts`
- 真实可达性判断：可达。`GET /api/v1/settings/public` 会下发 `home_content`，首页未登录即可访问，前端在 HTML 模式下直接 `v-html` 渲染。
- 利用条件：
  - 攻击者可控制或影响管理员填写的 `home_content`
  - 受害者访问公开首页
  - 浏览器中存在可窃取会话或敏感上下文
- 影响范围：
  - 访客与管理员浏览器均受影响
  - 可联动窃取 `auth_token`、`refresh_token`、页面上下文和管理操作能力
- 证据：
  - `backend/internal/service/setting_service_public.go` 将 `HomeContent` 暴露到公开设置结构体
  - `frontend/src/views/HomeView.vue` 在 `<div v-else v-html="homeContent"></div>` 直接渲染 HTML
  - `frontend/src/stores/auth.ts` 与 `frontend/src/api/client.ts` 长期把 `auth_token` / `refresh_token` 持久化到 `localStorage`
- 修复建议：
  - 取消原始 HTML 直渲染，改为白名单净化或 Markdown/受限 DSL
  - 对 `home_content` 引入发布前净化与审计
  - 降低浏览器长寿命令牌暴露面，优先迁移到 HttpOnly 会话方案
- 复审方法：
  - 检查 `home_content` 是否不再直接进入 `v-html`
  - 用受控 HTML 样本确认脚本/事件属性被剥离
  - 验证首页渲染后无法读取浏览器认证令牌
- 标准映射：
  - OWASP API Security Top 10 2023：API8 Security Misconfiguration、API3 Broken Object Property Level Authorization（配置内容暴露链路）
  - OWASP WSTG：Client-side Testing / XSS Testing
  - OWASP ASVS：V5 Validation, Sanitization and Encoding

### SR-002

- 风险等级：High
- 问题标题：iframe 嵌入 URL 仍通过查询参数传播实时 Bearer Token 与来源上下文
- 影响模块：购买订阅页、自定义页面、用户与管理员浏览器会话
- 具体文件/函数/路由/配置：
  - `frontend/src/utils/embedded-url.ts`
  - `frontend/src/views/user/PurchaseSubscriptionView.vue`
  - `frontend/src/views/user/CustomPageView.vue`
- 真实可达性判断：可达。前端会在构造外链 URL 时将 `authStore.token` 直接写入 `token=` 查询参数，并附带 `src_host`、`src_url`。
- 利用条件：
  - 管理员配置的 iframe 目标指向外部站点
  - 用户或管理员访问购买页/自定义页面
- 影响范围：
  - 外部站点、浏览器历史、代理日志、分析系统都可能拿到可复用 JWT
  - 同时泄露来源页面与站点上下文
- 证据：
  - `frontend/src/utils/embedded-url.ts` 调用 `url.searchParams.set("token", authToken)`
  - 同文件追加 `src_host` 和 `src_url`
  - `frontend/src/views/user/PurchaseSubscriptionView.vue` 与 `CustomPageView.vue` 都向该函数传入 `authStore.token`
- 修复建议：
  - 禁止通过 URL 查询参数传递 Bearer Token
  - 若必须传达登录态，改为短时效、单用途、受众受限的后端签发嵌入令牌
  - 为可配置 iframe 目标增加严格 allowlist
  - 评估已泄露 token 的失效与轮换
- 复审方法：
  - 检查生成后的 iframe URL 不再包含 `token=`、`src_url`
  - 若使用新嵌入票据，确认过期时间、受众和一次性语义
  - 验证外部站点无法直接复用主 JWT 调用 `/api/v1/*`
- 标准映射：
  - OWASP API Security Top 10 2023：API3 Broken Object Property Level Authorization、API8 Security Misconfiguration
  - OWASP WSTG：Sensitive Data Exposure / Client-side Testing
  - OWASP ASVS：V3 Session Management、V8 Data Protection

### SR-003

- 风险等级：High
- 问题标题：大量安全敏感链路仍直接信任原始转发头解析客户端 IP，绕过可信代理链
- 影响模块：认证、Turnstile、网关请求上下文、失败日志、审计与风控
- 具体文件/函数/路由/配置：
  - `backend/internal/server/http.go`
  - `backend/internal/pkg/ip/ip.go`
  - `backend/internal/handler/auth_handler.go`
  - `backend/internal/handler/gateway_handler_messages.go`
  - `backend/internal/handler/ops_error_logger.go`
  - 以及多个 `gateway` / `gemini` / `openai` / `usage` handler
- 真实可达性判断：可达。虽然 Gin 已支持 `SetTrustedProxies`，但大量业务代码仍调用 `GetClientIP()`，该函数优先直接信任 `CF-Connecting-IP`、`X-Real-IP`、`X-Forwarded-For`。
- 利用条件：
  - 反向代理未正确清洗/覆盖相关头
  - 存在源站直连、错误代理链或可控请求头
- 影响范围：
  - 验证码校验、请求轨迹、错误日志、网关上下文、审计与后续按 IP 风控结果都可能被伪造
- 证据：
  - `backend/internal/server/http.go` 在 trusted proxies 为空时会禁用 Gin 可信代理解析
  - `backend/internal/pkg/ip/ip.go` 的 `GetClientIP()` 直接优先读取原始转发头
  - `backend/internal/handler/auth_handler.go` 使用 `ip.GetClientIP(c)` 做 Turnstile 校验
  - 多个网关 handler 与 `ops_error_logger.go` 继续使用 `ip.GetClientIP(c)`
  - 仅 `api_key_auth.go` 的 IP 白名单链路使用了 `GetTrustedClientIP`
- 修复建议：
  - 安全敏感路径统一改用 `GetTrustedClientIP()`
  - 清理业务层对原始头优先信任的辅助函数调用
  - 明确生产反代只在可信链上设置转发头并阻断源站直连
- 复审方法：
  - 全局检索安全敏感路径不再调用 `GetClientIP()`
  - 验证 trusted proxies 为空或源站直连场景下不会被任意头伪造客户端 IP
  - 对 Turnstile、失败日志、请求追踪做针对性回归
- 标准映射：
  - OWASP API Security Top 10 2023：API8 Security Misconfiguration
  - OWASP WSTG：Configuration Testing / Access Control Testing
  - OWASP ASVS：V14 Configuration、V7 Error Handling and Logging

### SR-004

- 风险等级：High
- 问题标题：S3 连通性测试仍可复用已保存 `SecretAccessKey`，并作为管理员可控 SSRF 出网原语
- 影响模块：备份管理、data management、出站访问治理、凭据保护
- 具体文件/函数/路由/配置：
  - `backend/internal/service/backup_service.go`
  - `backend/internal/handler/admin/backup_handler.go`
  - `backend/internal/server/routes/admin.go`
- 真实可达性判断：可达。管理员请求 `POST /api/v1/admin/backups/s3-config/test` 或数据管理侧 S3 测试入口时，若请求体未提供新密钥，服务会复用已保存的 `SecretAccessKey`，然后基于可控 endpoint 执行 `HeadBucket`。
- 利用条件：
  - 拥有管理员能力或管理员账号被接管
  - 可提交自定义 S3 endpoint / bucket / access key 组合
- 影响范围：
  - 服务端可被驱动访问任意 HTTP(S) 目标
  - 已保存备份凭据可能被用于签名出站请求
  - 错误全文会回显给调用方，辅助探测网络行为
- 证据：
  - `backend/internal/service/backup_service.go` 在 `cfg.SecretAccessKey == ""` 时回退到已保存配置
  - 同文件随后构造存储客户端并执行 `store.HeadBucket(ctx)`
  - `backend/internal/handler/admin/backup_handler.go` 将 `err.Error()` 原样返回给调用方
  - `backend/internal/server/routes/admin.go` 挂载备份与 data-management 高权限入口
- 修复建议：
  - `test` 接口强制显式提交完整凭据，禁止自动复用已保存 secret
  - 对 endpoint 做 allowlist、scheme、端口和私网/环回拦截
  - 关闭或重新校验重定向
  - 对外错误统一脱敏
- 复审方法：
  - 验证未显式提供 secret 时测试接口直接拒绝
  - 验证本机、私网、保留地址和非 allowlist 域名会被拦截
  - 验证返回错误不再暴露底层网络细节全文
- 标准映射：
  - OWASP API Security Top 10 2023：API7 Server Side Request Forgery
  - OWASP WSTG：SSRF Testing / Authorization Testing
  - OWASP ASVS：V5 Input Validation、V8 Data Protection、V14 Configuration

### SR-005

- 风险等级：Medium
- 问题标题：默认 Docker Compose 仍将后端绑定到 `0.0.0.0:8080`，并默认启用 `AUTO_SETUP`
- 影响模块：默认部署面、安装向导、边缘代理边界
- 具体文件/函数/路由/配置：
  - `deploy/docker-compose.yml`
  - `deploy/docker-compose.local.yml`
  - `deploy/docker-compose.dev.yml`
  - `deploy/docker-compose.standalone.yml`
  - `backend/internal/setup/setup.go`
  - `backend/internal/server/routes/common.go`
- 真实可达性判断：可达。默认 Compose 会将服务暴露到公网或外部可达网卡；同时自动初始化机制扩大了首次部署窗口的敏感面。
- 利用条件：
  - 运维人员按默认 Compose 暴露部署
  - 反向代理、WAF、安全组未额外收敛
- 影响范围：
  - 明文 HTTP 直连入口可能绕过边缘 TLS、头部清洗、审计和访问控制
  - 初装阶段增加被探测与撞向导的风险
- 证据：
  - `deploy/docker-compose.yml` 配置 `"${BIND_HOST:-0.0.0.0}:${SERVER_PORT:-8080}:8080"`
  - 多个 Compose 变体默认 `AUTO_SETUP=true`
  - `backend/internal/server/routes/common.go` 暴露 `/setup/status`
- 修复建议：
  - 默认 Compose 仅绑定 `127.0.0.1` 或内网地址
  - 将对外发布交给反向代理
  - 限制生产默认 `AUTO_SETUP` 行为，要求显式初始化条件
- 复审方法：
  - 检查默认 Compose 是否不再映射公网 `8080`
  - 验证初装流程只能在预期环境和窗口内访问
  - 对生产部署说明与模板同步更新
- 标准映射：
  - OWASP API Security Top 10 2023：API8 Security Misconfiguration
  - OWASP WSTG：Infrastructure Configuration Testing
  - OWASP ASVS：V14 Configuration

### SR-006

- 风险等级：Medium
- 问题标题：自动初始化仍会输出一次性管理员密码，仓库文档与示例配置仍保留 `admin123`
- 影响模块：初始化流程、运维文档、凭据卫生
- 具体文件/函数/路由/配置：
  - `backend/internal/setup/setup.go`
  - `README.md`
  - `README_CN.md`
  - `README_EN.md`
  - `deploy/config.example.yaml`
- 真实可达性判断：可达。自动初始化在未提供管理员密码时会直接输出生成密码；README 和示例配置保留弱口令示例，会降低运维警惕。
- 利用条件：
  - 使用自动初始化且日志可被他人访问
  - 运维直接参考示例密码或忘记立即更换
- 影响范围：
  - 首次部署阶段管理员账号可能被日志观察者接管
  - 文档示例会放大弱口令使用倾向
- 证据：
  - `backend/internal/setup/setup.go` `fmt.Printf("Generated admin password (one-time): %s\n", cfg.Admin.Password)`
  - `README.md`、`README_CN.md`、`README_EN.md` 仍展示 `admin123`
  - `deploy/config.example.yaml` 默认 `admin_password: "admin123"`
- 修复建议：
  - 生产环境要求显式管理员初始口令或一次性安全引导，不输出到常规日志
  - 清理所有 `admin123` 示例，改为随机占位符和强制改密提示
- 复审方法：
  - 检查初始化日志不再打印完整管理员密码
  - 文档与示例配置中不再出现弱默认口令
  - 验证首次登录流程要求立即修改临时口令
- 标准映射：
  - OWASP API Security Top 10 2023：API2 Broken Authentication、API8 Security Misconfiguration
  - OWASP WSTG：Authentication Testing / Information Leakage
  - OWASP ASVS：V2 Authentication、V8 Data Protection

## 本轮未单独起编号但已覆盖的正向控制

- `/api/v1/admin/**` 路由组统一挂载了管理员鉴权中间件
- JWT 用户鉴权校验 `TokenVersion`，能在密码变更后使旧 token 失效
- API Key 白名单使用 `GetTrustedClientIP()`，说明可信代理链能力已存在
- 网关与用户态大多数关键入口已有服务端限流或错误归一化

## 下一步

- 按流程暂停，等待用户修复上述问题后进入步骤 2。
- 步骤 2 将以本文件为唯一对照清单，逐项判断 `fixed / partially_fixed / not_fixed / new_issue`。
