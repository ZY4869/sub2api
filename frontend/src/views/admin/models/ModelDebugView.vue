<template>
  <div class="space-y-6">
    <section class="overflow-hidden rounded-[2rem] border border-gray-200 bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.16),_transparent_36%),linear-gradient(135deg,_rgba(255,255,255,0.98),_rgba(240,249,255,0.92))] p-6 shadow-sm dark:border-dark-700 dark:bg-[radial-gradient(circle_at_top_left,_rgba(59,130,246,0.14),_transparent_36%),linear-gradient(135deg,_rgba(15,23,42,0.96),_rgba(17,24,39,0.92))]">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div class="max-w-3xl">
          <p class="text-xs font-semibold uppercase tracking-[0.24em] text-sky-700 dark:text-sky-300">
            {{ t("admin.models.pages.debug.eyebrow") }}
          </p>
          <h1 class="mt-3 text-3xl font-semibold tracking-tight text-gray-950 dark:text-white">
            {{ t("admin.models.pages.debug.title") }}
          </h1>
          <p class="mt-3 text-sm leading-7 text-gray-600 dark:text-gray-300">
            {{ t("admin.models.pages.debug.description") }}
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <span class="rounded-full border border-gray-200 bg-white/90 px-4 py-2 text-sm text-gray-700 dark:border-dark-700 dark:bg-dark-900/80 dark:text-gray-200">
            {{ t("admin.models.pages.debug.summary.models", { count: modelOptions.length }) }}
          </span>
          <span class="rounded-full border border-gray-200 bg-white/90 px-4 py-2 text-sm text-gray-700 dark:border-dark-700 dark:bg-dark-900/80 dark:text-gray-200">
            {{ t("admin.models.pages.debug.summary.keys", { count: savedKeyOptions.length }) }}
          </span>
        </div>
      </div>
    </section>

    <div class="grid gap-6 xl:grid-cols-[minmax(0,1.12fr)_minmax(0,0.88fr)]">
      <ModelDebugRequestPanel
        :form="form"
        :can-run="canRun"
        :running="running"
        :key-options="savedKeyOptions"
        :model-options="modelOptions"
        :endpoint-options="endpointOptions"
        :request-body-preview="requestBodyPreview"
        :advanced-json-error="advancedJSONErrorMessage"
        @update:form="updateForm"
        @run="runDebug"
        @cancel="cancelDebug"
      />
      <ModelDebugOutputPanel :events="outputEvents" :running="running" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { storeToRefs } from "pinia";
import { useI18n } from "vue-i18n";
import {
  runModelDebugStream,
  type AdminModelDebugEndpointKind,
  type AdminModelDebugStreamEvent,
} from "@/api/admin/modelDebug";
import keysAPI from "@/api/keys";
import userGroupsAPI from "@/api/groups";
import ModelDebugOutputPanel from "@/components/admin/models/debug/ModelDebugOutputPanel.vue";
import ModelDebugRequestPanel from "@/components/admin/models/debug/ModelDebugRequestPanel.vue";
import { useAppStore } from "@/stores/app";
import { usePublicModelCatalogStore } from "@/stores/publicModelCatalog";
import type { ApiKey, UserGroupModelOptionGroup } from "@/types";
import {
  buildBaseModelDebugRequestBody,
  defaultModelDebugEndpoint,
  filterModelDebugCatalogItems,
  mergeModelDebugRequestBody,
  MODEL_DEBUG_ENDPOINTS,
  type ModelDebugEditorState,
  type ModelDebugOutputEvent,
} from "@/utils/modelDebug";
import { buildPublicModelCatalogDisplayItem } from "@/utils/publicModelCatalog";

const { t } = useI18n();
const appStore = useAppStore();
const catalogStore = usePublicModelCatalogStore();
const { snapshot } = storeToRefs(catalogStore);

const form = ref<ModelDebugEditorState>({
  keyMode: "saved",
  apiKeyID: null,
  manualAPIKey: "",
  protocol: "openai",
  endpointKind: "responses",
  model: "",
  stream: true,
  systemPrompt: "You are a concise diagnostic assistant.",
  userPrompt: "Return a short confirmation message with the active model name.",
  temperature: "0.2",
  maxOutputTokens: "256",
  reasoningEffort: "medium",
  advancedJSON: "",
});

const savedKeys = ref<ApiKey[]>([]);
const groupOptions = ref<UserGroupModelOptionGroup[]>([]);
const outputEvents = ref<ModelDebugOutputEvent[]>([]);
const running = ref(false);
let activeAbortController: AbortController | null = null;

const endpointOptions = computed(() =>
  MODEL_DEBUG_ENDPOINTS[form.value.protocol].map((item) => ({
    id: item,
    label: endpointLabel(item),
  })),
);

const savedKeyOptions = computed(() =>
  savedKeys.value.map((item) => ({
    id: item.id,
    label: `${item.name} · #${item.id}`,
  })),
);

const selectedSavedKey = computed(
  () => savedKeys.value.find((item) => item.id === form.value.apiKeyID) || null,
);

const visibleCatalogItems = computed(() =>
  filterModelDebugCatalogItems(
    snapshot.value?.items || [],
    form.value.keyMode,
    selectedSavedKey.value,
    groupOptions.value,
  ),
);

const modelOptions = computed(() =>
  visibleCatalogItems.value.map((item) => {
    const displayItem = buildPublicModelCatalogDisplayItem(item);
    return {
      id: item.model,
      label: displayItem.subtitle
        ? `${displayItem.title} · ${displayItem.subtitle}`
        : displayItem.title,
    };
  }),
);

const mergedRequestBody = computed(() =>
  mergeModelDebugRequestBody(
    buildBaseModelDebugRequestBody(form.value),
    form.value.advancedJSON,
  ),
);

const requestBodyPreview = computed(() =>
  JSON.stringify(mergedRequestBody.value.body, null, 2),
);

const advancedJSONErrorMessage = computed(() => {
  if (!mergedRequestBody.value.error) {
    return "";
  }
  return mergedRequestBody.value.error === "advanced_json_must_be_object"
    ? t("admin.models.pages.debug.advancedJsonObjectError")
    : t("admin.models.pages.debug.advancedJsonInvalidError");
});

const canRun = computed(() => {
  if (!form.value.model.trim() || running.value || mergedRequestBody.value.error) {
    return false;
  }
  if (form.value.keyMode === "saved") {
    return Boolean(form.value.apiKeyID);
  }
  return Boolean(form.value.manualAPIKey.trim());
});

watch(
  () => form.value.protocol,
  (protocol) => {
    if (!MODEL_DEBUG_ENDPOINTS[protocol].includes(form.value.endpointKind)) {
      updateForm({
        ...form.value,
        endpointKind: defaultModelDebugEndpoint(protocol),
      });
    }
  },
);

watch(
  [() => form.value.keyMode, selectedSavedKey, modelOptions],
  () => {
    const next = { ...form.value };
    let changed = false;
    if (next.keyMode === "saved" && !next.apiKeyID && savedKeyOptions.value[0]) {
      const nextAPIKeyID = Number(savedKeyOptions.value[0].id);
      if (nextAPIKeyID !== next.apiKeyID) {
        next.apiKeyID = nextAPIKeyID;
        changed = true;
      }
    }
    if (!modelOptions.value.some((item) => item.id === next.model)) {
      const fallbackModel = String(modelOptions.value[0]?.id || "");
      if (fallbackModel !== next.model) {
        next.model = fallbackModel;
        changed = true;
      }
    }
    if (changed) {
      updateForm(next);
    }
  },
  { immediate: true },
);

onMounted(() => {
  void Promise.allSettled([catalogStore.initialize(), loadDebugContext()]);
});

onBeforeUnmount(() => {
  cancelDebug();
});

function updateForm(nextForm: ModelDebugEditorState) {
  form.value = {
    ...nextForm,
    protocol: nextForm.protocol,
    endpointKind: normalizeEndpoint(nextForm.protocol, nextForm.endpointKind),
  };
}

async function loadDebugContext() {
  try {
    const [keysResponse, groupsResponse] = await Promise.all([
      keysAPI.list(1, 1000),
      userGroupsAPI.getModelOptions(),
    ]);
    savedKeys.value = keysResponse.items || [];
    groupOptions.value = groupsResponse || [];
  } catch (error) {
    savedKeys.value = [];
    groupOptions.value = [];
    appStore.showError(t("admin.models.pages.debug.contextLoadFailed"));
  }
}

async function runDebug() {
  if (!canRun.value) {
    return;
  }

  outputEvents.value = [];
  running.value = true;
  activeAbortController?.abort();
  activeAbortController = new AbortController();

  try {
    await runModelDebugStream(
      {
        key_mode: form.value.keyMode,
        api_key_id: form.value.keyMode === "saved" ? form.value.apiKeyID || undefined : undefined,
        manual_api_key: form.value.keyMode === "manual" ? form.value.manualAPIKey.trim() : undefined,
        protocol: form.value.protocol,
        endpoint_kind: form.value.endpointKind,
        model: form.value.model.trim(),
        stream: form.value.stream,
        request_body: mergedRequestBody.value.body,
      },
      {
        signal: activeAbortController.signal,
        onEvent: handleStreamEvent,
      },
    );
  } catch (error) {
    if (isAbortError(error)) {
      pushEvent("final", t("admin.models.pages.debug.cancelled"), "", "warning");
    } else {
      const message = resolveErrorMessage(error, t("admin.models.pages.debug.runFailed"));
      pushEvent("error", t("admin.models.pages.debug.events.error"), message, "error");
    }
  } finally {
    running.value = false;
    activeAbortController = null;
  }
}

function cancelDebug() {
  activeAbortController?.abort();
  activeAbortController = null;
  running.value = false;
}

function handleStreamEvent(event: AdminModelDebugStreamEvent) {
  switch (event.type) {
    case "start":
      pushEvent(event.type, t("admin.models.pages.debug.events.start"), formatPayload(event), "info");
      break;
    case "request_preview":
      pushEvent(event.type, t("admin.models.pages.debug.events.request"), formatPayload(event), "info");
      break;
    case "response_headers":
      pushEvent(
        event.type,
        t("admin.models.pages.debug.events.headers"),
        formatPayload(event),
        Number(event.status_code || 0) >= 400 ? "warning" : "info",
      );
      break;
    case "content":
      pushEvent(event.type, t("admin.models.pages.debug.events.content"), String(event.chunk || event.raw || ""), "success");
      break;
    case "final":
      pushEvent(
        event.type,
        t("admin.models.pages.debug.events.final"),
        formatPayload(event),
        Number(event.status_code || 0) >= 400 ? "warning" : "success",
      );
      break;
    case "error":
      pushEvent(event.type, t("admin.models.pages.debug.events.error"), formatPayload(event), "error");
      break;
  }
}

function pushEvent(
  type: string,
  title: string,
  body: string,
  tone: ModelDebugOutputEvent["tone"],
) {
  outputEvents.value.push({
    id: `${type}-${outputEvents.value.length + 1}`,
    type,
    title,
    body,
    tone,
  });
}

function endpointLabel(endpoint: AdminModelDebugEndpointKind) {
  switch (endpoint) {
    case "chat_completions":
      return "Chat Completions";
    case "generate_content":
      return "Generate Content";
    case "messages":
      return "Messages";
    default:
      return "Responses";
  }
}

function normalizeEndpoint(
  protocol: ModelDebugEditorState["protocol"],
  endpoint: AdminModelDebugEndpointKind,
) {
  return MODEL_DEBUG_ENDPOINTS[protocol].includes(endpoint)
    ? endpoint
    : defaultModelDebugEndpoint(protocol);
}

function formatPayload(payload: Record<string, any>) {
  return JSON.stringify(payload, null, 2);
}

function resolveErrorMessage(error: unknown, fallback: string) {
  if (
    typeof error === "object" &&
    error &&
    "message" in error &&
    typeof (error as { message?: unknown }).message === "string"
  ) {
    return String((error as { message: string }).message);
  }
  return fallback;
}

function isAbortError(error: unknown) {
  return Boolean(
    error &&
      typeof error === "object" &&
      "name" in error &&
      String((error as { name?: unknown }).name || "").trim() === "AbortError",
  );
}
</script>
