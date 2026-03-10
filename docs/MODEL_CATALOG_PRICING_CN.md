# 模型库定价维护说明

本文档说明模型库的三层价格结构、管理员维护方式，以及“官方价审计”的推荐流程。

## 1. 定价分层

- `upstream_pricing`：LiteLLM 上游快照 / 本地 fallback 价格基线
- `official_pricing`：`upstream_pricing + model_official_price_overrides`
- `sale_pricing`：`official_pricing + model_price_overrides`
- 运行时计费只读取售价层；若某字段没有售价覆盖，则回落到官方生效价

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

- 审计范围默认聚焦 `OpenAI / Anthropic / Gemini`
- 以上游快照作为基线，仅对“确有公开依据”的字段写官方覆盖层
- 没有公开阶梯价时，不主动制造 `*_threshold` / `*_above_threshold`
- 人民币仅用于展示参考，存储和计费仍全部使用 USD

## 4. 官方价审计脚本

仓库提供离线脚本：`tools/model_catalog_official_price_audit.py`

作用：

- 读取本地 LiteLLM 价格快照
- 可选叠加当前官方价覆盖 JSON
- 生成一份 CSV 审计工作表，供人工对照官方文档复核

### 4.1 输入

- 基线快照：`backend/resources/model-pricing/model_prices_and_context_window.json`
- 可选覆盖层：导出的 `model_official_price_overrides` JSON 文件

说明：

- 若当前官方覆盖保存在数据库 / settings 存储中，请先把该 key 的 JSON 值导出为文件
- 脚本不会直接写库，也不会改动运行时配置

### 4.2 运行示例

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

### 4.3 输出字段

CSV 会包含以下几类字段：

- 模型标识：`model`、`display_name`、`family`、`provider`、`mode`
- 上游基线：`upstream_*`
- 官方覆盖：`official_override_*`
- 官方生效价：`effective_official_*`
- 人工复核列：`manual_review_status`、`manual_notes`

其中 token 单价统一转换成“每百万 token 的 USD 值”输出，便于人工核对。

## 5. 推荐维护流程

1. 拉取最新代码，确认本地快照最新
2. 导出当前 `model_official_price_overrides`（如有）
3. 运行审计脚本生成 CSV
4. 对照官方文档逐项确认差异
5. 只把有公开依据的差异写入官方覆盖层
6. 在管理员模型库页复核 `官方价 / 售价 / 汇率展示`
7. 运行相关测试后再提交

## 6. 回归测试命令

后端关键测试：

```bash
cd backend
go test ./internal/service -run "Test(ModelCatalog|CalculateCost_OfficialPricingOverridesSaleFallback)"
go test ./internal/handler/admin -run "TestModelCatalogHandler"
```

前端关键测试：

```bash
pnpm --dir frontend exec vitest run \
  src/utils/__tests__/modelCatalogPresentation.spec.ts \
  src/utils/__tests__/modelCatalogPricing.spec.ts \
  src/utils/__tests__/modelCatalogPricingExchange.spec.ts \
  src/components/admin/models/__tests__/ModelCatalogModelLabel.spec.ts \
  src/components/admin/models/__tests__/ModelCatalogPricingEditorSection.spec.ts
```
