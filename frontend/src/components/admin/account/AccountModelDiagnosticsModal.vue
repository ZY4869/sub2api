<template>
  <BaseDialog
    :show="show"
    :title="dialogTitle"
    width="wide"
    @close="emit('close')"
  >
    <div class="space-y-5">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div class="space-y-2">
          <div class="flex flex-wrap items-center gap-2">
            <span
              class="inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold"
              :class="statusToneClass"
            >
              {{ statusLabel }}
            </span>
            <span class="text-xs text-slate-500 dark:text-slate-400">
              ID #{{ result?.account_id ?? account?.id ?? '-' }}
            </span>
          </div>
          <p class="text-sm text-slate-500 dark:text-slate-400">
            {{ account?.name || emptyLabel }}
          </p>
        </div>
        <button
          type="button"
          class="btn btn-secondary btn-sm self-start"
          :disabled="loading || !account"
          @click="emit('refresh')"
        >
          {{ loading ? t('admin.accounts.modelDiagnostics.refreshing') : t('admin.accounts.modelDiagnostics.refresh') }}
        </button>
      </div>

      <div
        v-if="show && loading && !result"
        class="rounded-xl border border-slate-200 bg-slate-50 px-4 py-5 text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900/40 dark:text-slate-300"
      >
        {{ t('admin.accounts.modelDiagnostics.loading') }}
      </div>

      <div v-else-if="result" class="space-y-5">
        <div class="grid gap-4 lg:grid-cols-2">
          <section class="rounded-xl border border-slate-200 bg-slate-50/70 p-4 dark:border-slate-700 dark:bg-slate-900/40">
            <h3 class="text-sm font-semibold text-slate-900 dark:text-slate-100">
              {{ t('admin.accounts.modelDiagnostics.summaryTitle') }}
            </h3>
            <dl class="mt-3 space-y-3 text-sm">
              <div v-for="item in summaryRows" :key="item.label">
                <dt class="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">{{ item.label }}</dt>
                <dd class="mt-1 break-all text-slate-900 dark:text-slate-100">{{ item.value || emptyLabel }}</dd>
              </div>
            </dl>
          </section>

          <section class="rounded-xl border border-slate-200 bg-slate-50/70 p-4 dark:border-slate-700 dark:bg-slate-900/40">
            <h3 class="text-sm font-semibold text-slate-900 dark:text-slate-100">
              {{ t('admin.accounts.modelDiagnostics.modelsTitle') }}
            </h3>
            <div class="mt-3 space-y-4">
              <div v-for="section in modelSections" :key="section.key" class="space-y-2">
                <div class="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">
                  {{ section.label }}
                </div>
                <div v-if="section.items.length" class="flex flex-wrap gap-2">
                  <span
                    v-for="item in section.items"
                    :key="item.key"
                    class="rounded-full border border-slate-200 bg-white px-3 py-1 text-xs text-slate-700 dark:border-slate-700 dark:bg-slate-950/50 dark:text-slate-200"
                    :title="item.detail || item.label"
                  >
                    {{ item.label }}
                  </span>
                </div>
                <p v-else class="text-sm text-slate-500 dark:text-slate-400">{{ emptyLabel }}</p>
              </div>
            </div>
          </section>
        </div>

        <section class="rounded-xl border border-slate-200 bg-slate-50/70 p-4 dark:border-slate-700 dark:bg-slate-900/40">
          <h3 class="text-sm font-semibold text-slate-900 dark:text-slate-100">
            {{ t('admin.accounts.modelDiagnostics.warnings') }}
          </h3>
          <ul v-if="warnings.length" class="mt-3 space-y-2 text-sm text-amber-700 dark:text-amber-300">
            <li v-for="warning in warnings" :key="warning">{{ warning }}</li>
          </ul>
          <p v-else class="mt-3 text-sm text-slate-500 dark:text-slate-400">{{ t('admin.accounts.modelDiagnostics.noWarnings') }}</p>
        </section>

        <section class="space-y-3">
          <div class="flex items-center justify-between gap-3">
            <h3 class="text-sm font-semibold text-slate-900 dark:text-slate-100">
              {{ t('admin.accounts.modelDiagnostics.groupExposures') }}
            </h3>
            <span class="text-xs text-slate-500 dark:text-slate-400">
              {{ groupExposures.length }}
            </span>
          </div>
          <div v-if="groupExposures.length" class="space-y-3">
            <article
              v-for="group in groupExposures"
              :key="group.group_id"
              class="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-900/30"
            >
              <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                <div>
                  <h4 class="text-sm font-semibold text-slate-900 dark:text-slate-100">
                    {{ group.group_name || `#${group.group_id}` }}
                  </h4>
                  <p class="text-xs text-slate-500 dark:text-slate-400">
                    {{ group.group_platform || emptyLabel }}
                  </p>
                </div>
              </div>

              <div class="mt-3">
                <div class="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">
                  {{ t('admin.accounts.modelDiagnostics.groupPublicModels') }}
                </div>
                <div v-if="group.public_models.length" class="mt-2 flex flex-wrap gap-2">
                  <span
                    v-for="item in previewToChips(group.public_models)"
                    :key="item.key"
                    class="rounded-full border border-slate-200 bg-slate-50 px-3 py-1 text-xs text-slate-700 dark:border-slate-700 dark:bg-slate-950/50 dark:text-slate-200"
                    :title="item.detail || item.label"
                  >
                    {{ item.label }}
                  </span>
                </div>
                <p v-else class="mt-2 text-sm text-slate-500 dark:text-slate-400">{{ emptyLabel }}</p>
              </div>

              <ul v-if="group.warnings?.length" class="mt-3 space-y-1 text-sm text-amber-700 dark:text-amber-300">
                <li v-for="warning in group.warnings" :key="warning">{{ warning }}</li>
              </ul>

              <div class="mt-4 space-y-3">
                <div class="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">
                  {{ t('admin.accounts.modelDiagnostics.apiKeys') }}
                </div>
                <div v-if="group.api_keys.length" class="space-y-3">
                  <div
                    v-for="apiKey in group.api_keys"
                    :key="apiKey.api_key_id"
                    class="rounded-lg border border-slate-200 bg-slate-50/70 p-3 dark:border-slate-700 dark:bg-slate-950/40"
                  >
                    <div class="text-sm font-medium text-slate-900 dark:text-slate-100">
                      {{ apiKey.api_key_name || `#${apiKey.api_key_id}` }}
                    </div>
                    <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
                      {{ t('admin.accounts.modelDiagnostics.modelDisplayMode') }}: {{ apiKey.model_display_mode || emptyLabel }}
                    </p>
                    <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
                      {{ t('admin.accounts.modelDiagnostics.modelPatterns') }}:
                      {{ apiKey.model_patterns?.length ? apiKey.model_patterns.join(', ') : emptyLabel }}
                    </p>
                    <div v-if="apiKey.public_models.length" class="mt-2 flex flex-wrap gap-2">
                      <span
                        v-for="item in previewToChips(apiKey.public_models)"
                        :key="item.key"
                        class="rounded-full border border-slate-200 bg-white px-3 py-1 text-xs text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200"
                        :title="item.detail || item.label"
                      >
                        {{ item.label }}
                      </span>
                    </div>
                    <p v-else class="mt-2 text-sm text-slate-500 dark:text-slate-400">{{ emptyLabel }}</p>
                  </div>
                </div>
                <p v-else class="text-sm text-slate-500 dark:text-slate-400">{{ t('admin.accounts.modelDiagnostics.noApiKeys') }}</p>
              </div>
            </article>
          </div>
          <p v-else class="rounded-xl border border-slate-200 bg-slate-50 px-4 py-5 text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900/40 dark:text-slate-300">
            {{ emptyLabel }}
          </p>
        </section>
      </div>

      <div
        v-else
        class="rounded-xl border border-slate-200 bg-slate-50 px-4 py-5 text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900/40 dark:text-slate-300"
      >
        {{ t('admin.accounts.modelDiagnostics.emptyState') }}
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type {
  AccountModelDiagnosticsPreview,
  AccountModelDiagnosticsResponse
} from '@/api/admin/accounts'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { Account } from '@/types'

interface DiagnosticsChip {
  key: string
  label: string
  detail?: string
}

const props = defineProps<{
  show: boolean
  account: Account | null
  result: AccountModelDiagnosticsResponse | null
  loading: boolean
}>()

const emit = defineEmits<{
  close: []
  refresh: []
}>()

const { t } = useI18n()

const emptyLabel = computed(() => t('admin.accounts.modelDiagnostics.empty'))
const dialogTitle = computed(() =>
  t('admin.accounts.modelDiagnostics.title', { name: props.account?.name || '' })
)
const warnings = computed(() => props.result?.warnings || [])
const groupExposures = computed(() => props.result?.group_exposures || [])
const statusLabel = computed(() =>
  t(`admin.accounts.modelDiagnostics.statuses.${props.result?.status || 'probe_failed_empty'}`)
)
const statusToneClass = computed(() => {
  switch (props.result?.status) {
    case 'ok':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
    case 'degraded':
    case 'fallback_only':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
    case 'filtered_empty':
    case 'probe_failed_empty':
    default:
      return 'bg-rose-100 text-rose-700 dark:bg-rose-500/15 dark:text-rose-300'
  }
})

const summaryRows = computed(() => ([
  { label: t('admin.accounts.modelDiagnostics.routingPlatform'), value: props.result?.routing_platform || '' },
  { label: t('admin.accounts.modelDiagnostics.probeSource'), value: props.result?.probe_source || '' },
  { label: t('admin.accounts.modelDiagnostics.probeNotice'), value: props.result?.probe_notice || '' },
  { label: t('admin.accounts.modelDiagnostics.resolvedUpstreamUrl'), value: props.result?.resolved_upstream_url || '' },
  { label: t('admin.accounts.modelDiagnostics.resolvedUpstreamHost'), value: props.result?.resolved_upstream_host || '' },
  { label: t('admin.accounts.modelDiagnostics.resolvedUpstreamService'), value: props.result?.resolved_upstream_service || '' }
]))

const previewToChips = (items: AccountModelDiagnosticsPreview[]): DiagnosticsChip[] =>
  items.map((item) => ({
    key: `${item.public_id}::${item.source_id}::${item.alias_id || ''}`,
    label: item.public_id,
    detail: [item.source_id, item.alias_id, item.platform].filter(Boolean).join(' | ')
  }))

const modelSections = computed(() => ([
  {
    key: 'saved',
    label: t('admin.accounts.modelDiagnostics.savedModels'),
    items: (props.result?.saved_models || []).map((model) => ({ key: model, label: model, detail: '' }))
  },
  {
    key: 'detected',
    label: t('admin.accounts.modelDiagnostics.detectedModels'),
    items: (props.result?.detected_models || []).map((model) => ({ key: model, label: model, detail: '' }))
  },
  {
    key: 'public',
    label: t('admin.accounts.modelDiagnostics.publicModelsPreview'),
    items: previewToChips(props.result?.public_models_preview || [])
  }
]))
</script>
