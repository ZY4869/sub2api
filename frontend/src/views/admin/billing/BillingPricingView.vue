<template>
  <div class="space-y-6">
    <section class="rounded-3xl border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-800">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div class="inline-flex rounded-full bg-primary-600 px-3 py-1 text-xs font-semibold uppercase tracking-[0.16em] text-white">
            Billing Center V2
          </div>
          <h2 class="mt-4 text-2xl font-semibold text-gray-900 dark:text-white">模型定价</h2>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-gray-600 dark:text-gray-300">
            模型定价已切换到持久化快照。列表视图按模型编辑，九宫格视图按供应商直接打开工作集弹窗。
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="loading || refreshing"
            data-testid="billing-pricing-refresh"
            @click="handleManualRefresh"
          >
            {{ refreshing ? '刷新中...' : '手动刷新' }}
          </button>
          <BillingPricingModeToggle v-model="viewMode" />
        </div>
      </div>

      <BillingPricingAuditPanel
        :audit="audit"
        :loading="auditLoading"
        :snapshot-updated-at-label="snapshotUpdatedAtLabel"
      />

      <div class="mt-5 grid gap-3 xl:grid-cols-[minmax(0,1.4fr)_220px_180px_220px_240px]">
        <input
          v-model.trim="search"
          type="text"
          class="input"
          placeholder="搜索模型 / 供应商"
          @keyup.enter="applyFilters"
        />
        <select v-model="providerFilter" class="input" @change="applyFilters">
          <option value="">全部供应商</option>
          <option v-for="provider in providers" :key="provider.provider" :value="provider.provider">
            {{ provider.label }}
          </option>
        </select>
        <select v-model="modeFilter" class="input" @change="applyFilters">
          <option value="">全部模式</option>
          <option value="chat">chat</option>
          <option value="image">image</option>
          <option value="video">video</option>
          <option value="embedding">embedding</option>
        </select>
        <select
          v-model="sortMode"
          class="input"
          data-testid="billing-pricing-sort"
          @change="applySort"
        >
          <option value="display_name:asc">模型名 A-Z</option>
          <option value="display_name:desc">模型名 Z-A</option>
          <option value="provider:asc">供应商 A-Z</option>
          <option value="provider:desc">供应商 Z-A</option>
        </select>
        <select
          :value="groupPreviewId ?? ''"
          class="input"
          data-testid="billing-pricing-group-preview"
          @change="handleGroupPreviewChange(($event.target as HTMLSelectElement).value)"
        >
          <option value="">基础售价预览</option>
          <option v-for="group in previewGroups" :key="group.id" :value="group.id">
            {{ group.name }}
          </option>
        </select>
      </div>

      <div class="mt-3 rounded-2xl border border-sky-100 bg-sky-50 px-4 py-3 text-sm text-sky-800 dark:border-sky-900/60 dark:bg-sky-950/20 dark:text-sky-100">
        {{
          previewGroupName
            ? `当前按分组「${previewGroupName}」倍率预览有效售价，保存时仍只写入基础 sale price。`
            : '当前展示基础售价；选择分组后可预览该分组倍率作用后的有效售价。'
        }}
      </div>

      <div class="mt-4">
        <BillingPricingProviderQuickFilters
          :providers="providers"
          :active-provider="providerFilter"
          @select="handleQuickProviderSelect"
        />
      </div>
    </section>

    <BillingPricingModelList
      v-if="viewMode === 'list'"
      :items="items"
      :total="total"
      :page="page"
      :page-size="pageSize"
      :preview-group-name="previewGroupName"
      @open="openEditor"
      @update:page="updatePage"
      @update:page-size="updatePageSize"
    />

    <BillingPricingProviderGrid
      v-else
      :providers="visibleProviders"
      @open-provider="openProviderWorkset"
    />

    <BillingPricingEditorDialog
      :show="editorOpen"
      :details="editorDetails"
      :active-model="activeEditorModel"
      :busy="editorBusy"
      :preview-group-name="previewGroupName"
      @close="editorOpen = false"
      @update:activeModel="activeEditorModel = $event"
      @save-layer="handleSaveLayer"
      @copy-official="handleCopyOfficial"
      @apply-discount="handleApplyDiscount"
    />
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed, onMounted, ref, watch } from 'vue'
import {
  applyBillingPricingDiscount,
  copyBillingPricingOfficialToSale,
  getBillingPricingAudit,
  getBillingPricingDetails,
  getBillingPricingDetailsWithPreview,
  updateBillingPricingLayer,
  type BillingPricingAudit,
  type BillingPricingLayerForm,
  type BillingPricingSortBy,
  type BillingPricingSortOrder,
  type BillingPricingSheetDetail,
} from '@/api/admin/billing'
import { getAll as getAllGroups } from '@/api/admin/groups'
import BillingPricingAuditPanel from '@/components/admin/billing/BillingPricingAuditPanel.vue'
import BillingPricingEditorDialog from '@/components/admin/billing/BillingPricingEditorDialog.vue'
import BillingPricingModeToggle from '@/components/admin/billing/BillingPricingModeToggle.vue'
import BillingPricingModelList from '@/components/admin/billing/BillingPricingModelList.vue'
import BillingPricingProviderGrid from '@/components/admin/billing/BillingPricingProviderGrid.vue'
import BillingPricingProviderQuickFilters from '@/components/admin/billing/BillingPricingProviderQuickFilters.vue'
import { useBillingPricingStore } from '@/stores'
import { useAppStore } from '@/stores/app'
import type { AdminGroup } from '@/types'
import { formatDateTime } from '@/utils/format'

const PAGE_SIZE_STORAGE_KEY = 'admin.billing.pricing.page_size'

const appStore = useAppStore()
const billingPricingStore = useBillingPricingStore()

const {
  viewMode,
  search,
  providerFilter,
  modeFilter,
  groupPreviewId,
  sortBy,
  sortOrder,
  page,
  pageSize,
  total,
  items,
  providers,
  providerModels,
} = storeToRefs(billingPricingStore)

const editorOpen = ref(false)
const editorBusy = ref(false)
const loading = ref(false)
const refreshing = ref(false)
const auditLoading = ref(false)
const activeEditorModel = ref('')
const editorDetails = ref<BillingPricingSheetDetail[]>([])
const audit = ref<BillingPricingAudit | null>(null)
const previewGroups = ref<AdminGroup[]>([])

const sortMode = computed({
  get: () => `${sortBy.value}:${sortOrder.value}`,
  set: (value: string) => {
    const [nextSortBy, nextSortOrder] = value.split(':')
    if (
      (nextSortBy === 'display_name' || nextSortBy === 'provider')
      && (nextSortOrder === 'asc' || nextSortOrder === 'desc')
    ) {
      sortBy.value = nextSortBy as BillingPricingSortBy
      sortOrder.value = nextSortOrder as BillingPricingSortOrder
    }
  },
})

const visibleProviders = computed(() => {
  const filtered = providerFilter.value
    ? providers.value.filter((provider) => provider.provider === providerFilter.value)
    : [...providers.value]
  if (sortBy.value !== 'provider') {
    return filtered
  }
  return [...filtered].sort((left, right) => {
    const compared = left.label.localeCompare(right.label)
    if (compared === 0) {
      return left.provider.localeCompare(right.provider)
    }
    return sortOrder.value === 'desc' ? -compared : compared
  })
})

const previewGroupName = computed(() =>
  previewGroups.value.find((group) => group.id === groupPreviewId.value)?.name || '',
)

onMounted(async () => {
  await guardedLoad(async () => {
    await Promise.all([
      billingPricingStore.loadProviders(),
      billingPricingStore.loadModels(),
      loadAudit(),
      loadPreviewGroups(),
    ])
  })
})

watch(groupPreviewId, async () => {
  if (!editorOpen.value || editorDetails.value.length === 0) {
    return
  }
  await refreshEditorWorkset()
})

const snapshotUpdatedAtLabel = computed(() => {
  if (!audit.value?.snapshot_updated_at) {
    return '未刷新'
  }
  return formatDateTime(audit.value.snapshot_updated_at)
})

async function reloadProviders(force = false) {
  await billingPricingStore.loadProviders(force)
}

async function reloadModels(force = false) {
  await billingPricingStore.loadModels(force)
}

async function loadAudit() {
  auditLoading.value = true
  try {
    audit.value = await getBillingPricingAudit()
  } finally {
    auditLoading.value = false
  }
}

async function loadPreviewGroups() {
  previewGroups.value = await getAllGroups()
}

async function applyFilters() {
  page.value = 1
  await guardedLoad(reloadModels)
}

async function applySort() {
  page.value = 1
  await guardedLoad(reloadModels)
}

async function handleQuickProviderSelect(provider: string) {
  providerFilter.value = provider
  page.value = 1
  await guardedLoad(reloadModels)
}

async function updatePage(nextPage: number) {
  page.value = nextPage
  await guardedLoad(reloadModels)
}

async function updatePageSize(nextPageSize: number) {
  pageSize.value = nextPageSize
  page.value = 1
  localStorage.setItem(PAGE_SIZE_STORAGE_KEY, String(nextPageSize))
  await guardedLoad(reloadModels)
}

async function handleGroupPreviewChange(value: string) {
  const nextValue = value.trim()
  const parsed = Number(nextValue)
  groupPreviewId.value = nextValue && Number.isFinite(parsed) ? parsed : null
  page.value = 1
  await guardedLoad(reloadModels)
}

async function ensureProviderModelsLoaded(provider: string, force = false) {
  if (!provider) {
    return
  }
  await billingPricingStore.loadProviderModels(provider, force)
}

async function openEditor(model: string) {
  await openWorkset([model], model)
}

async function openProviderWorkset(provider: string) {
  editorBusy.value = true
  editorOpen.value = true
  try {
    await ensureProviderModelsLoaded(provider)
    const worksetModels = dedupeModels(
      (providerModels.value[provider] || []).map((item) => item.model),
    )
    const activeModel = worksetModels[0] || ''
    if (!activeModel) {
      editorOpen.value = false
      appStore.showError('当前供应商下没有可编辑的模型')
      return
    }
    await openWorkset(worksetModels, activeModel)
  } catch (error) {
    editorOpen.value = false
    editorBusy.value = false
    appStore.showError(resolveErrorMessage(error, '加载供应商模型定价详情失败'))
  }
}

async function openWorkset(models: string[], activeModel: string) {
  editorBusy.value = true
  editorOpen.value = true
  try {
    editorDetails.value = await loadWorksetDetails(dedupeModels(models))
    activeEditorModel.value = activeModel
  } catch (error) {
    editorOpen.value = false
    appStore.showError(resolveErrorMessage(error, '加载模型定价详情失败'))
  } finally {
    editorBusy.value = false
  }
}

async function handleSaveLayer(payload: {
  model: string
  layer: 'official' | 'sale'
  form: BillingPricingLayerForm
  currency: 'USD' | 'CNY'
}) {
  editorBusy.value = true
  try {
    const detail = await updateBillingPricingLayer(payload.model, payload.layer, {
      form: payload.form,
      currency: payload.currency,
      group_id: groupPreviewId.value,
    })
    mergeEditorDetail(detail)
    appStore.showSuccess(payload.layer === 'official' ? '官方价格已保存' : '出售价格已保存')
    billingPricingStore.invalidate()
    await guardedLoad(() => reloadAfterMutation(true))
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '保存模型定价失败'))
  } finally {
    editorBusy.value = false
  }
}

async function handleCopyOfficial(payload: { models: string[] }) {
  editorBusy.value = true
  try {
    const details = await copyBillingPricingOfficialToSale(payload)
    if (groupPreviewId.value) {
      await refreshEditorWorkset(payload.models)
    } else {
      details.forEach(mergeEditorDetail)
    }
    appStore.showSuccess('已将官方价格复制到出售价格')
    billingPricingStore.invalidate()
    await guardedLoad(() => reloadAfterMutation(true))
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '复制官方价格失败'))
  } finally {
    editorBusy.value = false
  }
}

async function handleApplyDiscount(payload: { models: string[]; itemIds?: string[]; discountRatio: number }) {
  editorBusy.value = true
  try {
    const details = await applyBillingPricingDiscount({
      models: payload.models,
      item_ids: payload.itemIds,
      discount_ratio: payload.discountRatio,
    })
    if (groupPreviewId.value) {
      await refreshEditorWorkset(payload.models)
    } else {
      details.forEach(mergeEditorDetail)
    }
    appStore.showSuccess('出售价格折扣已应用')
    billingPricingStore.invalidate()
    await guardedLoad(() => reloadAfterMutation(true))
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '应用折扣失败'))
  } finally {
    editorBusy.value = false
  }
}

async function handleManualRefresh() {
  refreshing.value = true
  try {
    const result = await billingPricingStore.refreshCatalog()
    await guardedLoad(async () => {
      await Promise.all([reloadAfterMutation(true), loadAudit()])
    })
    appStore.showSuccess(`模型列表已刷新，共 ${result.total_models} 个模型`)
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '刷新模型列表失败'))
  } finally {
    refreshing.value = false
  }
}

function mergeEditorDetail(detail: BillingPricingSheetDetail) {
  const next = [...editorDetails.value]
  const index = next.findIndex((item) => item.model === detail.model)
  if (index >= 0) {
    next[index] = detail
  } else {
    next.push(detail)
  }
  editorDetails.value = next
}

async function loadWorksetDetails(models: string[]): Promise<BillingPricingSheetDetail[]> {
  const normalizedModels = dedupeModels(models)
  if (!groupPreviewId.value) {
    return getBillingPricingDetails(normalizedModels)
  }
  return getBillingPricingDetailsWithPreview({
    models: normalizedModels,
    group_id: groupPreviewId.value,
  })
}

async function refreshEditorWorkset(models: string[] = editorDetails.value.map((detail) => detail.model)) {
  const normalizedModels = dedupeModels(models)
  if (normalizedModels.length === 0) {
    return
  }
  editorBusy.value = true
  try {
    editorDetails.value = await loadWorksetDetails(normalizedModels)
    if (!normalizedModels.includes(activeEditorModel.value)) {
      activeEditorModel.value = normalizedModels[0] || ''
    }
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '刷新倍率预览失败'))
  } finally {
    editorBusy.value = false
  }
}

async function reloadAfterMutation(force = false) {
  await Promise.all([reloadProviders(force), reloadModels(force)])
}

async function guardedLoad(loader: () => Promise<void>) {
  loading.value = true
  try {
    await loader()
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '加载计费中心数据失败'))
  } finally {
    loading.value = false
  }
}

function dedupeModels(models: string[]): string[] {
  return Array.from(new Set(models.filter(Boolean)))
}

function resolveErrorMessage(error: unknown, fallback: string): string {
  if (
    typeof error === 'object'
    && error
    && 'message' in error
    && typeof (error as { message?: unknown }).message === 'string'
  ) {
    return String((error as { message: string }).message)
  }
  return fallback
}
</script>
