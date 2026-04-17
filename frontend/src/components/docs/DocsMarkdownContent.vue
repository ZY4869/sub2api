<template>
  <div class="space-y-6">
    <div v-if="summaryCards.length > 0" class="grid gap-4 md:grid-cols-3">
      <article
        v-for="card in summaryCards"
        :key="card.label"
        class="rounded-3xl border border-slate-200 bg-white/85 p-5 shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-900/75"
      >
        <p class="text-xs font-semibold uppercase tracking-[0.24em] text-slate-800 dark:text-slate-200">
          {{ card.label }}
        </p>
        <h3 class="mt-3 text-2xl font-semibold text-slate-950 dark:text-white">
          {{ card.value }}
        </h3>
        <p class="mt-2 text-sm leading-6 text-slate-900 dark:text-slate-100">
          {{ card.description }}
        </p>
      </article>
    </div>

    <div class="space-y-3 md:hidden">
      <DocsProtocolNav
        :pages="document.pages"
        :current-page-id="currentPage.id"
        :base-path="basePath"
        :theme="theme"
        :title="navTitle"
        compact
      />
      <DocsToc
        v-if="tocHeadings.length > 0"
        :headings="tocHeadings"
        :theme="theme"
        :active-id="activeSectionId"
        :title="tocTitle"
        compact
      />
    </div>

    <div :class="gridClass">
      <aside class="hidden md:block">
        <div class="sticky top-0 self-start max-h-screen overflow-y-auto pr-1">
          <DocsProtocolNav
            :pages="document.pages"
            :current-page-id="currentPage.id"
            :base-path="basePath"
            :theme="theme"
            :title="navTitle"
          />
        </div>
      </aside>

      <div ref="articleRef" class="min-w-0">
        <DocsPageArticle :page="currentPage" :theme="theme" />
      </div>

      <aside v-if="showDesktopToc" class="hidden md:block">
        <div class="sticky top-0 self-start max-h-screen overflow-y-auto pr-1">
          <DocsToc
            v-if="tocHeadings.length > 0"
            :headings="tocHeadings"
            :theme="theme"
            :active-id="activeSectionId"
            :title="tocTitle"
          />
        </div>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import type { MarkdownHeading } from '@/utils/markdownDocs'
import { normalizeDocsPageId, parseDocsMarkdown } from '@/utils/markdownDocs'
import { getDocsTheme } from './docsTheme'
import DocsPageArticle from './DocsPageArticle.vue'
import DocsProtocolNav from './DocsProtocolNav.vue'
import DocsToc from './DocsToc.vue'

interface SummaryCard {
  label: string
  value: string
  description: string
}

const props = withDefaults(defineProps<{
  markdown: string
  pageId?: string
  basePath: string
  navTitle?: string
  tocTitle?: string
  previewMode?: boolean
  summaryCards?: SummaryCard[]
}>(), {
  pageId: 'common',
  navTitle: '支持协议',
  tocTitle: '本页内容',
  previewMode: false,
  summaryCards: () => []
})

const articleRef = ref<HTMLElement | null>(null)
const activeSectionId = ref('')

const document = computed(() => parseDocsMarkdown(props.markdown))
const currentPageId = computed(() => normalizeDocsPageId(props.pageId))
const currentPage = computed(() =>
  document.value.pages.find((page) => page.id === currentPageId.value) ?? document.value.pages[0]
)
const theme = computed(() => getDocsTheme(currentPage.value.id))
const tocHeadings = computed<MarkdownHeading[]>(() =>
  currentPage.value.sections.map((section) => ({
    id: section.id,
    text: section.title,
    level: section.level
  }))
)
const gridClass = computed(() =>
  props.previewMode
    ? 'grid gap-4 md:grid-cols-[220px,minmax(0,1fr)] lg:gap-5 lg:grid-cols-[250px,minmax(0,1fr)] xl:gap-6 xl:grid-cols-[290px,minmax(0,1fr)]'
    : 'grid gap-4 md:grid-cols-[200px,minmax(0,1fr),180px] lg:gap-5 lg:grid-cols-[240px,minmax(0,1fr),200px] xl:gap-6 xl:grid-cols-[290px,minmax(0,1fr),240px]'
)
const showDesktopToc = computed(() => !props.previewMode && tocHeadings.value.length > 0)

let sectionObserver: IntersectionObserver | null = null

function cleanupObserver() {
  if (sectionObserver) {
    sectionObserver.disconnect()
    sectionObserver = null
  }
}

async function observeSections() {
  cleanupObserver()
  await nextTick()

  const sectionIds = currentPage.value.sections.map((section) => section.id)
  activeSectionId.value = sectionIds[0] ?? ''

  if (typeof window === 'undefined' || !('IntersectionObserver' in window) || !articleRef.value) {
    return
  }

  const sections = Array.from(
    articleRef.value.querySelectorAll<HTMLElement>('[data-docs-section-id]')
  )
  if (sections.length === 0) {
    return
  }

  sectionObserver = new window.IntersectionObserver(
    (entries) => {
      const visible = entries
        .filter((entry) => entry.isIntersecting)
        .sort((left, right) => left.boundingClientRect.top - right.boundingClientRect.top)
      const nextActive = visible[0]?.target.getAttribute('data-docs-section-id')
      if (nextActive) {
        activeSectionId.value = nextActive
      }
    },
    {
      rootMargin: '-120px 0px -60% 0px',
      threshold: [0, 0.2, 1]
    }
  )

  sections.forEach((section) => sectionObserver?.observe(section))
}

watch(
  () => [props.markdown, currentPage.value.id] as const,
  () => {
    void observeSections()
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  cleanupObserver()
})
</script>
