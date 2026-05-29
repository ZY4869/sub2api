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
  GroupPlatform,
  OpenAIGroupImageProtocolMode,
  SubscriptionType
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

const columns = computed<Column[]>(() => [
  { key: 'name', label: t('admin.groups.columns.name'), sortable: true },
  { key: 'priority', label: t('admin.groups.columns.priority'), sortable: true },
  { key: 'platform', label: t('admin.groups.columns.platform'), sortable: true },
  { key: 'billing_type', label: t('admin.groups.columns.billingType'), sortable: true },
  { key: 'rate_multiplier', label: t('admin.groups.columns.rateMultiplier'), sortable: true },
  { key: 'is_exclusive', label: t('admin.groups.columns.type'), sortable: true },
  { key: 'account_count', label: t('admin.groups.columns.accounts'), sortable: true },
  { key: 'capacity', label: t('admin.groups.columns.capacity'), sortable: false },
  { key: 'usage', label: t('admin.groups.columns.usage'), sortable: false },
  { key: 'status', label: t('admin.groups.columns.status'), sortable: true },
  { key: 'actions', label: t('admin.groups.columns.actions'), sortable: false }
])

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

const createForm = reactive({
  name: '',
  description: '',
  platform: 'anthropic' as GroupPlatform,
  priority: 1,
  rate_multiplier: 1.0,
  is_exclusive: false,
  gemini_mixed_protocol_enabled: false,
  subscription_type: 'standard' as SubscriptionType,
  daily_limit_usd: null as number | null,
  weekly_limit_usd: null as number | null,
  monthly_limit_usd: null as number | null,
  image_price_1k: null as number | null,
  image_price_2k: null as number | null,
  image_price_4k: null as number | null,
  image_protocol_mode: 'inherit' as OpenAIGroupImageProtocolMode,
  claude_code_only: false,
  fallback_group_id: null as number | null,
  fallback_group_id_on_invalid_request: null as number | null,
  allow_messages_dispatch: false,
  default_mapped_model: 'gpt-5.4',
  visible_model_patterns_text: '',
  model_routing_enabled: false,
  supported_model_scopes: ['claude', 'gemini_text', 'gemini_image'] as string[],
  mcp_xml_inject: true,
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

const getCreateRuleRenderKey = (rule: ModelRoutingRule) => resolveCreateRuleKey(rule)
const getEditRuleRenderKey = (rule: ModelRoutingRule) => resolveEditRuleKey(rule)

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

const editForm = reactive({
  name: '',
  description: '',
  platform: 'anthropic' as GroupPlatform,
  priority: 1,
  rate_multiplier: 1.0,
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
  image_protocol_mode: 'inherit' as OpenAIGroupImageProtocolMode,
  claude_code_only: false,
  fallback_group_id: null as number | null,
  fallback_group_id_on_invalid_request: null as number | null,
  allow_messages_dispatch: false,
  default_mapped_model: '',
  visible_model_patterns_text: '',
  model_routing_enabled: false,
  supported_model_scopes: ['claude', 'gemini_text', 'gemini_image'] as string[],
  mcp_xml_inject: true,
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

const handleCreatePlatformChange = () => {
  createForm.copy_accounts_from_group_ids = []
  createCopyAccountsSelection.value = null
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
  createForm.is_exclusive = false
  createForm.gemini_mixed_protocol_enabled = false
  createForm.subscription_type = 'standard'
  createForm.daily_limit_usd = null
  createForm.weekly_limit_usd = null
  createForm.monthly_limit_usd = null
  createForm.image_price_1k = null
  createForm.image_price_2k = null
  createForm.image_price_4k = null
  createForm.image_protocol_mode = 'inherit'
  createForm.claude_code_only = false
  createForm.fallback_group_id = null
  createForm.fallback_group_id_on_invalid_request = null
  createForm.allow_messages_dispatch = false
  createForm.default_mapped_model = 'gpt-5.4'
  createForm.visible_model_patterns_text = ''
  createForm.supported_model_scopes = ['claude', 'gemini_text', 'gemini_image']
  createForm.mcp_xml_inject = true
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

const handleCreateGroup = async () => {
  if (!createForm.name.trim()) {
    appStore.showError(t('admin.groups.nameRequired'))
    return
  }
  submitting.value = true
  try {
    const requestData = {
      ...createForm,
      priority: normalizeGroupPriority(createForm.priority),
      daily_limit_usd: normalizeOptionalLimit(createForm.daily_limit_usd as number | string | null),
      weekly_limit_usd: normalizeOptionalLimit(createForm.weekly_limit_usd as number | string | null),
      monthly_limit_usd: normalizeOptionalLimit(createForm.monthly_limit_usd as number | string | null),
      visible_model_patterns: parseModelPatternText(createForm.visible_model_patterns_text),
      model_routing: convertRoutingRulesToApiFormat(createModelRoutingRules.value)
    }

    const emptyToNull = (v: any) => v === '' ?
 null : v
    requestData.daily_limit_usd = emptyToNull(requestData.daily_limit_usd)
    requestData.weekly_limit_usd = emptyToNull(requestData.weekly_limit_usd)
    requestData.monthly_limit_usd = emptyToNull(requestData.monthly_limit_usd)
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
  editForm.image_protocol_mode = normalizeOpenAIGroupImageProtocolMode(group.image_protocol_mode)
  editForm.claude_code_only = group.claude_code_only || false
  editForm.fallback_group_id = group.fallback_group_id
  editForm.fallback_group_id_on_invalid_request = group.fallback_group_id_on_invalid_request
  editForm.allow_messages_dispatch = group.allow_messages_dispatch || false
  editForm.default_mapped_model = group.default_mapped_model || ''
  editForm.visible_model_patterns_text = joinModelPatternText(group.visible_model_patterns)
  editForm.model_routing_enabled = group.model_routing_enabled || false
  editForm.supported_model_scopes = group.supported_model_scopes || ['claude', 'gemini_text', 'gemini_image']
  editForm.mcp_xml_inject = group.mcp_xml_inject ?? true
  editForm.copy_accounts_from_group_ids = []
  editModelRoutingRules.value = await convertApiFormatToRoutingRules(group.model_routing)
  editCopyAccountsSelection.value = null
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
  editForm.image_protocol_mode = 'inherit'
  editForm.visible_model_patterns_text = ''
}

const handleUpdateGroup = async () => {
  if (!editingGroup.value) return
  if (!editForm.name.trim()) {
    appStore.showError(t('admin.groups.nameRequired'))
    return
  }

  submitting.value = true
  try {
    const payload = {
      ...editForm,
      priority: normalizeGroupPriority(editForm.priority),
      daily_limit_usd: normalizeOptionalLimit(editForm.daily_limit_usd as number | string | null),
      weekly_limit_usd: normalizeOptionalLimit(editForm.weekly_limit_usd as number | string | null),
      monthly_limit_usd: normalizeOptionalLimit(editForm.monthly_limit_usd as number | string | null),
      fallback_group_id: editForm.fallback_group_id === null ? 0 : editForm.fallback_group_id,
      fallback_group_id_on_invalid_request:
        editForm.fallback_group_id_on_invalid_request === null
          ? 0
          : editForm.fallback_group_id_on_invalid_request,
      visible_model_patterns: parseModelPatternText(editForm.visible_model_patterns_text),
      model_routing: convertRoutingRulesToApiFormat(editModelRoutingRules.value)
    }

    const emptyToNull = (v: any) => v === '' ?
 null : v
    payload.daily_limit_usd = emptyToNull(payload.daily_limit_usd)
    payload.weekly_limit_usd = emptyToNull(payload.weekly_limit_usd)
    payload.monthly_limit_usd = emptyToNull(payload.monthly_limit_usd)
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
    }
  }
)

watch(
  () => createForm.platform,
  (newVal) => {
    if (newVal !== 'gemini') {
      createForm.gemini_mixed_protocol_enabled = false
    }
    if (!['anthropic', 'antigravity'].includes(newVal)) {
      createForm.fallback_group_id_on_invalid_request = null
    }
    if (newVal !== 'openai') {
      createForm.image_protocol_mode = 'inherit'
      createForm.allow_messages_dispatch = false
      createForm.default_mapped_model = ''
    }
  }
)

watch(
  () => editForm.platform,
  (newVal) => {
    if (newVal !== 'gemini') {
      editForm.gemini_mixed_protocol_enabled = false
    }
    if (newVal !== 'openai') {
      editForm.image_protocol_mode = 'inherit'
      editForm.allow_messages_dispatch = false
      editForm.default_mapped_model = ''
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
  isPlatformSelectOption,
  isGroupSelectOption,
  columns,
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
  toggleCreateScope,
  toggleEditScope,
  getCreateRuleRenderKey,
  getEditRuleRenderKey,
  getCreateRuleSearchKey,
  getEditRuleSearchKey,
  searchAccountsByRule,
  onAccountSearchFocus,
  selectAccount,
  removeSelectedAccount,
  addCreateRoutingRule,
  removeCreateRoutingRule,
  addEditRoutingRule,
  removeEditRoutingRule
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
