# 2026-05-10 Step 4 运行态补验记录

范围：`STEP3-001` 到 `STEP3-004` 的 WSL 本地运行态补验。

## 环境状态

- WSL `Ubuntu`：可启动。
- PostgreSQL：`127.0.0.1:5432` 后续恢复为 accepting connections。
- Redis：
  - 现有 `127.0.0.1:6379` 可 `PONG`。
  - `.env.audit` 配置为 `127.0.0.1:6380`；本轮临时启动审计 Redis `6380` 后登录恢复。
- mock upstream：通过 `security/scripts/start_step3_host_runtime.sh` 启动后监听 `0.0.0.0:19090`。
- server：通过同一脚本启动后监听 `127.0.0.1:18081`。
- `/health`：返回 `200 {"status":"ok"}`。

## 历史阻塞：旧二进制（已解除）

先前运行的 `backend/bin/server-test-linux` 不是最新修复代码构建产物：

- 二进制时间：`2026-05-09 16:14:34 +0800`。
- `strings backend/bin/server-test-linux | grep ACCOUNT_INVALID_BASE_URL` 无命中。
- 运行日志仍出现 `account_model_probe_refresh_started trigger=create`，说明仍包含已在源码中移除的保存后自动 probe 旧逻辑。

因此，对该二进制的运行态结果只用于证明审计运行态曾是旧版本，不能作为当前源码修复是否通过的证据。

## 旧二进制 replay 结果

在临时恢复 `6380` Redis 后：

- 管理员登录：`200`，拿到本地临时访问令牌。
- `POST /api/v1/admin/accounts`：
  - `file:///etc/passwd` 返回 `200`。
  - `gopher://127.0.0.1:70` 返回 `200`。
  - `http://127.0.0.1:99999` 返回 `200`。
  - `http:///missing-host` 返回 `200`。
  - 日志出现保存后 `account_model_probe_refresh_started` 与对应 probe 失败。
- `POST /api/v1/admin/accounts/probe-models` + `gopher://127.0.0.1:70`：返回 `503 MODEL_IMPORT_UPSTREAM_REQUEST_FAILED`，不是 `400 ACCOUNT_INVALID_BASE_URL`。
- `POST /api/v1/admin/accounts/protocol-gateway/probe-models` + `gopher://127.0.0.1:70`：返回 `503 MODEL_IMPORT_UPSTREAM_REQUEST_FAILED`，不是 `400 ACCOUNT_INVALID_BASE_URL`。

结论：这些结果与当前源码和定向测试不一致，确认运行态二进制滞后。

## 最新二进制重建尝试（历史）

- Windows 侧交叉构建：
  - 命令：`GOOS=linux GOARCH=amd64 go build -tags embed -o bin/server-test-linux ./cmd/server`
  - 结果：304 秒超时；残留 `go` 进程已结束。
- WSL 侧原生构建：
  - 命令：`cd backend && go build -tags embed -o bin/server-test-linux ./cmd/server`
  - 结果：尝试下载 Go `1.26.3` toolchain，访问 `proxy.golang.org` 超时。

## 最新二进制与运行态 replay

后续已成功构建并启动当前源码对应的 Linux 审计二进制：

- 二进制：`backend/bin/server-test-linux`
- 文件时间：`2026-05-10 20:35:13 +0800`
- 文件大小：`477616237`
- `ACCOUNT_INVALID_BASE_URL` 字符串命中：`5`
- `account_model_probe_refresh_started` 字符串命中：`0`
- 审计运行态：server `127.0.0.1:18081`，mock upstream `0.0.0.0:19090`，Redis `127.0.0.1:6380`，PostgreSQL `127.0.0.1:5432`
- `/health`：`200`

当前源码二进制 replay 结果：

- 管理员登录：`200`，访问令牌仅在本地脚本内使用，未落报告。
- `POST /api/v1/admin/accounts` 提交 `file://`、`gopher://`、非法端口、空 host：均返回 `400`，响应包含 `ACCOUNT_INVALID_BASE_URL`。
- `POST /api/v1/admin/accounts/probe-models` 使用非法 `base_url`：返回 `400`，响应包含 `ACCOUNT_INVALID_BASE_URL`。
- `POST /api/v1/admin/accounts/protocol-gateway/probe-models` 使用正确请求体与非法 `base_url`：返回 `400`，响应 `reason=ACCOUNT_INVALID_BASE_URL`。
- admin real_forward redirect：mock 只收到 `POST /redirect-local/v1/messages?beta=true` 一次；没有 `/final/` 第二跳，也没有 `GET` 降级。
- `/v1/messages` 上游 500 / failover：使用本地审计用户、正余额、分组绑定 runtime key 触发，mock 收到 `POST /status/500/v1/messages?beta=true`；网关最终返回受控 `502 {"error":{"message":"No available accounts","type":"api_error"},"type":"error"}`，未出现 panic recovery。
- recovery / access log 脱敏扫描：发送 `Cookie`、`X-Api-Key`、`X-Client-Request-Id` 哨兵及 runtime key；server 日志未出现这些哨兵值，也未出现 runtime key 明文。

补验摘要文件：

- 初次最新二进制 replay：`tmp/step4_replay_summary.json`
- `/v1/messages` 余额补齐后 follow-up replay：`tmp/step4_replay_followup_summary.json`

## 本轮结论

- 最新 Linux 审计二进制已代表当前源码完成 WSL 现场 replay。
- `STEP3-001` 到 `STEP3-004` 均有源码、定向回归与运行态证据支撑关闭。
- 不再保留 `WSL-RECHECK-LATEST-BINARY` / `WSL-RECHECK-RUNTIME` 阻塞。

## 清理

- replay 完成后已停止本轮启动的 server、mock upstream 与临时 Redis `6380`。
- 清理后审计运行态端口 `18081`、`19090`、`6380` 均不再监听。
