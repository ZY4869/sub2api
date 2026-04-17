<template>
  <div class="overflow-hidden rounded-[1.5rem] border border-slate-200 bg-slate-950 shadow-sm dark:border-dark-700">
    <div class="flex flex-wrap gap-2 border-b border-white/10 bg-slate-900/80 px-4 py-3">
      <button
        v-for="tab in group.tabs"
        :key="tab.id"
        type="button"
        class="rounded-full px-3 py-1.5 text-xs font-semibold tracking-[0.04em] transition-colors"
        :class="selectedTabId === tab.id ? theme.tabActiveClass : 'bg-white/10 text-slate-100 hover:bg-white/15'"
        @click="selectedTabId = tab.id"
      >
        {{ tab.label }}
      </button>
    </div>

    <div class="overflow-x-auto">
      <pre class="min-w-full px-5 py-5 text-sm leading-7 text-slate-100"><code>{{ activeTab?.code }}</code></pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { DocsCodeExampleGroup } from '@/utils/markdownDocs'
import type { DocsTheme } from './docsTheme'

const props = defineProps<{
  group: DocsCodeExampleGroup
  theme: DocsTheme
}>()

const selectedTabId = ref(props.group.tabs[0]?.id ?? '')

const activeTab = computed(() => props.group.tabs.find((tab) => tab.id === selectedTabId.value) ?? props.group.tabs[0])

watch(
  () => props.group.id,
  () => {
    selectedTabId.value = props.group.tabs[0]?.id ?? ''
  }
)
</script>
