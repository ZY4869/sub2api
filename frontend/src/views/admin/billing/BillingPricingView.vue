<template>
  <div class="space-y-6">
    <section class="rounded-3xl border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-800">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div class="inline-flex rounded-full bg-primary-600 px-3 py-1 text-xs font-semibold uppercase tracking-[0.16em] text-white">Billing Center V2</div>
          <h2 class="mt-4 text-2xl font-semibold text-gray-900 dark:text-white">模型定价</h2>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-gray-600 dark:text-gray-300">计费中心现在独立于模型库。默认列表模式支持分页与每页数量切换，也可以切到九宫格供应商模式做批量巡检。</p>
        </div>
        <BillingPricingModeToggle v-model="viewMode" />
      </div>

      <div class="mt-5 grid gap-3 md:grid-cols-[minmax(0,1.5fr)_220px_220px]">
        <input v-model.trim="search" type="text" class="input" placeholder="搜索模型 / 供应商" @keyup.enter="applyFilters" />
        <select v-model="providerFilter" class="input" @change="applyFilters">
          <option value="">全部供应商</option>
          <option v-for="provider in providers" :key="provider.provider" :value="provider.provider">{{ provider.label }}</option>
        </select>
        <select v-model="modeFilter" class="input" @change="applyFilters">
          <option value="">全部模式</option>
          <option value="chat">chat</option>
          <option value="image">image</option>
          <option value="video">video</option>
          <option value="embedding">embedding</option>
        </select>
      </div>
    </section>

    <BillingPricingModelList
      v-if="viewMode === 'list'"
      :items="items"
      :total="total"
      :page="page"
      :page-size="pageSize"
      @open="openEditor"
      @update:page="updatePage"
      @update:page-size="updatePageSize"
    />

    <BillingPricingProviderGrid
      v-else
      :providers="providers"
      :provider-models="providerModels"
      :expanded-provider="expandedProvider"
      @toggle-provider="toggleProvider"
      @open-model="openEditor"
    />

    <BillingPricingEditorDialog
      :show="editorOpen"
      :details="editorDetails"
      :active-model="activeEditorModel"
      :busy="editorBusy"
      @close="editorOpen = false"
      @update:activeModel="activeEditorModel = $event"
      @save-layer="handleSaveLayer"
      @copy-official="handleCopyOfficial"
      @apply-discount="handleApplyDiscount"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  applyBillingPricingDiscount,
  copyBillingPricingOfficialToSale,
  getBillingPricingDetails,
  listBillingPricingModels,
  listBillingPricingProviders,
  updateBillingPricingLayer,
  type BillingPricingListItem,
  type BillingPricingProviderGroup,
  type BillingPricingSheetDetail,
} from '@/api/admin/billing'
import BillingPricingEditorDialog from '@/components/admin/billing/BillingPricingEditorDialog.vue'
import BillingPricingModeToggle from '@/components/admin/billing/BillingPricingModeToggle.vue'
import BillingPricingModelList from '@/components/admin/billing/BillingPricingModelList.vue'
import BillingPricingProviderGrid from '@/components/admin/billing/BillingPricingProviderGrid.vue'
import { useAppStore } from '@/stores/app'

const PAGE_SIZE_STORAGE_KEY = 'admin.billing.pricing.page_size'

const appStore = useAppStore()

const viewMode = ref<'list' | 'grid'>('list')
const search = ref('')
const providerFilter = ref('')
const modeFilter = ref('')
const page = ref(1)
const pageSize = ref(readPageSize())
const total = ref(0)
const items = ref<BillingPricingListItem[]>([])
const providers = ref<BillingPricingProviderGroup[]>([])
const providerModels = ref<Record<string, BillingPricingListItem[]>>({})
const expandedProvider = ref('')
const editorOpen = ref(false)
const editorBusy = ref(false)
const activeEditorModel = ref('')
const editorDetails = ref<BillingPricingSheetDetail[]>([])

onMounted(async () => {
  await guardedLoad(async () => {
    await Promise.all([reloadProviders(), reloadModels()])
  })
})

async function reloadProviders() {
  providers.value = await listBillingPricingProviders()
}

async function reloadModels() {
  const data = await listBillingPricingModels({
    search: search.value || undefined,
    provider: providerFilter.value || undefined,
    mode: modeFilter.value || undefined,
    page: page.value,
    page_size: pageSize.value,
  })
  items.value = data.items || []
  total.value = data.total || 0
}

async function applyFilters() {
  page.value = 1
  providerModels.value = {}
  expandedProvider.value = ''
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

async function toggleProvider(provider: string) {
  expandedProvider.value = expandedProvider.value === provider ? '' : provider
  if (!expandedProvider.value || providerModels.value[expandedProvider.value]) return
  try {
    const data = await listBillingPricingModels({
      provider: expandedProvider.value,
      page: 1,
      page_size: 100,
    })
    providerModels.value = {
      ...providerModels.value,
      [expandedProvider.value]: data.items || [],
    }
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '加载供应商模型失败'))
  }
}

async function openEditor(model: string) {
  editorBusy.value = true
  editorOpen.value = true
  try {
    const workset = dedupeModels([...editorDetails.value.map((detail) => detail.model), model])
    editorDetails.value = await getBillingPricingDetails(workset)
    activeEditorModel.value = model
  } catch (error) {
    editorOpen.value = false
    appStore.showError(resolveErrorMessage(error, '加载模型定价详情失败'))
  } finally {
    editorBusy.value = false
  }
}

async function handleSaveLayer(payload: { model: string; layer: 'official' | 'sale'; items: any[] }) {
  editorBusy.value = true
  try {
    const detail = await updateBillingPricingLayer(payload.model, payload.layer, { items: payload.items })
    mergeEditorDetail(detail)
    appStore.showSuccess(payload.layer === 'official' ? '官方价格已保存' : '出售价格已保存')
    await guardedLoad(reloadModels)
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
    details.forEach(mergeEditorDetail)
    appStore.showSuccess('已套用官方格式到出售价格')
    await guardedLoad(reloadModels)
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '复制官方格式失败'))
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
    details.forEach(mergeEditorDetail)
    appStore.showSuccess('出售价格折扣已应用')
    await guardedLoad(reloadModels)
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '应用折扣失败'))
  } finally {
    editorBusy.value = false
  }
}

function mergeEditorDetail(detail: BillingPricingSheetDetail) {
  const next = [...editorDetails.value]
  const index = next.findIndex((item) => item.model === detail.model)
  if (index >= 0) next[index] = detail
  else next.push(detail)
  editorDetails.value = next
}

async function guardedLoad(loader: () => Promise<void>) {
  try {
    await loader()
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '加载计费中心数据失败'))
  }
}

function dedupeModels(models: string[]): string[] {
  return Array.from(new Set(models.filter(Boolean)))
}

function readPageSize(): number {
  const raw = localStorage.getItem(PAGE_SIZE_STORAGE_KEY)
  const parsed = Number(raw)
  return parsed === 50 || parsed === 100 ? parsed : 20
}

function resolveErrorMessage(error: unknown, fallback: string): string {
  if (typeof error === 'object' && error && 'message' in error && typeof (error as { message?: unknown }).message === 'string') {
    return String((error as { message: string }).message)
  }
  return fallback
}
</script>
