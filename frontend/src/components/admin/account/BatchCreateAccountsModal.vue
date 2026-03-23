<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.batchCreateTitle')"
    width="wide"
    close-on-click-outside
    @close="handleClose"
  >
    <form id="batch-create-accounts-form" class="space-y-5" @submit.prevent="handleSubmit">
      <div class="text-sm text-gray-600 dark:text-dark-300">
        {{ t('admin.accounts.batchCreateHint') }}
      </div>

      <div
        v-if="form.platform === 'kiro'"
        class="rounded-lg border border-sky-200 bg-sky-50 px-4 py-3 text-sm text-sky-700 dark:border-sky-800/60 dark:bg-sky-900/20 dark:text-sky-200"
      >
        {{ t('admin.accounts.batchCreateKiroTempHint') }}
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.accounts.platform') }}</label>
          <select v-model="form.platform" class="input">
            <option v-for="platform in platformOptions" :key="platform" :value="platform">
              {{ t(`admin.accounts.platforms.${platform}`) }}
            </option>
          </select>
        </div>

        <div>
          <label class="input-label">{{ t('admin.accounts.accountType') }}</label>
          <select v-model="form.type" class="input">
            <option v-for="type in currentTypeOptions" :key="type" :value="type">
              {{ resolveTypeLabel(type) }}
            </option>
          </select>
        </div>
      </div>

      <div>
        <label class="input-label">{{ t('admin.accounts.batchCreateNamePrefix') }}</label>
        <input
          v-model.trim="form.name_prefix"
          type="text"
          class="input"
          :placeholder="defaultNamePrefix(form.platform)"
        />
        <p class="input-hint">{{ t('admin.accounts.batchCreateNamePrefixHint') }}</p>
      </div>

      <div v-if="showBaseUrlField">
        <label class="input-label">{{ t('admin.accounts.baseUrl') }}</label>
        <input
          v-model.trim="form.base_url"
          type="text"
          class="input"
          :placeholder="resolveBaseUrlPlaceholder()"
        />
        <p class="input-hint">{{ resolveBaseUrlHint() }}</p>
      </div>

      <div>
        <div class="flex items-center justify-between gap-3">
          <label class="input-label">{{ t('admin.accounts.batchCreateLineInput') }}</label>
          <span class="text-xs text-gray-500 dark:text-dark-400">
            {{ t('admin.accounts.batchCreateNonEmptyLineCount', { count: nonEmptyLineCount }) }}
          </span>
        </div>
        <textarea
          v-model="form.items_text"
          rows="10"
          class="input font-mono text-sm"
          :placeholder="t('admin.accounts.batchCreateLineInputPlaceholder')"
        ></textarea>
        <p class="input-hint">{{ t('admin.accounts.batchCreateLineInputHint') }}</p>
      </div>

      <div>
        <label class="input-label">{{ t('admin.accounts.notes') }}</label>
        <textarea
          v-model.trim="form.notes"
          rows="3"
          class="input"
          :placeholder="t('admin.accounts.notesPlaceholder')"
        ></textarea>
      </div>

      <AccountRuntimeSettingsEditor
        v-model:proxy-id="form.proxy_id"
        v-model:concurrency="form.concurrency"
        v-model:load-factor="form.load_factor"
        v-model:priority="form.priority"
        v-model:rate-multiplier="form.rate_multiplier"
        v-model:expires-at-input="form.expires_at_input"
        :proxies="proxies"
      />

      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <label class="input-label">{{ t('admin.users.groups') }}</label>
        <div
          v-if="form.archive.enabled"
          class="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:border-amber-800/60 dark:bg-amber-900/20 dark:text-amber-200"
        >
          {{ t('admin.accounts.batchCreateArchiveGroupsDisabled') }}
        </div>
        <GroupSelector
          v-else
          v-model="form.group_ids"
          :groups="groups"
          :platform="form.platform"
        />
      </div>

      <div class="space-y-4 border-t border-gray-200 pt-4 dark:border-dark-600">
        <label class="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300">
          <input
            v-model="form.archive.enabled"
            type="checkbox"
            class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <span>{{ t('admin.accounts.batchCreateArchiveToggle') }}</span>
        </label>

        <div v-if="form.archive.enabled" class="space-y-3">
          <div>
            <label class="input-label">{{ t('admin.accounts.batchCreateArchiveGroupName') }}</label>
            <input
              v-model.trim="form.archive.group_name"
              type="text"
              class="input"
              :placeholder="t('admin.accounts.batchCreateArchiveGroupName')"
            />
            <p class="input-hint">{{ t('admin.accounts.batchCreateArchiveGroupHint') }}</p>
          </div>
          <div
            class="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:border-amber-800/60 dark:bg-amber-900/20 dark:text-amber-200"
          >
            {{ t('admin.accounts.batchCreateArchiveNotice') }}
          </div>
        </div>

        <div class="flex flex-wrap gap-4">
          <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <input
              v-model="form.auto_pause_on_expired"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            />
            <span>{{ t('admin.accounts.autoPauseOnExpired') }}</span>
          </label>

          <label
            :class="[
              'flex items-center gap-2 text-sm',
              form.archive.enabled
                ? 'cursor-not-allowed text-gray-400 dark:text-dark-500'
                : 'text-gray-700 dark:text-gray-300'
            ]"
          >
            <input
              v-model="form.auto_import_models"
              type="checkbox"
              :disabled="form.archive.enabled"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            />
            <span>{{ t('admin.accounts.autoImportModels') }}</span>
          </label>
        </div>

        <p v-if="form.archive.enabled" class="text-xs text-gray-500 dark:text-dark-400">
          {{ t('admin.accounts.batchCreateAutoImportDisabledHint') }}
        </p>
      </div>

      <div
        v-if="result"
        class="space-y-3 rounded-xl border border-gray-200 p-4 dark:border-dark-700"
      >
        <div class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.accounts.batchCreateResultTitle') }}
        </div>
        <div class="text-sm text-gray-700 dark:text-dark-300">
          {{ t('admin.accounts.batchCreateResultSummary', result) }}
        </div>
        <div
          v-if="result.archive_group_name"
          class="text-xs text-gray-500 dark:text-dark-400"
        >
          {{ t('admin.accounts.batchCreateResultArchive', { group: result.archive_group_name }) }}
        </div>

        <div class="max-h-64 overflow-auto rounded-lg bg-gray-50 p-3 text-xs dark:bg-dark-800">
          <div
            v-for="line in result.results"
            :key="`${line.line_index}-${line.raw_preview}`"
            :class="[
              'grid grid-cols-[auto,1fr] gap-x-3 gap-y-1 border-b border-gray-200 py-2 last:border-b-0 dark:border-dark-700',
              line.success ? 'text-emerald-700 dark:text-emerald-300' : 'text-red-600 dark:text-red-300'
            ]"
          >
            <span class="font-mono">#{{ line.line_index }}</span>
            <div class="space-y-1">
              <div class="font-mono text-[11px] opacity-80">{{ line.raw_preview || '-' }}</div>
              <div>{{ line.message }}</div>
            </div>
          </div>
        </div>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" type="button" :disabled="submitting" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button class="btn btn-primary" type="submit" form="batch-create-accounts-form" :disabled="submitting">
          {{ submitting ? t('admin.accounts.batchCreateSubmitting') : t('admin.accounts.batchCreateSubmit') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { BatchCreateAccountsResult } from '@/types'
import type { AccountPlatform, AccountType, AdminGroup, Proxy } from '@/types'
import { useAppStore } from '@/stores/app'
import { parseDateTimeLocalInput } from '@/utils/format'
import BaseDialog from '@/components/common/BaseDialog.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import AccountRuntimeSettingsEditor from '@/components/account/AccountRuntimeSettingsEditor.vue'

type BatchCreatePlatform = Exclude<AccountPlatform, 'copilot'>

const BATCH_CREATE_DEFAULT_PLATFORM: BatchCreatePlatform = 'anthropic'
const BATCH_CREATE_ARCHIVE_GROUP_STORAGE_PREFIX = 'accounts-batch-create-archive-group:'

const typeOptionsByPlatform: Record<BatchCreatePlatform, AccountType[]> = {
  anthropic: ['oauth', 'setup-token', 'apikey'],
  kiro: ['oauth'],
  openai: ['oauth', 'apikey'],
  gemini: ['oauth', 'apikey'],
  antigravity: ['oauth', 'apikey', 'upstream'],
  sora: ['oauth', 'apikey']
}

const baseURLRequiredTypes = new Set<string>(['upstream'])

const props = defineProps<{
  show: boolean
  proxies: Proxy[]
  groups: AdminGroup[]
}>()

const emit = defineEmits<{
  close: []
  created: [result: BatchCreateAccountsResult]
}>()

const { t } = useI18n()
const appStore = useAppStore()

const result = ref<BatchCreateAccountsResult | null>(null)
const submitting = ref(false)
const autoGeneratedNamePrefix = ref('')

const form = reactive({
  platform: BATCH_CREATE_DEFAULT_PLATFORM as BatchCreatePlatform,
  type: typeOptionsByPlatform[BATCH_CREATE_DEFAULT_PLATFORM][0],
  name_prefix: '',
  items_text: '',
  notes: '',
  base_url: '',
  proxy_id: null as number | null,
  concurrency: 10,
  load_factor: null as number | null,
  priority: 1,
  rate_multiplier: 1,
  expires_at_input: '',
  auto_pause_on_expired: false,
  group_ids: [] as number[],
  auto_import_models: false,
  archive: {
    enabled: false,
    group_name: ''
  }
})

const platformOptions = Object.keys(typeOptionsByPlatform) as BatchCreatePlatform[]

const currentTypeOptions = computed(() => typeOptionsByPlatform[form.platform] || ['oauth'])

const showBaseUrlField = computed(() => form.type === 'apikey' || form.type === 'upstream')

const requiresBaseURL = computed(() =>
  baseURLRequiredTypes.has(form.type) || (form.platform === 'sora' && form.type === 'apikey')
)

const nonEmptyLineCount = computed(() =>
  form.items_text
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean).length
)

const defaultNamePrefix = (platform: BatchCreatePlatform) => {
  const now = new Date()
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${platform}-batch-${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}-${pad(now.getHours())}${pad(now.getMinutes())}`
}

const archiveGroupStorageKey = (platform: BatchCreatePlatform) =>
  `${BATCH_CREATE_ARCHIVE_GROUP_STORAGE_PREFIX}${platform}`

const loadStoredArchiveGroupName = (platform: BatchCreatePlatform) => {
  try {
    return localStorage.getItem(archiveGroupStorageKey(platform)) || ''
  } catch {
    return ''
  }
}

const persistArchiveGroupName = (platform: BatchCreatePlatform, groupName: string) => {
  try {
    const trimmed = groupName.trim()
    const key = archiveGroupStorageKey(platform)
    if (trimmed) {
      localStorage.setItem(key, trimmed)
    } else {
      localStorage.removeItem(key)
    }
  } catch {
    // Ignore localStorage failures.
  }
}

const resetForm = () => {
  form.platform = BATCH_CREATE_DEFAULT_PLATFORM
  form.type = typeOptionsByPlatform[BATCH_CREATE_DEFAULT_PLATFORM][0]
  autoGeneratedNamePrefix.value = defaultNamePrefix(form.platform)
  form.name_prefix = autoGeneratedNamePrefix.value
  form.items_text = ''
  form.notes = ''
  form.base_url = ''
  form.proxy_id = null
  form.concurrency = 10
  form.load_factor = null
  form.priority = 1
  form.rate_multiplier = 1
  form.expires_at_input = ''
  form.auto_pause_on_expired = false
  form.group_ids = []
  form.auto_import_models = false
  form.archive.enabled = false
  form.archive.group_name = loadStoredArchiveGroupName(form.platform)
  result.value = null
}

const resolveTypeLabel = (type: AccountType) => {
  switch (type) {
    case 'apikey':
      return t('admin.accounts.apiKey')
    case 'setup-token':
      return t('admin.accounts.setupToken')
    case 'upstream':
      return t('admin.accounts.types.upstream')
    default:
      return t(`admin.accounts.types.${type}`)
  }
}

const resolveBaseUrlPlaceholder = () => {
  if (form.platform === 'antigravity') {
    return 'https://cloudcode-pa.googleapis.com'
  }
  if (form.platform === 'sora') {
    return 'https://your-upstream.example.com'
  }
  if (form.platform === 'openai') {
    return 'https://api.openai.com'
  }
  if (form.platform === 'gemini') {
    return 'https://generativelanguage.googleapis.com'
  }
  return 'https://api.anthropic.com'
}

const resolveBaseUrlHint = () => {
  if (form.platform === 'antigravity' && form.type === 'upstream') {
    return t('admin.accounts.upstream.baseUrlHint')
  }
  if (form.platform === 'sora') {
    return t('admin.accounts.types.soraApiKeyHint')
  }
  if (form.platform === 'openai') {
    return t('admin.accounts.openai.baseUrlHint')
  }
  return t('admin.accounts.baseUrlHint')
}

const buildCredentials = () => {
  const credentials: Record<string, unknown> = {}
  const baseURL = form.base_url.trim()
  if (baseURL) {
    credentials.base_url = baseURL
  }
  return credentials
}

const handleClose = () => {
  if (submitting.value) return
  emit('close')
}

const handleSubmit = async () => {
  const items = form.items_text
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean)

  if (items.length === 0) {
    appStore.showError(t('admin.accounts.batchCreateNoLines'))
    return
  }
  if (requiresBaseURL.value && !form.base_url.trim()) {
    appStore.showError(
      form.platform === 'sora'
        ? t('admin.accounts.types.soraBaseUrlRequired')
        : t('admin.accounts.upstream.pleaseEnterBaseUrl')
    )
    return
  }
  if (form.archive.enabled && !form.archive.group_name.trim()) {
    appStore.showError(t('admin.accounts.batchCreateArchiveGroupHint'))
    return
  }

  submitting.value = true
  try {
    const payload = {
      platform: form.platform,
      type: form.type,
      items,
      name_prefix: form.name_prefix.trim() || undefined,
      notes: form.notes.trim() || undefined,
      credentials: buildCredentials(),
      proxy_id: form.proxy_id ?? undefined,
      concurrency: form.concurrency,
      load_factor: form.load_factor ?? undefined,
      priority: form.priority,
      rate_multiplier: form.rate_multiplier,
      group_ids: form.archive.enabled ? [] : [...form.group_ids],
      expires_at: parseDateTimeLocalInput(form.expires_at_input),
      auto_pause_on_expired: form.auto_pause_on_expired,
      auto_import_models: form.archive.enabled ? false : form.auto_import_models,
      archive: {
        enabled: form.archive.enabled,
        group_name: form.archive.group_name.trim()
      }
    }

    const response = await adminAPI.accounts.batchCreateAccounts(payload)
    result.value = response
    emit('created', response)

    if (response.failed_count > 0 && response.created_count > 0) {
      appStore.showWarning(
        t('admin.accounts.batchCreatePartial', {
          created: response.created_count,
          failed: response.failed_count
        })
      )
    } else if (response.failed_count > 0) {
      appStore.showError(
        t('admin.accounts.batchCreateFailed', {
          failed: response.failed_count
        })
      )
    } else {
      appStore.showSuccess(
        t('admin.accounts.batchCreateSuccess', {
          created: response.created_count
        })
      )
    }
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.failedToCreate'))
  } finally {
    submitting.value = false
  }
}

watch(
  () => props.show,
  (open) => {
    if (open) {
      resetForm()
    }
  },
  { immediate: true }
)

watch(
  () => form.platform,
  (nextPlatform) => {
    const allowedTypes = typeOptionsByPlatform[nextPlatform]
    if (!allowedTypes.includes(form.type)) {
      form.type = allowedTypes[0]
    }
    if (!form.name_prefix || form.name_prefix === autoGeneratedNamePrefix.value) {
      autoGeneratedNamePrefix.value = defaultNamePrefix(nextPlatform)
      form.name_prefix = autoGeneratedNamePrefix.value
    }
    form.archive.group_name = loadStoredArchiveGroupName(nextPlatform)
  }
)

watch(
  () => form.archive.enabled,
  (enabled) => {
    if (enabled) {
      form.auto_import_models = false
    }
  }
)

watch(
  () => form.archive.group_name,
  (groupName) => {
    persistArchiveGroupName(form.platform, groupName)
  }
)
</script>
