<template>
  <AdminAccountTestModal :show="show" :account="account" @close="emit('close')" />
</template>

<script setup lang="ts">
import AdminAccountTestModal from '@/components/admin/account/AccountTestModal.vue'
import type { Account } from '@/types'

defineProps<{
  show: boolean
  account: Account | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()
</script>
