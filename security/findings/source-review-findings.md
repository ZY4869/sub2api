# Sub2API 新一轮步骤 1：本地源码深度审查问题单

审查日期：2026-05-11  
审查范围：当前未提交工作树、本地源码与配置、历史审计报告、`docs/审查测试/00-20`、`docs/审查日记/2026-05-09/问题跟踪.md`  
执行边界：仅源码与配置深审；未修业务代码；未对生产或未授权目标执行主动攻击；未输出完整密钥、Token、Cookie、密码。  
历史基线：2026-05-09 步骤 1 问题与后续复审/深测产物已作为输入，本轮不重复报告已关闭问题。旧 `source-review-findings.md` 已归档到 `security/findings/archive/source-review-findings-2026-05-09.md`。

## 结论摘要

- 本轮确认 3 条新问题：1 条高风险、2 条中风险。
- 近期修复方向整体有效：账号 `base_url` 保存/更新/批量更新与模型 Probe 已进入统一校验；共享 HTTP upstream 已阻断实际 redirect 跳转；panic recovery 定向回归已通过。
- 仍需补齐 redirect 阻断后的响应头净化、Document AI 专属 URL 字段的统一出站治理，以及 Document AI 文件/上游响应读取上限。
- 按 `docs/安全实施/README.md`，步骤 1 输出本文件后暂停，等待用户修复，再进入步骤 2 对照复审。

## 验证与证据命令

- `go test -tags unit ./internal/repository -run "TestHTTPUpstreamSuite/TestDo_RedirectBlocked_ReturnsControlled502" -count=1 -v`
- `go test -tags unit ./internal/service -run "TestAccountTestServiceFormatFailedTestResponseRedirectBlocked|Test.*BaseURL|Test.*BaiduDocumentAI|Test.*DocumentAI" -count=1 -v`
- `go test -tags unit ./internal/handler/admin -run "TestProbe.*BaseURL|Test.*Probe.*Models" -count=1 -v`
- `go test -tags unit ./internal/server/middleware -run "TestRecovery" -count=1 -v`

以上命令均通过。测试只用于验证相关控制存在，不把工具命中直接当漏洞。

## Findings

### SR-20260511-001

- 风险等级：Medium
- 问题标题：上游 redirect 被改写为 502 后仍可能透传原始 `Location` 响应头
- 影响模块：共享 HTTP upstream、OpenAI/Claude/Gemini/Grok 等网关响应写回、上游出站防护
- 具体文件/函数/路由/配置：
  - `backend/internal/repository/http_upstream_redirect.go`：`finalizeResponse`
  - `backend/internal/service/upstream_redirect_blocked.go`：`RewriteUpstreamRedirectBlockedResponse`、`cloneRedirectBlockedHeader`
  - `backend/internal/util/responseheaders/responseheaders.go`：默认响应头白名单包含 `location`
  - `backend/internal/service/openai_gateway_headers.go`：`writeOpenAIPassthroughResponseHeaders`
  - 多个网关写回路径：`gateway_response_nonstreaming.go`、`gateway_response_streaming.go`、`openai_gateway_passthrough.go` 等
- 真实可达性判断：可达。共享 upstream 遇到上游 3xx 时会改写为 `502 UPSTREAM_REDIRECT_NOT_ALLOWED`，但改写响应仍克隆上游原始 header；后续网关写回使用响应头白名单，而白名单默认允许 `Location`。因此客户端可能收到状态为 502、body 为受控错误，但 header 中仍带上游 redirect 目标。
- 利用条件：
  - 调用方拥有可访问相关网关的站内 API Key。
  - 当前调度账号的上游返回 3xx，或管理员配置的允许上游/中转服务返回 3xx。
  - 上游 `Location` 包含不应暴露给下游调用方的内部地址、签名地址、租户路径或诊断信息。
- 影响范围：
  - redirect 不会被服务端继续跟随，SSRF 二跳风险已显著降低。
  - 但 `Location` 仍可能泄露上游内部跳转目标；部分客户端、中间代理或日志系统也可能记录该 header，削弱“受控 502”语义。
- 证据：
  - `finalizeResponse` 将所有 `3xx` 交给 `RewriteUpstreamRedirectBlockedResponse`。
  - `RewriteUpstreamRedirectBlockedResponse` 使用 `cloneRedirectBlockedHeader(resp.Header)` 克隆原始 header。
  - `responseheaders.defaultAllowed` 包含 `location`。
  - `writeOpenAIPassthroughResponseHeaders` 与多处 `responseheaders.WriteFilteredHeaders` 会把过滤后的 header 写入客户端响应。
  - 定向单测仅断言“不跟随二跳”和“body/status 受控”，未断言 `Location` 被剥离。
- 修复建议：
  - 在 `RewriteUpstreamRedirectBlockedResponse` 或 `cloneRedirectBlockedHeader` 中删除 `Location`、`Refresh` 等 redirect 语义 header。
  - 或在响应头过滤器中按“已阻断 redirect”上下文强制剥离 `Location`，避免影响正常 201/3xx 业务语义。
  - 增加回归测试：构造上游 `302 Location: ...`，断言最终客户端响应为 502 且无 `Location`。
- 复审方法：
  - 复查 `RewriteUpstreamRedirectBlockedResponse` 生成的 header 不含 `Location`。
  - 跑通 repository 与至少一条网关写回路径的 redirect header 回归测试。
  - 确认正常非 redirect 响应的允许 header 行为未被破坏。
- 标准映射：
  - OWASP API Security Top 10 2023：API8 Security Misconfiguration、API7 Server Side Request Forgery
  - OWASP WSTG：Information Leakage、Configuration Testing
  - OWASP ASVS：V8 Data Protection、V14 Configuration

### SR-20260511-002

- 风险等级：Medium
- 问题标题：百度 Document AI 的 `async_base_url` / `direct_api_urls` 未纳入统一上游 allowlist 策略
- 影响模块：管理员账号保存、Document AI 异步任务、Document AI 直连解析、Document AI 结果下载、出站访问治理
- 具体文件/函数/路由/配置：
  - `backend/internal/service/admin_service_accounts.go`：`validateAccountBaseURL`、`validateBaiduDocumentAIAccountInput`
  - `backend/internal/service/document_ai_credentials.go`：`normalizeBaiduDocumentAICredentialsForStorage`
  - `backend/internal/service/account_baidu_document_ai.go`：`GetBaiduDocumentAIAsyncBaseURL`、`GetBaiduDocumentAIDirectAPIURL`
  - `backend/internal/service/document_ai_baidu_client.go`：`submitAsyncJob`、`getAsyncJobStatus`、`parseDirect`、`downloadResultURL`
  - 公开路由：`POST /document-ai/v1/jobs`、`POST /document-ai/v1/models/{model}:parse`
- 真实可达性判断：可达。普通账号 `credentials.base_url` 会在创建、更新、批量更新时调用统一 `validateAccountUpstreamBaseURL`，默认要求 HTTPS、非私网、命中 `security.url_allowlist.upstream_hosts`。但百度 Document AI 使用 `async_base_url` 和 `direct_api_urls`，保存阶段只做“是否绝对 URL/非空”检查，运行时也以 `ValidateHTTPURL(..., ValidationOptions{})` 做格式与私网字面量校验，不强制命中上游 allowlist。
- 利用条件：
  - 攻击者拥有管理员权限，或管理员账号/配置链路被接管。
  - 可创建或修改 `baidu_document_ai` 账号的 `async_base_url` 或 `direct_api_urls`。
  - API Key 绑定到百度 Document AI 分组后，普通调用方即可触发这些已保存出站目标。
- 影响范围：
  - 默认私网/localhost 字面量与解析后私网访问会被现有 `urlvalidator` / `httpUpstream` 拦截，这是已有正向控制。
  - 但任意公网 HTTPS 目标仍可作为 Document AI 上游或结果下载目标，绕过了普通模型账号新补齐的统一 `upstream_hosts` 治理。
  - 一旦账号配置被误填或被接管，普通 API Key 调用者可反复触发服务端向非预期公网目标发起带百度侧 token 的请求。
- 证据：
  - `validateAccountBaseURL` 只读取并规范化 `credentials["base_url"]`。
  - `validateBaiduDocumentAIAccountInput` 对 `async_base_url` 仅检查 `http://` / `https://` 前缀，对 `direct_api_urls` 仅检查对象键和值非空。
  - `document_ai_baidu_client.go` 对上述 URL 调用 `ValidateHTTPURL` 时没有传入 `AllowedHosts` 与 `RequireAllowlist`。
  - 默认配置已启用 `security.url_allowlist.enabled=true` 且 `allow_private_hosts=false`，说明当前项目期望对上游出站进行统一治理。
- 修复建议：
  - 为 Document AI 增加专属 allowlist 配置，例如 `security.url_allowlist.document_ai_hosts`，并把官方默认域名加入默认值。
  - 保存阶段统一校验 `async_base_url` 与 `direct_api_urls`，拒绝不在 allowlist 的目标；运行时保留重复校验作为纵深防御。
  - 对 Provider 返回的 `JSONResultURL` / `MarkdownResultURL` 也设置来源约束，至少限制为 Document AI allowlist 或同源/可信结果域。
  - 补充创建、更新、批量更新、直连解析、异步 poll/result download 的回归测试。
- 复审方法：
  - 尝试保存 `baidu_document_ai` 账号到非 allowlist 公网 HTTPS 目标，应得到统一结构化错误。
  - 尝试通过 Provider 响应返回非 allowlist 结果 URL，poller 应拒绝下载并记录脱敏错误。
  - 确认官方默认百度 Document AI endpoint 仍可通过。
- 标准映射：
  - OWASP API Security Top 10 2023：API7 Server Side Request Forgery、API8 Security Misconfiguration
  - OWASP WSTG：SSRF Testing、Input Validation Testing
  - OWASP ASVS：V5 Validation, Sanitization and Encoding、V14 Configuration

### SR-20260511-003

- 风险等级：High
- 问题标题：Document AI 文件与上游响应存在大体积 `io.ReadAll` 路径，可被认证调用方或异常上游放大为内存型 DoS
- 影响模块：Document AI 上传、直连解析、异步提交、异步 poller、结果下载、进程稳定性
- 具体文件/函数/路由/配置：
  - `backend/internal/server/routes/document_ai.go`：Document AI 路由使用 `RequestBodyLimit(cfg.Gateway.MaxBodySize)`
  - `backend/internal/config/config_defaults.go`：`gateway.max_body_size` 默认 `20000 * 1024 * 1024`
  - `backend/internal/handler/document_ai_handler.go`：`readDocumentAIFile`
  - `backend/internal/service/document_ai_service.go`：`normalizeSubmitInput`、`normalizeDirectInput`、`buildAsyncResultEnvelope`
  - `backend/internal/service/document_ai_baidu_client.go`：`doJSONRequest`、`downloadResultURL`
  - 公开路由：`POST /document-ai/v1/jobs`、`POST /document-ai/v1/models/{model}:parse`
- 真实可达性判断：可达。Document AI 路由虽挂载了请求体限制，但默认值约 20GB；`readDocumentAIFile` 会把 multipart 文件完整读入内存，JSON `file_base64` 也会完整解码到 `FileBytes`。此外，百度上游 JSON 响应、异步结果 Markdown/JSON 下载均使用未加 `LimitReader` 的 `io.ReadAll`。
- 利用条件：
  - 攻击者拥有绑定到 `baidu_document_ai` 分组的有效站内 API Key。
  - 发送接近默认请求体上限的大文件或超大 `file_base64`。
  - 或已配置/被接管的 Document AI 上游返回超大 JSON、Markdown 或结果文件。
- 影响范围：
  - 单次或少量并发请求即可显著占用后端内存，可能导致进程 OOM、服务重启或同机其它请求延迟升高。
  - 异步 poller 下载结果时也会进入同类无界读入路径，风险不只存在于前台请求。
- 证据：
  - `RegisterDocumentAIRoutes` 使用 `RequestBodyLimit(cfg.Gateway.MaxBodySize)`，而默认 `gateway.max_body_size` 为 `20000 * 1024 * 1024`。
  - `readDocumentAIFile` 对上传文件直接 `io.ReadAll(file)`。
  - `normalizeDirectInput` 对 `file_base64` 直接 `base64.StdEncoding.DecodeString` 并保存到 `FileBytes`。
  - `doJSONRequest` 和 `downloadResultURL` 对上游响应直接 `io.ReadAll(resp.Body)`。
  - 其它网关主链路已有 `readUpstreamResponseBodyLimited` 或 `io.LimitReader` 模式，Document AI 未复用该防护。
- 修复建议：
  - 为 Document AI 增加独立、显式的小得多的上传大小限制，并在 handler 读取前用 `LimitReader` / 流式处理控制内存。
  - 对 `file_base64` 解码前先校验编码长度与解码后上限，避免一次性分配超大切片。
  - `doJSONRequest`、`downloadResultURL` 复用 `readUpstreamResponseBodyLimited` 或新增 Document AI 专属响应上限。
  - 对异步结果下载设置单文件上限、内容类型/长度检查和超限错误落库策略。
  - 增加超限回归测试，断言请求不会造成大内存分配且返回统一错误。
- 复审方法：
  - 使用超过 Document AI 配置上限的 multipart 与 JSON base64 输入，确认请求被拒绝且进程内存不随 payload 线性暴涨。
  - 用模拟上游返回超过响应上限的 JSON/Markdown，确认前台请求和 poller 都返回/记录受控错误。
  - 检查所有 Document AI `io.ReadAll` 路径均有明确上限或改为流式处理。
- 标准映射：
  - OWASP API Security Top 10 2023：API4 Unrestricted Resource Consumption
  - OWASP WSTG：Input Validation Testing、Business Logic Testing
  - OWASP ASVS：V1 Architecture、V5 Validation、V14 Configuration

## 本轮覆盖但未新增编号的正向控制

- 普通模型账号 `base_url` 已在创建、更新、批量更新、测试和模型 Probe 中进入统一校验链路。
- 共享 HTTP upstream 对上游 3xx 不再自动跟随，并返回统一 `UPSTREAM_REDIRECT_NOT_ALLOWED` body。
- panic recovery 已改为自定义最小必要日志，定向测试未见 Authorization/Cookie/API Key 明文输出。
- 前端账号创建流程已去掉创建后自动导入模型的副作用，并提示手动 Probe/Test。
- 5 月 9 日问题跟踪中 SR-001 到 SR-006 均记录为已解决，本轮未重复报告。

## 下一步

- 按流程暂停，等待用户修复 `SR-20260511-001`、`SR-20260511-002`、`SR-20260511-003`。
- 用户修复后进入步骤 2，输出 `security/findings/source-review-recheck.md`。
