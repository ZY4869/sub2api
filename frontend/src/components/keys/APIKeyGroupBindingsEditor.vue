<template>
  <div class="space-y-2">
    <div
      v-for="(binding, index) in rows"
      :key="`${index}-${binding.group_id}`"
      class="rounded-xl border border-gray-200 bg-gray-50 p-2.5 dark:border-dark-500 dark:bg-dark-800"
    >
      <div
        :class="
          adminMode
            ? 'grid gap-2.5 lg:grid-cols-[minmax(0,1.5fr)_160px]'
            : 'grid gap-2.5'
        "
      >
        <div>
          <label class="input-label">{{
            adminMode ? t("admin.users.group") : t("keys.groupLabel")
          }}</label>
          <select
            class="input"
            :value="binding.group_id"
            @change="onGroupChange(index, $event)"
          >
            <option :value="0">{{ t("keys.selectGroup") }}</option>
            <option
              v-for="group in sortedGroups"
              :key="group.id"
              :value="group.id"
              :disabled="isGroupSelectedInOtherBinding(group.id, index)"
            >
              {{ group.name }} · {{ group.platform }} · P{{
                group.priority ?? 1
              }}
            </option>
          </select>
        </div>

        <div v-if="adminMode">
          <label class="input-label">{{ t("admin.users.groupQuota") }}</label>
          <input
            class="input"
            type="number"
            min="0"
            step="0.01"
            :value="binding.quota"
            :placeholder="t('admin.users.groupQuotaPlaceholder')"
            @input="onQuotaInput(index, $event)"
          />
        </div>
      </div>

      <div v-if="adminMode" class="mt-2">
        <label class="input-label">{{ t("admin.users.modelPatterns") }}</label>
        <textarea
          class="input"
          rows="2"
          :value="binding.model_patterns_text"
          placeholder="claude-opus-*"
          @input="onModelPatternsInput(index, $event)"
        ></textarea>
        <p class="input-hint">{{ t("admin.users.modelPatternsHint") }}</p>
      </div>

      <div v-else-if="binding.group_id > 0" class="mt-2 space-y-2">
        <div class="flex items-center justify-between gap-3">
          <label class="input-label mb-0">{{ t("keys.modelScopeLabel") }}</label>
          <button
            type="button"
            class="text-xs text-primary-600 transition-colors hover:text-primary-500 dark:text-primary-400"
            @click="clearModelSelection(index)"
          >
            {{ t("keys.modelScopeAll") }}
          </button>
        </div>

        <div
          v-if="groupModelOptionsLoading && modelsForBinding(binding).length === 0"
          class="rounded-xl border border-dashed border-gray-200 px-3 py-4 text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400"
        >
          {{ t("keys.modelScopeLoading") }}
        </div>

        <div
          v-else-if="modelsForBinding(binding).length"
          class="space-y-2"
        >
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{
              selectedModelCount(binding) > 0
                ? t("keys.modelScopeSelected", { count: selectedModelCount(binding) })
                : t("keys.modelScopeAllHint")
            }}
          </p>

          <p
            v-if="unmatchedModelPatterns(binding).length"
            class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200"
          >
            {{ t("keys.modelScopeLegacyHint") }}
          </p>

          <div class="max-h-48 overflow-y-auto rounded-xl border border-gray-200 bg-white p-2 dark:border-dark-600 dark:bg-dark-900">
            <label
              v-for="model in modelsForBinding(binding)"
              :key="`${binding.group_id}-${model.public_id}`"
              class="flex cursor-pointer items-start gap-3 rounded-lg px-2 py-2 transition hover:bg-gray-50 dark:hover:bg-dark-800"
            >
              <input
                type="checkbox"
                class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-500"
                :checked="isModelSelected(binding, model.public_id)"
                @change="toggleModelSelection(index, model.public_id)"
              />
              <div class="min-w-0">
                <div class="truncate text-sm font-medium text-gray-900 dark:text-white">
                  {{ model.display_name || model.public_id }}
                </div>
                <div class="truncate text-xs text-gray-500 dark:text-gray-400">
                  {{ model.public_id }}
                </div>
              </div>
            </label>
          </div>
        </div>

        <div
          v-else
          class="rounded-xl border border-dashed border-gray-200 px-3 py-4 text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400"
        >
          {{ t("keys.modelScopeEmpty") }}
        </div>
      </div>

      <div class="mt-2 flex justify-end">
        <button
          type="button"
          class="btn btn-secondary btn-sm"
          @click="removeRow(index)"
        >
          {{ t("admin.users.removeGroupBinding") }}
        </button>
      </div>
    </div>

    <div class="flex flex-wrap items-center justify-between gap-2">
      <div v-if="adminMode" class="text-xs text-gray-500 dark:text-gray-400">
        {{ t("admin.users.groupQuotaHint") }}
      </div>

      <button type="button" class="btn btn-secondary btn-sm" @click="addRow">
        {{ t("admin.users.addGroupBinding") }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { UserGroupModelOption } from "@/types";
import type {
  BindableGroup,
  EditableApiKeyGroupBinding,
} from "./apiKeyGroupBindings";
import {
  createEmptyEditableBinding,
  parseModelPatterns,
} from "./apiKeyGroupBindings";

const props = withDefaults(
  defineProps<{
    modelValue: EditableApiKeyGroupBinding[];
    groups: BindableGroup[];
    groupModelOptions?: Record<number, UserGroupModelOption[]>;
    groupModelOptionsLoading?: boolean;
    adminMode?: boolean;
  }>(),
  {
    adminMode: false,
    groupModelOptions: () => ({}),
    groupModelOptionsLoading: false,
  },
);

const emit = defineEmits<{
  (e: "update:modelValue", value: EditableApiKeyGroupBinding[]): void;
}>();

const { t } = useI18n();

const rows = computed(() => props.modelValue || []);
const sortedGroups = computed(() =>
  [...props.groups].sort((a, b) => {
    const priorityDiff = (a.priority ?? 1) - (b.priority ?? 1);
    if (priorityDiff !== 0) return priorityDiff;
    return a.name.localeCompare(b.name);
  }),
);

const updateRows = (rows: EditableApiKeyGroupBinding[]) => {
  emit("update:modelValue", rows);
};

const updateRow = (
  index: number,
  patch: Partial<EditableApiKeyGroupBinding>,
) => {
  updateRows(
    rows.value.map((item, currentIndex) =>
      currentIndex === index ? { ...item, ...patch } : item,
    ),
  );
};

const addRow = () => {
  updateRows([...rows.value, createEmptyEditableBinding()]);
};

const removeRow = (index: number) => {
  updateRows(rows.value.filter((_, currentIndex) => currentIndex !== index));
};

const isGroupSelectedInOtherBinding = (
  groupId: number,
  currentIndex: number,
): boolean => {
  return rows.value.some(
    (binding, index) => index !== currentIndex && binding.group_id === groupId,
  );
};

const onGroupChange = (index: number, event: Event) => {
  const target = event.target as HTMLSelectElement;
  updateRow(index, {
    group_id: Number(target.value) || 0,
    model_patterns_text: "",
    selected_models: [],
    model_selection_dirty: true,
  });
};

const onQuotaInput = (index: number, event: Event) => {
  const target = event.target as HTMLInputElement;
  updateRow(index, { quota: target.value === "" ? "" : Number(target.value) });
};

const onModelPatternsInput = (index: number, event: Event) => {
  const target = event.target as HTMLTextAreaElement;
  updateRow(index, { model_patterns_text: target.value });
};

const modelsForBinding = (
  binding: EditableApiKeyGroupBinding,
): UserGroupModelOption[] => {
  return props.groupModelOptions?.[binding.group_id] || [];
};

const effectiveModelPatterns = (
  binding: EditableApiKeyGroupBinding,
): string[] => {
  return binding.model_selection_dirty
    ? binding.selected_models
    : parseModelPatterns(binding.model_patterns_text);
};

const unmatchedModelPatterns = (
  binding: EditableApiKeyGroupBinding,
): string[] => {
  const available = new Set(modelsForBinding(binding).map((item) => item.public_id));
  return effectiveModelPatterns(binding).filter((pattern) => !available.has(pattern));
};

const selectedModelCount = (
  binding: EditableApiKeyGroupBinding,
): number => {
  const available = new Set(modelsForBinding(binding).map((item) => item.public_id));
  return effectiveModelPatterns(binding).filter((pattern) => available.has(pattern)).length;
};

const isModelSelected = (
  binding: EditableApiKeyGroupBinding,
  modelID: string,
): boolean => {
  return effectiveModelPatterns(binding).includes(modelID);
};

const toggleModelSelection = (index: number, modelID: string) => {
  const binding = rows.value[index];
  if (!binding) {
    return;
  }
  const available = new Set(modelsForBinding(binding).map((item) => item.public_id));
  const current = effectiveModelPatterns(binding).filter((pattern) => available.has(pattern));
  const next = current.includes(modelID)
    ? current.filter((item) => item !== modelID)
    : [...current, modelID];
  updateRow(index, {
    selected_models: next,
    model_selection_dirty: true,
  });
};

const clearModelSelection = (index: number) => {
  updateRow(index, {
    selected_models: [],
    model_selection_dirty: true,
  });
};
</script>
