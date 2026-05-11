# Sub2API 步骤 4：WSL 深测问题对照复审

复审日期：2026-05-10  
复审范围：`security/findings/wsl-deep-test-findings.md` 中 `STEP3-001` 到 `STEP3-004`  
复审方式：源码对照复审、定向 Go/Vitest 回归、编译切片验证、WSL 运行态现场 replay  
当前结论：四个 STEP3 问题的代码修复、定向回归与 WSL 最新 Linux 审计二进制现场 replay 均已满足关闭条件。

## 复审摘要

- `STEP3-001`：fixed
- `STEP3-002`：fixed
- `STEP3-003`：fixed
- `STEP3-004`：fixed
- 环境补记：初次探测时 `127.0.0.1:18081` 与 `127.0.0.1:19090` 未监听；后续通过审计脚本恢复运行态。曾确认 2026-05-09 旧二进制不代表当前源码；最终已用 2026-05-10 20:35 构建的新 `server-test-linux` 完成现场 replay。

## 关键变更证据

- 保存阶段 `base_url` 校验已统一复用 `validateAccountUpstreamBaseURL`，覆盖 create / update / bulk update。
- 手动 Probe 路径在构造 account-controlled upstream URL 前统一执行 `validateProbeBaseURL`，覆盖 OpenAI / DeepSeek、Anthropic / Antigravity API Key、Grok、protocol-gateway mixed probe。
- `AccountModelImportService` 已注入安全配置，生成的 `wire_gen.go` 同步传入 `configConfig`。
- 共享 HTTP upstream 保持禁止跟随 `3xx`，并把重定向改写为受控 `502 UPSTREAM_REDIRECT_NOT_ALLOWED`。
- `/v1/messages` 空 selection / 空 account 分支已有显式受控失败返回。
- recovery 中间件保持自定义最小必要日志，不再 dump 原始请求头。

## 复审明细

### STEP3-001

- 当前状态：fixed
- 原问题：管理员保存上游账号时允许异常 `base_url`，并立即触发后台模型 probe 出站。
- 当前证据：
  - `backend/internal/service/admin_service_accounts.go` 在 create / update / bulk update 保存面统一调用 `validateAccountBaseURL`。
  - `backend/internal/service/upstream_base_url_validation.go` 将校验失败包装为 `400 ACCOUNT_INVALID_BASE_URL`。
  - `backend/internal/service/admin_service_account_base_url_validation_test.go` 覆盖 create 非法地址拒绝、update 合法 HTTPS allowlist 规范化。
  - `backend/internal/service/admin_service_bulk_update_test.go` 覆盖批量更新提交 `file://`、`gopher://`、`http://127.0.0.1`、空 host、非法端口时拒绝且不调用 repo `BulkUpdate`，合法地址规范化后写入。
  - 保存后自动 probe refresh 的主体修复仍保留，前端保存提示与 API 文档已同步为手动 Probe/Test。
- 是否需要继续修改：否。
- 通过复审的方法：
  - `go test -tags unit ./internal/service -run "TestAdminService(CreateAccount_InvalidBaseURLReturnsStructuredError|UpdateAccount_NormalizesAllowedBaseURL)|TestAdminService_BulkUpdateAccounts_(InvalidBaseURLReturnsStructuredError|NormalizesAllowedBaseURL)|TestProbeAccountModels_MixedProtocolGateway" -count=1 -v`
  - `pnpm --dir frontend exec vitest run src/composables/__tests__/useCreateAccountSubmit.spec.ts src/components/account/__tests__/AccountCreateFooterActions.spec.ts src/components/account/__tests__/CreateAccountModal.spec.ts`

### STEP3-002

- 当前状态：fixed
- 原问题：`real_forward` 会跟随上游 `302`，并出现 `POST -> GET`。
- 当前证据：
  - `backend/internal/repository/http_upstream.go` / `backend/internal/repository/http_upstream_redirect.go` 在上游客户端层禁止跟随重定向。
  - `backend/internal/service/upstream_redirect_blocked.go` 提供统一 `UPSTREAM_REDIRECT_NOT_ALLOWED` 识别与受控响应体。
  - `backend/internal/repository/http_upstream_test.go` 验证 `302` 不发生第二跳，并返回受控 `502`。
  - `backend/internal/service/account_test_service_failed_response_test.go` 验证 admin test/real_forward 的 redirect blocked 错误文案稳定。
- 是否需要继续修改：否。
- 通过复审的方法：
  - `go test -tags unit ./internal/repository -run "TestHTTPUpstreamSuite/TestDo_RedirectBlocked_ReturnsControlled502" -count=1 -v`
  - `go test -tags unit ./internal/service -run "TestAccountTestServiceFormatFailedTestResponseRedirectBlocked|TestOpenAIInvalidBaseURLWhenAllowlistDisabled|TestOpenAIValidateUpstreamBaseURL" -count=1 -v`

### STEP3-003

- 当前状态：fixed
- 原问题：上游 `500` / failover 耗尽路径触发 `gateway_handler_messages.go` panic recovery。
- 当前证据：
  - `backend/internal/handler/gateway_handler_messages.go` 在 selection / account 取值前提供 `selectionAccountOrFail` 防御，空 selection 和空 account 返回受控 `502`。
  - `backend/internal/handler/gateway_handler_messages_nil_selection_unit_test.go` 覆盖 nil selection 与 nil account 两个分支。
- 是否需要继续修改：否。
- 通过复审的方法：
  - `go test -tags unit ./internal/handler -run "TestGatewayHandlerSelectionAccountOrFail" -count=1 -v`

### STEP3-004

- 当前状态：fixed
- 原问题：panic recovery 日志泄露 `Cookie`、`X-Api-Key`、`X-Client-Request-Id` 等用户输入头。
- 当前证据：
  - `backend/internal/server/middleware/recovery.go` 使用自定义恢复逻辑，仅输出 `request_id`、method、path、client ip、panic 摘要和 stack。
  - `backend/internal/server/middleware/recovery_test.go` 验证 panic 日志包含必要元数据和堆栈，但不包含 `Authorization`、`Cookie`、`X-Api-Key`、`X-Client-Request-Id` 的明文值。
- 是否需要继续修改：否。
- 通过复审的方法：
  - `go test -tags unit ./internal/server/middleware -run "TestRecovery" -count=1 -v`

## 手动 Probe 补齐复审

本轮专门复审了用户补齐计划中指出的两个缺口：

- `BulkUpdateAccounts`：已在服务层校验 `input.Credentials["base_url"]` 并写回规范化值。
- `ProbeModels` / `ProbeProtocolGatewayModels`：已在任何 account-controlled upstream 请求前执行共享校验；非法 `base_url` 时 mock upstream 请求列表为空。

通过命令：

```text
go test -tags unit ./internal/handler/admin -run "TestProbe(Models|ProtocolGatewayModels)_.*BaseURL|TestProbeProtocolGatewayModels_MixedReturnsSourceProtocolAndRegistryState" -count=1 -v
```

## 编译与回归

已通过：

```text
go test -tags unit ./cmd/server ./internal/service ./internal/handler/admin ./internal/handler ./internal/repository ./internal/server/middleware -run "TestDoesNotExist" -count=1
```

先前整包命令 `go test -tags unit ./internal/service ./internal/handler/admin` 在 120 秒内超时；随后改用上述定向回归与编译切片完成验证。

## 运行态复测状态

- 初次探测：
  - `127.0.0.1:18081`：未监听。
  - `127.0.0.1:19090`：未监听。
- 后续恢复：
  - WSL `Ubuntu` 可启动。
  - PostgreSQL `127.0.0.1:5432` 恢复为 accepting connections。
  - `.env.audit` 期望 Redis `6380`，本轮临时启动 `127.0.0.1:6380` 后管理员登录恢复。
  - `security/scripts/start_step3_host_runtime.sh` 启动后，mock upstream 监听 `19090`，server 监听 `18081`。
  - `GET /health` 返回 `200 {"status":"ok"}`。
- 旧二进制判定：
  - `backend/bin/server-test-linux` 时间为 `2026-05-09 16:14:34 +0800`。
  - `strings backend/bin/server-test-linux | grep ACCOUNT_INVALID_BASE_URL` 无命中。
  - 运行日志仍出现 `account_model_probe_refresh_started trigger=create`，说明旧的保存后自动 probe 逻辑仍在运行态中。
- 旧二进制 replay 结果：
  - `POST /api/v1/admin/accounts` 提交 `file://`、`gopher://`、非法端口、空 host 均返回 `200`，并触发保存后后台 probe。
  - `probe-models` 与 `protocol-gateway/probe-models` 对 `gopher://127.0.0.1:70` 返回 `503 MODEL_IMPORT_UPSTREAM_REQUEST_FAILED`，不是当前源码预期的 `400 ACCOUNT_INVALID_BASE_URL`。
  - 这些结果只证明运行态二进制滞后，不作为当前源码修复失败证据。
- 最新二进制：
  - `backend/bin/server-test-linux` 文件时间为 `2026-05-10 20:35:13 +0800`。
  - 文件大小为 `477616237`。
  - `ACCOUNT_INVALID_BASE_URL` 字符串命中 `5`。
  - `account_model_probe_refresh_started` 字符串命中 `0`。
- 最新二进制 replay：
  - `GET /health` 返回 `200`。
  - `POST /api/v1/admin/accounts` 提交 `file://`、`gopher://`、非法端口、空 host 均返回 `400 ACCOUNT_INVALID_BASE_URL`。
  - `probe-models` 非法 `base_url` 返回 `400 ACCOUNT_INVALID_BASE_URL`。
  - `protocol-gateway/probe-models` 正确请求体下非法 `base_url` 返回 `400`，`reason=ACCOUNT_INVALID_BASE_URL`。
  - admin real_forward redirect：mock 只收到 `POST /redirect-local/v1/messages?beta=true` 一次；无 `/final/` 第二跳，无 `GET` 降级。
  - `/v1/messages` 上游 500：本地审计 runtime key 通过余额与分组检查后命中 mock `POST /status/500/v1/messages?beta=true`；网关返回受控 `502`，无 `panic recovered`。
  - recovery / access log 脱敏扫描：日志未出现发送的 `Cookie`、`X-Api-Key`、`X-Client-Request-Id` 哨兵值，也未出现 runtime key 明文。
- 现场 replay 摘要：
  - `tmp/step4_replay_summary.json`
  - `tmp/step4_replay_followup_summary.json`
- 运行态补验报告：
  - `security/reports/manual/2026-05-10-step4-runtime-replay.md`
