<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <GroupsFilters :ctx="groupsViewContext" />
      </template>

      <template #table>
        <GroupsTable :ctx="groupsViewContext" />
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <GroupCreateDialog :ctx="groupsViewContext" />
    <GroupEditDialog :ctx="groupsViewContext" />
    <GroupsDialogs :ctx="groupsViewContext" />
  </AppLayout>
</template>
<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useOnboardingStore } from '@/stores/onboarding'
import { adminAPI } from '@/api/admin'
import type {
  AdminGroup,
  CompositeModelRoute,
  CompositeRoutePreviewResult,
  CreateGroupRequest,
  GroupPlatform,
  ReasoningEffortMapping,
  OpenAIGroupImageProtocolMode,
  SubscriptionType,
  UpdateGroupRequest
} from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import GroupsFilters from './groups/GroupsFilters.vue'
import GroupsTable from './groups/GroupsTable.vue'
import GroupCreateDialog from './groups/GroupCreateDialog.vue'
import GroupEditDialog from './groups/GroupEditDialog.vue'
import GroupsDialogs from './groups/GroupsDialogs.vue'
import {
  buildGroupSelectOption,
  normalizeOpenAIGroupImageProtocolMode,
  useGroupOptions
} from './groups/useGroupOptions'
import { createStableObjectKeyResolver } from '@/utils/stableObjectKey'
import { useKeyedDebouncedSearch } from '@/composables/useKeyedDebouncedSearch'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { joinModelPatternText, parseModelPatternText } from '@/utils/modelPatternText'

const { t } = useI18n()
const appStore = useAppStore()
const onboardingStore = useOnboardingStore()
const GROUP_COLUMNS_STORAGE_KEY = 'sub2api.admin.groups.hiddenColumns'
const ALWAYS_VISIBLE_GROUP_COLUMNS = ['name', 'actions']
const IMAGE_BATCH_DEFAULT_MAX_ITEMS = 50
const IMAGE_BATCH_DEFAULT_MAX_DOWNLOAD_BYTES = 100 * 1024 * 1024
const IMAGE_BATCH_DEFAULT_DOWNLOAD_CONCURRENCY = 2
const ANTIGRAVITY_DEFAULT_MODEL_SCOPES = ['claude', 'gemini_text', 'gemini_image']
const DEFAULT_REASONING_MAPPING: ReasoningEffortMapping = {
  model: '',
  from: '',
  to: 'medium',
  reasoning_effort: 'medium'
}

const columns = computed<Column[]>(() => [
  { key: 'id', label: t('admin.groups.columns.id'), sortable: true },
  { key: 'name', label: t('admin.groups.columns.name'), sortable: true },
  { key: 'priority', label: t('admin.groups.columns.priority'), sortable: true },
  { key: 'platform', label: t('admin.groups.columns.platform'), sortable: true },
  { key: 'billing_type', label: t('admin.groups.columns.billingType'), sortable: true },
  { key: 'rate_multiplier', label: t('admin.groups.columns.rateMultiplier'), sortable: true },
  { key: 'peak_rate', label: t('admin.groups.columns.peakRate'), sortable: false },
  { key: 'is_exclusive', label: t('admin.groups.columns.type'), sortable: true },
  { key: 'account_count', label: t('admin.groups.columns.accounts'), sortable: true },
  { key: 'capacity', label: t('admin.groups.columns.capacity'), sortable: false },
  { key: 'usage', label: t('admin.groups.columns.usage'), sortable: false },
  { key: 'status', label: t('admin.groups.columns.status'), sortable: true },
  { key: 'actions', label: t('admin.groups.columns.actions'), sortable: false }
])
const loadHiddenGroupColumns = () => {
  if (typeof window === 'undefined') {
    return new Set<string>()
  }
  try {
    const raw = window.localStorage.getItem(GROUP_COLUMNS_STORAGE_KEY)
    const parsed = raw ? JSON.parse(raw) : []
    return new Set(
      Array.isArray(parsed)
        ? parsed.filter((key) => typeof key === 'string' && !ALWAYS_VISIBLE_GROUP_COLUMNS.includes(key))
        : []
    )
  } catch {
    return new Set<string>()
  }
}
const hiddenGroupColumns = ref<Set<string>>(loadHiddenGroupColumns())
const visibleColumns = computed<Column[]>(() =>
  columns.value.filter((column) => !hiddenGroupColumns.value.has(column.key))
)
const toggleGroupColumn = (key: string) => {
  if (ALWAYS_VISIBLE_GROUP_COLUMNS.includes(key)) {
    return
  }
  const next = new Set(hiddenGroupColumns.value)
  if (next.has(key)) {
    next.delete(key)
  } else {
    next.add(key)
  }
  hiddenGroupColumns.value = next
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(GROUP_COLUMNS_STORAGE_KEY, JSON.stringify([...next]))
  }
}

const groups = ref<AdminGroup[]>([])
const loading = ref(false)
const usageMap = ref<Map<number, { today_cost: number; total_cost: number }>>(new Map())
const usageLoading = ref(false)
const capacityMap = ref<Map<number, { concurrencyUsed: number; concurrencyMax: number; sessionsUsed: number; sessionsMax: number; rpmUsed: number; rpmMax: number }>>(new Map())
const searchQuery = ref('')
const filters = reactive({
  platform: '',
  status: '',
  is_exclusive: ''
})
const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})

let abortController: AbortController | null = null

const showCreateModal = ref(false)
const showEditModal = ref(false)
const showDeleteDialog = ref(false)
const showSortModal = ref(false)
const submitting = ref(false)
const sortSubmitting = ref(false)
const editingGroup = ref<AdminGroup | null>(null)
const deletingGroup = ref<AdminGroup | null>(null)
const showRateMultipliersModal = ref(false)
const rateMultipliersGroup = ref<AdminGroup | null>(null)
const sortableGroups = ref<AdminGroup[]>([])
const createCopyAccountsSelection = ref<number | null>(null)
const editCopyAccountsSelection = ref<number | null>(null)
const createCompositePreviewModel = ref('')
const createCompositePreviewLoading = ref(false)
const createCompositePreviewResult = ref<CompositeRoutePreviewResult | null>(null)
const editCompositePreviewModel = ref('')
const editCompositePreviewLoading = ref(false)
const editCompositePreviewResult = ref<CompositeRoutePreviewResult | null>(null)

const createForm = reactive({
  name: '',
  description: '',
  platform: 'anthropic' as GroupPlatform,
  priority: 1,
  rate_multiplier: 1.0,
  profit_control_enabled: false,
  profit_min_margin: 0,
  profit_safety_buffer: 0,
  peak_rate_enabled: false,
  peak_start: '09:00',
  peak_end: '18:00',
  peak_rate_multiplier: 1.0,
  is_exclusive: false,
  gemini_mixed_protocol_enabled: false,
  subscription_type: 'standard' as SubscriptionType,
  daily_limit_usd: null as number | null,
  weekly_limit_usd: null as number | null,
  monthly_limit_usd: null as number | null,
  image_price_1k: null as number | null,
  image_price_2k: null as number | null,
  image_price_4k: null as number | null,
  web_search_price_per_call: null as number | string | null,
  image_protocol_mode: 'inherit' as OpenAIGroupImageProtocolMode,
  claude_code_only: false,
  fallback_group_id: null as number | null,
  fallback_group_id_on_invalid_request: null as number | null,
  allow_messages_dispatch: false,
  allow_live: false,
  max_reasoning_effort: '' as ReasoningEffortMapping['to'],
  max_reasoning_effort_over_limit: 'downgrade' as 'downgrade' | 'deny',
  force_openai_fast: false,
  free_openai_fast: false,
  reasoning_effort_mappings: [] as ReasoningEffortMapping[],
  default_mapped_model: 'gpt-5.4',
  visible_model_patterns_text: '',
  image_batch_enabled: false,
  image_batch_allowed_providers: [] as string[],
  image_batch_allowed_models: [] as string[],
  image_batch_max_items: IMAGE_BATCH_DEFAULT_MAX_ITEMS,
  image_batch_max_download_bytes: IMAGE_BATCH_DEFAULT_MAX_DOWNLOAD_BYTES,
  image_batch_download_concurrency: IMAGE_BATCH_DEFAULT_DOWNLOAD_CONCURRENCY,
  model_routing_enabled: false,
  supported_model_scopes: [] as string[],
  mcp_xml_inject: true,
  composite_routes: [] as CompositeModelRoute[],
  copy_accounts_from_group_ids: [] as number[]
})

interface SimpleAccount {
  id: number
  name: string
}


interface ModelRoutingRule {
  pattern: string
  accounts: SimpleAccount[]
}

const createModelRoutingRules = ref<ModelRoutingRule[]>([])

const editModelRoutingRules = ref<ModelRoutingRule[]>([])

const resolveCreateRuleKey = createStableObjectKeyResolver<ModelRoutingRule>('create-rule')
const resolveEditRuleKey = createStableObjectKeyResolver<ModelRoutingRule>('edit-rule')
const resolveCreateReasoningMappingKey =
  createStableObjectKeyResolver<ReasoningEffortMapping>('create-reasoning')
const resolveEditReasoningMappingKey =
  createStableObjectKeyResolver<ReasoningEffortMapping>('edit-reasoning')
const resolveCreateCompositeRouteKey =
  createStableObjectKeyResolver<CompositeModelRoute>('create-composite-route')
const resolveEditCompositeRouteKey =
  createStableObjectKeyResolver<CompositeModelRoute>('edit-composite-route')

const getCreateRuleRenderKey = (rule: ModelRoutingRule) => resolveCreateRuleKey(rule)
const getEditRuleRenderKey = (rule: ModelRoutingRule) => resolveEditRuleKey(rule)
const getCreateReasoningMappingKey = (mapping: ReasoningEffortMapping) =>
  resolveCreateReasoningMappingKey(mapping)
const getEditReasoningMappingKey = (mapping: ReasoningEffortMapping) =>
  resolveEditReasoningMappingKey(mapping)
const getCreateCompositeRouteKey = (route: CompositeModelRoute) =>
  resolveCreateCompositeRouteKey(route)
const getEditCompositeRouteKey = (route: CompositeModelRoute) =>
  resolveEditCompositeRouteKey(route)

const getCreateRuleSearchKey = (rule: ModelRoutingRule) => `create-${resolveCreateRuleKey(rule)}`
const getEditRuleSearchKey = (rule: ModelRoutingRule) => `edit-${resolveEditRuleKey(rule)}`

const getRuleSearchKey = (rule: ModelRoutingRule, isEdit: boolean = false) => {
  return isEdit ? getEditRuleSearchKey(rule) : getCreateRuleSearchKey(rule)
}

const accountSearchKeyword = ref<Record<string, string>>({})
const accountSearchResults = ref<Record<string, SimpleAccount[]>>({})
const showAccountDropdown = ref<Record<string, boolean>>({})

const clearAccountSearchStateByKey = (key: string) => {
  delete accountSearchKeyword.value[key]
  delete accountSearchResults.value[key]
  delete showAccountDropdown.value[key]
}

const clearAllAccountSearchState = () => {
  accountSearchKeyword.value = {}
  accountSearchResults.value = {}
  showAccountDropdown.value = {}
}

const accountSearchRunner = useKeyedDebouncedSearch<SimpleAccount[]>({
  delay: 300,
  search: async (keyword, { signal }) => {
    const res = await adminAPI.accounts.list(
      1,
      20,
      {
        search: keyword,
        platform: 'anthropic'
      },
      { signal }
    )
    return res.items.map((account) => ({ id: account.id, name: account.name }))
  },
  onSuccess: (key, result) => {
    accountSearchResults.value[key] = result
  },
  onError: (key) => {
    accountSearchResults.value[key] = []
  }
})

const searchAccounts = (key: string) => {
  accountSearchRunner.trigger(key, accountSearchKeyword.value[key] || '')
}

const searchAccountsByRule = (rule: ModelRoutingRule, isEdit: boolean = false) => {
  searchAccounts(getRuleSearchKey(rule, isEdit))
}


const selectAccount = (rule: ModelRoutingRule, account: SimpleAccount, isEdit: boolean = false) => {
  if (!rule) return


  if (!rule.accounts.some(a => a.id === account.id)) {
    rule.accounts.push(account)
  }


  const key = getRuleSearchKey(rule, isEdit)
  accountSearchKeyword.value[key] = ''
  showAccountDropdown.value[key] = false
}

const removeSelectedAccount = (rule: ModelRoutingRule, accountId: number, _isEdit: boolean = false) => {
  if (!rule) return

  rule.accounts = rule.accounts.filter(a => a.id !== accountId)
}


const toggleCreateScope = (scope: string) => {
  const idx = createForm.supported_model_scopes.indexOf(scope)
  if (idx === -1) {
    createForm.supported_model_scopes.push(scope)
  } else {
    createForm.supported_model_scopes.splice(idx, 1)
  }
}


const toggleEditScope = (scope: string) => {
  const idx = editForm.supported_model_scopes.indexOf(scope)
  if (idx === -1) {
    editForm.supported_model_scopes.push(scope)
  } else {
    editForm.supported_model_scopes.splice(idx, 1)
  }
}

const onAccountSearchFocus = (rule: ModelRoutingRule, isEdit: boolean = false) => {
  const key = getRuleSearchKey(rule, isEdit)
  showAccountDropdown.value[key] = true
  if (!accountSearchResults.value[key]?.length) {
    searchAccounts(key)
  }
}

const addCreateRoutingRule = () => {
  createModelRoutingRules.value.push({ pattern: '', accounts: [] })
}

const removeCreateRoutingRule = (rule: ModelRoutingRule) => {
  const index = createModelRoutingRules.value.indexOf(rule)
  if (index === -1) return

  const key = getCreateRuleSearchKey(rule)
  accountSearchRunner.clearKey(key)
  clearAccountSearchStateByKey(key)
  createModelRoutingRules.value.splice(index, 1)
}

const addEditRoutingRule = () => {
  editModelRoutingRules.value.push({ pattern: '', accounts: [] })
}

const removeEditRoutingRule = (rule: ModelRoutingRule) => {
  const index = editModelRoutingRules.value.indexOf(rule)
  if (index === -1) return

  const key = getEditRuleSearchKey(rule)
  accountSearchRunner.clearKey(key)
  clearAccountSearchStateByKey(key)
  editModelRoutingRules.value.splice(index, 1)
}

const addCreateReasoningMapping = () => {
  createForm.reasoning_effort_mappings.push(cloneReasoningMapping(DEFAULT_REASONING_MAPPING))
}

const removeCreateReasoningMapping = (mapping: ReasoningEffortMapping) => {
  const index = createForm.reasoning_effort_mappings.indexOf(mapping)
  if (index >= 0) {
    createForm.reasoning_effort_mappings.splice(index, 1)
  }
}

const addEditReasoningMapping = () => {
  editForm.reasoning_effort_mappings.push(cloneReasoningMapping(DEFAULT_REASONING_MAPPING))
}

const removeEditReasoningMapping = (mapping: ReasoningEffortMapping) => {
  const index = editForm.reasoning_effort_mappings.indexOf(mapping)
  if (index >= 0) {
    editForm.reasoning_effort_mappings.splice(index, 1)
  }
}

const addCreateCompositeRoute = () => {
  createForm.composite_routes.push(cloneCompositeRoute())
}

const removeCreateCompositeRoute = (route: CompositeModelRoute) => {
  const index = createForm.composite_routes.indexOf(route)
  if (index >= 0) {
    createForm.composite_routes.splice(index, 1)
  }
}

const addEditCompositeRoute = () => {
  editForm.composite_routes.push(cloneCompositeRoute())
}

const removeEditCompositeRoute = (route: CompositeModelRoute) => {
  const index = editForm.composite_routes.indexOf(route)
  if (index >= 0) {
    editForm.composite_routes.splice(index, 1)
  }
}

const previewCreateCompositeRoute = async () => {
  const model = createCompositePreviewModel.value.trim()
  if (!model) return
  createCompositePreviewLoading.value = true
  try {
    createCompositePreviewResult.value = previewCompositeRouteLocally(createForm.composite_routes, model)
  } finally {
    createCompositePreviewLoading.value = false
  }
}

const previewEditCompositeRoute = async () => {
  const model = editCompositePreviewModel.value.trim()
  if (!model) return
  editCompositePreviewLoading.value = true
  try {
    editCompositePreviewResult.value = previewCompositeRouteLocally(editForm.composite_routes, model)
  } finally {
    editCompositePreviewLoading.value = false
  }
}


const convertRoutingRulesToApiFormat = (rules: ModelRoutingRule[]): Record<string, number[]> | null => {
  const result: Record<string, number[]> = {}
  let hasValidRules = false

  for (const rule of rules) {
    const pattern = rule.pattern.trim()
    if (!pattern) continue

    const accountIds = rule.accounts.map(a => a.id).filter(id => id > 0)

    if (accountIds.length > 0) {
      result[pattern] = accountIds
      hasValidRules = true
    }
  }

  return hasValidRules ?
 result : null
}


const convertApiFormatToRoutingRules = async (apiFormat: Record<string, number[]> | null): Promise<ModelRoutingRule[]> => {
  if (!apiFormat) return []

  const rules: ModelRoutingRule[] = []
  for (const [pattern, accountIds] of Object.entries(apiFormat)) {
    const accounts: SimpleAccount[] = []
    for (const id of accountIds) {
      try {
        const account = await adminAPI.accounts.getById(id)
        accounts.push({ id: account.id, name: account.name })
      } catch {
        accounts.push({ id, name: `#${id}` })
      }
    }
    rules.push({ pattern, accounts })
  }
  return rules
}

const getAntigravityDefaultModelScopes = () => [...ANTIGRAVITY_DEFAULT_MODEL_SCOPES]

const editForm = reactive({
  name: '',
  description: '',
  platform: 'anthropic' as GroupPlatform,
  priority: 1,
  rate_multiplier: 1.0,
  profit_control_enabled: false,
  profit_min_margin: 0,
  profit_safety_buffer: 0,
  peak_rate_enabled: false,
  peak_start: '09:00',
  peak_end: '18:00',
  peak_rate_multiplier: 1.0,
  is_exclusive: false,
  gemini_mixed_protocol_enabled: false,
  status: 'active' as 'active' | 'inactive',
  subscription_type: 'standard' as SubscriptionType,
  daily_limit_usd: null as number | null,
  weekly_limit_usd: null as number | null,
  monthly_limit_usd: null as number | null,
  image_price_1k: null as number | null,
  image_price_2k: null as number | null,
  image_price_4k: null as number | null,
  web_search_price_per_call: null as number | string | null,
  image_protocol_mode: 'inherit' as OpenAIGroupImageProtocolMode,
  claude_code_only: false,
  fallback_group_id: null as number | null,
  fallback_group_id_on_invalid_request: null as number | null,
  allow_messages_dispatch: false,
  allow_live: false,
  max_reasoning_effort: '' as ReasoningEffortMapping['to'],
  max_reasoning_effort_over_limit: 'downgrade' as 'downgrade' | 'deny',
  force_openai_fast: false,
  free_openai_fast: false,
  reasoning_effort_mappings: [] as ReasoningEffortMapping[],
  default_mapped_model: '',
  visible_model_patterns_text: '',
  image_batch_enabled: false,
  image_batch_allowed_providers: [] as string[],
  image_batch_allowed_models: [] as string[],
  image_batch_max_items: IMAGE_BATCH_DEFAULT_MAX_ITEMS,
  image_batch_max_download_bytes: IMAGE_BATCH_DEFAULT_MAX_DOWNLOAD_BYTES,
  image_batch_download_concurrency: IMAGE_BATCH_DEFAULT_DOWNLOAD_CONCURRENCY,
  model_routing_enabled: false,
  supported_model_scopes: [] as string[],
  mcp_xml_inject: true,
  composite_routes: [] as CompositeModelRoute[],
  copy_accounts_from_group_ids: [] as number[]
})

const {
  statusOptions,
  exclusiveOptions,
  platformOptions,
  platformFilterOptions,
  editStatusOptions,
  subscriptionTypeOptions,
  openAIGroupImageProtocolModeOptions,
  fallbackGroupOptions,
  fallbackGroupOptionsForEdit,
  invalidRequestFallbackOptions,
  invalidRequestFallbackOptionsForEdit
} = useGroupOptions(t, groups, editingGroup)

const buildCopyAccountsDescription = (group: AdminGroup): string => {
  return `${group.account_count || 0} ${t('admin.groups.accountsTotal')}`
}

const copyAccountsGroupSelectOptions = computed(() => {
  const eligibleGroups = groups.value.filter(
    (g) => g.platform === createForm.platform && (g.account_count || 0) > 0
  )
  return eligibleGroups.map((g) => buildGroupSelectOption(g, buildCopyAccountsDescription(g)))
})

const copyAccountsGroupSelectOptionsForEdit = computed(() => {
  const currentId = editingGroup.value?.id
  const eligibleGroups = groups.value.filter(
    (g) => g.platform === editForm.platform && (g.account_count || 0) > 0 && g.id !== currentId
  )
  return eligibleGroups.map((g) => buildGroupSelectOption(g, buildCopyAccountsDescription(g)))
})

const buildCompositeTargetDescription = (group: AdminGroup): string => {
  return `${t(`admin.groups.platforms.${group.platform}`)} · ${group.account_count || 0} ${t('admin.groups.accountsTotal')}`
}

const compositeTargetGroupOptions = computed(() =>
  groups.value
    .filter((g) => g.platform !== 'composite' && g.status === 'active')
    .map((g) => buildGroupSelectOption(g, buildCompositeTargetDescription(g)))
)

const compositeTargetGroupOptionsForEdit = computed(() => {
  const currentId = editingGroup.value?.id
  return groups.value
    .filter((g) => g.platform !== 'composite' && g.status === 'active' && g.id !== currentId)
    .map((g) => buildGroupSelectOption(g, buildCompositeTargetDescription(g)))
})

function cloneReasoningMapping(mapping?: Partial<ReasoningEffortMapping>): ReasoningEffortMapping {
  const to = (mapping?.to || mapping?.reasoning_effort || DEFAULT_REASONING_MAPPING.to || 'medium') as ReasoningEffortMapping['to']
  return {
    model: String(mapping?.model || ''),
    from: (mapping?.from || '') as ReasoningEffortMapping['from'],
    to,
    reasoning_effort: to
  }
}

function cloneCompositeRoute(route?: Partial<CompositeModelRoute>): CompositeModelRoute {
  return {
    id: route?.id,
    parent_group_id: route?.parent_group_id,
    display_model_id: String(route?.display_model_id || ''),
    target_group_id: Number(route?.target_group_id || 0),
    target_model_id: String(route?.target_model_id || ''),
    priority: Number(route?.priority || 50),
    enabled: route?.enabled !== false,
    notes: String(route?.notes || ''),
    target_group: route?.target_group || null
  }
}

const normalizeRoutePriority = (value: number | null | undefined): number => {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? Math.floor(parsed) : 50
}

const normalizeReasoningMappingsForPayload = (
  mappings: ReasoningEffortMapping[]
): ReasoningEffortMapping[] =>
  mappings
    .map((mapping) => cloneReasoningMapping(mapping))
    .filter((mapping) => mapping.model.trim() && (mapping.to || mapping.reasoning_effort))
    .map((mapping) => ({
      model: mapping.model.trim(),
      from: mapping.from || '',
      to: mapping.to || mapping.reasoning_effort || '',
      reasoning_effort: mapping.to || mapping.reasoning_effort || ''
    }))

const normalizeCompositeRoutesForPayload = (
  routes: CompositeModelRoute[]
): CompositeModelRoute[] =>
  routes
    .map((route) => cloneCompositeRoute(route))
    .filter((route) => route.display_model_id.trim() && route.target_group_id > 0)
    .map((route) => ({
      display_model_id: route.display_model_id.trim(),
      target_group_id: route.target_group_id,
      target_model_id: route.target_model_id.trim(),
      priority: normalizeRoutePriority(route.priority),
      enabled: route.enabled !== false,
      notes: (route.notes || '').trim()
    }))

const resetCreateCompositePreview = () => {
  createCompositePreviewModel.value = ''
  createCompositePreviewResult.value = null
  createCompositePreviewLoading.value = false
}

const resetEditCompositePreview = () => {
  editCompositePreviewModel.value = ''
  editCompositePreviewResult.value = null
  editCompositePreviewLoading.value = false
}

const escapeRegExp = (value: string) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

const matchCompositeModelPattern = (pattern: string, model: string): boolean => {
  const normalizedPattern = pattern.trim()
  const normalizedModel = model.trim()
  if (!normalizedPattern || !normalizedModel) {
    return false
  }
  if (!normalizedPattern.includes('*')) {
    return normalizedPattern === normalizedModel
  }
  const regex = new RegExp(`^${normalizedPattern.split('*').map(escapeRegExp).join('.*')}$`)
  return regex.test(normalizedModel)
}

const previewCompositeRouteLocally = (
  routes: CompositeModelRoute[],
  model: string
): CompositeRoutePreviewResult => {
  const normalizedModel = model.trim()
  const normalizedRoutes = normalizeCompositeRoutesForPayload(routes)
  const matched = normalizedRoutes
    .filter((route) => route.enabled !== false)
    .sort((a, b) => a.priority - b.priority)
    .find((route) => matchCompositeModelPattern(route.display_model_id, normalizedModel))
  if (!matched) {
    return { matched: false, display_model_id: normalizedModel }
  }
  const targetGroup = groups.value.find((group) => group.id === matched.target_group_id) || null
  return {
    matched: true,
    display_model_id: normalizedModel,
    target_model_id: matched.target_model_id || normalizedModel,
    target_group_id: matched.target_group_id,
    target_group: targetGroup,
    route: matched
  }
}

function findGroupSelectOption(options: Array<{ value: number | null }>, groupID: number) {
  return options.find((option) => option.value === groupID) || null
}

function isPlatformSelectOption(option: unknown) {
  return typeof option === 'object' && option !== null && 'value' in option && 'label' in option
}

function isGroupSelectOption(option: unknown) {
  return typeof option === 'object' && option !== null && 'value' in option && 'label' in option
}
const deleteConfirmMessage = computed(() => {
  if (!deletingGroup.value) {
    return ''
  }
  if (deletingGroup.value.subscription_type === 'subscription') {
    return t('admin.groups.deleteConfirmSubscription', { name: deletingGroup.value.name })
  }
  return t('admin.groups.deleteConfirm', { name: deletingGroup.value.name })
})

const loadGroups = async () => {
  if (abortController) {
    abortController.abort()
  }
  const currentController = new AbortController()
  abortController = currentController
  const { signal } = currentController
  loading.value = true
  try {
    const response = await adminAPI.groups.list(pagination.page, pagination.page_size, {
      platform: (filters.platform as GroupPlatform) || undefined,
      status: filters.status as any,
      is_exclusive: filters.is_exclusive ? filters.is_exclusive === 'true' : undefined,
      search: searchQuery.value.trim() || undefined
    }, { signal })
    if (signal.aborted) return
    groups.value = response.items
    pagination.total = response.total
    pagination.pages = response.pages
    loadUsageSummary()
    loadCapacitySummary()
  } catch (error: any) {
    if (signal.aborted || error?.name === 'AbortError' || error?.code === 'ERR_CANCELED') {
      return
    }
    appStore.showError(t('admin.groups.failedToLoad'))
    console.error('Error loading groups:', error)
  } finally {
    if (abortController === currentController && !signal.aborted) {
      loading.value = false
    }
  }
}

const formatCost = (cost: number): string => {
  if (cost >= 1000) return cost.toFixed(0)
  if (cost >= 100) return cost.toFixed(1)
  return cost.toFixed(2)
}

const getGroupAvailableAccounts = (group: AdminGroup): number => {
  if (typeof group.available_account_count === 'number') {
    return Math.max(group.available_account_count, 0)
  }
  return Math.max((group.active_account_count || 0) - (group.rate_limited_account_count || 0), 0)
}

const getGroupDigitCount = (group: AdminGroup): number => {
  return Math.max(String(group.account_count || 0).length, 1)
}

const formatGroupAccountValue = (value: number, group: AdminGroup): string => {
  return String(Math.max(value, 0)).padStart(getGroupDigitCount(group), '0')
}

const formatGroupPeakRate = (group: AdminGroup): string => {
  if (!group.peak_rate_enabled) {
    return '-'
  }
  const start = group.peak_start || '--:--'
  const end = group.peak_end || '--:--'
  const multiplier = group.peak_rate_multiplier ?? 1
  return `${start}-${end} · ${multiplier}x`
}

const handleCreatePlatformChange = () => {
  createForm.copy_accounts_from_group_ids = []
  createCopyAccountsSelection.value = null
  if (createForm.platform !== 'anthropic') {
    createForm.claude_code_only = false
    createForm.fallback_group_id = null
    createForm.fallback_group_id_on_invalid_request = null
    createForm.model_routing_enabled = false
    createModelRoutingRules.value = []
  }
  if (createForm.platform !== 'openai') {
    createForm.allow_messages_dispatch = false
    createForm.default_mapped_model = ''
    createForm.image_protocol_mode = 'inherit'
  }
  if (!supportsOpenAIRuntimePolicy(createForm.platform)) {
    createForm.allow_live = false
    createForm.max_reasoning_effort = ''
    createForm.max_reasoning_effort_over_limit = 'downgrade'
    createForm.reasoning_effort_mappings = []
  }
  if (createForm.platform !== 'composite') {
    createForm.composite_routes = []
    resetCreateCompositePreview()
  }
  if (createForm.platform !== 'antigravity') {
    createForm.mcp_xml_inject = true
    createForm.supported_model_scopes = []
  } else {
    createForm.supported_model_scopes = getAntigravityDefaultModelScopes()
  }
  if (createForm.platform !== 'gemini') {
    createForm.gemini_mixed_protocol_enabled = false
    resetImageBatchConfig(createForm)
  }
  if (!['openai', 'grok'].includes(createForm.platform)) {
    createForm.web_search_price_per_call = null
  }
}

const handleCreateCopyAccountsSelect = (value: string | number | boolean | null) => {
  if (typeof value === 'number' && !createForm.copy_accounts_from_group_ids.includes(value)) {
    createForm.copy_accounts_from_group_ids.push(value)
  }
  createCopyAccountsSelection.value = null
}

const handleEditCopyAccountsSelect = (value: string | number | boolean | null) => {
  if (typeof value === 'number' && !editForm.copy_accounts_from_group_ids.includes(value)) {
    editForm.copy_accounts_from_group_ids.push(value)
  }
  editCopyAccountsSelection.value = null
}

const normalizeGroupPriority = (value: number | null | undefined): number => {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? Math.floor(parsed) : 1
}

const resetPeakRateConfig = (form: {
  peak_rate_enabled: boolean
  peak_start: string
  peak_end: string
  peak_rate_multiplier: number
}) => {
  form.peak_rate_enabled = false
  form.peak_start = '09:00'
  form.peak_end = '18:00'
  form.peak_rate_multiplier = 1.0
}

const parsePeakTimeMinutes = (value: string): number | null => {
  const match = /^([01]\d|2[0-3]):([0-5]\d)$/.exec(String(value || '').trim())
  if (!match) {
    return null
  }
  return Number(match[1]) * 60 + Number(match[2])
}

const validatePeakRateConfig = (form: {
  subscription_type: SubscriptionType
  peak_rate_enabled: boolean
  peak_start: string
  peak_end: string
  peak_rate_multiplier: number
}) => {
  if (form.subscription_type !== 'subscription' || !form.peak_rate_enabled) {
    return true
  }
  const startMinutes = parsePeakTimeMinutes(form.peak_start)
  const endMinutes = parsePeakTimeMinutes(form.peak_end)
  if (startMinutes == null || endMinutes == null) {
    appStore.showError(t('admin.groups.peakRate.invalidTime'))
    return false
  }
  if (startMinutes >= endMinutes) {
    appStore.showError(t('admin.groups.peakRate.invalidRange'))
    return false
  }
  const multiplier = Number(form.peak_rate_multiplier)
  if (!Number.isFinite(multiplier) || multiplier < 0) {
    appStore.showError(t('admin.groups.peakRate.invalidMultiplier'))
    return false
  }
  return true
}

const applyPeakRatePayload = <T extends {
  subscription_type: SubscriptionType
  peak_rate_enabled: boolean
  peak_start: string
  peak_end: string
  peak_rate_multiplier: number
}>(payload: T): T => {
  if (payload.subscription_type !== 'subscription' || !payload.peak_rate_enabled) {
    payload.peak_rate_enabled = false
    payload.peak_start = ''
    payload.peak_end = ''
    payload.peak_rate_multiplier = 1.0
    return payload
  }
  payload.peak_start = payload.peak_start.trim()
  payload.peak_end = payload.peak_end.trim()
  payload.peak_rate_multiplier = Number(payload.peak_rate_multiplier)
  return payload
}

const resetImageBatchConfig = (form: {
  image_batch_enabled: boolean
  image_batch_allowed_providers: string[]
  image_batch_allowed_models: string[]
  image_batch_max_items: number
  image_batch_max_download_bytes: number
  image_batch_download_concurrency: number
}) => {
  form.image_batch_enabled = false
  form.image_batch_allowed_providers = []
  form.image_batch_allowed_models = []
  form.image_batch_max_items = IMAGE_BATCH_DEFAULT_MAX_ITEMS
  form.image_batch_max_download_bytes = IMAGE_BATCH_DEFAULT_MAX_DOWNLOAD_BYTES
  form.image_batch_download_concurrency = IMAGE_BATCH_DEFAULT_DOWNLOAD_CONCURRENCY
}

const loadUsageSummary = async () => {
  usageLoading.value = true
  try {
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone
    const data = await adminAPI.groups.getUsageSummary(tz)
    const map = new Map<number, { today_cost: number; total_cost: number }>()
    for (const item of data) {
      map.set(item.group_id, { today_cost: item.today_cost, total_cost: item.total_cost })
    }
    usageMap.value = map
  } catch (error) {
    console.error('Error loading group usage summary:', error)
  } finally {
    usageLoading.value = false
  }
}

const loadCapacitySummary = async () => {
  try {
    const data = await adminAPI.groups.getCapacitySummary()
    const map = new Map<number, { concurrencyUsed: number; concurrencyMax: number; sessionsUsed: number; sessionsMax: number; rpmUsed: number; rpmMax: number }>()
    for (const item of data) {
      map.set(item.group_id, {
        concurrencyUsed: item.concurrency_used,
        concurrencyMax: item.concurrency_max,
        sessionsUsed: item.sessions_used,
        sessionsMax: item.sessions_max,
        rpmUsed: item.rpm_used,
        rpmMax: item.rpm_max
      })
    }
    capacityMap.value = map
  } catch (error) {
    console.error('Error loading group capacity summary:', error)
  }
}

let searchTimeout: ReturnType<typeof setTimeout>
const handleSearch = () => {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    pagination.page = 1
    loadGroups()
  }, 300)
}

const handlePageChange = (page: number) => {
  pagination.page = page
  loadGroups()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.page_size = pageSize
  pagination.page = 1
  loadGroups()
}

const closeCreateModal = () => {
  showCreateModal.value = false
  createModelRoutingRules.value.forEach((rule) => {
    accountSearchRunner.clearKey(getCreateRuleSearchKey(rule))
  })
  clearAllAccountSearchState()
  createForm.name = ''
  createForm.description = ''
  createForm.platform = 'anthropic'
  createForm.priority = 1
  createForm.rate_multiplier = 1.0
  createForm.profit_control_enabled = false
  createForm.profit_min_margin = 0
  createForm.profit_safety_buffer = 0
  resetPeakRateConfig(createForm)
  createForm.is_exclusive = false
  createForm.gemini_mixed_protocol_enabled = false
  createForm.subscription_type = 'standard'
  createForm.daily_limit_usd = null
  createForm.weekly_limit_usd = null
  createForm.monthly_limit_usd = null
  createForm.image_price_1k = null
  createForm.image_price_2k = null
  createForm.image_price_4k = null
  createForm.web_search_price_per_call = null
  createForm.image_protocol_mode = 'inherit'
  createForm.claude_code_only = false
  createForm.fallback_group_id = null
  createForm.fallback_group_id_on_invalid_request = null
  createForm.allow_messages_dispatch = false
  createForm.allow_live = false
  createForm.max_reasoning_effort = ''
  createForm.max_reasoning_effort_over_limit = 'downgrade'
  createForm.reasoning_effort_mappings = []
  createForm.default_mapped_model = 'gpt-5.4'
  createForm.visible_model_patterns_text = ''
  createForm.image_batch_enabled = false
  createForm.image_batch_allowed_providers = []
  createForm.image_batch_allowed_models = []
  createForm.image_batch_max_items = IMAGE_BATCH_DEFAULT_MAX_ITEMS
  createForm.image_batch_max_download_bytes = IMAGE_BATCH_DEFAULT_MAX_DOWNLOAD_BYTES
  createForm.image_batch_download_concurrency = IMAGE_BATCH_DEFAULT_DOWNLOAD_CONCURRENCY
  createForm.supported_model_scopes = []
  createForm.mcp_xml_inject = true
  createForm.composite_routes = []
  resetCreateCompositePreview()
  createForm.copy_accounts_from_group_ids = []
  createCopyAccountsSelection.value = null
  createModelRoutingRules.value = []
}

const normalizeOptionalLimit = (value: number | string | null | undefined): number | null => {
  if (value === null || value === undefined) {
    return null
  }

  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed) {
      return null
    }
    const parsed = Number(trimmed)
    return Number.isFinite(parsed) && parsed > 0 ?
 parsed : null
  }

  return Number.isFinite(value) && value > 0 ?
 value : null
}

const normalizeNullableNumber = (value: number | string | null | undefined): number | null => {
  if (value === null || value === undefined) {
    return null
  }
  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed) {
      return null
    }
    const parsed = Number(trimmed)
    return Number.isFinite(parsed) ? parsed : null
  }
  return Number.isFinite(value) ? value : null
}

type GroupPayloadForm = typeof createForm & Partial<typeof editForm>
type GroupPayloadMode = 'create' | 'update'

const supportsImagePricing = (platform: GroupPlatform) =>
  platform === 'antigravity' || platform === 'gemini' || platform === 'grok'

const supportsWebSearchPricing = (platform: GroupPlatform) =>
  platform === 'openai' || platform === 'grok'

const supportsOpenAIRuntimePolicy = (platform: GroupPlatform) =>
  platform === 'openai' || platform === 'kimi' || platform === 'composite'

const buildGroupPayload = (
  form: GroupPayloadForm,
  routingRules: ModelRoutingRule[],
  mode: GroupPayloadMode
): CreateGroupRequest | UpdateGroupRequest => {
  const platform = form.platform
  const payload: CreateGroupRequest | UpdateGroupRequest = {
    name: form.name,
    description: form.description,
    platform,
    priority: normalizeGroupPriority(form.priority),
    rate_multiplier: Number(form.rate_multiplier),
    profit_control_enabled: form.profit_control_enabled,
    profit_min_margin: normalizeNullableNumber(form.profit_min_margin) ?? 0,
    profit_safety_buffer: normalizeNullableNumber(form.profit_safety_buffer) ?? 0,
    peak_rate_enabled: form.peak_rate_enabled,
    peak_start: form.peak_start,
    peak_end: form.peak_end,
    peak_rate_multiplier: Number(form.peak_rate_multiplier),
    is_exclusive: form.is_exclusive,
    subscription_type: form.subscription_type,
    daily_limit_usd: normalizeOptionalLimit(form.daily_limit_usd as number | string | null),
    weekly_limit_usd: normalizeOptionalLimit(form.weekly_limit_usd as number | string | null),
    monthly_limit_usd: normalizeOptionalLimit(form.monthly_limit_usd as number | string | null),
    visible_model_patterns: parseModelPatternText(form.visible_model_patterns_text),
    copy_accounts_from_group_ids: [...form.copy_accounts_from_group_ids],
    supported_model_scopes: platform === 'antigravity'
      ? [...form.supported_model_scopes]
      : []
  }

  if (mode === 'update') {
    const updatePayload = payload as UpdateGroupRequest
    updatePayload.status = form.status
  }

  if (supportsImagePricing(platform)) {
    payload.image_price_1k = normalizeNullableNumber(form.image_price_1k)
    payload.image_price_2k = normalizeNullableNumber(form.image_price_2k)
    payload.image_price_4k = normalizeNullableNumber(form.image_price_4k)
  }

  if (supportsWebSearchPricing(platform)) {
    payload.web_search_price_per_call = normalizeNullableNumber(form.web_search_price_per_call)
  }

  if (supportsOpenAIRuntimePolicy(platform)) {
    payload.allow_live = form.allow_live
    payload.max_reasoning_effort = form.max_reasoning_effort || ''
    payload.max_reasoning_effort_over_limit = form.max_reasoning_effort_over_limit || 'downgrade'
    payload.force_openai_fast = form.force_openai_fast === true
    payload.free_openai_fast = form.free_openai_fast === true
    payload.reasoning_effort_mappings = normalizeReasoningMappingsForPayload(
      form.reasoning_effort_mappings
    )
  }

  if (platform === 'openai') {
    payload.allow_messages_dispatch = form.allow_messages_dispatch
    payload.default_mapped_model = form.default_mapped_model
    payload.image_protocol_mode = form.image_protocol_mode
  }

  if (platform === 'composite') {
    payload.composite_routes = normalizeCompositeRoutesForPayload(form.composite_routes)
  }

  if (platform === 'anthropic') {
    payload.claude_code_only = form.claude_code_only
    payload.fallback_group_id = mode === 'update' && form.fallback_group_id === null
      ? 0
      : form.fallback_group_id
    payload.fallback_group_id_on_invalid_request =
      mode === 'update' && form.fallback_group_id_on_invalid_request === null
        ? 0
        : form.fallback_group_id_on_invalid_request
    payload.model_routing_enabled = form.model_routing_enabled
    payload.model_routing = form.model_routing_enabled
      ? convertRoutingRulesToApiFormat(routingRules)
      : null
  }

  if (platform === 'antigravity') {
    payload.mcp_xml_inject = form.mcp_xml_inject
  }

  if (platform === 'gemini') {
    payload.gemini_mixed_protocol_enabled = form.gemini_mixed_protocol_enabled
    payload.image_batch_enabled = form.image_batch_enabled
    payload.image_batch_allowed_providers = [...form.image_batch_allowed_providers]
    payload.image_batch_allowed_models = [...form.image_batch_allowed_models]
    payload.image_batch_max_items = form.image_batch_max_items
    payload.image_batch_max_download_bytes = form.image_batch_max_download_bytes
    payload.image_batch_download_concurrency = form.image_batch_download_concurrency
  }

  applyPeakRatePayload(payload as CreateGroupRequest & {
    subscription_type: SubscriptionType
    peak_rate_enabled: boolean
    peak_start: string
    peak_end: string
    peak_rate_multiplier: number
  })
  return payload
}

const handleCreateGroup = async () => {
  if (!createForm.name.trim()) {
    appStore.showError(t('admin.groups.nameRequired'))
    return
  }
  if (!validatePeakRateConfig(createForm)) {
    return
  }
  submitting.value = true
  try {
    const requestData = buildGroupPayload(createForm, createModelRoutingRules.value, 'create') as CreateGroupRequest
    await adminAPI.groups.create(requestData)
    appStore.showSuccess(t('admin.groups.groupCreated'))
    closeCreateModal()
    loadGroups()
    if (onboardingStore.isCurrentStep('[data-tour="group-form-submit"]')) {
      onboardingStore.nextStep(500)
    }
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.groups.failedToCreate'))
    console.error('Error creating group:', error)
  } finally {
    submitting.value = false
  }
}

const handleEdit = async (group: AdminGroup) => {
  editingGroup.value = group
  editForm.name = group.name
  editForm.description = group.description || ''
  editForm.platform = group.platform
  editForm.priority = group.priority ?? 1
  editForm.rate_multiplier = group.rate_multiplier
  editForm.profit_control_enabled = group.profit_control_enabled === true
  editForm.profit_min_margin = group.profit_min_margin ?? 0
  editForm.profit_safety_buffer = group.profit_safety_buffer ?? 0
  editForm.peak_rate_enabled = group.peak_rate_enabled === true
  editForm.peak_start = group.peak_start || '09:00'
  editForm.peak_end = group.peak_end || '18:00'
  editForm.peak_rate_multiplier = group.peak_rate_multiplier ?? 1.0
  editForm.is_exclusive = group.is_exclusive
  editForm.gemini_mixed_protocol_enabled = group.gemini_mixed_protocol_enabled || false
  editForm.status = group.status
  editForm.subscription_type = group.subscription_type || 'standard'
  editForm.daily_limit_usd = group.daily_limit_usd
  editForm.weekly_limit_usd = group.weekly_limit_usd
  editForm.monthly_limit_usd = group.monthly_limit_usd
  editForm.image_price_1k = group.image_price_1k
  editForm.image_price_2k = group.image_price_2k
  editForm.image_price_4k = group.image_price_4k
  editForm.web_search_price_per_call = group.web_search_price_per_call ?? null
  editForm.image_protocol_mode = normalizeOpenAIGroupImageProtocolMode(group.image_protocol_mode)
  editForm.claude_code_only = group.claude_code_only || false
  editForm.fallback_group_id = group.fallback_group_id
  editForm.fallback_group_id_on_invalid_request = group.fallback_group_id_on_invalid_request
  editForm.allow_messages_dispatch = group.allow_messages_dispatch || false
  editForm.allow_live = group.allow_live || false
  editForm.max_reasoning_effort = group.max_reasoning_effort || ''
  editForm.max_reasoning_effort_over_limit = group.max_reasoning_effort_over_limit || 'downgrade'
  editForm.force_openai_fast = group.force_openai_fast === true
  editForm.free_openai_fast = group.free_openai_fast === true
  editForm.reasoning_effort_mappings = (group.reasoning_effort_mappings || []).map((mapping) =>
    cloneReasoningMapping(mapping)
  )
  editForm.default_mapped_model = group.default_mapped_model || ''
  editForm.visible_model_patterns_text = joinModelPatternText(group.visible_model_patterns)
  editForm.image_batch_enabled = group.image_batch_enabled || false
  editForm.image_batch_allowed_providers = group.image_batch_allowed_providers || []
  editForm.image_batch_allowed_models = group.image_batch_allowed_models || []
  editForm.image_batch_max_items = group.image_batch_max_items || IMAGE_BATCH_DEFAULT_MAX_ITEMS
  editForm.image_batch_max_download_bytes =
    group.image_batch_max_download_bytes || IMAGE_BATCH_DEFAULT_MAX_DOWNLOAD_BYTES
  editForm.image_batch_download_concurrency =
    group.image_batch_download_concurrency || IMAGE_BATCH_DEFAULT_DOWNLOAD_CONCURRENCY
  editForm.model_routing_enabled = group.model_routing_enabled || false
  editForm.supported_model_scopes = group.platform === 'antigravity'
    ? group.supported_model_scopes || getAntigravityDefaultModelScopes()
    : []
  editForm.mcp_xml_inject = group.mcp_xml_inject ?? true
  editForm.composite_routes = (group.composite_routes || []).map((route) => cloneCompositeRoute(route))
  editForm.copy_accounts_from_group_ids = []
  editModelRoutingRules.value = await convertApiFormatToRoutingRules(group.model_routing)
  editCopyAccountsSelection.value = null
  resetEditCompositePreview()
  showEditModal.value = true
}

const closeEditModal = () => {
  editModelRoutingRules.value.forEach((rule) => {
    accountSearchRunner.clearKey(getEditRuleSearchKey(rule))
  })
  clearAllAccountSearchState()
  showEditModal.value = false
  editingGroup.value = null
  editModelRoutingRules.value = []
  editForm.copy_accounts_from_group_ids = []
  editCopyAccountsSelection.value = null
  editForm.gemini_mixed_protocol_enabled = false
  editForm.profit_control_enabled = false
  editForm.profit_min_margin = 0
  editForm.profit_safety_buffer = 0
  resetPeakRateConfig(editForm)
  editForm.image_protocol_mode = 'inherit'
  editForm.allow_live = false
  editForm.max_reasoning_effort = ''
  editForm.max_reasoning_effort_over_limit = 'downgrade'
  editForm.reasoning_effort_mappings = []
  editForm.web_search_price_per_call = null
  editForm.visible_model_patterns_text = ''
  editForm.image_batch_enabled = false
  editForm.image_batch_allowed_providers = []
  editForm.image_batch_allowed_models = []
  editForm.image_batch_max_items = IMAGE_BATCH_DEFAULT_MAX_ITEMS
  editForm.image_batch_max_download_bytes = IMAGE_BATCH_DEFAULT_MAX_DOWNLOAD_BYTES
  editForm.image_batch_download_concurrency = IMAGE_BATCH_DEFAULT_DOWNLOAD_CONCURRENCY
  editForm.composite_routes = []
  resetEditCompositePreview()
}

const handleUpdateGroup = async () => {
  if (!editingGroup.value) return
  if (!editForm.name.trim()) {
    appStore.showError(t('admin.groups.nameRequired'))
    return
  }
  if (!validatePeakRateConfig(editForm)) {
    return
  }

  submitting.value = true
  try {
    const payload = buildGroupPayload(editForm, editModelRoutingRules.value, 'update') as UpdateGroupRequest
    await adminAPI.groups.update(editingGroup.value.id, payload)
    appStore.showSuccess(t('admin.groups.groupUpdated'))
    closeEditModal()
    loadGroups()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.groups.failedToUpdate'))
    console.error('Error updating group:', error)
  } finally {
    submitting.value = false
  }
}

const handleRateMultipliers = (group: AdminGroup) => {
  rateMultipliersGroup.value = group
  showRateMultipliersModal.value = true
}

const handleDuplicate = async (group: AdminGroup) => {
  try {
    await adminAPI.groups.duplicate(group.id, { copy_accounts: true })
    appStore.showSuccess(t('admin.groups.duplicateSuccess'))
    await loadGroups()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.groups.duplicateFailed'))
    console.error('Error duplicating group:', error)
  }
}

const handleDelete = (group: AdminGroup) => {
  deletingGroup.value = group
  showDeleteDialog.value = true
}

const confirmDelete = async () => {
  if (!deletingGroup.value) return

  try {
    await adminAPI.groups.delete(deletingGroup.value.id)
    appStore.showSuccess(t('admin.groups.groupDeleted'))
    showDeleteDialog.value = false
    deletingGroup.value = null
    loadGroups()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.groups.failedToDelete'))
    console.error('Error deleting group:', error)
  }
}

watch(
  () => createForm.subscription_type,
  (newVal) => {
    if (newVal === 'subscription') {
      createForm.is_exclusive = true
      createForm.fallback_group_id_on_invalid_request = null
      return
    }
    resetPeakRateConfig(createForm)
  }
)

watch(
  () => editForm.subscription_type,
  (newVal) => {
    if (newVal !== 'subscription') {
      resetPeakRateConfig(editForm)
    }
  }
)

watch(
  () => createForm.platform,
  (newVal) => {
    if (newVal !== 'gemini') {
      createForm.gemini_mixed_protocol_enabled = false
    }
    if (newVal !== 'anthropic') {
      createForm.fallback_group_id_on_invalid_request = null
    }
    if (newVal !== 'openai') {
      createForm.image_protocol_mode = 'inherit'
      createForm.allow_messages_dispatch = false
      createForm.default_mapped_model = ''
    }
    if (!supportsOpenAIRuntimePolicy(newVal)) {
      createForm.allow_live = false
      createForm.max_reasoning_effort = ''
      createForm.max_reasoning_effort_over_limit = 'downgrade'
      createForm.reasoning_effort_mappings = []
    }
    if (newVal !== 'composite') {
      createForm.composite_routes = []
      resetCreateCompositePreview()
    }
    if (!['openai', 'grok'].includes(newVal)) {
      createForm.web_search_price_per_call = null
    }
    if (newVal !== 'anthropic') {
      createForm.claude_code_only = false
      createForm.fallback_group_id = null
      createForm.model_routing_enabled = false
      createModelRoutingRules.value = []
    }
    if (newVal !== 'antigravity') {
      createForm.mcp_xml_inject = true
      createForm.supported_model_scopes = []
    } else if (createForm.supported_model_scopes.length === 0) {
      createForm.supported_model_scopes = getAntigravityDefaultModelScopes()
    }
    if (newVal !== 'gemini') {
      resetImageBatchConfig(createForm)
    }
  }
)

watch(
  () => editForm.platform,
  (newVal) => {
    if (newVal !== 'gemini') {
      editForm.gemini_mixed_protocol_enabled = false
    }
    if (newVal !== 'anthropic') {
      editForm.fallback_group_id_on_invalid_request = null
    }
    if (newVal !== 'openai') {
      editForm.image_protocol_mode = 'inherit'
      editForm.allow_messages_dispatch = false
      editForm.default_mapped_model = ''
    }
    if (!supportsOpenAIRuntimePolicy(newVal)) {
      editForm.allow_live = false
      editForm.max_reasoning_effort = ''
      editForm.max_reasoning_effort_over_limit = 'downgrade'
      editForm.reasoning_effort_mappings = []
    }
    if (newVal !== 'composite') {
      editForm.composite_routes = []
      resetEditCompositePreview()
    }
    if (!['openai', 'grok'].includes(newVal)) {
      editForm.web_search_price_per_call = null
    }
    if (newVal !== 'anthropic') {
      editForm.claude_code_only = false
      editForm.fallback_group_id = null
      editForm.model_routing_enabled = false
      editModelRoutingRules.value = []
    }
    if (newVal !== 'antigravity') {
      editForm.mcp_xml_inject = true
      editForm.supported_model_scopes = []
    } else if (editForm.supported_model_scopes.length === 0) {
      editForm.supported_model_scopes = getAntigravityDefaultModelScopes()
    }
    if (newVal !== 'gemini') {
      resetImageBatchConfig(editForm)
    }
  }
)

const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement

  if (!target.closest('.account-search-container')) {
    Object.keys(showAccountDropdown.value).forEach(key => {
      showAccountDropdown.value[key] = false
    })
  }
}


const openSortModal = async () => {
  try {

    const allGroups = await adminAPI.groups.getAll()
    sortableGroups.value = [...allGroups].sort((a, b) => a.sort_order - b.sort_order)
    showSortModal.value = true
  } catch (error) {
    appStore.showError(t('admin.groups.failedToLoad'))
    console.error('Error loading groups for sorting:', error)
  }
}


const closeSortModal = () => {
  showSortModal.value = false
  sortableGroups.value = []
}


const saveSortOrder = async () => {
  sortSubmitting.value = true
  try {
    const updates = sortableGroups.value.map((g, index) => ({
      id: g.id,
      sort_order: index * 10
    }))
    await adminAPI.groups.updateSortOrder(updates)
    appStore.showSuccess(t('admin.groups.sortOrderUpdated'))
    closeSortModal()
    loadGroups()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.groups.failedToUpdateSortOrder'))
    console.error('Error updating sort order:', error)
  } finally {
    sortSubmitting.value = false
  }
}
const groupsViewContext = {
  t,
  searchQuery,
  filters,
  platformFilterOptions,
  statusOptions,
  exclusiveOptions,
  platformOptions,
  editStatusOptions,
  subscriptionTypeOptions,
  openAIGroupImageProtocolModeOptions,
  fallbackGroupOptions,
  fallbackGroupOptionsForEdit,
  invalidRequestFallbackOptions,
  invalidRequestFallbackOptionsForEdit,
  copyAccountsGroupSelectOptions,
  copyAccountsGroupSelectOptionsForEdit,
  compositeTargetGroupOptions,
  compositeTargetGroupOptionsForEdit,
  isPlatformSelectOption,
  isGroupSelectOption,
  columns: visibleColumns,
  allColumns: columns,
  hiddenGroupColumns,
  alwaysVisibleGroupColumns: ALWAYS_VISIBLE_GROUP_COLUMNS,
  toggleGroupColumn,
  groups,
  loading,
  usageMap,
  usageLoading,
  capacityMap,
  pagination,
  showCreateModal,
  showEditModal,
  showDeleteDialog,
  showSortModal,
  submitting,
  sortSubmitting,
  editingGroup,
  deletingGroup,
  deleteConfirmMessage,
  showRateMultipliersModal,
  rateMultipliersGroup,
  sortableGroups,
  createCopyAccountsSelection,
  editCopyAccountsSelection,
  createCompositePreviewModel,
  createCompositePreviewLoading,
  createCompositePreviewResult,
  editCompositePreviewModel,
  editCompositePreviewLoading,
  editCompositePreviewResult,
  createForm,
  editForm,
  createModelRoutingRules,
  editModelRoutingRules,
  accountSearchKeyword,
  accountSearchResults,
  showAccountDropdown,
  loadGroups,
  handleSearch,
  openSortModal,
  closeSortModal,
  saveSortOrder,
  handleEdit,
  handleRateMultipliers,
  handleDuplicate,
  handleDelete,
  confirmDelete,
  closeCreateModal,
  closeEditModal,
  handleCreateGroup,
  handleUpdateGroup,
  handleCreatePlatformChange,
  handleCreateCopyAccountsSelect,
  handleEditCopyAccountsSelect,
  findGroupSelectOption,
  formatCost,
  getGroupAvailableAccounts,
  formatGroupAccountValue,
  formatGroupPeakRate,
  toggleCreateScope,
  toggleEditScope,
  getCreateRuleRenderKey,
  getEditRuleRenderKey,
  getCreateRuleSearchKey,
  getEditRuleSearchKey,
  getCreateReasoningMappingKey,
  getEditReasoningMappingKey,
  getCreateCompositeRouteKey,
  getEditCompositeRouteKey,
  searchAccountsByRule,
  onAccountSearchFocus,
  selectAccount,
  removeSelectedAccount,
  addCreateRoutingRule,
  removeCreateRoutingRule,
  addEditRoutingRule,
  removeEditRoutingRule,
  addCreateReasoningMapping,
  removeCreateReasoningMapping,
  addEditReasoningMapping,
  removeEditReasoningMapping,
  addCreateCompositeRoute,
  removeCreateCompositeRoute,
  addEditCompositeRoute,
  removeEditCompositeRoute,
  previewCreateCompositeRoute,
  previewEditCompositeRoute
}

onMounted(() => {
  loadGroups()
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  accountSearchRunner.clearAll()
  clearAllAccountSearchState()
})
</script>
