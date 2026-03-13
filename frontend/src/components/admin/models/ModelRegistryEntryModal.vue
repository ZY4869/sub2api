<template>
  <BaseDialog
    :show="show"
    :title="dialogTitle"
    width="wide"
    close-on-click-outside
    @close="emit('close')"
  >
    <form class="space-y-5" @submit.prevent="handleSubmit">
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

        <div class="space-y-3">
          <div>
            <label class="input-label" for="registry-provider-select">{{ t('admin.models.registry.fields.provider') }}</label>
            <select id="registry-provider-select" v-model="providerSelection" class="input">
              <option value="">自动推断</option>
              <option v-for="provider in providerOptions" :key="provider" :value="provider">
                {{ provider }}
              </option>
              <option :value="MODEL_REGISTRY_CUSTOM_PROVIDER">自定义输入</option>
            </select>
          </div>
          <div v-if="providerSelection === MODEL_REGISTRY_CUSTOM_PROVIDER">
            <input v-model.trim="customProvider" type="text" class="input" placeholder="输入自定义提供商，例如 openrouter" />
          </div>
        </div>

        <div>
          <label class="input-label" for="registry-ui-priority">{{ t('admin.models.registry.fields.uiPriority') }}</label>
          <input id="registry-ui-priority" v-model.number="form.ui_priority" type="number" min="0" class="input" />
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-3">
        <div>
          <label class="input-label" for="registry-status">{{ t('admin.models.registry.fields.lifecycleStatus') }}</label>
          <select id="registry-status" v-model="form.status" class="input">
            <option value="">{{ t('admin.models.registry.lifecycleLabels.stable') }}</option>
            <option v-for="status in lifecycleStatusOptions" :key="status" :value="status">
              {{ formatLifecycleLabel(status) }}
            </option>
          </select>
        </div>

        <div>
          <label class="input-label" for="registry-replaced-by">{{ t('admin.models.registry.fields.replacedBy') }}</label>
          <input id="registry-replaced-by" v-model.trim="form.replaced_by" type="text" class="input" />
        </div>

        <div>
          <label class="input-label" for="registry-deprecated-at">{{ t('admin.models.registry.fields.deprecatedAt') }}</label>
          <input id="registry-deprecated-at" v-model.trim="form.deprecated_at" type="text" class="input" placeholder="2026-03-13T00:00:00Z" />
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <div class="space-y-3">
          <div>
            <label class="input-label">{{ t('admin.models.registry.fields.platforms') }}</label>
            <div class="mt-2 flex flex-wrap gap-2">
              <button
                v-for="platform in platformPresets"
                :key="platform"
                type="button"
                class="rounded-full border px-3 py-1.5 text-xs font-medium transition-colors"
                :class="selectedPresetPlatforms.includes(platform)
                  ? 'border-primary-300 bg-primary-50 text-primary-700 dark:border-primary-500/40 dark:bg-primary-500/10 dark:text-primary-300'
                  : 'border-gray-200 bg-white text-gray-600 hover:border-gray-300 hover:text-gray-900 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-300 dark:hover:border-dark-500 dark:hover:text-white'"
                @click="togglePlatformPreset(platform)"
              >
                {{ platform }}
              </button>
            </div>
          </div>
          <div>
            <label class="input-label" for="registry-custom-platforms">额外平台</label>
            <input
              id="registry-custom-platforms"
              v-model="customPlatforms"
              type="text"
              class="input"
              placeholder="可继续输入自定义平台，多个值用逗号分隔"
            />
            <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">保存前会自动去重、去空格并转成小写。</p>
          </div>
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
          <label class="input-label" for="registry-preferred-protocols">{{ t('admin.models.registry.fields.preferredProtocolIds') }}</label>
          <textarea
            id="registry-preferred-protocols"
            v-model="form.preferred_protocol_ids"
            class="input min-h-[92px]"
            placeholder="anthropic_oauth=claude-sonnet-4-5-20250929&#10;default=claude-sonnet-4.5"
          />
        </div>

        <div>
          <label class="input-label" for="registry-modalities">{{ t('admin.models.registry.fields.modalities') }}</label>
          <textarea id="registry-modalities" v-model="form.modalities" class="input min-h-[92px]" />
        </div>

        <div class="space-y-3">
          <label class="input-label">{{ t('admin.models.registry.fields.capabilities') }}</label>
          <div class="grid gap-2 sm:grid-cols-2">
            <button
              v-for="capability in capabilityOptions"
              :key="capability.value"
              type="button"
              class="flex items-center gap-3 rounded-2xl border px-3 py-2 text-left text-sm transition-colors"
              :class="selectedCapabilities.includes(capability.value)
                ? 'border-primary-300 bg-primary-50 text-primary-700 dark:border-primary-500/40 dark:bg-primary-500/10 dark:text-primary-300'
                : 'border-gray-200 bg-white text-gray-600 hover:border-gray-300 hover:text-gray-900 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-300 dark:hover:border-dark-500 dark:hover:text-white'"
              @click="toggleCapability(capability.value)"
            >
              <Icon :name="capability.icon" size="sm" />
              <span>{{ capability.label }}</span>
            </button>
          </div>
          <p class="text-xs text-gray-500 dark:text-gray-400">这里表示程序已经确认该模型具备对应能力，用于后续展示与筛选。</p>
        </div>
      </div>

      <div>
        <label class="input-label" for="registry-deprecation-notice">{{ t('admin.models.registry.fields.deprecationNotice') }}</label>
        <textarea id="registry-deprecation-notice" v-model="form.deprecation_notice" class="input min-h-[92px]" />
      </div>

      <div class="space-y-3">
        <label class="input-label">{{ t('admin.models.registry.fields.exposedIn') }}</label>
        <div class="grid gap-2 sm:grid-cols-2">
          <button
            v-for="exposure in exposureOptions"
            :key="exposure.value"
            type="button"
            class="rounded-2xl border px-3 py-3 text-left text-sm transition-colors"
            :class="selectedExposures.includes(exposure.value)
              ? 'border-primary-300 bg-primary-50 text-primary-700 dark:border-primary-500/40 dark:bg-primary-500/10 dark:text-primary-300'
              : 'border-gray-200 bg-white text-gray-600 hover:border-gray-300 hover:text-gray-900 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-300 dark:hover:border-dark-500 dark:hover:text-white'"
            @click="toggleExposure(exposure.value)"
          >
            <p class="font-medium">{{ exposure.shortLabel }}</p>
            <p class="mt-1 text-xs opacity-80">{{ exposure.description }}</p>
          </button>
        </div>
      </div>

      <p class="text-xs text-gray-500 dark:text-gray-400">
        其余列表字段仍支持逗号或换行分隔；保存时会自动去重并规范化。
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
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import type { ModelRegistryDetail, UpsertModelRegistryEntryPayload } from '@/api/admin/modelRegistry'
import { getModelRegistrySnapshot } from '@/stores/modelRegistry'
import {
  MODEL_REGISTRY_CAPABILITY_OPTIONS,
  MODEL_REGISTRY_CUSTOM_PROVIDER,
  MODEL_REGISTRY_EXPOSURE_OPTIONS,
  MODEL_REGISTRY_PLATFORM_PRESETS,
  formatRegistryList,
  normalizeCapabilityList,
  normalizeExposureTargets,
  normalizePlatformList,
  normalizeProviderOptions,
  normalizeRegistryList,
  normalizeRegistryToken
} from '@/utils/modelRegistryMeta'

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
  ui_priority: 5000,
  status: '',
  deprecated_at: '',
  replaced_by: '',
  deprecation_notice: '',
  protocol_ids: '',
  aliases: '',
  pricing_lookup_ids: '',
  preferred_protocol_ids: '',
  modalities: ''
})

const providerSelection = ref('')
const customProvider = ref('')
const selectedPresetPlatforms = ref<string[]>([])
const customPlatforms = ref('')
const selectedCapabilities = ref<string[]>([])
const selectedExposures = ref<string[]>([])

const isEdit = computed(() => Boolean(props.entry))
const dialogTitle = computed(() =>
  isEdit.value ? t('admin.models.registry.editModel') : t('admin.models.registry.addModel')
)
const sourceLabel = computed(() => formatSourceLabel(props.entry?.source || ''))
const providerOptions = computed(() => normalizeProviderOptions(getModelRegistrySnapshot().models))
const platformPresets = [...MODEL_REGISTRY_PLATFORM_PRESETS]
const capabilityOptions = [...MODEL_REGISTRY_CAPABILITY_OPTIONS]
const exposureOptions = [...MODEL_REGISTRY_EXPOSURE_OPTIONS]
const lifecycleStatusOptions = ['beta', 'deprecated'] as const

watch(
  () => [props.show, props.entry] as const,
  ([show]) => {
    if (!show) {
      return
    }

    form.id = props.entry?.id || ''
    form.display_name = props.entry?.display_name || ''
    form.ui_priority = props.entry?.ui_priority || 5000
    form.status = props.entry?.status || ''
    form.deprecated_at = props.entry?.deprecated_at || ''
    form.replaced_by = props.entry?.replaced_by || ''
    form.deprecation_notice = props.entry?.deprecation_notice || ''
    form.protocol_ids = formatRegistryList(props.entry?.protocol_ids)
    form.aliases = formatRegistryList(props.entry?.aliases)
    form.pricing_lookup_ids = formatRegistryList(props.entry?.pricing_lookup_ids)
    form.preferred_protocol_ids = formatPreferredProtocolIDs(props.entry?.preferred_protocol_ids)
    form.modalities = formatRegistryList(props.entry?.modalities)

    const provider = normalizeRegistryToken(props.entry?.provider || '')
    if (provider && providerOptions.value.includes(provider)) {
      providerSelection.value = provider
      customProvider.value = ''
    } else if (provider) {
      providerSelection.value = MODEL_REGISTRY_CUSTOM_PROVIDER
      customProvider.value = provider
    } else {
      providerSelection.value = ''
      customProvider.value = ''
    }

    const platforms = normalizePlatformList(props.entry?.platforms || [])
    selectedPresetPlatforms.value = platformPresets.filter((item) => platforms.includes(item))
    customPlatforms.value = platforms.filter((item) => !platformPresets.includes(item as any)).join(', ')
    selectedCapabilities.value = normalizeCapabilityList(props.entry?.capabilities || [])
    selectedExposures.value = normalizeExposureTargets(props.entry?.exposed_in || [])
  },
  { immediate: true }
)

function formatSourceLabel(source: string) {
  if (!source) {
    return '-'
  }
  const normalizedSource = source === 'runtime' ? 'manual' : source
  const key = `admin.models.registry.sourceLabels.${normalizedSource}`
  const translated = t(key)
  return translated === key ? normalizedSource : translated
}

function formatLifecycleLabel(value: string) {
  const normalized = value.trim().toLowerCase() || 'stable'
  const key = `admin.models.registry.lifecycleLabels.${normalized}`
  const translated = t(key)
  return translated === key ? normalized : translated
}

function formatPreferredProtocolIDs(value?: Record<string, string> | null) {
  if (!value) {
    return ''
  }
  return Object.entries(value)
    .filter(([route, model]) => route.trim() && model.trim())
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([route, model]) => `${route}=${model}`)
    .join('\n')
}

function parsePreferredProtocolIDs(value: string): Record<string, string> | undefined {
  const entries = normalizeRegistryList(value)
    .map((line) => line.split('=').map((item) => item.trim()))
    .filter((parts): parts is [string, string] => parts.length === 2 && Boolean(parts[0]) && Boolean(parts[1]))
  if (entries.length === 0) {
    return undefined
  }
  return Object.fromEntries(entries)
}

function togglePlatformPreset(platform: string) {
  if (selectedPresetPlatforms.value.includes(platform)) {
    selectedPresetPlatforms.value = selectedPresetPlatforms.value.filter((item) => item !== platform)
    return
  }
  selectedPresetPlatforms.value = [...selectedPresetPlatforms.value, platform]
}

function toggleCapability(capability: string) {
  if (selectedCapabilities.value.includes(capability)) {
    selectedCapabilities.value = selectedCapabilities.value.filter((item) => item !== capability)
    return
  }
  selectedCapabilities.value = [...selectedCapabilities.value, capability]
}

function toggleExposure(exposure: string) {
  if (selectedExposures.value.includes(exposure)) {
    selectedExposures.value = selectedExposures.value.filter((item) => item !== exposure)
    return
  }
  selectedExposures.value = [...selectedExposures.value, exposure]
}

function handleSubmit() {
  const provider = providerSelection.value === MODEL_REGISTRY_CUSTOM_PROVIDER
    ? normalizeRegistryToken(customProvider.value)
    : normalizeRegistryToken(providerSelection.value)

  emit('submit', {
    id: form.id.trim(),
    display_name: form.display_name.trim(),
    provider,
    ui_priority: Number.isFinite(Number(form.ui_priority)) ? Number(form.ui_priority) : 5000,
    status: form.status.trim(),
    deprecated_at: form.deprecated_at.trim(),
    replaced_by: normalizeRegistryToken(form.replaced_by),
    deprecation_notice: form.deprecation_notice.trim(),
    platforms: normalizePlatformList([
      ...selectedPresetPlatforms.value,
      ...normalizeRegistryList(customPlatforms.value)
    ]),
    protocol_ids: normalizeRegistryList(form.protocol_ids),
    aliases: normalizeRegistryList(form.aliases),
    pricing_lookup_ids: normalizeRegistryList(form.pricing_lookup_ids),
    preferred_protocol_ids: parsePreferredProtocolIDs(form.preferred_protocol_ids),
    modalities: normalizeRegistryList(form.modalities),
    capabilities: normalizeCapabilityList(selectedCapabilities.value),
    exposed_in: normalizeExposureTargets(selectedExposures.value)
  })
}
</script>
