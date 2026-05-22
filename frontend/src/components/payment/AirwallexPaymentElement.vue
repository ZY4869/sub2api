<template>
  <div class="space-y-3">
    <div
      ref="mountRef"
      class="min-h-28 rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900"
      :aria-label="t('purchase.airwallexElement')"
    />
    <div v-if="loading" class="text-sm text-gray-500 dark:text-dark-400">
      {{ t('purchase.airwallexLoading') }}
    </div>
    <div v-else-if="error" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200">
      {{ t('purchase.airwallexLoadFailed') }}
    </div>
    <button
      v-else-if="mounted && order"
      type="button"
      class="btn btn-primary w-full"
      :disabled="confirming"
      @click="confirmPayment"
    >
      {{ confirming ? t('purchase.confirmingPayment') : t('purchase.confirmPayment') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAirwallexElements } from '@/composables/useAirwallexElements'
import type { PaymentCreateOrderResponse, PaymentResumeOrderResponse } from '@/types'

const props = defineProps<{
  order: (PaymentCreateOrderResponse | PaymentResumeOrderResponse) | null
}>()

const emit = defineEmits<{
  confirmed: []
}>()

const { t, locale } = useI18n()
const mountRef = ref<HTMLElement | null>(null)
const { mounted, loading, confirming, error, mount, confirm, destroy } = useAirwallexElements()

watch(
  () => props.order,
  async (order) => {
    destroy()
    if (!order?.client_secret) return
    await nextTick()
    if (mountRef.value) {
      await mount(mountRef.value, order, String(locale.value))
    }
  },
  { immediate: true }
)

async function confirmPayment() {
  if (!props.order) return
  const result = await confirm(props.order)
  if (result) {
    emit('confirmed')
  }
}

onBeforeUnmount(destroy)
</script>
