<template>
  <div class="min-h-screen bg-gray-50 px-4 py-6 dark:bg-dark-950 md:px-8">
    <div class="mx-auto max-w-5xl">
      <div v-if="loading" class="flex min-h-[50vh] items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <div
        v-else-if="errorMessage"
        class="rounded-xl border border-red-200 bg-red-50 p-6 text-sm text-red-700 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-400"
      >
        {{ errorMessage }}
      </div>

      <CustomMarkdownPageContent
        v-else
        :markdown="pageContent"
        :title="pageTitle"
        :toc-title="t('customPage.pageToc')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { pagesAPI } from '@/api'
import CustomMarkdownPageContent from '@/components/custom/CustomMarkdownPageContent.vue'

const route = useRoute()
const { t } = useI18n()

const loading = ref(false)
const errorMessage = ref('')
const pageTitle = ref('')
const pageContent = ref('')

const slug = computed(() => String(route.params.slug || '').trim())

async function loadPage() {
  if (!slug.value) {
    errorMessage.value = t('customPage.notFoundDesc')
    return
  }
  loading.value = true
  errorMessage.value = ''
  try {
    const page = await pagesAPI.getCustomPage(slug.value)
    pageTitle.value = page.label || slug.value
    pageContent.value = page.content || ''
  } catch (error: any) {
    errorMessage.value = error?.message || t('customPage.loadFailedDesc')
    pageTitle.value = ''
    pageContent.value = ''
  } finally {
    loading.value = false
  }
}

onMounted(loadPage)

watch(slug, () => {
  void loadPage()
})
</script>
