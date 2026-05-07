<template>
  <div class="space-y-6">
    <section class="rounded-3xl border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-800">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div class="inline-flex rounded-full bg-rose-600 px-3 py-1 text-xs font-semibold uppercase tracking-[0.16em] text-white">
            Billing Issues
          </div>
          <h2 class="mt-4 text-2xl font-semibold text-gray-900 dark:text-white">计费问题榜</h2>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-gray-600 dark:text-gray-300">
            聚焦冲突、缺价与回退模型，并按供应商聚合当前最值得优先处理的计费异常。
          </p>
        </div>
      </div>
    </section>

    <div class="rounded-3xl border border-gray-200 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-900/60">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div class="text-sm font-semibold text-gray-900 dark:text-white">问题榜快照</div>
          <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            当前模型目录 {{ audit?.total_models || 0 }} 个模型，最近快照：{{ snapshotUpdatedAtLabel }}
          </div>
        </div>
        <div
          v-if="audit?.refresh_required"
          class="rounded-full bg-amber-100 px-3 py-1 text-xs font-medium text-amber-800 dark:bg-amber-900/40 dark:text-amber-200"
        >
          需要刷新计费快照
        </div>
      </div>

      <div v-if="loading" class="mt-4 text-sm text-gray-500 dark:text-gray-400">
        正在加载问题榜...
      </div>

      <div v-else class="mt-4">
        <BillingPricingIssuesPanel :audit="audit" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { getBillingPricingAudit, type BillingPricingAudit } from '@/api/admin/billing'
import BillingPricingIssuesPanel from '@/components/admin/billing/BillingPricingIssuesPanel.vue'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'

const appStore = useAppStore()

const loading = ref(false)
const audit = ref<BillingPricingAudit | null>(null)

const snapshotUpdatedAtLabel = computed(() => {
  if (!audit.value?.snapshot_updated_at) {
    return '未刷新'
  }
  return formatDateTime(audit.value.snapshot_updated_at)
})

onMounted(async () => {
  loading.value = true
  try {
    audit.value = await getBillingPricingAudit()
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '加载计费问题榜失败'))
  } finally {
    loading.value = false
  }
})

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
