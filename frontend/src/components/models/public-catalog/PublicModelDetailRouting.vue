<template>
  <div class="flex flex-col gap-8 pb-10">
    <section class="w-full">
      <div class="mb-4 flex items-center justify-between">
        <h3 class="flex items-center gap-2 text-sm font-extrabold uppercase tracking-widest text-slate-800 dark:text-white">
          {{ labels.exampleTitle }}
        </h3>
      </div>
      <slot name="example"></slot>
    </section>

    <section class="w-full">
      <div class="relative flex flex-col gap-5 overflow-hidden rounded-3xl border border-amber-200/70 bg-amber-50 p-6 shadow-sm dark:border-amber-500/30 dark:bg-amber-500/10 md:flex-row md:items-center">
        <div class="shrink-0 rounded-2xl border border-amber-200/50 bg-amber-100/80 p-3.5 text-amber-600 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200">
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
      <div class="overflow-hidden rounded-3xl border border-slate-200/60 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div
          v-for="endpoint in endpoints"
          :key="endpoint.key"
          class="flex flex-col gap-3 border-b border-slate-100 p-5 last:border-b-0 dark:border-dark-700 md:flex-row md:items-center"
        >
          <div class="w-52 shrink-0 font-mono text-[13px] font-bold text-indigo-600 dark:text-indigo-300">{{ endpoint.key }}</div>
          <div class="w-28 shrink-0">
            <span class="rounded-md px-2.5 py-1 text-[11px] font-bold" :class="supportClass(endpoint.support)">
              {{ supportText(endpoint.support) }}
            </span>
          </div>
          <div class="flex-1 font-mono text-xs text-slate-500 dark:text-slate-300">{{ endpoint.method || 'POST' }} {{ endpoint.endpoint }}</div>
          <div class="text-xs text-slate-400">{{ sourceText(endpoint.source, endpoint.verified) }}</div>
        </div>
        <div v-if="endpoints.length === 0" class="p-5 text-sm text-slate-400">-</div>
      </div>
    </section>

    <section class="w-full">
      <h3 class="mb-4 flex items-center gap-2 text-sm font-extrabold uppercase tracking-widest text-slate-800 dark:text-white">
        {{ labels.parameters }}
      </h3>
      <div class="overflow-hidden rounded-3xl border border-slate-200/60 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div
          v-for="param in params"
          :key="param.name"
          class="flex flex-col gap-4 border-b border-slate-100 p-5 last:border-b-0 hover:bg-slate-50 dark:border-dark-700 dark:hover:bg-dark-800 md:flex-row md:items-center"
        >
          <div class="w-48 shrink-0 font-mono text-[14px] font-bold text-indigo-600 dark:text-indigo-300">{{ param.name }}</div>
          <div class="w-24 shrink-0">
            <span class="rounded-md border border-slate-200 bg-slate-100 px-2.5 py-1 text-[11px] font-bold text-slate-500 dark:border-dark-700 dark:bg-dark-800 dark:text-slate-300">{{ param.type }}</span>
          </div>
          <div class="w-32 shrink-0">
            <span class="w-fit rounded bg-emerald-50 px-2 py-0.5 font-mono text-[12px] font-bold text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-200">{{ param.defaultValue }}</span>
          </div>
          <div class="flex-1 text-[13px] leading-relaxed text-slate-600 dark:text-slate-300">
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
