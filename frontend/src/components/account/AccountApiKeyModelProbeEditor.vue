<template>
  <section class="space-y-3 rounded-2xl border border-gray-200 bg-white/80 p-4 dark:border-dark-600 dark:bg-dark-700/60">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div class="space-y-1">
        <div class="text-sm font-semibold text-gray-900 dark:text-gray-100">
          {{ t('admin.accounts.apiKeyProbe.title') }}
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.apiKeyProbe.hint') }}
        </p>
      </div>

      <button
        type="button"
        class="inline-flex items-center justify-center gap-2 rounded-xl bg-primary-500 px-4 py-2 text-sm font-medium text-white transition hover:bg-primary-600 disabled:cursor-not-allowed disabled:bg-primary-300"
        :disabled="probing || !probeReady"
        @click="handleProbe"
      >
        <Icon v-if="probing" name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
        <Icon v-else name="search" size="sm" :stroke-width="2" />
        <span>{{ t('admin.accounts.apiKeyProbe.action') }}</span>
      </button>
    </div>

    <div
      v-if="probeSource || probeNotice"
      class="rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-600 dark:border-dark-500 dark:bg-dark-800 dark:text-gray-300"
    >
      <div v-if="probeSource">
        {{ t('admin.accounts.apiKeyProbe.probeSourceLabel') }}{{ probeSource }}
      </div>
      <div v-if="probeNotice">
        {{ t('admin.accounts.apiKeyProbe.probeNoticeLabel') }}{{ probeNotice }}
      </div>
    </div>

    <AccountResolvedUpstreamPanel
      :upstream-url="resolvedUpstream?.upstream_url"
      :upstream-host="resolvedUpstream?.upstream_host"
      :upstream-service="resolvedUpstream?.upstream_service"
      :upstream-region="resolvedUpstream?.upstream_region"
      :probe-source="resolvedUpstream?.upstream_probe_source"
      :probed-at="resolvedUpstream?.upstream_probed_at"
    />

    <div
      v-if="probedModels.length > 0"
      class="flex flex-wrap items-center justify-between gap-2 text-xs text-gray-500 dark:text-gray-400"
    >
      <span>{{ t('admin.accounts.apiKeyProbe.selectionHint') }}</span>
      <div class="flex flex-wrap items-center gap-2">
        <span>{{ t('admin.accounts.apiKeyProbe.selectedCount', { count: allowedModels.length }) }}</span>
        <button
          type="button"
          class="rounded-lg border border-gray-200 px-3 py-1.5 font-medium text-gray-700 transition hover:border-primary-300 hover:text-primary-600 dark:border-dark-500 dark:text-gray-200 dark:hover:border-primary-500 dark:hover:text-primary-300"
          @click="selectAllCallableModels"
        >
          {{ t('admin.accounts.apiKeyProbe.selectCallableModels') }}
        </button>
        <button
          type="button"
          class="rounded-lg border border-gray-200 px-3 py-1.5 font-medium text-gray-700 transition hover:border-rose-300 hover:text-rose-600 dark:border-dark-500 dark:text-gray-200 dark:hover:border-rose-500 dark:hover:text-rose-300"
          @click="clearSelectedModels"
        >
          {{ t('admin.accounts.apiKeyProbe.clearSelection') }}
        </button>
      </div>
    </div>

    <div v-if="probedModels.length > 0" class="grid gap-4 lg:grid-cols-2 2xl:grid-cols-3">
      <button
        v-for="model in probedModels"
        :key="model.id"
        type="button"
        :title="model.id"
        :class="cardClasses(model)"
        @click="toggleModel(model)"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0 space-y-1.5">
            <AccountProbeModelIdentity
              :model-id="model.id"
              :display-name="resolveDisplayName(model)"
              :provider="resolveProvider(model)"
              :provider-text="resolveProviderLabel(model)"
            />
            <div
              v-if="resolveProviderLabel(model) || model.upstream_source || model.availability"
              class="mt-2 flex flex-wrap items-center gap-2"
            >
              <span
                v-if="resolveProviderLabel(model)"
                class="inline-flex items-center rounded-full bg-white/70 px-2 py-0.5 text-[11px] font-medium text-slate-700 dark:bg-white/10 dark:text-slate-200"
              >
                {{ resolveProviderLabel(model) }}
              </span>
              <span
                v-if="model.upstream_source"
                class="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium"
                :class="sourceBadgeClasses(model)"
              >
                {{ sourceBadgeText(model) }}
              </span>
              <span
                v-if="model.availability"
                class="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium"
                :class="availabilityBadgeClasses(model)"
              >
                {{ availabilityBadgeText(model) }}
              </span>
            </div>
            <div
              v-if="model.availability === 'uncallable' && model.availability_reason"
              class="mt-2 break-words text-xs text-rose-700 dark:text-rose-200"
            >
              {{ model.availability_reason }}
            </div>
          </div>
          <span
            v-if="isSelected(model.id)"
            class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-white/80 text-emerald-600 shadow-sm dark:bg-white/10 dark:text-emerald-300"
          >
            <Icon name="check" size="sm" :stroke-width="2" />
          </span>
        </div>

        <div class="mt-3 flex flex-wrap items-start justify-between gap-3 text-xs">
          <span class="break-words">
            {{
              model.registry_state === 'existing'
                ? t('admin.accounts.apiKeyProbe.registryExisting')
                : t('admin.accounts.apiKeyProbe.registryMissing')
            }}
          </span>
          <span v-if="model.registry_model_id" class="break-words text-right opacity-80" :title="model.registry_model_id">
            {{ model.registry_model_id }}
          </span>
        </div>

        <div
          v-if="isSelected(model.id) && model.availability === 'uncallable'"
          class="mt-3 rounded-xl border border-rose-200 bg-rose-50 px-3 py-2 text-xs text-rose-700 dark:border-rose-900/60 dark:bg-rose-950/30 dark:text-rose-200"
          @click.stop
        >
          {{ t('admin.accounts.apiKeyProbe.selectedUncallableWarning') }}
        </div>
      </button>
    </div>

    <div
      v-else
      class="rounded-xl border border-dashed border-gray-300 px-3 py-4 text-sm text-gray-500 dark:border-dark-500 dark:text-gray-400"
    >
      {{ t('admin.accounts.apiKeyProbe.empty') }}
    </div>

    <AccountManualModelsEditor
      v-model:rows="manualModels"
      :allow-source-protocol="false"
    />
  </section>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { AccountManualModel, ProtocolGatewayProbeModel } from '@/api/admin/accounts'
import AccountManualModelsEditor from '@/components/account/AccountManualModelsEditor.vue'
import AccountProbeModelIdentity from '@/components/account/AccountProbeModelIdentity.vue'
import AccountResolvedUpstreamPanel from '@/components/account/AccountResolvedUpstreamPanel.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import type { ModelMapping } from '@/utils/accountFormShared'
import {
  resolveAccountModelImportErrorMessage,
  resolveAccountModelImportProbeNoticeMessage
} from '@/utils/accountModelImport'
import {
  createAccountModelProbeSnapshotDraft,
  createResolvedUpstreamDraft,
  isProbeSnapshotEqual,
  isUpstreamDraftEqual,
  readAccountModelProbeSnapshot,
  readAccountResolvedUpstreamDraft,
  type AccountModelProbeSnapshotDraft,
  type AccountResolvedUpstreamDraft
} from '@/utils/accountProbeDraft'
import { formatModelDisplayName } from '@/utils/modelDisplayName'
import { formatProviderLabel, normalizeProviderSlug } from '@/utils/providerLabels'

const props = defineProps<{
  platform: string
  accountType: string
  credentials: Record<string, unknown>
  extra?: Record<string, unknown>
  proxyId?: number | null
  probeReady: boolean
}>()

const allowedModels = defineModel<string[]>('allowedModels', { required: true })
const modelMappings = defineModel<ModelMapping[]>('modelMappings', { required: true })
const probedModels = defineModel<ProtocolGatewayProbeModel[]>('probedModels', { required: true })
const manualModels = defineModel<AccountManualModel[]>('manualModels', { required: true })
const resolvedUpstream = defineModel<AccountResolvedUpstreamDraft | null>('resolvedUpstream', { required: true })
const probeSnapshot = defineModel<AccountModelProbeSnapshotDraft | null>('probeSnapshot', { default: null })

const { t } = useI18n()
const appStore = useAppStore()
const probing = ref(false)
const probeSource = ref('')
const probeNotice = ref('')
const hasInitializedFromMappings = ref(false)
const normalizeModelID = (value: string) => String(value || '').trim()
const normalizeExplicitMappingRows = (rows: ModelMapping[]) =>
  rows
    .map((row) => ({
      from: normalizeModelID(row.from),
      to: normalizeModelID(row.to)
    }))
    .filter((row) => Boolean(row.from) && Boolean(row.to) && row.from !== row.to)

const serializeMappings = (rows: ModelMapping[]) => JSON.stringify(normalizeExplicitMappingRows(rows))

watch(
  () => [probedModels.value.length, allowedModels.value.join('\x00'), modelMappings.value.length] as const,
  ([modelCount, selectedModelsSerialized, mappingCount]) => {
    if (hasInitializedFromMappings.value || modelCount > 0 || (!selectedModelsSerialized && mappingCount === 0)) return
    hasInitializedFromMappings.value = true
    const seen = new Set<string>()
    const selectedTargets = [
      ...allowedModels.value.map((modelId) => modelId.trim()),
      ...modelMappings.value.map((row) => row.to.trim())
    ]
    probedModels.value = selectedTargets
      .filter((target) => target && !seen.has(target) && (seen.add(target), true))
      .map((target) => ({
        id: target,
        display_name: formatModelDisplayName(target) || target,
        registry_state: 'existing' as const,
        registry_model_id: target
      }))
  },
  { immediate: true }
)

watch(
  () => props.extra,
  (extra) => {
    const snapshot = readAccountModelProbeSnapshot(extra)
    if (snapshot) {
      if (!isProbeSnapshotEqual(snapshot, probeSnapshot.value)) {
        probeSnapshot.value = snapshot
      }
      if (probedModels.value.length === 0) {
        probedModels.value = snapshot.models.map((modelId) => ({
          id: modelId,
          display_name: formatModelDisplayName(modelId) || modelId,
          registry_state: 'existing' as const,
          registry_model_id: modelId
        }))
      }
      if (!probeSource.value) {
        probeSource.value = snapshot.probe_source || snapshot.source || ''
      }
      if (!resolvedUpstream.value?.upstream_probed_at && snapshot.updated_at) {
        const nextUpstream = {
          ...(resolvedUpstream.value || {}),
          upstream_probed_at: snapshot.updated_at
        }
        if (!isUpstreamDraftEqual(nextUpstream, resolvedUpstream.value)) {
          resolvedUpstream.value = nextUpstream
        }
      }
    }
    const draft = readAccountResolvedUpstreamDraft(extra)
    if (draft && !isUpstreamDraftEqual(draft, resolvedUpstream.value)) {
      resolvedUpstream.value = draft
    }
  },
  { immediate: true, deep: true }
)

watch(
  () => [allowedModels.value.join('\x00'), modelMappings.value.length] as const,
  () => {
    const selected = new Set(allowedModels.value.map((item) => item.trim()).filter(Boolean))
    const nextMappings = normalizeExplicitMappingRows(modelMappings.value).filter((row) => selected.has(row.to))
    if (serializeMappings(nextMappings) !== serializeMappings(modelMappings.value)) {
      modelMappings.value = nextMappings
    }
  },
  { immediate: true }
)

const isSelected = (modelId: string) => allowedModels.value.includes(modelId)
const isCallableModel = (model: ProtocolGatewayProbeModel) => model.availability !== 'uncallable'

const sourceBadgeText = (model: ProtocolGatewayProbeModel) =>
  model.upstream_source === 'verified_extra'
    ? t('admin.accounts.apiKeyProbe.sourceVerifiedExtra')
    : t('admin.accounts.apiKeyProbe.sourceOfficial')

const availabilityBadgeText = (model: ProtocolGatewayProbeModel) =>
  model.availability === 'uncallable'
    ? t('admin.accounts.apiKeyProbe.availabilityUncallable')
    : t('admin.accounts.apiKeyProbe.availabilityCallable')

const sourceBadgeClasses = (model: ProtocolGatewayProbeModel) =>
  model.upstream_source === 'verified_extra'
    ? 'bg-sky-100 text-sky-700 dark:bg-sky-950/40 dark:text-sky-200'
    : 'bg-slate-100 text-slate-700 dark:bg-slate-900/60 dark:text-slate-200'

const availabilityBadgeClasses = (model: ProtocolGatewayProbeModel) =>
  model.availability === 'uncallable'
    ? 'bg-rose-100 text-rose-700 dark:bg-rose-950/40 dark:text-rose-200'
    : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-200'

const resolveProvider = (model: ProtocolGatewayProbeModel) =>
  normalizeProviderSlug(model.provider)

const resolveProviderLabel = (model: ProtocolGatewayProbeModel) => {
  const provider = resolveProvider(model)
  const providerLabel = String(model.provider_label || '').trim()
  if (!provider && !providerLabel) {
    return ''
  }
  return formatProviderLabel(provider, providerLabel)
}

const resolveDisplayName = (model: ProtocolGatewayProbeModel) =>
  String(model.display_name || '').trim() || formatModelDisplayName(model.id) || model.id

const syncSelectedModels = (modelIds: string[]) => {
  const nextAllowedModels = [...new Set(modelIds.map((item) => item.trim()).filter(Boolean))]
  const selectedTargets = new Set(nextAllowedModels)
  allowedModels.value = nextAllowedModels
  modelMappings.value = normalizeExplicitMappingRows(modelMappings.value).filter((row) => selectedTargets.has(row.to))
}

const toggleModel = (model: ProtocolGatewayProbeModel) => {
  const nextSelected = new Set(allowedModels.value.map((item) => item.trim()).filter(Boolean))
  if (nextSelected.has(model.id)) {
    nextSelected.delete(model.id)
  } else {
    nextSelected.add(model.id)
  }
  syncSelectedModels(
    probedModels.value.filter((item) => nextSelected.has(item.id)).map((item) => item.id)
  )
}

const cardClasses = (model: ProtocolGatewayProbeModel) => [
  'rounded-2xl border px-4 py-3 text-left transition',
  model.availability === 'uncallable'
    ? 'border-rose-200 bg-rose-50/90 text-rose-950 dark:border-rose-900/70 dark:bg-rose-950/30 dark:text-rose-100'
    : model.registry_state === 'existing'
    ? 'border-emerald-200 bg-emerald-50/90 text-emerald-900 dark:border-emerald-900/70 dark:bg-emerald-950/30 dark:text-emerald-100'
    : 'border-amber-200 bg-amber-50/90 text-amber-900 dark:border-amber-900/70 dark:bg-amber-950/30 dark:text-amber-100',
  isSelected(model.id)
    ? 'ring-2 ring-primary-400/60 shadow-md'
    : 'hover:border-primary-200 hover:shadow-sm dark:hover:border-primary-700/60'
]

const selectAllCallableModels = () => {
  const selected = new Set(allowedModels.value.map((item) => item.trim()).filter(Boolean))
  syncSelectedModels(
    probedModels.value
      .filter((model) => selected.has(model.id) || isCallableModel(model))
      .map((model) => model.id)
  )
}

const clearSelectedModels = () => {
  allowedModels.value = []
  modelMappings.value = []
}

const handleProbe = async () => {
  if (probing.value || !props.probeReady) return
  probing.value = true
  try {
    const result = await adminAPI.accounts.probeModels({
      platform: props.platform,
      type: props.accountType,
      credentials: props.credentials,
      extra: props.extra,
      manual_models: manualModels.value,
      proxy_id: props.proxyId ?? undefined
    })
    const probedAt = new Date().toISOString()
    const selectedTargets = new Set(allowedModels.value.map((item) => item.trim()).filter(Boolean))
    const nextAllowedModels = result.models
      .map((model) => model.id)
      .filter((modelId) => selectedTargets.has(modelId))
    probedModels.value = result.models
    syncSelectedModels(nextAllowedModels)
    probeSource.value = result.probe_source || ''
    probeNotice.value = resolveAccountModelImportProbeNoticeMessage(t, {
      imported_count: result.models.length,
      probe_source: result.probe_source,
      probe_notice: result.probe_notice
    })
    probeSnapshot.value = createAccountModelProbeSnapshotDraft({
      models: result.models.map((model) => model.id),
      updated_at: probedAt,
      source: 'manual_probe',
      probe_source: result.probe_source
    })
    resolvedUpstream.value =
      createResolvedUpstreamDraft({
        upstream_url: result.resolved_upstream_url,
        upstream_host: result.resolved_upstream_host,
        upstream_service: result.resolved_upstream_service,
        upstream_probe_source: result.probe_source,
        upstream_probed_at: probedAt
      }) || resolvedUpstream.value
  } catch (error: any) {
    console.error('Failed to probe account models:', error)
    appStore.showError(resolveAccountModelImportErrorMessage(t, error) || t('admin.accounts.apiKeyProbe.failed'))
  } finally {
    probing.value = false
  }
}
</script>
