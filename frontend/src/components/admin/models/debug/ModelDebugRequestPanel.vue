<template>
  <section class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
    <div class="flex items-start justify-between gap-3">
      <div>
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">
          {{ t("admin.models.pages.debug.requestTitle") }}
        </h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t("admin.models.pages.debug.requestDescription") }}
        </p>
      </div>
      <div class="flex items-center gap-2">
        <button
          v-if="running"
          type="button"
          class="btn btn-secondary"
          data-testid="debug-cancel"
          @click="emit('cancel')"
        >
          {{ t("admin.models.pages.debug.cancel") }}
        </button>
        <button
          v-else
          type="button"
          class="btn btn-primary"
          :disabled="!canRun"
          data-testid="debug-run"
          @click="emit('run')"
        >
          {{ t("admin.models.pages.debug.run") }}
        </button>
      </div>
    </div>

    <div class="mt-5 space-y-5">
      <div class="grid gap-3 md:grid-cols-3">
        <button
          v-for="protocol in protocols"
          :key="protocol.id"
          type="button"
          class="rounded-2xl border px-4 py-3 text-left transition"
          :class="form.protocol === protocol.id ? activeCardClass : inactiveCardClass"
          :data-testid="`debug-protocol-${protocol.id}`"
          @click="updateField('protocol', protocol.id)"
        >
          <div class="text-sm font-semibold">{{ protocol.label }}</div>
          <p class="mt-1 text-xs opacity-80">{{ protocol.description }}</p>
        </button>
      </div>

      <div class="grid gap-4 xl:grid-cols-2">
        <label class="space-y-1.5">
          <span class="input-label">{{ t("admin.models.pages.debug.keyModeLabel") }}</span>
          <select
            :value="form.keyMode"
            class="input"
            data-testid="debug-key-mode"
            @change="updateField('keyMode', ($event.target as HTMLSelectElement).value as ModelDebugEditorState['keyMode'])"
          >
            <option value="saved">{{ t("admin.models.pages.debug.keyModes.saved") }}</option>
            <option value="manual">{{ t("admin.models.pages.debug.keyModes.manual") }}</option>
          </select>
        </label>

        <label class="space-y-1.5">
          <span class="input-label">{{ t("admin.models.pages.debug.endpointLabel") }}</span>
          <select
            :value="form.endpointKind"
            class="input"
            data-testid="debug-endpoint-select"
            @change="updateField('endpointKind', ($event.target as HTMLSelectElement).value as ModelDebugEditorState['endpointKind'])"
          >
            <option v-for="option in endpointOptions" :key="option.id" :value="option.id">
              {{ option.label }}
            </option>
          </select>
        </label>

        <label v-if="form.keyMode === 'saved'" class="space-y-1.5">
          <span class="input-label">{{ t("admin.models.pages.debug.savedKeyLabel") }}</span>
          <select
            :value="form.apiKeyID ?? ''"
            class="input"
            data-testid="debug-key-select"
            @change="updateNumericField('apiKeyID', ($event.target as HTMLSelectElement).value)"
          >
            <option value="">{{ t("admin.models.pages.debug.savedKeyPlaceholder") }}</option>
            <option v-for="option in keyOptions" :key="option.id" :value="option.id">
              {{ option.label }}
            </option>
          </select>
        </label>

        <label v-else class="space-y-1.5">
          <span class="input-label">{{ t("admin.models.pages.debug.manualKeyLabel") }}</span>
          <input
            :value="form.manualAPIKey"
            type="password"
            class="input"
            data-testid="debug-manual-api-key"
            :placeholder="t('admin.models.pages.debug.manualKeyPlaceholder')"
            @input="updateField('manualAPIKey', ($event.target as HTMLInputElement).value)"
          />
        </label>

        <label class="space-y-1.5 xl:col-span-2">
          <span class="input-label">{{ t("admin.models.pages.debug.modelLabel") }}</span>
          <select
            :value="form.model"
            class="input"
            data-testid="debug-model-select"
            @change="updateField('model', ($event.target as HTMLSelectElement).value)"
          >
            <option value="">{{ t("admin.models.pages.debug.modelPlaceholder") }}</option>
            <option v-for="option in modelOptions" :key="option.id" :value="option.id">
              {{ option.label }}
            </option>
          </select>
        </label>
      </div>

      <div class="grid gap-4 xl:grid-cols-[minmax(0,1.4fr)_minmax(0,0.6fr)]">
        <div class="space-y-4">
          <TextArea
            :model-value="form.systemPrompt"
            :label="t('admin.models.pages.debug.systemPromptLabel')"
            :placeholder="t('admin.models.pages.debug.systemPromptPlaceholder')"
            :rows="3"
            @update:model-value="updateField('systemPrompt', $event)"
          />
          <TextArea
            :model-value="form.userPrompt"
            :label="t('admin.models.pages.debug.userPromptLabel')"
            :placeholder="t('admin.models.pages.debug.userPromptPlaceholder')"
            :rows="4"
            data-testid="debug-user-prompt"
            @update:model-value="updateField('userPrompt', $event)"
          />
        </div>

        <div class="space-y-4 rounded-2xl border border-gray-200 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-900/60">
          <label class="space-y-1.5">
            <span class="input-label">{{ t("admin.models.pages.debug.temperatureLabel") }}</span>
            <input
              :value="form.temperature"
              type="text"
              class="input"
              :placeholder="t('admin.models.pages.debug.temperaturePlaceholder')"
              @input="updateField('temperature', ($event.target as HTMLInputElement).value)"
            />
          </label>
          <label class="space-y-1.5">
            <span class="input-label">{{ t("admin.models.pages.debug.maxTokensLabel") }}</span>
            <input
              :value="form.maxOutputTokens"
              type="text"
              class="input"
              :placeholder="t('admin.models.pages.debug.maxTokensPlaceholder')"
              @input="updateField('maxOutputTokens', ($event.target as HTMLInputElement).value)"
            />
          </label>
          <label v-if="form.protocol === 'openai'" class="space-y-1.5">
            <span class="input-label">{{ t("admin.models.pages.debug.reasoningLabel") }}</span>
            <input
              :value="form.reasoningEffort"
              type="text"
              class="input"
              :placeholder="t('admin.models.pages.debug.reasoningPlaceholder')"
              @input="updateField('reasoningEffort', ($event.target as HTMLInputElement).value)"
            />
          </label>
          <label class="flex items-center gap-3 rounded-2xl border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-800">
            <input
              :checked="form.stream"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              data-testid="debug-stream-toggle"
              @change="updateField('stream', ($event.target as HTMLInputElement).checked)"
            />
            <span class="text-sm text-gray-700 dark:text-gray-200">
              {{ t("admin.models.pages.debug.streamLabel") }}
            </span>
          </label>
        </div>
      </div>

      <div class="grid gap-4 xl:grid-cols-2">
        <TextArea
          :model-value="form.advancedJSON"
          :label="t('admin.models.pages.debug.advancedJsonLabel')"
          :placeholder="t('admin.models.pages.debug.advancedJsonPlaceholder')"
          :rows="10"
          :error="advancedJsonError"
          data-testid="debug-advanced-json"
          @update:model-value="updateField('advancedJSON', $event)"
        />
        <div class="rounded-2xl border border-gray-200 bg-gray-950 p-4 dark:border-dark-700">
          <div class="mb-3 flex items-center justify-between gap-3">
            <span class="text-sm font-semibold text-white">
              {{ t("admin.models.pages.debug.requestPreviewLabel") }}
            </span>
            <span class="rounded-full bg-white/10 px-2.5 py-1 text-xs text-gray-300">
              JSON
            </span>
          </div>
          <pre
            class="max-h-[360px] overflow-auto whitespace-pre-wrap break-words font-mono text-xs text-emerald-300"
            data-testid="debug-request-preview"
          >{{ requestBodyPreview }}</pre>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import TextArea from "@/components/common/TextArea.vue";
import type { ModelDebugEditorState } from "@/utils/modelDebug";
import { useI18n } from "vue-i18n";

interface SelectOption {
  id: string | number;
  label: string;
}

const props = defineProps<{
  form: ModelDebugEditorState;
  canRun: boolean;
  running: boolean;
  keyOptions: SelectOption[];
  modelOptions: SelectOption[];
  endpointOptions: SelectOption[];
  requestBodyPreview: string;
  advancedJsonError: string;
}>();

const emit = defineEmits<{
  "update:form": [value: ModelDebugEditorState];
  run: [];
  cancel: [];
}>();

const { t } = useI18n();

const activeCardClass =
  "border-primary-400 bg-primary-50 text-primary-900 shadow-sm dark:border-primary-500/60 dark:bg-primary-500/10 dark:text-primary-100";
const inactiveCardClass =
  "border-gray-200 bg-gray-50 text-gray-700 hover:border-primary-300 dark:border-dark-700 dark:bg-dark-900/60 dark:text-gray-200";

const protocols: Array<{
  id: ModelDebugEditorState["protocol"];
  label: string;
  description: string;
}> = [
  {
    id: "openai",
    label: "OpenAI",
    description: t("admin.models.pages.debug.protocolHints.openai"),
  },
  {
    id: "anthropic",
    label: "Anthropic",
    description: t("admin.models.pages.debug.protocolHints.anthropic"),
  },
  {
    id: "gemini",
    label: "Gemini",
    description: t("admin.models.pages.debug.protocolHints.gemini"),
  },
];

function updateField<Key extends keyof ModelDebugEditorState>(
  key: Key,
  value: ModelDebugEditorState[Key],
) {
  emit("update:form", {
    ...props.form,
    [key]: value,
  });
}

function updateNumericField(key: "apiKeyID", value: string) {
  const parsed = Number(value);
  updateField(key, Number.isFinite(parsed) && parsed > 0 ? parsed : null);
}
</script>
