<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import type { SelectOption } from '@/components/common/Select.vue'
import type { AccountPlatform, AccountTier } from '@/types'
import {
  accountTierI18nKey,
  isAccountTierPlatform,
  resolveAccountTierCapacity,
  resolveAccountTierOptions,
} from '@/utils/accountTier'

const props = defineProps<{
  platform: AccountPlatform | string
  disabled?: boolean
  showApplyCapacity?: boolean
}>()

const tier = defineModel<AccountTier | ''>('tier', { required: true })
const emit = defineEmits<{
  'apply-capacity': [capacity: number]
}>()

const { t } = useI18n()

const options = computed<SelectOption[]>(() =>
  resolveAccountTierOptions(props.platform).map((value) => ({
    value,
    label: t(accountTierI18nKey(value), {
      capacity: resolveAccountTierCapacity(props.platform, value),
    }),
  }))
)

const selectedCapacity = computed(() =>
  resolveAccountTierCapacity(props.platform, tier.value)
)

const visible = computed(() => isAccountTierPlatform(props.platform))
</script>

<template>
  <div
    v-if="visible"
    class="space-y-2 rounded-xl border border-gray-200 bg-gray-50/70 p-4 dark:border-dark-600 dark:bg-dark-800/50"
  >
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <label class="input-label mb-0">{{ t('admin.accounts.accountTier.label') }}</label>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.accountTier.hint') }}
        </p>
      </div>
      <button
        v-if="showApplyCapacity !== false"
        type="button"
        class="btn btn-secondary btn-sm"
        :disabled="disabled || selectedCapacity <= 0"
        @click="emit('apply-capacity', selectedCapacity)"
      >
        {{ t('admin.accounts.accountTier.applyCapacity') }}
      </button>
    </div>
    <Select
      v-model="tier"
      :options="options"
      :disabled="disabled"
    />
    <p v-if="selectedCapacity > 0" class="text-xs text-gray-500 dark:text-gray-400">
      {{ t('admin.accounts.accountTier.capacityHint', { capacity: selectedCapacity }) }}
    </p>
  </div>
</template>
