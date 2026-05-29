<template>
  <div class="mb-4 flex items-start justify-between gap-4">
    <div class="flex min-w-0 items-center gap-3.5">
      <div class="relative flex h-[38px] w-[38px] shrink-0 items-center justify-center rounded-xl border border-slate-100 bg-slate-50 dark:border-dark-700 dark:bg-dark-800">
        <ModelPlatformIcon :platform="item.raw.provider_icon_key || item.raw.provider || ''" size="md" />
        <span class="absolute -bottom-1 -right-1 rounded-full bg-white p-[2px] dark:bg-dark-900">
          <span class="block h-2.5 w-2.5 rounded-full" :class="statusDotClass"></span>
        </span>
      </div>

      <div class="min-w-0">
        <h2 class="flex items-center gap-2 text-lg font-black leading-tight tracking-tight text-slate-800 dark:text-white">
          <ModelIcon
            :model="item.raw.model"
            :provider="item.raw.provider"
            :display-name="item.raw.display_name"
            size="20px"
          />
          <span class="truncate">{{ item.title }}</span>
        </h2>
        <div class="mt-1 flex flex-wrap items-center gap-2 text-[11px] font-mono text-slate-400">
          <button
            type="button"
            class="group/copy inline-flex min-w-0 items-center gap-1.5 transition-colors hover:text-indigo-500"
            :title="copyTitle"
            :data-testid="`public-model-copy-${item.raw.model}`"
            @click.stop="emit('copy', item.raw)"
          >
            <ModelIcon
              :model="item.raw.model"
              :provider="item.raw.provider"
              :display-name="item.raw.display_name"
              size="13px"
            />
            <span class="truncate">{{ item.raw.model }}</span>
            <Icon name="copy" size="xs" class="opacity-0 transition-opacity group-hover/copy:opacity-100" />
          </button>
          <span class="h-[3px] w-[3px] rounded-full bg-slate-300"></span>
          <span class="inline-flex items-center gap-1.5 rounded-md border px-2 py-[2px] text-[10px] font-black uppercase tracking-wider" :class="contextClass">
            <Icon name="book" size="xs" />
            {{ contextLabel }}
          </span>
        </div>
      </div>
    </div>

    <button
      type="button"
      class="flex h-8 shrink-0 items-center gap-1 rounded-full border border-slate-200/80 px-3 text-[11px] font-bold text-slate-500 transition-colors hover:border-indigo-200 hover:bg-indigo-50 hover:text-indigo-600 dark:border-dark-700 dark:text-slate-300 dark:hover:border-indigo-500/50 dark:hover:bg-indigo-500/10 dark:hover:text-indigo-200"
      :title="detailTitle"
      :data-testid="`public-model-detail-${item.raw.model}`"
      @click.stop="emit('openDetail', item.raw)"
    >
      <span>{{ detailLabel }}</span>
      <Icon name="chevronRight" size="xs" />
    </button>
  </div>
</template>

<script setup lang="ts">
import type { PublicModelCatalogItem } from '@/api/meta'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import type { PublicModelCatalogDisplayItem } from '@/utils/publicModelCatalog'

defineProps<{
  item: PublicModelCatalogDisplayItem
  statusDotClass: string
  contextLabel: string
  contextClass: string
  copyTitle: string
  detailLabel: string
  detailTitle: string
}>()

const emit = defineEmits<{
  copy: [item: PublicModelCatalogItem]
  openDetail: [item: PublicModelCatalogItem]
}>()
</script>
