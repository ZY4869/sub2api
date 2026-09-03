<template>
  <div class="flex min-w-0 flex-col gap-6 pb-8">
    <section class="w-full">
      <div class="mb-4 flex items-center justify-between">
        <h3 class="flex items-center gap-2 text-sm font-extrabold uppercase tracking-widest text-slate-800 dark:text-white">
          {{ labels.exampleTitle }}
        </h3>
      </div>
      <slot name="example"></slot>
    </section>

    <section class="w-full">
      <div class="relative flex flex-col gap-4 overflow-hidden rounded-lg border border-amber-200/70 bg-amber-50 p-4 shadow-sm dark:border-amber-500/30 dark:bg-amber-500/10 md:flex-row md:items-center 2xl:p-5">
        <div class="shrink-0 rounded-lg border border-amber-200/50 bg-amber-100/80 p-3 text-amber-600 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200">
          <Icon name="key" size="xl" />
        </div>
        <div class="relative z-10 flex-1">
          <h4 class="mb-2 flex items-center gap-1.5 text-[15px] font-black uppercase tracking-widest text-amber-900 dark:text-amber-100">
            {{ labels.authentication }}
          </h4>
          <div class="text-[13px] leading-relaxed text-amber-800/90 dark:text-amber-100/90">
            {{ labels.authenticationText }}
            <code class="mx-1 inline-block rounded-md border border-amber-300/50 bg-amber-200/60 px-2.5 py-1 font-mono text-[13px] font-black text-amber-900 shadow-sm dark:border-amber-500/30 dark:bg-amber-500/20 dark:text-amber-100">
              Authorization: Bearer &lt;TOKEN&gt;
            </code>
          </div>
        </div>
      </div>
    </section>

    <section class="w-full">
      <h3 class="mb-4 flex items-center gap-2 text-sm font-extrabold uppercase tracking-widest text-slate-800 dark:text-white">
        {{ labels.endpoints }}
      </h3>
      <div class="min-w-0 overflow-hidden rounded-lg border border-slate-200/60 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div
          v-for="endpoint in endpoints"
          :key="endpoint.key"
          class="grid grid-cols-1 gap-2 border-b border-slate-100 p-4 last:border-b-0 dark:border-dark-700 md:grid-cols-[minmax(9rem,14rem)_7rem_minmax(0,1fr)_auto] md:items-center 2xl:p-5"
        >
          <div class="min-w-0 truncate font-mono text-[13px] font-bold text-indigo-600 dark:text-indigo-300" :title="endpoint.key">{{ endpoint.key }}</div>
          <div>
            <span class="rounded-md px-2.5 py-1 text-[11px] font-bold" :class="supportClass(endpoint.support)">
              {{ supportText(endpoint.support) }}
            </span>
          </div>
          <div class="min-w-0 break-all font-mono text-xs text-slate-500 dark:text-slate-300">{{ endpoint.method || 'POST' }} {{ endpoint.endpoint }}</div>
          <div class="text-xs text-slate-400 md:text-right">{{ sourceText(endpoint.source, endpoint.verified) }}</div>
        </div>
        <div v-if="endpoints.length === 0" class="p-5 text-sm text-slate-400">-</div>
      </div>
    </section>

    <section class="w-full">
      <h3 class="mb-4 flex items-center gap-2 text-sm font-extrabold uppercase tracking-widest text-slate-800 dark:text-white">
        {{ labels.parameters }}
      </h3>
      <div class="min-w-0 overflow-hidden rounded-lg border border-slate-200/60 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div
          v-for="param in params"
          :key="param.name"
          class="grid grid-cols-1 gap-2 border-b border-slate-100 p-4 last:border-b-0 hover:bg-slate-50 dark:border-dark-700 dark:hover:bg-dark-800 md:grid-cols-[minmax(9rem,12rem)_6rem_8rem_minmax(0,1fr)] md:items-center 2xl:p-5"
        >
          <div class="min-w-0 truncate font-mono text-[14px] font-bold text-indigo-600 dark:text-indigo-300" :title="param.name">{{ param.name }}</div>
          <div>
            <span class="rounded-md border border-slate-200 bg-slate-100 px-2.5 py-1 text-[11px] font-bold text-slate-500 dark:border-dark-700 dark:bg-dark-800 dark:text-slate-300">{{ param.type }}</span>
          </div>
          <div class="min-w-0">
            <span class="inline-block max-w-full truncate rounded bg-emerald-50 px-2 py-0.5 font-mono text-[12px] font-bold text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-200" :title="param.defaultValue">{{ param.defaultValue }}</span>
          </div>
          <div class="min-w-0 text-[13px] leading-relaxed text-slate-600 dark:text-slate-300">
            {{ param.description }}
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'
import type { PublicModelProtocolEndpoint } from '@/api/meta'
import { sourceLabel, supportLabel, type Translate } from './publicModelCatalogView'

const props = defineProps<{
  labels: Record<string, string>
  endpoints: PublicModelProtocolEndpoint[]
  t: Translate
  params: Array<{
    name: string
    type: string
    defaultValue: string
    description: string
  }>
}>()

function supportText(value?: string): string {
  return supportLabel(props.t, value)
}

function sourceText(source?: string, verified?: boolean): string {
  return sourceLabel(props.t, source, verified)
}

function supportClass(value?: string): string {
  switch (value) {
    case 'supported':
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-200'
    case 'partial':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-200'
    case 'unsupported':
      return 'bg-rose-50 text-rose-700 dark:bg-rose-500/10 dark:text-rose-200'
    default:
      return 'bg-slate-100 text-slate-500 dark:bg-dark-700 dark:text-slate-300'
  }
}
</script>
