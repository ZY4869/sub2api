<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t('admin.settings.customMenu.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.settings.customMenu.description') }}
      </p>
    </div>
    <div class="space-y-4 p-6">
      <div
        v-for="(item, index) in modelValue"
        :key="item.id || index"
        class="rounded-lg border border-gray-200 p-4 dark:border-dark-600"
      >
        <div class="mb-3 flex items-center justify-between">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.settings.customMenu.itemLabel', { n: index + 1 }) }}
          </span>
          <div class="flex items-center gap-2">
            <button
              v-if="index > 0"
              type="button"
              class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700"
              :title="t('admin.settings.customMenu.moveUp')"
              @click="moveItem(index, -1)"
            >
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5 15l7-7 7 7" /></svg>
            </button>
            <button
              v-if="index < modelValue.length - 1"
              type="button"
              class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700"
              :title="t('admin.settings.customMenu.moveDown')"
              @click="moveItem(index, 1)"
            >
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" /></svg>
            </button>
            <button
              type="button"
              class="rounded p-1 text-red-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
              :title="t('admin.settings.customMenu.remove')"
              @click="removeItem(index)"
            >
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
            </button>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Input
            v-model="item.label"
            :label="t('admin.settings.customMenu.name')"
            :placeholder="t('admin.settings.customMenu.namePlaceholder')"
          />

          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.settings.customMenu.visibility') }}
            </label>
            <select v-model="item.visibility" class="input text-sm">
              <option value="user">{{ t('admin.settings.customMenu.visibilityUser') }}</option>
              <option value="admin">{{ t('admin.settings.customMenu.visibilityAdmin') }}</option>
            </select>
          </div>

          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.settings.customMenu.mode') }}
            </label>
            <select v-model="item.page_mode" class="input text-sm" @change="normalizeMode(item)">
              <option value="iframe">{{ t('admin.settings.customMenu.modeIframe') }}</option>
              <option value="markdown">{{ t('admin.settings.customMenu.modeMarkdown') }}</option>
            </select>
          </div>

          <div v-if="item.page_mode === 'markdown'" class="flex items-center justify-between rounded-lg border border-gray-200 px-3 py-2 dark:border-dark-600">
            <div>
              <div class="text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.customMenu.published') }}
              </div>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.customMenu.publishedHint') }}
              </p>
            </div>
            <Toggle
              :model-value="item.page_published ?? false"
              @update:model-value="(value: boolean) => item.page_published = value"
            />
          </div>

          <div v-if="item.page_mode === 'iframe'" class="sm:col-span-2">
            <Input
              v-model="item.url"
              type="url"
              :label="t('admin.settings.customMenu.url')"
              :placeholder="t('admin.settings.customMenu.urlPlaceholder')"
            />
          </div>

          <template v-else>
            <Input
              :model-value="item.page_slug"
              :label="t('admin.settings.customMenu.slug')"
              :placeholder="t('admin.settings.customMenu.slugPlaceholder')"
              :hint="t('admin.settings.customMenu.slugHint')"
              @update:model-value="(value: string) => updateMarkdownSlug(item, value)"
            />

            <div class="sm:col-span-2">
              <TextArea
                v-model="item.page_content"
                :label="t('admin.settings.customMenu.content')"
                :placeholder="t('admin.settings.customMenu.contentPlaceholder')"
                :hint="t('admin.settings.customMenu.contentHint')"
                :rows="10"
              />
            </div>
          </template>

          <div class="sm:col-span-2">
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.settings.customMenu.iconSvg') }}
            </label>
            <ImageUpload
              :model-value="item.icon_svg"
              mode="svg"
              size="sm"
              :upload-label="t('admin.settings.customMenu.uploadSvg')"
              :remove-label="t('admin.settings.customMenu.removeSvg')"
              @update:model-value="(value: string) => item.icon_svg = value"
            />
          </div>
        </div>
      </div>

      <button
        type="button"
        class="flex w-full items-center justify-center gap-2 rounded-lg border-2 border-dashed border-gray-300 py-3 text-sm text-gray-500 transition-colors hover:border-primary-400 hover:text-primary-600 dark:border-dark-600 dark:text-gray-400 dark:hover:border-primary-500 dark:hover:text-primary-400"
        @click="addItem"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" /></svg>
        {{ t('admin.settings.customMenu.add') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CustomMenuItem } from '@/types'
import ImageUpload from '@/components/common/ImageUpload.vue'
import Input from '@/components/common/Input.vue'
import TextArea from '@/components/common/TextArea.vue'
import Toggle from '@/components/common/Toggle.vue'

const props = defineProps<{
  modelValue: CustomMenuItem[]
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: CustomMenuItem[]): void
}>()

const { t } = useI18n()

function normalizeMarkdownSlug(value: string) {
  const normalized = value.trim().toLowerCase()
  if (!normalized) {
    return ''
  }

  let output = ''
  let lastDash = false
  for (const char of normalized) {
    const isAlphaNum = /[a-z0-9]/.test(char)
    if (isAlphaNum) {
      output += char
      lastDash = false
      continue
    }
    if ((char === '-' || char === '_' || char === ' ' || char === '/') && output && !lastDash) {
      output += '-'
      lastDash = true
    }
  }

  return output.replace(/^-+|-+$/g, '').slice(0, 64)
}

function updateItems(items: CustomMenuItem[]) {
  items.forEach((item, index) => {
    item.sort_order = index
  })
  emit('update:modelValue', [...items])
}

function updateMarkdownSlug(item: CustomMenuItem, value: string) {
  item.page_slug = normalizeMarkdownSlug(value)
}

function normalizeMode(item: CustomMenuItem) {
  if (item.page_mode === 'markdown') {
    item.url = ''
    item.page_slug = normalizeMarkdownSlug(item.page_slug || '')
    item.page_published = item.page_published ?? true
    return
  }
  item.page_mode = 'iframe'
  item.page_slug = ''
  item.page_content = ''
  item.page_public = false
  item.page_published = false
}

function addItem() {
  updateItems([
    ...props.modelValue,
    {
      id: '',
      label: '',
      icon_svg: '',
      url: '',
      visibility: 'user',
      sort_order: props.modelValue.length,
      page_mode: 'iframe',
      page_slug: '',
      page_content: '',
      page_public: false,
      page_published: false,
    },
  ])
}

function removeItem(index: number) {
  const next = props.modelValue.slice()
  next.splice(index, 1)
  updateItems(next)
}

function moveItem(index: number, direction: -1 | 1) {
  const targetIndex = index + direction
  if (targetIndex < 0 || targetIndex >= props.modelValue.length) {
    return
  }
  const next = props.modelValue.slice()
  const current = next[index]
  next[index] = next[targetIndex]
  next[targetIndex] = current
  updateItems(next)
}

watch(
  () => props.modelValue,
  (items) => {
    items.forEach((item) => {
      if (!item.page_mode) {
        item.page_mode = item.page_slug ? 'markdown' : 'iframe'
      }
      if (item.page_mode === 'markdown') {
        item.page_slug = normalizeMarkdownSlug(item.page_slug || '')
        item.page_published = item.page_published ?? true
      }
    })
  },
  { immediate: true, deep: true },
)
</script>
