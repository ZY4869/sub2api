<template>
  <section class="space-y-4 rounded-xl border border-sky-200 bg-sky-50/70 p-4 dark:border-sky-900/40 dark:bg-sky-950/20">
    <div class="space-y-1">
      <h3 class="text-sm font-semibold text-sky-950 dark:text-sky-100">
        {{ t('admin.accounts.gemini.vertex.formTitle') }}
      </h3>
      <p class="text-xs leading-5 text-sky-800 dark:text-sky-300">
        {{ t('admin.accounts.gemini.vertex.formDescription') }}
      </p>
    </div>

    <div class="grid gap-4 md:grid-cols-2">
      <label class="space-y-1">
        <span class="input-label">{{ t('admin.accounts.gemini.vertex.projectId') }}</span>
        <input
          v-model="projectId"
          type="text"
          class="input"
          :placeholder="t('admin.accounts.gemini.vertex.projectIdPlaceholder')"
        />
        <p class="input-hint">{{ t('admin.accounts.gemini.vertex.projectIdHint') }}</p>
      </label>

      <label class="space-y-1">
        <span class="input-label">{{ t('admin.accounts.gemini.vertex.location') }}</span>
        <input
          v-model="location"
          type="text"
          class="input"
          :placeholder="t('admin.accounts.gemini.vertex.locationPlaceholder')"
        />
        <p class="input-hint">{{ t('admin.accounts.gemini.vertex.locationHint') }}</p>
      </label>
    </div>

    <label class="space-y-1">
      <span class="input-label">{{ t('admin.accounts.gemini.vertex.accessToken') }}</span>
      <textarea
        v-model="accessToken"
        rows="4"
        class="input w-full resize-y font-mono text-sm"
        :placeholder="
          mode === 'edit'
            ? t('admin.accounts.gemini.vertex.accessTokenPlaceholderEdit')
            : t('admin.accounts.gemini.vertex.accessTokenPlaceholder')
        "
      ></textarea>
      <p class="input-hint">
        {{
          mode === 'edit'
            ? t('admin.accounts.gemini.vertex.accessTokenHintEdit')
            : t('admin.accounts.gemini.vertex.accessTokenHint')
        }}
      </p>
    </label>

    <div class="grid gap-4 md:grid-cols-2">
      <label class="space-y-1">
        <span class="input-label">{{ t('admin.accounts.gemini.vertex.expiresAt') }}</span>
        <input
          v-model="expiresAtInput"
          type="datetime-local"
          class="input"
        />
        <p class="input-hint">{{ t('admin.accounts.gemini.vertex.expiresAtHint') }}</p>
      </label>

      <label class="space-y-1">
        <span class="input-label">{{ t('admin.accounts.gemini.vertex.baseUrl') }}</span>
        <input
          v-model="baseUrl"
          type="text"
          class="input"
          :placeholder="defaultBaseUrl"
        />
        <p class="input-hint">{{ t('admin.accounts.gemini.vertex.baseUrlHint') }}</p>
      </label>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { GEMINI_VERTEX_DEFAULT_BASE_URL } from '@/utils/geminiAccount'

interface Props {
  mode?: 'create' | 'edit'
}

withDefaults(defineProps<Props>(), {
  mode: 'create'
})

const projectId = defineModel<string>('projectId', { required: true })
const location = defineModel<string>('location', { required: true })
const accessToken = defineModel<string>('accessToken', { required: true })
const expiresAtInput = defineModel<string>('expiresAtInput', { required: true })
const baseUrl = defineModel<string>('baseUrl', { required: true })

const { t } = useI18n()
const defaultBaseUrl = GEMINI_VERTEX_DEFAULT_BASE_URL
</script>
