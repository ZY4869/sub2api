# Sub2API AI 安全实施总控

本目录用于给 AI 一个长期复用的安全审查执行入口，但流程只保留你真正需要的两轮闭环：

1. 本地源码深度审查 -> 输出问题 MD -> 你修复 -> AI 对照复审
2. 本地 WSL 部署测试版 -> 深度逆向/渗透/攻击验证 -> 输出问题 MD -> 你修复 -> AI 对照复审

这份文档默认不让 AI 直接改代码，而是让 AI 先把问题查清、写清、复审清。
同时，`docs/审查测试/` 里的每一项流程都必须强制执行，不能只挑一部分。

## 快速使用

把本文件交给 AI，并使用下面这句作为起始指令：

```text
请按 docs/安全实施/README.md 执行 Sub2API 安全审查流程。
如果 docs/安全实施/SECURITY_IMPLEMENTATION_PROGRESS.md 不存在，请先创建；
如果已存在，请读取后从当前步骤继续。
```

## 审查日记要求

为了方便以后按天追踪，每次开始执行时，AI 都必须先处理 `docs/审查日记/`。

强制规则：

- 审查日记目录固定为 `docs/审查日记/`
- 每次执行时，根据当天本地日期创建或复用当天目录
- 日期目录格式固定为 `YYYY-MM-DD`
- 例如 2026-05-08 当天，对应目录为 `docs/审查日记/2026-05-08/`
- 如果当天目录已存在，则继续复用，不重复创建第二个同日目录
- 当天目录中至少维护一个 `审查记录.md`
- 当天目录中还必须维护一个 `问题跟踪.md`

`审查记录.md` 至少要记录：

- 今天日期
- 当前执行步骤
- 今天已覆盖的 `docs/审查测试/00-20` 条目
- 今天新生成的问题单或复审单
- 当前阻塞项
- 下一步动作

`问题跟踪.md` 用于按天快速记录问题处理状态，要求尽量简短，一行一条。

每条记录建议格式：

```text
问题：04 是否解决：是 是否本地压测：是 备注：源码复审通过
```

最少字段：

- 问题编号
- 是否解决
- 是否本地压测

可选字段：

- 问题来源
- 备注

用途区分：

- `docs/审查日记/YYYY-MM-DD/审查记录.md`：按天记录执行过程
- `docs/审查日记/YYYY-MM-DD/问题跟踪.md`：按天记录问题状态简表
- `security/findings/*.md`：正式问题单与复审结果
- `security/*.md`：正式审计产物

## 这份文档的真实工作方式

它不是一个复杂的大总控状态机，而是一个很轻量的四步闭环：

1. 第一步：本地源码深度审查
2. 第二步：你修复后，AI 按问题单对照复审
3. 第三步：WSL 部署测试版并做本地深度逆向/渗透/攻击验证
4. 第四步：你修复后，AI 再按问题单对照复审

只有两个核心确认点：

- 第一步输出源码问题 MD 后停下，等你修复
- 第三步输出 WSL 深测问题 MD 后停下，等你修复

除非你明确要求，否则 AI 不在这套流程里直接实施修复。

## 必读输入

### 每次开始都必须先读

1. `docs/安全实施/README.md`
2. `docs/安全实施/SECURITY_IMPLEMENTATION_PROGRESS.md`
3. `docs/security_audit_20260324.md`
4. `docs/security_audit_20260508.md`
5. 仓库中的 `README.md`、`README_CN.md`、`README_EN.md`
6. 仓库中的 `backend/`、`frontend/`、`deploy/`、`tools/`

### 必须强制执行的审查测试文档

AI 每轮执行都必须覆盖并留痕以下全部文档，不允许跳过，不允许只做“按需参考”：

- `docs/审查测试/00_README_EXECUTION_ORDER.md`
- `docs/审查测试/01_SCOPE_AND_RULES.md`
- `docs/审查测试/02_ENV_AND_WSL_ISOLATED_LAB.md`
- `docs/审查测试/03_CODEX_CLI_MASTER_PROMPTS.md`
- `docs/审查测试/04_ATTACK_SURFACE_MAP.md`
- `docs/审查测试/05_LOCAL_SOURCE_REVIEW_BACKEND_AUTH_APIKEY.md`
- `docs/审查测试/06_LOCAL_SOURCE_REVIEW_AI_GATEWAY_BILLING_UPSTREAM.md`
- `docs/审查测试/07_LOCAL_SOURCE_REVIEW_FRONTEND_DEPLOY_DATAMANAGEMENT.md`
- `docs/审查测试/08_LOCAL_SCANNERS_AND_REPORT_IMPORT.md`
- `docs/审查测试/09_REMOTE_TEST_LEVELS_AND_PRODUCTION_BASELINE.md`
- `docs/审查测试/10_REMOTE_ASSET_ENUM_AND_STATIC_REVERSE.md`
- `docs/审查测试/11_REMOTE_AUTH_MATRIX_AND_APIKEY_LIFECYCLE.md`
- `docs/审查测试/12_REMOTE_AI_GATEWAY_BUSINESS_LOGIC.md`
- `docs/审查测试/13_REMOTE_UPSTREAM_SSRF_AND_EGRESS.md`
- `docs/审查测试/14_REMOTE_NUCLEI_ZAP_API_SCANNING.md`
- `docs/审查测试/15_DEPLOYMENT_HARDENING_NGINX_DOCKER_VPS.md`
- `docs/审查测试/16_LOGGING_AUDIT_FORENSICS.md`
- `docs/审查测试/17_BACKUP_UPGRADE_DATAMANAGEMENTD.md`
- `docs/审查测试/18_REVERSE_MAPPING_SOURCE_CONFIG.md`
- `docs/审查测试/19_FINAL_REPORT_REMEDIATION_RETEST.md`
- `docs/审查测试/20_STANDARDS_MAPPING_CHECKLIST.md`

### 4 步闭环与审查测试映射

虽然总控只保留 4 步，但 `00-20` 的每项都必须在 4 步中被执行、被记录、被标记状态。

#### 步骤 1 和步骤 2 主要消化这些项

- `docs/审查测试/05_LOCAL_SOURCE_REVIEW_BACKEND_AUTH_APIKEY.md`
- `docs/审查测试/06_LOCAL_SOURCE_REVIEW_AI_GATEWAY_BILLING_UPSTREAM.md`
- `docs/审查测试/07_LOCAL_SOURCE_REVIEW_FRONTEND_DEPLOY_DATAMANAGEMENT.md`
- `docs/审查测试/16_LOGGING_AUDIT_FORENSICS.md`
- `docs/审查测试/04_ATTACK_SURFACE_MAP.md`
- `docs/审查测试/15_DEPLOYMENT_HARDENING_NGINX_DOCKER_VPS.md`
- `docs/审查测试/19_FINAL_REPORT_REMEDIATION_RETEST.md`
- `docs/审查测试/20_STANDARDS_MAPPING_CHECKLIST.md`

#### 步骤 3 和步骤 4 主要消化这些项

- `docs/审查测试/01_SCOPE_AND_RULES.md`
- `docs/审查测试/02_ENV_AND_WSL_ISOLATED_LAB.md`
- `docs/审查测试/03_CODEX_CLI_MASTER_PROMPTS.md`
- `docs/审查测试/08_LOCAL_SCANNERS_AND_REPORT_IMPORT.md`
- `docs/审查测试/09_REMOTE_TEST_LEVELS_AND_PRODUCTION_BASELINE.md`
- `docs/审查测试/10_REMOTE_ASSET_ENUM_AND_STATIC_REVERSE.md`
- `docs/审查测试/11_REMOTE_AUTH_MATRIX_AND_APIKEY_LIFECYCLE.md`
- `docs/审查测试/12_REMOTE_AI_GATEWAY_BUSINESS_LOGIC.md`
- `docs/审查测试/13_REMOTE_UPSTREAM_SSRF_AND_EGRESS.md`
- `docs/审查测试/14_REMOTE_NUCLEI_ZAP_API_SCANNING.md`
- `docs/审查测试/17_BACKUP_UPGRADE_DATAMANAGEMENTD.md`
- `docs/审查测试/18_REVERSE_MAPPING_SOURCE_CONFIG.md`
- `docs/审查测试/19_FINAL_REPORT_REMEDIATION_RETEST.md`
- `docs/审查测试/20_STANDARDS_MAPPING_CHECKLIST.md`

要求：

- AI 必须执行并记录 `00-20` 每一项
- 如果某一项在当前轮次不适合真正动手，也必须写明“为什么现在不执行主动动作”以及“以什么替代方式完成核对”
- 不允许出现“未覆盖”“按需跳过”“以后再看”但没有留痕说明的情况

## 固定执行顺序

以后 AI 必须只按下面 4 步走，不要自行扩展成更多阶段。
但在这 4 步内部，必须把 `docs/审查测试/00-20` 全部执行完并记录状态：

### 步骤 1：本地源码深度审查

目标：

- 只做本地源码与配置深审
- 结合两份现有审计报告，避免重复劳动
- 找出真实问题，不做泛泛建议

必须输出：

```text
security/findings/source-review-findings.md
```

问题单必须至少包含：

- 编号
- 风险等级
- 问题标题
- 影响模块
- 具体文件/函数/路由/配置
- 真实可达性判断
- 利用条件
- 影响范围
- 证据
- 修复建议
- 复审方法

步骤 1 完成后：

- 更新进度文件
- 更新 `00-20` 覆盖状态
- 停止继续执行
- 等你修复后再进入步骤 2

### 步骤 2：源码修复后的对照复审

目标：

- 按 `source-review-findings.md` 逐项复审
- 判断每个问题是已修复、部分修复、未修复还是引入新问题

必须输出：

```text
security/findings/source-review-recheck.md
```

复审结果必须至少包含：

- 问题编号
- 当前状态：`fixed / partially_fixed / not_fixed / new_issue`
- 当前证据
- 是否需要继续修改
- 通过复审的方法

如果源码复审仍有未关闭问题：

- 更新进度文件
- 更新 `00-20` 覆盖状态
- 停在步骤 2
- 等你继续修复

只有当源码复审结果没有高优先级未关闭问题时，才进入步骤 3。

### 步骤 3：WSL 部署测试版 + 本地深度逆向/渗透/攻击验证

目标：

- 在本地 WSL 隔离环境部署测试版本
- 做尽可能深入的本地逆向、渗透、攻击验证
- 优先发现源码审查看不到、但运行态能暴露的问题

允许的方向：

- 认证态矩阵验证
- API Key 生命周期验证
- 模型权限、计费、额度、并发、限流验证
- Base URL / SSRF / 出站访问验证
- 前端静态资源逆向
- 本地扫描工具结果归并
- 备份、恢复、升级、`datamanagementd` 验证

必须保持边界：

- 只在本地 WSL 或明确授权的隔离环境做深测
- 不对未授权目标执行任何主动攻击
- 不默认进入 production / staging

必须输出：

```text
security/findings/wsl-deep-test-findings.md
```

问题单必须至少包含：

- 编号
- 风险等级
- 问题标题
- 测试场景
- 触发步骤
- 真实可利用性
- 影响范围
- 证据
- 可能根因
- 修复建议
- 复审方法

步骤 3 完成后：

- 更新进度文件
- 更新 `00-20` 覆盖状态
- 停止继续执行
- 等你修复后再进入步骤 4

### 步骤 4：WSL 深测问题的对照复审

目标：

- 按 `wsl-deep-test-findings.md` 逐项复审
- 确认运行态问题是否真正关闭

必须输出：

```text
security/findings/wsl-deep-test-recheck.md
```

复审结果必须至少包含：

- 问题编号
- 当前状态：`fixed / partially_fixed / not_fixed / new_issue`
- 当前证据
- 是否需要继续修改
- 通过复审的方法

如果步骤 4 通过，可以再按需生成：

```text
security/security-audit-report.md
```

但这不是默认必须动作，除非你明确要求做最终汇总。

## 审查测试强制覆盖规则

AI 必须为 `docs/审查测试/00-20` 每一项写出覆盖状态，并写入进度文件。

每项只允许以下状态：

- `done`
- `done_with_note`
- `blocked`

判定规则：

- `done`：该项要求已实际执行并产出证据
- `done_with_note`：该项因当前闭环边界未做主动动作，但已完成替代核对，并写明原因
- `blocked`：缺少必要条件，当前无法完成，必须写明阻塞项

不允许使用“未处理”“以后再说”“暂时略过”这种没有状态定义的写法。

## 仅保留的确认点

这套流程里只保留以下确认点：

1. 步骤 1 输出 `source-review-findings.md` 后
2. 步骤 3 输出 `wsl-deep-test-findings.md` 后

除此以外，不要频繁打断，不要每个小阶段都要求确认。

只有下面几种情况允许额外暂停：

- 缺少 WSL 环境或关键运行条件
- 缺少必要凭据，但这些凭据只用于本地授权测试
- 当前步骤已经无法继续推进

## 进度文件协议

`docs/安全实施/SECURITY_IMPLEMENTATION_PROGRESS.md` 只用于记录当前做到哪一步，不要把它做成复杂项目管理表。

### 总状态枚举

- `not_started`
- `in_progress`
- `waiting_user_fix`
- `completed`

### 步骤状态枚举

- `pending`
- `in_progress`
- `done`
- `waiting_user_fix`

### 推荐模板

```md
# SECURITY_IMPLEMENTATION_PROGRESS

## 执行信息
- 日期：
- 操作者：
- 当前总状态：not_started
- 当前步骤：1

## 当日审查日记
- 目录：docs/审查日记/YYYY-MM-DD/
- 记录文件：docs/审查日记/YYYY-MM-DD/审查记录.md
- 问题跟踪：docs/审查日记/YYYY-MM-DD/问题跟踪.md

## 步骤状态
| 步骤 | 状态 | 最后更新时间 | 说明 |
|---|---|---|---|
| 1. 本地源码深度审查 | pending |  |  |
| 2. 源码修复后的对照复审 | pending |  |  |
| 3. WSL 部署测试版 + 本地深度逆向/渗透/攻击验证 | pending |  |  |
| 4. WSL 深测问题的对照复审 | pending |  |  |

## 审查测试覆盖状态
| 文档 | 状态 | 最后更新时间 | 说明 |
|---|---|---|---|
| 00_README_EXECUTION_ORDER.md | blocked |  |  |
| 01_SCOPE_AND_RULES.md | blocked |  |  |
| 02_ENV_AND_WSL_ISOLATED_LAB.md | blocked |  |  |
| 03_CODEX_CLI_MASTER_PROMPTS.md | blocked |  |  |
| 04_ATTACK_SURFACE_MAP.md | blocked |  |  |
| 05_LOCAL_SOURCE_REVIEW_BACKEND_AUTH_APIKEY.md | blocked |  |  |
| 06_LOCAL_SOURCE_REVIEW_AI_GATEWAY_BILLING_UPSTREAM.md | blocked |  |  |
| 07_LOCAL_SOURCE_REVIEW_FRONTEND_DEPLOY_DATAMANAGEMENT.md | blocked |  |  |
| 08_LOCAL_SCANNERS_AND_REPORT_IMPORT.md | blocked |  |  |
| 09_REMOTE_TEST_LEVELS_AND_PRODUCTION_BASELINE.md | blocked |  |  |
| 10_REMOTE_ASSET_ENUM_AND_STATIC_REVERSE.md | blocked |  |  |
| 11_REMOTE_AUTH_MATRIX_AND_APIKEY_LIFECYCLE.md | blocked |  |  |
| 12_REMOTE_AI_GATEWAY_BUSINESS_LOGIC.md | blocked |  |  |
| 13_REMOTE_UPSTREAM_SSRF_AND_EGRESS.md | blocked |  |  |
| 14_REMOTE_NUCLEI_ZAP_API_SCANNING.md | blocked |  |  |
| 15_DEPLOYMENT_HARDENING_NGINX_DOCKER_VPS.md | blocked |  |  |
| 16_LOGGING_AUDIT_FORENSICS.md | blocked |  |  |
| 17_BACKUP_UPGRADE_DATAMANAGEMENTD.md | blocked |  |  |
| 18_REVERSE_MAPPING_SOURCE_CONFIG.md | blocked |  |  |
| 19_FINAL_REPORT_REMEDIATION_RETEST.md | blocked |  |  |
| 20_STANDARDS_MAPPING_CHECKLIST.md | blocked |  |  |

## 当前问题单
- source review: 未生成
- source recheck: 未生成
- wsl deep test: 未生成
- wsl recheck: 未生成

## 当前阻塞项
- 无

## 下一步唯一动作
- 开始步骤 1：本地源码深度审查
```

更新规则：

- 文件不存在则先创建
- 每做完一步更新一次
- 如果在等你修复，就把当前总状态写成 `waiting_user_fix`
- 新会话恢复时，只需看“当前步骤”和“下一步唯一动作”
- 同时必须更新 `00-20` 每一项的覆盖状态
- 同时必须更新当天 `docs/审查日记/YYYY-MM-DD/审查记录.md`
- 同时必须更新当天 `docs/审查日记/YYYY-MM-DD/问题跟踪.md`

## 恢复执行方法

新会话恢复时，AI 只做以下事情：

1. 读取 `docs/安全实施/README.md`
2. 读取 `docs/安全实施/SECURITY_IMPLEMENTATION_PROGRESS.md`
3. 根据“当前步骤”继续
4. 如果状态是 `waiting_user_fix`，则先检查你是否已经修复，再执行对应复审步骤

不要重新设计流程，不要从头走 12 个阶段。
但必须继续补齐 `00-20` 中尚未标记为 `done` 或 `done_with_note` 的项。

## 唯一主提示词

把下面这段完整交给 AI，即可按这套简化流程执行：

```text
你现在是我的 Sub2API 防御性安全审查代理。

你只按 4 步闭环工作，不要扩展阶段，不要自己拆出更多确认点：

1. 本地源码深度审查
2. 我修复后的源码对照复审
3. WSL 部署测试版 + 本地深度逆向/渗透/攻击验证
4. 我修复后的 WSL 问题对照复审

执行前必须先读取：
1. docs/安全实施/README.md
2. docs/安全实施/SECURITY_IMPLEMENTATION_PROGRESS.md
3. docs/security_audit_20260324.md
4. docs/security_audit_20260508.md
5. 当前仓库中的 README.md、README_CN.md、README_EN.md、backend/、frontend/、deploy/、tools/
6. docs/审查测试/00_README_EXECUTION_ORDER.md 到 20_STANDARDS_MAPPING_CHECKLIST.md

如果 docs/安全实施/SECURITY_IMPLEMENTATION_PROGRESS.md 不存在：
- 先按 README.md 中的模板创建它
- 当前总状态写为 in_progress
- 当前步骤写为 1
- 将步骤 1 标记为 in_progress

开始写入任何审查产物前，必须先处理审查日记：
- 根据当天本地日期创建或复用 `docs/审查日记/YYYY-MM-DD/`
- 在该目录中创建或更新 `审查记录.md`
- 在该目录中创建或更新 `问题跟踪.md`
- 在 `审查记录.md` 中写明今天日期、当前步骤、今天已覆盖的审查测试项、今天新生成的文件、当前阻塞项、下一步动作
- 在 `问题跟踪.md` 中用一行一条的方式记录当天发现或复审的问题，至少写明：问题编号、是否解决、是否本地压测

执行原则：
1. 默认这是一套“查问题和复审问题”的流程，不是自动修复流程。
2. 没有我的明确要求时，不要直接修改仓库代码。
3. 做源码深审时，输出 security/findings/source-review-findings.md。
4. 做源码复审时，输出 security/findings/source-review-recheck.md。
5. 做 WSL 深测时，输出 security/findings/wsl-deep-test-findings.md。
6. 做 WSL 复审时，输出 security/findings/wsl-deep-test-recheck.md。
7. 每个问题都必须定位到具体文件、函数、路由、配置、测试场景或运行链路。
8. 不把工具命中直接当漏洞，必须判断真实可达性。
9. 不输出完整密钥、Token、Cookie、密码。
10. docs/审查测试/00-20 每一项都必须执行并写入覆盖状态，不允许跳过。
11. 如果某一项在当前闭环边界下不能做主动动作，也必须写成 done_with_note，并说明替代核对方式。

暂停规则：
1. 步骤 1 完成并输出 source-review-findings.md 后，停止并等待我修复。
2. 步骤 3 完成并输出 wsl-deep-test-findings.md 后，停止并等待我修复。
3. 除非缺少关键运行条件，否则不要因为小问题频繁暂停。

进度规则：
1. 每做完一步就更新 SECURITY_IMPLEMENTATION_PROGRESS.md。
2. 如果当前在等我修复，把当前总状态写成 waiting_user_fix。
3. 新会话恢复时，只根据“当前步骤”和“下一步唯一动作”继续。
4. 同时更新 docs/审查测试/00-20 的覆盖状态表。
5. 同时更新当天 `docs/审查日记/YYYY-MM-DD/审查记录.md`。
6. 同时更新当天 `docs/审查日记/YYYY-MM-DD/问题跟踪.md`。

步骤目标：
- 步骤 1：本地源码深度审查，输出问题单，不修代码。
- 步骤 2：对照 source-review-findings.md 复审我修复后的代码。
- 步骤 3：在本地 WSL 隔离环境部署测试版，做尽可能深入的本地逆向、渗透、攻击验证，输出问题单。
- 步骤 4：对照 wsl-deep-test-findings.md 复审我修复后的代码和运行态问题。

现在开始执行：
- 先读取进度文件
- 如果不存在则创建
- 然后从当前步骤继续
```

## 目录分工

- `docs/安全实施/README.md`：总控入口和唯一主提示词
- `docs/安全实施/SECURITY_IMPLEMENTATION_PROGRESS.md`：轻量进度文件
- `docs/审查日记/YYYY-MM-DD/审查记录.md`：按日期记录当天执行过程
- `docs/审查日记/YYYY-MM-DD/问题跟踪.md`：按日期记录问题是否解决、是否本地压测
- `docs/security_audit_20260324.md`、`docs/security_audit_20260508.md`：已有问题输入源
- `docs/审查测试/`：必须全部执行并留痕的强制流程库
- `security/findings/*.md`：问题单与复审结果
- `security/security-audit-report.md`：可选的最终汇总，不是默认必做
