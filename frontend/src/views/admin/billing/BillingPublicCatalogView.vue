<template>
  <div class="space-y-6">
    <section class="rounded-3xl border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-800">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div class="max-w-3xl">
          <div class="inline-flex rounded-full bg-emerald-600 px-3 py-1 text-xs font-semibold uppercase tracking-[0.16em] text-white">
            Publish Snapshot
          </div>
          <h2 class="mt-4 text-2xl font-semibold text-gray-900 dark:text-white">对外模型展示</h2>
          <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-300">
            先在这里维护草稿，再手动“推送更新”生成对外发布快照。公开模型库只会读取已发布版本，未推送前前台不会变化。
          </p>
        </div>

        <div class="flex flex-wrap items-center gap-3">
          <button type="button" class="btn btn-secondary" :disabled="loading || saving || publishing" @click="loadDraft">
            {{ loading ? '加载中...' : '重新加载' }}
          </button>
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="loading || saving || publishing"
            data-testid="billing-public-catalog-save"
            @click="saveDraftAction"
          >
            {{ saving ? '保存中...' : '保存草稿' }}
          </button>
          <button
            type="button"
            class="btn btn-primary"
            :disabled="loading || publishing || selectedModels.length === 0"
            data-testid="billing-public-catalog-publish"
            @click="publishAction"
          >
            {{ publishing ? '推送中...' : '推送更新' }}
          </button>
        </div>
      </div>

      <div class="mt-5 grid gap-4 lg:grid-cols-3">
        <div class="rounded-2xl border border-gray-200 bg-gray-50/70 px-4 py-4 dark:border-dark-700 dark:bg-dark-900/40">
          <div class="text-xs font-medium uppercase tracking-[0.18em] text-gray-500 dark:text-gray-400">草稿</div>
          <div class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">{{ selectedModels.length }} 个模型</div>
          <div class="mt-1 text-sm text-gray-600 dark:text-gray-300">每页 {{ pageSize }} 条</div>
          <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">最近保存：{{ formatTimestamp(draftUpdatedAt) }}</div>
        </div>
        <div class="rounded-2xl border border-gray-200 bg-gray-50/70 px-4 py-4 dark:border-dark-700 dark:bg-dark-900/40">
          <div class="text-xs font-medium uppercase tracking-[0.18em] text-gray-500 dark:text-gray-400">已发布</div>
          <div class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">{{ published?.model_count ?? 0 }} 个模型</div>
          <div class="mt-1 text-sm text-gray-600 dark:text-gray-300">每页 {{ published?.page_size ?? 10 }} 条</div>
          <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">最近推送：{{ formatTimestamp(published?.updated_at) }}</div>
        </div>
        <div class="rounded-2xl border border-sky-200 bg-sky-50 px-4 py-4 dark:border-sky-900/60 dark:bg-sky-950/20">
          <div class="text-xs font-medium uppercase tracking-[0.18em] text-sky-700 dark:text-sky-200">发布语义</div>
          <div class="mt-2 text-sm leading-6 text-sky-800 dark:text-sky-100">
            推送时会同时冻结列表排序、展示名称、分页大小与模型详情调用示例。公开模型库不会再按 TTL 自动重建。
          </div>
        </div>
      </div>
    </section>

    <div class="grid gap-6 xl:grid-cols-[minmax(360px,0.9fr)_minmax(0,1.1fr)]">
      <section class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">已选展示模型</h3>
            <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">这里的顺序就是公开模型库的发布顺序。</p>
          </div>
          <div class="w-full max-w-[180px]">
            <label class="mb-1 block text-xs font-medium uppercase tracking-[0.16em] text-gray-500 dark:text-gray-400">每页数量</label>
            <input v-model.number="pageSizeInput" type="number" min="1" max="100" class="input" data-testid="billing-public-catalog-page-size" />
          </div>
        </div>

        <div class="mt-4 flex items-center justify-between gap-2 text-sm text-gray-500 dark:text-gray-400">
          <span>已选 {{ selectedModels.length }} 个</span>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="selectedModels.length === 0" @click="clearSelection">清空</button>
        </div>

        <div class="mt-4 space-y-3">
          <div
            v-for="(entry, index) in orderedSelectedEntries"
            :key="entry.model"
            class="rounded-2xl border px-4 py-3"
            :class="entry.missing ? 'border-amber-300 bg-amber-50 dark:border-amber-900/60 dark:bg-amber-950/20' : 'border-gray-200 bg-gray-50/70 dark:border-dark-700 dark:bg-dark-900/40'"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="flex items-center gap-3">
                  <ModelIcon :model="entry.model" :provider="entry.provider" :display-name="entry.display_name" size="20px" />
                  <div class="min-w-0">
                    <div class="truncate font-medium text-gray-900 dark:text-white">{{ entry.display_name || entry.model }}</div>
                    <div class="truncate text-xs text-gray-500 dark:text-gray-400">{{ entry.model }}</div>
                  </div>
                </div>
                <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                  {{ entry.missing ? '当前模型已不在最新可发布集合中，推送时会被自动跳过。' : `${entry.provider || '-'} / ${entry.mode || '-'}` }}
                </div>
              </div>

              <div class="flex shrink-0 flex-col gap-2">
                <button type="button" class="btn btn-secondary btn-sm" :disabled="index === 0" @click="moveSelection(index, -1)">上移</button>
                <button type="button" class="btn btn-secondary btn-sm" :disabled="index === orderedSelectedEntries.length - 1" @click="moveSelection(index, 1)">下移</button>
                <button type="button" class="btn btn-danger btn-sm" @click="removeSelection(entry.model)">移除</button>
              </div>
            </div>
          </div>

          <div v-if="orderedSelectedEntries.length === 0" class="rounded-2xl border border-dashed border-gray-300 px-4 py-10 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
            还没有选择任何对外展示模型。
          </div>
        </div>
      </section>

      <section class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">可发布模型集合</h3>
            <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">来源于当前可售卖的实时公共目录，用于构建下一次发布快照。</p>
          </div>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="filteredAvailableItems.length === 0" @click="selectFilteredItems">选中当前筛选结果</button>
        </div>

        <input v-model.trim="search" type="search" class="input mt-4" placeholder="搜索展示名 / 模型 ID / 供应商" data-testid="billing-public-catalog-search" />

        <div class="mt-4 max-h-[760px] space-y-3 overflow-y-auto pr-1">
          <label
            v-for="item in filteredAvailableItems"
            :key="item.model"
            class="flex cursor-pointer items-start gap-3 rounded-2xl border border-gray-200 bg-gray-50/70 px-4 py-3 transition hover:border-primary-300 dark:border-dark-700 dark:bg-dark-900/40"
          >
            <input :checked="isSelected(item.model)" type="checkbox" class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" @change="toggleSelection(item.model)" />
            <ModelIcon :model="item.model" :provider="item.provider" :display-name="item.display_name" size="20px" />
            <div class="min-w-0 flex-1">
              <div class="truncate font-medium text-gray-900 dark:text-white">{{ item.display_name || item.model }}</div>
              <div class="truncate text-xs text-gray-500 dark:text-gray-400">{{ item.model }}</div>
              <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                {{ item.provider || '-' }} / {{ item.mode || '-' }} / {{ item.request_protocols?.join(' · ') || '-' }}
              </div>
            </div>
          </label>

          <div v-if="filteredAvailableItems.length === 0" class="rounded-2xl border border-dashed border-gray-300 px-4 py-10 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
            当前筛选下没有可发布模型。
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  getBillingPublicCatalogDraft,
  publishBillingPublicCatalog,
  saveBillingPublicCatalogDraft,
  type BillingPublicCatalogDraft,
  type BillingPublicCatalogPublishedSummary,
} from '@/api/admin/billing'
import type { PublicModelCatalogItem } from '@/api/meta'
import ModelIcon from '@/components/common/ModelIcon.vue'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'

type SelectedCatalogEntry = PublicModelCatalogItem & { missing?: boolean }

const appStore = useAppStore()

const loading = ref(false)
const saving = ref(false)
const publishing = ref(false)
const search = ref('')
const selectedModels = ref<string[]>([])
const pageSize = ref(10)
const draftUpdatedAt = ref('')
const availableItems = ref<PublicModelCatalogItem[]>([])
const published = ref<BillingPublicCatalogPublishedSummary | null>(null)

const availableItemMap = computed(() => {
  const next = new Map<string, PublicModelCatalogItem>()
  availableItems.value.forEach((item) => next.set(item.model, item))
  return next
})

const orderedSelectedEntries = computed<SelectedCatalogEntry[]>(() =>
  selectedModels.value.map((model) => {
    const item = availableItemMap.value.get(model)
    if (item) {
      return item
    }
    return {
      model,
      display_name: model,
      currency: 'USD',
      price_display: { primary: [] },
      multiplier_summary: { enabled: false, kind: 'disabled' },
      missing: true,
    }
  }),
)

const filteredAvailableItems = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) {
    return availableItems.value
  }
  return availableItems.value.filter((item) =>
    [item.display_name, item.model, item.provider].some((value) =>
      String(value || '').toLowerCase().includes(keyword),
    ),
  )
})

const pageSizeInput = computed({
  get: () => pageSize.value,
  set: (value: number) => {
    pageSize.value = normalizePageSize(value)
  },
})

onMounted(async () => {
  await loadDraft()
})

async function loadDraft() {
  loading.value = true
  try {
    const payload = await getBillingPublicCatalogDraft()
    availableItems.value = payload.available_items || []
    selectedModels.value = normalizeSelectedModels(payload.draft?.selected_models || [])
    pageSize.value = normalizePageSize(payload.draft?.page_size || 10)
    draftUpdatedAt.value = payload.draft?.updated_at || ''
    published.value = payload.published || null
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '加载对外模型展示草稿失败'))
  } finally {
    loading.value = false
  }
}

async function saveDraftAction() {
  saving.value = true
  try {
    const result = await saveBillingPublicCatalogDraft(buildDraftPayload())
    selectedModels.value = normalizeSelectedModels(result.selected_models || [])
    pageSize.value = normalizePageSize(result.page_size || 10)
    draftUpdatedAt.value = result.updated_at || draftUpdatedAt.value
    appStore.showSuccess('对外模型展示草稿已保存')
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '保存对外模型展示草稿失败'))
  } finally {
    saving.value = false
  }
}

async function publishAction() {
  publishing.value = true
  try {
    published.value = await publishBillingPublicCatalog(buildDraftPayload())
    draftUpdatedAt.value = published.value?.updated_at || draftUpdatedAt.value
    appStore.showSuccess('公开模型库已推送更新')
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '推送公开模型库失败'))
  } finally {
    publishing.value = false
  }
}

function toggleSelection(model: string) {
  selectedModels.value = isSelected(model)
    ? selectedModels.value.filter((item) => item !== model)
    : [...selectedModels.value, model]
}

function isSelected(model: string): boolean {
  return selectedModels.value.includes(model)
}

function removeSelection(model: string) {
  selectedModels.value = selectedModels.value.filter((item) => item !== model)
}

function clearSelection() {
  selectedModels.value = []
}

function selectFilteredItems() {
  const merged = new Set(selectedModels.value)
  filteredAvailableItems.value.forEach((item) => merged.add(item.model))
  selectedModels.value = Array.from(merged)
}

function moveSelection(index: number, delta: number) {
  const nextIndex = index + delta
  if (nextIndex < 0 || nextIndex >= selectedModels.value.length) {
    return
  }
  const next = [...selectedModels.value]
  const [target] = next.splice(index, 1)
  next.splice(nextIndex, 0, target)
  selectedModels.value = next
}

function buildDraftPayload(): BillingPublicCatalogDraft {
  return {
    selected_models: normalizeSelectedModels(selectedModels.value),
    page_size: normalizePageSize(pageSize.value),
    updated_at: draftUpdatedAt.value,
  }
}

function normalizeSelectedModels(models: string[]): string[] {
  return Array.from(new Set(models.map((model) => String(model || '').trim()).filter(Boolean)))
}

function normalizePageSize(value: number): number {
  if (!Number.isFinite(value) || value <= 0) {
    return 10
  }
  return Math.min(100, Math.max(1, Math.round(value)))
}

function formatTimestamp(value?: string | null): string {
  if (!value) {
    return '未保存'
  }
  return formatDateTime(value)
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
