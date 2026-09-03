<template>
  <header class="shrink-0 bg-white px-4 pt-5 dark:bg-dark-900 sm:px-6 md:px-8 md:pt-7 2xl:px-10">
    <div class="flex items-start justify-between gap-4">
      <div class="flex min-w-0 items-start gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg border border-slate-200/80 bg-white text-slate-800 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-white sm:h-14 sm:w-14">
          <ModelPlatformIcon
            v-if="item"
            :platform="item.provider_icon_key || item.provider || ''"
            size="lg"
          />
        </div>
        <div class="min-w-0">
          <div class="flex min-w-0 items-center gap-2 sm:gap-3">
            <ModelIcon
              v-if="item"
              :model="item.model"
              :provider="item.provider"
              :display-name="item.display_name"
              size="24px"
            />
            <h1 class="truncate text-xl font-extrabold leading-tight text-slate-900 dark:text-white sm:text-2xl">
              {{ item?.model || title }}
            </h1>
            <button
              type="button"
              class="rounded-md p-1.5 text-slate-400 transition-colors hover:bg-indigo-50 hover:text-indigo-600 dark:hover:bg-indigo-500/10 dark:hover:text-indigo-200"
              :title="copyTitle"
              @click="emit('copy')"
            >
              <Icon name="copy" size="sm" />
            </button>
          </div>
          <div class="mt-2 flex flex-wrap items-center gap-2 sm:gap-3">
            <span class="flex items-center gap-1.5 border-r border-slate-200 pr-3 text-sm font-bold text-slate-600 dark:border-dark-700 dark:text-slate-200">
              <span class="h-2 w-2 rounded-full" :class="statusClass"></span>
              {{ providerLabel }}
            </span>
            <span class="text-sm tracking-wide text-slate-500 dark:text-slate-400">
              {{ hostedSummary }}
            </span>
          </div>
        </div>
      </div>

      <button
        type="button"
        class="shrink-0 rounded-lg border border-slate-200 bg-white p-2.5 text-slate-400 shadow-sm transition-all hover:bg-slate-100 hover:text-slate-800 dark:border-dark-700 dark:bg-dark-800 dark:hover:bg-dark-700 dark:hover:text-white"
        :title="closeTitle"
        @click="emit('close')"
      >
        <Icon name="x" size="md" :stroke-width="2.5" />
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import type { PublicModelCatalogItem } from '@/api/meta'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  item: PublicModelCatalogItem | null
  title: string
  providerLabel: string
  statusClass: string
  copyTitle: string
  closeTitle: string
  hostedSummary: string
}>()

const emit = defineEmits<{
  copy: []
  close: []
}>()
</script>
