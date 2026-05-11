# Sub2API 步骤 3：WSL 深度逆向 / 渗透 / 攻击验证问题单

审计日期：2026-05-11  
审计范围：本地 WSL 隔离运行时、WSL 主机服务回退部署、网关/API Key/Document AI 运行态、嵌入式前端静态资源、历史步骤 3 问题回归  
执行边界：仅在本地 WSL/本机隔离环境执行；未对 production/staging 或未授权目标执行主动攻击；未输出完整密钥、Token、Cookie、密码  
当前状态：步骤 3 已完成，等待用户修复后进入步骤 4

## 结论摘要

- 本文件是 2026-05-11 新一轮步骤 3 的正式运行态问题单，覆盖当前工作树在步骤 2 通过后的运行态表现。
- 历史 `2026-05-09` 步骤 3 问题作为基线复查；本轮未将已关闭且未复现的问题重复登记，除非出现回归。
- Windows 侧未发现可用 `docker` 命令，WSL/Docker 标准容器链路不可作为稳定基线；本轮采用 WSL 主机服务回退路径完成运行态验证。
- 当前 Linux 后端二进制使用非 embed 构建完成，审计运行时监听 `127.0.0.1:18081`，mock upstream 监听 `127.0.0.1:19090`。
- 本轮发现 1 条新的步骤 3 运行态问题：
  - `STEP3-20260511-001`：Document AI 非 allowlist URL 保存/批量更新被校验拒绝后，对外返回 `500 internal error`，未返回受控 `4xx`。

## 正向控制与回归观察

- `/health` 返回 `200`，运行时可用。
- 匿名访问管理端与网关均被拒绝。
- 管理员登录可用，普通用户访问管理端被拒绝。
- API Key 生命周期通过官方 API 复测：
  - inactive key 返回 `401 API_KEY_DISABLED`。
  - expired key 返回 `403 API_KEY_EXPIRED`。
  - deleted key 返回 `401 INVALID_API_KEY`。
  - disabled user 的 API Key 返回 `401 USER_INACTIVE`。
- 上游 redirect 隔离场景返回受控 `502`，未透传 `Location` / `Refresh`，mock 未观察到第二跳 `/final/`。
- 上游 `500` 隔离场景返回受控 `502`，响应体未出现 panic，服务日志未出现 `panic recovered`，未观察到测试哨兵敏感值泄露。
- Document AI base64 超限场景返回 `400 document_ai_invalid_request`。
- Document AI direct provider 异常场景返回受控 `502 document_ai_provider_error`。
- 嵌入式前端静态资源 `backend/internal/web/dist` 未发现 `.map` 文件；针对 `token=`、`src_url`、`src_host`、默认弱口令、私钥标记和常见云密钥前缀的定向扫描未发现可用明文秘密。
- 本轮 `pnpm --dir frontend build` 未在限定时间内完成，未以新的 `frontend/dist` 作为静态基线；采用仓库当前嵌入式 dist 做替代静态反向核对。
- 本地扫描工具链核对中，nuclei、ZAP、trivy、semgrep、gitleaks、nmap、sqlmap 等工具不可用或缺失；本轮以手工运行态验证、定向脚本和源码证据归并替代，不作为业务阻塞。

## Findings

### STEP3-20260511-001

- 风险等级：Medium
- 问题标题：Document AI 非 allowlist URL 保存/批量更新被拒绝后返回 `500 internal error`，未返回受控客户端错误
- 测试场景：
  - 管理员在本地 WSL 审计运行时创建或批量更新 `baidu_document_ai` 账号。
  - `credentials.direct_api_urls` 指向不在 `security.url_allowlist.document_ai_hosts` 中的 URL。
- 触发步骤：
  - 登录管理员账号。
  - 调用 `POST /api/v1/admin/accounts` 创建 `baidu_document_ai` 账号，提交非 allowlist 的 `credentials.direct_api_urls`。
  - 调用 `POST /api/v1/admin/accounts/bulk-update` 批量更新同类非 allowlist URL。
  - 观察接口状态码和服务日志。
- 真实可利用性：
  - 可稳定触发。URL 安全校验本身已经生效，未观察到非 allowlist 目标被放行或产生 SSRF 出站。
  - 但校验失败最终映射为 `500 internal error`，导致客户端收到服务端内部错误，而不是受控 `400`/业务错误码。
- 影响范围：
  - 影响 Document AI 管理端账号创建和批量更新体验。
  - 影响错误语义、审计归因与自动化客户端处理；安全策略拒绝会被误判为服务器异常。
  - 可能导致监控误报 5xx，掩盖真实的客户端配置错误。
- 证据：
  - `tmp/step3_runtime_audit_20260511_summary.json`：
    - `document_ai_disallowed_url_save.status = 500`，响应前缀为 `{"code":500,"message":"internal error"}`。
    - `document_ai_disallowed_url_bulk_update.status = 500`，响应前缀为 `{"code":500,"message":"internal error"}`。
  - `security/reports/manual/server-audit.log`：
    - 管理端创建和批量更新链路均记录 `baidu document ai account validation failed`。
    - `reason` 为 `baidu_document_ai credentials.direct_api_urls contains a disallowed API URL`。
  - 源码证据：
    - `backend/internal/service/admin_service_accounts.go:622-653` 已在保存/批量更新校验 `async_base_url` 与 `direct_api_urls`。
    - `backend/internal/service/admin_service_accounts.go:664-681` 的 `newBaiduDocumentAIValidationError` 记录校验失败后返回普通 `errors.New(reason)`。
    - `backend/internal/pkg/errors/errors.go:146-158` 将普通 error 兜底转换为 generic internal error。
    - `backend/internal/pkg/response/response.go:86-99` 再将该错误输出为 `500 internal error`。
- 可能根因：
  - Document AI 账号校验失败使用普通 error 表达业务输入错误。
  - 管理端响应层只认识统一应用错误类型；普通 error 被兜底当作内部错误处理。
- 修复建议：
  - 将 `newBaiduDocumentAIValidationError` 改为统一应用错误类型，映射到 `400` 和稳定业务 reason，例如 `ACCOUNT_INVALID_CREDENTIALS` 或专门的 Document AI 配置错误 reason。
  - 创建、更新、批量更新三条链路保持一致的错误结构。
  - 日志继续保留脱敏后的结构化 reason，不向响应输出内部堆栈或完整敏感配置。
  - 补充单测/运行态复测：非 allowlist `async_base_url` / `direct_api_urls` 保存与批量更新应返回受控 `4xx`，官方默认 endpoint 仍应通过。
- 复审方法：
  - 在步骤 4 中重新启动本地 WSL 审计运行时。
  - 重放 `POST /api/v1/admin/accounts` 与 `POST /api/v1/admin/accounts/bulk-update` 的非 allowlist URL 用例。
  - 确认接口返回受控 `400` 或等价客户端错误，响应体包含稳定业务 reason，且服务日志无敏感值泄露。
  - 同时复测默认百度 Document AI endpoint 创建/更新仍可通过。

## 历史步骤 3 问题回归状态

- `STEP3-001`：保存异常 `base_url` 并触发 probe 出站的历史问题，本轮运行态中 `file://`、`gopher://`、非法端口、空 host 均返回受控 `400 ACCOUNT_INVALID_BASE_URL`，未重报。
- `STEP3-002`：`real_forward` 跟随 `302` 的历史问题，本轮隔离 redirect 复测未观察到第二跳，未重报。
- `STEP3-003`：上游 `500` failover 触发 panic 的历史问题，本轮隔离上游 `500` 复测未出现 `panic recovered`，未重报。
- `STEP3-004`：panic recovery 日志泄露敏感请求头的历史问题，本轮未复现 panic 日志泄露；上游 `500` 失败路径未观察到测试哨兵值进入服务日志，未重报。

## 审查测试覆盖状态

| 文档 | 状态 | 说明 |
|---|---|---|
| 00_README_EXECUTION_ORDER.md | done | 已按 README 固定四步闭环进入并完成步骤 3。 |
| 01_SCOPE_AND_RULES.md | done | 已限定在本地 WSL/本机隔离环境，不做未授权主动攻击。 |
| 02_ENV_AND_WSL_ISOLATED_LAB.md | done_with_note | WSL 可用；标准 Docker 链路不可作为稳定基线，采用 WSL 主机服务回退完成验证。 |
| 03_CODEX_CLI_MASTER_PROMPTS.md | done | 已按防御性安全代理要求执行，工具命中未直接当作漏洞。 |
| 04_ATTACK_SURFACE_MAP.md | done | 已覆盖认证、API Key、网关、出站、Document AI、静态资源等运行态攻击面。 |
| 05_LOCAL_SOURCE_REVIEW_BACKEND_AUTH_APIKEY.md | done | 已完成本地认证态与 API Key 生命周期运行态复测。 |
| 06_LOCAL_SOURCE_REVIEW_AI_GATEWAY_BILLING_UPSTREAM.md | done | 已复测网关 redirect、上游 500、Document AI provider 错误路径。 |
| 07_LOCAL_SOURCE_REVIEW_FRONTEND_DEPLOY_DATAMANAGEMENT.md | done_with_note | 前端新 build 超时；已对当前嵌入式 dist 做静态反向核对。 |
| 08_LOCAL_SCANNERS_AND_REPORT_IMPORT.md | done_with_note | 扫描器缺失；已导入手工运行态、脚本摘要和日志证据。 |
| 09_REMOTE_TEST_LEVELS_AND_PRODUCTION_BASELINE.md | done_with_note | 本轮不进入远程/生产；以本地授权运行态基线替代。 |
| 10_REMOTE_ASSET_ENUM_AND_STATIC_REVERSE.md | done_with_note | 未做远程资产枚举；已完成本地嵌入式静态资源反向核对。 |
| 11_REMOTE_AUTH_MATRIX_AND_APIKEY_LIFECYCLE.md | done | 已在本地授权环境复测认证态和 API Key 生命周期。 |
| 12_REMOTE_AI_GATEWAY_BUSINESS_LOGIC.md | done | 已在本地授权环境复测网关业务失败与成功路径。 |
| 13_REMOTE_UPSTREAM_SSRF_AND_EGRESS.md | done | 已复测 base_url 与 Document AI URL allowlist 运行态表现。 |
| 14_REMOTE_NUCLEI_ZAP_API_SCANNING.md | done_with_note | 本地缺少 Nuclei/ZAP 等工具；未做远程主动扫描。 |
| 15_DEPLOYMENT_HARDENING_NGINX_DOCKER_VPS.md | done_with_note | 未启动 VPS/Docker 标准链路；以 WSL 主机服务回退和配置核对替代。 |
| 16_LOGGING_AUDIT_FORENSICS.md | done | 已核对失败路径日志与敏感测试值，未观察到本轮哨兵泄露。 |
| 17_BACKUP_UPGRADE_DATAMANAGEMENTD.md | done_with_note | 本轮未做完整 data-management agent 联调；沿用历史基线并确认未触及相关代码。 |
| 18_REVERSE_MAPPING_SOURCE_CONFIG.md | done | 新 finding 已映射到运行脚本、日志、源码函数和错误映射链路。 |
| 19_FINAL_REPORT_REMEDIATION_RETEST.md | done | 已输出本文件作为步骤 3 正式问题单。 |
| 20_STANDARDS_MAPPING_CHECKLIST.md | done | 已按 API 错误处理、出站治理、日志脱敏和本地授权边界做标准化核对。 |

## 当前暂停点

- 步骤 3 已完成并输出正式问题单。
- 按 `docs/安全实施/README.md` 暂停等待用户修复 `STEP3-20260511-001`。
- 修复完成后进入步骤 4，输出 `security/findings/wsl-deep-test-recheck.md`。
