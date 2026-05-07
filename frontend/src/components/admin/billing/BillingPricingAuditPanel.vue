<template>
  <div class="mt-5 rounded-3xl border border-gray-200 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-900/60">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <div class="text-sm font-semibold text-gray-900 dark:text-white">计费审计</div>
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
      正在加载审计结果...
    </div>

    <div v-else class="mt-4 space-y-4">
      <div class="grid gap-4 xl:grid-cols-3">
        <section class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-sm font-semibold text-gray-900 dark:text-white">状态分布</div>
          <div class="mt-3 grid gap-3 sm:grid-cols-2">
            <div
              v-for="stat in statusStats"
              :key="stat.key"
              class="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/60"
            >
              <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                {{ stat.label }}
              </div>
              <div class="mt-2 text-2xl font-semibold" :class="stat.tone">
                {{ stat.value }}
              </div>
            </div>
          </div>
        </section>

        <section class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-sm font-semibold text-gray-900 dark:text-white">冲突来源</div>
          <div class="mt-3 space-y-3">
            <div
              v-for="stat in collisionStats"
              :key="stat.key"
              class="flex items-center justify-between rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 text-sm dark:border-dark-700 dark:bg-dark-900/60"
            >
              <span class="text-gray-600 dark:text-gray-300">{{ stat.label }}</span>
              <span class="font-semibold" :class="stat.tone">{{ stat.value }}</span>
            </div>
          </div>
        </section>

        <section class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-sm font-semibold text-gray-900 dark:text-white">快照健康度</div>
          <div class="mt-3 space-y-3">
            <div
              v-for="stat in snapshotStats"
              :key="stat.key"
              class="flex items-center justify-between rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 text-sm dark:border-dark-700 dark:bg-dark-900/60"
            >
              <span class="text-gray-600 dark:text-gray-300">{{ stat.label }}</span>
              <span class="font-semibold" :class="stat.tone">{{ stat.value }}</span>
            </div>
          </div>
        </section>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type {
  BillingPricingAudit,
} from '@/api/admin/billing'

const props = defineProps<{
  audit: BillingPricingAudit | null
  loading?: boolean
  snapshotUpdatedAtLabel: string
}>()

const statusStats = computed(() => {
  const counts = props.audit?.pricing_status_counts
  return [
    {
      key: 'ok',
      label: '正常',
      value: counts?.ok || 0,
      tone: 'text-emerald-600 dark:text-emerald-300',
    },
    {
      key: 'fallback',
      label: '回退',
      value: counts?.fallback || 0,
      tone: (counts?.fallback || 0) > 0 ? 'text-amber-600 dark:text-amber-300' : 'text-emerald-600 dark:text-emerald-300',
    },
    {
      key: 'conflict',
      label: '冲突',
      value: counts?.conflict || 0,
      tone: (counts?.conflict || 0) > 0 ? 'text-rose-600 dark:text-rose-300' : 'text-emerald-600 dark:text-emerald-300',
    },
    {
      key: 'missing',
      label: '缺价',
      value: counts?.missing || 0,
      tone: (counts?.missing || 0) > 0 ? 'text-rose-600 dark:text-rose-300' : 'text-emerald-600 dark:text-emerald-300',
    },
  ]
})

const collisionStats = computed(() => {
  const counts = props.audit?.collision_counts_by_source
  return [
    {
      key: 'aliases',
      label: 'Alias 冲突',
      value: counts?.aliases || 0,
      tone: (counts?.aliases || 0) > 0 ? 'text-rose-600 dark:text-rose-300' : 'text-emerald-600 dark:text-emerald-300',
    },
    {
      key: 'protocol_ids',
      label: '协议 ID 冲突',
      value: counts?.protocol_ids || 0,
      tone: (counts?.protocol_ids || 0) > 0 ? 'text-rose-600 dark:text-rose-300' : 'text-emerald-600 dark:text-emerald-300',
    },
    {
      key: 'pricing_lookup_ids',
      label: '共享价格源',
      value: counts?.pricing_lookup_ids || 0,
      tone: (counts?.pricing_lookup_ids || 0) > 0 ? 'text-amber-600 dark:text-amber-300' : 'text-emerald-600 dark:text-emerald-300',
    },
  ]
})

const snapshotStats = computed(() => [
  {
    key: 'duplicate',
    label: '主 ID 重复',
    value: props.audit?.duplicate_model_ids.length || 0,
    tone: (props.audit?.duplicate_model_ids.length || 0) > 0 ? 'text-rose-600 dark:text-rose-300' : 'text-emerald-600 dark:text-emerald-300',
  },
  {
    key: 'gap',
    label: '快照缺口',
    value: props.audit?.missing_in_snapshot_count || 0,
    tone: (props.audit?.missing_in_snapshot_count || 0) > 0 ? 'text-amber-600 dark:text-amber-300' : 'text-emerald-600 dark:text-emerald-300',
  },
  {
    key: 'snapshot-only',
    label: '仅快照模型',
    value: props.audit?.snapshot_only_count || 0,
    tone: (props.audit?.snapshot_only_count || 0) > 0 ? 'text-gray-700 dark:text-gray-200' : 'text-emerald-600 dark:text-emerald-300',
  },
])

</script>
