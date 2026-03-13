Ready for review
Select text to add comments on the plan
模型ID管理系统彻底重构计划
Context（背景与问题）
当前系统的模型ID管理存在严重的架构混乱问题，导致维护困难、易出错、性能低下。核心问题包括：

问题诊断
多层ID概念混乱：系统中存在至少5种不同的模型ID概念

短名（Alias）：claude-sonnet-4.5 - 用户界面使用
完整ID（Canonical）：claude-sonnet-4-5-20250929 - 上游API使用
定价查询ID：用于计费系统
协议ID（Protocol）：各平台特定格式
显示名称：UI展示
重复定义与数据源分散：

claude/constants.go 中的 ModelIDOverrides 硬编码映射
model_catalog_seed.json 中的 canonical_model_id 配置
modelregistry/registry_seed.json 中的 protocol_ids 数组
modelregistry/overlays.go 中的 modelCatalogCanonicalDefaults 映射
同一个映射关系在4个地方维护，极易不同步！
转换函数散落各处：至少6个不同的规范化函数

claude.NormalizeModelID() - Claude短名→完整名
claude.DenormalizeModelID() - Claude完整名→短名
CanonicalizeModelNameForPricing() - 定价系统规范化
NormalizeModelCatalogModelID() - 目录系统规范化
normalizeModelCatalogAlias() - 别名规范化
normalizePlatform() - 平台名规范化
复杂的查询逻辑：resolveModelCatalogRecord 函数需要：

先查canonical → 再查normalized → 遍历所有记录匹配
比较优先级、日期后缀、字符串长度...
典型的"补丁式修复"堆积
带日期模型ID的实际作用：

关键发现：claude-sonnet-4-5-20250929 这种带日期的ID 仅在调用Anthropic OAuth API时强制要求
在 gateway_claude_normalize.go:184 中，只有 account.Platform == PlatformAnthropic && account.Type != AccountTypeAPIKey 时才调用 claude.NormalizeModelID() 转换
API Key账号、OpenAI平台、Gemini平台都不需要带日期ID
数据库存储（usage_logs.model）、前端展示、计费系统都使用短名
为什么要彻底重构
计费风险：模型ID匹配失败导致定价查询错误
维护困难：新增模型需要修改5+处代码
性能问题：复杂的查询逻辑增加延迟
测试覆盖不足：转换逻辑分散，难以全面测试
代码臃肿：6638行的 registry_seed.json + 多处重复映射
重构目标
统一数据源：所有模型ID映射关系集中在 modelregistry/registry_seed.json
简化ID层级：从5层简化为3层（Display / Canonical / Protocol）
边界转换：转换只在系统边界发生（API入口/出口），内部统一使用Canonical ID
废弃冗余：删除 claude/constants.go 中的硬编码映射
性能优化：简化查询逻辑，建立内存索引
重构方案
阶段一：统一模型注册表（ModelRegistry）为唯一数据源
1.1 增强 ModelRegistry 数据结构
文件：backend/internal/modelregistry/types.go

type ModelEntry struct {
    ID               string   `json:"id"`                // Canonical ID（系统内部统一使用）
    DisplayName      string   `json:"display_name"`      // UI展示名称
    Provider         string   `json:"provider"`          // anthropic/openai/gemini
    Platforms        []string `json:"platforms"`         // 支持的平台列表
    ProtocolIDs      []string `json:"protocol_ids"`      // 上游API协议ID（第一个为主ID）
    Aliases          []string `json:"aliases"`           // 用户可输入的别名
    PricingLookupIDs []string `json:"pricing_lookup_ids"` // 定价查询ID
    // ... 其他字段保持不变
}
关键设计：

ID：系统内部统一使用的Canonical ID（如 claude-sonnet-4.5）
ProtocolIDs[0]：调用上游API时使用的协议ID（如 claude-sonnet-4-5-20250929）
Aliases：用户可输入的所有别名（包括带日期的、带点号的等）
1.2 迁移现有映射到 registry_seed.json
示例（claude-sonnet-4.5）：

{
  "id": "claude-sonnet-4.5",
  "display_name": "Claude Sonnet 4.5",
  "provider": "anthropic",
  "platforms": ["anthropic", "antigravity"],
  "protocol_ids": [
    "claude-sonnet-4-5-20250929",
    "claude-sonnet-4.5"
  ],
  "aliases": [
    "claude-sonnet-4-5",
    "claude-sonnet-4-5-20250929",
    "claude-sonnet-4.5-20250929"
  ],
  "pricing_lookup_ids": ["claude-sonnet-4-5-20250929"]
}
迁移清单：

从 claude/constants.go 的 ModelIDOverrides 迁移
从 model_catalog_seed.json 的 canonical_model_id 迁移
从 modelregistry/overlays.go 的 modelCatalogCanonicalDefaults 迁移
1.3 建立快速查询索引
文件：backend/internal/modelregistry/loader.go

var (
    seedModels       []ModelEntry
    seedModelMap     map[string]ModelEntry  // ID -> Entry
    aliasToIDMap     map[string]string      // Alias -> Canonical ID
    protocolToIDMap  map[string]string      // Protocol ID -> Canonical ID
)

func init() {
    // 加载种子数据
    json.Unmarshal(registrySeedJSON, &seedModels)

    // 建立索引
    seedModelMap = make(map[string]ModelEntry)
    aliasToIDMap = make(map[string]string)
    protocolToIDMap = make(map[string]string)

    for _, entry := range seedModels {
        seedModelMap[entry.ID] = entry

        // 索引所有别名
        for _, alias := range entry.Aliases {
            aliasToIDMap[normalize(alias)] = entry.ID
        }

        // 索引所有协议ID
        for _, protocolID := range entry.ProtocolIDs {
            protocolToIDMap[normalize(protocolID)] = entry.ID
        }
    }
}

// ResolveToCanonicalID 将任意输入解析为Canonical ID
func ResolveToCanonicalID(input string) (string, bool) {
    normalized := normalize(input)

    // 1. 直接匹配Canonical ID
    if _, ok := seedModelMap[normalized]; ok {
        return normalized, true
    }

    // 2. 匹配别名
    if canonicalID, ok := aliasToIDMap[normalized]; ok {
        return canonicalID, true
    }

    // 3. 匹配协议ID
    if canonicalID, ok := protocolToIDMap[normalized]; ok {
        return canonicalID, true
    }

    return "", false
}

// GetProtocolID 获取上游API协议ID
func GetProtocolID(canonicalID string, platform string) (string, bool) {
    entry, ok := seedModelMap[canonicalID]
    if !ok || len(entry.ProtocolIDs) == 0 {
        return "", false
    }

    // 对于Anthropic OAuth，返回带日期的完整ID
    if platform == "anthropic" && len(entry.ProtocolIDs) > 0 {
        return entry.ProtocolIDs[0], true
    }

    // 其他情况返回短名（通常是ProtocolIDs的第二个或Canonical ID）
    if len(entry.ProtocolIDs) > 1 {
        return entry.ProtocolIDs[1], true
    }
    return entry.ID, true
}
阶段二：简化网关层模型ID转换逻辑
2.1 统一入口转换
文件：backend/internal/service/gateway_request.go

在 ParseGatewayRequest 中统一处理：

func ParseGatewayRequest(body []byte, protocol string) (*ParsedRequest, error) {
    // ... 现有解析逻辑 ...

    // 统一模型ID解析
    if modelResult.Exists() {
        rawModel := modelResult.String()
        canonicalID, ok := modelregistry.ResolveToCanonicalID(rawModel)
        if ok {
            parsed.Model = canonicalID  // 内部统一使用Canonical ID
            parsed.RawModel = rawModel   // 保留原始输入用于日志
        } else {
            parsed.Model = rawModel      // 未识别的模型保持原样
        }
    }

    return parsed, nil
}
2.2 简化上游请求构建
文件：backend/internal/service/gateway_claude_normalize.go

删除 normalizeClaudeOAuthRequestBody 中的模型ID转换逻辑（183-189行）：

// 删除这段代码：
if rawModel, ok := req["model"].(string); ok {
    normalized := claude.NormalizeModelID(rawModel)
    if normalized != rawModel {
        req["model"] = normalized
        modelID = normalized
        modified = true
    }
}
在 buildUpstreamRequest 中统一处理：

func (s *GatewayService) buildUpstreamRequest(..., canonicalModelID string, ...) (*http.Request, error) {
    // 根据平台和账号类型获取协议ID
    protocolID := canonicalModelID
    if account.Platform == PlatformAnthropic && account.Type != AccountTypeAPIKey {
        if pid, ok := modelregistry.GetProtocolID(canonicalModelID, "anthropic"); ok {
            protocolID = pid
        }
    }

    // 替换请求体中的model字段
    if protocolID != canonicalModelID {
        body = s.replaceModelInBody(body, protocolID)
    }

    // ... 构建请求 ...
}
2.3 简化账号选择逻辑
文件：backend/internal/service/gateway_account_selection.go

删除1737行的特殊处理：

// 删除这段代码：
if account.Platform == PlatformAnthropic && account.Type != AccountTypeAPIKey {
    requestedModel = claude.NormalizeModelID(requestedModel)
}
改为统一使用Canonical ID匹配：

func (s *GatewayService) isModelSupportedByAccount(account *Account, requestedModel string) bool {
    // requestedModel 已经在入口处转换为Canonical ID
    return account.IsModelSupported(requestedModel)
}
阶段三：废弃冗余代码
3.1 删除 claude/constants.go 中的映射
文件：backend/internal/pkg/claude/constants.go

删除以下内容（102-136行）：

// 删除：
var ModelIDOverrides = map[string]string{...}
var ModelIDReverseOverrides = map[string]string{...}
func NormalizeModelID(id string) string {...}
func DenormalizeModelID(id string) string {...}
保留：

Beta header 常量
DefaultHeaders
DefaultModels（从 modelregistry 生成）
3.2 删除 model_catalog_seed.json
文件：backend/internal/service/model_catalog_seed.json

删除整个文件，改为从 modelregistry 动态生成：

// backend/internal/service/model_catalog_entries.go
func loadSeedModelCatalogEntries() []ModelCatalogEntry {
    registryModels := modelregistry.SeedModels()
    entries := make([]ModelCatalogEntry, 0, len(registryModels))

    for _, model := range registryModels {
        entries = append(entries, ModelCatalogEntry{
            Model:                model.ID,
            DisplayName:          model.DisplayName,
            Provider:             model.Provider,
            Mode:                 inferModelMode(model.ID, ""),
            CanonicalModelID:     model.ID,
            PricingLookupModelID: firstString(model.PricingLookupIDs...),
        })
    }

    return entries
}
3.3 删除 modelregistry/overlays.go 中的重复映射
文件：backend/internal/modelregistry/overlays.go

删除以下内容（56-120行）：

// 删除：
var modelCatalogExplicitAliases = map[string]string{...}
var modelCatalogCanonicalDefaults = map[string]string{...}
保留：

defaultAntigravityModelMapping（这是业务逻辑，不是ID映射）
阶段四：简化定价和计费系统
4.1 统一定价查询
文件：backend/internal/service/billing_service.go

简化定价查询逻辑（388-425行）：

func (s *BillingService) getModelPricing(model string) *ModelPricing {
    // 1. 解析为Canonical ID
    canonicalID, ok := modelregistry.ResolveToCanonicalID(model)
    if !ok {
        canonicalID = CanonicalizeModelNameForPricing(model)
    }

    // 2. 获取定价查询ID
    pricingID := canonicalID
    if entry, ok := modelregistry.SeedModelByID(canonicalID); ok {
        if len(entry.PricingLookupIDs) > 0 {
            pricingID = entry.PricingLookupIDs[0]
        }
    }

    // 3. 查询定价
    if pricing, ok := s.modelPrices[pricingID]; ok {
        return pricing
    }

    // 4. Fallback：去掉日期后缀再查
    baseName := s.pricingService.extractBaseName(pricingID)
    if pricing, ok := s.modelPrices[baseName]; ok {
        return pricing
    }

    return nil
}
4.2 简化 model_catalog_identity.go
文件：backend/internal/service/model_catalog_identity.go

大幅简化 resolveModelCatalogRecord（5-27行）：

func resolveModelCatalogRecord(records map[string]*modelCatalogRecord, model string) (*modelCatalogRecord, bool) {
    // 1. 解析为Canonical ID
    canonicalID, ok := modelregistry.ResolveToCanonicalID(model)
    if !ok {
        canonicalID = CanonicalizeModelNameForPricing(model)
    }

    // 2. 直接查询
    if record, ok := records[canonicalID]; ok {
        return record, true
    }

    // 3. Fallback：去掉日期后缀再查
    normalized := NormalizeModelCatalogModelID(canonicalID)
    if record, ok := records[normalized]; ok {
        return record, true
    }

    return nil, false
}

// 删除 preferModelCatalogRecord 函数（29-81行）- 不再需要复杂的优先级比较
阶段五：前端适配
5.1 统一前端模型ID类型
文件：frontend/src/types/index.ts

// 统一模型ID字段命名
export interface ModelInfo {
  id: string              // Canonical ID（与后端一致）
  display_name: string    // 显示名称
  provider: string        // 提供商
  platforms: string[]     // 支持的平台
  aliases?: string[]      // 别名（用于搜索）
}

// 使用记录统一使用 model 字段
export interface UsageLog {
  // ... 其他字段 ...
  model: string  // Canonical ID（不再使用 model_id）
}
5.2 更新API调用
文件：frontend/src/api/admin/accounts.ts

确保所有API调用使用统一的字段名：

export interface TestAccountRequest {
  account_id: number
  model: string  // 使用 model 而不是 model_id
  // ...
}
关键文件清单
需要修改的文件
后端核心：

backend/internal/modelregistry/types.go - 增强数据结构
backend/internal/modelregistry/loader.go - 建立索引和查询函数
backend/internal/modelregistry/registry_seed.json - 补充完整映射数据
backend/internal/service/gateway_request.go - 统一入口转换
backend/internal/service/gateway_claude_normalize.go - 简化转换逻辑
backend/internal/service/gateway_account_selection.go - 删除特殊处理
backend/internal/service/gateway_response_stream.go - 更新调用
backend/internal/service/gateway_count_tokens.go - 更新调用
backend/internal/service/billing_service.go - 简化定价查询
backend/internal/service/model_catalog_identity.go - 简化查询逻辑
backend/internal/service/model_catalog_entries.go - 从registry生成
后端删除： 12. backend/internal/pkg/claude/constants.go - 删除映射相关代码（保留其他常量） 13. backend/internal/service/model_catalog_seed.json - 删除整个文件 14. backend/internal/modelregistry/overlays.go - 删除重复映射（保留业务逻辑）

前端： 15. frontend/src/types/index.ts - 统一类型定义 16. frontend/src/api/admin/accounts.ts - 更新API接口 17. frontend/src/components/account/AccountTestModal.vue - 更新字段名 18. frontend/src/components/admin/account/AccountTestModal.vue - 更新字段名

实施步骤
Step 1: 准备阶段（无破坏性）
增强 modelregistry/types.go 数据结构
完善 registry_seed.json 中的映射数据（补充所有 protocol_ids 和 aliases）
在 modelregistry/loader.go 中实现新的索引和查询函数
编写单元测试验证新查询函数的正确性
Step 2: 网关层迁移
修改 gateway_request.go 的入口转换逻辑
修改 gateway_claude_normalize.go 的上游请求构建
修改 gateway_account_selection.go 删除特殊处理
更新 gateway_response_stream.go 和 gateway_count_tokens.go 的调用
运行集成测试验证网关功能
Step 3: 计费系统迁移
修改 billing_service.go 的定价查询逻辑
修改 model_catalog_identity.go 简化查询
修改 model_catalog_entries.go 从registry生成
运行计费相关测试
Step 4: 清理冗余代码
删除 claude/constants.go 中的映射代码
删除 model_catalog_seed.json 文件
删除 modelregistry/overlays.go 中的重复映射
更新所有引用这些代码的地方
Step 5: 前端适配
更新 types/index.ts 类型定义
更新API调用接口
更新组件中的字段引用
前端测试验证
Step 6: 全面测试
单元测试：所有模型ID转换函数
集成测试：网关请求流程（Anthropic OAuth/API Key、OpenAI、Gemini）
端到端测试：完整请求链路（前端→网关→上游API→计费）
回归测试：现有功能不受影响
验证方案
功能验证
模型ID解析验证：

# 测试各种输入格式都能正确解析
curl -X POST /v1/messages -d '{"model":"claude-sonnet-4.5",...}'
curl -X POST /v1/messages -d '{"model":"claude-sonnet-4-5",...}'
curl -X POST /v1/messages -d '{"model":"claude-sonnet-4-5-20250929",...}'
# 验证内部都转换为 claude-sonnet-4.5
上游API调用验证：

# Anthropic OAuth账号：验证发送带日期ID
# 检查日志：Model mapping applied: claude-sonnet-4.5 -> claude-sonnet-4-5-20250929

# Anthropic API Key账号：验证发送短名
# 检查日志：使用 claude-sonnet-4.5

# OpenAI账号：验证不转换
# 检查日志：使用原始model
计费验证：

# 查询usage_logs表
SELECT model, COUNT(*) FROM usage_logs GROUP BY model;
# 验证存储的是Canonical ID（claude-sonnet-4.5）

# 验证定价查询
# 检查日志：使用 claude-sonnet-4-5-20250929 查询定价
前端验证：

模型选择器显示正确的display_name
账号测试功能正常
使用统计显示正确
性能验证
查询性能：

模型ID解析：从O(n)遍历优化到O(1)哈希查询
定价查询：从多次遍历优化到单次查询
目标：模型ID解析 < 1μs
内存占用：

索引内存：约 278个模型 × 3个索引 × 50字节 ≈ 40KB
可接受范围
回归测试
运行现有单元测试套件
运行集成测试（账号测试、网关转发）
手动测试关键流程（创建账号、测试账号、发起请求、查看计费）
风险与缓解
风险1：上游API兼容性
风险：某些上游API可能对模型ID格式有特殊要求
缓解：
保留 ProtocolIDs 数组支持多种格式
分平台测试（Anthropic OAuth/API Key、OpenAI、Gemini）
保留 replaceModelInBody 函数用于运行时替换
风险2：历史数据兼容
风险：数据库中可能存在旧格式的模型ID
缓解：
ResolveToCanonicalID 支持所有历史格式
计费查询保留 Fallback 逻辑
不修改数据库中的历史数据
风险3：前后端不同步
风险：前后端部署时间差导致字段不匹配
缓解：
后端同时支持 model 和 model_id 字段（兼容期）
前端优先使用 model 字段
分阶段部署：后端先部署兼容版本，前端再更新
风险4：第三方集成
风险：外部系统可能依赖特定的模型ID格式
缓解：
API响应保持向后兼容
文档明确说明Canonical ID格式
提供迁移指南
预期收益
维护性提升：

新增模型：只需修改 registry_seed.json 一个文件
代码量减少：删除约500行冗余代码
数据源统一：从4个数据源减少到1个
性能提升：

模型ID解析：从O(n)优化到O(1)
定价查询：减少50%的查询次数
内存占用：增加约40KB（可接受）
可靠性提升：

消除数据不同步风险
简化测试：集中测试一个查询函数
减少人为错误：不再需要手动同步多处配置
可扩展性提升：

支持新平台：只需在 registry_seed.json 添加配置
支持新ID格式：在 Aliases 或 ProtocolIDs 添加即可
支持动态更新：可从数据库或远程配置加载
阶段六：高级功能增强（本次一并实现）
6.1 动态模型注册与热更新
目标：支持运行时动态添加/更新模型，无需重启服务

实现方案
文件：backend/internal/modelregistry/dynamic.go

type DynamicRegistry struct {
    mu              sync.RWMutex
    seedModels      []ModelEntry
    customModels    map[string]ModelEntry  // 自定义模型
    aliasIndex      map[string]string      // 统一索引
    protocolIndex   map[string]string
    version         int64                  // 配置版本号
}

// UpsertModel 添加或更新模型（支持热更新）
func (r *DynamicRegistry) UpsertModel(entry ModelEntry) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    // 验证必填字段
    if entry.ID == "" || entry.Provider == "" {
        return errors.New("id and provider are required")
    }

    // 更新自定义模型
    r.customModels[entry.ID] = entry

    // 重建索引
    r.rebuildIndexes()
    r.version++

    return nil
}

// DeleteModel 删除自定义模型
func (r *DynamicRegistry) DeleteModel(id string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, ok := r.customModels[id]; !ok {
        return errors.New("model not found")
    }

    delete(r.customModels, id)
    r.rebuildIndexes()
    r.version++

    return nil
}

// ResolveToCanonicalID 支持动态模型的查询
func (r *DynamicRegistry) ResolveToCanonicalID(input string) (string, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    normalized := normalize(input)

    // 1. 查询自定义模型
    if entry, ok := r.customModels[normalized]; ok {
        return entry.ID, true
    }

    // 2. 查询索引
    if canonicalID, ok := r.aliasIndex[normalized]; ok {
        return canonicalID, true
    }

    return "", false
}
数据库持久化：

-- backend/migrations/XXX_add_custom_models_table.sql
CREATE TABLE IF NOT EXISTS custom_models (
    id                VARCHAR(100) PRIMARY KEY,
    display_name      VARCHAR(200) NOT NULL,
    provider          VARCHAR(50) NOT NULL,
    platforms         JSONB NOT NULL DEFAULT '[]',
    protocol_ids      JSONB NOT NULL DEFAULT '[]',
    aliases           JSONB NOT NULL DEFAULT '[]',
    pricing_lookup_ids JSONB NOT NULL DEFAULT '[]',
    modalities        JSONB NOT NULL DEFAULT '[]',
    capabilities      JSONB NOT NULL DEFAULT '[]',
    ui_priority       INT NOT NULL DEFAULT 100,
    exposed_in        JSONB NOT NULL DEFAULT '[]',
    status            VARCHAR(20) NOT NULL DEFAULT 'active',  -- active/deprecated
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_custom_models_provider ON custom_models(provider);
CREATE INDEX idx_custom_models_status ON custom_models(status);
服务层接口：

// backend/internal/service/model_registry_service.go
func (s *ModelRegistryService) UpsertCustomModel(ctx context.Context, input UpsertModelRegistryEntryInput) error {
    // 1. 验证输入
    if err := validateModelEntry(input); err != nil {
        return err
    }

    // 2. 保存到数据库
    if err := s.customModelRepo.Upsert(ctx, input); err != nil {
        return err
    }

    // 3. 热更新内存注册表
    entry := toModelEntry(input)
    if err := s.dynamicRegistry.UpsertModel(entry); err != nil {
        return err
    }

    // 4. 通知其他服务刷新缓存
    s.notifyModelRegistryUpdate()

    return nil
}
6.2 模型版本管理与自动迁移
目标：标记模型状态（stable/beta/deprecated），自动迁移废弃模型

数据结构增强
文件：backend/internal/modelregistry/types.go

type ModelEntry struct {
    ID               string   `json:"id"`
    DisplayName      string   `json:"display_name"`
    Provider         string   `json:"provider"`
    Platforms        []string `json:"platforms"`
    ProtocolIDs      []string `json:"protocol_ids"`
    Aliases          []string `json:"aliases"`
    PricingLookupIDs []string `json:"pricing_lookup_ids"`
    Modalities       []string `json:"modalities"`
    Capabilities     []string `json:"capabilities"`
    UIPriority       int      `json:"ui_priority"`
    ExposedIn        []string `json:"exposed_in"`

    // 新增：版本管理字段
    Status           string   `json:"status"`            // stable/beta/deprecated
    DeprecatedAt     string   `json:"deprecated_at"`     // 废弃时间
    ReplacedBy       string   `json:"replaced_by"`       // 替代模型ID
    DeprecationNotice string  `json:"deprecation_notice"` // 废弃说明
}
自动迁移逻辑
文件：backend/internal/service/gateway_request.go

func ParseGatewayRequest(body []byte, protocol string) (*ParsedRequest, error) {
    // ... 现有解析逻辑 ...

    // 模型ID解析与自动迁移
    if modelResult.Exists() {
        rawModel := modelResult.String()
        canonicalID, ok := modelregistry.ResolveToCanonicalID(rawModel)
        if ok {
            // 检查模型状态
            if entry, ok := modelregistry.GetModelEntry(canonicalID); ok {
                switch entry.Status {
                case "deprecated":
                    // 自动迁移到替代模型
                    if entry.ReplacedBy != "" {
                        logger.Warn("Model deprecated, auto-migrating",
                            zap.String("from", canonicalID),
                            zap.String("to", entry.ReplacedBy),
                            zap.String("notice", entry.DeprecationNotice))
                        canonicalID = entry.ReplacedBy
                    }
                case "beta":
                    // 记录beta模型使用
                    logger.Info("Using beta model", zap.String("model", canonicalID))
                }
            }

            parsed.Model = canonicalID
            parsed.RawModel = rawModel
        } else {
            parsed.Model = rawModel
        }
    }

    return parsed, nil
}
前端废弃提示
文件：frontend/src/components/account/ModelSelector.vue

<template>
  <div class="model-option">
    <span>{{ model.display_name }}</span>
    <span v-if="model.status === 'beta'" class="badge badge-warning">Beta</span>
    <span v-if="model.status === 'deprecated'" class="badge badge-danger">
      Deprecated
      <span v-if="model.replaced_by"> → {{ model.replaced_by }}</span>
    </span>
  </div>
</template>
6.3 监控与告警系统
目标：实时监控模型ID使用情况，及时发现异常

指标收集
文件：backend/internal/service/gateway_metrics.go

type ModelMetrics struct {
    UnrecognizedModels  *prometheus.CounterVec   // 未识别的模型ID
    ModelResolutions    *prometheus.CounterVec   // 模型解析成功/失败
    DeprecatedUsage     *prometheus.CounterVec   // 废弃模型使用次数
    ProtocolConversions *prometheus.CounterVec   // 协议ID转换次数
}

func (m *ModelMetrics) RecordUnrecognizedModel(rawModel string, source string) {
    m.UnrecognizedModels.WithLabelValues(rawModel, source).Inc()

    // 记录到日志用于分析
    logger.Warn("Unrecognized model ID",
        zap.String("raw_model", rawModel),
        zap.String("source", source))
}

func (m *ModelMetrics) RecordDeprecatedUsage(model string, replacedBy string) {
    m.DeprecatedUsage.WithLabelValues(model, replacedBy).Inc()
}
告警规则
文件：deploy/prometheus/alerts.yml

groups:
  - name: model_registry
    interval: 1m
    rules:
      # 未识别模型ID告警
      - alert: UnrecognizedModelIDHigh
        expr: rate(model_unrecognized_total[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High rate of unrecognized model IDs"
          description: "{{ $value }} unrecognized model IDs per second"

      # 废弃模型使用告警
      - alert: DeprecatedModelUsage
        expr: sum(rate(model_deprecated_usage_total[1h])) by (model) > 100
        for: 1h
        labels:
          severity: info
        annotations:
          summary: "Deprecated model {{ $labels.model }} still in use"
          description: "Consider migrating to {{ $labels.replaced_by }}"

      # 模型解析失败率告警
      - alert: ModelResolutionFailureHigh
        expr: rate(model_resolution_failures_total[5m]) / rate(model_resolution_total[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Model resolution failure rate > 5%"
管理后台统计
文件：backend/internal/handler/admin/model_analytics_handler.go

type ModelAnalyticsResponse struct {
    TopModels           []ModelUsageStat      `json:"top_models"`
    UnrecognizedModels  []UnrecognizedStat    `json:"unrecognized_models"`
    DeprecatedUsage     []DeprecatedUsageStat `json:"deprecated_usage"`
    ResolutionStats     ResolutionStats       `json:"resolution_stats"`
}

type ModelUsageStat struct {
    Model       string `json:"model"`
    DisplayName string `json:"display_name"`
    Count       int64  `json:"count"`
    Percentage  float64 `json:"percentage"`
}

type UnrecognizedStat struct {
    RawModel    string `json:"raw_model"`
    Count       int64  `json:"count"`
    FirstSeen   string `json:"first_seen"`
    LastSeen    string `json:"last_seen"`
}

func (h *ModelAnalyticsHandler) GetAnalytics(c *gin.Context) {
    // 查询最近7天的统计数据
    stats, err := h.analyticsService.GetModelAnalytics(c.Request.Context(), 7)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, stats)
}
6.4 自动文档生成
目标：从 registry_seed.json 自动生成API文档和兼容性矩阵

文档生成工具
文件：backend/cmd/gendocs/main.go

func main() {
    models := modelregistry.SeedModels()

    // 1. 生成Markdown文档
    generateMarkdownDocs(models, "docs/models.md")

    // 2. 生成OpenAPI规范
    generateOpenAPISpec(models, "docs/openapi.yaml")

    // 3. 生成兼容性矩阵
    generateCompatibilityMatrix(models, "docs/compatibility.md")

    // 4. 生成迁移指南
    generateMigrationGuide(models, "docs/migration.md")
}

func generateMarkdownDocs(models []ModelEntry, output string) {
    var buf bytes.Buffer

    buf.WriteString("# 支持的模型列表\n\n")
    buf.WriteString("本文档由工具自动生成，请勿手动编辑。\n\n")

    // 按提供商分组
    byProvider := groupByProvider(models)

    for provider, entries := range byProvider {
        buf.WriteString(fmt.Sprintf("## %s\n\n", strings.Title(provider)))
        buf.WriteString("| 模型ID | 显示名称 | 平台 | 别名 | 状态 |\n")
        buf.WriteString("|--------|----------|------|------|------|\n")

        for _, entry := range entries {
            status := entry.Status
            if status == "" {
                status = "stable"
            }
            buf.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s | %s |\n",
                entry.ID,
                entry.DisplayName,
                strings.Join(entry.Platforms, ", "),
                strings.Join(entry.Aliases, ", "),
                status))
        }
        buf.WriteString("\n")
    }

    os.WriteFile(output, buf.Bytes(), 0644)
}

func generateCompatibilityMatrix(models []ModelEntry, output string) {
    var buf bytes.Buffer

    buf.WriteString("# 模型兼容性矩阵\n\n")
    buf.WriteString("| 模型 | Anthropic OAuth | Anthropic API Key | OpenAI | Gemini | Antigravity |\n")
    buf.WriteString("|------|----------------|-------------------|--------|--------|-------------|\n")

    for _, entry := range models {
        buf.WriteString(fmt.Sprintf("| %s ", entry.ID))
        for _, platform := range []string{"anthropic", "openai", "gemini", "antigravity"} {
            if contains(entry.Platforms, platform) {
                buf.WriteString("| ✅ ")
            } else {
                buf.WriteString("| ❌ ")
            }
        }
        buf.WriteString("|\n")
    }

    os.WriteFile(output, buf.Bytes(), 0644)
}
CI/CD集成
文件：.github/workflows/docs.yml

name: Generate Documentation

on:
  push:
    paths:
      - 'backend/internal/modelregistry/registry_seed.json'
  workflow_dispatch:

jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Generate docs
        run: |
          cd backend
          go run cmd/gendocs/main.go

      - name: Commit docs
        run: |
          git config user.name "GitHub Actions"
          git config user.email "actions@github.com"
          git add docs/
          git commit -m "docs: auto-generate model documentation" || exit 0
          git push
实施步骤（更新版）
Step 1: 准备阶段（无破坏性）
增强 modelregistry/types.go 数据结构（包含版本管理字段）
完善 registry_seed.json 中的映射数据（补充所有 protocol_ids、aliases、status）
在 modelregistry/loader.go 中实现新的索引和查询函数
创建 custom_models 数据库表
编写单元测试验证新查询函数的正确性
Step 2: 网关层迁移
修改 gateway_request.go 的入口转换逻辑（包含自动迁移）
修改 gateway_claude_normalize.go 的上游请求构建
修改 gateway_account_selection.go 删除特殊处理
更新 gateway_response_stream.go 和 gateway_count_tokens.go 的调用
添加 gateway_metrics.go 指标收集
运行集成测试验证网关功能
Step 3: 计费系统迁移
修改 billing_service.go 的定价查询逻辑
修改 model_catalog_identity.go 简化查询
修改 model_catalog_entries.go 从registry生成
运行计费相关测试
Step 4: 动态注册与热更新
实现 modelregistry/dynamic.go 动态注册表
实现 service/model_registry_service.go 服务层接口
实现 handler/admin/model_registry_handler.go HTTP接口
实现 repository/custom_model_repo.go 数据持久化
测试热更新功能
Step 5: 监控与告警
实现 gateway_metrics.go 指标收集
实现 handler/admin/model_analytics_handler.go 统计接口
配置 Prometheus 告警规则
创建 Grafana 监控面板
Step 6: 文档生成
实现 cmd/gendocs/main.go 文档生成工具
配置 CI/CD 自动生成文档
生成初始文档
Step 7: 清理冗余代码
删除 claude/constants.go 中的映射代码
删除 model_catalog_seed.json 文件
删除 modelregistry/overlays.go 中的重复映射
更新所有引用这些代码的地方
Step 8: 前端适配
更新 types/index.ts 类型定义（包含status字段）
更新API调用接口
实现模型状态标记UI（beta/deprecated）
实现模型分析统计页面
前端测试验证
Step 9: 全面测试
单元测试：所有模型ID转换函数
集成测试：网关请求流程（Anthropic OAuth/API Key、OpenAI、Gemini）
端到端测试：完整请求链路（前端→网关→上游API→计费）
性能测试：热更新、并发查询
回归测试：现有功能不受影响
Step 10: 上线与监控
灰度发布：先发布到测试环境
监控指标：观察未识别模型ID、解析失败率
全量发布：逐步扩大流量
持续优化：根据监控数据调整
验证方案（更新版）
功能验证
模型ID解析验证（同前）

上游API调用验证（同前）

计费验证（同前）

前端验证（同前）

动态注册验证：

# 添加自定义模型
curl -X POST /admin/models/registry \
  -d '{"id":"custom-model-1","display_name":"Custom Model","provider":"openai",...}'

# 验证立即生效（无需重启）
curl -X POST /v1/messages -d '{"model":"custom-model-1",...}'

# 删除自定义模型
curl -X DELETE /admin/models/registry/custom-model-1
版本管理验证：

# 标记模型为deprecated
curl -X PATCH /admin/models/registry/old-model \
  -d '{"status":"deprecated","replaced_by":"new-model"}'

# 请求old-model，验证自动迁移到new-model
curl -X POST /v1/messages -d '{"model":"old-model",...}'
# 检查日志：Model deprecated, auto-migrating: old-model -> new-model
监控验证：

# 查看Prometheus指标
curl http://localhost:9090/metrics | grep model_

# 查看统计数据
curl /admin/analytics/models
文档生成验证：

# 运行文档生成工具
go run backend/cmd/gendocs/main.go

# 验证生成的文档
cat docs/models.md
cat docs/compatibility.md
性能验证
查询性能（同前）

热更新性能：

添加模型：< 10ms
索引重建：< 50ms
并发查询：支持10000 QPS
内存占用：

种子模型索引：约40KB
自定义模型（100个）：约50KB
总计：< 100KB（可接受）
回归测试（同前）
风险与缓解（更新版）
风险1-4（同前）
风险5：热更新并发安全
风险：并发读写导致数据不一致
缓解：
使用 sync.RWMutex 保护临界区
写操作加写锁，读操作加读锁
索引重建采用Copy-on-Write策略
风险6：自动迁移误判
风险：错误地将正常模型标记为deprecated
缓解：
需要管理员权限才能修改模型状态
自动迁移记录详细日志
提供回滚机制
风险7：监控数据量过大
风险：未识别模型ID过多导致存储压力
缓解：
使用Prometheus的标签限制（max 1000个unique labels）
定期清理过期数据
采样记录（高频模型全量，低频模型采样）
预期收益（更新版）
立即收益（同前）
长期收益
运维效率提升：

新增模型：从"修改代码+重启服务"到"API调用+热更新"
故障排查：从"查日志"到"看监控面板"
文档维护：从"手动编写"到"自动生成"
业务灵活性提升：

支持A/B测试：动态切换模型映射
快速响应：上游API新增模型，立即支持
平滑迁移：废弃模型自动迁移，用户无感知
数据驱动决策：

模型使用趋势分析
未识别模型ID统计（发现新需求）
废弃模型使用情况（评估迁移进度）
开发体验提升：

文档自动生成，始终保持最新
兼容性矩阵一目了然
迁移指南自动更新