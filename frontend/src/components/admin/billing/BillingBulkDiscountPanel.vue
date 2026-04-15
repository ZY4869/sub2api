<template>
  <div class="rounded-2xl border border-emerald-200 bg-emerald-50/80 p-4 dark:border-emerald-500/20 dark:bg-emerald-500/10">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h4 class="text-sm font-semibold text-emerald-900 dark:text-emerald-100">出售价格快捷操作</h4>
        <p class="mt-1 text-xs text-emerald-800 dark:text-emerald-200">可整单折扣，也可只对选中的价格项打折，并支持跨工作集模型应用。</p>
      </div>
      <button type="button" class="btn btn-secondary btn-sm" @click="emit('copy-official')">套用官方格式</button>
    </div>

    <div class="mt-4 grid gap-3 md:grid-cols-3">
      <label class="space-y-1 text-xs text-emerald-900 dark:text-emerald-100">
        <span>折扣比例</span>
        <input
          class="input"
          type="number"
          min="0.01"
          max="10"
          step="0.01"
          :value="discountRatio"
          @input="emit('update:discountRatio', Number(($event.target as HTMLInputElement).value))"
        />
      </label>
      <label class="space-y-1 text-xs text-emerald-900 dark:text-emerald-100">
        <span>应用范围</span>
        <select class="input" :value="scope" @change="emit('update:scope', ($event.target as HTMLSelectElement).value as 'current' | 'workset')">
          <option value="current">当前模型</option>
          <option value="workset">工作集全部模型</option>
        </select>
      </label>
      <div class="flex items-end gap-2">
        <button type="button" class="btn btn-primary btn-sm" @click="emit('apply-all')">整单折扣</button>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="selectedCount === 0" @click="emit('apply-selected')">
          仅选中项
        </button>
      </div>
    </div>
    <p class="mt-2 text-xs text-emerald-800 dark:text-emerald-200">已选价格项 {{ selectedCount }} 个</p>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  discountRatio: number
  scope: 'current' | 'workset'
  selectedCount: number
}>()

const emit = defineEmits<{
  (e: 'copy-official'): void
  (e: 'apply-all'): void
  (e: 'apply-selected'): void
  (e: 'update:discountRatio', value: number): void
  (e: 'update:scope', value: 'current' | 'workset'): void
}>()
</script>
