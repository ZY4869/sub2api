<template>
  <span
    class="inline-flex shrink-0 items-center justify-center"
    :style="{ width: sizePx, height: sizePx }"
    :title="label"
    :aria-label="label"
    role="img"
  >
    <svg viewBox="0 0 120 120" width="100%" height="100%">
      <defs>
        <linearGradient :id="gradientId" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" :stop-color="palette[0]" />
          <stop offset="50%" :stop-color="palette[1]" />
          <stop offset="100%" :stop-color="palette[2]" />
        </linearGradient>
        <filter :id="filterId" x="-20%" y="-20%" width="140%" height="140%">
          <feGaussianBlur stdDeviation="3" result="blur" />
          <feComposite in="SourceGraphic" in2="blur" operator="over" />
        </filter>
      </defs>

      <circle
        cx="60"
        cy="60"
        r="50"
        fill="none"
        :stroke="`url(#${gradientId})`"
        stroke-width="2"
        stroke-dasharray="8 16"
        stroke-linecap="round"
        opacity="0.7"
      >
        <animateTransform
          attributeName="transform"
          type="rotate"
          :from="rotateFrom"
          :to="rotateTo"
          dur="20s"
          repeatCount="indefinite"
        />
      </circle>
      <circle cx="60" cy="60" r="42" fill="none" :stroke="`url(#${gradientId})`" stroke-width="1" opacity="0.2" />

      <path
        v-if="status === 'ok'"
        d="M 38 62 L 52 76 L 82 44"
        fill="none"
        :stroke="`url(#${gradientId})`"
        stroke-width="8"
        stroke-linecap="round"
        stroke-linejoin="round"
        :filter="`url(#${filterId})`"
      />

      <path
        v-else-if="status === 'error'"
        d="M 40 40 L 80 80 M 80 40 L 40 80"
        fill="none"
        :stroke="`url(#${gradientId})`"
        stroke-width="8"
        stroke-linecap="round"
        stroke-linejoin="round"
        :filter="`url(#${filterId})`"
      />

      <g v-else-if="status === 'maintenance'" :filter="`url(#${filterId})`">
        <g :stroke="`url(#${gradientId})`" fill="none">
          <animateTransform attributeName="transform" type="rotate" from="0 60 60" to="360 60 60" dur="15s" repeatCount="indefinite" />
          <circle cx="60" cy="60" r="26" stroke-width="5" stroke-dasharray="10 10.4" stroke-linecap="round" />
          <circle cx="60" cy="60" r="21" stroke-width="3" />
        </g>
        <path d="M 60 46 L 60 60 M 60 72 L 60 73" fill="none" :stroke="`url(#${gradientId})`" stroke-width="6" stroke-linecap="round" />
      </g>

      <g v-else-if="status === 'warning'" :filter="`url(#${filterId})`">
        <path d="M 60 30 L 85 75 L 35 75 Z" fill="none" :stroke="`url(#${gradientId})`" stroke-width="6" stroke-linecap="round" stroke-linejoin="round" />
        <path d="M 60 45 L 60 58 M 60 66 L 60 68" fill="none" :stroke="`url(#${gradientId})`" stroke-width="6" stroke-linecap="round" />
      </g>

      <g v-else :filter="`url(#${filterId})`">
        <circle cx="60" cy="60" r="22" fill="none" :stroke="`url(#${gradientId})`" stroke-width="6" stroke-linecap="round" />
        <path d="M 60 48 L 60 50 M 60 56 L 60 70 M 56 70 L 64 70" fill="none" :stroke="`url(#${gradientId})`" stroke-width="5" stroke-linecap="round" />
      </g>
    </svg>
  </span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { PublicModelCatalogStatus } from "@/api/meta";

const props = withDefaults(defineProps<{
  status?: PublicModelCatalogStatus;
  label: string;
  size?: number | string;
}>(), {
  status: "info",
  size: 30,
});

const uid = `public-model-status-${Math.random().toString(36).slice(2, 10)}`;

const sizePx = computed(() =>
  typeof props.size === "number" ? `${props.size}px` : props.size,
);
const gradientId = `${uid}-gradient`;
const filterId = `${uid}-filter`;

const palette = computed(() => {
  switch (props.status) {
    case "ok":
      return ["#10b981", "#0ea5e9", "#8b5cf6"];
    case "error":
      return ["#f43f5e", "#f97316", "#f59e0b"];
    case "maintenance":
      return ["#3b82f6", "#8b5cf6", "#f59e0b"];
    case "warning":
      return ["#fcd34d", "#f59e0b", "#ef4444"];
    default:
      return ["#22d3ee", "#3b82f6", "#6366f1"];
  }
});

const rotateFrom = computed(() =>
  props.status === "error" || props.status === "info" ? "360 60 60" : "0 60 60",
);
const rotateTo = computed(() =>
  props.status === "error" || props.status === "info" ? "0 60 60" : "360 60 60",
);
</script>
