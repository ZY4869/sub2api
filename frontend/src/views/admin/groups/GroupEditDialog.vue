<template>
  <BaseDialog
    :show="showEditModal"
    :title="t('admin.groups.editGroup')"
    width="normal"
    @close="closeEditModal"
  >
      <form
        v-if="editingGroup"
        id="edit-group-form"
        @submit.prevent="handleUpdateGroup"
        class="space-y-5"
      >
        <div>
          <label class="input-label">{{ t('admin.groups.form.name') }}</label>
          <input
            v-model="editForm.name"
            type="text"
            required
            class="input"
            data-tour="edit-group-form-name"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.groups.form.description') }}</label>
          <textarea v-model="editForm.description" rows="3" class="input"></textarea>
        </div>
        <div>
          <label class="input-label">{{ t('admin.groups.form.platform') }}</label>
          <Select
            v-model="editForm.platform"
            :options="platformOptions"
            :disabled="true"
            data-tour="group-form-platform"
          >
            <template #selected="{ option }">
              <PlatformLabel
                v-if="isPlatformSelectOption(option)"
                :platform="selectOption(option).platform"
                :label="selectOption(option).label"
              />
            </template>
            <template #option="{ option }">
              <PlatformLabel
                v-if="isPlatformSelectOption(option)"
                :platform="selectOption(option).platform"
                :label="selectOption(option).label"
              />
            </template>
          </Select>
          <p class="input-hint">{{ t('admin.groups.platformNotEditable') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.groups.form.priorityLabel') }}</label>
          <input
            v-model.number="editForm.priority"
            type="number"
            min="1"
            required
            class="input"
          />
          <p class="input-hint">{{ t('admin.groups.form.priorityHint') }}</p>
        </div>
        <div v-if="copyAccountsGroupSelectOptionsForEdit.length > 0">
          <div class="mb-1.5 flex items-center gap-1">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.groups.copyAccounts.title') }}
            </label>
            <div class="group relative inline-flex">
              <Icon
                name="questionCircle"
                size="sm"
                :stroke-width="2"
                class="cursor-help text-gray-400 transition-colors hover:text-primary-500 dark:text-gray-500 dark:hover:text-primary-400"
              />
              <div class="pointer-events-none absolute bottom-full left-0 z-50 mb-2 w-72 opacity-0 transition-all duration-200 group-hover:pointer-events-auto group-hover:opacity-100">
                <div class="rounded-lg bg-gray-900 p-3 text-white shadow-lg dark:bg-gray-800">
                  <p class="text-xs leading-relaxed text-gray-300">
                    {{ t('admin.groups.copyAccounts.tooltipEdit') }}
                  </p>
                  <div class="absolute -bottom-1.5 left-3 h-3 w-3 rotate-45 bg-gray-900 dark:bg-gray-800"></div>
                </div>
              </div>
            </div>
          </div>
          <div v-if="editForm.copy_accounts_from_group_ids.length > 0" class="flex flex-wrap gap-1.5 mb-2">
            <span
              v-for="groupId in editForm.copy_accounts_from_group_ids"
              :key="groupId"
              class="inline-flex items-center gap-1 rounded-full bg-primary-100 px-2.5 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
            >
              <GroupBadge
                v-if="findGroupSelectOption(copyAccountsGroupSelectOptionsForEdit, groupId)?.platform"
                :name="findGroupSelectOption(copyAccountsGroupSelectOptionsForEdit, groupId)?.name || `#${groupId}`"
                :platform="findGroupSelectOption(copyAccountsGroupSelectOptionsForEdit, groupId)?.platform"
                :show-rate="false"
              />
              <span v-else>{{ `#${groupId}` }}</span>
              <button
                type="button"
                @click="removeEditCopyGroup(groupId)"
                class="ml-0.5 text-primary-500 hover:text-primary-700 dark:hover:text-primary-200"
              >
                <Icon name="x" size="xs" />
              </button>
            </span>
          </div>
          <Select
            v-model="editCopyAccountsSelection"
            :options="availableEditCopyOptions()"
            :placeholder="t('admin.groups.copyAccounts.selectPlaceholder')"
            @change="handleEditCopyAccountsSelect"
          >
            <template #option="{ option, selected }">
              <GroupOptionItem
                v-if="isGroupSelectOption(option)"
                :name="selectOption(option).name"
                :platform="selectOption(option).platform"
                :description="selectOption(option).description"
                :selected="selected"
              />
            </template>
          </Select>
          <p class="input-hint">{{ t('admin.groups.copyAccounts.hintEdit') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.groups.form.rateMultiplier') }}</label>
          <input
            v-model.number="editForm.rate_multiplier"
            type="number"
            step="0.001"
            min="0.001"
            required
            class="input"
            data-tour="group-form-multiplier"
          />
        </div>
        <div
          v-if="editForm.platform === 'gemini'"
          class="rounded-lg border border-sky-200 bg-sky-50 p-4 dark:border-sky-900/40 dark:bg-sky-950/20"
        >
          <div class="flex items-center justify-between gap-4">
            <div>
              <label class="text-sm font-medium text-gray-900 dark:text-white">
                {{ t('admin.groups.geminiMixedProtocol.title') }}
              </label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.geminiMixedProtocol.hint') }}
              </p>
            </div>
            <button
              type="button"
              class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors"
              :class="editForm.gemini_mixed_protocol_enabled ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'"
              @click="editForm.gemini_mixed_protocol_enabled = !editForm.gemini_mixed_protocol_enabled"
            >
              <span
                class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
                :class="editForm.gemini_mixed_protocol_enabled ? 'translate-x-6' : 'translate-x-1'"
              />
            </button>
          </div>
        </div>
        <div v-if="editForm.subscription_type !== 'subscription'">
          <div class="mb-1.5 flex items-center gap-1">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.groups.form.exclusive') }}
            </label>
            <div class="group relative inline-flex">
              <Icon
                name="questionCircle"
                size="sm"
                :stroke-width="2"
                class="cursor-help text-gray-400 transition-colors hover:text-primary-500 dark:text-gray-500 dark:hover:text-primary-400"
              />
              <div class="pointer-events-none absolute bottom-full left-0 z-50 mb-2 w-72 opacity-0 transition-all duration-200 group-hover:pointer-events-auto group-hover:opacity-100">
                <div class="rounded-lg bg-gray-900 p-3 text-white shadow-lg dark:bg-gray-800">
                  <p class="mb-2 text-xs font-medium">{{ t('admin.groups.exclusiveTooltip.title') }}</p>
                  <p class="mb-2 text-xs leading-relaxed text-gray-300">
                    {{ t('admin.groups.exclusiveTooltip.description') }}
                  </p>
                  <div class="rounded bg-gray-800 p-2 dark:bg-gray-700">
                    <p class="text-xs leading-relaxed text-gray-300">
                      <span class="inline-flex items-center gap-1 text-primary-400"><Icon name="lightbulb" size="xs" /> {{ t('admin.groups.exclusiveTooltip.example') }}</span>
                      {{ t('admin.groups.exclusiveTooltip.exampleContent') }}
                    </p>
                  </div>
                  <div class="absolute -bottom-1.5 left-3 h-3 w-3 rotate-45 bg-gray-900 dark:bg-gray-800"></div>
                </div>
              </div>
            </div>
          </div>
          <div class="flex items-center gap-3">
            <button
              type="button"
              @click="editForm.is_exclusive = !editForm.is_exclusive"
              :class="[
                'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
                editForm.is_exclusive ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform',
                  editForm.is_exclusive ? 'translate-x-6' : 'translate-x-1'
                ]"
              />
            </button>
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ editForm.is_exclusive ? t('admin.groups.exclusive') : t('admin.groups.public') }}
            </span>
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('admin.groups.form.status') }}</label>
          <Select v-model="editForm.status" :options="editStatusOptions" />
        </div>

        <div class="mt-4 border-t pt-4">
          <div>
            <label class="input-label">{{ t('admin.groups.subscription.type') }}</label>
            <Select
              v-model="editForm.subscription_type"
              :options="subscriptionTypeOptions"
              :disabled="true"
            />
            <p class="input-hint">{{ t('admin.groups.subscription.typeNotEditable') }}</p>
          </div>

          <div
            v-if="editForm.subscription_type === 'subscription'"
            class="space-y-4 border-l-2 border-primary-200 pl-4 dark:border-primary-800"
          >
            <div>
              <label class="input-label">{{ t('admin.groups.subscription.dailyLimit') }}</label>
              <input
                v-model.number="editForm.daily_limit_usd"
                type="number"
                step="0.01"
                min="0"
                class="input"
                :placeholder="t('admin.groups.subscription.noLimit')"
              />
            </div>
            <div>
              <label class="input-label">{{ t('admin.groups.subscription.weeklyLimit') }}</label>
              <input
                v-model.number="editForm.weekly_limit_usd"
                type="number"
                step="0.01"
                min="0"
                class="input"
                :placeholder="t('admin.groups.subscription.noLimit')"
              />
            </div>
            <div>
              <label class="input-label">{{ t('admin.groups.subscription.monthlyLimit') }}</label>
              <input
                v-model.number="editForm.monthly_limit_usd"
                type="number"
                step="0.01"
                min="0"
                class="input"
                :placeholder="t('admin.groups.subscription.noLimit')"
              />
            </div>
          </div>
        </div>

        <div v-if="editForm.platform === 'antigravity' || editForm.platform === 'gemini'" class="border-t pt-4">
          <label class="block mb-2 font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.groups.imagePricing.title') }}
          </label>
          <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">
            {{ t('admin.groups.imagePricing.description') }}
          </p>
          <div class="grid grid-cols-3 gap-3">
            <div>
              <label class="input-label">1K ($)</label>
              <input
                v-model.number="editForm.image_price_1k"
                type="number"
                step="0.001"
                min="0"
                class="input"
                placeholder="0.134"
              />
            </div>
            <div>
              <label class="input-label">2K ($)</label>
              <input
                v-model.number="editForm.image_price_2k"
                type="number"
                step="0.001"
                min="0"
                class="input"
                placeholder="0.201"
              />
            </div>
            <div>
              <label class="input-label">4K ($)</label>
              <input
                v-model.number="editForm.image_price_4k"
                type="number"
                step="0.001"
                min="0"
                class="input"
                placeholder="0.268"
              />
            </div>
          </div>
        </div>

        <div v-if="false" class="hidden"></div>

        <div class="border-t pt-4">
          <label class="input-label">{{ t('admin.groups.visibleModels.title') }}</label>
          <textarea
            v-model="editForm.visible_model_patterns_text"
            rows="4"
            class="input font-mono text-xs"
            :placeholder="t('admin.groups.visibleModels.placeholder')"
          ></textarea>
          <p class="input-hint">{{ t('admin.groups.visibleModels.hint') }}</p>
        </div>

        <div v-if="editForm.platform === 'antigravity'" class="border-t pt-4">
          <div class="mb-1.5 flex items-center gap-1">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.groups.supportedScopes.title') }}
            </label>
            <div class="group relative inline-flex">
              <Icon
                name="questionCircle"
                size="sm"
                :stroke-width="2"
                class="cursor-help text-gray-400 transition-colors hover:text-primary-500 dark:text-gray-500 dark:hover:text-primary-400"
              />
              <div class="pointer-events-none absolute bottom-full left-0 z-50 mb-2 w-72 opacity-0 transition-all duration-200 group-hover:pointer-events-auto group-hover:opacity-100">
                <div class="rounded-lg bg-gray-900 p-3 text-white shadow-lg dark:bg-gray-800">
                  <p class="text-xs leading-relaxed text-gray-300">
                    {{ t('admin.groups.supportedScopes.tooltip') }}
                  </p>
                  <div class="absolute -bottom-1.5 left-3 h-3 w-3 rotate-45 bg-gray-900 dark:bg-gray-800"></div>
                </div>
              </div>
            </div>
          </div>
          <div class="space-y-2">
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                :checked="editForm.supported_model_scopes.includes('claude')"
                @change="toggleEditScope('claude')"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-700"
              />
              <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.groups.supportedScopes.claude') }}</span>
            </label>
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                :checked="editForm.supported_model_scopes.includes('gemini_text')"
                @change="toggleEditScope('gemini_text')"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-700"
              />
              <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.groups.supportedScopes.geminiText') }}</span>
            </label>
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                :checked="editForm.supported_model_scopes.includes('gemini_image')"
                @change="toggleEditScope('gemini_image')"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-700"
              />
              <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.groups.supportedScopes.geminiImage') }}</span>
            </label>
          </div>
          <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.groups.supportedScopes.hint') }}</p>
        </div>

        <div v-if="editForm.platform === 'antigravity'" class="border-t pt-4">
          <div class="mb-1.5 flex items-center gap-1">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.groups.mcpXml.title') }}
            </label>
            <div class="group relative inline-flex">
              <Icon
                name="questionCircle"
                size="sm"
                :stroke-width="2"
                class="cursor-help text-gray-400 transition-colors hover:text-primary-500 dark:text-gray-500 dark:hover:text-primary-400"
              />
              <div class="pointer-events-none absolute bottom-full left-0 z-50 mb-2 w-72 opacity-0 transition-all duration-200 group-hover:pointer-events-auto group-hover:opacity-100">
                <div class="rounded-lg bg-gray-900 p-3 text-white shadow-lg dark:bg-gray-800">
                  <p class="text-xs leading-relaxed text-gray-300">
                    {{ t('admin.groups.mcpXml.tooltip') }}
                  </p>
                  <div class="absolute -bottom-1.5 left-3 h-3 w-3 rotate-45 bg-gray-900 dark:bg-gray-800"></div>
                </div>
              </div>
            </div>
          </div>
          <div class="flex items-center gap-3">
            <button
              type="button"
              @click="editForm.mcp_xml_inject = !editForm.mcp_xml_inject"
              :class="[
                'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
                editForm.mcp_xml_inject ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform',
                  editForm.mcp_xml_inject ? 'translate-x-6' : 'translate-x-1'
                ]"
              />
            </button>
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ editForm.mcp_xml_inject ? t('admin.groups.mcpXml.enabled') : t('admin.groups.mcpXml.disabled') }}
            </span>
          </div>
        </div>

        <div v-if="editForm.platform === 'anthropic'" class="border-t pt-4">
          <div class="mb-1.5 flex items-center gap-1">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.groups.claudeCode.title') }}
            </label>
            <div class="group relative inline-flex">
              <Icon
                name="questionCircle"
                size="sm"
                :stroke-width="2"
                class="cursor-help text-gray-400 transition-colors hover:text-primary-500 dark:text-gray-500 dark:hover:text-primary-400"
              />
              <div class="pointer-events-none absolute bottom-full left-0 z-50 mb-2 w-72 opacity-0 transition-all duration-200 group-hover:pointer-events-auto group-hover:opacity-100">
                <div class="rounded-lg bg-gray-900 p-3 text-white shadow-lg dark:bg-gray-800">
                  <p class="text-xs leading-relaxed text-gray-300">
                    {{ t('admin.groups.claudeCode.tooltip') }}
                  </p>
                  <div class="absolute -bottom-1.5 left-3 h-3 w-3 rotate-45 bg-gray-900 dark:bg-gray-800"></div>
                </div>
              </div>
            </div>
          </div>
          <div class="flex items-center gap-3">
            <button
              type="button"
              @click="editForm.claude_code_only = !editForm.claude_code_only"
              :class="[
                'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
                editForm.claude_code_only ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform',
                  editForm.claude_code_only ? 'translate-x-6' : 'translate-x-1'
                ]"
              />
            </button>
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ editForm.claude_code_only ? t('admin.groups.claudeCode.enabled') : t('admin.groups.claudeCode.disabled') }}
            </span>
          </div>
          <div v-if="editForm.claude_code_only" class="mt-3">
            <label class="input-label">{{ t('admin.groups.claudeCode.fallbackGroup') }}</label>
            <Select
              v-model="editForm.fallback_group_id"
              :options="fallbackGroupOptionsForEdit"
              :placeholder="t('admin.groups.claudeCode.noFallback')"
            >
              <template #selected="{ option }">
                <GroupBadge
                  v-if="isGroupSelectOption(option) && selectOption(option).platform"
                  :name="selectOption(option).name"
                  :platform="selectOption(option).platform"
                  :show-rate="false"
                />
                <span v-else>{{ isGroupSelectOption(option) ? selectOption(option).label : '' }}</span>
              </template>
              <template #option="{ option, selected }">
                <GroupOptionItem
                  v-if="isGroupSelectOption(option) && selectOption(option).platform"
                  :name="selectOption(option).name"
                  :platform="selectOption(option).platform"
                  :description="selectOption(option).description"
                  :selected="selected"
                />
                <span v-else class="text-sm text-gray-700 dark:text-gray-300">
                  {{ isGroupSelectOption(option) ? selectOption(option).label : '' }}
                </span>
              </template>
            </Select>
            <p class="input-hint">{{ t('admin.groups.claudeCode.fallbackHint') }}</p>
          </div>
        </div>

        <div v-if="editForm.platform === 'openai'" class="border-t border-gray-200 dark:border-dark-400 pt-4 mt-4">
          <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">{{ t('admin.groups.openaiMessages.title') }}</h4>

          <div>
            <label class="input-label">{{ t('admin.groups.imageProtocol.label') }}</label>
            <Select
              v-model="editForm.image_protocol_mode"
              :options="openAIGroupImageProtocolModeOptions"
            />
            <p class="input-hint">{{ t('admin.groups.imageProtocol.hint') }}</p>
          </div>

          <div class="mt-4 flex items-center justify-between">
            <label class="text-sm text-gray-600 dark:text-gray-400">{{ t('admin.groups.openaiMessages.allowDispatch') }}</label>
            <button
              type="button"
              @click="editForm.allow_messages_dispatch = !editForm.allow_messages_dispatch"
              class="relative inline-flex h-6 w-12 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none"
              :class="
                editForm.allow_messages_dispatch ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
              "
            >
              <span
                class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
                :class="
                  editForm.allow_messages_dispatch ? 'translate-x-6' : 'translate-x-1'
                "
              />
            </button>
          </div>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{{ t('admin.groups.openaiMessages.allowDispatchHint') }}</p>

          <div v-if="editForm.allow_messages_dispatch" class="mt-3">
            <label class="input-label">{{ t('admin.groups.openaiMessages.defaultModel') }}</label>
            <input
              v-model="editForm.default_mapped_model"
              type="text"
              :placeholder="t('admin.groups.openaiMessages.defaultModelPlaceholder')"
              class="input"
            />
            <p class="input-hint">{{ t('admin.groups.openaiMessages.defaultModelHint') }}</p>
          </div>
        </div>

        <div
          v-if="['anthropic', 'antigravity'].includes(editForm.platform) && editForm.subscription_type !== 'subscription'"
          class="border-t pt-4"
        >
          <label class="input-label">{{ t('admin.groups.invalidRequestFallback.title') }}</label>
          <Select
            v-model="editForm.fallback_group_id_on_invalid_request"
            :options="invalidRequestFallbackOptionsForEdit"
            :placeholder="t('admin.groups.invalidRequestFallback.noFallback')"
          >
            <template #selected="{ option }">
              <GroupBadge
                v-if="isGroupSelectOption(option) && selectOption(option).platform"
                :name="selectOption(option).name"
                :platform="selectOption(option).platform"
                :show-rate="false"
              />
              <span v-else>{{ isGroupSelectOption(option) ? selectOption(option).label : '' }}</span>
            </template>
            <template #option="{ option, selected }">
              <GroupOptionItem
                v-if="isGroupSelectOption(option) && selectOption(option).platform"
                :name="selectOption(option).name"
                :platform="selectOption(option).platform"
                :description="selectOption(option).description"
                :selected="selected"
              />
              <span v-else class="text-sm text-gray-700 dark:text-gray-300">
                {{ isGroupSelectOption(option) ? selectOption(option).label : '' }}
              </span>
            </template>
          </Select>
          <p class="input-hint">{{ t('admin.groups.invalidRequestFallback.hint') }}</p>
        </div>

        <div v-if="editForm.platform === 'anthropic'" class="border-t pt-4">
          <div class="mb-1.5 flex items-center gap-1">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.groups.modelRouting.title') }}
            </label>
            <div class="group relative inline-flex">
              <Icon
                name="questionCircle"
                size="sm"
                :stroke-width="2"
                class="cursor-help text-gray-400 transition-colors hover:text-primary-500 dark:text-gray-500 dark:hover:text-primary-400"
              />
              <div class="pointer-events-none absolute bottom-full left-0 z-50 mb-2 w-80 opacity-0 transition-all duration-200 group-hover:pointer-events-auto group-hover:opacity-100">
                <div class="rounded-lg bg-gray-900 p-3 text-white shadow-lg dark:bg-gray-800">
                  <p class="text-xs leading-relaxed text-gray-300">
                    {{ t('admin.groups.modelRouting.tooltip') }}
                  </p>
                  <div class="absolute -bottom-1.5 left-3 h-3 w-3 rotate-45 bg-gray-900 dark:bg-gray-800"></div>
                </div>
              </div>
            </div>
          </div>
          <div class="flex items-center gap-3 mb-3">
            <button
              type="button"
              @click="editForm.model_routing_enabled = !editForm.model_routing_enabled"
              :class="[
                'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
                editForm.model_routing_enabled ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform',
                  editForm.model_routing_enabled ? 'translate-x-6' : 'translate-x-1'
                ]"
              />
            </button>
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ editForm.model_routing_enabled ? t('admin.groups.modelRouting.enabled') : t('admin.groups.modelRouting.disabled') }}
            </span>
          </div>
          <p v-if="!editForm.model_routing_enabled" class="text-xs text-gray-500 dark:text-gray-400 mb-3">
            {{ t('admin.groups.modelRouting.disabledHint') }}
          </p>
          <p v-else class="text-xs text-gray-500 dark:text-gray-400 mb-3">
            {{ t('admin.groups.modelRouting.noRulesHint') }}
          </p>
          <div v-if="editForm.model_routing_enabled" class="space-y-3">
            <div
              v-for="rule in editModelRoutingRules"
              :key="getEditRuleRenderKey(rule)"
              class="rounded-lg border border-gray-200 p-3 dark:border-dark-600"
            >
              <div class="flex items-start gap-3">
                <div class="flex-1 space-y-2">
                  <div>
                    <label class="input-label text-xs">{{ t('admin.groups.modelRouting.modelPattern') }}</label>
                    <input
                      v-model="rule.pattern"
                      type="text"
                      class="input text-sm"
                      :placeholder="t('admin.groups.modelRouting.modelPatternPlaceholder')"
                    />
                  </div>
                  <div>
                    <label class="input-label text-xs">{{ t('admin.groups.modelRouting.accounts') }}</label>
                    <div v-if="rule.accounts.length > 0" class="flex flex-wrap gap-1.5 mb-2">
                      <span
                        v-for="account in rule.accounts"
                        :key="account.id"
                        class="inline-flex items-center gap-1 rounded-full bg-primary-100 px-2.5 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
                      >
                        {{ account.name }}
                        <button
                          type="button"
                          @click="removeSelectedAccount(rule, account.id, true)"
                          class="ml-0.5 text-primary-500 hover:text-primary-700 dark:hover:text-primary-200"
                        >
                          <Icon name="x" size="xs" />
                        </button>
                      </span>
                    </div>
                    <div class="relative account-search-container">
                      <input
                        v-model="accountSearchKeyword[getEditRuleSearchKey(rule)]"
                        type="text"
                        class="input text-sm"
                        :placeholder="t('admin.groups.modelRouting.searchAccountPlaceholder')"
                        @input="searchAccountsByRule(rule, true)"
                        @focus="onAccountSearchFocus(rule, true)"
                      />
                      <div
                        v-if="showAccountDropdown[getEditRuleSearchKey(rule)] && accountSearchResults[getEditRuleSearchKey(rule)]?.length > 0"
                        class="absolute z-50 mt-1 max-h-48 w-full overflow-auto rounded-lg border bg-white shadow-lg dark:border-dark-600 dark:bg-dark-800"
                      >
                        <button
                          v-for="account in accountSearchResults[getEditRuleSearchKey(rule)]"
                          :key="account.id"
                          type="button"
                          @click="selectAccount(rule, account, true)"
                          class="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-dark-700"
                          :class="{ 'opacity-50': ruleHasAccount(rule, account.id) }"
                          :disabled="ruleHasAccount(rule, account.id)"
                        >
                          <span>{{ account.name }}</span>
                          <span class="ml-2 text-xs text-gray-400">#{{ account.id }}</span>
                        </button>
                      </div>
                    </div>
                    <p class="text-xs text-gray-400 mt-1">{{ t('admin.groups.modelRouting.accountsHint') }}</p>
                  </div>
                </div>
                <button
                  type="button"
                  @click="removeEditRoutingRule(rule)"
                  class="mt-5 p-1.5 text-gray-400 hover:text-red-500 transition-colors"
                  :title="t('admin.groups.modelRouting.removeRule')"
                >
                  <Icon name="trash" size="sm" />
                </button>
              </div>
            </div>
          </div>
          <button
            v-if="editForm.model_routing_enabled"
            type="button"
            @click="addEditRoutingRule"
            class="mt-3 flex items-center gap-1.5 text-sm text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
          >
            <Icon name="plus" size="sm" />
            {{ t('admin.groups.modelRouting.addRule') }}
          </button>
        </div>

      </form>

      <template #footer>
        <div class="flex justify-end gap-3 pt-4">
          <button @click="closeEditModal" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button
            type="submit"
            form="edit-group-form"
            :disabled="submitting"
            class="btn btn-primary"
            data-tour="group-form-submit"
          >
            <svg
              v-if="submitting"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{ submitting ? t('admin.groups.updating') : t('common.update') }}
          </button>
        </div>
      </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import PlatformLabel from '@/components/common/PlatformLabel.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
import Icon from '@/components/icons/Icon.vue'
import { unref } from 'vue'

const props = defineProps<{ ctx: any }>()
const {
  t,
  showEditModal,
  submitting,
  editingGroup,
  editCopyAccountsSelection,
  editForm,
  editModelRoutingRules,
  accountSearchKeyword,
  accountSearchResults,
  showAccountDropdown,
  platformOptions,
  editStatusOptions,
  subscriptionTypeOptions,
  openAIGroupImageProtocolModeOptions,
  fallbackGroupOptionsForEdit,
  invalidRequestFallbackOptionsForEdit,
  copyAccountsGroupSelectOptionsForEdit,
  isPlatformSelectOption,
  isGroupSelectOption,
  closeEditModal,
  handleUpdateGroup,
  handleEditCopyAccountsSelect,
  findGroupSelectOption,
  toggleEditScope,
  getEditRuleRenderKey,
  getEditRuleSearchKey,
  searchAccountsByRule,
  onAccountSearchFocus,
  selectAccount,
  removeSelectedAccount,
  addEditRoutingRule,
  removeEditRoutingRule
} = props.ctx

const selectOption = (option: unknown) => (option ?? {}) as Record<string, any>
const removeEditCopyGroup = (groupId: number) => {
  editForm.copy_accounts_from_group_ids = editForm.copy_accounts_from_group_ids.filter(
    (id: number) => id !== groupId
  )
}
const availableEditCopyOptions = () =>
  unref(copyAccountsGroupSelectOptionsForEdit).filter(
    (option: Record<string, any>) => !editForm.copy_accounts_from_group_ids.includes(option.value as number)
  )
const ruleHasAccount = (rule: any, accountId: number) =>
  Array.isArray(rule?.accounts) && rule.accounts.some((account: any) => account.id === accountId)
</script>
