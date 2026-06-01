<script setup lang="ts">
import AccountConcurrencySettings from '@/components/account/AccountConcurrencySettings.vue'
import AccountExpirationSettings from '@/components/account/AccountExpirationSettings.vue'
import AccountProxySettings from '@/components/account/AccountProxySettings.vue'
import type { Proxy } from '@/types'
import type { AccountAutoRenewPeriod } from '@/types'

defineProps<{
  proxies: Proxy[]
}>()

const proxyId = defineModel<number | null>('proxyId', { required: true })
const concurrency = defineModel<number>('concurrency', { required: true })
const loadFactor = defineModel<number | null>('loadFactor', { required: true })
const priority = defineModel<number>('priority', { required: true })
const rateMultiplier = defineModel<number>('rateMultiplier', { required: true })
const expiresAtInput = defineModel<string>('expiresAtInput', { required: true })
const expiryProbeExtensionDays = defineModel<number>('expiryProbeExtensionDays', { required: true })
const autoRenewEnabled = defineModel<boolean>('autoRenewEnabled', { required: true })
const autoRenewPeriod = defineModel<AccountAutoRenewPeriod>('autoRenewPeriod', { required: true })
</script>

<template>
  <AccountProxySettings
    v-model:proxy-id="proxyId"
    :proxies="proxies"
  />

  <AccountConcurrencySettings
    v-model:concurrency="concurrency"
    v-model:load-factor="loadFactor"
    v-model:priority="priority"
    v-model:rate-multiplier="rateMultiplier"
  />

  <AccountExpirationSettings
    v-model:expires-at-input="expiresAtInput"
    v-model:expiry-probe-extension-days="expiryProbeExtensionDays"
    v-model:auto-renew-enabled="autoRenewEnabled"
    v-model:auto-renew-period="autoRenewPeriod"
  />
</template>
