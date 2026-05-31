<template>
  <div class="card">
    <div class="px-6 py-4">
      <div class="flex flex-wrap items-end gap-4">
        <div class="min-w-[180px]">
          <label class="input-label">{{ t("usage.apiKeyFilter") }}</label>
          <Select
            :model-value="filters.api_key_id"
            :options="apiKeyOptions"
            :placeholder="t('usage.allApiKeys')"
            @update:model-value="updateFilter('api_key_id', $event)"
            @change="$emit('apply')"
          />
        </div>

        <div class="min-w-[180px]">
          <label class="input-label">{{ t("usage.platform") }}</label>
          <Select
            :model-value="filters.platform"
            :options="platformOptions"
            :placeholder="t('usage.allPlatforms')"
            @update:model-value="updateFilter('platform', $event)"
            @change="$emit('apply')"
          >
            <template #selected="{ option }">
              <span class="flex min-w-0 items-center gap-2">
                <PlatformIcon
                  v-if="option?.value"
                  :platform="String(option.value)"
                  size="sm"
                  class="shrink-0"
                />
                <span class="truncate">{{ option?.label || t("usage.allPlatforms") }}</span>
              </span>
            </template>
            <template #option="{ option }">
              <span class="flex min-w-0 items-center gap-2">
                <PlatformIcon
                  v-if="option.value"
                  :platform="String(option.value)"
                  size="sm"
                  class="shrink-0"
                />
                <span class="truncate">{{ option.label }}</span>
              </span>
            </template>
          </Select>
        </div>

        <div>
          <label class="input-label">{{ t("usage.timeRange") }}</label>
          <DateRangePicker
            :start-date="startDate"
            :end-date="endDate"
            @update:start-date="$emit('update:startDate', $event)"
            @update:end-date="$emit('update:endDate', $event)"
            @change="$emit('date-range-change', $event)"
          />
        </div>

        <div
          class="ml-auto flex flex-1 flex-wrap items-center justify-end gap-3"
          data-testid="usage-filter-toolbar-row"
        >
          <TokenDisplayModeToggle />
          <UsageModelDisplayModeToggle
            :model-value="usageModelDisplayMode"
            :disabled="updatingUsageModelDisplayMode"
            :label-text="t('usage.modelDisplay')"
            @update:modelValue="$emit('update-usage-model-display-mode', $event)"
          />
          <button @click="$emit('apply')" :disabled="loading" class="btn btn-secondary">
            {{ t("common.refresh") }}
          </button>
          <button @click="$emit('reset')" class="btn btn-secondary">
            {{ t("common.reset") }}
          </button>
          <button @click="$emit('export')" :disabled="exporting" class="btn btn-primary">
            <svg
              v-if="exporting"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              />
            </svg>
            {{ exporting ? t("usage.exporting") : t("usage.exportCsv") }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import Select from "@/components/common/Select.vue";
import DateRangePicker from "@/components/common/DateRangePicker.vue";
import TokenDisplayModeToggle from "@/components/common/TokenDisplayModeToggle.vue";
import UsageModelDisplayModeToggle from "@/components/common/UsageModelDisplayModeToggle.vue";
import PlatformIcon from "@/components/common/PlatformIcon.vue";
import type {
  UsageModelDisplayMode,
  UsageQueryParams,
} from "@/types";

const props = defineProps<{
  filters: UsageQueryParams;
  apiKeyOptions: Array<{ value: number | null; label: string }>;
  platformOptions: Array<{ value: string | null; label: string }>;
  startDate: string;
  endDate: string;
  loading: boolean;
  exporting: boolean;
  usageModelDisplayMode: UsageModelDisplayMode;
  updatingUsageModelDisplayMode: boolean;
}>();

const emit = defineEmits<{
  apply: [];
  reset: [];
  export: [];
  "update:filters": [filters: UsageQueryParams];
  "update:startDate": [value: string];
  "update:endDate": [value: string];
  "date-range-change": [range: { startDate: string; endDate: string; preset: string | null }];
  "update-usage-model-display-mode": [mode: UsageModelDisplayMode];
}>();

const { t } = useI18n();

function updateFilter(key: keyof UsageQueryParams, value: string | number | boolean | null) {
  emit("update:filters", {
    ...props.filters,
    [key]: value == null || value === "" ? undefined : value,
  });
}
</script>
