<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ACCOUNT_UPSTREAM_API_KEY_PLACEHOLDER,
  ACCOUNT_UPSTREAM_BASE_URL_PLACEHOLDER,
  resolveAccountUpstreamApiKeyHintKey,
  type AccountUpstreamSettingsMode
} from '@/utils/accountUpstreamSettings'

const props = defineProps<{
  mode: AccountUpstreamSettingsMode
}>()

const baseUrl = defineModel<string>('baseUrl', { required: true })
const apiKey = defineModel<string>('apiKey', { required: true })

const { t } = useI18n()

const isCreateMode = computed(() => props.mode === 'create')
const apiKeyHint = computed(() => t(resolveAccountUpstreamApiKeyHintKey(props.mode)))
</script>

<template>
  <div class="space-y-4">
    <div>
      <label class="input-label">{{ t('admin.accounts.upstream.baseUrl') }}</label>
      <input
        v-model="baseUrl"
        type="text"
        class="input"
        :required="isCreateMode"
        :placeholder="ACCOUNT_UPSTREAM_BASE_URL_PLACEHOLDER"
      />
      <p class="input-hint">{{ t('admin.accounts.upstream.baseUrlHint') }}</p>
    </div>

    <div>
      <label class="input-label">{{ t('admin.accounts.upstream.apiKey') }}</label>
      <input
        v-model="apiKey"
        type="password"
        class="input font-mono"
        :required="isCreateMode"
        :placeholder="ACCOUNT_UPSTREAM_API_KEY_PLACEHOLDER"
      />
      <p class="input-hint">{{ apiKeyHint }}</p>
    </div>
  </div>
</template>
