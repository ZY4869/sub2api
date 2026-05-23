<template>
  <div class="flex items-center justify-end gap-1.5" data-testid="accounts-view-airy-row-actions">
    <button
      type="button"
      :class="toggleButtonClass"
      :disabled="togglingSchedulable === account.id"
      :title="toggleTitle"
      :aria-label="toggleTitle"
      @click="emit('toggle-schedulable')"
    >
      <span
        class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
        :class="account.schedulable ? 'translate-x-4' : 'translate-x-0'"
      />
    </button>

    <button
      type="button"
      :class="actionButtonClass"
      :title="t('common.copy')"
      :aria-label="t('common.copy')"
      @click="copySummary"
    >
      <Icon :name="copied ? 'check' : 'copy'" size="sm" :stroke-width="2.3" />
    </button>

    <button
      type="button"
      :class="actionButtonClass"
      :title="t('common.edit')"
      :aria-label="t('common.edit')"
      @click="emit('edit')"
    >
      <Icon name="edit" size="sm" :stroke-width="2.2" />
    </button>

    <button
      type="button"
      :class="dangerButtonClass"
      :title="t('common.delete')"
      :aria-label="t('common.delete')"
      @click="emit('delete')"
    >
      <Icon name="trash" size="sm" :stroke-width="2.2" />
    </button>

    <button
      type="button"
      :class="actionButtonClass"
      :title="t('common.more')"
      :aria-label="t('common.more')"
      @click="emit('more', $event)"
    >
      <Icon name="more" size="sm" :stroke-width="2.2" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Account } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'

const props = defineProps<{
  account: Account
  togglingSchedulable: number | null
}>()

const emit = defineEmits<{
  'toggle-schedulable': []
  edit: []
  delete: []
  more: [event: MouseEvent]
}>()

const { t } = useI18n()
const { copied, copyToClipboard } = useClipboard()

const toggleTitle = computed(() =>
  props.account.schedulable
    ? t('admin.accounts.schedulableEnabled')
    : t('admin.accounts.schedulableDisabled')
)

const toggleButtonClass = computed(() => [
  'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:focus:ring-offset-dark-800',
  props.account.schedulable
    ? 'bg-emerald-500 hover:bg-emerald-600'
    : 'bg-slate-200 hover:bg-slate-300 dark:bg-slate-700 dark:hover:bg-slate-600'
])

const actionButtonClass = 'inline-flex h-7 w-7 items-center justify-center rounded-full border border-slate-200/80 bg-white text-slate-500 shadow-sm transition hover:border-primary-200 hover:bg-primary-50 hover:text-primary-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-400/60 dark:border-slate-700/80 dark:bg-slate-900/80 dark:text-slate-300 dark:hover:border-primary-400/30 dark:hover:bg-primary-500/10 dark:hover:text-primary-200'
const dangerButtonClass = 'inline-flex h-7 w-7 items-center justify-center rounded-full border border-rose-200/80 bg-white text-rose-500 shadow-sm transition hover:border-rose-300 hover:bg-rose-50 hover:text-rose-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-rose-400/60 dark:border-rose-400/20 dark:bg-slate-900/80 dark:text-rose-200 dark:hover:bg-rose-500/10 dark:hover:text-rose-100'

const sanitizedSummary = computed(() => ({
  id: props.account.id,
  name: props.account.name,
  platform: props.account.platform,
  gateway_protocol: props.account.gateway_protocol ?? null,
  type: props.account.type,
  status: props.account.status,
  schedulable: props.account.schedulable,
  concurrency: props.account.concurrency,
  priority: props.account.priority,
  rate_multiplier: props.account.rate_multiplier ?? 1,
  group_names: (props.account.groups || []).map((group) => group.name)
}))

const copySummary = async () => {
  await copyToClipboard(
    JSON.stringify(sanitizedSummary.value, null, 2),
    t('common.copied'),
  )
}
</script>
