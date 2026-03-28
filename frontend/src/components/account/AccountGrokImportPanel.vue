<template>
  <section class="space-y-4 rounded-xl border border-slate-200 bg-white/80 p-4 dark:border-slate-700 dark:bg-slate-950/40">
    <div class="space-y-1">
      <div class="flex items-center justify-between gap-3">
        <h3 class="text-sm font-semibold text-slate-900 dark:text-slate-100">
          {{ t('admin.accounts.grokImport.title') }}
        </h3>
        <span class="rounded-full bg-slate-100 px-2.5 py-1 text-[11px] font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-300">
          {{ t(activeSource.badgeKey) }}
        </span>
      </div>
      <p class="text-xs leading-5 text-slate-500 dark:text-slate-400">
        {{ t('admin.accounts.grokImport.description') }}
      </p>
    </div>

    <div class="grid gap-2 md:grid-cols-3">
      <button
        v-for="source in sourceOptions"
        :key="source.id"
        type="button"
        :class="sourceButtonClass(source.id)"
        @click="activeSourceId = source.id"
      >
        <span class="text-sm font-semibold">{{ t(source.labelKey) }}</span>
        <span class="text-xs opacity-80">{{ t(source.hintKey) }}</span>
      </button>
    </div>

    <div class="space-y-2">
      <label class="input-label">{{ t('admin.accounts.grokImport.contentLabel') }}</label>
      <textarea
        v-model="content"
        rows="7"
        class="input font-mono text-xs"
        :placeholder="t(activeSource.placeholderKey)"
      />
      <label class="flex items-start gap-2 text-xs text-slate-500 dark:text-slate-400">
        <input v-model="skipDefaultGroupBind" type="checkbox" class="mt-0.5 rounded border-slate-300 text-primary-600" />
        <span>{{ t('admin.accounts.grokImport.skipDefaultGroupBind') }}</span>
      </label>
    </div>

    <div class="flex flex-wrap gap-3">
      <button type="button" class="btn btn-secondary" :disabled="busy || !trimmedContent" @click="handlePreview">
        {{ previewing ? t('admin.accounts.grokImport.previewing') : t('admin.accounts.grokImport.previewAction') }}
      </button>
      <button type="button" class="btn btn-primary" :disabled="importDisabled" @click="handleImport">
        {{ importing ? t('admin.accounts.grokImport.importing') : t('common.import') }}
      </button>
      <button v-if="hasAnyResult" type="button" class="btn btn-secondary" :disabled="busy" @click="resetState">
        {{ t('common.reset') }}
      </button>
    </div>

    <div
      v-if="needsPreviewRefresh"
      class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-300"
    >
      {{ t('admin.accounts.grokImport.previewExpired') }}
    </div>

    <div v-if="previewResult" class="space-y-3 rounded-lg border border-sky-200 bg-sky-50/80 p-3 dark:border-sky-900/60 dark:bg-sky-950/30">
      <div class="flex flex-wrap items-center gap-2 text-xs text-sky-900 dark:text-sky-100">
        <span class="font-semibold">{{ t('admin.accounts.grokImport.previewSummary') }}</span>
        <span v-if="previewResult.detected_kind" class="rounded-full bg-white/80 px-2 py-0.5 dark:bg-white/10">
          {{ t('admin.accounts.grokImport.detectedKind', { kind: previewResult.detected_kind }) }}
        </span>
        <span class="rounded-full bg-white/80 px-2 py-0.5 dark:bg-white/10">
          {{ t('admin.accounts.grokImport.previewCounts', previewCounts) }}
        </span>
      </div>
      <div v-if="previewResult.errors?.length" class="space-y-1 text-xs text-rose-600 dark:text-rose-300">
        <div v-for="error in previewResult.errors" :key="`preview-error-${error.index}-${error.message}`">
          #{{ error.index }} {{ error.message }}
        </div>
      </div>
      <div class="space-y-2">
        <article
          v-for="item in previewResult.items"
          :key="`preview-${item.index}`"
          class="rounded-lg border border-sky-100 bg-white/90 px-3 py-2 text-xs dark:border-sky-900/40 dark:bg-slate-950/50"
        >
          <div class="flex flex-wrap items-center gap-2">
            <span class="font-semibold text-slate-900 dark:text-slate-100">#{{ item.index }} {{ item.name || item.type }}</span>
            <span :class="statusClass(item.status)">{{ item.status }}</span>
            <span class="rounded-full bg-slate-100 px-2 py-0.5 text-slate-600 dark:bg-slate-800 dark:text-slate-300">{{ item.type }}</span>
            <span class="rounded-full bg-slate-100 px-2 py-0.5 text-slate-600 dark:bg-slate-800 dark:text-slate-300">{{ item.grok_tier }}</span>
          </div>
          <div class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-slate-500 dark:text-slate-400">
            <span>{{ item.credential_masked }}</span>
            <span>P{{ item.priority }}</span>
            <span>C{{ item.concurrency }}</span>
            <span v-if="item.source_pool">{{ item.source_pool }}</span>
            <span v-if="item.reason">{{ item.reason }}</span>
          </div>
        </article>
      </div>
    </div>

    <div v-if="importResult" class="space-y-3 rounded-lg border border-emerald-200 bg-emerald-50/80 p-3 dark:border-emerald-900/60 dark:bg-emerald-950/30">
      <div class="flex flex-wrap items-center gap-2 text-xs text-emerald-900 dark:text-emerald-100">
        <span class="font-semibold">{{ t('admin.accounts.grokImport.importSummary') }}</span>
        <span v-if="importResult.detected_kind" class="rounded-full bg-white/80 px-2 py-0.5 dark:bg-white/10">
          {{ t('admin.accounts.grokImport.detectedKind', { kind: importResult.detected_kind }) }}
        </span>
        <span class="rounded-full bg-white/80 px-2 py-0.5 dark:bg-white/10">
          {{ t('admin.accounts.grokImport.importCounts', importCounts) }}
        </span>
      </div>
      <div v-if="importResult.errors?.length" class="space-y-1 text-xs text-rose-600 dark:text-rose-300">
        <div v-for="error in importResult.errors" :key="`import-error-${error.index}-${error.message}`">
          #{{ error.index }} {{ error.message }}
        </div>
      </div>
      <div class="space-y-2">
        <article
          v-for="item in importResult.results"
          :key="`result-${item.index}-${item.name}`"
          class="rounded-lg border border-emerald-100 bg-white/90 px-3 py-2 text-xs dark:border-emerald-900/40 dark:bg-slate-950/50"
        >
          <div class="flex flex-wrap items-center gap-2">
            <span class="font-semibold text-slate-900 dark:text-slate-100">#{{ item.index }} {{ item.name || item.type }}</span>
            <span :class="statusClass(item.status)">{{ item.status }}</span>
            <span class="rounded-full bg-slate-100 px-2 py-0.5 text-slate-600 dark:bg-slate-800 dark:text-slate-300">{{ item.type }}</span>
            <span v-if="item.account_id" class="rounded-full bg-slate-100 px-2 py-0.5 text-slate-600 dark:bg-slate-800 dark:text-slate-300">ID {{ item.account_id }}</span>
          </div>
          <div class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-slate-500 dark:text-slate-400">
            <span v-if="item.source_pool">{{ item.source_pool }}</span>
            <span v-if="item.reason">{{ item.reason }}</span>
          </div>
        </article>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { GrokImportPreviewResponse, GrokImportResult } from '@/api/admin/accounts'
import { useAppStore } from '@/stores/app'

type GrokImportSource = 'legacy_pool' | 'sso' | 'apikey'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'imported', result: GrokImportResult): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const content = ref('')
const skipDefaultGroupBind = ref(false)
const activeSourceId = ref<GrokImportSource>('legacy_pool')
const previewing = ref(false)
const importing = ref(false)
const previewResult = ref<GrokImportPreviewResponse | null>(null)
const importResult = ref<GrokImportResult | null>(null)
const previewSignature = ref('')

const sourceOptions = [
  {
    id: 'legacy_pool' as const,
    badgeKey: 'admin.accounts.grokImport.legacyBadge',
    labelKey: 'admin.accounts.grokImport.sources.legacy',
    hintKey: 'admin.accounts.grokImport.sourceHints.legacy',
    placeholderKey: 'admin.accounts.grokImport.placeholders.legacy'
  },
  {
    id: 'sso' as const,
    badgeKey: 'admin.accounts.grokImport.ssoBadge',
    labelKey: 'admin.accounts.grokImport.sources.sso',
    hintKey: 'admin.accounts.grokImport.sourceHints.sso',
    placeholderKey: 'admin.accounts.grokImport.placeholders.sso'
  },
  {
    id: 'apikey' as const,
    badgeKey: 'admin.accounts.grokImport.apikeyBadge',
    labelKey: 'admin.accounts.grokImport.sources.apikey',
    hintKey: 'admin.accounts.grokImport.sourceHints.apikey',
    placeholderKey: 'admin.accounts.grokImport.placeholders.apikey'
  }
]

const activeSource = computed(() => sourceOptions.find((item) => item.id === activeSourceId.value) || sourceOptions[0])
const trimmedContent = computed(() => content.value.trim())
const busy = computed(() => previewing.value || importing.value)
const needsPreviewRefresh = computed(() => Boolean(previewResult.value) && previewSignature.value !== trimmedContent.value)
const hasAnyResult = computed(() => Boolean(previewResult.value || importResult.value))
const importDisabled = computed(() => busy.value || !previewResult.value || needsPreviewRefresh.value)
const previewCounts = computed(() => {
  const items = previewResult.value?.items || []
  return {
    total: items.length,
    ready: items.filter((item) => item.status === 'ready').length,
    skipped: items.filter((item) => item.status === 'skipped').length,
    failed: items.filter((item) => item.status === 'failed').length
  }
})
const importCounts = computed(() => ({
  created: importResult.value?.created || 0,
  skipped: importResult.value?.skipped || 0,
  failed: importResult.value?.failed || 0
}))

watch(
  () => props.show,
  (show) => {
    if (!show) {
      resetState()
    }
  }
)

function resetState() {
  content.value = ''
  skipDefaultGroupBind.value = false
  previewResult.value = null
  importResult.value = null
  previewSignature.value = ''
  previewing.value = false
  importing.value = false
  activeSourceId.value = 'legacy_pool'
}

function sourceButtonClass(sourceId: GrokImportSource) {
  return [
    'flex flex-col rounded-lg border px-3 py-3 text-left transition',
    activeSourceId.value === sourceId
      ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-500/10 dark:text-primary-200'
      : 'border-slate-200 bg-slate-50 text-slate-700 hover:border-slate-300 dark:border-slate-700 dark:bg-slate-900/50 dark:text-slate-200 dark:hover:border-slate-500'
  ]
}

function statusClass(status: string) {
  if (status === 'created' || status === 'ready') {
    return 'rounded-full bg-emerald-100 px-2 py-0.5 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
  }
  if (status === 'skipped') {
    return 'rounded-full bg-amber-100 px-2 py-0.5 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
  }
  return 'rounded-full bg-rose-100 px-2 py-0.5 text-rose-700 dark:bg-rose-500/15 dark:text-rose-300'
}

async function handlePreview() {
  if (!trimmedContent.value) {
    return
  }
  previewing.value = true
  importResult.value = null
  try {
    const result = await adminAPI.accounts.previewGrokImport({
      content: trimmedContent.value,
      skip_default_group_bind: skipDefaultGroupBind.value
    })
    previewResult.value = result
    previewSignature.value = trimmedContent.value
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.grokImport.previewFailed'))
  } finally {
    previewing.value = false
  }
}

async function handleImport() {
  if (importDisabled.value) {
    if (needsPreviewRefresh.value) {
      appStore.showInfo(t('admin.accounts.grokImport.previewExpired'))
    }
    return
  }
  importing.value = true
  try {
    const result = await adminAPI.accounts.importGrok({
      content: trimmedContent.value,
      skip_default_group_bind: skipDefaultGroupBind.value
    })
    importResult.value = result
    emit('imported', result)
    appStore.showSuccess(t('admin.accounts.grokImport.importCounts', importCounts.value))
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.grokImport.importFailed'))
  } finally {
    importing.value = false
  }
}
</script>
