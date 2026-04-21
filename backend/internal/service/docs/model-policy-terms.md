# Model Policy Terms

## Display Model

- `display model` 指账号对外公开展示、对外调用和公开 `/models` / `models detail` 接口里出现的模型 ID。
- 当账号配置了映射名时，映射名就是唯一公开 display ID。

## Target Model

- `target model` 指内部路由到下游时使用的模型 ID。
- `target model` 仅用于内部转发、日志诊断和后台调试，不应作为额外公开可调用 ID 暴露给外部调用方。

## Default Library

- `default library` 指平台本地模型库 / 注册表生成的默认模型集合。
- 当账号没有显式白名单或映射时，系统只从 default library 生成账号模型投影。

## Policy Projection

- `policy projection` 指根据账号模型策略解析出的标准化结果集合。
- 它统一驱动后台 available models、测试模型列表、公开 `/models`、运行时支持判断和模型选择。

## Availability Snapshot

- `availability snapshot` 指保存在 `extra.model_probe_snapshot` 中、按 entry 记录的本地可用性状态快照。
- 它只能补充 `verified / unavailable / unknown` 与 `fresh / stale / unverified` 等状态，不能扩展 policy projection 的模型集合。
