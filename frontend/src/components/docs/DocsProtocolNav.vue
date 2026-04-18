<template>
  <nav
    :class="compact
      ? 'flex gap-2 overflow-x-auto pb-2'
      : 'rounded-[1.75rem] border border-slate-200 bg-white/90 p-4 shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-900/80'"
  >
    <template v-if="compact">
      <RouterLink
        v-for="page in pages"
        :key="page.id"
        :to="buildPagePath(page.id)"
        class="inline-flex items-center gap-2 whitespace-nowrap rounded-full border px-3 py-1.5 text-sm font-medium transition-colors"
        :class="page.id === currentPageId
          ? theme.navActiveClass
          : 'border-slate-200 text-slate-950 hover:border-slate-300 hover:text-slate-950 dark:border-dark-700 dark:text-white dark:hover:text-white'"
      >
        <span class="inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-950 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-white">
          <PlatformIcon
            v-if="getPlatformVisual(page.id)"
            :platform="getPlatformVisual(page.id)"
            size="sm"
          />
          <Icon
            v-else
            :name="getGenericIcon(page.id)"
            size="xs"
          />
        </span>
        <span>{{ page.shortTitle }}</span>
      </RouterLink>
    </template>

    <template v-else>
      <p class="mb-3 text-xs font-semibold uppercase tracking-[0.24em] text-slate-800 dark:text-slate-200">
        {{ title }}
      </p>
      <div class="space-y-2">
        <RouterLink
          v-for="page in pages"
          :key="page.id"
          :to="buildPagePath(page.id)"
          class="block rounded-[1.35rem] border px-4 py-3.5 transition-colors"
          :class="page.id === currentPageId
            ? theme.navActiveClass
            : 'border-transparent text-slate-950 hover:border-slate-200 hover:bg-slate-50 dark:text-white dark:hover:border-dark-700 dark:hover:bg-dark-800'"
        >
          <div class="flex items-start gap-3">
            <span
              class="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border shadow-sm"
              :class="page.id === currentPageId
                ? 'border-white/80 bg-white text-slate-950 dark:border-white/10 dark:bg-dark-800 dark:text-white'
                : 'border-slate-200 bg-slate-50 text-slate-950 dark:border-dark-700 dark:bg-dark-800/80 dark:text-white'"
            >
              <PlatformIcon
                v-if="getPlatformVisual(page.id)"
                :platform="getPlatformVisual(page.id)"
                size="lg"
              />
              <Icon
                v-else
                :name="getGenericIcon(page.id)"
                size="md"
              />
            </span>

            <div class="min-w-0 flex-1">
              <p class="text-sm font-semibold text-slate-950 dark:text-white">{{ page.title }}</p>
              <p class="mt-1 text-xs leading-5 text-slate-900 dark:text-slate-100">
                {{ page.description }}
              </p>
            </div>
          </div>
        </RouterLink>
      </div>
    </template>
  </nav>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import type { AccountPlatform } from '@/types'
import type { DocsPage, DocsPageId } from '@/utils/markdownDocs'
import type { DocsTheme } from './docsTheme'

const props = withDefaults(defineProps<{
  pages: DocsPage[]
  currentPageId: DocsPageId
  basePath: string
  theme: DocsTheme
  compact?: boolean
  title?: string
}>(), {
  compact: false,
  title: '支持协议',
})

const platformVisuals: Partial<Record<DocsPageId, AccountPlatform>> = {
  'openai-native': 'openai',
  anthropic: 'anthropic',
  gemini: 'gemini',
  grok: 'grok',
  antigravity: 'antigravity',
  'vertex-batch': 'gemini',
  'document-ai': 'baidu_document_ai',
}

const genericVisuals: Partial<Record<DocsPageId, 'book' | 'swap'>> = {
  common: 'book',
  openai: 'swap',
}

function buildPagePath(pageId: DocsPageId) {
  return `${props.basePath}/${pageId}`
}

function getPlatformVisual(pageId: DocsPageId) {
  return platformVisuals[pageId]
}

function getGenericIcon(pageId: DocsPageId): 'book' | 'swap' {
  return genericVisuals[pageId] ?? 'book'
}
</script>
