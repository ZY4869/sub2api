<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AnthropicQuotaRPMStrategy } from '@/utils/accountQuotaControl'

defineProps<{
  showRpmLimit: boolean
}>()

const enableInterceptWarmup = defineModel<boolean>('enableInterceptWarmup', { required: true })
const interceptWarmupRequests = defineModel<boolean>('interceptWarmupRequests', { required: true })
const enableRpmLimit = defineModel<boolean>('enableRpmLimit', { required: true })
const rpmLimitEnabled = defineModel<boolean>('rpmLimitEnabled', { required: true })
const bulkBaseRpm = defineModel<number | null>('bulkBaseRpm', { required: true })
const bulkRpmStrategy = defineModel<AnthropicQuotaRPMStrategy>('bulkRpmStrategy', { required: true })
const bulkRpmStickyBuffer = defineModel<number | null>('bulkRpmStickyBuffer', { required: true })
const userMsgQueueMode = defineModel<string | null>('userMsgQueueMode', { required: true })

const { t } = useI18n()

const umqModeOptions = computed(() => [
  { value: '', label: t('admin.accounts.quotaControl.rpmLimit.umqModeOff') },
  { value: 'throttle', label: t('admin.accounts.quotaControl.rpmLimit.umqModeThrottle') },
  { value: 'serialize', label: t('admin.accounts.quotaControl.rpmLimit.umqModeSerialize') }
])

const toggleUserMsgQueueMode = (value: string) => {
  userMsgQueueMode.value = userMsgQueueMode.value === value ? null : value
}
</script>

<template>
  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <div class="flex items-center justify-between">
      <div class="flex-1 pr-4">
        <label
          id="bulk-edit-intercept-warmup-label"
          class="input-label mb-0"
          for="bulk-edit-intercept-warmup-enabled"
        >
          {{ t('admin.accounts.interceptWarmupRequests') }}
        </label>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.interceptWarmupRequestsDesc') }}
        </p>
      </div>
      <input
        v-model="enableInterceptWarmup"
        id="bulk-edit-intercept-warmup-enabled"
        type="checkbox"
        aria-controls="bulk-edit-intercept-warmup-body"
        class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
      />
    </div>
    <div v-if="enableInterceptWarmup" id="bulk-edit-intercept-warmup-body" class="mt-3">
      <button
        type="button"
        :class="[
          'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
          interceptWarmupRequests ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
        ]"
        @click="interceptWarmupRequests = !interceptWarmupRequests"
      >
        <span
          :class="[
            'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
            interceptWarmupRequests ? 'translate-x-5' : 'translate-x-0'
          ]"
        />
      </button>
    </div>
  </div>

  <div v-if="showRpmLimit" class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <div class="mb-3 flex items-center justify-between">
      <label
        id="bulk-edit-rpm-limit-label"
        class="input-label mb-0"
        for="bulk-edit-rpm-limit-enabled"
      >
        {{ t('admin.accounts.quotaControl.rpmLimit.label') }}
      </label>
      <input
        v-model="enableRpmLimit"
        id="bulk-edit-rpm-limit-enabled"
        type="checkbox"
        aria-controls="bulk-edit-rpm-limit-body"
        class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
      />
    </div>

    <div
      id="bulk-edit-rpm-limit-body"
      :class="!enableRpmLimit && 'pointer-events-none opacity-50'"
      role="group"
      aria-labelledby="bulk-edit-rpm-limit-label"
    >
      <div class="mb-3 flex items-center justify-between">
        <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.accounts.quotaControl.rpmLimit.hint') }}</span>
        <button
          type="button"
          @click="rpmLimitEnabled = !rpmLimitEnabled"
          :class="[
            'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
            rpmLimitEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
          ]"
        >
          <span
            :class="[
              'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
              rpmLimitEnabled ? 'translate-x-5' : 'translate-x-0'
            ]"
          />
        </button>
      </div>

      <div v-if="rpmLimitEnabled" class="space-y-3">
        <div>
          <label class="input-label text-xs">{{ t('admin.accounts.quotaControl.rpmLimit.baseRpm') }}</label>
          <input
            v-model.number="bulkBaseRpm"
            type="number"
            min="1"
            max="1000"
            step="1"
            class="input"
            :placeholder="t('admin.accounts.quotaControl.rpmLimit.baseRpmPlaceholder')"
          />
          <p class="input-hint">{{ t('admin.accounts.quotaControl.rpmLimit.baseRpmHint') }}</p>
        </div>

        <div>
          <label class="input-label text-xs">{{ t('admin.accounts.quotaControl.rpmLimit.strategy') }}</label>
          <div class="flex gap-2">
            <button
              type="button"
              @click="bulkRpmStrategy = 'tiered'"
              :class="[
                'flex-1 rounded-lg px-3 py-2 text-sm font-medium transition-all',
                bulkRpmStrategy === 'tiered'
                  ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
              ]"
            >
              {{ t('admin.accounts.quotaControl.rpmLimit.strategyTiered') }}
            </button>
            <button
              type="button"
              @click="bulkRpmStrategy = 'sticky_exempt'"
              :class="[
                'flex-1 rounded-lg px-3 py-2 text-sm font-medium transition-all',
                bulkRpmStrategy === 'sticky_exempt'
                  ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
              ]"
            >
              {{ t('admin.accounts.quotaControl.rpmLimit.strategyStickyExempt') }}
            </button>
          </div>
        </div>

        <div v-if="bulkRpmStrategy === 'tiered'">
          <label class="input-label text-xs">{{ t('admin.accounts.quotaControl.rpmLimit.stickyBuffer') }}</label>
          <input
            v-model.number="bulkRpmStickyBuffer"
            type="number"
            min="1"
            step="1"
            class="input"
            :placeholder="t('admin.accounts.quotaControl.rpmLimit.stickyBufferPlaceholder')"
          />
          <p class="input-hint">{{ t('admin.accounts.quotaControl.rpmLimit.stickyBufferHint') }}</p>
        </div>
      </div>

      <div class="mt-4">
        <label class="input-label">{{ t('admin.accounts.quotaControl.rpmLimit.userMsgQueue') }}</label>
        <p class="mb-2 mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.quotaControl.rpmLimit.userMsgQueueHint') }}
        </p>
        <div class="flex space-x-2">
          <button
            v-for="opt in umqModeOptions"
            :key="opt.value"
            type="button"
            @click="toggleUserMsgQueueMode(opt.value)"
            :class="[
              'rounded-md border px-3 py-1.5 text-sm transition-colors',
              userMsgQueueMode === opt.value
                ? 'border-primary-600 bg-primary-600 text-white'
                : 'border-gray-300 bg-white text-gray-700 hover:bg-gray-50 dark:border-dark-500 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'
            ]"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
