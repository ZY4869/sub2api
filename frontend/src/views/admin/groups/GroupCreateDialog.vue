<template>
  <BaseDialog
    :show="showCreateModal"
    :title="t('admin.groups.createGroup')"
    width="normal"
    @close="closeCreateModal"
  >
      <form id="create-group-form" @submit.prevent="handleCreateGroup" class="space-y-5">
        <div>
          <label class="input-label">{{ t('admin.groups.form.name') }}</label>
          <input
            v-model="createForm.name"
            type="text"
            required
            class="input"
            :placeholder="t('admin.groups.enterGroupName')"
            data-tour="group-form-name"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.groups.form.description') }}</label>
          <textarea
            v-model="createForm.description"
            rows="3"
            class="input"
            :placeholder="t('admin.groups.optionalDescription')"
          ></textarea>
        </div>
        <div>
          <label class="input-label">{{ t('admin.groups.form.platform') }}</label>
          <Select
            v-model="createForm.platform"
            :options="platformOptions"
            data-tour="group-form-platform"
            @change="handleCreatePlatformChange"
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
          <p class="input-hint">{{ t('admin.groups.platformHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.groups.form.priorityLabel') }}</label>
          <input
            v-model.number="createForm.priority"
            type="number"
            min="1"
            required
            class="input"
          />
          <p class="input-hint">{{ t('admin.groups.form.priorityHint') }}</p>
        </div>
        <div v-if="copyAccountsGroupSelectOptions.length > 0">
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
                    {{ t('admin.groups.copyAccounts.tooltip') }}
                  </p>
                  <div class="absolute -bottom-1.5 left-3 h-3 w-3 rotate-45 bg-gray-900 dark:bg-gray-800"></div>
                </div>
              </div>
            </div>
          </div>
          <div v-if="createForm.copy_accounts_from_group_ids.length > 0" class="flex flex-wrap gap-1.5 mb-2">
            <span
              v-for="groupId in createForm.copy_accounts_from_group_ids"
              :key="groupId"
              class="inline-flex items-center gap-1 rounded-full bg-primary-100 px-2.5 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
            >
              <GroupBadge
                v-if="findGroupSelectOption(copyAccountsGroupSelectOptions, groupId)?.platform"
                :name="findGroupSelectOption(copyAccountsGroupSelectOptions, groupId)?.name || `#${groupId}`"
                :platform="findGroupSelectOption(copyAccountsGroupSelectOptions, groupId)?.platform"
                :show-rate="false"
              />
              <span v-else>{{ `#${groupId}` }}</span>
              <button
                type="button"
                @click="removeCreateCopyGroup(groupId)"
                class="ml-0.5 text-primary-500 hover:text-primary-700 dark:hover:text-primary-200"
              >
                <Icon name="x" size="xs" />
              </button>
            </span>
          </div>
          <Select
            v-model="createCopyAccountsSelection"
            :options="availableCreateCopyOptions()"
            :placeholder="t('admin.groups.copyAccounts.selectPlaceholder')"
            @change="handleCreateCopyAccountsSelect"
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
          <p class="input-hint">{{ t('admin.groups.copyAccounts.hint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.groups.form.rateMultiplier') }}</label>
          <input
            v-model.number="createForm.rate_multiplier"
            type="number"
            step="0.001"
            min="0.001"
            required
            class="input"
            data-tour="group-form-multiplier"
          />
          <p class="input-hint">{{ t('admin.groups.rateMultiplierHint') }}</p>
        </div>
        <div
          v-if="createForm.platform === 'gemini'"
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
              :class="createForm.gemini_mixed_protocol_enabled ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'"
              @click="createForm.gemini_mixed_protocol_enabled = !createForm.gemini_mixed_protocol_enabled"
            >
              <span
                class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
                :class="createForm.gemini_mixed_protocol_enabled ? 'translate-x-6' : 'translate-x-1'"
              />
            </button>
          </div>
        </div>
        <div v-if="createForm.subscription_type !== 'subscription'" data-tour="group-form-exclusive">
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
              @click="createForm.is_exclusive = !createForm.is_exclusive"
              :class="[
                'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
                createForm.is_exclusive ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform',
                  createForm.is_exclusive ? 'translate-x-6' : 'translate-x-1'
                ]"
              />
            </button>
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ createForm.is_exclusive ? t('admin.groups.exclusive') : t('admin.groups.public') }}
            </span>
          </div>
        </div>

        <div class="mt-4 border-t pt-4">
          <div>
            <label class="input-label">{{ t('admin.groups.subscription.type') }}</label>
            <Select v-model="createForm.subscription_type" :options="subscriptionTypeOptions" />
            <p class="input-hint">{{ t('admin.groups.subscription.typeHint') }}</p>
          </div>

          <div
            v-if="createForm.subscription_type === 'subscription'"
            class="space-y-4 border-l-2 border-primary-200 pl-4 dark:border-primary-800"
          >
            <div>
              <label class="input-label">{{ t('admin.groups.subscription.dailyLimit') }}</label>
              <input
                v-model.number="createForm.daily_limit_usd"
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
                v-model.number="createForm.weekly_limit_usd"
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
                v-model.number="createForm.monthly_limit_usd"
                type="number"
                step="0.01"
                min="0"
                class="input"
                :placeholder="t('admin.groups.subscription.noLimit')"
              />
            </div>
          </div>
        </div>

        <div v-if="createForm.platform === 'antigravity' || createForm.platform === 'gemini'" class="border-t pt-4">
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
                v-model.number="createForm.image_price_1k"
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
                v-model.number="createForm.image_price_2k"
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
                v-model.number="createForm.image_price_4k"
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
            v-model="createForm.visible_model_patterns_text"
            rows="4"
            class="input font-mono text-xs"
            :placeholder="t('admin.groups.visibleModels.placeholder')"
          ></textarea>
          <p class="input-hint">{{ t('admin.groups.visibleModels.hint') }}</p>
        </div>

        <div v-if="createForm.platform === 'antigravity'" class="border-t pt-4">
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
                :checked="createForm.supported_model_scopes.includes('claude')"
                @change="toggleCreateScope('claude')"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-700"
              />
              <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.groups.supportedScopes.claude') }}</span>
            </label>
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                :checked="createForm.supported_model_scopes.includes('gemini_text')"
                @change="toggleCreateScope('gemini_text')"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-700"
              />
              <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.groups.supportedScopes.geminiText') }}</span>
            </label>
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                :checked="createForm.supported_model_scopes.includes('gemini_image')"
                @change="toggleCreateScope('gemini_image')"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-700"
              />
              <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.groups.supportedScopes.geminiImage') }}</span>
            </label>
          </div>
          <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.groups.supportedScopes.hint') }}</p>
        </div>

        <div v-if="createForm.platform === 'antigravity'" class="border-t pt-4">
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
              @click="createForm.mcp_xml_inject = !createForm.mcp_xml_inject"
              :class="[
                'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
                createForm.mcp_xml_inject ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform',
                  createForm.mcp_xml_inject ? 'translate-x-6' : 'translate-x-1'
                ]"
              />
            </button>
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ createForm.mcp_xml_inject ? t('admin.groups.mcpXml.enabled') : t('admin.groups.mcpXml.disabled') }}
            </span>
          </div>
        </div>

        <div v-if="createForm.platform === 'anthropic'" class="border-t pt-4">
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
              @click="createForm.claude_code_only = !createForm.claude_code_only"
              :class="[
                'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
                createForm.claude_code_only ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform',
                  createForm.claude_code_only ? 'translate-x-6' : 'translate-x-1'
                ]"
              />
            </button>
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ createForm.claude_code_only ? t('admin.groups.claudeCode.enabled') : t('admin.groups.claudeCode.disabled') }}
            </span>
          </div>
          <div v-if="createForm.claude_code_only" class="mt-3">
            <label class="input-label">{{ t('admin.groups.claudeCode.fallbackGroup') }}</label>
            <Select
              v-model="createForm.fallback_group_id"
              :options="fallbackGroupOptions"
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

        <div v-if="createForm.platform === 'openai'" class="border-t border-gray-200 dark:border-dark-400 pt-4 mt-4">
          <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">{{ t('admin.groups.openaiMessages.title') }}</h4>

          <div>
            <label class="input-label">{{ t('admin.groups.imageProtocol.label') }}</label>
            <Select
              v-model="createForm.image_protocol_mode"
              :options="openAIGroupImageProtocolModeOptions"
            />
            <p class="input-hint">{{ t('admin.groups.imageProtocol.hint') }}</p>
          </div>

          <div class="mt-4 flex items-center justify-between">
            <label class="text-sm text-gray-600 dark:text-gray-400">{{ t('admin.groups.openaiMessages.allowDispatch') }}</label>
            <button
              type="button"
              @click="createForm.allow_messages_dispatch = !createForm.allow_messages_dispatch"
              class="relative inline-flex h-6 w-12 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none"
              :class="
                createForm.allow_messages_dispatch ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
              "
            >
              <span
                class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
                :class="
                  createForm.allow_messages_dispatch ? 'translate-x-6' : 'translate-x-1'
                "
              />
            </button>
          </div>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{{ t('admin.groups.openaiMessages.allowDispatchHint') }}</p>

          <div v-if="createForm.allow_messages_dispatch" class="mt-3">
            <label class="input-label">{{ t('admin.groups.openaiMessages.defaultModel') }}</label>
            <input
              v-model="createForm.default_mapped_model"
              type="text"
              :placeholder="t('admin.groups.openaiMessages.defaultModelPlaceholder')"
              class="input"
            />
            <p class="input-hint">{{ t('admin.groups.openaiMessages.defaultModelHint') }}</p>
          </div>
        </div>

        <div
          v-if="['anthropic', 'antigravity'].includes(createForm.platform) && createForm.subscription_type !== 'subscription'"
          class="border-t pt-4"
        >
          <label class="input-label">{{ t('admin.groups.invalidRequestFallback.title') }}</label>
          <Select
            v-model="createForm.fallback_group_id_on_invalid_request"
            :options="invalidRequestFallbackOptions"
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

        <div v-if="createForm.platform === 'anthropic'" class="border-t pt-4">
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
              @click="createForm.model_routing_enabled = !createForm.model_routing_enabled"
              :class="[
                'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
                createForm.model_routing_enabled ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform',
                  createForm.model_routing_enabled ? 'translate-x-6' : 'translate-x-1'
                ]"
              />
            </button>
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ createForm.model_routing_enabled ? t('admin.groups.modelRouting.enabled') : t('admin.groups.modelRouting.disabled') }}
            </span>
          </div>
          <p v-if="!createForm.model_routing_enabled" class="text-xs text-gray-500 dark:text-gray-400 mb-3">
            {{ t('admin.groups.modelRouting.disabledHint') }}
          </p>
          <p v-else class="text-xs text-gray-500 dark:text-gray-400 mb-3">
            {{ t('admin.groups.modelRouting.noRulesHint') }}
          </p>
          <div v-if="createForm.model_routing_enabled" class="space-y-3">
            <div
              v-for="rule in createModelRoutingRules"
              :key="getCreateRuleRenderKey(rule)"
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
                          @click="removeSelectedAccount(rule, account.id)"
                          class="ml-0.5 text-primary-500 hover:text-primary-700 dark:hover:text-primary-200"
                        >
                          <Icon name="x" size="xs" />
                        </button>
                      </span>
                    </div>
                    <div class="relative account-search-container">
                      <input
                        v-model="accountSearchKeyword[getCreateRuleSearchKey(rule)]"
                        type="text"
                        class="input text-sm"
                        :placeholder="t('admin.groups.modelRouting.searchAccountPlaceholder')"
                        @input="searchAccountsByRule(rule)"
                        @focus="onAccountSearchFocus(rule)"
                      />
                      <div
                        v-if="showAccountDropdown[getCreateRuleSearchKey(rule)] && accountSearchResults[getCreateRuleSearchKey(rule)]?.length > 0"
                        class="absolute z-50 mt-1 max-h-48 w-full overflow-auto rounded-lg border bg-white shadow-lg dark:border-dark-600 dark:bg-dark-800"
                      >
                        <button
                          v-for="account in accountSearchResults[getCreateRuleSearchKey(rule)]"
                          :key="account.id"
                          type="button"
                          @click="selectAccount(rule, account)"
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
                  @click="removeCreateRoutingRule(rule)"
                  class="mt-5 p-1.5 text-gray-400 hover:text-red-500 transition-colors"
                  :title="t('admin.groups.modelRouting.removeRule')"
                >
                  <Icon name="trash" size="sm" />
                </button>
              </div>
            </div>
          </div>
          <button
            v-if="createForm.model_routing_enabled"
            type="button"
            @click="addCreateRoutingRule"
            class="mt-3 flex items-center gap-1.5 text-sm text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
          >
            <Icon name="plus" size="sm" />
            {{ t('admin.groups.modelRouting.addRule') }}
          </button>
        </div>

      </form>

      <template #footer>
        <div class="flex justify-end gap-3 pt-4">
          <button @click="closeCreateModal" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button
            type="submit"
            form="create-group-form"
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
            {{ submitting ? t('admin.groups.creating') : t('common.create') }}
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
  showCreateModal,
  submitting,
  createCopyAccountsSelection,
  createForm,
  createModelRoutingRules,
  accountSearchKeyword,
  accountSearchResults,
  showAccountDropdown,
  platformOptions,
  subscriptionTypeOptions,
  openAIGroupImageProtocolModeOptions,
  fallbackGroupOptions,
  invalidRequestFallbackOptions,
  copyAccountsGroupSelectOptions,
  isPlatformSelectOption,
  isGroupSelectOption,
  closeCreateModal,
  handleCreateGroup,
  handleCreatePlatformChange,
  handleCreateCopyAccountsSelect,
  findGroupSelectOption,
  toggleCreateScope,
  getCreateRuleRenderKey,
  getCreateRuleSearchKey,
  searchAccountsByRule,
  onAccountSearchFocus,
  selectAccount,
  removeSelectedAccount,
  addCreateRoutingRule,
  removeCreateRoutingRule
} = props.ctx

const selectOption = (option: unknown) => (option ?? {}) as Record<string, any>
const removeCreateCopyGroup = (groupId: number) => {
  createForm.copy_accounts_from_group_ids = createForm.copy_accounts_from_group_ids.filter(
    (id: number) => id !== groupId
  )
}
const availableCreateCopyOptions = () =>
  unref(copyAccountsGroupSelectOptions).filter(
    (option: Record<string, any>) => !createForm.copy_accounts_from_group_ids.includes(option.value as number)
  )
const ruleHasAccount = (rule: any, accountId: number) =>
  Array.isArray(rule?.accounts) && rule.accounts.some((account: any) => account.id === accountId)
</script>
