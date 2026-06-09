<template>
  <div class="grid grid-cols-2 gap-4">
    <div>
      <label class="input-label">{{ t('admin.proxies.expiresAt') }}</label>
      <input
        :value="form.expires_at"
        type="datetime-local"
        class="input"
        @input="(event) => updateField('expires_at', (event.target as HTMLInputElement).value)"
      />
      <p class="input-hint mt-1">{{ t('admin.proxies.expiresAtHint') }}</p>
    </div>
    <div>
      <label class="input-label">{{ t('admin.proxies.expiryRemindDays') }}</label>
      <input
        :value="form.expiry_remind_days"
        type="number"
        min="0"
        max="3650"
        class="input"
        @input="(event) => updateField('expiry_remind_days', Number((event.target as HTMLInputElement).value))"
      />
      <p class="input-hint mt-1">{{ t('admin.proxies.expiryRemindDaysHint') }}</p>
    </div>
  </div>
  <div>
    <label class="input-label">{{ t('admin.proxies.fallbackProxy') }}</label>
    <Select
      :model-value="form.fallback_proxy_id"
      :options="fallbackProxyOptions"
      searchable
      @update:model-value="(value) => updateField('fallback_proxy_id', normalizeFallbackValue(value))"
    />
    <p class="input-hint mt-1">{{ t('admin.proxies.fallbackProxyHint') }}</p>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import type { ProxyLifecycleFormFields } from './utils'

const props = defineProps<{
  form: ProxyLifecycleFormFields
  fallbackProxyOptions: Array<{ value: number | null; label: string }>
}>()

const emit = defineEmits<{
  update: [value: ProxyLifecycleFormFields]
}>()

const { t } = useI18n()

const updateField = (key: keyof ProxyLifecycleFormFields, value: string | number | null) => {
  emit('update', { ...props.form, [key]: value })
}

const normalizeFallbackValue = (value: string | number | boolean | null): number | null => {
  if (typeof value === 'number' && value > 0) return value
  if (typeof value === 'string' && value.trim()) {
    const parsed = Number(value)
    return Number.isFinite(parsed) && parsed > 0 ? parsed : null
  }
  return null
}
</script>
