<template>
  <section
    v-if="hasContent"
    class="space-y-3 rounded-2xl border border-sky-200 bg-sky-50/80 p-4 dark:border-sky-900/50 dark:bg-sky-950/20"
  >
    <div class="space-y-1">
      <div class="text-sm font-semibold text-sky-950 dark:text-sky-100">
        {{ title || t('admin.accounts.probeFinalize.upstreamTitle') }}
      </div>
      <p class="text-xs text-sky-800 dark:text-sky-300">
        {{ hint || t('admin.accounts.probeFinalize.upstreamHint') }}
      </p>
    </div>

    <div class="grid gap-3 md:grid-cols-2">
      <div v-if="upstreamUrl" class="space-y-1 rounded-xl bg-white/80 px-3 py-2 dark:bg-slate-900/60">
        <div class="text-[11px] font-medium uppercase tracking-wide text-sky-700 dark:text-sky-300">
          {{ t('admin.accounts.probeFinalize.upstreamUrl') }}
        </div>
        <div class="break-all text-sm text-slate-700 dark:text-slate-100">{{ upstreamUrl }}</div>
      </div>
      <div v-if="upstreamHost" class="space-y-1 rounded-xl bg-white/80 px-3 py-2 dark:bg-slate-900/60">
        <div class="text-[11px] font-medium uppercase tracking-wide text-sky-700 dark:text-sky-300">
          {{ t('admin.accounts.probeFinalize.upstreamHost') }}
        </div>
        <div class="break-all text-sm text-slate-700 dark:text-slate-100">{{ upstreamHost }}</div>
      </div>
      <div v-if="upstreamService" class="space-y-1 rounded-xl bg-white/80 px-3 py-2 dark:bg-slate-900/60">
        <div class="text-[11px] font-medium uppercase tracking-wide text-sky-700 dark:text-sky-300">
          {{ t('admin.accounts.probeFinalize.upstreamService') }}
        </div>
        <div class="break-all text-sm text-slate-700 dark:text-slate-100">{{ upstreamService }}</div>
      </div>
      <div v-if="upstreamRegion" class="space-y-1 rounded-xl bg-white/80 px-3 py-2 dark:bg-slate-900/60">
        <div class="text-[11px] font-medium uppercase tracking-wide text-sky-700 dark:text-sky-300">
          {{ t('admin.accounts.probeFinalize.upstreamRegion') }}
        </div>
        <div class="break-all text-sm text-slate-700 dark:text-slate-100">{{ upstreamRegion }}</div>
      </div>
      <div v-if="probeSource" class="space-y-1 rounded-xl bg-white/80 px-3 py-2 dark:bg-slate-900/60">
        <div class="text-[11px] font-medium uppercase tracking-wide text-sky-700 dark:text-sky-300">
          {{ t('admin.accounts.probeFinalize.upstreamProbeSource') }}
        </div>
        <div class="break-all text-sm text-slate-700 dark:text-slate-100">{{ probeSource }}</div>
      </div>
      <div v-if="probedAt" class="space-y-1 rounded-xl bg-white/80 px-3 py-2 dark:bg-slate-900/60">
        <div class="text-[11px] font-medium uppercase tracking-wide text-sky-700 dark:text-sky-300">
          {{ t('admin.accounts.probeFinalize.upstreamProbedAt') }}
        </div>
        <div class="break-all text-sm text-slate-700 dark:text-slate-100">{{ probedAt }}</div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  upstreamUrl?: string
  upstreamHost?: string
  upstreamService?: string
  upstreamRegion?: string
  probeSource?: string
  probedAt?: string
  title?: string
  hint?: string
}>()

const { t } = useI18n()

const hasContent = computed(() =>
  Boolean(
    props.upstreamUrl ||
      props.upstreamHost ||
      props.upstreamService ||
      props.upstreamRegion ||
      props.probeSource ||
      props.probedAt
  )
)
</script>
