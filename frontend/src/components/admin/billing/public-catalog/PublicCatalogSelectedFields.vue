<template>
  <div class="mt-3 grid gap-3 lg:grid-cols-2">
    <label class="space-y-1 text-xs text-slate-600 dark:text-slate-300">
      <span>{{ t('admin.billing.publicCatalog.card.publicModelId') }}</span>
      <input
        :value="publicModelLabel"
        class="input font-mono"
        :class="duplicate ? 'border-rose-300 focus:border-rose-500 focus:ring-rose-500' : ''"
        :data-testid="`public-id-${entryId}`"
        @input="emit('update-public-id', ($event.target as HTMLInputElement).value)"
      />
    </label>
    <label class="space-y-1 text-xs text-slate-600 dark:text-slate-300">
      <span>{{ t('admin.billing.publicCatalog.card.sourceAlias') }}</span>
      <input
        :value="sourceAliasLabel"
        class="input"
        :data-testid="`source-alias-${entryId}`"
        @input="emit('update-source-alias', ($event.target as HTMLInputElement).value)"
      />
    </label>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  entryId: string
  publicModelLabel: string
  sourceAliasLabel: string
  duplicate: boolean
}>()

const emit = defineEmits<{
  (e: 'update-public-id', value: string): void
  (e: 'update-source-alias', value: string): void
}>()

const { t } = useI18n()
</script>
