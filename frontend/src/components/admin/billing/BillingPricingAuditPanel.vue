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

      <div class="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
        <section class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="flex items-center justify-between gap-2">
            <div class="text-sm font-semibold text-gray-900 dark:text-white">重点问题模型</div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              按 冲突 → 缺价 → 回退 排序
            </div>
          </div>
          <div v-if="issueExamples.length === 0" class="mt-4 text-sm text-gray-500 dark:text-gray-400">
            当前没有需要优先处理的模型问题。
          </div>
          <div v-else class="mt-4 space-y-3">
            <article
              v-for="item in issueExamples"
              :key="item.model"
              data-testid="billing-audit-issue-card"
              class="rounded-2xl border px-4 py-3"
              :class="issueCardClass(item.pricing_status)"
            >
              <div class="flex flex-wrap items-center gap-2">
                <ModelIcon
                  :model="item.model"
                  :provider="item.provider"
                  :display-name="item.display_name"
                  size="18px"
                  class="shrink-0"
                />
                <div class="font-medium text-gray-900 dark:text-white">
                  {{ item.display_name || item.model }}
                </div>
                <span
                  data-testid="billing-audit-issue-status"
                  class="inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium"
                  :class="statusBadgeClass(item.pricing_status)"
                >
                  {{ statusLabel(item.pricing_status) }}
                </span>
              </div>
              <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ item.model }} <span v-if="item.provider">/ {{ formatProviderLabel(item.provider) }}</span>
              </div>
              <div v-if="item.first_warning" class="mt-2 text-xs leading-5" :class="issueTextClass(item.pricing_status)">
                {{ item.first_warning }}
              </div>
            </article>
          </div>
        </section>

        <section class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-sm font-semibold text-gray-900 dark:text-white">供应商问题榜</div>
          <div v-if="providerIssueCounts.length === 0" class="mt-4 text-sm text-gray-500 dark:text-gray-400">
            当前没有供应商级计费异常。
          </div>
          <div v-else class="mt-4 space-y-3">
            <div
              v-for="provider in providerIssueCounts"
              :key="provider.provider"
              data-testid="billing-audit-provider-card"
              class="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/60"
            >
              <div class="flex items-center justify-between gap-3">
                <div class="flex items-center gap-2 font-medium text-gray-900 dark:text-white">
                  <ModelPlatformIcon :platform="provider.provider" size="sm" class="shrink-0" />
                  <span>{{ formatProviderLabel(provider.provider) }}</span>
                </div>
                <div class="text-sm font-semibold text-rose-600 dark:text-rose-300">
                  {{ provider.total }}
                </div>
              </div>
              <div class="mt-3 flex flex-wrap gap-2 text-[11px]">
                <span class="inline-flex rounded-full bg-rose-100 px-2 py-1 text-rose-700 dark:bg-rose-500/15 dark:text-rose-200">
                  冲突 {{ provider.conflict }}
                </span>
                <span class="inline-flex rounded-full bg-rose-100 px-2 py-1 text-rose-700 dark:bg-rose-500/15 dark:text-rose-200">
                  缺价 {{ provider.missing }}
                </span>
                <span class="inline-flex rounded-full bg-amber-100 px-2 py-1 text-amber-700 dark:bg-amber-500/15 dark:text-amber-200">
                  回退 {{ provider.fallback }}
                </span>
              </div>
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
  BillingPricingIssueExample,
  BillingPricingProviderIssueCount,
  BillingPricingStatus,
} from '@/api/admin/billing'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'

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

const issueExamples = computed<BillingPricingIssueExample[]>(() => props.audit?.pricing_issue_examples || [])
const providerIssueCounts = computed<BillingPricingProviderIssueCount[]>(() => props.audit?.provider_issue_counts || [])

function statusLabel(status: BillingPricingStatus): string {
  switch (status) {
    case 'conflict':
      return '冲突'
    case 'missing':
      return '缺价'
    case 'fallback':
      return '回退'
    default:
      return '正常'
  }
}

function statusBadgeClass(status: BillingPricingStatus): string {
  switch (status) {
    case 'conflict':
    case 'missing':
      return 'bg-rose-100 text-rose-700 dark:bg-rose-500/15 dark:text-rose-200'
    case 'fallback':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-200'
    default:
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-200'
  }
}

function issueCardClass(status: BillingPricingStatus): string {
  switch (status) {
    case 'conflict':
    case 'missing':
      return 'border-rose-200 bg-rose-50/70 dark:border-rose-500/30 dark:bg-rose-500/10'
    case 'fallback':
      return 'border-amber-200 bg-amber-50/70 dark:border-amber-500/30 dark:bg-amber-500/10'
    default:
      return 'border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-900/60'
  }
}

function issueTextClass(status: BillingPricingStatus): string {
  switch (status) {
    case 'conflict':
    case 'missing':
      return 'text-rose-700 dark:text-rose-200'
    case 'fallback':
      return 'text-amber-700 dark:text-amber-200'
    default:
      return 'text-gray-600 dark:text-gray-300'
  }
}

function formatProviderLabel(provider?: string): string {
  const normalized = String(provider || '').trim().toLowerCase()
  switch (normalized) {
    case 'openai':
      return 'OpenAI'
    case 'anthropic':
      return 'Anthropic'
    case 'gemini':
      return 'Gemini'
    case 'grok':
      return 'Grok'
    case 'antigravity':
      return 'Antigravity'
    case 'baidu_document_ai':
      return 'Baidu Document AI'
    default:
      return normalized
        .split('_')
        .filter(Boolean)
        .map((item) => item.charAt(0).toUpperCase() + item.slice(1))
        .join(' ')
    }
}
</script>
