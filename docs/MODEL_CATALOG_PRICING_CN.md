# 模型库定价维护说明

本文档说明模型库的三层价格结构、管理员维护方式，以及“官方价审计”的推荐流程。

## 1. 定价分层

- `upstream_pricing`：LiteLLM 上游快照 / 本地 fallback 价格基线
- `official_pricing`：`upstream_pricing + model_official_price_overrides`
- `sale_pricing`：`official_pricing + model_price_overrides`
- 运行时计费读取 sale 层的生效值；若某字段没有售价覆盖，则逐字段回落到官方生效价
- 公共模型目录、分组目录、后台价格页也统一改为读取这套“有效 sale 价格”口径，不再直接以 `sale_form` 是否为空判断“有无价格”

## 2. 管理员接口

- `GET /api/v1/admin/models`
- `GET /api/v1/admin/models/detail`
- `GET /api/v1/admin/models/exchange-rate`
- `PUT /api/v1/admin/models/official-pricing-override`
- `DELETE /api/v1/admin/models/official-pricing-override`
- `PUT /api/v1/admin/models/pricing-override`
- `DELETE /api/v1/admin/models/pricing-override`

其中：

- `official-pricing-override` 维护真实价格补丁
- `pricing-override` 维护出售价格补丁
- 前端模型库页面显示 `display_name`，点击名称复制的仍然是 raw model id

## 3. 官方价审计原则

- 审计范围优先覆盖缺价模型，其次复核 `OpenAI / Anthropic / Gemini / DeepSeek / Qwen / Moonshot` 等高频供应商
- 官方价格页优先；官方缺失时可以使用 LiteLLM / OpenRouter 等可信聚合源，但必须标记 `source_type=aggregator`
- 以上游快照作为基线，仅对“确有公开依据”的字段写官方覆盖层
- 没有公开阶梯价时，不主动制造 `*_threshold` / `*_above_threshold`
- 源价币种必须保留：官方 CNY 价格以 CNY 存储、展示和计费，不再折算成 USD 保存
- USD 兼容字段仍保留，但只代表 USD 钱包 / USD 等值视图；跨币种汇总必须使用 `cost_by_currency`

## 4. 混合币种计费口径

- `currency` 是价格源币种，也是运行时扣费币种；未设置时默认 `USD`
- CNY 模型的 `ActualCost` 按 CNY 计算，并写入 `usage_logs.billing_currency=CNY`
- `usage_logs.cost_by_currency` / `actual_cost_by_currency` 承载多币种分列；旧 `cost` / `actual_cost` 继续作为 USD 兼容字段
- 用户余额以 `billing_wallets(user_id,currency,balance)` 为准，`users.balance` 只作为 USD 影子兼容字段
- CNY 钱包不足时，按价格保存时锁定的 USD/CNY 汇率从 USD 钱包自动换入刚好覆盖缺口的 CNY，再扣 CNY
- 自动换汇账本写三条审计记录：USD 换出、CNY 换入、CNY 消费；日志只记录 requestId、model、currency、amount、fx_rate 等非敏感字段

## 5. Patch 文件与批量维护

- `billing_pricing_patch_*.json` 默认可导出为两种模式：
  - `issue_worklist`：问题工作清单，默认 `patch` 为空，供人工补录
  - `executable_template`：可执行模板，直接带完整字段骨架，可填值或填 `null` 清空
- V1 patch 现已支持：
  - 基础字段：`input_price`、`output_price`、`cache_price`
  - 特殊字段：`special_enabled`、`special.*`
  - 阶梯字段：`tiered_enabled`、`tier_threshold_tokens`、`input_price_above_threshold`、`output_price_above_threshold`
  - 倍率字段：`multiplier_enabled`、`multiplier_mode`、`shared_multiplier`、`item_multipliers`
  - `null` 清空可选字段或删除某个 item multiplier
- 导入时按模型逐条校验；单模型失败不会中断整个批次。
- 管理端保存接口 `PUT /admin/billing/pricing/models/:model/layers/:layer` 保持 `400 + reason=BILLING_PRICE_INVALID` 不变，但现在会额外返回扁平 `metadata.field_errors.*`，用于把错误精确回填到 `tier_threshold_tokens`、`input_price_above_threshold`、`output_price_above_threshold`、`shared_multiplier`、`item_multipliers.<field_id>` 等字段。

## 6. 价格补全与未确认清单

- `billing_pricing_patch_20260427_122653.json` 用于批量导入缺价模型补丁
- `docs/model-pricing/MODEL_PRICING_UNRESOLVED_*.md` 记录无法公开确认的模型和原因，等待人工决定是否删除或继续查价
- 仓库新增离线产物脚本：`tools/build_billing_pricing_artifacts.py`
  - 输入：当前 `billing_pricing_patch_*.json` 问题工作清单
  - 输出 1：`billing_pricing_patch_confirmed_*.json`，只复用 `current.official/current.sale` 中已经存在的可确认字段，不猜未知价格
  - 输出 2：`docs/model-pricing/MODEL_PRICING_UNRESOLVED_*.md`，收集“当前官方/售价都无可确认字段且 patch 仍为空”的模型，并附带 `pricing_status`、`pricing_warnings`、`currency`
- `backend/resources/model-pricing/model_prices_and_context_window.json` 是运行时 fallback 基线；新增 CNY 价格必须保留 `currency: "CNY"`
- 每个补价模型应尽量带上 `source_url`、`source_type`、`checked_at`
- 官方 CNY 来源不得只因为后台需要 USD 视图而改写成 USD；参考等值和自动换汇使用锁定汇率另算

## 7. CNY 保存与运行时锁汇

- 后台现在允许先保存 `CNY` 价格，即使当前没有可用 USD/CNY 汇率也不会阻止保存。
- 这类价格会先进入“待锁汇”状态，后台继续提示 warning，但不再因为保存阶段的硬校验导致无法录价。
- 运行时首次真正使用到该 CNY 价格时，系统会尝试：
  1. 拉取当前 USD/CNY
  2. 用该汇率完成本次计费
  3. 回写 `model_pricing_currencies` 和对应价格元数据，形成锁汇
- 如果运行时也拿不到汇率，会返回明确错误 `BILLING_FX_RATE_UNAVAILABLE`，而不是静默错误扣费。

## 8. 官方价审计脚本

仓库提供离线脚本：`tools/model_catalog_official_price_audit.py`

作用：

- 读取本地 LiteLLM 价格快照
- 可选叠加当前官方价覆盖 JSON
- 生成一份 CSV 审计工作表，供人工对照官方文档复核

### 6.1 输入

- 基线快照：`backend/resources/model-pricing/model_prices_and_context_window.json`
- 可选覆盖层：导出的 `model_official_price_overrides` JSON 文件

说明：

- 若当前官方覆盖保存在数据库 / settings 存储中，请先把该 key 的 JSON 值导出为文件
- 脚本不会直接写库，也不会改动运行时配置

### 6.2 运行示例

只看基线：

```bash
python tools/model_catalog_official_price_audit.py \
  --output docs/model_catalog_official_pricing_audit.csv
```

叠加当前官方覆盖层一起审计：

```bash
python tools/model_catalog_official_price_audit.py \
  --overrides-json tmp/model_official_price_overrides.json \
  --output docs/model_catalog_official_pricing_audit.csv
```

只审某些家族：

```bash
python tools/model_catalog_official_price_audit.py \
  --families openai anthropic gemini \
  --output docs/model_catalog_official_pricing_audit.csv
```

### 6.3 输出字段

CSV 会包含以下几类字段：

- 模型标识：`model`、`display_name`、`family`、`provider`、`mode`
- 上游基线：`upstream_*`
- 官方覆盖：`official_override_*`
- 官方生效价：`effective_official_*`
- 人工复核列：`manual_review_status`、`manual_notes`

其中 token 单价按源币种输出；CSV 必须保留 `currency`，人工核对时再按供应商官方单位换算到每 token。

## 9. 推荐维护流程

1. 拉取最新代码，确认本地快照最新
2. 导出当前 `model_official_price_overrides`（如有）
3. 运行审计脚本生成 CSV
4. 对照官方文档逐项确认差异
5. 只把有公开依据的差异写入官方覆盖层；无法确认的模型追加到未确认 MD
6. 在管理员模型库页复核 `官方价 / 售价 / 生效 sale 展示价 / 源币种 / 汇率参考`
7. 对 CNY 模型额外验证：无汇率时也能保存、后台持续 warning、首次运行时自动补锁汇、汇率不可用时返回明确错误
8. 对只配官方价但 sale 为空的模型，验证公开目录和分组目录仍可显示价格
9. 运行相关测试后再提交

## 10. 回归测试命令

后端关键测试：

```bash
cd backend
go test ./internal/service -run "Test(ModelCatalog|CalculateCost_OfficialPricingOverridesSaleFallback)"
go test ./internal/handler/admin -run "TestModelCatalogHandler"
go test ./internal/service ./internal/repository ./internal/handler/dto
```

前端关键测试：

```bash
pnpm --dir frontend exec vitest run \
  src/utils/__tests__/modelCatalogPresentation.spec.ts \
  src/utils/__tests__/modelCatalogPricing.spec.ts \
  src/utils/__tests__/modelCatalogPricingExchange.spec.ts \
  src/components/admin/models/__tests__/ModelCatalogModelLabel.spec.ts \
  src/components/admin/models/__tests__/ModelCatalogPricingEditorSection.spec.ts

pnpm --dir frontend exec vitest run \
  src/components/admin/billing/__tests__/pricingCurrency.spec.ts \
  src/components/admin/billing/__tests__/BillingPricingEditorDialog.spec.ts \
  src/views/admin/billing/__tests__/BillingPricingView.spec.ts
```
