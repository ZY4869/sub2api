<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t('admin.settings.purchase.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.settings.purchase.description') }}
      </p>
    </div>
    <div class="space-y-6 p-6">
      <div class="flex items-center justify-between">
        <div>
          <label class="font-medium text-gray-900 dark:text-white">
            {{ t('admin.settings.purchase.enabled') }}
          </label>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.settings.purchase.enabledHint') }}
          </p>
        </div>
        <Toggle v-model="enabled" />
      </div>

      <div class="rounded-lg border border-gray-100 p-4 dark:border-dark-700">
        <div class="flex items-center justify-between gap-4">
          <div>
            <label class="font-medium text-gray-900 dark:text-white">
              {{ t('admin.settings.purchase.airwallexEnabled') }}
            </label>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.purchase.airwallexEnabledHint') }}
            </p>
            <p class="mt-1 text-xs" :class="effectiveEnabled ? 'text-emerald-600 dark:text-emerald-300' : 'text-amber-600 dark:text-amber-300'">
              {{ effectiveEnabled ? t('admin.settings.purchase.effectiveEnabled') : t('admin.settings.purchase.effectiveDisabled') }}
            </p>
          </div>
          <Toggle v-model="airwallexEnabled" />
        </div>

        <div class="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
          <label class="space-y-2">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.settings.purchase.airwallexEnv') }}
            </span>
            <Select v-model="airwallexEnv" :options="airwallexEnvOptions" />
          </label>
          <label class="space-y-2">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.settings.purchase.airwallexClientId') }}
            </span>
            <input v-model.trim="airwallexClientId" type="text" class="input font-mono text-sm" />
          </label>
          <label class="space-y-2">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.settings.purchase.airwallexApiKey') }}
            </span>
            <input
              v-model.trim="airwallexApiKey"
              type="password"
              class="input font-mono text-sm"
              :placeholder="apiKeyConfigured ? t('admin.settings.purchase.secretConfigured') : ''"
            />
          </label>
          <label class="space-y-2">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.settings.purchase.airwallexWebhookSecret') }}
            </span>
            <input
              v-model.trim="airwallexWebhookSecret"
              type="password"
              class="input font-mono text-sm"
              :placeholder="webhookSecretConfigured ? t('admin.settings.purchase.secretConfigured') : ''"
            />
          </label>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <label class="space-y-2">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.settings.purchase.allowedCurrencies') }}
          </span>
          <input v-model="allowedCurrenciesText" type="text" class="input font-mono text-sm" />
          <span class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.settings.purchase.allowedCurrenciesHint') }}
          </span>
        </label>
        <label class="space-y-2">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.settings.purchase.defaultCurrency') }}
          </span>
          <input v-model.trim="defaultCurrency" type="text" class="input font-mono text-sm uppercase" />
        </label>
        <label class="space-y-2">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.settings.purchase.minTopup') }}
          </span>
          <input v-model.number="minTopupAmount" type="number" min="0" step="0.01" class="input" />
        </label>
        <label class="space-y-2">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.settings.purchase.maxTopup') }}
          </span>
          <input v-model.number="maxTopupAmount" type="number" min="0" step="0.01" class="input" />
        </label>
      </div>

      <div>
        <div class="mb-3 flex items-center justify-between gap-3">
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.settings.purchase.subscriptionPlans') }}
          </label>
          <button type="button" class="btn btn-secondary btn-sm" @click="addPlan">
            <Icon name="plus" size="sm" class="mr-1.5" />
            {{ t('admin.settings.purchase.addPlan') }}
          </button>
        </div>
        <div class="space-y-3">
          <div
            v-for="(plan, index) in subscriptionPlans"
            :key="`${plan.plan_id || 'plan'}-${index}`"
            class="rounded-lg border border-gray-100 p-4 dark:border-dark-700"
          >
            <div class="mb-3 flex items-center justify-between gap-3">
              <span class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('admin.settings.purchase.planItem', { n: index + 1 }) }}
              </span>
              <button type="button" class="btn btn-secondary btn-xs" @click="removePlan(index)">
                <Icon name="trash" size="xs" />
              </button>
            </div>
            <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
              <input v-model.trim="plan.plan_id" class="input font-mono text-sm" :placeholder="t('admin.settings.purchase.planId')" />
              <input v-model.trim="plan.name" class="input" :placeholder="t('admin.settings.purchase.planName')" />
              <input v-model.number="plan.group_id" type="number" min="1" class="input" :placeholder="t('admin.settings.purchase.groupId')" />
              <input v-model.number="plan.validity_days" type="number" min="1" class="input" :placeholder="t('admin.settings.purchase.validityDays')" />
              <input
                :value="formatPrices(plan.prices_by_currency)"
                class="input font-mono text-sm md:col-span-2"
                :placeholder="t('admin.settings.purchase.pricesPlaceholder')"
                @input="updatePrices(plan, ($event.target as HTMLInputElement).value)"
              />
              <label class="flex items-center justify-between md:col-span-2">
                <span class="text-sm text-gray-600 dark:text-gray-300">
                  {{ t('admin.settings.purchase.planEnabled') }}
                </span>
                <Toggle v-model="plan.enabled" />
              </label>
            </div>
          </div>
        </div>
      </div>

      <div>
        <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.settings.purchase.url') }}
        </label>
        <input v-model="purchaseUrl" type="url" class="input font-mono text-sm" :placeholder="t('admin.settings.purchase.urlPlaceholder')" />
        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.purchase.urlHint') }}
        </p>
        <p class="mt-2 text-xs text-amber-600 dark:text-amber-400">
          {{ t('admin.settings.purchase.iframeWarning') }}
        </p>
      </div>

      <label class="space-y-2">
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.settings.purchase.antigravityUserAgentVersion') }}
        </span>
        <input
          v-model.trim="antigravityUserAgentVersion"
          type="text"
          class="input font-mono text-sm"
          :placeholder="t('admin.settings.purchase.antigravityUserAgentPlaceholder')"
        />
        <span class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.purchase.antigravityUserAgentHint') }}
        </span>
      </label>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import type { PaymentSubscriptionPlan } from '@/types'

const { t } = useI18n()

const enabled = defineModel<boolean>('enabled', { required: true })
const purchaseUrl = defineModel<string>('purchaseUrl', { required: true })
const airwallexEnabled = defineModel<boolean>('airwallexEnabled', { required: true })
const airwallexEnv = defineModel<string>('airwallexEnv', { required: true })
const airwallexClientId = defineModel<string>('airwallexClientId', { required: true })
const airwallexApiKey = defineModel<string>('airwallexApiKey', { required: true })
const airwallexWebhookSecret = defineModel<string>('airwallexWebhookSecret', { required: true })
const allowedCurrencies = defineModel<string[]>('allowedCurrencies', { required: true })
const defaultCurrency = defineModel<string>('defaultCurrency', { required: true })
const minTopupAmount = defineModel<number>('minTopupAmount', { required: true })
const maxTopupAmount = defineModel<number>('maxTopupAmount', { required: true })
const subscriptionPlans = defineModel<PaymentSubscriptionPlan[]>('subscriptionPlans', { required: true })
const antigravityUserAgentVersion = defineModel<string>('antigravityUserAgentVersion', { required: true })

defineProps<{
  apiKeyConfigured: boolean
  webhookSecretConfigured: boolean
  effectiveEnabled?: boolean
}>()

const airwallexEnvOptions = computed(() => [
  { value: 'demo', label: t('admin.settings.purchase.airwallexEnvDemo') },
  { value: 'prod', label: t('admin.settings.purchase.airwallexEnvProd') }
])

const allowedCurrenciesText = computed({
  get: () => allowedCurrencies.value.join(', '),
  set: (value: string) => {
    allowedCurrencies.value = value
      .split(',')
      .map((item) => item.trim().toUpperCase())
      .filter(Boolean)
  }
})

function addPlan() {
  subscriptionPlans.value.push({
    plan_id: '',
    name: '',
    group_id: 0,
    validity_days: 30,
    prices_by_currency: {},
    enabled: true
  })
}

function removePlan(index: number) {
  subscriptionPlans.value.splice(index, 1)
}

function formatPrices(prices: Record<string, number>): string {
  return Object.entries(prices || {})
    .map(([currency, price]) => `${currency}:${price}`)
    .join(', ')
}

function updatePrices(plan: PaymentSubscriptionPlan, value: string) {
  const next: Record<string, number> = {}
  for (const part of value.split(',')) {
    const [currency, amount] = part.split(':').map((item) => item.trim())
    const price = Number(amount)
    if (currency && Number.isFinite(price) && price > 0) {
      next[currency.toUpperCase()] = price
    }
  }
  plan.prices_by_currency = next
}
</script>
