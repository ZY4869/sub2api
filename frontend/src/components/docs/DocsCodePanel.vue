<template>
  <div class="overflow-x-auto">
    <pre class="docs-code-pre"><code class="docs-code-block">
      <span
        v-for="(line, index) in highlightedLines"
        :key="`${language}-${index + 1}`"
        class="docs-code-line"
        :class="{
          'docs-code-line-empty': line.isEmpty,
          'docs-code-line-focus': focusLineSet.has(index + 1),
        }"
      >
        <span class="docs-code-line-number" aria-hidden="true">{{ index + 1 }}</span>
        <span class="docs-code-line-content" v-html="line.html"></span>
      </span>
    </code></pre>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { highlightCode } from '@/utils/codeHighlight'

const props = defineProps<{
  code: string
  focusLines?: number[]
  language: string
}>()

const highlightedLines = computed(() => highlightCode(props.code, props.language))
const focusLineSet = computed(() => new Set(props.focusLines ?? []))
</script>

<style scoped>
.docs-code-pre {
  margin: 0;
  min-width: 100%;
  padding: 1.1rem 1.25rem 1.3rem;
}

.docs-code-block {
  display: block;
  font-family: 'JetBrains Mono', 'Cascadia Code', Consolas, monospace;
  font-size: 0.86rem;
  line-height: 1.8;
}

.docs-code-line {
  display: grid;
  grid-template-columns: 2.5rem minmax(0, 1fr);
  align-items: stretch;
  border-left: 3px solid transparent;
  border-radius: 0.9rem;
  color: rgb(226 232 240);
}

.docs-code-line + .docs-code-line {
  margin-top: 0.15rem;
}

.docs-code-line-focus {
  border-left-color: rgb(14 165 233);
  background: linear-gradient(90deg, rgba(14, 165, 233, 0.18), rgba(14, 165, 233, 0.05));
  box-shadow: inset 0 0 0 1px rgba(56, 189, 248, 0.08);
}

.docs-code-line-number {
  padding: 0 0.75rem 0 0.35rem;
  color: rgba(148, 163, 184, 0.9);
  text-align: right;
  user-select: none;
}

.docs-code-line-content {
  display: block;
  min-width: 0;
  overflow-wrap: anywhere;
  padding-right: 0.25rem;
}

.docs-code-line-empty .docs-code-line-content {
  min-height: 1.5rem;
}

.docs-code-line-content :deep(.docs-code-token) {
  font-weight: 500;
}

.docs-code-line-content :deep(.docs-code-token-comment) {
  color: rgb(148 163 184);
}

.docs-code-line-content :deep(.docs-code-token-string) {
  color: rgb(250 204 21);
}

.docs-code-line-content :deep(.docs-code-token-string-value) {
  color: rgb(253 224 71);
}

.docs-code-line-content :deep(.docs-code-token-url) {
  color: rgb(56 189 248);
}

.docs-code-line-content :deep(.docs-code-token-number) {
  color: rgb(196 181 253);
}

.docs-code-line-content :deep(.docs-code-token-keyword) {
  color: rgb(251 113 133);
}

.docs-code-line-content :deep(.docs-code-token-env) {
  color: rgb(74 222 128);
}

.docs-code-line-content :deep(.docs-code-token-method) {
  color: rgb(34 211 238);
}

.docs-code-line-content :deep(.docs-code-token-flag) {
  color: rgb(163 230 53);
}

.docs-code-line-content :deep(.docs-code-token-header) {
  color: rgb(251 146 60);
}

.docs-code-line-content :deep(.docs-code-token-json-key),
.docs-code-line-content :deep(.docs-code-token-property) {
  color: rgb(45 212 191);
}

.docs-code-line-content :deep(.docs-code-token-path) {
  color: rgb(125 211 252);
}

.docs-code-line-content :deep(.docs-code-token-function) {
  color: rgb(244 114 182);
}

.docs-code-line-focus .docs-code-line-content :deep(.docs-code-token-comment) {
  color: rgb(203 213 225);
}

.docs-code-line-focus .docs-code-line-content :deep(.docs-code-token-string),
.docs-code-line-focus .docs-code-line-content :deep(.docs-code-token-string-value) {
  color: rgb(254 240 138);
}

.docs-code-line-focus .docs-code-line-content :deep(.docs-code-token-url),
.docs-code-line-focus .docs-code-line-content :deep(.docs-code-token-method),
.docs-code-line-focus .docs-code-line-content :deep(.docs-code-token-path) {
  color: rgb(125 211 252);
}

.docs-code-line-focus .docs-code-line-content :deep(.docs-code-token-keyword),
.docs-code-line-focus .docs-code-line-content :deep(.docs-code-token-function) {
  color: rgb(253 164 175);
}

.docs-code-line-focus .docs-code-line-content :deep(.docs-code-token-env),
.docs-code-line-focus .docs-code-line-content :deep(.docs-code-token-flag) {
  color: rgb(190 242 100);
}

.docs-code-line-focus .docs-code-line-content :deep(.docs-code-token-header),
.docs-code-line-focus .docs-code-line-content :deep(.docs-code-token-json-key),
.docs-code-line-focus .docs-code-line-content :deep(.docs-code-token-property) {
  color: rgb(110 231 183);
}
</style>
