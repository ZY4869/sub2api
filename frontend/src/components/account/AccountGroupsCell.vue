<template>
  <div v-if="groups && groups.length > 0" :class="rootClass">
    <!-- 分组容器：固定最大宽度，最多显示2行 -->
    <div :class="containerClass">
      <template v-if="displayMode === 'icon'">
        <span
          v-for="group in displayGroups"
          :key="group.id"
          class="group relative inline-flex"
        >
          <button
            type="button"
            :class="iconBadgeClass"
            :title="group.name"
            :aria-label="group.name"
          >
            <span aria-hidden="true">{{ groupInitial(group.name) }}</span>
          </button>
          <span :class="iconTooltipClass" role="tooltip">
            {{ group.name }}
          </span>
        </span>
      </template>
      <template v-else>
        <GroupBadge
          v-for="group in displayGroups"
          :key="group.id"
          :name="group.name"
          :platform="group.platform"
          :subscription-type="group.subscription_type"
          :rate-multiplier="group.rate_multiplier"
          :show-rate="false"
          :visual-variant="visualVariant"
          wrap
        />
      </template>
      <!-- 更多数量徽章 -->
      <button
        v-if="hiddenCount > 0"
        ref="moreButtonRef"
        @click.stop="showPopover = !showPopover"
        :class="moreButtonClass"
      >
        <span>+{{ hiddenCount }}</span>
      </button>
    </div>

    <!-- Popover 显示完整列表 -->
    <Teleport to="body">
      <Transition
        enter-active-class="transition duration-150 ease-out"
        enter-from-class="opacity-0 scale-95"
        enter-to-class="opacity-100 scale-100"
        leave-active-class="transition duration-100 ease-in"
        leave-from-class="opacity-100 scale-100"
        leave-to-class="opacity-0 scale-95"
      >
        <div
          v-if="showPopover"
          ref="popoverRef"
          class="fixed z-50 min-w-48 max-w-96 rounded-lg border border-gray-200 bg-white p-3 shadow-lg dark:border-dark-600 dark:bg-dark-800"
          :style="popoverStyle"
        >
          <div class="mb-2 flex items-center justify-between">
            <span class="text-xs font-medium text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.groupCountTotal', { count: groups.length }) }}
            </span>
            <button
              @click="showPopover = false"
              class="rounded p-0.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700 dark:hover:text-gray-300"
            >
              <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          <div class="flex flex-wrap gap-1.5 max-h-64 overflow-y-auto">
            <GroupBadge
              v-for="group in groups"
              :key="group.id"
              :name="group.name"
              :platform="group.platform"
              :subscription-type="group.subscription_type"
              :rate-multiplier="group.rate_multiplier"
              :show-rate="false"
              :visual-variant="visualVariant"
            />
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- 点击外部关闭 popover -->
    <div
      v-if="showPopover"
      class="fixed inset-0 z-40"
      @click="showPopover = false"
    />
  </div>
  <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import GroupBadge from '@/components/common/GroupBadge.vue'
import type { AccountGroupDisplayMode, Group } from '@/types'

interface Props {
  groups: Group[] | null | undefined
  maxDisplay?: number
  visualVariant?: 'default' | 'airy'
  displayMode?: AccountGroupDisplayMode
}

const props = withDefaults(defineProps<Props>(), {
  maxDisplay: 4,
  visualVariant: 'default',
  displayMode: 'full'
})

const { t } = useI18n()

const moreButtonRef = ref<HTMLElement | null>(null)
const popoverRef = ref<HTMLElement | null>(null)
const showPopover = ref(false)
const displayMode = computed(() => props.displayMode === 'icon' ? 'icon' : 'full')

// 显示的分组（最多显示 maxDisplay 个）
const displayGroups = computed(() => {
  if (!props.groups) return []
  if (props.groups.length <= props.maxDisplay) {
    return props.groups
  }
  // 留一个位置给 +N 按钮
  return props.groups.slice(0, props.maxDisplay - 1)
})

// 隐藏的数量
const hiddenCount = computed(() => {
  if (!props.groups) return 0
  if (props.groups.length <= props.maxDisplay) return 0
  return props.groups.length - (props.maxDisplay - 1)
})

const moreButtonClass = computed(() => {
  if (props.visualVariant === 'airy') {
    return 'inline-flex cursor-pointer items-center gap-0.5 whitespace-nowrap rounded border border-slate-200 bg-slate-100 px-1.5 py-[2.5px] text-[9px] font-extrabold text-slate-700 transition-colors hover:bg-slate-200 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700'
  }
  return 'inline-flex cursor-pointer items-center gap-0.5 whitespace-nowrap rounded-md bg-gray-100 px-1.5 py-0.5 text-xs font-medium text-gray-600 transition-colors hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500'
})

const rootClass = computed(() =>
  displayMode.value === 'icon'
    ? 'relative max-w-[104px]'
    : 'relative max-w-full'
)

const containerClass = computed(() =>
  displayMode.value === 'icon'
    ? 'flex max-h-14 flex-wrap gap-1 overflow-visible'
    : 'flex max-h-none flex-wrap gap-1 overflow-visible'
)

const iconBadgeClass = computed(() => {
  const base = 'inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full border text-[10px] font-black uppercase leading-none transition focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-1'
  if (props.visualVariant === 'airy') {
    return `${base} border-slate-200 bg-slate-100 text-slate-700 hover:bg-slate-200 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700`
  }
  return `${base} border-gray-200 bg-gray-100 text-gray-700 hover:bg-gray-200 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200 dark:hover:bg-dark-600`
})

const iconTooltipClass = computed(() => {
  const chrome = props.visualVariant === 'airy'
    ? 'border-slate-200 bg-white text-slate-700 shadow-lg dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100'
    : 'border-gray-200 bg-white text-gray-700 shadow-lg dark:border-dark-600 dark:bg-dark-800 dark:text-gray-100'
  return [
    'pointer-events-none absolute left-1/2 top-full z-30 mt-1 max-w-52 -translate-x-1/2 whitespace-normal break-words rounded-md border px-2 py-1 text-xs font-medium opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100',
    chrome,
  ].join(' ')
})

const groupInitial = (name: string) => {
  const trimmed = String(name || '').trim()
  return trimmed ? Array.from(trimmed)[0] : '#'
}

// Popover 位置样式
const popoverStyle = computed(() => {
  if (!moreButtonRef.value) return {}
  const rect = moreButtonRef.value.getBoundingClientRect()
  const viewportHeight = window.innerHeight
  const viewportWidth = window.innerWidth

  let top = rect.bottom + 8
  let left = rect.left

  // 如果下方空间不足，显示在上方
  if (top + 280 > viewportHeight) {
    top = Math.max(8, rect.top - 280)
  }

  // 如果右侧空间不足，向左偏移
  if (left + 384 > viewportWidth) {
    left = Math.max(8, viewportWidth - 392)
  }

  return {
    top: `${top}px`,
    left: `${left}px`
  }
})

// 关闭 popover 的键盘事件
const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    showPopover.value = false
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>
