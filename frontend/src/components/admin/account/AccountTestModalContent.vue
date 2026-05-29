<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.testAccountConnection')"
    width="normal"
    @close="handleClose"
  >
    <div class="space-y-4">
      <!-- Account Info Card -->
      <div
        v-if="account"
        class="flex items-center justify-between rounded-xl border border-gray-200 bg-gradient-to-r from-gray-50 to-gray-100 p-3 dark:border-dark-500 dark:from-dark-700 dark:to-dark-600"
      >
        <div class="flex items-center gap-3">
          <div
            class="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-primary-500 to-primary-600"
          >
            <Icon name="play" size="md" class="text-white" :stroke-width="2" />
          </div>
          <div>
            <div class="font-semibold text-gray-900 dark:text-gray-100">{{ account.name }}</div>
            <div class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
              <span
                class="rounded bg-gray-200 px-1.5 py-0.5 text-[10px] font-medium uppercase dark:bg-dark-500"
              >
                {{ account.type }}
              </span>
              <span>{{ t('admin.accounts.account') }}</span>
            </div>
          </div>
        </div>
        <span
          :class="[
            'rounded-full px-2.5 py-1 text-xs font-semibold',
            account.status === 'active'
              ? 'bg-green-100 text-green-700 dark:bg-green-500/20 dark:text-green-400'
              : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
          ]"
        >
          {{ account.status }}
        </span>
      </div>

      <div v-if="supportsTestModes" class="space-y-1.5">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.testModeLabel') }}
        </label>
        <div class="grid gap-2 sm:grid-cols-2">
          <button
            v-for="option in testModeOptions"
            :key="option.value"
            :data-test="`test-mode-${option.value}`"
            type="button"
            :disabled="status === 'connecting'"
            :class="[
              'rounded-xl border px-4 py-3 text-left transition-all',
              selectedTestMode === option.value
                ? 'border-primary-500 bg-primary-50 text-primary-700 shadow-sm dark:border-primary-400 dark:bg-primary-500/10 dark:text-primary-200'
                : 'border-gray-200 bg-white text-gray-700 hover:border-primary-300 dark:border-dark-500 dark:bg-dark-700 dark:text-gray-200 dark:hover:border-primary-500/60',
              status === 'connecting' ? 'cursor-not-allowed opacity-70' : ''
            ]"
            @click="selectTestMode(option.value)"
          >
            <div class="text-sm font-semibold">
              {{ option.label }}
            </div>
            <p class="mt-1 text-xs leading-5 opacity-80">
              {{ option.description }}
            </p>
          </button>
        </div>
      </div>

      <div class="space-y-3">
        <div class="flex items-center justify-between gap-3">
          <div class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.accounts.selectTestModel') }}
          </div>
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            :disabled="loadingModels || status === 'connecting' || !account"
            @click="loadAvailableModels(true)"
          >
            <Icon
              v-if="loadingModels"
              name="refresh"
              size="sm"
              class="animate-spin"
              :stroke-width="2"
            />
            <Icon v-else name="refresh" size="sm" :stroke-width="2" />
            <span>{{ t('admin.accounts.refreshTestModels') }}</span>
          </button>
        </div>

        <AccountTestModelSelectionFields
          v-model:model-input-mode="modelInputMode"
          v-model:selected-model-key="selectedModelKey"
          v-model:manual-model-id="manualModelId"
          v-model:manual-source-protocol="manualSourceProtocol"
          :available-models="availableModels"
          :loading-models="loadingModels"
          :disabled="status === 'connecting'"
          :show-manual-source-protocol-field="isProtocolGatewayAccount"
        />

        <label v-if="modelInputMode === 'manual'" class="space-y-1.5">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.accounts.probeFinalize.manualRequestAlias') }}
          </span>
          <input
            v-model="manualRequestAlias"
            type="text"
            class="input"
            :disabled="status === 'connecting'"
            :placeholder="manualModelId || t('admin.accounts.probeFinalize.manualRequestAliasPlaceholder')"
          />
        </label>
      </div>
      <div
        v-if="isKiroAccount"
        class="rounded-lg border border-sky-200 bg-sky-50 px-3 py-2 text-xs text-sky-700 dark:border-sky-700 dark:bg-sky-900/20 dark:text-sky-300"
      >
        {{ t('admin.accounts.kiroTestModelSourceHint') }}
      </div>
      <div
        v-else-if="isGrokAccount"
        class="rounded-lg border border-violet-200 bg-violet-50 px-3 py-2 text-xs text-violet-700 dark:border-violet-700 dark:bg-violet-900/20 dark:text-violet-300"
      >
        {{ t(grokTestHintKey) }}
      </div>

      <div v-if="supportsPromptInput" class="space-y-1.5">
        <TextArea
          v-model="testPrompt"
          :label="promptInputLabel"
          :placeholder="promptInputPlaceholder"
          :hint="promptInputHint"
          :disabled="status === 'connecting'"
          rows="3"
        />
      </div>

      <!-- Terminal Output -->
      <div class="group relative">
        <div
          ref="terminalRef"
          class="max-h-[240px] min-h-[120px] overflow-y-auto rounded-xl border border-gray-700 bg-gray-900 p-4 font-mono text-sm dark:border-gray-800 dark:bg-black"
        >
          <!-- Status Line -->
          <div v-if="status === 'idle'" class="flex items-center gap-2 text-gray-500">
            <Icon name="play" size="sm" :stroke-width="2" />
            <span>{{ t('admin.accounts.readyToTest') }}</span>
          </div>
          <div v-else-if="status === 'connecting'" class="flex items-center gap-2 text-yellow-400">
            <Icon name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
            <span>{{ t('admin.accounts.connectingToApi') }}</span>
          </div>

          <!-- Output Lines -->
          <div v-for="(line, index) in outputLines" :key="index" :class="line.class">
            {{ line.text }}
          </div>

          <!-- Streaming Content -->
          <div v-if="streamingContent" class="text-green-400">
            {{ streamingContent }}<span class="animate-pulse">_</span>
          </div>

          <!-- Result Status -->
          <div
            v-if="status === 'success'"
            class="mt-3 flex items-center gap-2 border-t border-gray-700 pt-3 text-green-400"
          >
            <Icon name="check" size="sm" :stroke-width="2" />
            <span>{{ t('admin.accounts.testCompleted') }}</span>
          </div>
          <div
            v-else-if="status === 'error'"
            class="mt-3 flex items-center gap-2 border-t border-gray-700 pt-3 text-red-400"
          >
            <Icon name="x" size="sm" :stroke-width="2" />
            <span>{{ errorMessage }}</span>
          </div>
        </div>

        <!-- Copy Button -->
        <button
          v-if="outputLines.length > 0"
          @click="copyOutput"
          class="absolute right-2 top-2 rounded-lg bg-gray-800/80 p-1.5 text-gray-400 opacity-0 transition-all hover:bg-gray-700 hover:text-white group-hover:opacity-100"
          :title="t('admin.accounts.copyOutput')"
        >
          <Icon name="link" size="sm" :stroke-width="2" />
        </button>
      </div>

      <div v-if="generatedImages.length > 0" class="space-y-2">
        <div class="text-xs font-medium text-gray-600 dark:text-gray-300">
          {{ t('admin.accounts.imageTestPreview') }}
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <a
            v-for="(image, index) in generatedImages"
            :key="`${image.url}-${index}`"
            :href="image.url"
            target="_blank"
            rel="noopener noreferrer"
            class="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm transition hover:border-primary-300 hover:shadow-md dark:border-dark-500 dark:bg-dark-700"
          >
            <img :src="image.url" :alt="`account-test-image-${index + 1}`" class="h-48 w-full object-cover" />
            <div class="border-t border-gray-100 px-3 py-2 text-xs text-gray-500 dark:border-dark-500 dark:text-gray-300">
              {{ image.mimeType || 'image/*' }}
            </div>
          </a>
        </div>
      </div>

      <div
        v-if="runtimeContextItems.length > 0"
        class="rounded-xl border border-sky-200 bg-sky-50 px-4 py-3 text-xs text-sky-900 dark:border-sky-900/60 dark:bg-sky-950/30 dark:text-sky-100"
      >
        <div class="text-sm font-semibold">
          {{ t('admin.accounts.testRuntimeContextTitle') }}
        </div>
        <div class="mt-2 flex flex-wrap items-center gap-2">
          <span
            v-for="item in runtimeContextItems"
            :key="item.key"
            class="inline-flex items-center rounded-full bg-white/80 px-2.5 py-1 font-medium text-sky-700 dark:bg-white/10 dark:text-sky-200"
          >
            {{ item.label }}
          </span>
        </div>
      </div>

      <div
        v-if="visibleBlacklistAdvice"
        class="rounded-xl border px-4 py-3"
        :class="blacklistAdviceClasses"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="text-sm font-semibold">
              {{ blacklistAdviceTitle }}
            </div>
            <p class="mt-1 whitespace-pre-wrap text-xs leading-5 opacity-90">
              {{ blacklistAdviceMessage }}
            </p>
          </div>
          <span
            class="shrink-0 rounded-full px-2.5 py-1 text-[11px] font-semibold"
            :class="blacklistAdviceBadgeClasses"
          >
            {{ blacklistAdviceBadge }}
          </span>
        </div>
      </div>

      <!-- Test Info -->
      <div class="flex items-center justify-between px-1 text-xs text-gray-500 dark:text-gray-400">
        <div class="flex items-center gap-3">
          <span class="flex items-center gap-1">
            <Icon name="grid" size="sm" :stroke-width="2" />
            {{ t('admin.accounts.testModel') }}
          </span>
        </div>
        <span class="flex items-center gap-1">
          <Icon name="chat" size="sm" :stroke-width="2" />
          {{
            supportsImageTest
              ? t('admin.accounts.imageTestMode')
              : supportsPromptInput
                ? t('admin.accounts.textTestPromptLabel')
                : t('admin.accounts.testPrompt')
          }}
        </span>
      </div>
    </div>


    <template #footer>
      <div class="flex justify-end gap-3">
        <button
          @click="handleClose"
          class="rounded-lg bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500"
        >
          {{ t('common.close') }}
        </button>
        <button
          @click="handleBlacklist"
          :disabled="blacklistButtonDisabled"
          :class="[
            'flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-all',
            blacklistButtonDisabled
              ? 'cursor-not-allowed bg-rose-200 text-rose-500 dark:bg-rose-950/40 dark:text-rose-300/60'
              : blacklistAdvice?.decision === 'not_recommended'
                ? 'bg-amber-500 text-white hover:bg-amber-600'
                : 'bg-rose-500 text-white hover:bg-rose-600'
          ]"
        >
          <Icon name="ban" size="sm" :stroke-width="2" />
          <span>{{ blacklistButtonLabel }}</span>
        </button>
        <button
          @click="startTest"
          :disabled="status === 'connecting' || !effectiveSelectedModelId"
          :class="[
            'flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-all',
            status === 'connecting' || !effectiveSelectedModelId
              ? 'cursor-not-allowed bg-primary-400 text-white'
              : status === 'success'
                ? 'bg-green-500 text-white hover:bg-green-600'
                : status === 'error'
                  ? 'bg-orange-500 text-white hover:bg-orange-600'
                  : 'bg-primary-500 text-white hover:bg-primary-600'
          ]"
        >
          <Icon
            v-if="status === 'connecting'"
            name="refresh"
            size="sm"
            class="animate-spin"
            :stroke-width="2"
          />
          <Icon v-else-if="status === 'idle'" name="play" size="sm" :stroke-width="2" />
          <Icon v-else name="refresh" size="sm" :stroke-width="2" />
          <span>
            {{
              status === 'connecting'
                ? t('admin.accounts.testing')
                : status === 'idle'
                  ? t('admin.accounts.startTest')
                  : t('admin.accounts.retry')
            }}
          </span>
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import BaseDialog from '@/components/common/BaseDialog.vue'
import TextArea from '@/components/common/TextArea.vue'
import AccountTestModelSelectionFields from './AccountTestModelSelectionFields.vue'
import { Icon } from '@/components/icons'

const props = defineProps<{ ctx: any }>()
const {
  t,
  show,
  account,
  supportsTestModes,
  testModeOptions,
  selectedTestMode,
  status,
  selectTestMode,
  loadingModels,
  loadAvailableModels,
  modelInputMode,
  selectedModelKey,
  manualModelId,
  manualSourceProtocol,
  availableModels,
  isProtocolGatewayAccount,
  manualRequestAlias,
  isKiroAccount,
  isGrokAccount,
  grokTestHintKey,
  supportsPromptInput,
  testPrompt,
  promptInputLabel,
  promptInputPlaceholder,
  promptInputHint,
  terminalRef,
  outputLines,
  streamingContent,
  errorMessage,
  copyOutput,
  generatedImages,
  runtimeContextItems,
  visibleBlacklistAdvice,
  blacklistAdviceClasses,
  blacklistAdviceTitle,
  blacklistAdviceMessage,
  blacklistAdviceBadgeClasses,
  blacklistAdviceBadge,
  supportsImageTest,
  handleClose,
  handleBlacklist,
  blacklistButtonDisabled,
  blacklistAdvice,
  blacklistButtonLabel,
  startTest,
  effectiveSelectedModelId,
} = props.ctx
</script>
