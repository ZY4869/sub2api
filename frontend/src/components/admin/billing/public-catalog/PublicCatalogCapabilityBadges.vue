<template>
  <div class="mt-2 flex flex-wrap items-center gap-1.5">
    <span
      v-if="demoLabel"
      class="inline-flex items-center gap-1 rounded-md border border-rose-200 bg-rose-50 px-1.5 py-0.5 text-[10px] font-medium leading-none text-rose-700 shadow-sm dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-200"
    >
      <Icon name="infoCircle" size="xs" />
      {{ demoLabel }}
    </span>
    <span
      v-if="contextLabel"
      class="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-white px-1.5 py-0.5 text-[10px] font-medium leading-none text-slate-600 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-slate-300"
      :title="contextTitle"
    >
      <Icon name="book" size="xs" class="text-emerald-500" />
      {{ contextLabel }}
    </span>
    <span
      v-if="endpointLabel"
      class="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-white px-1.5 py-0.5 text-[10px] font-medium leading-none text-slate-600 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-slate-300"
      :title="endpointTitle"
    >
      <Icon name="terminal" size="xs" class="text-blue-500" />
      {{ endpointLabel }}
    </span>
    <span
      v-if="capabilityLabel"
      class="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-white px-1.5 py-0.5 text-[10px] font-medium leading-none text-slate-600 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-slate-300"
      :title="capabilityTitle"
    >
      <Icon name="beaker" size="xs" class="text-amber-500" />
      {{ capabilityLabel }}
    </span>
    <span
      v-if="lifecycleSourceLabel"
      class="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-white px-1.5 py-0.5 text-[10px] font-medium leading-none text-slate-600 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-slate-300"
    >
      <Icon name="shield" size="xs" class="text-slate-400" />
      {{ lifecycleSourceLabel }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BillingPublicCatalogAdminEntry } from '@/api/admin/billing'
import Icon from '@/components/icons/Icon.vue'
import {
  formatContextWindow,
  lifecycleSourceLabel as formatLifecycleSourceLabel,
  sourceLabel,
  supportLabel,
} from '@/components/models/public-catalog/publicModelCatalogView'

const props = defineProps<{
  item: BillingPublicCatalogAdminEntry
}>()

const { t } = useI18n()

const demoLabel = computed(() => (props.item.is_demo ? t('admin.billing.publicCatalog.card.demo') : ''))
const contextSourceLabel = computed(() => sourceLabel(t, props.item.context_window?.source, props.item.context_window?.verified))
const contextLabel = computed(() => {
  const label = formatContextWindow(props.item.context_window?.tokens || props.item.context_window_tokens)
  return label === '-' ? '' : `${label} · ${contextSourceLabel.value}`
})
const contextTitle = computed(() => t('admin.billing.publicCatalog.card.contextSource'))

const visibleEndpoint = computed(() =>
  (props.item.protocol_endpoints || []).find((endpoint) => endpoint.support === 'supported' || endpoint.support === 'partial'),
)
const endpointLabel = computed(() => {
  const endpoint = visibleEndpoint.value
  if (!endpoint) return ''
  return `${endpoint.key || endpoint.protocol} · ${sourceLabel(t, endpoint.source, endpoint.verified)}`
})
const endpointTitle = computed(() => {
  const endpoint = visibleEndpoint.value
  if (!endpoint) return ''
  return `${supportLabel(t, endpoint.support)} · ${endpoint.endpoint || endpoint.protocol || endpoint.key}`
})

const visibleCapability = computed(() =>
  (props.item.capability_matrix || []).find((entry) => entry.support === 'supported' || entry.support === 'partial'),
)
const capabilityLabel = computed(() => {
  const capability = visibleCapability.value
  if (!capability) return ''
  return `${capability.capability} · ${sourceLabel(t, capability.source, capability.verified)}`
})
const capabilityTitle = computed(() => {
  const capability = visibleCapability.value
  if (!capability) return ''
  return `${supportLabel(t, capability.support)} · ${capability.endpoint || capability.protocol || capability.capability}`
})

const lifecycleSourceLabel = computed(() => {
  if (!props.item.lifecycle?.source && !props.item.lifecycle?.confidence) return ''
  return formatLifecycleSourceLabel(t, props.item.lifecycle)
})
</script>
