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
import type {
  BindableGroup,
  EditableApiKeyGroupBinding,
} from "./apiKeyGroupBindings";
import { createEmptyEditableBinding } from "./apiKeyGroupBindings";

const props = withDefaults(
  defineProps<{
    modelValue: EditableApiKeyGroupBinding[];
    groups: BindableGroup[];
    adminMode?: boolean;
  }>(),
  {
    adminMode: false,
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
  updateRow(index, { group_id: Number(target.value) || 0 });
};

const onQuotaInput = (index: number, event: Event) => {
  const target = event.target as HTMLInputElement;
  updateRow(index, { quota: target.value === "" ? "" : Number(target.value) });
};

const onModelPatternsInput = (index: number, event: Event) => {
  const target = event.target as HTMLTextAreaElement;
  updateRow(index, { model_patterns_text: target.value });
};
</script>
