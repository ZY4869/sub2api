<template>
  <AppLayout>
    <div class="space-y-6">
      <BillingSubnav />
      <RouterView />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { RouterView } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import BillingSubnav from '@/components/admin/billing/BillingSubnav.vue'
</script>
