<template>
  <nav
    v-if="headings.length > 0"
    :class="compact
      ? 'flex gap-2 overflow-x-auto pb-2'
      : 'rounded-[1.75rem] border border-slate-200 bg-white/90 p-4 shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-900/80'"
  >
    <template v-if="compact">
      <a
        v-for="heading in headings"
        :key="heading.id"
        :href="`#${heading.id}`"
        class="whitespace-nowrap rounded-full border px-3 py-1.5 text-sm font-medium transition-colors"
        :class="heading.id === activeId
          ? theme.tocActiveClass
          : 'border-slate-200 text-slate-950 hover:border-slate-300 hover:text-slate-950 dark:border-dark-700 dark:text-white dark:hover:text-white'"
      >
        {{ heading.text }}
      </a>
    </template>

    <template v-else>
      <p class="mb-3 text-xs font-semibold uppercase tracking-[0.24em] text-slate-800 dark:text-slate-200">
        {{ title }}
      </p>
      <div class="space-y-1">
        <a
          v-for="heading in headings"
          :key="heading.id"
          :href="`#${heading.id}`"
          class="block rounded-2xl border px-3 py-2 text-sm font-medium transition-colors"
          :class="heading.id === activeId
            ? theme.tocActiveClass
            : 'border-transparent text-slate-950 hover:border-slate-200 hover:bg-slate-50 hover:text-slate-950 dark:text-white dark:hover:border-dark-700 dark:hover:bg-dark-800 dark:hover:text-white'"
        >
          {{ heading.text }}
        </a>
      </div>
    </template>
  </nav>
</template>

<script setup lang="ts">
import type { MarkdownHeading } from '@/utils/markdownDocs'
import type { DocsTheme } from './docsTheme'

withDefaults(defineProps<{
  headings: MarkdownHeading[]
  theme: DocsTheme
  activeId?: string
  compact?: boolean
  title?: string
}>(), {
  activeId: '',
  compact: false,
  title: '本页内容'
})
</script>
