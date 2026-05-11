# sub2api 全链路安全审计报告

## 1. 审计范围

- 本地源码：
  - 当前本地仓库 `sub2api`
  - `README.md`、`README_CN.md`、`README_EN.md`
  - `backend/`、`frontend/`、`deploy/`、`tools/`
- WSL 环境：
  - Windows 主机下的 WSL `Ubuntu`
  - WSL 主机服务回退运行态：`server-test-linux` + `mock_upstream.py`
  - 本地 PostgreSQL / Redis / mock upstream
- Staging：
  - 仅限 `security/SECURITY_SCOPE.md` 中定义的 WSL 内 staging-like 隔离环境思路
  - 本轮未实际进入独立 staging 域名环境
- Production：
  - 仅保留在授权边界内的低风险基线思路
  - 本轮最终闭环未对 production 执行主动扫描或深测
- 未包含范围：
  - 未授权第三方资产
  - 未经明确授权的 production 主动 DAST、压测、破坏性 payload
- 禁止行为：
  - 不暴力破解
  - 不撞库
  - 不高并发压测
  - 不破坏生产数据
  - 不泄露完整密钥、Token、Cookie、密码

## 2. 执行摘要

- Critical：0
- High：1（源码阶段发现，后续已修复并复审关闭）
- Medium：3（2 条源码阶段、1 条运行态阶段发现；均已修复并复审关闭）
- Low：0
- 最高优先级风险：
  - `SR-20260511-003`：Document AI 文件 / 上游响应无显式上限，存在内存型 DoS 风险
  - `STEP3-20260511-001`：Document AI 非 allowlist URL 被拒绝后错误映射为 `500 internal error`

当前最终结论：

- 本轮四步闭环已全部完成。
- 步骤 1 发现的 `SR-20260511-001/002/003` 在步骤 2 全部复审为 `fixed`。
- 步骤 3 发现的 `STEP3-20260511-001` 在步骤 4 复审为 `fixed`。
- 本轮闭环结束时，无高优先级未关闭问题。

## 3. 系统架构理解

- 后端：
  - `backend/internal/server/` 提供 HTTP 路由与中间件
  - `backend/internal/service/` 承担鉴权、计费、网关调度、Document AI、备份等核心业务
  - `backend/internal/repository/` 负责数据库与上游 HTTP 抽象
- 前端：
  - Vue 3 + Vite，管理台与用户台共用同一前端工程
  - 静态产物嵌入 `backend/internal/web/dist`
- 核心数据流：
  - 浏览器 / API Key 调用方 → 后端路由 → 鉴权/授权 → 配额 / 调度 / 出站 → PostgreSQL / Redis / 上游
- 信任边界：
  - 浏览器前端 → 后端 API
  - JWT 用户 → 用户资源
  - 管理员 → `/api/v1/admin/**`
  - API Key → 网关 `/v1/**`、`/document-ai/v1/**`
  - 后端 → 上游模型服务 / 百度 Document AI / S3 / OAuth 提供商
  - 后端 → PostgreSQL / Redis / datamanagementd
- 敏感数据：
  - JWT / refresh token
  - 用户 API Key / admin API key
  - 上游 access token / refresh token / API key
  - 备份凭据与备份文件
  - 日志中的请求上下文与失败样本

## 4. 攻击面地图

- 公开入口：
  - `/health`
  - `/api/v1/auth/**`
  - `/api/v1/settings/public`
  - `/api/v1/pages/:slug`
- 高权限入口：
  - `/api/v1/admin/**`
  - `/api/v1/admin/backups/**`
  - `/api/v1/admin/data-management/**`
- API Key 网关入口：
  - `/v1/**`
  - `/v1beta/**`
  - `/document-ai/v1/**`
- 前端敏感集成点：
  - 首页公共设置渲染
  - iframe 外链参数
  - OAuth 回调页
- 部署面：
  - `deploy/docker-compose*.yml`
  - `deploy/config.example.yaml`
  - `deploy/install.sh` / `docker-upgrade.sh`

详细映射已记录在：

- `security/attack-surface-map.md`
- `security/DEPLOYMENT_MAP.md`

## 5. 远程暴露面

| 资产 | 端口/URL | 服务 | 是否预期 | 风险 | 对应源码/配置 |
|---|---|---|---|---|---|
| 本地 WSL 审计服务 | `127.0.0.1:18081` | `server-test-linux` | 是 | 仅本地隔离验证 | `backend/bin/server-test-linux` |
| 本地 mock upstream | `127.0.0.1:19090` | `mock_upstream.py` | 是 | 仅本地隔离验证 | `security/mock/mock_upstream.py` |
| 生产目标（授权边界） | `https://api.zyxai.de` | Web/API | 预期 | 本轮未主动深测 | `security/SECURITY_SCOPE.md` |
| 生产目标（授权边界） | `https://demo.sub2api.org/` | Demo Web/API | 预期 | 本轮未主动深测 | `security/SECURITY_SCOPE.md` |

说明：

- 本轮最终闭环主要消化本地源码与本地 WSL / 主机隔离运行态。
- 远程 / production 相关内容仅做范围确认与低风险策略约束，不作为本轮正式 finding 主要来源。

## 6. 漏洞清单

| ID | 等级 | 风险 | 来源 | 模块 | 源码/配置 | 是否可达 | 当前状态 | 修复优先级 |
|---|---|---|---|---|---|---|---|---|
| SR-20260511-001 | Medium | redirect blocked 后可能透传 `Location` / `Refresh` | 步骤 1 源码深审 | 共享 HTTP upstream / 网关响应头 | `backend/internal/service/upstream_redirect_blocked.go` 等 | 是 | fixed | P1 |
| SR-20260511-002 | Medium | Document AI `async_base_url` / `direct_api_urls` 未纳入统一 allowlist | 步骤 1 源码深审 | Document AI 出站治理 | `backend/internal/service/admin_service_accounts.go` 等 | 是 | fixed | P1 |
| SR-20260511-003 | High | Document AI 文件 / 上游响应无显式上限，可导致内存型 DoS | 步骤 1 源码深审 | Document AI 上传、响应读取、结果下载 | `document_ai_handler.go`、`document_ai_service.go`、`document_ai_baidu_client.go` | 是 | fixed | P0 |
| STEP3-20260511-001 | Medium | Document AI 非 allowlist URL 被拒绝后错误映射为 `500 internal error` | 步骤 3 运行态深测 | 管理端账号创建/批量更新错误语义 | `admin_service_accounts.go`、`pkg/errors`、`pkg/response` | 是 | fixed | P1 |

## 7. High/Critical 详情

本轮无未关闭的 High/Critical。为便于归档，保留已关闭的高优先级问题摘要。

### SR-20260511-003

- 等级：High
- 标题：Document AI 文件与上游响应存在大体积 `io.ReadAll` 路径，可被放大为内存型 DoS
- 发现来源：步骤 1 本地源码深度审查
- 影响资产：
  - `/document-ai/v1/jobs`
  - `/document-ai/v1/models/{model}:parse`
  - Document AI 上游 JSON 响应读取
  - 异步结果下载
- 对应源码/配置：
  - `backend/internal/handler/document_ai_handler.go`
  - `backend/internal/service/document_ai_service.go`
  - `backend/internal/service/document_ai_baidu_client.go`
  - `backend/internal/config/config_types_gateway.go`
- 触发入口：
  - multipart 文件上传
  - JSON `file_base64`
  - Provider 超大 JSON
  - 异步结果超大 Markdown / JSON
- 数据流：
  - API Key 调用方 → Document AI handler → service normalize → provider client / result download
- 可利用条件：
  - 有效站内 API Key 且绑定到 `baidu_document_ai` 分组
  - 或异常上游返回超大响应
- 影响范围：
  - 可造成显著内存占用、延迟抖动或进程 OOM
- 证据：
  - 步骤 1 问题单 `security/findings/source-review-findings.md`
- 修复建议与落地结果：
  - 已新增：
    - `gateway.document_ai_upload_max_bytes`
    - `gateway.document_ai_upstream_json_read_max_bytes`
    - `gateway.document_ai_result_read_max_bytes`
  - multipart / base64 / provider JSON / result download 均已接入显式大小限制
- 复测方式：
  - 步骤 2 定向 Go 回归已通过
- 当前状态：
  - fixed

## 8. 误报与待确认项

- 扫描器缺失项：
  - `nuclei`、`ZAP`、`trivy`、`semgrep`、`gitleaks`、`nmap`、`sqlmap` 在本地不可用或未纳入本轮稳定链路
  - 已按 `done_with_note` 记录，不直接视为漏洞遗漏
- 远程静态抓取与远程认证矩阵：
  - 因授权窗口 / 目标补充信息不足，本轮未把远程结果作为正式 finding 来源
- 2026-05-10 的旧步骤 4 复审单：
  - 已归档为历史基线，不再代表本轮最终状态

## 9. 修复路线图

### P0 已完成

- `SR-20260511-003`
  - Document AI 上传、base64、上游 JSON、结果下载显式大小限制

### P1 已完成

- `SR-20260511-001`
  - redirect blocked 响应剥离 `Location` / `Refresh`
- `SR-20260511-002`
  - Document AI 专属 `document_ai_hosts` allowlist
- `STEP3-20260511-001`
  - Document AI 账号 URL 校验失败统一映射为 `400 ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS`

### 后续建议

- 继续把远程低风险基线、授权远程静态抓取、远程认证矩阵补齐到独立材料
- 若需要对外沟通，可再生成独立的 `remediation-plan.md` 与 `retest-checklist.md`

## 10. 复测清单

| ID | 修复项 | 复测环境 | 复测命令/步骤 | 预期 | 实际 | 通过 |
|---|---|---|---|---|---|---|
| SR-20260511-001 | redirect blocked 剥离 `Location` / `Refresh` | 本地源码 | `go test -tags unit ./internal/repository -run "RedirectBlocked|HTTPUpstream" -count=1 -v` | 502 且无 `Location`/`Refresh` | 通过 | ✅ |
| SR-20260511-002 | Document AI URL allowlist | 本地源码 | `go test -tags unit ./internal/service -run "BaiduDocumentAI|DocumentAI|BaseURL|Account.*BaseURL|BulkUpdateAccounts_.*BaiduDocumentAI" -count=1 -v` | 非 allowlist 拒绝、allowlist 通过 | 通过 | ✅ |
| SR-20260511-003 | Document AI 大小限制 | 本地源码 | 同上 + handler/config 定向测试 | 超限受控失败 | 通过 | ✅ |
| STEP3-20260511-001 | Document AI 错误语义映射 | 本地 WSL / 主机运行态 | `python tmp/step4_document_ai_recheck_20260511.py` | 非 allowlist `400 ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS`；正向 `200` | 通过 | ✅ |

## 11. 标准映射

- OWASP API Security Top 10 2023：
  - API4 Unrestricted Resource Consumption
  - API7 Server Side Request Forgery
  - API8 Security Misconfiguration
- OWASP WSTG：
  - SSRF Testing
  - Input Validation Testing
  - Information Leakage / Configuration Testing
- OWASP ASVS：
  - V5 Validation, Sanitization and Encoding
  - V8 Data Protection
  - V14 Configuration
- OWASP Top 10：
  - A05 Security Misconfiguration
  - A06 Vulnerable and Outdated Components（本轮仅方法论引用，无正式依赖漏洞落单）
  - A10 Server-Side Request Forgery

## 12. 结论

- 本轮四步闭环已完整执行并收口。
- 正式问题单与复审单链路如下：
  - `security/findings/source-review-findings.md`
  - `security/findings/source-review-recheck.md`
  - `security/findings/wsl-deep-test-findings.md`
  - `security/findings/wsl-deep-test-recheck.md`
- 当前最终状态：
  - 无 High / Critical 未关闭问题
  - 进度文件已收口为 `completed`
  - 若后续需要对外汇报或走发布前 gate，可直接以本报告作为当前审计基线
