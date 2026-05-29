<template>
  <div class="rounded-3xl border border-slate-800 bg-slate-900 shadow-xl">
    <div class="flex flex-col gap-3 border-b border-slate-800 px-6 py-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="hidden items-center gap-2 sm:flex">
        <span class="h-3 w-3 rounded-full bg-rose-500"></span>
        <span class="h-3 w-3 rounded-full bg-amber-500"></span>
        <span class="h-3 w-3 rounded-full bg-emerald-500"></span>
      </div>
      <select
        v-if="supportedKeys.length > 1"
        :value="selectedKeyID ?? ''"
        class="input max-w-xs border-slate-700 bg-slate-800 text-slate-100"
        data-testid="public-model-detail-key-selector"
        @change="emit('update:selectedKeyID', Number(($event.target as HTMLSelectElement).value))"
      >
        <option v-for="item in supportedKeys" :key="item.id" :value="item.id">
          {{ item.name }}
        </option>
      </select>
    </div>

    <div v-if="loading" class="p-6 text-sm text-slate-300">
      {{ labels.loading }}
    </div>
    <div v-else-if="errorMessage" class="m-6 rounded-2xl border border-rose-500/30 bg-rose-500/10 p-4 text-sm text-rose-100">
      {{ errorMessage }}
    </div>
    <div v-else-if="exampleGroup" class="p-6">
      <div class="mb-4 flex flex-wrap gap-2 text-xs">
        <span class="rounded-full bg-slate-800 px-2.5 py-1 text-slate-200">
          {{ protocol || 'openai' }}
        </span>
        <span class="rounded-full bg-emerald-500/15 px-2.5 py-1 text-emerald-200">
          {{ exampleSource === 'docs_section' ? labels.exampleSourceDocs : labels.exampleSourceOverride }}
        </span>
      </div>
      <div class="mb-4 rounded-2xl border border-indigo-500/20 bg-indigo-500/10 px-4 py-3 text-xs text-indigo-100">
        {{ keyHint }}
      </div>
      <DocsCodeTabs :group="exampleGroup" :theme="docsTheme" />
    </div>
    <div v-else class="p-6 text-sm text-slate-300">
      {{ labels.exampleUnavailable }}
    </div>
  </div>
</template>

<script setup lang="ts">
import DocsCodeTabs from '@/components/docs/DocsCodeTabs.vue'
import type { DocsCodeExampleGroup } from '@/utils/docsCodeExamples'
import type { DocsTheme } from '@/components/docs/docsTheme'
import type { PublicModelSupportedKey } from '@/utils/publicModelCatalogKeys'

defineProps<{
  supportedKeys: PublicModelSupportedKey[]
  selectedKeyID: number | null
  loading: boolean
  errorMessage: string
  exampleGroup: DocsCodeExampleGroup | null
  docsTheme: DocsTheme
  protocol: string
  exampleSource?: string
  keyHint: string
  labels: {
    loading: string
    exampleSourceDocs: string
    exampleSourceOverride: string
    exampleUnavailable: string
  }
}>()

const emit = defineEmits<{
  'update:selectedKeyID': [value: number]
}>()
</script>
