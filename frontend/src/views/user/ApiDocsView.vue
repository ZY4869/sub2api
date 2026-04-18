<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="overflow-hidden rounded-[2rem] border border-slate-200 bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.14),_transparent_34%),linear-gradient(135deg,_rgba(255,255,255,0.98),_rgba(241,245,249,0.92))] p-6 shadow-sm dark:border-dark-700 dark:bg-[radial-gradient(circle_at_top_left,_rgba(56,189,248,0.14),_transparent_34%),linear-gradient(135deg,_rgba(17,24,39,0.96),_rgba(15,23,42,0.92))] md:p-8">
        <div class="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div class="max-w-3xl">
            <p class="text-xs font-semibold uppercase tracking-[0.28em] text-sky-600 dark:text-sky-300">
              {{ t('ui.apiDocs.eyebrow') }}
            </p>
            <h1 class="mt-3 text-3xl font-semibold tracking-tight text-slate-950 dark:text-white">
              {{ t('ui.apiDocs.title') }}
            </h1>
            <p class="mt-3 text-sm leading-7 text-slate-900 dark:text-slate-100 md:text-base">
              {{ t('ui.apiDocs.description') }}
            </p>
          </div>
          <button type="button" class="btn btn-primary" :disabled="loading || !content" @click="handleCopy">
            {{ t('ui.apiDocs.copy') }}
          </button>
        </div>
      </section>

      <div v-if="loading" class="rounded-3xl border border-slate-200 bg-white/90 px-6 py-16 text-center text-sm text-slate-800 dark:border-dark-700 dark:bg-dark-900/80 dark:text-slate-100">
        {{ t('ui.apiDocs.loading') }}
      </div>

      <div v-else-if="errorMessage" class="rounded-3xl border border-rose-200 bg-rose-50 px-6 py-6 text-sm text-rose-700 dark:border-rose-900/60 dark:bg-rose-950/30 dark:text-rose-200">
        {{ errorMessage }}
      </div>

      <DocsMarkdownContent
        v-else
        :markdown="content"
        :page-id="routePageId"
        base-path="/api-docs"
        :nav-title="t('ui.apiDocs.protocolsTitle')"
        :toc-title="t('ui.apiDocs.pageTocTitle')"
        :summary-cards="summaryCards"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import DocsMarkdownContent from '@/components/docs/DocsMarkdownContent.vue'
import docsAPI from '@/api/docs'
import { useClipboard } from '@/composables/useClipboard'
import { normalizeDocsPageId } from '@/utils/markdownDocs'

const { t } = useI18n()
const route = useRoute()
const { copyToClipboard } = useClipboard()

const loading = ref(true)
const content = ref('')
const errorMessage = ref('')

const routePageId = computed(() => normalizeDocsPageId(route.params.pageId as string | undefined))

const summaryCards = computed(() => [
  {
    label: t('ui.apiDocs.summary.protocols.label'),
    value: t('ui.apiDocs.summary.protocols.value'),
    description: t('ui.apiDocs.summary.protocols.description')
  },
  {
    label: t('ui.apiDocs.summary.auth.label'),
    value: t('ui.apiDocs.summary.auth.value'),
    description: t('ui.apiDocs.summary.auth.description')
  },
  {
    label: t('ui.apiDocs.summary.languages.label'),
    value: t('ui.apiDocs.summary.languages.value'),
    description: t('ui.apiDocs.summary.languages.description')
  }
])

async function loadDocument() {
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await docsAPI.getAPIDocs()
    content.value = response.content
  } catch (error: any) {
    errorMessage.value = error?.message || t('ui.apiDocs.loadFailed')
  } finally {
    loading.value = false
  }
}

async function handleCopy() {
  await copyToClipboard(content.value, t('ui.apiDocs.copySuccess'))
}

onMounted(() => {
  loadDocument().catch((error) => {
    console.error('Failed to load API docs:', error)
  })
})
</script>
