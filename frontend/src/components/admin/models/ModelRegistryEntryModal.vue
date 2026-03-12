<template>
  <BaseDialog
    :show="show"
    :title="dialogTitle"
    width="wide"
    close-on-click-outside
    @close="emit('close')"
  >
    <form class="space-y-4" @submit.prevent="handleSubmit">
      <div v-if="entry" class="grid gap-3 rounded-2xl border border-gray-200 bg-gray-50 p-4 text-sm dark:border-dark-700 dark:bg-dark-900/40 md:grid-cols-3">
        <div>
          <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.models.registry.source') }}</p>
          <p class="mt-1 font-medium text-gray-900 dark:text-white">{{ sourceLabel }}</p>
        </div>
        <div>
          <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.models.registry.status') }}</p>
          <div class="mt-1 flex flex-wrap gap-2">
            <span class="inline-flex rounded-full bg-gray-200 px-2.5 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200">
              {{ entry.hidden ? t('admin.models.registry.statusLabels.hidden') : t('admin.models.registry.statusLabels.active') }}
            </span>
            <span
              v-if="entry.tombstoned"
              class="inline-flex rounded-full bg-red-100 px-2.5 py-1 text-xs font-medium text-red-700 dark:bg-red-500/15 dark:text-red-300"
            >
              {{ t('admin.models.registry.statusLabels.tombstoned') }}
            </span>
          </div>
        </div>
        <div>
          <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.models.registry.fields.id') }}</p>
          <p class="mt-1 break-all font-mono text-gray-900 dark:text-white">{{ entry.id }}</p>
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <div>
          <label class="input-label" for="registry-model-id">{{ t('admin.models.registry.fields.id') }}</label>
          <input
            id="registry-model-id"
            v-model.trim="form.id"
            type="text"
            class="input"
            :disabled="isEdit"
            required
          />
          <p v-if="isEdit" class="mt-2 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.models.registry.idLockedHint') }}
          </p>
        </div>

        <div>
          <label class="input-label" for="registry-display-name">{{ t('admin.models.registry.fields.displayName') }}</label>
          <input id="registry-display-name" v-model.trim="form.display_name" type="text" class="input" />
        </div>

        <div>
          <label class="input-label" for="registry-provider">{{ t('admin.models.registry.fields.provider') }}</label>
          <input id="registry-provider" v-model.trim="form.provider" type="text" class="input" />
        </div>

        <div>
          <label class="input-label" for="registry-ui-priority">{{ t('admin.models.registry.fields.uiPriority') }}</label>
          <input id="registry-ui-priority" v-model.number="form.ui_priority" type="number" min="0" class="input" />
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <div>
          <label class="input-label" for="registry-platforms">{{ t('admin.models.registry.fields.platforms') }}</label>
          <textarea id="registry-platforms" v-model="form.platforms" class="input min-h-[92px]" />
        </div>

        <div>
          <label class="input-label" for="registry-protocol-ids">{{ t('admin.models.registry.fields.protocolIds') }}</label>
          <textarea id="registry-protocol-ids" v-model="form.protocol_ids" class="input min-h-[92px]" />
        </div>

        <div>
          <label class="input-label" for="registry-aliases">{{ t('admin.models.registry.fields.aliases') }}</label>
          <textarea id="registry-aliases" v-model="form.aliases" class="input min-h-[92px]" />
        </div>

        <div>
          <label class="input-label" for="registry-pricing-ids">{{ t('admin.models.registry.fields.pricingLookupIds') }}</label>
          <textarea id="registry-pricing-ids" v-model="form.pricing_lookup_ids" class="input min-h-[92px]" />
        </div>

        <div>
          <label class="input-label" for="registry-modalities">{{ t('admin.models.registry.fields.modalities') }}</label>
          <textarea id="registry-modalities" v-model="form.modalities" class="input min-h-[92px]" />
        </div>

        <div>
          <label class="input-label" for="registry-capabilities">{{ t('admin.models.registry.fields.capabilities') }}</label>
          <textarea id="registry-capabilities" v-model="form.capabilities" class="input min-h-[92px]" />
        </div>
      </div>

      <div class="space-y-3">
        <label class="input-label" for="registry-exposed-in">{{ t('admin.models.registry.fields.exposedIn') }}</label>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="action in exposureQuickActions"
            :key="action.target"
            type="button"
            class="rounded-full border px-3 py-1.5 text-xs font-medium transition-colors"
            :class="activeExposureTargets.has(action.target)
              ? 'border-primary-300 bg-primary-50 text-primary-700 dark:border-primary-500/40 dark:bg-primary-500/10 dark:text-primary-300'
              : 'border-gray-200 bg-white text-gray-600 hover:border-gray-300 hover:text-gray-900 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-300 dark:hover:border-dark-500 dark:hover:text-white'"
            @click="toggleExposure(action.target)"
          >
            {{ action.label }}
          </button>
        </div>
        <textarea id="registry-exposed-in" v-model="form.exposed_in" class="input min-h-[92px]" />
      </div>

      <p class="text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.models.registry.commaSeparatedHint') }}
      </p>

      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button type="submit" class="btn btn-primary" :disabled="saving || !form.id.trim()">
          {{ saving ? t('admin.models.saving') : t('common.save') }}
        </button>
      </div>
    </form>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { ModelRegistryDetail, UpsertModelRegistryEntryPayload } from '@/api/admin/modelRegistry'

const props = withDefaults(defineProps<{
  show: boolean
  entry?: ModelRegistryDetail | null
  saving?: boolean
}>(), {
  entry: null,
  saving: false
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'submit', payload: UpsertModelRegistryEntryPayload): void
}>()

const { t } = useI18n()

const form = reactive({
  id: '',
  display_name: '',
  provider: '',
  ui_priority: 5000,
  platforms: '',
  protocol_ids: '',
  aliases: '',
  pricing_lookup_ids: '',
  modalities: '',
  capabilities: '',
  exposed_in: ''
})

const exposureQuickActions = [
  { target: 'whitelist', label: '白名单页' },
  { target: 'use_key', label: 'Use Key' },
  { target: 'test', label: '测试页' },
  { target: 'runtime', label: '运行时' }
] as const

const isEdit = computed(() => Boolean(props.entry))
const dialogTitle = computed(() =>
  isEdit.value ? t('admin.models.registry.editModel') : t('admin.models.registry.addModel')
)
const sourceLabel = computed(() => formatSourceLabel(props.entry?.source || ''))
const activeExposureTargets = computed(() => new Set(parseList(form.exposed_in)))

watch(
  () => [props.show, props.entry] as const,
  ([show]) => {
    if (!show) {
      return
    }
    form.id = props.entry?.id || ''
    form.display_name = props.entry?.display_name || ''
    form.provider = props.entry?.provider || ''
    form.ui_priority = props.entry?.ui_priority || 5000
    form.platforms = formatList(props.entry?.platforms)
    form.protocol_ids = formatList(props.entry?.protocol_ids)
    form.aliases = formatList(props.entry?.aliases)
    form.pricing_lookup_ids = formatList(props.entry?.pricing_lookup_ids)
    form.modalities = formatList(props.entry?.modalities)
    form.capabilities = formatList(props.entry?.capabilities)
    form.exposed_in = formatList(props.entry?.exposed_in)
  },
  { immediate: true }
)

function formatList(items?: string[] | null) {
  return Array.isArray(items) ? items.join(', ') : ''
}

function parseList(value: string): string[] {
  return Array.from(
    new Set(
      value
        .replace(/\r/g, '')
        .split('\n')
        .flatMap((item: string) => item.split(','))
        .map((item: string) => item.trim())
        .filter(Boolean)
    )
  )
}

function formatSourceLabel(source: string) {
  if (!source) {
    return '-'
  }
  const normalizedSource = source === 'runtime' ? 'manual' : source
  const key = `admin.models.registry.sourceLabels.${normalizedSource}`
  const translated = t(key)
  return translated === key ? normalizedSource : translated
}

function toggleExposure(target: string) {
  const next = new Set(parseList(form.exposed_in))
  if (next.has(target)) {
    next.delete(target)
  } else {
    next.add(target)
  }
  form.exposed_in = Array.from(next).join(', ')
}

function handleSubmit() {
  emit('submit', {
    id: form.id.trim(),
    display_name: form.display_name.trim(),
    provider: form.provider.trim(),
    ui_priority: Number.isFinite(Number(form.ui_priority)) ? Number(form.ui_priority) : 5000,
    platforms: parseList(form.platforms),
    protocol_ids: parseList(form.protocol_ids),
    aliases: parseList(form.aliases),
    pricing_lookup_ids: parseList(form.pricing_lookup_ids),
    modalities: parseList(form.modalities),
    capabilities: parseList(form.capabilities),
    exposed_in: parseList(form.exposed_in)
  })
}
</script>
