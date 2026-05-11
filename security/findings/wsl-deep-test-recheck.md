# Sub2API 步骤 4：WSL 深测问题对照复审

复审日期：2026-05-11  
复审范围：`security/findings/wsl-deep-test-findings.md` 中 `STEP3-20260511-001`  
复审方式：源码对照复审、定向 Go 回归、Linux 审计二进制校验、本地 WSL / 主机运行态 replay  
当前结论：`STEP3-20260511-001` 的代码修复、定向回归与本地运行态 replay 均满足关闭条件。

## 复审摘要

- `STEP3-20260511-001`：fixed
- 续跑起点：严格沿用 `docs/安全实施/SECURITY_IMPLEMENTATION_PROGRESS.md` 当前步骤 4，不回退步骤 1-3。
- 历史 `security/findings/wsl-deep-test-recheck.md` 已归档到 `security/findings/archive/wsl-deep-test-recheck-2026-05-10.md`。

## 关键变更证据

- `backend/internal/service/admin_service_accounts.go`
  - `newBaiduDocumentAIValidationError` 不再返回普通 `errors.New(reason)`。
  - 当前统一返回 `infraerrors.BadRequest("ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS", reason)`。
- `backend/internal/pkg/response/localization.go`
  - 已补充 `ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS` 本地化文案。
- `backend/internal/handler/admin/account_handler_document_ai_error_test.go`
  - 创建与批量更新 handler 均断言返回 `400 ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS`
  - 响应体不含 `internal error`
- `backend/internal/service/docs/pages/common.md`
- `backend/internal/service/docs/pages/document-ai.md`
  - 已同步说明 Document AI 账号 URL / 凭证校验失败返回统一 `400 ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS`

## 复审明细

### STEP3-20260511-001

- 当前状态：fixed
- 原问题：Document AI 非 allowlist URL 保存 / 批量更新被校验拒绝后，对外仍返回 `500 internal error`
- 当前证据：
  - Linux 审计二进制 `backend/bin/server-test-linux` 文件时间为 `2026-05-11 14:30`，且已命中 `ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS` 字符串，无需先重建二进制。
  - service 定向回归通过：
    - `go test -tags unit ./internal/service -run "BaiduDocumentAI|BulkUpdateAccounts_.*BaiduDocumentAI|Account.*BaseURL" -count=1 -v`
  - handler 定向回归通过：
    - `go test -tags unit ./internal/handler/admin -run "DocumentAIValidationError|Localization|TouchedAccount" -count=1 -v`
  - response 定向回归通过：
    - `go test ./internal/pkg/response -run "Localization|ErrorFrom" -count=1 -v`
  - 运行态 replay 摘要：`tmp/step4_document_ai_recheck_20260511_summary.json`
    - `document_ai_disallowed_url_save.status = 400`
    - `document_ai_disallowed_url_save.reason = ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS`
    - `document_ai_disallowed_url_bulk_update.status = 400`
    - `document_ai_disallowed_url_bulk_update.reason = ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS`
    - 两条负向响应均不含 `internal error`
    - `document_ai_allowed_url_save.status = 200`
    - `document_ai_allowed_url_bulk_update.status = 200`
  - 服务日志：
    - `baidu document ai account validation failed`
    - `POST /api/v1/admin/accounts` 负向 replay 返回 `400`
    - `POST /api/v1/admin/accounts/bulk-update` 负向 replay 返回 `400`
    - 正向 allowlist 保存 / 批量更新返回 `200`
  - 本轮 manual report：`security/reports/manual/2026-05-11-step4-document-ai-recheck.md`
- 是否需要继续修改：否
- 通过复审的方法：
  - 复用当前 WSL / 主机运行态 `127.0.0.1:18081`
  - 重放两条非 allowlist URL 用例，确认统一 `400 ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS`
  - 重放两条 allowlist 内默认百度 endpoint 用例，确认 `200`
  - 核对 `server-audit.log` 无 `internal error`、无管理员 token / 哨兵 header 泄露
  - 核对 mock 日志本轮 `mock_request_count = 0`，确认校验拒绝未产生非预期出站

## 运行态复测状态

- 当前监听：
  - `127.0.0.1:18081`：`server-test-linux`
  - `0.0.0.0:19090`：`mock_upstream.py`
- `/health`：返回 `200 {"status":"ok"}`
- 本轮 replay：
  - Document AI 非 allowlist `direct_api_urls` 保存：`400`
  - Document AI 非 allowlist `direct_api_urls` 批量更新：`400`
  - Document AI 默认百度 allowlist endpoint 保存：`200`
  - Document AI 默认百度 allowlist endpoint 批量更新：`200`
- 日志与出站：
  - 已观察到结构化 validation warn
  - 未观察到 `internal error`
  - 未观察到管理员 token 或本轮哨兵值泄露
  - mock 未收到本轮请求，符合“保存面即阻断”的预期

## 最终结论

- `STEP3-20260511-001` 已关闭，状态为 `fixed`
- 本轮步骤 4 已完成；无高优先级未关闭问题
- 按 `docs/安全实施/README.md`，本次续跑在步骤 4 复审产物与状态文件更新后停止，不默认生成 `security/security-audit-report.md`
