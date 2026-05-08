<template>
  <div v-if="enabled" class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/50">
    <label class="flex items-start gap-3">
      <input
        :checked="accepted"
        type="checkbox"
        class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600"
        :aria-label="t('auth.agreementAcceptAria')"
        @change="handleAcceptedChange"
      />
      <span class="text-sm leading-6 text-gray-700 dark:text-dark-300">
        {{ t('auth.agreementPrefix') }}
        <template v-for="(doc, index) in documents" :key="doc.page_slug">
          <RouterLink
            :to="`/legal/${doc.page_slug}`"
            target="_blank"
            class="font-medium text-primary-600 hover:text-primary-500 dark:text-primary-400"
          >
            {{ doc.title }}
          </RouterLink>
          <span v-if="index < documents.length - 1">{{ t('auth.agreementSeparator') }}</span>
        </template>
      </span>
    </label>
    <p v-if="error" class="input-error-text mt-2">
      {{ error }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import type { LoginAgreementDocument } from '@/types'

defineProps<{
  enabled: boolean
  accepted: boolean
  documents: LoginAgreementDocument[]
  error?: string
}>()

const emit = defineEmits<{
  'update:accepted': [value: boolean]
}>()

const { t } = useI18n()

function handleAcceptedChange(event: Event) {
  const target = event.target as HTMLInputElement | null
  emit('update:accepted', target?.checked === true)
}
</script>
