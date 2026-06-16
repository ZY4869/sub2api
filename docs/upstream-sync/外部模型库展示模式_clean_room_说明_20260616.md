# 外部模型库展示模式 Clean-Room 说明

## 背景

本次外部模型库展示模式改造用于支持单用户级页面形态配置：

- 跟随 Key 创建方式
- 分组优先
- 仅模型

## 版权边界

- 本实现仅参考公开交互语义：用户可以先看分组再看模型，或直接看聚合模型列表。
- 未复制 New API 源码、组件、样式、字段结构、数据库结构或实现细节。
- 后端字段、DTO、路由与前端组件均按本项目既有风格自研实现。
- 对外协议字段优先沿用项目现有公开模型目录字段，以及 OpenAI / Anthropic / Gemini 等通用公开语义。

## 推送边界

公开模型目录发布后只允许推送轻量事件：

- `etag`
- `published_at`
- `model_count`
- `changed_count`

前端收到事件后必须重新拉取当前用户视图，不得直接信任推送 payload 构造页面。

事件 payload 不得包含：

- 内部账号 ID
- `target_model_id`
- 真实上游路由
- 密钥、凭据或代理信息
- 任何可反推出内部调度拓扑的字段

## 实现声明

Clean-room implementation based on public API behavior.
