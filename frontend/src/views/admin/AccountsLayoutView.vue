<template>
  <AppLayout>
    <div class="space-y-6">
      <AccountsManagementSubnav />
      <RouterView />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { RouterView } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import AccountsManagementSubnav from '@/components/admin/account/AccountsManagementSubnav.vue'
</script>
