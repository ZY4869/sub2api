<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.daily5h.dialogTitle')"
    width="wide"
    @close="emit('close')"
  >
    <div class="space-y-5">
      <div class="rounded-xl border border-gray-200 bg-gray-50/80 p-4 text-sm text-gray-700 dark:border-dark-600 dark:bg-dark-700/30 dark:text-gray-200">
        <div class="font-medium text-gray-900 dark:text-white">
          {{ t('admin.accounts.daily5h.dialogSummaryTitle') }}
        </div>
        <p class="mt-1 leading-6">
          {{ t('admin.accounts.daily5h.dialogSummaryBody') }}
        </p>
      </div>

      <div class="space-y-3">
        <div class="flex items-center justify-between gap-3">
          <div>
            <label class="input-label mb-0">{{ t('admin.accounts.daily5h.enableLabel') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.daily5h.enableHint') }}
            </p>
          </div>
          <button
            type="button"
            :class="toggleButtonClass(localSettings.enabled)"
            @click="localSettings.enabled = !localSettings.enabled"
          >
            <span :class="toggleThumbClass(localSettings.enabled)" />
          </button>
        </div>

        <div class="flex items-center justify-between gap-3">
          <div>
            <label class="input-label mb-0">{{ t('admin.accounts.daily5h.includePausedLabel') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.daily5h.includePausedHint') }}
            </p>
          </div>
          <button
            type="button"
            :class="toggleButtonClass(localSettings.include_paused_accounts)"
            @click="localSettings.include_paused_accounts = !localSettings.include_paused_accounts"
          >
            <span :class="toggleThumbClass(localSettings.include_paused_accounts)" />
          </button>
        </div>
      </div>

      <div class="space-y-3">
        <div>
          <label class="input-label">{{ t('admin.accounts.daily5h.accountTypesLabel') }}</label>
          <p class="input-hint">{{ t('admin.accounts.daily5h.accountTypesHint') }}</p>
        </div>
        <div class="grid gap-3">
          <label
            v-for="accountType in accountTypes"
            :key="accountType.value"
            class="flex cursor-pointer items-start gap-3 rounded-xl border border-gray-200 p-4 transition hover:border-primary-300 hover:bg-primary-50/40 dark:border-dark-600 dark:hover:border-primary-700 dark:hover:bg-primary-900/10"
          >
            <input
              :checked="isSelectedAccountType(accountType.value)"
              type="checkbox"
              class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              @change="toggleAccountType(accountType.value)"
            />
            <div class="min-w-0 flex-1">
              <PlatformLabel
                :platform="accountType.platform"
                :label="t(accountType.labelKey)"
                :description="t(accountType.descriptionKey)"
              />
              <div class="mt-2 flex flex-wrap gap-2 text-xs text-gray-500 dark:text-gray-400">
                <span class="inline-flex rounded-full bg-gray-100 px-2 py-1 dark:bg-dark-700">
                  {{ t('admin.accounts.daily5h.candidateCount', { count: candidateCount(accountType.value) }) }}
                </span>
                <span class="inline-flex rounded-full bg-gray-100 px-2 py-1 dark:bg-dark-700">
                  {{ t('admin.accounts.daily5h.modelCount', { count: candidateModels(accountType.value).length }) }}
                </span>
              </div>
            </div>
          </label>
        </div>
      </div>

      <div class="space-y-4">
        <div>
          <label class="input-label">{{ t('admin.accounts.daily5h.modelConfigLabel') }}</label>
          <p class="input-hint">{{ t('admin.accounts.daily5h.modelConfigHint') }}</p>
        </div>

        <section
          v-for="section in modelSections"
          :key="section.accountType"
          class="rounded-xl border border-gray-200 p-4 dark:border-dark-600"
        >
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="min-w-0">
              <PlatformLabel
                :platform="section.platform"
                :label="t(section.labelKey)"
                :description="t(section.familyHintKey)"
              />
              <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.daily5h.modelCandidatesHint', { count: section.models.length }) }}
              </p>
            </div>
            <div class="grid min-w-[14rem] gap-2 sm:grid-cols-2">
              <button
                type="button"
                class="rounded-xl border px-3 py-2 text-left text-sm transition-all"
                :class="modeButtonClass(section.settings.mode === 'auto')"
                @click="section.settings.mode = 'auto'"
              >
                <div class="font-semibold">{{ t('admin.accounts.daily5h.modelModeAuto') }}</div>
                <div class="mt-1 text-xs opacity-80">{{ t('admin.accounts.daily5h.modelModeAutoHint') }}</div>
              </button>
              <button
                type="button"
                class="rounded-xl border px-3 py-2 text-left text-sm transition-all"
                :class="modeButtonClass(section.settings.mode === 'fixed')"
                @click="section.settings.mode = 'fixed'"
              >
                <div class="font-semibold">{{ t('admin.accounts.daily5h.modelModeFixed') }}</div>
                <div class="mt-1 text-xs opacity-80">{{ t('admin.accounts.daily5h.modelModeFixedHint') }}</div>
              </button>
            </div>
          </div>

          <div v-if="section.settings.mode === 'fixed'" class="mt-4">
            <Select
              :model-value="section.settings.fixed_model_id || null"
              :options="section.models"
              value-key="model_id"
              label-key="display_name"
              searchable
              :placeholder="t('admin.accounts.daily5h.fixedModelPlaceholder')"
              @update:model-value="section.settings.fixed_model_id = String($event || '')"
            >
              <template #selected="{ option }">
                <div v-if="option" class="flex min-w-0 items-center gap-2">
                  <ModelIcon
                    :model="modelOptionModelID(option)"
                    :provider="modelOptionProvider(option)"
                    :display-name="modelOptionDisplayName(option)"
                    size="16px"
                  />
                  <span class="truncate font-medium text-gray-900 dark:text-white">
                    {{ modelOptionDisplayName(option) }}
                  </span>
                  <span class="truncate text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.accounts.daily5h.supportedAccountsCount', { count: modelOptionAccountCount(option) }) }}
                  </span>
                </div>
                <span v-else>{{ t('admin.accounts.daily5h.fixedModelPlaceholder') }}</span>
              </template>
              <template #option="{ option }">
                <div class="flex min-w-0 flex-1 items-center justify-between gap-3">
                  <div class="min-w-0">
                    <div class="flex min-w-0 items-center gap-2">
                      <ModelIcon
                        :model="modelOptionModelID(option)"
                        :provider="modelOptionProvider(option)"
                        :display-name="modelOptionDisplayName(option)"
                        size="16px"
                      />
                      <span class="truncate font-medium text-gray-900 dark:text-white">
                        {{ modelOptionDisplayName(option) }}
                      </span>
                    </div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                      {{ modelOptionModelID(option) }}
                    </div>
                  </div>
                  <span class="shrink-0 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.accounts.daily5h.supportedAccountsCount', { count: modelOptionAccountCount(option) }) }}
                  </span>
                </div>
              </template>
            </Select>
            <p v-if="section.models.length === 0" class="mt-2 text-xs text-amber-600 dark:text-amber-300">
              {{ t('admin.accounts.daily5h.noFamilyModelsHint') }}
            </p>
          </div>
        </section>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="saving" @click="emit('close')">
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="btn btn-primary" :disabled="saving" @click="handleSave">
          <Icon v-if="saving" name="refresh" size="sm" class="mr-1 animate-spin" />
          {{ saving ? t('common.saving') : t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import PlatformLabel from '@/components/common/PlatformLabel.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import type {
  AccountDaily5HTriggerAccountType,
  AccountDaily5HTriggerAccountTypeSummary,
  AccountDaily5HTriggerModelOption,
  AccountDaily5HTriggerSettings
} from '@/types'

const props = defineProps<{
  show: boolean
  saving: boolean
  settings: AccountDaily5HTriggerSettings
  candidates: AccountDaily5HTriggerAccountTypeSummary[]
}>()

const emit = defineEmits<{
  close: []
  save: [value: AccountDaily5HTriggerSettings]
}>()

const { t } = useI18n()

type Daily5HModelSelectOption = SelectOption & AccountDaily5HTriggerModelOption

const accountTypes = [
  {
    value: 'chatgpt_oauth' as AccountDaily5HTriggerAccountType,
    platform: 'openai' as const,
    labelKey: 'admin.accounts.daily5h.accountTypeOpenAI',
    descriptionKey: 'admin.accounts.daily5h.accountTypeOpenAIHint'
  },
  {
    value: 'claude_code_oauth_setup_token' as AccountDaily5HTriggerAccountType,
    platform: 'anthropic' as const,
    labelKey: 'admin.accounts.daily5h.accountTypeAnthropic',
    descriptionKey: 'admin.accounts.daily5h.accountTypeAnthropicHint'
  },
  {
    value: 'google_oauth' as AccountDaily5HTriggerAccountType,
    platform: 'gemini' as const,
    labelKey: 'admin.accounts.daily5h.accountTypeGemini',
    descriptionKey: 'admin.accounts.daily5h.accountTypeGeminiHint'
  }
] as const

const localSettings = reactive<AccountDaily5HTriggerSettings>(createLocalSettings(props.settings))

watch(
  () => props.settings,
  (value) => {
    Object.assign(localSettings, createLocalSettings(value))
  },
  { deep: true, immediate: true }
)

const candidateMap = computed(() => {
  return props.candidates.reduce<Record<string, AccountDaily5HTriggerAccountTypeSummary>>((acc, item) => {
    acc[item.account_type] = item
    return acc
  }, {})
})

const modelSections = computed(() => [
  {
    accountType: 'chatgpt_oauth' as AccountDaily5HTriggerAccountType,
    platform: 'openai' as const,
    labelKey: 'admin.accounts.daily5h.accountTypeOpenAI',
    familyHintKey: 'admin.accounts.daily5h.familyHintOpenAI',
    settings: localSettings.openai_model_mode,
    models: candidateModels('chatgpt_oauth')
  },
  {
    accountType: 'claude_code_oauth_setup_token' as AccountDaily5HTriggerAccountType,
    platform: 'anthropic' as const,
    labelKey: 'admin.accounts.daily5h.accountTypeAnthropic',
    familyHintKey: 'admin.accounts.daily5h.familyHintAnthropic',
    settings: localSettings.anthropic_model_mode,
    models: candidateModels('claude_code_oauth_setup_token')
  },
  {
    accountType: 'google_oauth' as AccountDaily5HTriggerAccountType,
    platform: 'gemini' as const,
    labelKey: 'admin.accounts.daily5h.accountTypeGemini',
    familyHintKey: 'admin.accounts.daily5h.familyHintGemini',
    settings: localSettings.gemini_model_mode,
    models: candidateModels('google_oauth')
  }
])

function createLocalSettings(settings: AccountDaily5HTriggerSettings): AccountDaily5HTriggerSettings {
  return {
    enabled: settings?.enabled === true,
    selected_account_types: [...(settings?.selected_account_types || ['chatgpt_oauth'])],
    include_paused_accounts: settings?.include_paused_accounts === true,
    openai_model_mode: {
      mode: settings?.openai_model_mode?.mode === 'fixed' ? 'fixed' : 'auto',
      fixed_model_id: settings?.openai_model_mode?.fixed_model_id || ''
    },
    anthropic_model_mode: {
      mode: settings?.anthropic_model_mode?.mode === 'fixed' ? 'fixed' : 'auto',
      fixed_model_id: settings?.anthropic_model_mode?.fixed_model_id || ''
    },
    gemini_model_mode: {
      mode: settings?.gemini_model_mode?.mode === 'fixed' ? 'fixed' : 'auto',
      fixed_model_id: settings?.gemini_model_mode?.fixed_model_id || ''
    }
  }
}

function toggleButtonClass(enabled: boolean) {
  return [
    'relative inline-flex h-6 w-11 flex-shrink-0 rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
    enabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
  ]
}

function toggleThumbClass(enabled: boolean) {
  return [
    'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
    enabled ? 'translate-x-5' : 'translate-x-0'
  ]
}

function modeButtonClass(selected: boolean) {
  return selected
    ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-500 dark:bg-primary-900/20 dark:text-primary-200'
    : 'border-gray-200 text-gray-700 hover:border-gray-300 hover:bg-gray-50 dark:border-dark-600 dark:text-gray-300 dark:hover:border-dark-500 dark:hover:bg-dark-700/40'
}

function isSelectedAccountType(value: AccountDaily5HTriggerAccountType): boolean {
  return localSettings.selected_account_types.includes(value)
}

function toggleAccountType(value: AccountDaily5HTriggerAccountType) {
  if (isSelectedAccountType(value)) {
    localSettings.selected_account_types = localSettings.selected_account_types.filter((item) => item !== value)
    return
  }
  localSettings.selected_account_types = [...localSettings.selected_account_types, value]
}

function candidateCount(value: AccountDaily5HTriggerAccountType): number {
  return candidateMap.value[value]?.count || 0
}

function candidateModels(value: AccountDaily5HTriggerAccountType): Daily5HModelSelectOption[] {
  return (candidateMap.value[value]?.models || []).map((item) => ({
    ...item,
    value: item.model_id,
    label: item.display_name
  }))
}

function asDaily5HModelOption(option: unknown): Partial<Daily5HModelSelectOption> {
  if (typeof option !== 'object' || option === null) {
    return {}
  }
  return option as Partial<Daily5HModelSelectOption>
}

function modelOptionModelID(option: unknown): string {
  return String(asDaily5HModelOption(option).model_id || '')
}

function modelOptionDisplayName(option: unknown): string {
  return String(asDaily5HModelOption(option).display_name || '')
}

function modelOptionProvider(option: unknown): string | undefined {
  const value = asDaily5HModelOption(option).provider
  return typeof value === 'string' && value ? value : undefined
}

function modelOptionAccountCount(option: unknown): number {
  const value = asDaily5HModelOption(option).account_count
  return typeof value === 'number' ? value : 0
}

function handleSave() {
  emit('save', createLocalSettings(localSettings))
}
</script>
