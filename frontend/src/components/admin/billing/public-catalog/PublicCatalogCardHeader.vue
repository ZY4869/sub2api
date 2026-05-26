<template>
  <div class="flex items-start justify-between gap-3">
    <div class="flex min-w-0 items-start gap-3">
      <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-slate-100 bg-slate-50 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <ModelIcon
          :model="baseModel"
          :provider="item.provider"
          :icon-key="item.provider_icon_key"
          :display-name="item.display_name"
          size="24px"
        />
      </div>
      <PublicCatalogEntryMeta :item="item" :missing="missing" class="min-w-0" />
    </div>

    <div class="flex shrink-0 items-start gap-1">
      <template v-if="mode === 'selected'">
        <button
          type="button"
          class="rounded-md p-1.5 text-slate-300 transition hover:bg-emerald-50 hover:text-emerald-600 dark:hover:bg-emerald-500/10 dark:hover:text-emerald-200"
          :aria-label="t('admin.billing.publicCatalog.card.edit')"
          :data-testid="`edit-entry-${entryId}`"
          @click="emit('edit')"
        >
          <Icon name="edit" size="sm" />
        </button>
        <button
          type="button"
          class="rounded-md p-1.5 text-slate-300 transition hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-500/10 dark:hover:text-rose-200"
          :aria-label="t('admin.billing.publicCatalog.card.remove')"
          :data-testid="`remove-entry-${entryId}`"
          @click="emit('remove')"
        >
          <Icon name="trash" size="sm" />
        </button>
      </template>
      <span
        v-else
        class="inline-flex items-center gap-1 rounded-md bg-slate-100 px-2 py-1 text-[10px] font-medium tracking-wide text-slate-500 dark:bg-dark-700 dark:text-slate-300"
      >
        <ModelPlatformIcon :platform="item.provider || item.source_protocol" size="xs" />
        {{ providerLabel }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BillingPublicCatalogAdminEntry } from '@/api/admin/billing'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import PublicCatalogEntryMeta from './PublicCatalogEntryMeta.vue'

const props = defineProps<{
  item: BillingPublicCatalogAdminEntry
  mode: 'available' | 'selected'
  entryId: string
  baseModel: string
  missing: boolean
}>()

const emit = defineEmits<{
  (e: 'edit'): void
  (e: 'remove'): void
}>()

const { t } = useI18n()
const providerLabel = computed(() => formatProviderName(props.item.provider || props.item.source_protocol || '-'))

function formatProviderName(value: string): string {
  const normalized = value.trim().toLowerCase()
  const labels: Record<string, string> = {
    openai: 'OpenAI',
    anthropic: 'Anthropic',
    gemini: 'Gemini',
    google: 'Google',
    deepseek: 'DeepSeek',
  }
  return labels[normalized] || value
}
</script>
