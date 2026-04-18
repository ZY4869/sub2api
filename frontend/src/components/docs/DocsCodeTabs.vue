<template>
  <div class="overflow-hidden rounded-[1.5rem] border border-slate-200 bg-slate-950 shadow-sm dark:border-dark-700">
    <div class="border-b border-white/10 bg-slate-900/85 px-4 py-3">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="flex flex-wrap gap-2">
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

        <button
          type="button"
          class="inline-flex items-center rounded-full border border-white/10 bg-white/10 px-3 py-1.5 text-xs font-semibold text-slate-100 transition hover:bg-white/15"
          data-test="docs-code-copy"
          @click="handleCopy"
        >
          {{ t('ui.apiDocs.copyCode') }}
        </button>
      </div>
    </div>
    <DocsCodePanel
      :code="activeTab?.code ?? ''"
      :focus-lines="activeTab?.focusLines"
      :language="activeTab?.language ?? 'text'"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useClipboard } from '@/composables/useClipboard'
import type { DocsCodeExampleGroup } from '@/utils/markdownDocs'
import type { DocsTheme } from './docsTheme'
import DocsCodePanel from './DocsCodePanel.vue'

const props = defineProps<{
  group: DocsCodeExampleGroup
  theme: DocsTheme
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const selectedTabId = ref(props.group.tabs[0]?.id ?? '')

const activeTab = computed(
  () => props.group.tabs.find((tab) => tab.id === selectedTabId.value) ?? props.group.tabs[0],
)

async function handleCopy() {
  if (!activeTab.value) {
    return
  }
  await copyToClipboard(activeTab.value.code)
}

watch(
  () => props.group.id,
  () => {
    selectedTabId.value = props.group.tabs[0]?.id ?? ''
  },
)
</script>
