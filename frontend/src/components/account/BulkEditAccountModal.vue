<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.bulkEdit.title')"
    width="wide"
    @close="handleClose"
  >
    <form id="bulk-edit-account-form" class="space-y-5" @submit.prevent="handleSubmit">
      <!-- Info -->
      <div class="rounded-lg bg-blue-50 p-4 dark:bg-blue-900/20">
        <p class="text-sm text-blue-700 dark:text-blue-400">
          <svg class="mr-1.5 inline h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          {{ t('admin.accounts.bulkEdit.selectionInfo', { count: accountIds.length }) }}
        </p>
      </div>

      <!-- Mixed platform warning -->
      <div v-if="isMixedPlatform" class="rounded-lg bg-amber-50 p-4 dark:bg-amber-900/20">
        <p class="text-sm text-amber-700 dark:text-amber-400">
          <svg class="mr-1.5 inline h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          {{ t('admin.accounts.bulkEdit.mixedPlatformWarning', { platforms: selectedPlatforms.join(', ') }) }}
        </p>
      </div>

      <!-- Base URL (API Key only) -->
      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div class="mb-3 flex items-center justify-between">
          <label
            id="bulk-edit-base-url-label"
            class="input-label mb-0"
            for="bulk-edit-base-url-enabled"
          >
            {{ t('admin.accounts.baseUrl') }}
          </label>
          <input
            v-model="enableBaseUrl"
            id="bulk-edit-base-url-enabled"
            type="checkbox"
            aria-controls="bulk-edit-base-url"
            class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
        </div>
        <input
          v-model="baseUrl"
          id="bulk-edit-base-url"
          type="text"
          :disabled="!enableBaseUrl"
          class="input"
          :class="!enableBaseUrl && 'cursor-not-allowed opacity-50'"
          :placeholder="t('admin.accounts.bulkEdit.baseUrlPlaceholder')"
          aria-labelledby="bulk-edit-base-url-label"
        />
        <p class="input-hint">
          {{ t('admin.accounts.bulkEdit.baseUrlNotice') }}
        </p>
      </div>

      <BulkEditModelRestrictionSection
        v-model:enabled="enableModelRestriction"
        v-model:mode="modelRestrictionMode"
        v-model:allowed-models="allowedModels"
        v-model:model-mappings="modelMappings"
        :models="allModels"
        :presets="presetMappings"
      />

      <BulkEditCustomErrorCodesSection
        v-model:enabled="enableCustomErrorCodes"
        v-model:selected-codes="selectedErrorCodes"
        v-model:input="customErrorCodeInput"
        :error-code-options="commonErrorCodes"
      />

      <BulkEditAnthropicControlSection
        v-model:enable-intercept-warmup="enableInterceptWarmup"
        v-model:intercept-warmup-requests="interceptWarmupRequests"
        v-model:enable-rpm-limit="enableRpmLimit"
        v-model:rpm-limit-enabled="rpmLimitEnabled"
        v-model:bulk-base-rpm="bulkBaseRpm"
        v-model:bulk-rpm-strategy="bulkRpmStrategy"
        v-model:bulk-rpm-sticky-buffer="bulkRpmStickyBuffer"
        v-model:user-msg-queue-mode="userMsgQueueMode"
        :show-rpm-limit="allAnthropicOAuthOrSetupToken"
      />

      <BulkEditRuntimeFieldsSection
        v-model:enable-proxy="enableProxy"
        v-model:proxy-id="proxyId"
        v-model:enable-concurrency="enableConcurrency"
        v-model:concurrency="concurrency"
        v-model:enable-load-factor="enableLoadFactor"
        v-model:load-factor="loadFactor"
        v-model:enable-priority="enablePriority"
        v-model:priority="priority"
        v-model:enable-rate-multiplier="enableRateMultiplier"
        v-model:rate-multiplier="rateMultiplier"
        :proxies="proxies"
      />

      <BulkEditStatusGroupSection
        v-model:enable-status="enableStatus"
        v-model:status="status"
        v-model:enable-groups="enableGroups"
        v-model:group-ids="groupIds"
        :groups="groups"
      />
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="bulk-edit-account-form"
          :disabled="submitting"
          class="btn btn-primary"
        >
          <svg
            v-if="submitting"
            class="-ml-1 mr-2 h-4 w-4 animate-spin"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              class="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              stroke-width="4"
            />
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            />
          </svg>
          {{
            submitting ? t('admin.accounts.bulkEdit.updating') : t('admin.accounts.bulkEdit.submit')
          }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <ConfirmDialog
    :show="showMixedChannelWarning"
    :title="t('admin.accounts.mixedChannelWarningTitle')"
    :message="mixedChannelWarningMessage"
    :confirm-text="t('common.confirm')"
    :cancel-text="t('common.cancel')"
    :danger="true"
    @confirm="handleMixedChannelConfirm"
    @cancel="handleMixedChannelCancel"
  />
</template>

<script setup lang="ts">
import { computed, ref, toRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type {
  Proxy as ProxyConfig,
  AdminGroup,
  AccountPlatform,
  AccountType,
  UpdateAccountRequest
} from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import BulkEditAnthropicControlSection from './BulkEditAnthropicControlSection.vue'
import BulkEditCustomErrorCodesSection from './BulkEditCustomErrorCodesSection.vue'
import BulkEditModelRestrictionSection from './BulkEditModelRestrictionSection.vue'
import BulkEditRuntimeFieldsSection from './BulkEditRuntimeFieldsSection.vue'
import BulkEditStatusGroupSection from './BulkEditStatusGroupSection.vue'
import { useBulkEditAccountForm } from '@/composables/useBulkEditAccountForm'
import { useAccountMixedChannelRisk } from '@/composables/useAccountMixedChannelRisk'
import { commonErrorCodes } from '@/composables/useModelWhitelist'

interface Props {
  show: boolean
  accountIds: number[]
  selectedPlatforms: AccountPlatform[]
  selectedTypes: AccountType[]
  proxies: ProxyConfig[]
  groups: AdminGroup[]
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
  updated: []
}>()

const { t } = useI18n()
const appStore = useAppStore()

// Platform awareness
const isMixedPlatform = computed(() => props.selectedPlatforms.length > 1)

const submitting = ref(false)
const {
  allAnthropicOAuthOrSetupToken,
  allModels,
  allowedModels,
  baseUrl,
  buildUpdatePayload,
  bulkBaseRpm,
  bulkRpmStickyBuffer,
  bulkRpmStrategy,
  canPreCheck,
  concurrency,
  customErrorCodeInput,
  enableBaseUrl,
  enableConcurrency,
  enableCustomErrorCodes,
  enableGroups,
  enableInterceptWarmup,
  enableLoadFactor,
  enableModelRestriction,
  enablePriority,
  enableProxy,
  enableRateMultiplier,
  enableRpmLimit,
  enableStatus,
  groupIds,
  hasAnyFieldEnabled,
  interceptWarmupRequests,
  loadFactor,
  modelMappings,
  modelRestrictionMode,
  presetMappings,
  priority,
  proxyId,
  rateMultiplier,
  resetFormState,
  rpmLimitEnabled,
  selectedErrorCodes,
  status,
  userMsgQueueMode
} = useBulkEditAccountForm({
  selectedPlatforms: toRef(props, 'selectedPlatforms'),
  selectedTypes: toRef(props, 'selectedTypes')
})

const {
  showWarning: showMixedChannelWarning,
  warningMessageText: mixedChannelWarningMessage,
  openDialog: openMixedChannelDialog,
  withConfirmFlag,
  ensureConfirmed: ensureMixedChannelConfirmed,
  handleConfirm: handleMixedChannelConfirm,
  handleCancel: handleMixedChannelCancel,
  reset: resetMixedChannelRisk
} = useAccountMixedChannelRisk({
  currentPlatform: () => (canPreCheck() ? props.selectedPlatforms[0] : null),
  buildCheckPayload: () => {
    if (!canPreCheck()) {
      return null
    }
    return {
      platform: props.selectedPlatforms[0],
      group_ids: groupIds.value
    }
  },
  buildWarningText: (details) => t('admin.accounts.mixedChannelWarning', { ...details }),
  fallbackMessage: () => t('admin.accounts.bulkEdit.failed'),
  showError: (message) => appStore.showError(message)
})

const handleClose = () => {
  resetMixedChannelRisk()
  emit('close')
}

const handleSubmit = async () => {
  if (props.accountIds.length === 0) {
    appStore.showError(t('admin.accounts.bulkEdit.noSelection'))
    return
  }

  if (!hasAnyFieldEnabled.value) {
    appStore.showError(t('admin.accounts.bulkEdit.noFieldsSelected'))
    return
  }

  const built = buildUpdatePayload()
  if (!built) {
    appStore.showError(t('admin.accounts.bulkEdit.noFieldsSelected'))
    return
  }

  const canContinue = await ensureMixedChannelConfirmed(async () => {
    await submitBulkUpdate(built)
  })
  if (!canContinue) return

  await submitBulkUpdate(built)
}

const submitBulkUpdate = async (baseUpdates: Partial<UpdateAccountRequest>) => {
  submitting.value = true

  try {
    const res = await adminAPI.accounts.bulkUpdate(
      props.accountIds,
      withConfirmFlag(baseUpdates) as Record<string, unknown>
    )
    const success = res.success || 0
    const failed = res.failed || 0

    if (success > 0 && failed === 0) {
      appStore.showSuccess(t('admin.accounts.bulkEdit.success', { count: success }))
    } else if (success > 0) {
      appStore.showError(t('admin.accounts.bulkEdit.partialSuccess', { success, failed }))
    } else {
      appStore.showError(t('admin.accounts.bulkEdit.failed'))
    }

    if (success > 0) {
      emit('updated')
      handleClose()
    }
  } catch (error: any) {
    if (error.status === 409 && error.error === 'mixed_channel_warning') {
      openMixedChannelDialog({
        message: error.message,
        onConfirm: async () => submitBulkUpdate(baseUpdates)
      })
    } else {
      appStore.showError(error.message || t('admin.accounts.bulkEdit.failed'))
      console.error('Error bulk updating accounts:', error)
    }
  } finally {
    submitting.value = false
  }
}

// Reset form when modal closes
watch(
  () => props.show,
  (newShow) => {
    if (!newShow) {
      resetMixedChannelRisk()
      resetFormState()
    }
  }
)
</script>
