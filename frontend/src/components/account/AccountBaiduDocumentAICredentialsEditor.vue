<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL } from '@/utils/baiduDocumentAI'

const props = withDefaults(defineProps<{
  mode: 'create' | 'edit'
}>(), {
  mode: 'create'
})

const asyncBearerToken = defineModel<string>('asyncBearerToken', { required: true })
const asyncBaseUrl = defineModel<string>('asyncBaseUrl', { required: true })
const directToken = defineModel<string>('directToken', { required: true })
const directApiUrlsText = defineModel<string>('directApiUrlsText', { required: true })

const { t } = useI18n()

const tokenPlaceholderKey = computed(() =>
  props.mode === 'edit'
    ? 'admin.accounts.leaveEmptyToKeep'
    : 'admin.accounts.apiKeyIsRequired'
)
</script>

<template>
  <div
    data-testid="baidu-document-ai-credentials-editor"
    class="space-y-4 rounded-xl border border-sky-200 bg-sky-50/70 p-4 dark:border-sky-900/40 dark:bg-sky-950/20"
  >
    <div>
      <div class="text-sm font-semibold text-sky-900 dark:text-sky-100">
        {{ t('admin.accounts.baiduDocumentAI.title') }}
      </div>
      <p class="mt-1 text-xs leading-5 text-sky-700 dark:text-sky-300">
        {{ t('admin.accounts.baiduDocumentAI.description') }}
      </p>
    </div>

    <div>
      <label class="input-label">{{ t('admin.accounts.baiduDocumentAI.asyncBaseUrl') }}</label>
      <input
        v-model="asyncBaseUrl"
        type="text"
        class="input"
        :placeholder="BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL"
        data-testid="baidu-document-ai-async-base-url"
        spellcheck="false"
      />
      <p class="input-hint">{{ t('admin.accounts.baiduDocumentAI.asyncBaseUrlHint') }}</p>
    </div>

    <div>
      <label class="input-label">{{ t('admin.accounts.baiduDocumentAI.asyncBearerToken') }}</label>
      <input
        v-model="asyncBearerToken"
        type="password"
        class="input font-mono"
        :placeholder="t(tokenPlaceholderKey)"
        autocomplete="off"
        data-testid="baidu-document-ai-async-bearer-token"
        spellcheck="false"
      />
      <p class="input-hint">{{ t('admin.accounts.baiduDocumentAI.asyncBearerTokenHint') }}</p>
    </div>

    <div>
      <label class="input-label">{{ t('admin.accounts.baiduDocumentAI.directToken') }}</label>
      <input
        v-model="directToken"
        type="password"
        class="input font-mono"
        :placeholder="t(tokenPlaceholderKey)"
        autocomplete="off"
        data-testid="baidu-document-ai-direct-token"
        spellcheck="false"
      />
      <p class="input-hint">{{ t('admin.accounts.baiduDocumentAI.directTokenHint') }}</p>
    </div>

    <div>
      <label class="input-label">{{ t('admin.accounts.baiduDocumentAI.directApiUrls') }}</label>
      <textarea
        v-model="directApiUrlsText"
        rows="6"
        class="input font-mono"
        :placeholder="t('admin.accounts.baiduDocumentAI.directApiUrlsPlaceholder')"
        data-testid="baidu-document-ai-direct-api-urls"
        spellcheck="false"
      />
      <p class="input-hint">{{ t('admin.accounts.baiduDocumentAI.directApiUrlsHint') }}</p>
    </div>
  </div>
</template>
