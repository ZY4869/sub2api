<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { TLSFingerprintProfile } from '@/api/admin/tlsFingerprintProfile'
import type { AnthropicQuotaControlState } from '@/utils/accountQuotaControl'

interface Option {
  value: string
  label: string
}

defineProps<{
  umqModeOptions: Option[]
}>()
const state = defineModel<AnthropicQuotaControlState>('state', { required: true })

const { t } = useI18n()
const tlsFingerprintProfiles = ref<TLSFingerprintProfile[]>([])
const tlsFingerprintProfilesLoading = ref(false)
const tlsFingerprintProfilesLoaded = ref(false)

const loadTLSFingerprintProfiles = async () => {
  if (tlsFingerprintProfilesLoading.value || tlsFingerprintProfilesLoaded.value) {
    return
  }
  tlsFingerprintProfilesLoading.value = true
  try {
    tlsFingerprintProfiles.value = await adminAPI.tlsFingerprintProfiles.list()
    tlsFingerprintProfilesLoaded.value = true
  } catch (error) {
    console.error('Failed to load TLS fingerprint profiles:', error)
  } finally {
    tlsFingerprintProfilesLoading.value = false
  }
}

watch(
  () => state.value.tlsFingerprintEnabled,
  (enabled) => {
    if (enabled) {
      loadTLSFingerprintProfiles().catch((error) => {
        console.error('Failed to initialize TLS fingerprint profiles:', error)
      })
    }
  },
  { immediate: true }
)

const tlsFingerprintProfileValue = computed({
  get: () =>
    state.value.tlsFingerprintProfileId === null
      ? 'default'
      : String(state.value.tlsFingerprintProfileId),
  set: (value: string) => {
    if (value === 'default') {
      state.value.tlsFingerprintProfileId = null
      return
    }
    const parsed = Number.parseInt(value, 10)
    state.value.tlsFingerprintProfileId = Number.isFinite(parsed) ? parsed : null
  }
})

const selectedTLSFingerprintProfile = computed(() =>
  tlsFingerprintProfiles.value.find((profile) => profile.id === state.value.tlsFingerprintProfileId) ?? null
)

const missingTLSFingerprintProfileId = computed(() => {
  const profileId = state.value.tlsFingerprintProfileId
  if (profileId === null || profileId <= 0) {
    return null
  }
  return selectedTLSFingerprintProfile.value ? null : profileId
})
</script>

<template>
  <div class="border-t border-gray-200 pt-4 dark:border-dark-600 space-y-4">
    <div class="mb-3">
      <h3 class="input-label mb-0 text-base font-semibold">{{ t('admin.accounts.quotaControl.title') }}</h3>
      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.accounts.quotaControl.hint') }}
      </p>
    </div>

    <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
      <div class="mb-3 flex items-center justify-between">
        <div>
          <label class="input-label mb-0">{{ t('admin.accounts.quotaControl.windowCost.label') }}</label>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.quotaControl.windowCost.hint') }}
          </p>
        </div>
        <button
          type="button"
          @click="state.windowCostEnabled = !state.windowCostEnabled"
          :class="[
            'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
            state.windowCostEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
          ]"
        >
          <span
            :class="[
              'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
              state.windowCostEnabled ? 'translate-x-5' : 'translate-x-0'
            ]"
          />
        </button>
      </div>

      <div v-if="state.windowCostEnabled" class="grid grid-cols-2 gap-4">
        <div>
          <label class="input-label">{{ t('admin.accounts.quotaControl.windowCost.limit') }}</label>
          <div class="relative">
            <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 dark:text-gray-400">$</span>
            <input
              v-model.number="state.windowCostLimit"
              type="number"
              min="0"
              step="1"
              class="input pl-7"
              :placeholder="t('admin.accounts.quotaControl.windowCost.limitPlaceholder')"
            />
          </div>
          <p class="input-hint">{{ t('admin.accounts.quotaControl.windowCost.limitHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.quotaControl.windowCost.stickyReserve') }}</label>
          <div class="relative">
            <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 dark:text-gray-400">$</span>
            <input
              v-model.number="state.windowCostStickyReserve"
              type="number"
              min="0"
              step="1"
              class="input pl-7"
              :placeholder="t('admin.accounts.quotaControl.windowCost.stickyReservePlaceholder')"
            />
          </div>
          <p class="input-hint">{{ t('admin.accounts.quotaControl.windowCost.stickyReserveHint') }}</p>
        </div>
      </div>
    </div>

    <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
      <div class="mb-3 flex items-center justify-between">
        <div>
          <label class="input-label mb-0">{{ t('admin.accounts.quotaControl.sessionLimit.label') }}</label>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.quotaControl.sessionLimit.hint') }}
          </p>
        </div>
        <button
          type="button"
          @click="state.sessionLimitEnabled = !state.sessionLimitEnabled"
          :class="[
            'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
            state.sessionLimitEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
          ]"
        >
          <span
            :class="[
              'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
              state.sessionLimitEnabled ? 'translate-x-5' : 'translate-x-0'
            ]"
          />
        </button>
      </div>

      <div v-if="state.sessionLimitEnabled" class="grid grid-cols-2 gap-4">
        <div>
          <label class="input-label">{{ t('admin.accounts.quotaControl.sessionLimit.maxSessions') }}</label>
          <input
            v-model.number="state.maxSessions"
            type="number"
            min="1"
            step="1"
            class="input"
            :placeholder="t('admin.accounts.quotaControl.sessionLimit.maxSessionsPlaceholder')"
          />
          <p class="input-hint">{{ t('admin.accounts.quotaControl.sessionLimit.maxSessionsHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.quotaControl.sessionLimit.idleTimeout') }}</label>
          <div class="relative">
            <input
              v-model.number="state.sessionIdleTimeout"
              type="number"
              min="1"
              step="1"
              class="input pr-12"
              :placeholder="t('admin.accounts.quotaControl.sessionLimit.idleTimeoutPlaceholder')"
            />
            <span class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 dark:text-gray-400">{{ t('common.minutes') }}</span>
          </div>
          <p class="input-hint">{{ t('admin.accounts.quotaControl.sessionLimit.idleTimeoutHint') }}</p>
        </div>
      </div>
    </div>

    <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
      <div class="mb-3 flex items-center justify-between">
        <div>
          <label class="input-label mb-0">{{ t('admin.accounts.quotaControl.rpmLimit.label') }}</label>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.quotaControl.rpmLimit.hint') }}
          </p>
        </div>
        <button
          type="button"
          @click="state.rpmLimitEnabled = !state.rpmLimitEnabled"
          :class="[
            'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
            state.rpmLimitEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
          ]"
        >
          <span
            :class="[
              'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
              state.rpmLimitEnabled ? 'translate-x-5' : 'translate-x-0'
            ]"
          />
        </button>
      </div>

      <div v-if="state.rpmLimitEnabled" class="space-y-4">
        <div>
          <label class="input-label">{{ t('admin.accounts.quotaControl.rpmLimit.baseRpm') }}</label>
          <input
            v-model.number="state.baseRpm"
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
          <label class="input-label">{{ t('admin.accounts.quotaControl.rpmLimit.strategy') }}</label>
          <div class="flex gap-2">
            <button
              type="button"
              @click="state.rpmStrategy = 'tiered'"
              :class="[
                'flex-1 rounded-lg px-3 py-2 text-sm font-medium transition-all',
                state.rpmStrategy === 'tiered'
                  ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
              ]"
            >
              <div class="text-center">
                <div>{{ t('admin.accounts.quotaControl.rpmLimit.strategyTiered') }}</div>
                <div class="mt-0.5 text-[10px] opacity-70">{{ t('admin.accounts.quotaControl.rpmLimit.strategyTieredHint') }}</div>
              </div>
            </button>
            <button
              type="button"
              @click="state.rpmStrategy = 'sticky_exempt'"
              :class="[
                'flex-1 rounded-lg px-3 py-2 text-sm font-medium transition-all',
                state.rpmStrategy === 'sticky_exempt'
                  ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
              ]"
            >
              <div class="text-center">
                <div>{{ t('admin.accounts.quotaControl.rpmLimit.strategyStickyExempt') }}</div>
                <div class="mt-0.5 text-[10px] opacity-70">{{ t('admin.accounts.quotaControl.rpmLimit.strategyStickyExemptHint') }}</div>
              </div>
            </button>
          </div>
        </div>

        <div v-if="state.rpmStrategy === 'tiered'">
          <label class="input-label">{{ t('admin.accounts.quotaControl.rpmLimit.stickyBuffer') }}</label>
          <input
            v-model.number="state.rpmStickyBuffer"
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
            @click="state.userMsgQueueMode = opt.value"
            :class="[
              'rounded-md border px-3 py-1.5 text-sm transition-colors',
              state.userMsgQueueMode === opt.value
                ? 'border-primary-600 bg-primary-600 text-white'
                : 'border-gray-300 bg-white text-gray-700 hover:bg-gray-50 dark:border-dark-500 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'
            ]"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>
    </div>

    <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
      <div class="flex items-center justify-between">
        <div>
          <label class="input-label mb-0">{{ t('admin.accounts.quotaControl.tlsFingerprint.label') }}</label>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.quotaControl.tlsFingerprint.hint') }}
          </p>
        </div>
        <button
          type="button"
          @click="state.tlsFingerprintEnabled = !state.tlsFingerprintEnabled"
          :class="[
            'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
            state.tlsFingerprintEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
          ]"
        >
          <span
            :class="[
              'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
              state.tlsFingerprintEnabled ? 'translate-x-5' : 'translate-x-0'
            ]"
          />
        </button>
      </div>
      <div v-if="state.tlsFingerprintEnabled" class="mt-3 space-y-3">
        <div>
          <label class="input-label text-xs">{{ t('admin.accounts.quotaControl.tlsFingerprint.profileLabel') }}</label>
          <select
            v-model="tlsFingerprintProfileValue"
            class="mt-1 block w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm shadow-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-dark-500 dark:bg-dark-700 dark:text-white"
          >
            <option value="default">{{ t('admin.accounts.quotaControl.tlsFingerprint.profileDefault') }}</option>
            <option value="-1">{{ t('admin.accounts.quotaControl.tlsFingerprint.profileRandom') }}</option>
            <option
              v-if="missingTLSFingerprintProfileId !== null"
              :value="String(missingTLSFingerprintProfileId)"
            >
              {{ t('admin.accounts.quotaControl.tlsFingerprint.profileMissing', { id: missingTLSFingerprintProfileId }) }}
            </option>
            <option v-for="profile in tlsFingerprintProfiles" :key="profile.id" :value="String(profile.id)">
              {{ profile.name }}
            </option>
          </select>
          <p class="input-hint">
            {{
              tlsFingerprintProfilesLoading
                ? t('admin.accounts.quotaControl.tlsFingerprint.loadingProfiles')
                : t('admin.accounts.quotaControl.tlsFingerprint.profileHint')
            }}
          </p>
          <p
            v-if="selectedTLSFingerprintProfile?.description"
            class="mt-1 text-xs text-gray-500 dark:text-gray-400"
          >
            {{ selectedTLSFingerprintProfile.description }}
          </p>
          <p
            v-else-if="!tlsFingerprintProfilesLoading && tlsFingerprintProfiles.length === 0"
            class="mt-1 text-xs text-gray-500 dark:text-gray-400"
          >
            {{ t('admin.accounts.quotaControl.tlsFingerprint.profileEmpty') }}
          </p>
        </div>
      </div>
    </div>

    <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
      <div class="flex items-center justify-between">
        <div>
          <label class="input-label mb-0">{{ t('admin.accounts.quotaControl.sessionIdMasking.label') }}</label>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.quotaControl.sessionIdMasking.hint') }}
          </p>
        </div>
        <button
          type="button"
          @click="state.sessionIdMaskingEnabled = !state.sessionIdMaskingEnabled"
          :class="[
            'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
            state.sessionIdMaskingEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
          ]"
        >
          <span
            :class="[
              'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
              state.sessionIdMaskingEnabled ? 'translate-x-5' : 'translate-x-0'
            ]"
          />
        </button>
      </div>
    </div>

    <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
      <div class="flex items-center justify-between">
        <div>
          <label class="input-label mb-0">{{ t('admin.accounts.quotaControl.cacheTTLOverride.label') }}</label>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.quotaControl.cacheTTLOverride.hint') }}
          </p>
        </div>
        <button
          type="button"
          @click="state.cacheTTLOverrideEnabled = !state.cacheTTLOverrideEnabled"
          :class="[
            'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
            state.cacheTTLOverrideEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
          ]"
        >
          <span
            :class="[
              'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
              state.cacheTTLOverrideEnabled ? 'translate-x-5' : 'translate-x-0'
            ]"
          />
        </button>
      </div>
      <div v-if="state.cacheTTLOverrideEnabled" class="mt-3">
        <label class="input-label text-xs">{{ t('admin.accounts.quotaControl.cacheTTLOverride.target') }}</label>
        <select
          v-model="state.cacheTTLOverrideTarget"
          class="mt-1 block w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm shadow-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-dark-500 dark:bg-dark-700 dark:text-white"
        >
          <option value="5m">5m</option>
          <option value="1h">1h</option>
        </select>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.quotaControl.cacheTTLOverride.targetHint') }}
        </p>
      </div>
    </div>
  </div>
</template>
