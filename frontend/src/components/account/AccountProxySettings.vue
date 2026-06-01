<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ProxySelector from '@/components/common/ProxySelector.vue'
import type { Proxy } from '@/types'

defineProps<{
  proxies: Proxy[]
}>()

const proxyId = defineModel<number | null>('proxyId', { required: true })

const { t } = useI18n()
</script>

<template>
  <div>
    <label class="input-label">{{ t('admin.accounts.proxy') }}</label>
    <ProxySelector v-model="proxyId" :proxies="proxies" />
  </div>
</template>
