<template>
  <div class="space-y-3">
    <div class="space-y-3 md:hidden">
      <DocsToc
        v-if="headings.length > 0"
        :headings="headings"
        :theme="theme"
        :active-id="activeHeadingId"
        :title="tocTitle"
        compact
      />
    </div>

    <div class="grid gap-4 md:grid-cols-[minmax(0,1fr),200px] lg:gap-5 xl:gap-6 xl:grid-cols-[minmax(0,1fr),240px]">
      <article
        ref="articleRef"
        class="min-w-0 overflow-hidden rounded-[2rem] border border-slate-200 bg-white/90 px-6 py-6 shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-900/80 md:px-8 md:py-8"
      >
        <header class="mb-8 border-b border-slate-200 pb-6 dark:border-dark-700">
          <h2 class="text-2xl font-semibold tracking-tight text-slate-950 dark:text-white">
            {{ title }}
          </h2>
        </header>

        <div class="custom-markdown-page prose max-w-none" v-html="renderedHtml"></div>
      </article>

      <aside v-if="headings.length > 0" class="hidden md:block">
        <div class="sticky top-0 self-start max-h-screen overflow-y-auto pr-1">
          <DocsToc
            :headings="headings"
            :theme="theme"
            :active-id="activeHeadingId"
            :title="tocTitle"
          />
        </div>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import DocsToc from '@/components/docs/DocsToc.vue'
import { getDocsTheme } from '@/components/docs/docsTheme'
import { extractMarkdownHeadings, renderMarkdownDocument } from '@/utils/markdownDocs'
import { sanitizeUrl } from '@/utils/url'

const props = defineProps<{
  markdown: string
  title: string
  tocTitle: string
}>()

const theme = getDocsTheme('common')
const articleRef = ref<HTMLElement | null>(null)
const activeHeadingId = ref('')

const renderedHtml = computed(() => {
  const document = renderMarkdownDocument(props.markdown)
  return hardenRenderedMarkdown(document.html)
})

const headings = computed(() => extractMarkdownHeadings(props.markdown))

let sectionObserver: IntersectionObserver | null = null

function cleanupObserver() {
  if (sectionObserver) {
    sectionObserver.disconnect()
    sectionObserver = null
  }
}

async function observeHeadings() {
  cleanupObserver()
  await nextTick()

  activeHeadingId.value = headings.value[0]?.id || ''

  if (typeof window === 'undefined' || !('IntersectionObserver' in window) || !articleRef.value) {
    return
  }

  const sections = Array.from(articleRef.value.querySelectorAll<HTMLElement>('h1, h2, h3, h4'))
    .filter((node) => !!node.id)

  if (sections.length === 0) {
    return
  }

  sectionObserver = new window.IntersectionObserver(
    (entries) => {
      const visible = entries
        .filter((entry) => entry.isIntersecting)
        .sort((left, right) => left.boundingClientRect.top - right.boundingClientRect.top)
      const nextActive = visible[0]?.target.id
      if (nextActive) {
        activeHeadingId.value = nextActive
      }
    },
    {
      rootMargin: '-120px 0px -60% 0px',
      threshold: [0, 0.2, 1],
    },
  )

  sections.forEach((section) => sectionObserver?.observe(section))
}

function hardenRenderedMarkdown(html: string): string {
  if (typeof DOMParser === 'undefined') {
    return html
  }

  const doc = new DOMParser().parseFromString(html, 'text/html')

  doc.querySelectorAll<HTMLAnchorElement>('a[href]').forEach((anchor) => {
    const href = anchor.getAttribute('href') || ''
    if (href.startsWith('#')) {
      return
    }
    const safeHref = sanitizeUrl(href, { allowRelative: true })
    if (!safeHref) {
      anchor.removeAttribute('href')
      return
    }
    anchor.setAttribute('href', safeHref)
    if (/^https?:\/\//i.test(safeHref)) {
      anchor.setAttribute('target', '_blank')
      anchor.setAttribute('rel', 'noopener noreferrer')
    }
  })

  doc.querySelectorAll<HTMLImageElement>('img[src]').forEach((image) => {
    const safeSrc = sanitizeUrl(image.getAttribute('src') || '', {
      allowRelative: false,
      allowDataUrl: true,
    })
    if (!safeSrc) {
      image.remove()
      return
    }
    image.setAttribute('src', safeSrc)
    image.setAttribute('loading', 'lazy')
  })

  return doc.body.innerHTML
}

watch(
  () => [props.markdown, props.title] as const,
  () => {
    void observeHeadings()
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  cleanupObserver()
})
</script>

<style scoped>
.custom-markdown-page :deep(h1),
.custom-markdown-page :deep(h2),
.custom-markdown-page :deep(h3),
.custom-markdown-page :deep(h4),
.custom-markdown-page :deep(h5),
.custom-markdown-page :deep(h6) {
  scroll-margin-top: 7rem;
  color: rgb(15 23 42);
}

.custom-markdown-page :deep(p),
.custom-markdown-page :deep(li),
.custom-markdown-page :deep(blockquote),
.custom-markdown-page :deep(td),
.custom-markdown-page :deep(th),
.custom-markdown-page :deep(strong),
.custom-markdown-page :deep(em) {
  color: rgb(17 24 39);
}

.custom-markdown-page :deep(a) {
  color: rgb(15 23 42);
  text-decoration: underline;
  text-underline-offset: 0.2em;
}

.custom-markdown-page :deep(pre) {
  overflow-x: auto;
  border-radius: 1rem;
  border: 1px solid rgb(226 232 240);
  background: rgb(248 250 252);
  color: rgb(15 23 42);
  padding: 1rem 1.25rem;
}

.custom-markdown-page :deep(pre code) {
  background: transparent;
  color: rgb(15 23 42);
  padding: 0;
}

.custom-markdown-page :deep(code) {
  border-radius: 0.5rem;
  background: rgb(241 245 249);
  color: rgb(15 23 42);
  padding: 0.18rem 0.45rem;
}

.custom-markdown-page :deep(blockquote) {
  border-left: 4px solid rgb(148 163 184);
  background: rgb(248 250 252);
  padding: 0.9rem 1rem;
}

.custom-markdown-page :deep(table) {
  display: table;
  width: 100%;
  min-width: 100%;
  table-layout: auto;
}

.custom-markdown-page :deep(th) {
  font-weight: 700;
}

.custom-markdown-page :deep(img) {
  border-radius: 1rem;
}
</style>
