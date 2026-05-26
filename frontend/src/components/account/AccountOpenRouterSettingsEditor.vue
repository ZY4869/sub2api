<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const httpReferer = defineModel<string>('httpReferer', { required: true })
const openrouterTitle = defineModel<string>('openrouterTitle', { required: true })

const { t } = useI18n()
</script>

<template>
  <div class="space-y-4 rounded-lg border border-cyan-200 bg-cyan-50/60 p-4 dark:border-cyan-900/50 dark:bg-cyan-950/20">
    <div>
      <label class="input-label">{{ t('admin.accounts.openrouter.httpReferer') }}</label>
      <input
        v-model="httpReferer"
        type="url"
        class="input"
        placeholder="https://example.com"
        data-testid="openrouter-http-referer-input"
      />
      <p class="input-hint">{{ t('admin.accounts.openrouter.httpRefererHint') }}</p>
    </div>

    <div>
      <label class="input-label">{{ t('admin.accounts.openrouter.title') }}</label>
      <input
        v-model="openrouterTitle"
        type="text"
        class="input"
        :placeholder="t('admin.accounts.openrouter.titlePlaceholder')"
        data-testid="openrouter-title-input"
      />
      <p class="input-hint">{{ t('admin.accounts.openrouter.titleHint') }}</p>
    </div>
  </div>
</template>
