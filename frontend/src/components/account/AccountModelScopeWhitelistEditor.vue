<template>
  <div class="space-y-3">
    <p class="text-xs text-gray-500 dark:text-gray-400">
      {{ t("admin.accounts.selectAllowedModels") }}
    </p>

    <div class="flex flex-wrap items-center gap-3">
      <input
        v-model="searchQuery"
        type="text"
        class="input min-w-[220px] flex-1"
        :placeholder="t('admin.accounts.searchModels')"
      />

      <label
        class="flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 text-xs text-gray-600 dark:border-dark-600 dark:text-gray-300"
      >
        <input v-model="showAllModels" type="checkbox" class="h-3.5 w-3.5" />
        <span>{{ t("admin.accounts.showAllModels") }}</span>
      </label>

      <button
        type="button"
        class="btn btn-secondary"
        @click="emit('update:allowedModels', [])"
      >
        {{ t("admin.accounts.clearAllModels") }}
      </button>
    </div>

    <p
      v-if="viewMode === 'default'"
      class="text-xs text-gray-500 dark:text-gray-400"
    >
      {{ t("admin.accounts.modelScopeDefaultVisibleHint") }}
    </p>

    <div
      v-if="providerGroups.length === 0"
      class="rounded-lg border border-dashed border-gray-300 p-4 text-sm text-gray-500 dark:border-dark-500 dark:text-gray-400"
    >
      {{ t("admin.accounts.noMatchingModels") }}
    </div>

    <div
      v-for="group in providerGroups"
      :key="group.provider"
      class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800"
    >
      <div class="mb-3 flex flex-wrap items-start justify-between gap-3">
        <div>
          <div class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ group.label }}
          </div>
          <div class="text-xs text-gray-500 dark:text-gray-400">
            {{ group.selectedCount }}/{{ group.entries.length }}
            {{ t("common.selectedCount", { count: group.selectedCount }) }}
            <span
              v-if="viewMode === 'search' && group.truncated"
              class="ml-2 text-amber-600 dark:text-amber-400"
            >
              {{ t("admin.accounts.searchResultsTruncated") }}
            </span>
          </div>
        </div>
        <div class="flex gap-2">
          <button
            type="button"
            class="rounded-lg border border-gray-200 px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-700"
            :disabled="!providerAllModelIds.has(group.provider)"
            @click="selectProvider(group.provider)"
          >
            {{ t("common.all") }}
          </button>
          <button
            type="button"
            class="rounded-lg border border-gray-200 px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-700"
            :disabled="!providerAllModelIds.has(group.provider)"
            @click="clearProvider(group.provider)"
          >
            {{ t("common.none") }}
          </button>
        </div>
      </div>

      <div class="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
        <button
          v-for="entry in group.entries"
          :key="entry.id"
          type="button"
          class="flex items-start gap-3 rounded-lg border px-3 py-3 text-left transition-colors"
          :class="
            selectedSet.has(entry.id)
              ? 'border-primary-500 bg-primary-50 dark:border-primary-500 dark:bg-primary-900/20'
              : 'border-gray-200 bg-white hover:border-gray-300 dark:border-dark-600 dark:bg-dark-700 dark:hover:border-dark-500'
          "
          @click="toggleModel(entry.id)"
        >
          <span
            class="mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded border"
            :class="
              selectedSet.has(entry.id)
                ? 'border-primary-500 bg-primary-500 text-white'
                : 'border-gray-300 dark:border-dark-500'
            "
          >
            <svg
              v-if="selectedSet.has(entry.id)"
              class="h-3 w-3"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="3"
                d="M5 13l4 4L19 7"
              />
            </svg>
          </span>
          <ModelIcon :model="entry.id" :provider="entry.provider" size="18px" />
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span
                class="truncate text-sm font-medium text-gray-900 dark:text-white"
                >{{ entry.display_name || entry.id }}</span
              >
              <span
                v-if="
                  entry.status &&
                  entry.status !== 'stable' &&
                  entry.status !== 'deprecated'
                "
                class="shrink-0 rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-medium uppercase text-amber-700 dark:bg-amber-900/30 dark:text-amber-300"
              >
                {{ entry.status }}
              </span>
            </div>
            <div class="truncate text-xs text-gray-500 dark:text-gray-400">
              {{ entry.id }}
            </div>
          </div>
        </button>
      </div>
    </div>

    <p class="text-xs text-gray-500 dark:text-gray-400">
      {{ t("admin.accounts.selectedModels", { count: allowedModels.length }) }}
      <span v-if="allowedModels.length === 0">
        {{ t("admin.accounts.noModelsSelected") }}
      </span>
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import ModelIcon from "@/components/common/ModelIcon.vue";
import { getModelRegistrySnapshot } from "@/stores/modelRegistry";
import {
  COMMON_MAX_PER_PROVIDER,
  MAX_RESULTS_PER_PROVIDER,
  getModelScopeWhitelistGroups,
} from "@/utils/accountModelScopeCandidates";

interface Props {
  platform: string;
  allowedModels: string[];
}

const props = defineProps<Props>();
const emit = defineEmits<{
  "update:allowedModels": [value: string[]];
}>();

const { t } = useI18n();
const searchQuery = ref("");
const showAllModels = ref(false);
const selectedSet = computed(() => new Set(props.allowedModels));

const whitelistState = computed(() => {
  const snapshot = getModelRegistrySnapshot();
  return getModelScopeWhitelistGroups(snapshot.models, {
    platform: props.platform,
    selectedModelIds: selectedSet.value,
    query: searchQuery.value,
    showAllModels: showAllModels.value,
    commonMaxPerProvider: COMMON_MAX_PER_PROVIDER,
    maxResultsPerProvider: MAX_RESULTS_PER_PROVIDER,
  });
});

const viewMode = computed(() => whitelistState.value.mode);
const providerGroups = computed(() => whitelistState.value.providerGroups);
const providerAllModelIds = computed(
  () => whitelistState.value.providerAllModelIds,
);

function toggleModel(modelId: string) {
  const next = new Set(props.allowedModels);
  next.has(modelId) ? next.delete(modelId) : next.add(modelId);
  emit("update:allowedModels", [...next].sort());
}

function selectProvider(provider: string) {
  const ids = providerAllModelIds.value.get(provider);
  if (!ids) return;
  const next = new Set(props.allowedModels);
  ids.forEach((id) => next.add(id));
  emit("update:allowedModels", [...next].sort());
}

function clearProvider(provider: string) {
  const ids = providerAllModelIds.value.get(provider);
  if (!ids) return;
  const removeSet = new Set(ids);
  emit(
    "update:allowedModels",
    props.allowedModels.filter((modelId) => !removeSet.has(modelId)),
  );
}
</script>
