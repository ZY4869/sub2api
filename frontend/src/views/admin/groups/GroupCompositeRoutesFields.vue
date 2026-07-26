<template>
  <div v-if="form.platform === 'composite'" class="border-t border-gray-200 pt-4 dark:border-dark-400">
    <div class="mb-4 flex items-start justify-between gap-4">
      <div>
        <label class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.groups.compositeRoutes.title') }}
        </label>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.groups.compositeRoutes.hint') }}
        </p>
      </div>
      <button
        type="button"
        class="flex shrink-0 items-center gap-1.5 text-sm text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
        @click="$emit('addRoute')"
      >
        <Icon name="plus" size="sm" />
        {{ t('admin.groups.compositeRoutes.addRoute') }}
      </button>
    </div>

    <div class="mb-4 flex flex-col gap-2 rounded-lg border border-gray-200 p-3 dark:border-dark-600 sm:flex-row sm:items-end">
      <div class="min-w-0 flex-1">
        <label class="input-label text-xs">
          {{ t('admin.groups.compositeRoutes.previewModel') }}
        </label>
        <input
          :value="previewModel"
          type="text"
          class="input text-sm"
          :placeholder="t('admin.groups.compositeRoutes.displayModelPlaceholder')"
          @input="$emit('updatePreviewModel', ($event.target as HTMLInputElement).value)"
        />
      </div>
      <button
        type="button"
        class="btn btn-secondary shrink-0"
        :disabled="previewLoading || !previewModel.trim()"
        @click="$emit('previewRoute')"
      >
        {{ t('admin.groups.compositeRoutes.preview') }}
      </button>
    </div>

    <div
      v-if="previewResult"
      class="mb-4 rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm text-gray-700 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300"
    >
      <span v-if="previewResult.matched">
        {{
          t('admin.groups.compositeRoutes.previewMatched', {
            group: previewResult.target_group?.name || `#${previewResult.target_group_id || ''}`,
            model: previewResult.target_model_id || previewResult.display_model_id || previewModel
          })
        }}
      </span>
      <span v-else>{{ t('admin.groups.compositeRoutes.previewNoMatch') }}</span>
    </div>

    <p
      v-if="form.composite_routes.length === 0"
      class="rounded-lg border border-dashed border-gray-300 p-4 text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400"
    >
      {{ t('admin.groups.compositeRoutes.emptyRoutes') }}
    </p>

    <div v-else class="space-y-3">
      <div
        v-for="route in form.composite_routes"
        :key="getRouteKey(route)"
        class="rounded-lg border border-gray-200 p-3 dark:border-dark-600"
      >
        <div class="grid grid-cols-1 gap-3 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)_minmax(0,1fr)_7rem_auto] lg:items-end">
          <div>
            <label class="input-label text-xs">
              {{ t('admin.groups.compositeRoutes.displayModel') }}
            </label>
            <input
              v-model="route.display_model_id"
              type="text"
              class="input text-sm"
              :placeholder="t('admin.groups.compositeRoutes.displayModelPlaceholder')"
            />
          </div>

          <div>
            <label class="input-label text-xs">
              {{ t('admin.groups.compositeRoutes.targetGroup') }}
            </label>
            <Select
              v-model="route.target_group_id"
              :options="targetGroupOptions"
              :placeholder="t('admin.groups.compositeRoutes.targetGroupPlaceholder')"
              searchable
            >
              <template #selected="{ option }">
                <GroupBadge
                  v-if="isGroupSelectOption(option) && selectOption(option).platform"
                  :name="selectOption(option).name"
                  :platform="selectOption(option).platform"
                  :show-rate="false"
                />
                <span v-else class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.compositeRoutes.targetGroupPlaceholder') }}
                </span>
              </template>
              <template #option="{ option, selected }">
                <GroupOptionItem
                  v-if="isGroupSelectOption(option)"
                  :name="selectOption(option).name"
                  :platform="selectOption(option).platform"
                  :description="selectOption(option).description"
                  :selected="selected"
                />
              </template>
            </Select>
          </div>

          <div>
            <label class="input-label text-xs">
              {{ t('admin.groups.compositeRoutes.targetModel') }}
            </label>
            <input
              v-model="route.target_model_id"
              type="text"
              class="input text-sm"
              :placeholder="t('admin.groups.compositeRoutes.targetModelPlaceholder')"
            />
          </div>

          <div>
            <label class="input-label text-xs">
              {{ t('admin.groups.compositeRoutes.priority') }}
            </label>
            <input
              v-model.number="route.priority"
              type="number"
              min="1"
              class="input text-sm"
            />
          </div>

          <div class="flex items-center gap-3 lg:pb-2">
            <Toggle v-model="route.enabled" />
            <button
              type="button"
              class="rounded-lg p-2 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/20"
              :title="t('admin.groups.compositeRoutes.removeRoute')"
              @click="$emit('removeRoute', route)"
            >
              <Icon name="trash" size="sm" />
            </button>
          </div>
        </div>

        <div class="mt-3">
          <label class="input-label text-xs">
            {{ t('admin.groups.compositeRoutes.notes') }}
          </label>
          <input
            v-model="route.notes"
            type="text"
            class="input text-sm"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import Select from '@/components/common/Select.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  form: Record<string, any>
  t: (key: string, params?: Record<string, unknown>) => string
  targetGroupOptions: Array<Record<string, unknown>>
  previewModel: string
  previewLoading: boolean
  previewResult: Record<string, any> | null
  isGroupSelectOption: (option: unknown) => boolean
  getRouteKey: (route: Record<string, any>) => string
}>()

defineEmits<{
  (e: 'addRoute'): void
  (e: 'removeRoute', route: Record<string, any>): void
  (e: 'updatePreviewModel', value: string): void
  (e: 'previewRoute'): void
}>()

const selectOption = (option: unknown) => (option ?? {}) as Record<string, any>
</script>
