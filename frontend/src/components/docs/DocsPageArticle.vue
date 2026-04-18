<template>
  <section class="min-w-0 overflow-hidden rounded-[2rem] border border-slate-200 bg-white/90 shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-900/80">
    <div class="relative overflow-hidden border-b border-slate-200 px-6 py-6 dark:border-dark-700 md:px-8">
      <div :class="`pointer-events-none absolute inset-0 bg-gradient-to-br ${theme.glowClass}`"></div>
      <div class="relative">
        <span class="inline-flex rounded-full px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em]" :class="theme.badgeClass">
          {{ page.shortTitle }}
        </span>
        <h2 class="mt-4 text-2xl font-semibold tracking-tight text-slate-950 dark:text-white">
          {{ page.title }}
        </h2>
        <p class="mt-3 max-w-3xl text-sm leading-7 text-slate-900 dark:text-slate-100">
          {{ page.description }}
        </p>
      </div>
    </div>

    <article class="space-y-8 px-6 py-6 md:px-8 md:py-8">
      <div v-for="block in page.introBlocks" :key="block.id">
        <div
          v-if="block.kind === 'markdown'"
          class="docs-markdown prose max-w-none"
          v-html="block.html"
        ></div>
        <DocsCodeTabs v-else :group="block.group" :theme="theme" />
      </div>

      <section
        v-for="section in page.sections"
        :id="section.id"
        :key="section.id"
        :data-docs-section-id="section.id"
        class="scroll-mt-28 space-y-5"
      >
        <header>
          <h3 class="text-xl font-semibold text-slate-950 dark:text-white">
            {{ section.title }}
          </h3>
        </header>

        <div v-for="block in section.contentBlocks" :key="block.id">
          <div
            v-if="block.kind === 'markdown'"
            class="docs-markdown prose max-w-none"
            v-html="block.html"
          ></div>
          <DocsCodeTabs v-else :group="block.group" :theme="theme" />
        </div>
      </section>
    </article>
  </section>
</template>

<script setup lang="ts">
import type { DocsPage } from '@/utils/markdownDocs'
import type { DocsTheme } from './docsTheme'
import DocsCodeTabs from './DocsCodeTabs.vue'

defineProps<{
  page: DocsPage
  theme: DocsTheme
}>()
</script>

<style scoped>
.docs-markdown :deep(h1),
.docs-markdown :deep(h2),
.docs-markdown :deep(h3),
.docs-markdown :deep(h4),
.docs-markdown :deep(h5),
.docs-markdown :deep(h6) {
  scroll-margin-top: 7rem;
  color: rgb(15 23 42);
}

.docs-markdown :deep(p),
.docs-markdown :deep(li),
.docs-markdown :deep(blockquote),
.docs-markdown :deep(td),
.docs-markdown :deep(th),
.docs-markdown :deep(strong),
.docs-markdown :deep(em) {
  color: rgb(17 24 39);
}

.docs-markdown :deep(a) {
  color: rgb(15 23 42);
  text-decoration: underline;
  text-underline-offset: 0.2em;
}

.docs-markdown :deep(pre) {
  overflow-x: auto;
  border-radius: 1rem;
  border: 1px solid rgb(226 232 240);
  background: rgb(248 250 252);
  color: rgb(15 23 42);
  padding: 1rem 1.25rem;
}

.docs-markdown :deep(pre code) {
  background: transparent;
  color: rgb(15 23 42);
  padding: 0;
}

.docs-markdown :deep(code) {
  border-radius: 0.5rem;
  background: rgb(241 245 249);
  color: rgb(15 23 42);
  padding: 0.18rem 0.45rem;
}

.docs-markdown :deep(blockquote) {
  border-left: 4px solid rgb(148 163 184);
  background: rgb(248 250 252);
  padding: 0.9rem 1rem;
}

.docs-markdown :deep(table) {
  display: table;
  width: 100%;
  min-width: 100%;
  table-layout: auto;
}

.docs-markdown :deep(th) {
  font-weight: 700;
}
</style>
