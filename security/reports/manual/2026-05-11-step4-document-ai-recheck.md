# 2026-05-11 Step 4 Document AI 运行态复审记录

范围：`security/findings/wsl-deep-test-findings.md` 中 `STEP3-20260511-001`

## 执行边界

- 仅复用本地 WSL / 主机隔离环境，不访问 production / staging。
- 不访问真实第三方上游；本轮只复审管理端账号保存 / 批量更新错误语义与 allowlist 正常路径。
- 不输出完整密钥、Token、Cookie、密码。

## 预检结论

- `backend/bin/server-test-linux` 文件时间为 `2026-05-11 14:30`，且已命中：
  - `ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS`
  - `ACCOUNT_INVALID_BASE_URL`
- 因此本轮无需先重建 Linux 审计二进制，直接复用当前运行态。
- WSL 当前监听状态：
  - `127.0.0.1:18081`：`server-test-linux`
  - `0.0.0.0:19090`：`mock_upstream.py`

## 定向测试

已通过：

```text
cd backend
go test -tags unit ./internal/service -run "BaiduDocumentAI|BulkUpdateAccounts_.*BaiduDocumentAI|Account.*BaseURL" -count=1 -v
go test -tags unit ./internal/handler/admin -run "DocumentAIValidationError|Localization|TouchedAccount" -count=1 -v
go test ./internal/pkg/response -run "Localization|ErrorFrom" -count=1 -v
```

结论：

- service 层已将 Document AI 非 allowlist URL 拒绝映射为结构化 `BadRequest`
- handler 层创建 / 批量更新均返回 `400 ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS`
- response 层不再退化成 `500 internal error`

## 运行态 replay

脚本：

```text
python tmp/step4_document_ai_recheck_20260511.py
```

摘要文件：

```text
tmp/step4_document_ai_recheck_20260511_summary.json
```

### 负向用例

- `POST /api/v1/admin/accounts`
  - `credentials.direct_api_urls.pp-ocrv5-server = gopher://127.0.0.1:70`
  - 返回：`400`
  - `reason = ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS`
  - 响应体不含 `internal error`

- `POST /api/v1/admin/accounts/bulk-update`
  - 同类非 allowlist URL
  - 返回：`400`
  - `reason = ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS`
  - 响应体不含 `internal error`

### 正向用例

- `POST /api/v1/admin/accounts`
  - `async_base_url = https://paddleocr.aistudio-app.com/api/v2/ocr`
  - `direct_api_urls.pp-ocrv5-server = https://paddleocr.aistudio-app.com/api/v2/ocr/pp-ocrv5`
  - 返回：`200`
  - 成功创建账号 `id=34`

- `POST /api/v1/admin/accounts/bulk-update`
  - 对同一账号提交 allowlist 内 URL
  - 返回：`200`
  - `success=1 failed=0`

## 日志核对

服务日志命中：

- `baidu document ai account validation failed`
- `POST /api/v1/admin/accounts` 负向请求：`400`
- `POST /api/v1/admin/accounts/bulk-update` 负向请求：`400`
- 正向保存 / 批量更新：`200`

未观察到：

- `internal error`
- 管理员 token 明文
- 本轮脚本注入的 `Cookie` / `X-Client-Request-Id` 哨兵值

mock 日志观察：

- 本轮 replay `mock_request_count = 0`
- 与预期一致：Document AI 账号 URL 校验在保存 / 批量更新阶段即拦截，未发生非预期出站

## 复审结论

- `STEP3-20260511-001`：满足关闭条件
  - 负向保存：`400 ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS`
  - 负向批量更新：`400 ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS`
  - 正向默认百度 allowlist endpoint：创建 / 更新均通过
  - 日志保留结构化校验 reason，未退化为 `500 internal error`
  - 未观察到敏感值泄露或非预期出站
