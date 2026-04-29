<template>
  <section class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
    <div class="flex items-start justify-between gap-3">
      <div>
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">
          {{ t("admin.models.pages.debug.outputTitle") }}
        </h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ running ? t("admin.models.pages.debug.outputRunning") : t("admin.models.pages.debug.outputIdle") }}
        </p>
      </div>
      <span
        class="rounded-full px-3 py-1 text-xs font-semibold"
        :class="running ? 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-200' : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-200'"
      >
        {{ running ? t("admin.models.pages.debug.running") : t("admin.models.pages.debug.ready") }}
      </span>
    </div>

    <div class="mt-5 rounded-[1.5rem] border border-gray-800 bg-gray-950 p-4">
      <div
        v-if="events.length === 0"
        class="rounded-2xl border border-dashed border-gray-700 px-4 py-10 text-center text-sm text-gray-400"
      >
        {{ t("admin.models.pages.debug.outputEmpty") }}
      </div>

      <div v-else class="space-y-3" data-testid="debug-output-events">
        <article
          v-for="event in events"
          :key="event.id"
          class="rounded-2xl border px-4 py-3"
          :class="eventClass(event.tone)"
        >
          <div class="flex items-center justify-between gap-3">
            <div class="text-xs font-semibold uppercase tracking-[0.24em]">
              {{ event.title }}
            </div>
            <span class="rounded-full bg-white/10 px-2 py-0.5 text-[11px]">
              {{ event.type }}
            </span>
          </div>
          <pre class="mt-3 whitespace-pre-wrap break-words font-mono text-xs leading-6">{{ event.body }}</pre>
        </article>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { ModelDebugOutputEvent } from "@/utils/modelDebug";
import { useI18n } from "vue-i18n";

defineProps<{
  events: ModelDebugOutputEvent[];
  running: boolean;
}>();

const { t } = useI18n();

function eventClass(tone: ModelDebugOutputEvent["tone"]) {
  switch (tone) {
    case "success":
      return "border-emerald-500/30 bg-emerald-500/10 text-emerald-100";
    case "warning":
      return "border-amber-500/30 bg-amber-500/10 text-amber-100";
    case "error":
      return "border-rose-500/30 bg-rose-500/10 text-rose-100";
    default:
      return "border-sky-500/20 bg-sky-500/10 text-sky-100";
  }
}
</script>
