<template>
  <AppLayout>
    <div class="space-y-6">
      <ModelCatalogSubnav />
      <RouterView />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { RouterView } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import ModelCatalogSubnav from '@/components/admin/models/ModelCatalogSubnav.vue'
</script>
