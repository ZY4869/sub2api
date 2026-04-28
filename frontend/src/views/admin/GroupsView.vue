<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col justify-between gap-4 lg:flex-row lg:items-start">
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <div class="relative w-full sm:w-64">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
              />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('admin.groups.searchGroups')"
                class="input pl-10"
                @input="handleSearch"
              />
            </div>
          <Select
            v-model="filters.platform"
            :options="platformFilterOptions"
            :placeholder="t('admin.groups.allPlatforms')"
            class="w-44"
            @change="loadGroups"
          >
            <template #selected="{ option }">
              <PlatformLabel
                v-if="isPlatformSelectOption(option)"
                :platform="option.platform"
                :label="option.label"
              />
            </template>
            <template #option="{ option }">
              <PlatformLabel
                v-if="isPlatformSelectOption(option)"
                :platform="option.platform"
                :label="option.label"
              />
            </template>
          </Select>
          <Select
            v-model="filters.status"
            :options="statusOptions"
            :placeholder="t('admin.groups.allStatus')"
            class="w-40"
            @change="loadGroups"
          />
          <Select
            v-model="filters.is_exclusive"
            :options="exclusiveOptions"
            :placeholder="t('admin.groups.allGroups')"
            class="w-44"
            @change="loadGroups"
          />
          </div>

          <div class="flex w-full flex-shrink-0 flex-wrap items-center justify-end gap-3 lg:w-auto">
            <button
              @click="loadGroups"
              :disabled="loading"
              class="btn btn-secondary"
              :title="t('common.refresh')"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button
              @click="openSortModal"
              class="btn btn-secondary"
              :title="t('admin.groups.sortOrder')"
            >
              <Icon name="arrowsUpDown" size="md" class="mr-2" />
              {{ t('admin.groups.sortOrder') }}
            </button>
            <button
              @click="showCreateModal = true"
              class="btn btn-primary"
              data-tour="groups-create-btn"
            >
              <Icon name="plus" size="md" class="mr-2" />
              {{ t('admin.groups.createGroup') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="groups" :loading="loading">
          <template #cell-name="{ value }">
            <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
          </template>

          <template #cell-platform="{ value }">
            <span
              :class="[
                'inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium',
                value === 'anthropic'
                  ? 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
                  : value === 'kiro'
                    ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
                  : value === 'openai'
                    ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                    : value === 'copilot'
                      ? 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400'
                      : value === 'grok'
                        ? 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-200'
                      : value === 'deepseek'
                        ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400'
                      : value === 'antigravity'
                        ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
                        : 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
              ]"
            >
              <PlatformIcon :platform="value" size="xs" />
              {{ t('admin.groups.platforms.' + value) }}
            </span>
          </template>

          <template #cell-billing_type="{ row }">
            <div class="space-y-1">
              <span
                :class="[
                  'inline-block rounded-full px-2 py-0.5 text-xs font-medium',
                  row.subscription_type === 'subscription'
                    ? 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400'
                    : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
                ]"
              >
                {{
                  row.subscription_type === 'subscription'
                    ? t('admin.groups.subscription.subscription')
                    : t('admin.groups.subscription.standard')
                }}
              </span>
              <div
                v-if="row.subscription_type === 'subscription'"
                class="text-xs text-gray-500 dark:text-gray-400"
              >
                <template
                  v-if="row.daily_limit_usd || row.weekly_limit_usd || row.monthly_limit_usd"
                >
                  <span v-if="row.daily_limit_usd">${{ row.daily_limit_usd }}/{{ t('admin.groups.limitDay') }}</span>
                  <span
                    v-if="row.daily_limit_usd && (row.weekly_limit_usd || row.monthly_limit_usd)"
                    class="mx-1 text-gray-300 dark:text-gray-600"
                  >/</span>
                  <span v-if="row.weekly_limit_usd">${{ row.weekly_limit_usd }}/{{ t('admin.groups.limitWeek') }}</span>
                  <span
                    v-if="row.weekly_limit_usd && row.monthly_limit_usd"
                    class="mx-1 text-gray-300 dark:text-gray-600"
                  >/</span>
                  <span v-if="row.monthly_limit_usd">${{ row.monthly_limit_usd }}/{{ t('admin.groups.limitMonth') }}</span>
                </template>
                <span v-else class="text-gray-400 dark:text-gray-500">{{
                  t('admin.groups.subscription.noLimit')
                }}</span>








              </div>
            </div>
          </template>

          <template #cell-rate_multiplier="{ value }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ value }}x</span>
          </template>

          <template #cell-is_exclusive="{ value }">
            <span :class="['badge', value ? 'badge-primary' : 'badge-gray']">
              {{ value ? t('admin.groups.exclusive') : t('admin.groups.public') }}
            </span>
          </template>

          <template #cell-account_count="{ row }">
            <div class="flex min-w-[7.5rem] justify-end gap-3 text-right">
              <div>
                <div class="text-[1.55rem] font-bold leading-none text-emerald-600 dark:text-emerald-400">
                  {{ formatGroupAccountValue(getGroupAvailableAccounts(row), row) }}
                </div>
                <div class="mt-1 text-[11px] tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.accountsAvailable') }}
                </div>
              </div>
              <div>
                <div class="text-[1.55rem] font-bold leading-none text-rose-600 dark:text-rose-400">
                  {{ formatGroupAccountValue(row.rate_limited_account_count || 0, row) }}
                </div>
                <div class="mt-1 text-[11px] tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.accountsRateLimited') }}
                </div>
              </div>
              <div>
                <div class="text-[1.55rem] font-bold leading-none text-gray-700 dark:text-gray-200">
                  {{ formatGroupAccountValue(row.account_count || 0, row) }}
                </div>
                <div class="mt-1 text-[11px] tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.accountsTotal') }}
                </div>
              </div>
            </div>
          </template>

          <template #cell-capacity="{ row }">
            <GroupCapacityBadge
              v-if="capacityMap.get(row.id)"
              :concurrency-used="capacityMap.get(row.id)!.concurrencyUsed"
              :concurrency-max="capacityMap.get(row.id)!.concurrencyMax"
              :sessions-used="capacityMap.get(row.id)!.sessionsUsed"
              :sessions-max="capacityMap.get(row.id)!.sessionsMax"
              :rpm-used="capacityMap.get(row.id)!.rpmUsed"
              :rpm-max="capacityMap.get(row.id)!.rpmMax"
            />
            <span v-else class="text-xs text-gray-400">-</span>
          </template>

          <template #cell-usage="{ row }">
            <div v-if="usageLoading" class="text-xs text-gray-400">-</div>
            <div v-else class="space-y-0.5 text-xs">
              <div class="text-gray-500 dark:text-gray-400">
                <span class="text-gray-400 dark:text-gray-500">{{ t('admin.groups.usageToday') }}</span>
                <span class="ml-1 font-medium text-gray-700 dark:text-gray-300">${{ formatCost(usageMap.get(row.id)?.today_cost ?? 0) }}</span>
              </div>
              <div class="text-gray-500 dark:text-gray-400">
                <span class="text-gray-400 dark:text-gray-500">{{ t('admin.groups.usageTotal') }}</span>
                <span class="ml-1 font-medium text-gray-700 dark:text-gray-300">${{ formatCost(usageMap.get(row.id)?.total_cost ?? 0) }}</span>
              </div>
            </div>
          </template>

          <template #cell-status="{ value }">
            <span :class="['badge', value === 'active' ? 'badge-success' : 'badge-danger']">
              {{ t('admin.accounts.status.' + value) }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button
                @click="handleEdit(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
              >
                <Icon name="edit" size="sm" />
                <span class="text-xs">{{ t('common.edit') }}</span>
              </button>
              <button
                @click="handleRateMultipliers(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-purple-600 dark:hover:bg-dark-700 dark:hover:text-purple-400"
              >
                <Icon name="dollar" size="sm" />
                <span class="text-xs">{{ t('admin.groups.rateMultipliers') }}</span>
              </button>
              <button
                @click="handleDelete(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
              >
                <Icon name="trash" size="sm" />
                <span class="text-xs">{{ t('common.delete') }}</span>
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('admin.groups.noGroupsYet')"
              :description="t('admin.groups.createFirstGroup')"
              :action-text="t('admin.groups.createGroup')"
              @action="showCreateModal = true"
            />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

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
                :platform="option.platform"
                :label="option.label"
              />
            </template>
            <template #option="{ option }">
              <PlatformLabel
                v-if="isPlatformSelectOption(option)"
                :platform="option.platform"
                :label="option.label"
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
                @click="createForm.copy_accounts_from_group_ids = createForm.copy_accounts_from_group_ids.filter(id => id !== groupId)"
                class="ml-0.5 text-primary-500 hover:text-primary-700 dark:hover:text-primary-200"
              >
                <Icon name="x" size="xs" />
              </button>
            </span>
          </div>
          <Select
            v-model="createCopyAccountsSelection"
            :options="copyAccountsGroupSelectOptions.filter(option => !createForm.copy_accounts_from_group_ids.includes(option.value as number))"
            :placeholder="t('admin.groups.copyAccounts.selectPlaceholder')"
            @change="handleCreateCopyAccountsSelect"
          >
            <template #option="{ option, selected }">
              <GroupOptionItem
                v-if="isGroupSelectOption(option)"
                :name="option.name"
                :platform="option.platform"
                :description="option.description"
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
                  v-if="isGroupSelectOption(option) && option.platform"
                  :name="option.name"
                  :platform="option.platform"
                  :show-rate="false"
                />
                <span v-else>{{ isGroupSelectOption(option) ? option.label : '' }}</span>
              </template>
              <template #option="{ option, selected }">
                <GroupOptionItem
                  v-if="isGroupSelectOption(option) && option.platform"
                  :name="option.name"
                  :platform="option.platform"
                  :description="option.description"
                  :selected="selected"
                />
                <span v-else class="text-sm text-gray-700 dark:text-gray-300">
                  {{ isGroupSelectOption(option) ? option.label : '' }}
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
                v-if="isGroupSelectOption(option) && option.platform"
                :name="option.name"
                :platform="option.platform"
                :show-rate="false"
              />
              <span v-else>{{ isGroupSelectOption(option) ? option.label : '' }}</span>
            </template>
            <template #option="{ option, selected }">
              <GroupOptionItem
                v-if="isGroupSelectOption(option) && option.platform"
                :name="option.name"
                :platform="option.platform"
                :description="option.description"
                :selected="selected"
              />
              <span v-else class="text-sm text-gray-700 dark:text-gray-300">
                {{ isGroupSelectOption(option) ? option.label : '' }}
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
                          :class="{ 'opacity-50': rule.accounts.some(a => a.id === account.id) }"
                          :disabled="rule.accounts.some(a => a.id === account.id)"
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
                :platform="option.platform"
                :label="option.label"
              />
            </template>
            <template #option="{ option }">
              <PlatformLabel
                v-if="isPlatformSelectOption(option)"
                :platform="option.platform"
                :label="option.label"
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
                @click="editForm.copy_accounts_from_group_ids = editForm.copy_accounts_from_group_ids.filter(id => id !== groupId)"
                class="ml-0.5 text-primary-500 hover:text-primary-700 dark:hover:text-primary-200"
              >
                <Icon name="x" size="xs" />
              </button>
            </span>
          </div>
          <Select
            v-model="editCopyAccountsSelection"
            :options="copyAccountsGroupSelectOptionsForEdit.filter(option => !editForm.copy_accounts_from_group_ids.includes(option.value as number))"
            :placeholder="t('admin.groups.copyAccounts.selectPlaceholder')"
            @change="handleEditCopyAccountsSelect"
          >
            <template #option="{ option, selected }">
              <GroupOptionItem
                v-if="isGroupSelectOption(option)"
                :name="option.name"
                :platform="option.platform"
                :description="option.description"
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
                  v-if="isGroupSelectOption(option) && option.platform"
                  :name="option.name"
                  :platform="option.platform"
                  :show-rate="false"
                />
                <span v-else>{{ isGroupSelectOption(option) ? option.label : '' }}</span>
              </template>
              <template #option="{ option, selected }">
                <GroupOptionItem
                  v-if="isGroupSelectOption(option) && option.platform"
                  :name="option.name"
                  :platform="option.platform"
                  :description="option.description"
                  :selected="selected"
                />
                <span v-else class="text-sm text-gray-700 dark:text-gray-300">
                  {{ isGroupSelectOption(option) ? option.label : '' }}
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
                v-if="isGroupSelectOption(option) && option.platform"
                :name="option.name"
                :platform="option.platform"
                :show-rate="false"
              />
              <span v-else>{{ isGroupSelectOption(option) ? option.label : '' }}</span>
            </template>
            <template #option="{ option, selected }">
              <GroupOptionItem
                v-if="isGroupSelectOption(option) && option.platform"
                :name="option.name"
                :platform="option.platform"
                :description="option.description"
                :selected="selected"
              />
              <span v-else class="text-sm text-gray-700 dark:text-gray-300">
                {{ isGroupSelectOption(option) ? option.label : '' }}
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
                          :class="{ 'opacity-50': rule.accounts.some(a => a.id === account.id) }"
                          :disabled="rule.accounts.some(a => a.id === account.id)"
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

    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('admin.groups.deleteGroup')"
      :message="deleteConfirmMessage"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />

    <BaseDialog
      :show="showSortModal"
      :title="t('admin.groups.sortOrder')"
      width="normal"
      @close="closeSortModal"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.groups.sortOrderHint') }}
        </p>
        <VueDraggable
          v-model="sortableGroups"
          :animation="200"
          class="space-y-2"
        >
          <div
            v-for="group in sortableGroups"
            :key="group.id"
            class="flex cursor-grab items-center gap-3 rounded-lg border border-gray-200 bg-white p-3 transition-shadow hover:shadow-md active:cursor-grabbing dark:border-dark-600 dark:bg-dark-700"
          >
            <div class="text-gray-400">
              <Icon name="menu" size="md" />
            </div>
            <div class="flex-1">
              <div class="font-medium text-gray-900 dark:text-white">{{ group.name }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                <span
                  :class="[
                    'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium',
                    group.platform === 'anthropic'
                      ? 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
                      : group.platform === 'kiro'
                        ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
                      : group.platform === 'openai'
                        ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                        : group.platform === 'copilot'
                          ? 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400'
                        : group.platform === 'grok'
                          ? 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-200'
                        : group.platform === 'deepseek'
                          ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400'
                        : group.platform === 'antigravity'
                          ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
                          : 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                  ]"
                >
                  {{ t('admin.groups.platforms.' + group.platform) }}
                </span>
              </div>
            </div>
            <div class="text-sm text-gray-400">
              #{{ group.id }}
            </div>
          </div>
        </VueDraggable>
      </div>

      <template #footer>
        <div class="flex justify-end gap-3 pt-4">
          <button @click="closeSortModal" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button
            @click="saveSortOrder"
            :disabled="sortSubmitting"
            class="btn btn-primary"
          >
            <svg
              v-if="sortSubmitting"
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
            {{ sortSubmitting ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <GroupRateMultipliersModal
      :show="showRateMultipliersModal"
      :group="rateMultipliersGroup"
      @close="showRateMultipliersModal = false"
      @success="loadGroups"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useOnboardingStore } from '@/stores/onboarding'
import { adminAPI } from '@/api/admin'
import type {
  AdminGroup,
  GroupPlatform,
  OpenAIGroupImageProtocolMode,
  SubscriptionType
} from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select from '@/components/common/Select.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import PlatformLabel from '@/components/common/PlatformLabel.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupRateMultipliersModal from '@/components/admin/group/GroupRateMultipliersModal.vue'
import GroupCapacityBadge from '@/components/common/GroupCapacityBadge.vue'
import { VueDraggable } from 'vue-draggable-plus'
import { createStableObjectKeyResolver } from '@/utils/stableObjectKey'
import { useKeyedDebouncedSearch } from '@/composables/useKeyedDebouncedSearch'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'

const { t } = useI18n()
const appStore = useAppStore()
const onboardingStore = useOnboardingStore()

const columns = computed<Column[]>(() => [
  { key: 'name', label: t('admin.groups.columns.name'), sortable: true },
  { key: 'priority', label: t('admin.groups.columns.priority'), sortable: true },
  { key: 'platform', label: t('admin.groups.columns.platform'), sortable: true },
  { key: 'billing_type', label: t('admin.groups.columns.billingType'), sortable: true },
  { key: 'rate_multiplier', label: t('admin.groups.columns.rateMultiplier'), sortable: true },
  { key: 'is_exclusive', label: t('admin.groups.columns.type'), sortable: true },
  { key: 'account_count', label: t('admin.groups.columns.accounts'), sortable: true },
  { key: 'capacity', label: t('admin.groups.columns.capacity'), sortable: false },
  { key: 'usage', label: t('admin.groups.columns.usage'), sortable: false },
  { key: 'status', label: t('admin.groups.columns.status'), sortable: true },
  { key: 'actions', label: t('admin.groups.columns.actions'), sortable: false }
])

interface PlatformSelectOption {
  value: GroupPlatform | ''
  label: string
  platform?: GroupPlatform
  [key: string]: unknown
}

interface GroupSelectOption {
  value: number | null
  label: string
  name: string
  platform?: GroupPlatform
  description?: string
  [key: string]: unknown
}

function normalizeOpenAIGroupImageProtocolMode(value: unknown): OpenAIGroupImageProtocolMode {
  const normalized = String(value || '').trim().toLowerCase()
  if (normalized === 'native' || normalized === 'compat') {
    return normalized
  }
  return 'inherit'
}

function buildPlatformSelectOption(platform: GroupPlatform): PlatformSelectOption {
  return {
    value: platform,
    label: t(`admin.groups.platforms.${platform}`),
    platform
  }
}

function buildGroupSelectOption(group: AdminGroup, description?: string): GroupSelectOption {
  return {
    value: group.id,
    label: group.name,
    name: group.name,
    platform: group.platform,
    description
  }
}

function buildEmptyGroupSelectOption(label: string): GroupSelectOption {
  return {
    value: null,
    label,
    name: label
  }
}

function buildCopyAccountsDescription(group: AdminGroup): string {
  return `${group.account_count || 0} ${t('admin.groups.accountsTotal')}`
}

function findGroupSelectOption(
  options: GroupSelectOption[],
  groupID: number
): GroupSelectOption | null {
  return options.find((option) => option.value === groupID) || null
}

function isPlatformSelectOption(option: unknown): option is PlatformSelectOption {
  return typeof option === 'object' && option !== null && 'value' in option && 'label' in option
}

function isGroupSelectOption(option: unknown): option is GroupSelectOption {
  return typeof option === 'object' && option !== null && 'value' in option && 'label' in option
}

const statusOptions = computed(() => [
  { value: '', label: t('admin.groups.allStatus') },
  { value: 'active', label: t('admin.accounts.status.active') },
  { value: 'inactive', label: t('admin.accounts.status.inactive') }
])

const exclusiveOptions = computed(() => [
  { value: '', label: t('admin.groups.allGroups') },
  { value: 'true', label: t('admin.groups.exclusive') },
  { value: 'false', label: t('admin.groups.nonExclusive') }
])

const platformOptions = computed<PlatformSelectOption[]>(() => [
  buildPlatformSelectOption('anthropic'),
  buildPlatformSelectOption('kiro'),
  buildPlatformSelectOption('openai'),
  buildPlatformSelectOption('copilot'),
  buildPlatformSelectOption('grok'),
  buildPlatformSelectOption('deepseek'),
  buildPlatformSelectOption('gemini'),
  buildPlatformSelectOption('antigravity'),
  buildPlatformSelectOption('baidu_document_ai')
])

const platformFilterOptions = computed<PlatformSelectOption[]>(() => [
  { value: '', label: t('admin.groups.allPlatforms') },
  ...platformOptions.value
])

const editStatusOptions = computed(() => [
  { value: 'active', label: t('admin.accounts.status.active') },
  { value: 'inactive', label: t('admin.accounts.status.inactive') }
])

const subscriptionTypeOptions = computed(() => [
  { value: 'standard', label: t('admin.groups.subscription.standard') },
  { value: 'subscription', label: t('admin.groups.subscription.subscription') }
])

const openAIGroupImageProtocolModeOptions = computed(() => [
  { value: 'inherit', label: t('admin.groups.imageProtocol.options.inherit') },
  { value: 'native', label: t('admin.groups.imageProtocol.options.native') },
  { value: 'compat', label: t('admin.groups.imageProtocol.options.compat') }
])

const fallbackGroupOptions = computed<GroupSelectOption[]>(() => {
  const options: GroupSelectOption[] = [
    buildEmptyGroupSelectOption(t('admin.groups.claudeCode.noFallback'))
  ]
  const eligibleGroups = groups.value.filter(
    (g) => g.platform === 'anthropic' && !g.claude_code_only && g.status === 'active'
  )
  eligibleGroups.forEach((g) => {
    options.push(buildGroupSelectOption(g))
  })
  return options
})


const fallbackGroupOptionsForEdit = computed<GroupSelectOption[]>(() => {
  const options: GroupSelectOption[] = [
    buildEmptyGroupSelectOption(t('admin.groups.claudeCode.noFallback'))
  ]
  const currentId = editingGroup.value?.id
  const eligibleGroups = groups.value.filter(
    (g) => g.platform === 'anthropic' && !g.claude_code_only && g.status === 'active' && g.id !== currentId
  )
  eligibleGroups.forEach((g) => {
    options.push(buildGroupSelectOption(g))
  })
  return options
})

const invalidRequestFallbackOptions = computed<GroupSelectOption[]>(() => {
  const options: GroupSelectOption[] = [
    buildEmptyGroupSelectOption(t('admin.groups.invalidRequestFallback.noFallback'))
  ]
  const eligibleGroups = groups.value.filter(
    (g) =>
      g.platform === 'anthropic' &&
      g.status === 'active' &&
      g.subscription_type !== 'subscription' &&
      g.fallback_group_id_on_invalid_request === null
  )
  eligibleGroups.forEach((g) => {
    options.push(buildGroupSelectOption(g))
  })
  return options
})


const invalidRequestFallbackOptionsForEdit = computed<GroupSelectOption[]>(() => {
  const options: GroupSelectOption[] = [
    buildEmptyGroupSelectOption(t('admin.groups.invalidRequestFallback.noFallback'))
  ]
  const currentId = editingGroup.value?.id
  const eligibleGroups = groups.value.filter(
    (g) =>
      g.platform === 'anthropic' &&
      g.status === 'active' &&
      g.subscription_type !== 'subscription' &&
      g.fallback_group_id_on_invalid_request === null &&
      g.id !== currentId
  )
  eligibleGroups.forEach((g) => {
    options.push(buildGroupSelectOption(g))
  })
  return options
})


const copyAccountsGroupSelectOptions = computed<GroupSelectOption[]>(() => {
  const eligibleGroups = groups.value.filter(
    (g) => g.platform === createForm.platform && (g.account_count || 0) > 0
  )
  return eligibleGroups.map((g) => buildGroupSelectOption(g, buildCopyAccountsDescription(g)))
})

const copyAccountsGroupSelectOptionsForEdit = computed<GroupSelectOption[]>(() => {
  const currentId = editingGroup.value?.id
  const eligibleGroups = groups.value.filter(
    (g) => g.platform === editForm.platform && (g.account_count || 0) > 0 && g.id !== currentId
  )
  return eligibleGroups.map((g) => buildGroupSelectOption(g, buildCopyAccountsDescription(g)))
})

const groups = ref<AdminGroup[]>([])
const loading = ref(false)
const usageMap = ref<Map<number, { today_cost: number; total_cost: number }>>(new Map())
const usageLoading = ref(false)
const capacityMap = ref<Map<number, { concurrencyUsed: number; concurrencyMax: number; sessionsUsed: number; sessionsMax: number; rpmUsed: number; rpmMax: number }>>(new Map())
const searchQuery = ref('')
const filters = reactive({
  platform: '',
  status: '',
  is_exclusive: ''
})
const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})

let abortController: AbortController | null = null

const showCreateModal = ref(false)
const showEditModal = ref(false)
const showDeleteDialog = ref(false)
const showSortModal = ref(false)
const submitting = ref(false)
const sortSubmitting = ref(false)
const editingGroup = ref<AdminGroup | null>(null)
const deletingGroup = ref<AdminGroup | null>(null)
const showRateMultipliersModal = ref(false)
const rateMultipliersGroup = ref<AdminGroup | null>(null)
const sortableGroups = ref<AdminGroup[]>([])
const createCopyAccountsSelection = ref<number | null>(null)
const editCopyAccountsSelection = ref<number | null>(null)

const createForm = reactive({
  name: '',
  description: '',
  platform: 'anthropic' as GroupPlatform,
  priority: 1,
  rate_multiplier: 1.0,
  is_exclusive: false,
  gemini_mixed_protocol_enabled: false,
  subscription_type: 'standard' as SubscriptionType,
  daily_limit_usd: null as number | null,
  weekly_limit_usd: null as number | null,
  monthly_limit_usd: null as number | null,
  image_price_1k: null as number | null,
  image_price_2k: null as number | null,
  image_price_4k: null as number | null,
  image_protocol_mode: 'inherit' as OpenAIGroupImageProtocolMode,
  claude_code_only: false,
  fallback_group_id: null as number | null,
  fallback_group_id_on_invalid_request: null as number | null,
  allow_messages_dispatch: false,
  default_mapped_model: 'gpt-5.4',
  model_routing_enabled: false,
  supported_model_scopes: ['claude', 'gemini_text', 'gemini_image'] as string[],
  mcp_xml_inject: true,
  copy_accounts_from_group_ids: [] as number[]
})

interface SimpleAccount {
  id: number
  name: string
}


interface ModelRoutingRule {
  pattern: string
  accounts: SimpleAccount[]
}

const createModelRoutingRules = ref<ModelRoutingRule[]>([])

const editModelRoutingRules = ref<ModelRoutingRule[]>([])

const resolveCreateRuleKey = createStableObjectKeyResolver<ModelRoutingRule>('create-rule')
const resolveEditRuleKey = createStableObjectKeyResolver<ModelRoutingRule>('edit-rule')

const getCreateRuleRenderKey = (rule: ModelRoutingRule) => resolveCreateRuleKey(rule)
const getEditRuleRenderKey = (rule: ModelRoutingRule) => resolveEditRuleKey(rule)

const getCreateRuleSearchKey = (rule: ModelRoutingRule) => `create-${resolveCreateRuleKey(rule)}`
const getEditRuleSearchKey = (rule: ModelRoutingRule) => `edit-${resolveEditRuleKey(rule)}`

const getRuleSearchKey = (rule: ModelRoutingRule, isEdit: boolean = false) => {
  return isEdit ? getEditRuleSearchKey(rule) : getCreateRuleSearchKey(rule)
}

const accountSearchKeyword = ref<Record<string, string>>({})
const accountSearchResults = ref<Record<string, SimpleAccount[]>>({})
const showAccountDropdown = ref<Record<string, boolean>>({})

const clearAccountSearchStateByKey = (key: string) => {
  delete accountSearchKeyword.value[key]
  delete accountSearchResults.value[key]
  delete showAccountDropdown.value[key]
}

const clearAllAccountSearchState = () => {
  accountSearchKeyword.value = {}
  accountSearchResults.value = {}
  showAccountDropdown.value = {}
}

const accountSearchRunner = useKeyedDebouncedSearch<SimpleAccount[]>({
  delay: 300,
  search: async (keyword, { signal }) => {
    const res = await adminAPI.accounts.list(
      1,
      20,
      {
        search: keyword,
        platform: 'anthropic'
      },
      { signal }
    )
    return res.items.map((account) => ({ id: account.id, name: account.name }))
  },
  onSuccess: (key, result) => {
    accountSearchResults.value[key] = result
  },
  onError: (key) => {
    accountSearchResults.value[key] = []
  }
})

const searchAccounts = (key: string) => {
  accountSearchRunner.trigger(key, accountSearchKeyword.value[key] || '')
}

const searchAccountsByRule = (rule: ModelRoutingRule, isEdit: boolean = false) => {
  searchAccounts(getRuleSearchKey(rule, isEdit))
}


const selectAccount = (rule: ModelRoutingRule, account: SimpleAccount, isEdit: boolean = false) => {
  if (!rule) return


  if (!rule.accounts.some(a => a.id === account.id)) {
    rule.accounts.push(account)
  }


  const key = getRuleSearchKey(rule, isEdit)
  accountSearchKeyword.value[key] = ''
  showAccountDropdown.value[key] = false
}

const removeSelectedAccount = (rule: ModelRoutingRule, accountId: number, _isEdit: boolean = false) => {
  if (!rule) return

  rule.accounts = rule.accounts.filter(a => a.id !== accountId)
}


const toggleCreateScope = (scope: string) => {
  const idx = createForm.supported_model_scopes.indexOf(scope)
  if (idx === -1) {
    createForm.supported_model_scopes.push(scope)
  } else {
    createForm.supported_model_scopes.splice(idx, 1)
  }
}


const toggleEditScope = (scope: string) => {
  const idx = editForm.supported_model_scopes.indexOf(scope)
  if (idx === -1) {
    editForm.supported_model_scopes.push(scope)
  } else {
    editForm.supported_model_scopes.splice(idx, 1)
  }
}

const onAccountSearchFocus = (rule: ModelRoutingRule, isEdit: boolean = false) => {
  const key = getRuleSearchKey(rule, isEdit)
  showAccountDropdown.value[key] = true
  if (!accountSearchResults.value[key]?.length) {
    searchAccounts(key)
  }
}

const addCreateRoutingRule = () => {
  createModelRoutingRules.value.push({ pattern: '', accounts: [] })
}

const removeCreateRoutingRule = (rule: ModelRoutingRule) => {
  const index = createModelRoutingRules.value.indexOf(rule)
  if (index === -1) return

  const key = getCreateRuleSearchKey(rule)
  accountSearchRunner.clearKey(key)
  clearAccountSearchStateByKey(key)
  createModelRoutingRules.value.splice(index, 1)
}

const addEditRoutingRule = () => {
  editModelRoutingRules.value.push({ pattern: '', accounts: [] })
}

const removeEditRoutingRule = (rule: ModelRoutingRule) => {
  const index = editModelRoutingRules.value.indexOf(rule)
  if (index === -1) return

  const key = getEditRuleSearchKey(rule)
  accountSearchRunner.clearKey(key)
  clearAccountSearchStateByKey(key)
  editModelRoutingRules.value.splice(index, 1)
}


const convertRoutingRulesToApiFormat = (rules: ModelRoutingRule[]): Record<string, number[]> | null => {
  const result: Record<string, number[]> = {}
  let hasValidRules = false

  for (const rule of rules) {
    const pattern = rule.pattern.trim()
    if (!pattern) continue

    const accountIds = rule.accounts.map(a => a.id).filter(id => id > 0)

    if (accountIds.length > 0) {
      result[pattern] = accountIds
      hasValidRules = true
    }
  }

  return hasValidRules ?
 result : null
}


const convertApiFormatToRoutingRules = async (apiFormat: Record<string, number[]> | null): Promise<ModelRoutingRule[]> => {
  if (!apiFormat) return []

  const rules: ModelRoutingRule[] = []
  for (const [pattern, accountIds] of Object.entries(apiFormat)) {
    const accounts: SimpleAccount[] = []
    for (const id of accountIds) {
      try {
        const account = await adminAPI.accounts.getById(id)
        accounts.push({ id: account.id, name: account.name })
      } catch {
        accounts.push({ id, name: `#${id}` })
      }
    }
    rules.push({ pattern, accounts })
  }
  return rules
}

const editForm = reactive({
  name: '',
  description: '',
  platform: 'anthropic' as GroupPlatform,
  priority: 1,
  rate_multiplier: 1.0,
  is_exclusive: false,
  gemini_mixed_protocol_enabled: false,
  status: 'active' as 'active' | 'inactive',
  subscription_type: 'standard' as SubscriptionType,
  daily_limit_usd: null as number | null,
  weekly_limit_usd: null as number | null,
  monthly_limit_usd: null as number | null,
  image_price_1k: null as number | null,
  image_price_2k: null as number | null,
  image_price_4k: null as number | null,
  image_protocol_mode: 'inherit' as OpenAIGroupImageProtocolMode,
  claude_code_only: false,
  fallback_group_id: null as number | null,
  fallback_group_id_on_invalid_request: null as number | null,
  allow_messages_dispatch: false,
  default_mapped_model: '',
  model_routing_enabled: false,
  supported_model_scopes: ['claude', 'gemini_text', 'gemini_image'] as string[],
  mcp_xml_inject: true,
  copy_accounts_from_group_ids: [] as number[]
})

const deleteConfirmMessage = computed(() => {
  if (!deletingGroup.value) {
    return ''
  }
  if (deletingGroup.value.subscription_type === 'subscription') {
    return t('admin.groups.deleteConfirmSubscription', { name: deletingGroup.value.name })
  }
  return t('admin.groups.deleteConfirm', { name: deletingGroup.value.name })
})

const loadGroups = async () => {
  if (abortController) {
    abortController.abort()
  }
  const currentController = new AbortController()
  abortController = currentController
  const { signal } = currentController
  loading.value = true
  try {
    const response = await adminAPI.groups.list(pagination.page, pagination.page_size, {
      platform: (filters.platform as GroupPlatform) || undefined,
      status: filters.status as any,
      is_exclusive: filters.is_exclusive ? filters.is_exclusive === 'true' : undefined,
      search: searchQuery.value.trim() || undefined
    }, { signal })
    if (signal.aborted) return
    groups.value = response.items
    pagination.total = response.total
    pagination.pages = response.pages
    loadUsageSummary()
    loadCapacitySummary()
  } catch (error: any) {
    if (signal.aborted || error?.name === 'AbortError' || error?.code === 'ERR_CANCELED') {
      return
    }
    appStore.showError(t('admin.groups.failedToLoad'))
    console.error('Error loading groups:', error)
  } finally {
    if (abortController === currentController && !signal.aborted) {
      loading.value = false
    }
  }
}

const formatCost = (cost: number): string => {
  if (cost >= 1000) return cost.toFixed(0)
  if (cost >= 100) return cost.toFixed(1)
  return cost.toFixed(2)
}

const getGroupAvailableAccounts = (group: AdminGroup): number => {
  return Math.max((group.active_account_count || 0) - (group.rate_limited_account_count || 0), 0)
}

const getGroupDigitCount = (group: AdminGroup): number => {
  return Math.max(String(group.account_count || 0).length, 1)
}

const formatGroupAccountValue = (value: number, group: AdminGroup): string => {
  return String(Math.max(value, 0)).padStart(getGroupDigitCount(group), '0')
}

const handleCreatePlatformChange = () => {
  createForm.copy_accounts_from_group_ids = []
  createCopyAccountsSelection.value = null
}

const handleCreateCopyAccountsSelect = (value: string | number | boolean | null) => {
  if (typeof value === 'number' && !createForm.copy_accounts_from_group_ids.includes(value)) {
    createForm.copy_accounts_from_group_ids.push(value)
  }
  createCopyAccountsSelection.value = null
}

const handleEditCopyAccountsSelect = (value: string | number | boolean | null) => {
  if (typeof value === 'number' && !editForm.copy_accounts_from_group_ids.includes(value)) {
    editForm.copy_accounts_from_group_ids.push(value)
  }
  editCopyAccountsSelection.value = null
}

const normalizeGroupPriority = (value: number | null | undefined): number => {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? Math.floor(parsed) : 1
}

const loadUsageSummary = async () => {
  usageLoading.value = true
  try {
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone
    const data = await adminAPI.groups.getUsageSummary(tz)
    const map = new Map<number, { today_cost: number; total_cost: number }>()
    for (const item of data) {
      map.set(item.group_id, { today_cost: item.today_cost, total_cost: item.total_cost })
    }
    usageMap.value = map
  } catch (error) {
    console.error('Error loading group usage summary:', error)
  } finally {
    usageLoading.value = false
  }
}

const loadCapacitySummary = async () => {
  try {
    const data = await adminAPI.groups.getCapacitySummary()
    const map = new Map<number, { concurrencyUsed: number; concurrencyMax: number; sessionsUsed: number; sessionsMax: number; rpmUsed: number; rpmMax: number }>()
    for (const item of data) {
      map.set(item.group_id, {
        concurrencyUsed: item.concurrency_used,
        concurrencyMax: item.concurrency_max,
        sessionsUsed: item.sessions_used,
        sessionsMax: item.sessions_max,
        rpmUsed: item.rpm_used,
        rpmMax: item.rpm_max
      })
    }
    capacityMap.value = map
  } catch (error) {
    console.error('Error loading group capacity summary:', error)
  }
}

let searchTimeout: ReturnType<typeof setTimeout>
const handleSearch = () => {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    pagination.page = 1
    loadGroups()
  }, 300)
}

const handlePageChange = (page: number) => {
  pagination.page = page
  loadGroups()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.page_size = pageSize
  pagination.page = 1
  loadGroups()
}

const closeCreateModal = () => {
  showCreateModal.value = false
  createModelRoutingRules.value.forEach((rule) => {
    accountSearchRunner.clearKey(getCreateRuleSearchKey(rule))
  })
  clearAllAccountSearchState()
  createForm.name = ''
  createForm.description = ''
  createForm.platform = 'anthropic'
  createForm.priority = 1
  createForm.rate_multiplier = 1.0
  createForm.is_exclusive = false
  createForm.gemini_mixed_protocol_enabled = false
  createForm.subscription_type = 'standard'
  createForm.daily_limit_usd = null
  createForm.weekly_limit_usd = null
  createForm.monthly_limit_usd = null
  createForm.image_price_1k = null
  createForm.image_price_2k = null
  createForm.image_price_4k = null
  createForm.image_protocol_mode = 'inherit'
  createForm.claude_code_only = false
  createForm.fallback_group_id = null
  createForm.fallback_group_id_on_invalid_request = null
  createForm.allow_messages_dispatch = false
  createForm.default_mapped_model = 'gpt-5.4'
  createForm.supported_model_scopes = ['claude', 'gemini_text', 'gemini_image']
  createForm.mcp_xml_inject = true
  createForm.copy_accounts_from_group_ids = []
  createCopyAccountsSelection.value = null
  createModelRoutingRules.value = []
}

const normalizeOptionalLimit = (value: number | string | null | undefined): number | null => {
  if (value === null || value === undefined) {
    return null
  }

  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed) {
      return null
    }
    const parsed = Number(trimmed)
    return Number.isFinite(parsed) && parsed > 0 ?
 parsed : null
  }

  return Number.isFinite(value) && value > 0 ?
 value : null
}

const handleCreateGroup = async () => {
  if (!createForm.name.trim()) {
    appStore.showError(t('admin.groups.nameRequired'))
    return
  }
  submitting.value = true
  try {
    const requestData = {
      ...createForm,
      priority: normalizeGroupPriority(createForm.priority),
      daily_limit_usd: normalizeOptionalLimit(createForm.daily_limit_usd as number | string | null),
      weekly_limit_usd: normalizeOptionalLimit(createForm.weekly_limit_usd as number | string | null),
      monthly_limit_usd: normalizeOptionalLimit(createForm.monthly_limit_usd as number | string | null),
      model_routing: convertRoutingRulesToApiFormat(createModelRoutingRules.value)
    }

    const emptyToNull = (v: any) => v === '' ?
 null : v
    requestData.daily_limit_usd = emptyToNull(requestData.daily_limit_usd)
    requestData.weekly_limit_usd = emptyToNull(requestData.weekly_limit_usd)
    requestData.monthly_limit_usd = emptyToNull(requestData.monthly_limit_usd)
    await adminAPI.groups.create(requestData)
    appStore.showSuccess(t('admin.groups.groupCreated'))
    closeCreateModal()
    loadGroups()
    if (onboardingStore.isCurrentStep('[data-tour="group-form-submit"]')) {
      onboardingStore.nextStep(500)
    }
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.groups.failedToCreate'))
    console.error('Error creating group:', error)
  } finally {
    submitting.value = false
  }
}

const handleEdit = async (group: AdminGroup) => {
  editingGroup.value = group
  editForm.name = group.name
  editForm.description = group.description || ''
  editForm.platform = group.platform
  editForm.priority = group.priority ?? 1
  editForm.rate_multiplier = group.rate_multiplier
  editForm.is_exclusive = group.is_exclusive
  editForm.gemini_mixed_protocol_enabled = group.gemini_mixed_protocol_enabled || false
  editForm.status = group.status
  editForm.subscription_type = group.subscription_type || 'standard'
  editForm.daily_limit_usd = group.daily_limit_usd
  editForm.weekly_limit_usd = group.weekly_limit_usd
  editForm.monthly_limit_usd = group.monthly_limit_usd
  editForm.image_price_1k = group.image_price_1k
  editForm.image_price_2k = group.image_price_2k
  editForm.image_price_4k = group.image_price_4k
  editForm.image_protocol_mode = normalizeOpenAIGroupImageProtocolMode(group.image_protocol_mode)
  editForm.claude_code_only = group.claude_code_only || false
  editForm.fallback_group_id = group.fallback_group_id
  editForm.fallback_group_id_on_invalid_request = group.fallback_group_id_on_invalid_request
  editForm.allow_messages_dispatch = group.allow_messages_dispatch || false
  editForm.default_mapped_model = group.default_mapped_model || ''
  editForm.model_routing_enabled = group.model_routing_enabled || false
  editForm.supported_model_scopes = group.supported_model_scopes || ['claude', 'gemini_text', 'gemini_image']
  editForm.mcp_xml_inject = group.mcp_xml_inject ?? true
  editForm.copy_accounts_from_group_ids = []
  editModelRoutingRules.value = await convertApiFormatToRoutingRules(group.model_routing)
  editCopyAccountsSelection.value = null
  showEditModal.value = true
}

const closeEditModal = () => {
  editModelRoutingRules.value.forEach((rule) => {
    accountSearchRunner.clearKey(getEditRuleSearchKey(rule))
  })
  clearAllAccountSearchState()
  showEditModal.value = false
  editingGroup.value = null
  editModelRoutingRules.value = []
  editForm.copy_accounts_from_group_ids = []
  editCopyAccountsSelection.value = null
  editForm.gemini_mixed_protocol_enabled = false
  editForm.image_protocol_mode = 'inherit'
}

const handleUpdateGroup = async () => {
  if (!editingGroup.value) return
  if (!editForm.name.trim()) {
    appStore.showError(t('admin.groups.nameRequired'))
    return
  }

  submitting.value = true
  try {
    const payload = {
      ...editForm,
      priority: normalizeGroupPriority(editForm.priority),
      daily_limit_usd: normalizeOptionalLimit(editForm.daily_limit_usd as number | string | null),
      weekly_limit_usd: normalizeOptionalLimit(editForm.weekly_limit_usd as number | string | null),
      monthly_limit_usd: normalizeOptionalLimit(editForm.monthly_limit_usd as number | string | null),
      fallback_group_id: editForm.fallback_group_id === null ? 0 : editForm.fallback_group_id,
      fallback_group_id_on_invalid_request:
        editForm.fallback_group_id_on_invalid_request === null
          ? 0
          : editForm.fallback_group_id_on_invalid_request,
      model_routing: convertRoutingRulesToApiFormat(editModelRoutingRules.value)
    }

    const emptyToNull = (v: any) => v === '' ?
 null : v
    payload.daily_limit_usd = emptyToNull(payload.daily_limit_usd)
    payload.weekly_limit_usd = emptyToNull(payload.weekly_limit_usd)
    payload.monthly_limit_usd = emptyToNull(payload.monthly_limit_usd)
    await adminAPI.groups.update(editingGroup.value.id, payload)
    appStore.showSuccess(t('admin.groups.groupUpdated'))
    closeEditModal()
    loadGroups()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.groups.failedToUpdate'))
    console.error('Error updating group:', error)
  } finally {
    submitting.value = false
  }
}

const handleRateMultipliers = (group: AdminGroup) => {
  rateMultipliersGroup.value = group
  showRateMultipliersModal.value = true
}

const handleDelete = (group: AdminGroup) => {
  deletingGroup.value = group
  showDeleteDialog.value = true
}

const confirmDelete = async () => {
  if (!deletingGroup.value) return

  try {
    await adminAPI.groups.delete(deletingGroup.value.id)
    appStore.showSuccess(t('admin.groups.groupDeleted'))
    showDeleteDialog.value = false
    deletingGroup.value = null
    loadGroups()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.groups.failedToDelete'))
    console.error('Error deleting group:', error)
  }
}

watch(
  () => createForm.subscription_type,
  (newVal) => {
    if (newVal === 'subscription') {
      createForm.is_exclusive = true
      createForm.fallback_group_id_on_invalid_request = null
    }
  }
)

watch(
  () => createForm.platform,
  (newVal) => {
    if (newVal !== 'gemini') {
      createForm.gemini_mixed_protocol_enabled = false
    }
    if (!['anthropic', 'antigravity'].includes(newVal)) {
      createForm.fallback_group_id_on_invalid_request = null
    }
    if (newVal !== 'openai') {
      createForm.image_protocol_mode = 'inherit'
      createForm.allow_messages_dispatch = false
      createForm.default_mapped_model = ''
    }
  }
)

watch(
  () => editForm.platform,
  (newVal) => {
    if (newVal !== 'gemini') {
      editForm.gemini_mixed_protocol_enabled = false
    }
    if (newVal !== 'openai') {
      editForm.image_protocol_mode = 'inherit'
      editForm.allow_messages_dispatch = false
      editForm.default_mapped_model = ''
    }
  }
)

const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement

  if (!target.closest('.account-search-container')) {
    Object.keys(showAccountDropdown.value).forEach(key => {
      showAccountDropdown.value[key] = false
    })
  }
}


const openSortModal = async () => {
  try {

    const allGroups = await adminAPI.groups.getAll()
    sortableGroups.value = [...allGroups].sort((a, b) => a.sort_order - b.sort_order)
    showSortModal.value = true
  } catch (error) {
    appStore.showError(t('admin.groups.failedToLoad'))
    console.error('Error loading groups for sorting:', error)
  }
}


const closeSortModal = () => {
  showSortModal.value = false
  sortableGroups.value = []
}


const saveSortOrder = async () => {
  sortSubmitting.value = true
  try {
    const updates = sortableGroups.value.map((g, index) => ({
      id: g.id,
      sort_order: index * 10
    }))
    await adminAPI.groups.updateSortOrder(updates)
    appStore.showSuccess(t('admin.groups.sortOrderUpdated'))
    closeSortModal()
    loadGroups()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.groups.failedToUpdateSortOrder'))
    console.error('Error updating sort order:', error)
  } finally {
    sortSubmitting.value = false
  }
}

onMounted(() => {
  loadGroups()
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  accountSearchRunner.clearAll()
  clearAllAccountSearchState()
})
</script>
