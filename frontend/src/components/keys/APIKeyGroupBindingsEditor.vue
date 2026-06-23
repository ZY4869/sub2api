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
          <ApiKeyGroupSelect
            :model-value="binding.group_id"
            :groups="sortedGroups"
            :disabled-group-ids="selectedGroupIdsForOtherBindings(index)"
            :user-group-rates="userGroupRates"
            @update:model-value="(groupId) => onGroupChange(index, groupId)"
          />
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
            v-if="!modelSelectionRequired"
            type="button"
            class="text-xs text-primary-600 transition-colors hover:text-primary-500 dark:text-primary-400"
            @click="clearModelSelection(index)"
          >
            {{ t("keys.modelScopeAll") }}
          </button>
        </div>

        <div
          v-if="groupModelOptionsLoading && bindingView(binding).models.length === 0"
          class="rounded-xl border border-dashed border-gray-200 px-3 py-4 text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400"
        >
          {{ t("keys.modelScopeLoading") }}
        </div>

        <div
          v-else-if="bindingView(binding).models.length"
          class="space-y-2"
        >
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{
              bindingView(binding).selectedCount > 0
                ? t("keys.modelScopeSelected", { count: bindingView(binding).selectedCount })
                : modelSelectionRequired
                  ? t("keys.modelScopeRequiredHint")
                : imageOnly
                  ? t("keys.modelScopeAllImageHint")
                  : t("keys.modelScopeAllHint")
            }}
          </p>

          <p
            v-if="bindingView(binding).unmatchedPatterns.length"
            class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200"
          >
            {{ t("keys.modelScopeLegacyHint") }}
          </p>

          <p
            v-if="modelSelectionRequired && bindingView(binding).selectedCount === 0"
            class="rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-xs text-rose-700 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-200"
          >
            {{ t("keys.modelSelectionRequired") }}
          </p>

          <label
            v-if="bindingView(binding).hasUnavailableModels"
            class="inline-flex items-center gap-2 text-xs font-medium text-gray-600 dark:text-gray-300"
          >
            <input
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-500"
              :checked="showUnavailableModels"
              @change="emit('update:showUnavailableModels', ($event.target as HTMLInputElement).checked)"
            />
            {{ t("keys.modelScopeShowUnavailable") }}
          </label>

          <div class="max-h-48 overflow-y-auto rounded-xl border border-gray-200 bg-white p-2 dark:border-dark-600 dark:bg-dark-900">
            <label
              v-for="modelView in bindingView(binding).models"
              :key="`${binding.group_id}-${modelView.model.public_id}`"
              class="flex items-start gap-3 rounded-lg px-2 py-2 transition"
              :class="modelView.selectionDisabled ? 'cursor-not-allowed opacity-60' : 'cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-800'"
            >
              <input
                type="checkbox"
                class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-500"
                :checked="modelView.selected"
                :disabled="modelView.selectionDisabled"
                @change="toggleModelSelection(index, modelView.model.public_id)"
              />
              <ModelIcon
                :model="modelView.model.public_id"
                :provider="modelView.catalogItem?.provider"
                :display-name="modelView.displayName"
                size="18px"
              />
              <div class="min-w-0 flex-1">
                <div class="truncate text-sm font-medium text-gray-900 dark:text-white">
                  {{ modelView.displayName }}
                </div>
                <div class="truncate text-xs text-gray-500 dark:text-gray-400">
                  {{ modelView.model.public_id }}
                </div>
                <div
                  v-if="modelView.priceSummary"
                  class="mt-1 text-xs text-emerald-700 dark:text-emerald-300"
                >
                  {{ modelView.priceSummary }}
                </div>
                <div
                  v-if="modelView.unavailableReasonLabel"
                  class="mt-1 text-xs text-amber-700 dark:text-amber-300"
                >
                  {{ modelView.unavailableReasonLabel }}
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
import type { PublicModelCatalogItem } from "@/api/meta";
import ModelIcon from "@/components/common/ModelIcon.vue";
import type { UserGroupModelOption } from "@/types";
import ApiKeyGroupSelect from "./ApiKeyGroupSelect.vue";
import type {
  BindableGroup,
  EditableApiKeyGroupBinding,
} from "./apiKeyGroupBindings";
import {
  formatCatalogPrice,
  priceEntryLabel,
} from "@/utils/publicModelCatalog";
import {
  createEmptyEditableBinding,
  parseModelPatterns,
} from "./apiKeyGroupBindings";

const props = withDefaults(
  defineProps<{
    modelValue: EditableApiKeyGroupBinding[];
    groups: BindableGroup[];
    groupModelOptions?: Record<number, UserGroupModelOption[]>;
    groupModelCatalogItems?: Record<number, PublicModelCatalogItem[]>;
    groupModelOptionsLoading?: boolean;
    adminMode?: boolean;
    imageOnly?: boolean;
    modelSelectionRequired?: boolean;
    showUnavailableModels?: boolean;
    userGroupRates?: Record<number, number>;
  }>(),
  {
    adminMode: false,
    imageOnly: false,
    modelSelectionRequired: false,
    groupModelCatalogItems: () => ({}),
    groupModelOptions: () => ({}),
    groupModelOptionsLoading: false,
    showUnavailableModels: false,
    userGroupRates: () => ({}),
  },
);

const emit = defineEmits<{
  (e: "update:modelValue", value: EditableApiKeyGroupBinding[]): void;
  (e: "update:showUnavailableModels", value: boolean): void;
}>();

const { t } = useI18n();

interface ModelOptionView {
  model: UserGroupModelOption;
  catalogItem?: PublicModelCatalogItem;
  displayName: string;
  selected: boolean;
  unavailable: boolean;
  selectionDisabled: boolean;
  unavailableReasonLabel: string;
  priceSummary: string;
}

interface BindingView {
  models: ModelOptionView[];
  selectedCount: number;
  unmatchedPatterns: string[];
  hasUnavailableModels: boolean;
}

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

const selectedGroupIdsForOtherBindings = (currentIndex: number): number[] => {
  return rows.value
    .filter((binding, index) => index !== currentIndex && binding.group_id > 0)
    .map((binding) => binding.group_id);
};

const onGroupChange = (index: number, groupId: number) => {
  updateRow(index, {
    group_id: groupId || 0,
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

const buildBindingView = (binding: EditableApiKeyGroupBinding): BindingView => {
  const models = props.groupModelOptions?.[binding.group_id] || [];
  const catalogByModel = new Map(
    catalogItemsForBinding(binding).map((item) => [item.model, item]),
  );
  const selectedPatterns = effectiveModelPatterns(binding);
  const selectedSet = new Set(selectedPatterns);
  const visibleModels: ModelOptionView[] = [];
  let hasUnavailableModels = false;

  for (const model of models) {
    const item = catalogByModel.get(model.public_id);
    const unavailable = item?.key_availability === "unavailable";
    hasUnavailableModels = hasUnavailableModels || unavailable;
    if (!props.showUnavailableModels && unavailable) {
      continue;
    }
    if (props.imageOnly && !unavailable && !isImageModelOption(model, item)) {
      continue;
    }
    const unavailableReason = item?.unavailable_reason || "";
    visibleModels.push({
      model,
      catalogItem: item,
      displayName: item?.display_name || model.display_name || model.public_id,
      selected: selectedSet.has(model.public_id),
      unavailable,
      selectionDisabled: unavailable && unavailableReason !== "not_selected_by_key",
      unavailableReasonLabel: unavailableReason
        ? t(`keys.modelUnavailableReasons.${unavailableReason}`)
        : "",
      priceSummary: modelPriceSummary(item),
    });
  }

  const available = new Set(visibleModels.map((item) => item.model.public_id));
  const unmatchedPatterns = selectedPatterns.filter(
    (pattern) => !available.has(pattern),
  );
  const selectedCount = selectedPatterns.filter((pattern) =>
    available.has(pattern),
  ).length;
  return {
    models: visibleModels,
    selectedCount,
    unmatchedPatterns,
    hasUnavailableModels,
  };
};

const bindingViews = computed(() => {
  const views = new Map<EditableApiKeyGroupBinding, BindingView>();
  rows.value.forEach((binding) => {
    views.set(binding, buildBindingView(binding));
  });
  return views;
});

const bindingView = (binding: EditableApiKeyGroupBinding): BindingView =>
  bindingViews.value.get(binding) ?? buildBindingView(binding);

const catalogItemsForBinding = (
  binding: EditableApiKeyGroupBinding,
): PublicModelCatalogItem[] => {
  return props.groupModelCatalogItems?.[binding.group_id] || [];
};

const isImageModelOption = (
  model: UserGroupModelOption,
  catalogItem?: PublicModelCatalogItem,
): boolean => {
  if (catalogItem?.mode === "image") {
    return true;
  }
  const protocols = catalogItem?.request_protocols || model.request_protocols || [];
  return protocols.some((protocol) => String(protocol).toLowerCase().includes("image"));
};

const modelPriceSummary = (item?: PublicModelCatalogItem): string => {
  if (!item || !item.currency || !item.price_display?.primary?.length) {
    return "";
  }
  return item.price_display.primary
    .map((entry) => `${priceEntryLabel(t, entry.id)} ${formatCatalogPrice(t, entry, item.currency, null)}`)
    .join(" · ");
};

const effectiveModelPatterns = (
  binding: EditableApiKeyGroupBinding,
): string[] => {
  return binding.model_selection_dirty
    ? binding.selected_models
    : parseModelPatterns(binding.model_patterns_text);
};

const toggleModelSelection = (index: number, modelID: string) => {
  const binding = rows.value[index];
  if (!binding) {
    return;
  }
  const view = bindingView(binding);
  const target = view.models.find((item) => item.model.public_id === modelID);
  if (!target || target.selectionDisabled) {
    return;
  }
  const available = new Set(view.models.map((item) => item.model.public_id));
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
