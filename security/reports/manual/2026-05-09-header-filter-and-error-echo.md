# 2026-05-09 Header Filter / Error Echo 验证证据

## 运行基线

- `server-test-linux` 监听：`127.0.0.1:18081`
- `mock_upstream.py` 监听：`127.0.0.1:19090`
- 仅访问本地 mock upstream，不访问真实第三方、云 metadata 或未授权目标

## 用户态头过滤验证

- 测试路径：`POST /v1/messages` -> mock `/echo-headers/v1/messages?beta=true`
- 输入侧刻意注入了 `Authorization`、`Cookie`、`X-Api-Key`、`X-Client-Request-Id`、`Anthropic-Beta`、`User-Agent`
- 观测结果：
  - 上游认证头由服务端重写为固定 `x-api-key=sk-ant-mock-audit`
  - 用户注入的 `Authorization` 与 `Cookie` 未透传到 mock upstream
  - 允许透传的 `anthropic-beta`、`x-claude-code-session-id`、`x-client-request-id`、`user-agent` 仍可到达上游

## 错误路径与回显验证

- 管理端测试路径：`POST /api/v1/admin/accounts/10/test`
  - 指向 mock `/status/500/v1/messages?beta=true`
  - 结果：`200` SSE，仅返回通用 `upstream error: 500 (failover)`，未见上游 body 回显到客户端
- 用户态测试路径：`POST /v1/messages`
  - 指向同一 `500` mock
  - 结果：`500 {"code":500,"message":"internal error"}`
  - 但服务端触发 panic recovered，恢复日志中出现明文请求头

## 关键日志锚点

- `security/reports/manual/mock-upstream.log`
  - `2026-05-09T18:34:58+0800`：`POST /echo-headers/v1/messages?beta=true`
  - `2026-05-09T18:34:58+0800`：`POST /status/500/v1/messages?beta=true`
- `security/reports/manual/server-audit.log`
  - `2026-05-09T18:34:58.885+0800`：`account_upstream_error`
  - `2026-05-09T18:34:58.885+0800`：`gateway.failover_switch_account`
  - `2026/05/09 18:34:58 [Recovery] ... panic recovered`

## 当前结论

- 用户态可成功阻断对上游 `Authorization` / `Cookie` / `x-api-key` 的直接覆盖。
- 但上游 `500` 的失败路径仍存在 panic 与恢复日志泄露风险，需继续修复与复审。
