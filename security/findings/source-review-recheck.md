# Sub2API 步骤 2：源码修复后的对照复审

复审日期：2026-05-11  
对照问题单：`security/findings/source-review-findings.md`  
复审范围：`SR-20260511-001`、`SR-20260511-002`、`SR-20260511-003` 对应的源码、配置、测试与内置文档  
执行边界：仅执行源码对照复审与本地定向验证；未执行生产主动攻击；未输出完整密钥、Token、Cookie、密码；未生成 `wsl-deep-test-findings.md`。

## 复审结论

- 本轮已按 `security/findings/source-review-findings.md` 对 3 条 finding 逐项复审。
- `SR-20260511-001`、`SR-20260511-002`、`SR-20260511-003` 当前状态均为 `fixed`。
- 未发现本轮修复引入新的高优先级源码问题。
- 可将流程推进到步骤 3 准备态：WSL 部署测试版 + 本地深度逆向/渗透/攻击验证。

## 复审结果

### SR-20260511-001

- 当前状态：fixed
- 当前证据：
  - `backend/internal/service/upstream_redirect_blocked.go` 的 `cloneRedirectBlockedHeader` 在克隆上游响应头后显式删除 `Location` 与 `Refresh`。
  - `RewriteUpstreamRedirectBlockedResponse` 仍保留受控 `502`、JSON 错误体与 `X-Sub2API-Upstream-Redirect-Blocked: true` 标记。
  - `backend/internal/repository/http_upstream_test.go` 的 `TestDo_RedirectBlocked_ReturnsControlled502` 构造上游 `302 Location/Refresh`，断言最终响应状态为 `502`，错误码为 `UPSTREAM_REDIRECT_NOT_ALLOWED`，且 `Location` / `Refresh` 为空。
- 是否需要继续修改：否。
- 通过复审的方法：
  - 人工核对 `RewriteUpstreamRedirectBlockedResponse` 与 `cloneRedirectBlockedHeader`。
  - 运行 `go test -tags unit ./internal/repository -run "RedirectBlocked|HTTPUpstream" -count=1 -v`：通过。

### SR-20260511-002

- 当前状态：fixed
- 当前证据：
  - `backend/internal/config/config_types_core.go` 增加 `security.url_allowlist.document_ai_hosts`，`backend/internal/config/config_defaults.go` 默认包含 `paddleocr.aistudio-app.com`。
  - `backend/internal/service/document_ai_security.go` 新增 `validateDocumentAIURLWithConfig`：allowlist 开启时强制命中 `document_ai_hosts`；allowlist 关闭时仍执行最小 URL 格式校验并遵守 `allow_insecure_http`。
  - `backend/internal/service/admin_service_accounts.go` 在创建、更新与批量更新链路中调用 `validateBaiduDocumentAIAccountInput` / `validateBaiduDocumentAIURLFields`，覆盖 `credentials.async_base_url` 与 `credentials.direct_api_urls`。
  - `backend/internal/service/document_ai_baidu_client.go` 在运行时对 async submit、async status、direct parse 与结果下载 URL 再次调用 `validateDocumentAIURLWithConfig`。
  - `file_url` 未被加入 provider allowlist，仍由现有私网/localhost 拦截策略处理，未破坏用户文件来源 URL 场景。
  - `backend/internal/service/docs/pages/document-ai.md`、`backend/internal/service/docs/pages/common.md`、`deploy/config.example.yaml` 与 README 已同步说明 Document AI allowlist。
- 是否需要继续修改：否。
- 通过复审的方法：
  - 人工核对保存、批量更新、运行时 async/direct/result URL 链路。
  - 运行 `go test -tags unit ./internal/service -run "BaiduDocumentAI|DocumentAI|BaseURL|Account.*BaseURL|BulkUpdateAccounts_.*BaiduDocumentAI" -count=1 -v`：通过。
  - 运行 `go test ./internal/config -run "DefaultSecurityToggles|DocumentAI|URLAllowlist|Defaults" -count=1 -v`：通过。

### SR-20260511-003

- 当前状态：fixed
- 当前证据：
  - `backend/internal/config/config_types_gateway.go`、`backend/internal/config/config_defaults.go` 与 `backend/internal/config/config_validate.go` 增加并校验三类 Document AI 专属限制：
    - `gateway.document_ai_upload_max_bytes` 默认 50MB。
    - `gateway.document_ai_upstream_json_read_max_bytes` 默认 10MB。
    - `gateway.document_ai_result_read_max_bytes` 默认 100MB。
  - `backend/internal/handler/document_ai_handler.go` 的 multipart 文件读取使用 `io.LimitReader(file, maxBytes+1)`，超限返回 `document_ai_invalid_request`。
  - `backend/internal/service/document_ai_service.go` 在 submit/direct 归一化时检查 `FileBytes` 长度；对 JSON `file_base64` 先用 `decodedBase64SizeWithinLimit` 估算解码后大小，再解码并二次校验。
  - `backend/internal/service/document_ai_baidu_client.go` 的 provider JSON 读取和异步结果下载均改为 `readAllLimited`，分别使用上游 JSON 与结果下载上限。
  - `backend/internal/service/docs/pages/document-ai.md`、README 与 `deploy/config.example.yaml` 已同步说明大小限制。
- 是否需要继续修改：否。
- 通过复审的方法：
  - 人工核对 multipart、base64、provider JSON 与结果下载读取路径。
  - 运行 `go test -tags unit ./internal/handler -run "DocumentAI" -count=1 -v`：通过。
  - 运行 `go test -tags unit ./internal/service -run "BaiduDocumentAI|DocumentAI|BaseURL|Account.*BaseURL|BulkUpdateAccounts_.*BaiduDocumentAI" -count=1 -v`：通过。
  - 运行临时复审测试 `TestBaiduDocumentAIClientLimitsResultDownloadResponseRecheck` 验证结果下载超限返回受控错误：通过；临时测试文件已删除，未保留业务源码改动。
  - 未执行内存耗尽压测。

## 验证命令

- `go test -tags unit ./internal/repository -run "RedirectBlocked|HTTPUpstream" -count=1 -v`：通过。
- `go test -tags unit ./internal/service -run "BaiduDocumentAI|DocumentAI|BaseURL|Account.*BaseURL|BulkUpdateAccounts_.*BaiduDocumentAI" -count=1 -v`：通过。
- `go test -tags unit ./internal/handler -run "DocumentAI" -count=1 -v`：通过。
- `go test ./internal/config -run "DefaultSecurityToggles|DocumentAI|URLAllowlist|Defaults" -count=1 -v`：通过。
- `go test ./cmd/server -count=1`：通过。
- 临时复审测试 `TestBaiduDocumentAIClientLimitsResultDownloadResponseRecheck`：通过，测试文件已删除。

## 新问题

- 当前未发现 `new_issue`。

## 下一步

- 按 README 推进到步骤 3：WSL 部署测试版 + 本地深度逆向/渗透/攻击验证。
- 本轮不生成 `security/findings/wsl-deep-test-findings.md`，该文件应在步骤 3 正式开始后生成。
