<template>
  <div>
    <div class="mb-1.5 flex flex-wrap items-center gap-2">
      <div class="truncate text-sm font-bold text-slate-800 dark:text-slate-100">
        {{ item.display_name || baseModel }}
      </div>
      <span
        v-if="lifecycleLabel"
        class="inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 text-[10px] font-medium leading-none shadow-sm"
        :class="lifecycleClass"
      >
        <Icon :name="lifecycleIcon" size="xs" />
        {{ lifecycleLabel }}
      </span>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <button
        type="button"
        class="group/copy inline-flex min-w-0 items-center gap-1 text-left"
        :aria-label="t('admin.billing.publicCatalog.card.copyPublicId')"
        @click="copyPublicID"
      >
        <span class="truncate font-mono text-xs text-slate-400 transition group-hover/copy:text-slate-600 dark:text-slate-500 dark:group-hover/copy:text-slate-300">
          {{ publicModelLabel }}
        </span>
        <Icon :name="copied ? 'check' : 'copy'" size="xs" class="shrink-0 text-slate-300 group-hover/copy:text-emerald-500" />
      </button>

      <span class="inline-flex max-w-[180px] items-center gap-1 rounded-md border border-slate-200 bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium leading-none text-slate-500 dark:border-dark-700 dark:bg-dark-700 dark:text-slate-300">
        <Icon name="link" size="xs" />
        <span class="truncate">{{ sourceAliasLabel }}</span>
      </span>

      <span
        class="inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 text-[10px] font-mono font-medium leading-none"
        :class="healthClass"
      >
        <span class="h-1.5 w-1.5 rounded-full" :class="healthDotClass" />
        {{ healthLabel }}
      </span>
    </div>

    <div class="mt-2 flex flex-wrap items-center gap-1.5">
      <span
        v-for="badge in badges"
        :key="badge.key"
        class="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-white px-1.5 py-0.5 text-[10px] font-medium leading-none text-slate-600 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-slate-300"
        :title="badge.title"
      >
        <Icon :name="badge.icon" size="xs" :class="badge.iconClass" />
        {{ badge.label }}
      </span>
    </div>

    <PublicCatalogCapabilityBadges :item="item" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BillingPublicCatalogAdminEntry } from '@/api/admin/billing'
import Icon from '@/components/icons/Icon.vue'
import PublicCatalogCapabilityBadges from './PublicCatalogCapabilityBadges.vue'

const props = withDefaults(defineProps<{
  item: BillingPublicCatalogAdminEntry
  missing?: boolean
}>(), {
  missing: false,
})

const { t } = useI18n()
const copied = ref(false)

const baseModel = computed(() => props.item.base_model || props.item.source_model_id || props.item.model)
const publicModelLabel = computed(() => props.item.public_model_id || props.item.model)
const sourceAliasLabel = computed(() => props.item.source_alias || t('admin.billing.publicCatalog.card.defaultSource'))
const protocolLabel = computed(() => props.item.source_protocol || props.item.request_protocols?.[0] || '-')
const providerLabel = computed(() => formatProviderName(props.item.provider || protocolLabel.value))

const lifecycleLabel = computed(() => {
  if (props.item.lifecycle_status === 'deprecated') return t('admin.billing.publicCatalog.card.lifecycle.deprecated')
  if (props.item.lifecycle_status === 'beta') return t('admin.billing.publicCatalog.card.lifecycle.beta')
  if (props.item.availability_state === 'verified') return t('admin.billing.publicCatalog.card.lifecycle.stable')
  return ''
})

const lifecycleIcon = computed(() => {
  if (props.item.lifecycle_status === 'deprecated') return 'exclamationTriangle'
  if (props.item.lifecycle_status === 'beta') return 'sparkles'
  return 'shield'
})

const lifecycleClass = computed(() => {
  if (props.item.lifecycle_status === 'deprecated') {
    return 'border-rose-200 bg-rose-50 text-rose-600 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-200'
  }
  if (props.item.lifecycle_status === 'beta') {
    return 'border-indigo-200 bg-indigo-50 text-indigo-600 dark:border-indigo-500/30 dark:bg-indigo-500/10 dark:text-indigo-200'
  }
  return 'border-emerald-200 bg-emerald-50 text-emerald-600 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200'
})

const healthLabel = computed(() => {
  if (props.missing) return t('admin.billing.publicCatalog.card.statuses.expired')
  if (props.item.availability_state === 'unavailable' || props.item.status === 'error') {
    return t('admin.billing.publicCatalog.card.statuses.unavailable')
  }
  if (props.item.status === 'warning' || props.item.stale_state === 'stale') {
    return t('admin.billing.publicCatalog.card.statuses.pending')
  }
  return t('admin.billing.publicCatalog.card.statuses.available')
})

const healthClass = computed(() => {
  if (props.missing || props.item.availability_state === 'unavailable' || props.item.status === 'error') {
    return 'border-rose-100 bg-rose-50 text-rose-600 dark:border-rose-500/20 dark:bg-rose-500/10 dark:text-rose-200'
  }
  if (props.item.status === 'warning' || props.item.stale_state === 'stale') {
    return 'border-amber-100 bg-amber-50 text-amber-600 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-200'
  }
  return 'border-emerald-100 bg-emerald-50 text-emerald-600 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-200'
})

const healthDotClass = computed(() => {
  if (props.missing || props.item.availability_state === 'unavailable' || props.item.status === 'error') return 'bg-rose-500'
  if (props.item.status === 'warning' || props.item.stale_state === 'stale') return 'bg-amber-500'
  return 'bg-emerald-500'
})

const badges = computed(() => [
  {
    key: 'provider',
    label: providerLabel.value,
    title: t('admin.billing.publicCatalog.card.provider'),
    icon: 'badge' as const,
    iconClass: 'text-slate-400',
  },
  {
    key: 'protocol',
    label: protocolLabel.value,
    title: t('admin.billing.publicCatalog.card.protocolPlain'),
    icon: 'terminal' as const,
    iconClass: 'text-blue-500',
  },
  {
    key: 'mode',
    label: props.item.mode || '-',
    title: t('admin.billing.publicCatalog.card.modePlain'),
    icon: 'chat' as const,
    iconClass: 'text-amber-500',
  },
  {
    key: 'base',
    label: baseModel.value,
    title: t('admin.billing.publicCatalog.card.baseModelPlain'),
    icon: 'database' as const,
    iconClass: 'text-slate-400',
  },
].filter((badge) => badge.label && badge.label !== '-'))

async function copyPublicID() {
  try {
    await navigator.clipboard?.writeText(publicModelLabel.value)
    copied.value = true
    window.setTimeout(() => {
      copied.value = false
    }, 1400)
  } catch {
    copied.value = false
  }
}

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
