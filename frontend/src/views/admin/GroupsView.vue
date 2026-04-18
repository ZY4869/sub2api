<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col justify-between gap-4 lg:flex-row lg:items-start">
          <!-- Left: fuzzy search + filters (can wrap to multiple lines) -->
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

          <!-- Right: actions -->
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
              <!-- Type Badge -->
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
              <!-- Subscription Limits - compact single line -->
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

    <!-- Create Group Modal -->
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
        <!-- 濠电姷鏁告慨鐑藉极閹间礁纾绘繛鎴欏焺閺佸銇勯幘璺烘瀾闁告瑥绻橀幃妤€鈽夊▎娆庣返濠电偛鐗呯划娆撳蓟閻斿吋鈷掗悗闈涘濡差噣姊洪幖鐐插闁稿﹤娼￠悰顕€寮介妸锔剧Ф闂佸憡鎸嗛崟顐¤繕缂傚倷鑳堕崑鎾诲磿閹惰棄围闁归棿绀侀拑鐔哥箾閹存瑥鐏╅柛妤佸▕閺屾洘绻涢崹顔煎闂佺厧澹婃禍婊堝煘閹达箑鐒洪柛鎰╁妿缁佸嘲顪冮妶搴″箻闁稿繑锚椤曪絿鎷犲ù瀣潔闂侀潧绻掓慨鐑筋敊婵犲洦鈷戦悷娆忓閸斻倝鏌涢悢绋款嚋闁逛究鍔戝畷銊︾節閸曨厾妲囬梻渚€娼ф蹇曞緤閸撗勫厹濡わ絽鍟崐鍨叏濮楀棗骞楃紒鑸电叀閺?-->
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
          <!-- 闂傚倸鍊峰ù鍥敋瑜嶉湁闁绘垼妫勭粻鐘绘煙閹规劦鍤欑紒鐘靛枛閺屻劑鎮㈤崫鍕戯綁鏌涚€ｎ亜鈧潡寮婚妸鈺傚亜闁告繂瀚呴姀銈嗙厽闁圭儤鍨规禒娑㈡煏閸パ冾伃妤犵偞甯″畷鍗烆渻閹屾婵犵绱曢崑鎴﹀磹瑜忛埀顒勬涧閻倸鐣烽崷顓熷磯闁惧繗顫夊▓楣冩煟閻斿摜鎳冮悗姘煎弮瀹曟洟鎮㈤崗鑲╁弳濠电娀娼уΛ娆忣啅濠靛鐓ユ繛鎴炵懅閻帗鎱ㄦ繝鍐┿仢鐎规洏鍔嶇换婵嬪礃椤垶袩闂傚倷绀侀幉锟犲箰婵犳碍鍎庢い鏍ㄦ皑閺嗭妇鎲搁悧鍫濈瑲闁搞倕鍟撮弻宥夊传閸曨偅娈堕梺?-->
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
          <!-- 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸弫宥夊礋椤掍焦顔囬梻浣告贡閸庛倝宕甸敃鈧埥澶娢熼柨瀣垫綌婵犳鍠楅敃鈺呭礈濞嗘挻鍊跺┑鐘叉处閸婂灚顨ラ悙鑼虎闁告梹纰嶉妵鍕晜閸喖绁梺璇″枤閸嬬喖骞忛悩宸晠妞ゆ梹鍎抽弫鎼佹⒑閼姐倕鞋婵℃ぜ鍔庨幏鍐晝閸屾氨锛熼棅顐㈡处閺岋絾绂嶅鍫㈠彄闁搞儜灞藉壈闂佸憡姊瑰畝鎼佸蓟瀹ュ瀵犲鑸瞪戦埢鍫澪旈悩闈涗沪闁挎洏鍨介悰顕€宕堕妸锕€顎撶紓浣割儏閻忔繈藟鎼淬垻绡€?-->
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
            <!-- Help Tooltip -->
            <div class="group relative inline-flex">
              <Icon
                name="questionCircle"
                size="sm"
                :stroke-width="2"
                class="cursor-help text-gray-400 transition-colors hover:text-primary-500 dark:text-gray-500 dark:hover:text-primary-400"
              />
              <!-- Tooltip Popover -->
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
                  <!-- Arrow -->
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

        <!-- Subscription Configuration -->
        <div class="mt-4 border-t pt-4">
          <div>
            <label class="input-label">{{ t('admin.groups.subscription.type') }}</label>
            <Select v-model="createForm.subscription_type" :options="subscriptionTypeOptions" />
            <p class="input-hint">{{ t('admin.groups.subscription.typeHint') }}</p>
          </div>

          <!-- Subscription limits (only show when subscription type is selected) -->
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

        <!-- 闂傚倸鍊搁崐鎼佸磹閻戣姤鍊块柨鏇炲€哥粻鏍煕椤愶絾绀€缁炬儳娼″鍫曞醇濞戞ê顬夊┑鐐叉噽婵炩偓闁哄被鍊濋獮渚€骞掗幋婵嗩潥婵犵數鍋涢幊鎰箾閳ь剟鏌＄仦绯曞亾閹颁礁鎮戦柟鑲╄ˉ閳ь剙纾鎴︽⒒娴ｄ警鐒炬い鎴濆暣瀹曟繈骞嬮敃鈧拑鐔兼煥濠靛棭妲哥紒顐㈢Ч閺屾稓浠︾拠娴嬪亾濡ゅ懎绀夐柟闂寸劍閳锋垿鎮归崶顏勭毢缂佺姷澧楃换娑橆啅椤旂厧绫嶅Δ鐘靛仜閸熶即骞夐幘顔肩妞ゆ劦鍋勫▓鍏间繆閻愵亜鈧牠寮婚妸鈺佺妞ゆ劧绠戦悞鍨亜閹哄秶鍔嶇紒鈧€ｎ喗鐓欐い鏃傜摂濞堟粍銇勯姀鈽呰€垮┑顔瑰亾闂佹娊鏁崑鎾绘煙闁垮銇濇慨濠冩そ瀹曨偊宕熼崹顐嶎亜鈹戦悙宸Ч婵炲弶绻堝畷鎰板箻椤旇В鎷绘繛杈剧到濠€鍗烇耿娴犲鐓曞┑鐘插暞缁€鈧柧鑽ゅ仦缁绘繈妫冨☉鍗炲壈闂佸搫顑勯懗鍫曞焵椤掆偓缁犲秹宕曢柆宓ュ洭顢涢悙瀵稿弳闂侀潧鐗嗛ˇ浼存偂閺囩喍绻嗛柕鍫濇噹閺嗙偤鏌涢悢鍛婂€杇ravity 闂?gemini 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷?-->
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

        <!-- 闂傚倸鍊搁崐鎼佸磹瀹勬噴褰掑炊椤掆偓杩濋梺閫炲苯澧撮柡灞剧〒閳ь剨缍嗛崑鍛焊娴煎瓨鐓忛柛銉戝喚浼冮悗娈垮枙缁瑦淇婂宀婃Х濠碘剝褰冮悧鎾愁潖閻戞ê顕辨繛鍡楃箳濡诧綁姊洪棃鈺冪Ф缂傚秳绀侀锝嗙節濮橆厽娅滄繝銏ｆ硾璋╅柍鍝勬噺閻撳繐顭跨捄铏瑰闁告梹娼欓湁闁绘宕甸悾娲煛鐏炶鈧繂鐣烽悷鎵虫婵炲棙鍔楄ぐ瀣⒒娴ｅ憡鎯堥柡鍫墴閹嫰顢涘☉妤冪畾闂佸綊妫跨粈浣告暜婵＄偑鍊栧濠氬储瑜庣粩鐔衡偓锝庡枟閳锋帡鏌涚仦鍓ф噮閻犳劒鍗抽弻娑㈡偐閹颁焦鐤佸Δ鐘靛仦閸旀瑥鐣峰鈧幊鐘活敆娴ｈ鍟庡┑鐘愁問閸犳鏁冮埡鍛？闁汇垻顭堢猾宥夋煕鐏炵虎娈斿ù婊堢畺閺屻劌鈹戦崱娑扁偓妤€顭胯婢ф濡甸崟顖涙櫆閻熸瑥瀚悵鏇犵磽?antigravity 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷?-->
        <div v-if="createForm.platform === 'antigravity'" class="border-t pt-4">
          <div class="mb-1.5 flex items-center gap-1">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.groups.supportedScopes.title') }}
            </label>
            <!-- Help Tooltip -->
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

        <!-- MCP XML 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸弫鎾绘偐椤旂懓浜鹃柛鎰靛枛瀹告繃銇勯弽銊р槈閹兼潙锕ら埞鎴炲箠闁稿﹥娲熼獮蹇曗偓锝庡枛閺嬩礁鈹戦悩鍙夊闁绘挻娲樼换娑㈠箣濠靛棜鍩炲Δ鐘靛仦閸旀瑩寮婚敍鍕ㄥ亾閿濆骸浜炲褏鏁婚弻锛勪沪閻愵剛顦ㄧ紓浣虹帛閻╊垰鐣疯ぐ鎺濇晩闁绘挸瀵掑娑㈡⒑鐠囨彃顒㈡い鏃€鐗犲畷鏉课旀担铏诡啎婵犵數濮村ú銈夋嫅閻斿吋鐓忓┑鐐茬仢閸旀淇婇幓鎺斿濞ｅ洤锕、娑橆煥閸愩劋绮俊鐐€х徊钘夛耿鏉堚晜顫?antigravity 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷?-->
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

        <!-- Claude Code 闂傚倸鍊搁崐宄懊归崶顒夋晪鐟滃酣銆冮妷鈺佷紶闁靛／鍌滅憹闁诲骸绠嶉崕閬嵥囬鐐插瀭闁稿本绋撶粻鍓р偓鐟板閸犳洜鑺辨繝姘厸闁告洍鏅涢崝婊呯磼缂佹娲存鐐差儔閹瑧鈧潧鎲￠濠氭⒒娴ｅ憡鍟炴い銊ユ閸犲﹤顓兼径濞箓鏌涢弴銊ョ仩闁告劏鍋撴俊鐐€栭崝锕€顭块埀顒佺箾瀹€濠佺盎妞ゎ亜鍟存俊鍫曞川椤撗冨Ψ闂備浇宕甸崯鍧楀疾閻樺樊鍤曢悹鍥ㄧゴ濡插牓鏌曡箛鏇炐ユい鏂匡躬濮婃椽宕崟顒€鍋嶉梺鍛婃煥閻倸顕ｉ弻銉︽櫜闁搞儮鏅濋敍婵囩箾鏉堝墽瀵肩紒顔界懇瀹曨偄煤椤忓懐鍘介梺缁樻礀閸婃悂銆呴鍌滅＜?anthropic 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷?-->
        <div v-if="createForm.platform === 'anthropic'" class="border-t pt-4">
          <div class="mb-1.5 flex items-center gap-1">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.groups.claudeCode.title') }}
            </label>
            <!-- Help Tooltip -->
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
          <!-- 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌涘┑鍕姉闁稿鎸搁～婵嬫偂鎼粹槅娼剧紓鍌欑贰閸犳螞閸愵喖绠犳繝濠傚暊閺嬫棃鏌熺粙鍨槰闁哥喎绉瑰缁樻媴鐟欏嫬浠╅梺绋垮濡炶棄鐣峰鍐ｆ瀻闁瑰濮烽悞濂告⒑缁洖澧茬紒瀣浮閹繝寮撮姀锛勫帗闂佸疇妗ㄧ粈渚€寮抽悢鍏肩厵闁告劕寮堕崵鍥煛瀹€瀣М妞ゃ垺锕㈤幃婊堝幢濡も偓婢瑰孩绻濆▓鍨灈闁挎洏鍎遍—鍐╃鐎ｃ劉鍋撴笟鈧鎾閻欌偓濞煎﹪姊洪崘鍙夋儓闁稿﹥鍔橀ˇ褰掓煛瀹€瀣ɑ闁诡垱妫冩慨鈧柣妯挎珪閸幯囨⒒娴ｅ憡鍟炴慨濠傜秺瀹曞綊宕归鍛濡炪倖鍔戦崺鍕触鐎ｎ亶鐔嗛悹铏瑰皑闊剟鏌涢悙鑸电【闁宠鍨块幃鈺冩嫚瑜嶆导鎰版⒑绾懏鐝柛鏃€鐟╅獮鍐晸閻樿尙锛滈梺缁樺姈濞兼瑦绂掗鐐╂斀闁绘顕滃銉╂煙閸愭煡鍙勯柕鍡楀€圭缓浠嬪川婵犲嫬骞嶉梻浣告啞缁嬫垿鏁冮妷鈺佺９闁归偊鍙庡▓浠嬫煟閹邦剙绾фい銉у仱閺岀喖顢涘☉娆樻婵犵鍓濋幃鍌炲极閸愵喖鐒垫い鎺嶈兌椤?claude_code_only 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偞鐗犻、鏇㈡晝閳ь剟鎮块鈧弻鐔煎箚閻楀牜妫勯梺鍝勫閸庣敻寮婚垾鎰佸悑閹艰揪绲肩划鍫曟⒑缂佹ɑ鈷掓い顓炵墢濞嗐垽鎮欑喊妯轰壕闁稿繐顦禍楣冩⒑闁偛鑻晶鎾煙椤栨氨鐒哥€规洖宕埥澶娾枎閹存繂绠哄┑鐘愁問閸犳鏁冮埡鍛闁挎洖鍊搁崥褰掓煟閺冨洦顏犵痪鎯с偢閺?-->
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

        <!-- OpenAI Messages 闂傚倸鍊搁崐宄懊归崶褏鏆﹂柛顭戝亝閸欏繘鏌熺紒銏犳珮闁轰礁瀚伴弻娑樷槈濞嗘劗绋囬梺姹囧€ら崰妤呭Φ閸曨垰绠涢柍杞拌閸嬫捇骞囬弶璺紱闂佽宕橀崺鏍窗閸℃稒鐓曢柡鍥ュ妼娴滄劙鏌＄€ｎ偆鎳囨慨濠冩そ瀹曨偊宕熼纰变純闂備焦瀵уú蹇涘磹濠靛鈧礁螣閼姐倝妾紓浣割儓濞夋洟宕愰悙鐑樷拺闁告劕寮堕幆鍫ユ煕婵犲偆鐓奸柛鈹惧亾濡炪倖甯掗ˇ顖氼嚕椤旇姤鍙忓┑鐘插暞閵囨繃顨ラ悙鏉戝闁诡垱妫冮弫鎰板磼濞戣鲸缍岄梻鍌氬€烽懗鍓佸垝椤栫偑鈧啴宕ㄩ弶鎴犵枃闂佸湱澧楀妯肩矆?openai 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷?-->
        <div v-if="createForm.platform === 'openai'" class="border-t border-gray-200 dark:border-dark-400 pt-4 mt-4">
          <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">{{ t('admin.groups.openaiMessages.title') }}</h4>

          <!-- 闂傚倸鍊搁崐鎼佸磹閻戣姤鍤勯柛顐ｆ磵閳ь剨绠撳畷濂稿閳ュ啿绨ラ梺璇插嚱缂嶅棝鍩€椤戞儳鐏╃紓宥咃工閻ｇ兘宕奸弴銊︽櫌婵犮垼娉涢鍡椻枍?Messages 闂傚倸鍊搁崐宄懊归崶褏鏆﹂柛顭戝亝閸欏繘鏌熺紒銏犳珮闁轰礁瀚伴弻娑樷槈濞嗘劗绋囬梺姹囧€ら崰妤呭Φ閸曨垰绠涢柍杞拌閸嬫捇骞囬弶璺紱闂佺懓澧界划顖炲磻閿熺姵鐓涘璺侯儏椤曟粍鎱ㄩ敐鍡楀妞ゎ亜鍟存俊鍫曞幢濡皷鏁嶇紓鍌氬€哥粔宕囨濮橆剛鏆︽繛宸簻閻掑灚銇勯幒宥夋濞存粍绮撻弻鐔兼倻濡櫣浠村銈呯箚閺呮繄妲愰幒妤€鐒?-->
          <div class="flex items-center justify-between">
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

          <!-- 濠电姷鏁告慨鐢割敊閺嶎厼绐楁俊銈呭暞閺嗘粍淇婇妶鍛殶闁活厽鐟╅弻鐔兼倻濡晲绮堕梺閫炲苯澧剧紒鐘虫尭閻ｉ攱绺界粙璇俱劍銇勯弮鍥撴繛鍛Т閳规垿鎮╅崹顐ｆ瘎婵犳鍠氶弫濠氬箠濠靛洢鍋呴柛鎰╁妿閻撴垶绻濋悽闈浶㈡繛璇х畵閹繝鎮㈤梹鎰畾濡炪倖鐗楀銊︾閵忋倖鐓熼柕濞у啫濡虹紓浣虹帛閻╊垰鐣烽悡搴樻斀闁搞儜鍕暭婵犵數鍎戠徊钘壝洪妶澶嬫櫇妞ゅ繐鐗勯埀顑跨閳藉顫滈崱妯虹槣闂佽崵濮村ú鈺侇嚕閹惧鐝堕柡鍥ュ灪閸婄敻鎮峰▎蹇擃仾缂佲偓閳ь剙鈹戦悙璺虹毢濠电偐鍋撻悗瑙勬礃閸ㄥ潡鐛Ο鑲╃＜婵☆垵銆€閸嬫挻绻濆顓犲幘闂佽鍘界敮鎺楀礉濠婂嫮绠鹃柛娑卞枤婢у灚鎱ㄦ繝鍐┿仢鐎规洘绮撻獮鎾诲箳瀹ュ洤鍤紓鍌氬€风欢锟犲窗閺嶎厽鐓€闁挎繂顦粻姘舵倶閻愬灚娅曢柡鍡楁閺屽秷顧侀柛鎾跺枛閻涱喗寰勯幇顒傤啇婵炶揪绲块幊鎾寸闁秵鈷戦柛鎾村絻娴滅偤鏌涢悩鍙夋崳闁哥姴锕ら濂稿炊閳哄喛绱冲┑鐐舵彧缁茶偐鍒掑▎鎾充紶婵炲樊浜濋悡鏇㈡煏婵炲灝鍔ゅù鐘灲閺岋綁鏁愰崨顓熜╅柧缁樼墵閺屾稑鈽夐崡鐐寸亶闂佺瀵掓禍婊堚€旈崘顔嘉ч柛鈩冾殔椤懘姊洪悷鏉挎毐婵炲樊鍘奸悾鐑芥晲閸氥倖妞介、鏃堝川椤撴稑浜鹃柛顭戝亞缁犻箖鏌涢锝囩畼濞寸姰鍨介弻锛勨偓锝庡亝瀹曞矂鏌＄仦鍓с€掗柍褜鍓ㄧ紞鍡涘磻閸涱厾鏆︾€光偓閸曨剛鍘搁悗鍏夊亾閻庯綆鍓涢惁鍫ユ倵鐟欏嫭绀冮柨鏇樺灲閵嗕礁顫滈埀顒勫箖濞嗘挻顥堟繛鎴烆焽缁辩増绻濋悽闈浶ユい锝堟鍗遍柛娑欐綑閸ㄥ倸霉閸忕⒈鍔冮柛銉ｅ妽缂嶅洭鏌嶆潪鎵槮闁诲繐锕娲礈閹绘帊绨撮梺绋垮閻撯€崇暦閹达箑鍐€妞ゆ挾鍠庨埀顒傛暬濮婂宕奸悢琛″闂佽褰冮幉锟犲Φ?-->
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

        <!-- 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偞鐗犻、鏇㈡晝閳ь剟鎮块鈧弻锝呂旈埀顒勬偋婵犲洤鐭楅煫鍥ㄧ⊕閻撶喖鏌曡箛瀣労闁绘帡绠栭弻锝夊Χ婢跺瞼鏆梺鍝勬湰濞叉绮╅悢纰辨晬婵﹩鍓︽禒濂告⒑缂佹绠戦柛妤€鍟块～蹇曠磼濡顎撻梺鍛婄缚閸庝即鎼规惔鈽嗘富闁靛牆妫楅悘銉︾箾鐏炲倸鈧繈寮崘顔碱潊闁靛牆鎳愰鎺戭渻閵堝棙鈷掗柡鍜佸亰瀹曘垽鏌嗗鍡╂濡炪倖鍔戦崹鐑樺緞閸曨剚鍙忓┑鐘插亞閻撳ジ鏌熼姘拱缂佺粯绻堝畷鐔碱敇閻欏懐搴婇梻鍌欑窔濞佳団€﹂鐘典笉闁硅揪绠戦悡婵嬫煛閸ャ儱鐏柣鎾寸☉闇夐柨婵嗘噺閹叉悂寮崼銉﹀€甸悷娆忓缁€鍫ユ煕濡姴娲犻埀顒婄畵婵℃悂鍩℃担鍝ョ崺婵＄偑鍊栭悧妤冨垝瀹€鈧弫?anthropic/antigravity 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷戦柛婵嗗婵ジ鏌涢幘璺烘灈妤犵偛鍟～婊堝焵椤掑嫬绠栨繛鍡樺灦瀹曞寮堕悙闈涱暭闁活偄绻樺缁樻媴閽樺鎯為梺闈╃祷閸庨潧鐣峰鍫熷亜濠靛倸顦扮紞搴㈢節閻㈤潧校闁煎綊绠栭弻瀣炊椤掍胶鍘棅顐㈡处濞叉牕鐨梻浣告啞缁诲啴銆冮崨绮光偓鏃堝礃椤斿槈褔鏌涢埄鍏狀亞鍠婂鍥╃＝闁稿本姘ㄥ瓭闂佹寧娲忛崐婵嬫晲閻愬墎鐤€闁哄啫鍊垮顔剧磽娴ｅ壊鍎忛柣蹇旂箘缁絽鈽夊鍡樺瘜闂侀潧鐗嗗Λ娆戜焊閻㈠憡鐓曢悗锝庡墮閺嬫盯鏌涢埞鎯т壕婵＄偑鍊栫敮鎺楀疮椤愶箑瑙﹂柛锔诲幘绾惧ジ寮堕崼娑樺閻忓繋鍗抽弻鐔风暋閻楀牆娅х紓渚囧枟閻熴儵鍩㈡惔銊ョ煑闁靛／鍠版洜绱?-->
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

        <!-- 婵犵數濮烽弫鍛婃叏閻戝鈧倿鎸婃竟鈺嬬秮瀹曘劑寮堕幋婵堚偓顓烆渻閵堝懐绠伴柣妤€妫濋幃鐐哄垂椤愮姳绨婚梺鐟版惈濡绂嶉崜褏纾奸柛鎾楀棙顎楅梺鍛婄懃閸熸潙鐣峰ú顏勭劦妞ゆ帊闄嶆禍婊堟煙閻戞ê鐏ユい蹇撶摠娣囧﹪顢曢敐鍛紝闂佸搫鐬奸崰鏍箖閻戣棄绾ф繛鍡楀綖缁辨梹淇婇悙顏勨偓鎴﹀垂閾忓湱鐭欓柟杈捐吂閳ь剨绠撴俊鎼佸煛娴ｄ警妲规俊鐐€栫敮濠勬閿熺姴鐤柛娑樼摠閳锋帡鏌涚仦鎹愬闁逞屽墰閸忔﹢骞冮悙鐑樻櫇闁逞屽墴閳ワ箓宕归銉у枛閹虫牠鍩￠崘顏勫脯闂傚倷绀佹竟濠囧磻閸涱劶鍝勵潨閳ь剙鐣烽幋锔绘晜闁割偆鍠撻崢閬嶆⒑閺傘儲娅呴柛鐔跺嵆閸╁﹪寮撮姀锛勫幐闂佸憡绮堢粈浣规櫠閹绢喗鐓涢悘鐐插⒔濞插瓨銇勯姀鈩冪闁轰焦鍔欏畷鍫曞煛閸愩劌绗撻梻?anthropic 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷?-->
        <div v-if="createForm.platform === 'anthropic'" class="border-t pt-4">
          <div class="mb-1.5 flex items-center gap-1">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.groups.modelRouting.title') }}
            </label>
            <!-- Help Tooltip -->
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
          <!-- 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸弫鎾绘偐閼碱剦妲遍柣鐔哥矌婢ф鏁幒妤€鍨傞柛宀€鍋為悡娆撴煙椤栨粌顣兼い銉ヮ樀閺岋紕鈧絺鏅濈粣鏃堟煛鐏炵偓绀嬬€规洜鍘ч埞鎴﹀炊瑜滈埀顒佺洴濮婃椽骞栭悙娴嬪亾閺嶎灐娲偄閻撳氦鎽曢悗骞垮劚閻楁粌顬婇妸鈺傗拺闁告稑锕ョ亸鐢告煕閻樻煡鍙勯柟?-->
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
          <!-- 闂傚倸鍊搁崐宄懊归崶褏鏆﹂柛顭戝亝閸欏繒鈧娲栧ú锔藉垔娴煎瓨鐓ラ柡鍥╁仜閳ь剚鎮傚畷褰掑磼閻愬鍘甸梺璇″灣婢ф藟婢舵劖鐓涢悗锝傛櫇缁愭梻鈧鍠栭悥鐓庣暦閻撳寒娼╂い鎾跺枔瑜板棝姊绘担鍛婃喐濠殿喗娼欒灋婵犻潧顑呴拑鐔哥箾閹存瑥鐏╅柛妤佸▕閺屾洘绻涢崹顔煎缂備降鍔岄…宄邦潖閾忚瀚氶柤纰卞墰椤斿﹪姊洪崫銉バｉ柣妤冨Т椤曪絿鎷犲ù瀣潔闂侀潧绻掓慨鐑筋敊婵犲洦鈷戦柣鐔稿娴犮垽鏌涢悤浣哥仸闁诡喚鍋撶粋鎺斺偓锝庡亐閹锋椽姊婚崒姘卞缂佸甯¤棢婵犲﹤鐗婇悡鏇㈡煟閺傛寧鍟為悘蹇ュ閳ь剚顔栭崳顕€宕戦崟顖ｆ晣闁稿繒鍘х欢鐐测攽閻樺弶鎼愰柡浣筋潐缁绘繈鎮介棃娴躲垽鏌ㄩ弴妯衡偓婵嬬嵁婵犲洤绠涢柣妤€鐗嗘禍妤呮⒑缁嬭法鐏遍柛瀣仱閸╂盯骞嬮敂瑙ｆ嫽闂佸壊鍋呯喊宥呪枍閸垻纾奸柍褜鍓涢埀顒婄秵閸犳鎮￠弴銏＄厸闁稿本绻冪涵鍫曟煟閹烘垶鍟炴い銊ｅ劦閹瑩宕ｆ径妯伙紒闂備浇顕栭崹鍗炍涢崘鈺€绻嗛柟闂寸劍閺呮繈鏌嶈閸撴氨绮嬮幒鎳ㄦ椽顢旈崨顏呭闂傚倸鍊搁悧濠冪瑹濡ゅ懏鍋傛い鎾卞灪閳锋帡鏌涢弴銊ヤ簻妞ゅ浚鍙冮幐濠傗攽閸喎鏋戦梺鍝勫暙閻楀棝鎷戦悢鍏肩叆闁绘柨鎼瓭缂備讲鍋撻悗锝庡亖娴滄粓鏌熼懜顒€濡介柛锝勫嵆閺岀喓绮欓崹顔芥濠殿喖锕ュ钘壩涢崘銊㈡婵炲棗娴氬Σ閬嶆煟鎼粹€冲辅闁稿鎹囬弻鐔兼⒒鐎靛壊妲梺?-->
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
                    <!-- 闂傚倸鍊峰ù鍥敋瑜嶉湁闁绘垼妫勭粻鐘绘煙閹规劦鍤欑紒鐘靛枛閺屻劑鎮㈤崫鍕戯綁鏌涚€ｎ亜鈧潡寮婚妸鈺傚亜闁告繂瀚呴姀銈嗙厽闁圭儤鍨规禒娑㈡煏閸パ冾伃妤犵偞甯″畷鍗烆渻閹屾婵犵绱曢崑鎴﹀磹瑜忕划濠氬箳閹存梹鐎洪梺鍝勬川閸庢劙宕靛Δ鍛厱闁哄洢鍔岄悘鐘电磼閳ь剛鈧綆鍋佹禍婊堟煛瀹ュ啫濡介柣銊﹀灴閺岋綁骞橀弶鎴犱紝濠殿喖锕︾划顖炲箯閸涙潙浼犻柕澶堝€涘鍛婁繆閻愵亜鈧牠宕归棃娴㈡椽顢橀悙鍨閻熸粎澧楃敮妤呭吹閸愵喗鐓冮柛婵嗗閺嗙喖鏌?-->
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
                    <!-- 闂傚倸鍊搁崐宄懊归崶褏鏆﹂柛顭戝亝閸欏繘鏌ｉ姀銏╃劸缂佲偓婢跺本鍠愰煫鍥ㄦ礀閸ㄦ繂鈹戦悩瀹犲缂佺媴缍侀弻銊モ攽閸℃娈ㄥ┑顔款潐椤ㄥ﹤顫忛搹鍦煋闁糕剝顨呴瀛樼箾閸喐鈷掔紒杈ㄥ浮閹晜绗熼娑氱Х缂傚倷鑳剁划顖滄崲閸愵亝宕叉繝闈涱儏绾惧吋绻涢幋鐐嗘垿顢旈悩缁樷拻濞达絼璀﹂悞鐐亜閹存繃鍣界紒顔芥閹兘鏌囬敃鈧▓銊╂⒑閸撴彃浜濇繛鍙夌墵閺屽宕堕浣哄幗闂佸搫鍟崑鍡涙倿閸撗呮／闁哄娉曟晥濠殿喖锕ュ钘夌暦閵婏妇绡€闁告洦鍓欏▍鎴炵節绾版ɑ顫婇柛瀣嚇閵嗗啯绻濋崶顬?-->
                    <div class="relative account-search-container">
                      <input
                        v-model="accountSearchKeyword[getCreateRuleSearchKey(rule)]"
                        type="text"
                        class="input text-sm"
                        :placeholder="t('admin.groups.modelRouting.searchAccountPlaceholder')"
                        @input="searchAccountsByRule(rule)"
                        @focus="onAccountSearchFocus(rule)"
                      />
                      <!-- 闂傚倸鍊搁崐鎼佸磹閻戣姤鍤勯柛顐ｆ礀缁犵娀鏌熼幑鎰靛殭閻熸瑱绠撻幃妤呮晲鎼粹€愁潻闂佹悶鍔嶇换鍫ョ嵁閺嶎灔搴敆閳ь剚淇婇懖鈺冩／闁诡垎浣镐划闂佸搫鏈ú妯兼崲濞戞粠妲婚梺纭呮珪閸旀瑩寮鍜佺叆闁割偆鍟块幏娲煟閻斿摜鎳冮悗姘煎墴瀹曟繈濡堕崱鎰盎闂佸搫娲㈤崝宀勭嵁濡ゅ啰纾奸柛灞炬皑鏁堝銈冨灪缁嬫垿锝炲┑瀣櫜闁糕檧鏅滅紞宀勬⒒閸屾瑧顦︽繝鈧柆宥呯；闁规崘顕х粈鍫熸叏濡寧纭剧痪鎯ф健閺岀喖鏌囬敃鈧崢鎾煛閳ь剟鎳為妷锝勭盎闂佸搫绉查崝搴ㄣ€傞弻銉︾厵妞ゆ牗姘ㄦ晶娑㈡煏?-->
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
          <!-- 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柟闂寸绾惧潡鏌熺€电啸缂佲偓婵犲洦鐓冪憸婊堝礈濮樿泛桅闁告洦鍨伴崡鎶芥煕閳╁喚娈旀い蹇ｄ邯閺屾稑鈻庤箛鏇狀啋闂佸搫鐭夌紞渚€鐛崶顒夋晢闁稿本鐟фす鎶芥⒒娴ｅ憡鎲稿┑顔芥綑铻炴繝闈涱儏閽冪喐绻涢幋娆忕仼闁告濞婇弻鏇熺箾閸喖濮嶇紓浣靛妼椤嘲顫忛搹瑙勫珰闁肩⒈鍓涢濠囨⒑閸濄儱校闁绘濮撮锝嗙節濮橆厼浜滈梺鎯х箰濠€閬嶆偂閹炬枼鏀介柣鎰级椤ョ娀鏌涚€ｎ偅宕岄柛鈹垮灲瀹曞ジ濡烽敂瑙勫闂備浇宕甸崰鎾存櫠濡ゅ懎鐤柣鎰ゴ閺€浠嬫煟閹邦垼鍤嬮棅顒夊墰閳ь剚顔栭崳顕€宕戦崟顖ｆ晣闁稿繒鍘х欢鐐测攽閻樺弶鎼愰柡浣筋潐缁绘繈鎮介棃娴躲垽鏌ㄩ弴妯衡偓婵嬬嵁婵犲洤绠涢柣妤€鐗嗘禍妤呮⒑缁嬭法鐏遍柛瀣仱閸╂盯骞嬮敂瑙ｆ嫽闂佸壊鍋呯喊宥呪枍閸垻纾奸柍褜鍓涢埀顒婄秵閸犳鎮￠弴銏＄厸闁稿本绻冪涵鍫曟煟閹烘垶鍟炴い銊ｅ劦閹瑩宕ｆ径妯伙紒闂備浇顕栭崹鍗炍涢崘鈺€绻嗛柟闂寸劍閺呮繈鏌嶈閸撴氨绮嬮幒鎳ㄦ椽顢旈崨顏呭闂傚倸鍊搁悧濠冪瑹濡ゅ懏鍋傛い鎾卞灪閳锋帡鏌涢弴銊ヤ簻妞ゅ浚鍙冮幐濠傗攽閸喎鏋戦梺鍝勫暙閻楀棝鎷戦悢鍏肩叆闁绘柨鎼瓭缂備讲鍋撻悗锝庡亖娴滄粓鏌熼懜顒€濡介柛锝勫嵆閺岀喓绮欓崹顔芥濠殿喖锕ュ钘壩涢崘銊㈡婵炲棗娴氬Σ閬嶆煟鎼粹€冲辅闁稿鎹囬弻鐔兼⒒鐎靛壊妲梺?-->
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

    <!-- Edit Group Modal -->
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
        <!-- 濠电姷鏁告慨鐑藉极閹间礁纾绘繛鎴欏焺閺佸銇勯幘璺烘瀾闁告瑥绻橀幃妤€鈽夊▎娆庣返濠电偛鐗呯划娆撳蓟閻斿吋鈷掗悗闈涘濡差噣姊洪幖鐐插闁稿﹤娼￠悰顕€寮介妸锔剧Ф闂佸憡鎸嗛崟顐¤繕缂傚倷鑳堕崑鎾诲磿閹惰棄围闁归棿绀侀拑鐔哥箾閹存瑥鐏╅柛妤佸▕閺屾洘绻涢崹顔煎闂佺厧澹婃禍婊堝煘閹达箑鐒洪柛鎰╁妿缁佸嘲顪冮妶搴″箻闁稿繑锚椤曪絿鎷犲ù瀣潔闂侀潧绻掓慨鐑筋敊婵犲洦鈷戦悷娆忓閸斻倝鏌涢悢绋款嚋闁逛究鍔戝畷銊︾節閸曨厾妲囬梻渚€娼ф蹇曞緤閸撗勫厹濡わ絽鍟崐鍨叏濮楀棗骞楃紒鑸电叀閺岋綁鏁愰崶褍骞嬮梺杞扮劍閸旀牕顕ラ崟顒傜瘈闁告洖鐏氳ⅸ闂傚倸鍊风欢姘焽瑜旈幃褔宕卞鏇熸そ婵℃悂鍩炶濞差參宕洪敓鐘茬妞ゅ繐妫寸槐鎶芥⒒娴ｄ警鐒鹃柡鍫墴閹虫繃銈ｉ崘銊у姦濡炪倖宸婚崑鎾剁磽瀹ュ拑韬鐐插暢椤﹀綊鏌熼鐣屾噰鐎规洖缍婇、鏇㈩敃椤厾绱︽繝纰夌磿閸嬫垿宕愰弽顓炵鐟滃繒鍒掓繝姘闁绘﹢娼ч弳妤呮倵楠炲灝鍔氭い锔垮嵆瀹曟垿鏁愰崥銈呯秺閺佹劙宕ㄩ鍏兼畼闂佽崵濮甸崝褏妲愰弴鐘愁潟闁圭儤顨呴悞?-->
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
          <!-- 闂傚倸鍊峰ù鍥敋瑜嶉湁闁绘垼妫勭粻鐘绘煙閹规劦鍤欑紒鐘靛枛閺屻劑鎮㈤崫鍕戯綁鏌涚€ｎ亜鈧潡寮婚妸鈺傚亜闁告繂瀚呴姀銈嗙厽闁圭儤鍨规禒娑㈡煏閸パ冾伃妤犵偞甯″畷鍗烆渻閹屾婵犵绱曢崑鎴﹀磹瑜忛埀顒勬涧閻倸鐣烽崷顓熷磯闁惧繗顫夊▓楣冩煟閻斿摜鎳冮悗姘煎弮瀹曟洟鎮㈤崗鑲╁弳濠电娀娼уΛ娆忣啅濠靛鐓ユ繛鎴炵懅閻帗鎱ㄦ繝鍐┿仢鐎规洏鍔嶇换婵嬪礃椤垶袩闂傚倷绀侀幉锟犲箰婵犳碍鍎庢い鏍ㄦ皑閺嗭妇鎲搁悧鍫濈瑲闁搞倕鍟撮弻宥夊传閸曨偅娈堕梺?-->
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
          <!-- 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸弫宥夊礋椤掍焦顔囬梻浣告贡閸庛倝宕甸敃鈧埥澶娢熼柨瀣垫綌婵犳鍠楅敃鈺呭礈濞嗘挻鍊跺┑鐘叉处閸婂灚顨ラ悙鑼虎闁告梹纰嶉妵鍕晜閸喖绁梺璇″枤閸嬬喖骞忛悩宸晠妞ゆ梹鍎抽弫鎼佹⒑閼姐倕鞋婵℃ぜ鍔庨幏鍐晝閸屾氨锛熼棅顐㈡处閺岋絾绂嶅鍫㈠彄闁搞儜灞藉壈闂佸憡姊瑰畝鎼佸蓟瀹ュ瀵犲鑸瞪戦埢鍫澪旈悩闈涗沪闁挎洏鍨介悰顕€宕堕妸锕€顎撶紓浣割儏閻忔繈藟鎼淬垻绡€?-->
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
            <!-- Help Tooltip -->
            <div class="group relative inline-flex">
              <Icon
                name="questionCircle"
                size="sm"
                :stroke-width="2"
                class="cursor-help text-gray-400 transition-colors hover:text-primary-500 dark:text-gray-500 dark:hover:text-primary-400"
              />
              <!-- Tooltip Popover -->
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
                  <!-- Arrow -->
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

        <!-- Subscription Configuration -->
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

          <!-- Subscription limits (only show when subscription type is selected) -->
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

        <!-- 闂傚倸鍊搁崐鎼佸磹閻戣姤鍊块柨鏇炲€哥粻鏍煕椤愶絾绀€缁炬儳娼″鍫曞醇濞戞ê顬夊┑鐐叉噽婵炩偓闁哄被鍊濋獮渚€骞掗幋婵嗩潥婵犵數鍋涢幊鎰箾閳ь剟鏌＄仦绯曞亾閹颁礁鎮戦柟鑲╄ˉ閳ь剙纾鎴︽⒒娴ｄ警鐒炬い鎴濆暣瀹曟繈骞嬮敃鈧拑鐔兼煥濠靛棭妲哥紒顐㈢Ч閺屾稓浠︾拠娴嬪亾濡ゅ懎绀夐柟闂寸劍閳锋垿鎮归崶顏勭毢缂佺姷澧楃换娑橆啅椤旂厧绫嶅Δ鐘靛仜閸熶即骞夐幘顔肩妞ゆ劦鍋勫▓鍏间繆閻愵亜鈧牠寮婚妸鈺佺妞ゆ劧绠戦悞鍨亜閹哄秶鍔嶇紒鈧€ｎ喗鐓欐い鏃傜摂濞堟粍銇勯姀鈽呰€垮┑顔瑰亾闂佹娊鏁崑鎾绘煙闁垮銇濇慨濠冩そ瀹曨偊宕熼崹顐嶎亜鈹戦悙宸Ч婵炲弶绻堝畷鎰板箻椤旇В鎷绘繛杈剧到濠€鍗烇耿娴犲鐓曞┑鐘插暞缁€鈧柧鑽ゅ仦缁绘繈妫冨☉鍗炲壈闂佸搫顑勯懗鍫曞焵椤掆偓缁犲秹宕曢柆宓ュ洭顢涢悙瀵稿弳闂侀潧鐗嗛ˇ浼存偂閺囩喍绻嗛柕鍫濇噹閺嗙偤鏌涢悢鍛婂€杇ravity 闂?gemini 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷?-->
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

        <!-- 闂傚倸鍊搁崐鎼佸磹瀹勬噴褰掑炊椤掆偓杩濋梺閫炲苯澧撮柡灞剧〒閳ь剨缍嗛崑鍛焊娴煎瓨鐓忛柛銉戝喚浼冮悗娈垮枙缁瑦淇婂宀婃Х濠碘剝褰冮悧鎾愁潖閻戞ê顕辨繛鍡楃箳濡诧綁姊洪棃鈺冪Ф缂傚秳绀侀锝嗙節濮橆厽娅滄繝銏ｆ硾璋╅柍鍝勬噺閻撳繐顭跨捄铏瑰闁告梹娼欓湁闁绘宕甸悾娲煛鐏炶鈧繂鐣烽悷鎵虫婵炲棙鍔楄ぐ瀣⒒娴ｅ憡鎯堥柡鍫墴閹嫰顢涘☉妤冪畾闂佸綊妫跨粈浣告暜婵＄偑鍊栧濠氬储瑜庣粩鐔衡偓锝庡枟閳锋帡鏌涚仦鍓ф噮閻犳劒鍗抽弻娑㈡偐閹颁焦鐤佸Δ鐘靛仦閸旀瑥鐣峰鈧幊鐘活敆娴ｈ鍟庡┑鐘愁問閸犳鏁冮埡鍛？闁汇垻顭堢猾宥夋煕鐏炵虎娈斿ù婊堢畺閺屻劌鈹戦崱娑扁偓妤€顭胯婢ф濡甸崟顖涙櫆閻熸瑥瀚悵鏇犵磽?antigravity 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷?-->
        <div v-if="editForm.platform === 'antigravity'" class="border-t pt-4">
          <div class="mb-1.5 flex items-center gap-1">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.groups.supportedScopes.title') }}
            </label>
            <!-- Help Tooltip -->
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

        <!-- MCP XML 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸弫鎾绘偐椤旂懓浜鹃柛鎰靛枛瀹告繃銇勯弽銊р槈閹兼潙锕ら埞鎴炲箠闁稿﹥娲熼獮蹇曗偓锝庡枛閺嬩礁鈹戦悩鍙夊闁绘挻娲樼换娑㈠箣濠靛棜鍩炲Δ鐘靛仦閸旀瑩寮婚敍鍕ㄥ亾閿濆骸浜炲褏鏁婚弻锛勪沪閻愵剛顦ㄧ紓浣虹帛閻╊垰鐣疯ぐ鎺濇晩闁绘挸瀵掑娑㈡⒑鐠囨彃顒㈡い鏃€鐗犲畷鏉课旀担铏诡啎婵犵數濮村ú銈夋嫅閻斿吋鐓忓┑鐐茬仢閸旀淇婇幓鎺斿濞ｅ洤锕、娑橆煥閸愩劋绮俊鐐€х徊钘夛耿鏉堚晜顫?antigravity 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷?-->
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

        <!-- Claude Code 闂傚倸鍊搁崐宄懊归崶顒夋晪鐟滃酣銆冮妷鈺佷紶闁靛／鍌滅憹闁诲骸绠嶉崕閬嵥囬鐐插瀭闁稿本绋撶粻鍓р偓鐟板閸犳洜鑺辨繝姘厸闁告洍鏅涢崝婊呯磼缂佹娲存鐐差儔閹瑧鈧潧鎲￠濠氭⒒娴ｅ憡鍟炴い銊ユ閸犲﹤顓兼径濞箓鏌涢弴銊ョ仩闁告劏鍋撴俊鐐€栭崝锕€顭块埀顒佺箾瀹€濠佺盎妞ゎ亜鍟存俊鍫曞川椤撗冨Ψ闂備浇宕甸崯鍧楀疾閻樺樊鍤曢悹鍥ㄧゴ濡插牓鏌曡箛鏇炐ユい鏂匡躬濮婃椽宕崟顒€鍋嶉梺鍛婃煥閻倸顕ｉ弻銉︽櫜闁搞儮鏅濋敍婵囩箾鏉堝墽瀵肩紒顔界懇瀹曨偄煤椤忓懐鍘介梺缁樻礀閸婃悂銆呴鍌滅＜?anthropic 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷?-->
        <div v-if="editForm.platform === 'anthropic'" class="border-t pt-4">
          <div class="mb-1.5 flex items-center gap-1">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.groups.claudeCode.title') }}
            </label>
            <!-- Help Tooltip -->
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
          <!-- 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌涘┑鍕姉闁稿鎸搁～婵嬫偂鎼粹槅娼剧紓鍌欑贰閸犳螞閸愵喖绠犳繝濠傚暊閺嬫棃鏌熺粙鍨槰闁哥喎绉瑰缁樻媴鐟欏嫬浠╅梺绋垮濡炶棄鐣峰鍐ｆ瀻闁瑰濮烽悞濂告⒑缁洖澧茬紒瀣浮閹繝寮撮姀锛勫帗闂佸疇妗ㄧ粈渚€寮抽悢鍏肩厵闁告劕寮堕崵鍥煛瀹€瀣М妞ゃ垺锕㈤幃婊堝幢濡も偓婢瑰孩绻濆▓鍨灈闁挎洏鍎遍—鍐╃鐎ｃ劉鍋撴笟鈧鎾閻欌偓濞煎﹪姊洪崘鍙夋儓闁稿﹥鍔橀ˇ褰掓煛瀹€瀣ɑ闁诡垱妫冩慨鈧柣妯挎珪閸幯囨⒒娴ｅ憡鍟炴慨濠傜秺瀹曞綊宕归鍛濡炪倖鍔戦崺鍕触鐎ｎ亶鐔嗛悹铏瑰皑闊剟鏌涢悙鑸电【闁宠鍨块幃鈺冩嫚瑜嶆导鎰版⒑绾懏鐝柛鏃€鐟╅獮鍐晸閻樿尙锛滈梺缁樺姈濞兼瑦绂掗鐐╂斀闁绘顕滃銉╂煙閸愭煡鍙勯柕鍡楀€圭缓浠嬪川婵犲嫬骞嶉梻浣告啞缁嬫垿鏁冮妷鈺佺９闁归偊鍙庡▓浠嬫煟閹邦剙绾фい銉у仱閺岀喖顢涘☉娆樻婵犵鍓濋幃鍌炲极閸愵喖鐒垫い鎺嶈兌椤?claude_code_only 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偞鐗犻、鏇㈡晝閳ь剟鎮块鈧弻鐔煎箚閻楀牜妫勯梺鍝勫閸庣敻寮婚垾鎰佸悑閹艰揪绲肩划鍫曟⒑缂佹ɑ鈷掓い顓炵墢濞嗐垽鎮欑喊妯轰壕闁稿繐顦禍楣冩⒑闁偛鑻晶鎾煙椤栨氨鐒哥€规洖宕埥澶娾枎閹存繂绠哄┑鐘愁問閸犳鏁冮埡鍛闁挎洖鍊搁崥褰掓煟閺冨洦顏犵痪鎯с偢閺?-->
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

        <!-- OpenAI Messages 闂傚倸鍊搁崐宄懊归崶褏鏆﹂柛顭戝亝閸欏繘鏌熺紒銏犳珮闁轰礁瀚伴弻娑樷槈濞嗘劗绋囬梺姹囧€ら崰妤呭Φ閸曨垰绠涢柍杞拌閸嬫捇骞囬弶璺紱闂佽宕橀崺鏍窗閸℃稒鐓曢柡鍥ュ妼娴滄劙鏌＄€ｎ偆鎳囨慨濠冩そ瀹曨偊宕熼纰变純闂備焦瀵уú蹇涘磹濠靛鈧礁螣閼姐倝妾紓浣割儓濞夋洟宕愰悙鐑樷拺闁告劕寮堕幆鍫ユ煕婵犲偆鐓奸柛鈹惧亾濡炪倖甯掗ˇ顖氼嚕椤旇姤鍙忓┑鐘插暞閵囨繃顨ラ悙鏉戝闁诡垱妫冮弫鎰板磼濞戣鲸缍岄梻鍌氬€烽懗鍓佸垝椤栫偑鈧啴宕ㄩ弶鎴犵枃闂佸湱澧楀妯肩矆?openai 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷?-->
        <div v-if="editForm.platform === 'openai'" class="border-t border-gray-200 dark:border-dark-400 pt-4 mt-4">
          <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">{{ t('admin.groups.openaiMessages.title') }}</h4>

          <!-- 闂傚倸鍊搁崐鎼佸磹閻戣姤鍤勯柛顐ｆ磵閳ь剨绠撳畷濂稿閳ュ啿绨ラ梺璇插嚱缂嶅棝鍩€椤戞儳鐏╃紓宥咃工閻ｇ兘宕奸弴銊︽櫌婵犮垼娉涢鍡椻枍?Messages 闂傚倸鍊搁崐宄懊归崶褏鏆﹂柛顭戝亝閸欏繘鏌熺紒銏犳珮闁轰礁瀚伴弻娑樷槈濞嗘劗绋囬梺姹囧€ら崰妤呭Φ閸曨垰绠涢柍杞拌閸嬫捇骞囬弶璺紱闂佺懓澧界划顖炲磻閿熺姵鐓涘璺侯儏椤曟粍鎱ㄩ敐鍡楀妞ゎ亜鍟存俊鍫曞幢濡皷鏁嶇紓鍌氬€哥粔宕囨濮橆剛鏆︽繛宸簻閻掑灚銇勯幒宥夋濞存粍绮撻弻鐔兼倻濡櫣浠村銈呯箚閺呮繄妲愰幒妤€鐒?-->
          <div class="flex items-center justify-between">
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

          <!-- 濠电姷鏁告慨鐢割敊閺嶎厼绐楁俊銈呭暞閺嗘粍淇婇妶鍛殶闁活厽鐟╅弻鐔兼倻濡晲绮堕梺閫炲苯澧剧紒鐘虫尭閻ｉ攱绺界粙璇俱劍銇勯弮鍥撴繛鍛Т閳规垿鎮╅崹顐ｆ瘎婵犳鍠氶弫濠氬箠濠靛洢鍋呴柛鎰╁妿閻撴垶绻濋悽闈浶㈡繛璇х畵閹繝鎮㈤梹鎰畾濡炪倖鐗楀銊︾閵忋倖鐓熼柕濞у啫濡虹紓浣虹帛閻╊垰鐣烽悡搴樻斀闁搞儜鍕暭婵犵數鍎戠徊钘壝洪妶澶嬫櫇妞ゅ繐鐗勯埀顑跨閳藉顫滈崱妯虹槣闂佽崵濮村ú鈺侇嚕閹惧鐝堕柡鍥ュ灪閸婄敻鎮峰▎蹇擃仾缂佲偓閳ь剙鈹戦悙璺虹毢濠电偐鍋撻悗瑙勬礃閸ㄥ潡鐛Ο鑲╃＜婵☆垵銆€閸嬫挻绻濆顓犲幘闂佽鍘界敮鎺楀礉濠婂嫮绠鹃柛娑卞枤婢у灚鎱ㄦ繝鍐┿仢鐎规洘绮撻獮鎾诲箳瀹ュ洤鍤紓鍌氬€风欢锟犲窗閺嶎厽鐓€闁挎繂顦粻姘舵倶閻愬灚娅曢柡鍡楁閺屽秷顧侀柛鎾跺枛閻涱喗寰勯幇顒傤啇婵炶揪绲块幊鎾寸闁秵鈷戦柛鎾村絻娴滅偤鏌涢悩鍙夋崳闁哥姴锕ら濂稿炊閳哄喛绱冲┑鐐舵彧缁茶偐鍒掑▎鎾充紶婵炲樊浜濋悡鏇㈡煏婵炲灝鍔ゅù鐘灲閺岋綁鏁愰崨顓熜╅柧缁樼墵閺屾稑鈽夐崡鐐寸亶闂佺瀵掓禍婊堚€旈崘顔嘉ч柛鈩冾殔椤懘姊洪悷鏉挎毐婵炲樊鍘奸悾鐑芥晲閸氥倖妞介、鏃堝川椤撴稑浜鹃柛顭戝亞缁犻箖鏌涢锝囩畼濞寸姰鍨介弻锛勨偓锝庡亝瀹曞矂鏌＄仦鍓с€掗柍褜鍓ㄧ紞鍡涘磻閸涱厾鏆︾€光偓閸曨剛鍘搁悗鍏夊亾閻庯綆鍓涢惁鍫ユ倵鐟欏嫭绀冮柨鏇樺灲閵嗕礁顫滈埀顒勫箖濞嗘挻顥堟繛鎴烆焽缁辩増绻濋悽闈浶ユい锝堟鍗遍柛娑欐綑閸ㄥ倸霉閸忕⒈鍔冮柛銉ｅ妽缂嶅洭鏌嶆潪鎵槮闁诲繐锕娲礈閹绘帊绨撮梺绋垮閻撯€崇暦閹达箑鍐€妞ゆ挾鍠庨埀顒傛暬濮婂宕奸悢琛″闂佽褰冮幉锟犲Φ?-->
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

        <!-- 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偞鐗犻、鏇㈡晝閳ь剟鎮块鈧弻锝呂旈埀顒勬偋婵犲洤鐭楅煫鍥ㄧ⊕閻撶喖鏌曡箛瀣労闁绘帡绠栭弻锝夊Χ婢跺瞼鏆梺鍝勬湰濞叉绮╅悢纰辨晬婵﹩鍓︽禒濂告⒑缂佹绠戦柛妤€鍟块～蹇曠磼濡顎撻梺鍛婄缚閸庝即鎼规惔鈽嗘富闁靛牆妫楅悘銉︾箾鐏炲倸鈧繈寮崘顔碱潊闁靛牆鎳愰鎺戭渻閵堝棙鈷掗柡鍜佸亰瀹曘垽鏌嗗鍡╂濡炪倖鍔戦崹鐑樺緞閸曨剚鍙忓┑鐘插亞閻撳ジ鏌熼姘拱缂佺粯绻堝畷鐔碱敇閻欏懐搴婇梻鍌欑窔濞佳団€﹂鐘典笉闁硅揪绠戦悡婵嬫煛閸ャ儱鐏柣鎾寸☉闇夐柨婵嗘噺閹叉悂寮崼銉﹀€甸悷娆忓缁€鍫ユ煕濡姴娲犻埀顒婄畵婵℃悂鍩℃担鍝ョ崺婵＄偑鍊栭悧妤冨垝瀹€鈧弫?anthropic/antigravity 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷戦柛婵嗗婵ジ鏌涢幘璺烘灈妤犵偛鍟～婊堝焵椤掑嫬绠栨繛鍡樺灦瀹曞寮堕悙闈涱暭闁活偄绻樺缁樻媴閽樺鎯為梺闈╃祷閸庨潧鐣峰鍫熷亜濠靛倸顦扮紞搴㈢節閻㈤潧校闁煎綊绠栭弻瀣炊椤掍胶鍘棅顐㈡处濞叉牕鐨梻浣告啞缁诲啴銆冮崨绮光偓鏃堝礃椤斿槈褔鏌涢埄鍏狀亞鍠婂鍥╃＝闁稿本姘ㄥ瓭闂佹寧娲忛崐婵嬫晲閻愬墎鐤€闁哄啫鍊垮顔剧磽娴ｅ壊鍎忛柣蹇旂箘缁絽鈽夊鍡樺瘜闂侀潧鐗嗗Λ娆戜焊閻㈠憡鐓曢悗锝庡墮閺嬫盯鏌涢埞鎯т壕婵＄偑鍊栫敮鎺楀疮椤愶箑瑙﹂柛锔诲幘绾惧ジ寮堕崼娑樺閻忓繋鍗抽弻鐔风暋閻楀牆娅х紓渚囧枟閻熴儵鍩㈡惔銊ョ煑闁靛／鍠版洜绱?-->
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

        <!-- 婵犵數濮烽弫鍛婃叏閻戝鈧倿鎸婃竟鈺嬬秮瀹曘劑寮堕幋婵堚偓顓烆渻閵堝懐绠伴柣妤€妫濋幃鐐哄垂椤愮姳绨婚梺鐟版惈濡绂嶉崜褏纾奸柛鎾楀棙顎楅梺鍛婄懃閸熸潙鐣峰ú顏勭劦妞ゆ帊闄嶆禍婊堟煙閻戞ê鐏ユい蹇撶摠娣囧﹪顢曢敐鍛紝闂佸搫鐬奸崰鏍箖閻戣棄绾ф繛鍡楀綖缁辨梹淇婇悙顏勨偓鎴﹀垂閾忓湱鐭欓柟杈捐吂閳ь剨绠撴俊鎼佸煛娴ｄ警妲规俊鐐€栫敮濠勬閿熺姴鐤柛娑樼摠閳锋帡鏌涚仦鎹愬闁逞屽墰閸忔﹢骞冮悙鐑樻櫇闁逞屽墴閳ワ箓宕归銉у枛閹虫牠鍩￠崘顏勫脯闂傚倷绀佹竟濠囧磻閸涱劶鍝勵潨閳ь剙鐣烽幋锔绘晜闁割偆鍠撻崢閬嶆⒑閺傘儲娅呴柛鐔跺嵆閸╁﹪寮撮姀锛勫幐闂佸憡绮堢粈浣规櫠閹绢喗鐓涢悘鐐插⒔濞插瓨銇勯姀鈩冪闁轰焦鍔欏畷鍫曞煛閸愩劌绗撻梻?anthropic 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷?-->
        <div v-if="editForm.platform === 'anthropic'" class="border-t pt-4">
          <div class="mb-1.5 flex items-center gap-1">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.groups.modelRouting.title') }}
            </label>
            <!-- Help Tooltip -->
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
          <!-- 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸弫鎾绘偐閼碱剦妲遍柣鐔哥矌婢ф鏁幒妤€鍨傞柛宀€鍋為悡娆撴煙椤栨粌顣兼い銉ヮ樀閺岋紕鈧絺鏅濈粣鏃堟煛鐏炵偓绀嬬€规洜鍘ч埞鎴﹀炊瑜滈埀顒佺洴濮婃椽骞栭悙娴嬪亾閺嶎灐娲偄閻撳氦鎽曢悗骞垮劚閻楁粌顬婇妸鈺傗拺闁告稑锕ョ亸鐢告煕閻樻煡鍙勯柟?-->
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
          <!-- 闂傚倸鍊搁崐宄懊归崶褏鏆﹂柛顭戝亝閸欏繒鈧娲栧ú锔藉垔娴煎瓨鐓ラ柡鍥╁仜閳ь剚鎮傚畷褰掑磼閻愬鍘甸梺璇″灣婢ф藟婢舵劖鐓涢悗锝傛櫇缁愭梻鈧鍠栭悥鐓庣暦閻撳寒娼╂い鎾跺枔瑜板棝姊绘担鍛婃喐濠殿喗娼欒灋婵犻潧顑呴拑鐔哥箾閹存瑥鐏╅柛妤佸▕閺屾洘绻涢崹顔煎缂備降鍔岄…宄邦潖閾忚瀚氶柤纰卞墰椤斿﹪姊洪崫銉バｉ柣妤冨Т椤曪絿鎷犲ù瀣潔闂侀潧绻掓慨鐑筋敊婵犲洦鈷戦柣鐔稿娴犮垽鏌涢悤浣哥仸闁诡喚鍋撶粋鎺斺偓锝庡亐閹锋椽姊婚崒姘卞缂佸甯¤棢婵犲﹤鐗婇悡鏇㈡煟閺傛寧鍟為悘蹇ュ閳ь剚顔栭崳顕€宕戦崟顖ｆ晣闁稿繒鍘х欢鐐测攽閻樺弶鎼愰柡浣筋潐缁绘繈鎮介棃娴躲垽鏌ㄩ弴妯衡偓婵嬬嵁婵犲洤绠涢柣妤€鐗嗘禍妤呮⒑缁嬭法鐏遍柛瀣仱閸╂盯骞嬮敂瑙ｆ嫽闂佸壊鍋呯喊宥呪枍閸垻纾奸柍褜鍓涢埀顒婄秵閸犳鎮￠弴銏＄厸闁稿本绻冪涵鍫曟煟閹烘垶鍟炴い銊ｅ劦閹瑩宕ｆ径妯伙紒闂備浇顕栭崹鍗炍涢崘鈺€绻嗛柟闂寸劍閺呮繈鏌嶈閸撴氨绮嬮幒鎳ㄦ椽顢旈崨顏呭闂傚倸鍊搁悧濠冪瑹濡ゅ懏鍋傛い鎾卞灪閳锋帡鏌涢弴銊ヤ簻妞ゅ浚鍙冮幐濠傗攽閸喎鏋戦梺鍝勫暙閻楀棝鎷戦悢鍏肩叆闁绘柨鎼瓭缂備讲鍋撻悗锝庡亖娴滄粓鏌熼懜顒€濡介柛锝勫嵆閺岀喓绮欓崹顔芥濠殿喖锕ュ钘壩涢崘銊㈡婵炲棗娴氬Σ閬嶆煟鎼粹€冲辅闁稿鎹囬弻鐔兼⒒鐎靛壊妲梺?-->
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
                    <!-- 闂傚倸鍊峰ù鍥敋瑜嶉湁闁绘垼妫勭粻鐘绘煙閹规劦鍤欑紒鐘靛枛閺屻劑鎮㈤崫鍕戯綁鏌涚€ｎ亜鈧潡寮婚妸鈺傚亜闁告繂瀚呴姀銈嗙厽闁圭儤鍨规禒娑㈡煏閸パ冾伃妤犵偞甯″畷鍗烆渻閹屾婵犵绱曢崑鎴﹀磹瑜忕划濠氬箳閹存梹鐎洪梺鍝勬川閸庢劙宕靛Δ鍛厱闁哄洢鍔岄悘鐘电磼閳ь剛鈧綆鍋佹禍婊堟煛瀹ュ啫濡介柣銊﹀灴閺岋綁骞橀弶鎴犱紝濠殿喖锕︾划顖炲箯閸涙潙浼犻柕澶堝€涘鍛婁繆閻愵亜鈧牠宕归棃娴㈡椽顢橀悙鍨閻熸粎澧楃敮妤呭吹閸愵喗鐓冮柛婵嗗閺嗙喖鏌?-->
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
                    <!-- 闂傚倸鍊搁崐宄懊归崶褏鏆﹂柛顭戝亝閸欏繘鏌ｉ姀銏╃劸缂佲偓婢跺本鍠愰煫鍥ㄦ礀閸ㄦ繂鈹戦悩瀹犲缂佺媴缍侀弻銊モ攽閸℃娈ㄥ┑顔款潐椤ㄥ﹤顫忛搹鍦煋闁糕剝顨呴瀛樼箾閸喐鈷掔紒杈ㄥ浮閹晜绗熼娑氱Х缂傚倷鑳剁划顖滄崲閸愵亝宕叉繝闈涱儏绾惧吋绻涢幋鐐嗘垿顢旈悩缁樷拻濞达絼璀﹂悞鐐亜閹存繃鍣界紒顔芥閹兘鏌囬敃鈧▓銊╂⒑閸撴彃浜濇繛鍙夌墵閺屽宕堕浣哄幗闂佸搫鍟崑鍡涙倿閸撗呮／闁哄娉曟晥濠殿喖锕ュ钘夌暦閵婏妇绡€闁告洦鍓欏▍鎴炵節绾版ɑ顫婇柛瀣嚇閵嗗啯绻濋崶顬?-->
                    <div class="relative account-search-container">
                      <input
                        v-model="accountSearchKeyword[getEditRuleSearchKey(rule)]"
                        type="text"
                        class="input text-sm"
                        :placeholder="t('admin.groups.modelRouting.searchAccountPlaceholder')"
                        @input="searchAccountsByRule(rule, true)"
                        @focus="onAccountSearchFocus(rule, true)"
                      />
                      <!-- 闂傚倸鍊搁崐鎼佸磹閻戣姤鍤勯柛顐ｆ礀缁犵娀鏌熼幑鎰靛殭閻熸瑱绠撻幃妤呮晲鎼粹€愁潻闂佹悶鍔嶇换鍫ョ嵁閺嶎灔搴敆閳ь剚淇婇懖鈺冩／闁诡垎浣镐划闂佸搫鏈ú妯兼崲濞戞粠妲婚梺纭呮珪閸旀瑩寮鍜佺叆闁割偆鍟块幏娲煟閻斿摜鎳冮悗姘煎墴瀹曟繈濡堕崱鎰盎闂佸搫娲㈤崝宀勭嵁濡ゅ啰纾奸柛灞炬皑鏁堝銈冨灪缁嬫垿锝炲┑瀣櫜闁糕檧鏅滅紞宀勬⒒閸屾瑧顦︽繝鈧柆宥呯；闁规崘顕х粈鍫熸叏濡寧纭剧痪鎯ф健閺岀喖鏌囬敃鈧崢鎾煛閳ь剟鎳為妷锝勭盎闂佸搫绉查崝搴ㄣ€傞弻銉︾厵妞ゆ牗姘ㄦ晶娑㈡煏?-->
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
          <!-- 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柟闂寸绾惧潡鏌熺€电啸缂佲偓婵犲洦鐓冪憸婊堝礈濮樿泛桅闁告洦鍨伴崡鎶芥煕閳╁喚娈旀い蹇ｄ邯閺屾稑鈻庤箛鏇狀啋闂佸搫鐭夌紞渚€鐛崶顒夋晢闁稿本鐟фす鎶芥⒒娴ｅ憡鎲稿┑顔芥綑铻炴繝闈涱儏閽冪喐绻涢幋娆忕仼闁告濞婇弻鏇熺箾閸喖濮嶇紓浣靛妼椤嘲顫忛搹瑙勫珰闁肩⒈鍓涢濠囨⒑閸濄儱校闁绘濮撮锝嗙節濮橆厼浜滈梺鎯х箰濠€閬嶆偂閹炬枼鏀介柣鎰级椤ョ娀鏌涚€ｎ偅宕岄柛鈹垮灲瀹曞ジ濡烽敂瑙勫闂備浇宕甸崰鎾存櫠濡ゅ懎鐤柣鎰ゴ閺€浠嬫煟閹邦垼鍤嬮棅顒夊墰閳ь剚顔栭崳顕€宕戦崟顖ｆ晣闁稿繒鍘х欢鐐测攽閻樺弶鎼愰柡浣筋潐缁绘繈鎮介棃娴躲垽鏌ㄩ弴妯衡偓婵嬬嵁婵犲洤绠涢柣妤€鐗嗘禍妤呮⒑缁嬭法鐏遍柛瀣仱閸╂盯骞嬮敂瑙ｆ嫽闂佸壊鍋呯喊宥呪枍閸垻纾奸柍褜鍓涢埀顒婄秵閸犳鎮￠弴銏＄厸闁稿本绻冪涵鍫曟煟閹烘垶鍟炴い銊ｅ劦閹瑩宕ｆ径妯伙紒闂備浇顕栭崹鍗炍涢崘鈺€绻嗛柟闂寸劍閺呮繈鏌嶈閸撴氨绮嬮幒鎳ㄦ椽顢旈崨顏呭闂傚倸鍊搁悧濠冪瑹濡ゅ懏鍋傛い鎾卞灪閳锋帡鏌涢弴銊ヤ簻妞ゅ浚鍙冮幐濠傗攽閸喎鏋戦梺鍝勫暙閻楀棝鎷戦悢鍏肩叆闁绘柨鎼瓭缂備讲鍋撻悗锝庡亖娴滄粓鏌熼懜顒€濡介柛锝勫嵆閺岀喓绮欓崹顔芥濠殿喖锕ュ钘壩涢崘銊㈡婵炲棗娴氬Σ閬嶆煟鎼粹€冲辅闁稿鎹囬弻鐔兼⒒鐎靛壊妲梺?-->
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

    <!-- Delete Confirmation Dialog -->
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

    <!-- Sort Order Modal -->
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

    <!-- Group Rate Multipliers Modal -->
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
import type { AdminGroup, GroupPlatform, SubscriptionType } from '@/types'
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

// Filter options
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

// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌涘┑鍕姉闁稿鎸搁～婵嬫偂鎼粹槅娼剧紓鍌欑贰閸犳螞閸愵喖绠犳繝濠傚暊閺嬫棃鏌熺粙鍨槰闁哥喎绉瑰缁樻媴鐟欏嫬浠╅梺绋垮濡炶棄鐣峰鍐ｆ瀻闁瑰濮烽悞濂告⒑缁洖澧茬紒瀣浮閹繝寮撮姀锛勫帗闂佸疇妗ㄧ粈渚€寮抽悢鍏肩厵闁告劕寮堕崵鍥煛瀹€瀣М妞ゃ垺锕㈤幃婊堝幢濡も偓婢瑰孩绻濆▓鍨灈闁挎洏鍎遍—鍐╃鐎ｃ劉鍋撴笟鈧鎾閻欌偓濞煎﹪姊洪崘鍙夋儓闁稿﹦鏁诲鎼佸川鐎涙ǚ鎷绘繛鎾村焹閸嬫挻绻涙担鍐插悩濞戞鏃堝礃椤忓棛鍘犻柣鐔哥矌婢ф鏁幒妤€鍑犲ù锝堛€€閸嬫捇鐛崹顔煎濡炪倧瀵岄崹鍫曞蓟鐎ｎ喖鐐婇柕濞у懐妲囬梻鍌氬€搁悧濠勭矙閹烘澶愭偐鐟佷礁缍婇幃顏堝焵椤掑嫬鐭楅柛鎰靛枛缁犳牗淇婇妶鍛櫤闁稿﹦鍏橀弻锝夋偄缁嬫妫庨梺绋款儐閹告瓕鐏冮梺鍛婁緱閸橀箖鏁嶉悢鍏尖拺闂傚牊绋撴晶鏇㈡煙瀹勯偊鍎忛柕鍡樺笚缁绘繂顫濋鐘插箞闁诲骸鍘滈崑鎾绘煃瑜滈崜鐔风暦閹达附鍊烽柛婵嗗閺? 濠电姷鏁告慨鐑藉极閹间礁纾绘繛鎴欏焺閺佸銇勯幘璺烘瀾闁告瑥绻橀弻鐔虹磼閵忕姵鐏堢紓浣哄Х婵炩偓妤犵偞鐗楀蹇涘礈瑜嶉惌婵嬫⒑缁嬫鍎愰柟鐟版搐閻ｇ柉銇愰幒婵囨櫇濡炪倖甯掗崯鐘诲磻閹捐埖宕夐柕濞垮灩娴?anthropic 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬楠炲啫螖閳ь剟鍩ユ径濞炬瀻婵☆垳鍘ф慨娲⒒娴ｅ摜锛嶇紒顕呭灠铻為柛鎰靛枛閽冪喖鏌ㄩ悢鍝勑㈢紒鈧崘顔界叆婵犻潧妫欓崳绋款熆鐟欏嫭绀冪紒缁樼箞閹粙妫冨ù鑸电洴閺屾稓鈧綆鍋呯亸浼存煏閸パ冾伃妤犵偞锕㈠畷锟犳倷閸忓憡鍋呴梻浣藉吹閸犳劙鎮烽柆宥嗏挃闁告洦鍨版闂佸憡娲﹂崹浼村礃閳ь剟姊洪棃娴ゆ盯宕ㄩ姘瑨闂?claude_code_only 闂傚倸鍊搁崐鎼佸磹閻戣姤鍊块柨鏇炲€归崕鎴犳喐閻楀牆绗掔紒鈧径灞稿亾閸忓浜鹃梺閫炲苯澧撮柛鈹惧亾濡炪倖甯婄粈渚€宕甸鍕厱闁规崘娉涢弸搴ㄦ煟閿濆妫戦柟鍙夋尦瀹曠喖顢楅埀顒勫礉閸涘瓨鈷戦柟绋挎捣缁犳挻绻涚仦鍌氬闁?
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

// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌涘┑鍕姉闁稿鎸搁～婵嬫偂鎼粹槅娼剧紓鍌欑贰閸犳螞閸愵喖绠犳繝濠傚暊閺嬫棃鏌熺粙鍨槰闁哥喎绉瑰缁樻媴鐟欏嫬浠╅梺绋垮濡炶棄鐣峰鍐ｆ瀻闁瑰濮烽悞濂告⒑缁洖澧茬紒瀣浮閹繝寮撮姀锛勫帗闂佸疇妗ㄧ粈渚€寮抽悢鍏肩厵闁告劕寮堕崵鍥煛瀹€瀣М妞ゃ垺锕㈤幃婊堝幢濡も偓婢瑰孩绻濆▓鍨灈闁挎洏鍎遍—鍐╃鐎ｃ劉鍋撴笟鈧鎾閻欌偓濞煎﹪姊洪崘鍙夋儓闁稿﹦鏁诲鎼佸川鐎涙ǚ鎷绘繛鎾村焹閸嬫挻绻涙担鍐插悩濞戞鏃堝礃椤忓棛鍘犻柣鐔哥矌婢ф鏁幒妤€鍑犲ù锝堛€€閸嬫捇鐛崹顔煎濡炪倧瀵岄崹鍫曞蓟鐎ｎ喖鐐婃い鎺嶈兌閸橆亪妫呴銏℃悙妞ゆ垵鎳橀幃姗€顢旈崼鐔哄帾闂佹悶鍎崝搴ｇ不閻愮儤鐓涢悘鐐殿焾婢ц尙鈧灚婢樼€氼厾鎹㈠┑瀣妞ゆ劦鍋呭鎴︽⒒閸屾瑧鍔嶉柟顔肩埣瀹曟繆绠涢幘顖涚亙濠电偞鍨熼幊鍥焵椤掍焦顥堢€规洘锕㈤、娆撴嚃閳哄搴婇梻鍌欒兌缁垶宕濋弴銏″仱闁靛鍎弸? 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗ù锝夋交閼板潡姊洪鈧粔瀵稿閸ф鐓忛柛顐ｇ箖婢跺嫰鏌￠崱妯肩煉闁哄苯绉规俊鐑芥晜閻ｅ奔绱橀梻浣告惈椤戝懐绮旇ぐ鎺斿祦闁哄稁鐏旀惔顭戞晢闁逞屽墯娣囧﹥绂掔€ｎ偆鍘遍梺闈涚墕閹锋垿寮ㄦ繝姘厪闁搞儜鍐句紓缂備胶濮甸惄顖炵嵁濮椻偓瀵爼骞嬮悙鍏告倣闂?

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

// 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偞鐗犻、鏇㈡晝閳ь剟鎮块鈧弻锝呂旈埀顒勬偋婵犲洤鐭楅煫鍥ㄧ⊕閻撶喖鏌曡箛瀣労闁绘帡绠栭弻锝夊Χ婢跺瞼鏆梺鍝勬湰濞叉绮╅悢纰辨晬婵﹩鍓︽禒濂告⒑缂佹绠戦柛妤€鍟块～蹇曠磼濡顎撻梺鍛婄缚閸庝即鎼规惔鈽嗘富闁靛牆妫楅悘銉︾箾鐏炲倸鈧繈寮崘顔碱潊闁靛牆鎳愰鎺戭渻閵堝棙鈷掗柡鍜佸亰瀹曘垽鏌嗗鍡╂濡炪倖鍔戦崹鐑樺緞閸曨剚鍙忓┑鐘插亞閻撳ジ鏌熼姘拱缂佺粯绻堝畷鐔碱敇閻欏懐搴婇梻鍌欑窔濞佳団€﹂鐘典笉闁硅揪绠戦悡婵嬫煛閸モ晛袥闁稿鎸鹃幉鎾礋椤掑偆妲伴柣搴″帨閸嬫挾绱掔€ｎ厽纭堕柡鍡畵閺屾稑螖閸愩劋姹楅梺鍝勵儏闁帮綁寮婚悢琛″亾閻㈡鐒鹃崯鎼佹⒑閸涘鎴﹀垂閸噮娼栧┑鐘宠壘绾惧吋绻涢崱妯虹伇缂佹劖顨婂铏圭矙濞嗘儳鍓伴柣搴㈢婵棄螞閻斿吋鈷戞慨鐟版搐閻忣喗銇勯鐐靛ⅲ缂佺粯宀搁弻锝嗘償閿涘嫮鏆涢梺绋块叄娴滄儼妫熸繛鏉戝悑濞兼瑩鎮块悙顑句簻闁圭儤鍨甸顏堟煟閹惧瓨绀冮柕鍥у楠炲洭宕滄担鐟颁还缂傚倷鑳舵慨鐢稿箲閸ヮ剙钃熼柨婵嗙墢閻も偓闂侀潧锛忛崪鍐惞闂傚倷绶氶埀顒傚仜閼活垱鏅舵导瀛樼厱閻庯絽澧庣粔顔锯偓瑙勬礃閸旀洝鐏冮梺鍛婁緱閸橀箖鏁嶉悢鍏尖拺闂傚牊绋撴晶鏇㈡煙瀹勯偊鍎忛柕鍡樺笚缁绘繂顫濋鐘插箞闁诲骸鍘滈崑鎾绘煃瑜滈崜鐔风暦閹达附鍊烽柛婵嗗閺? 濠电姷鏁告慨鐑藉极閹间礁纾绘繛鎴欏焺閺佸銇勯幘璺烘瀾闁告瑥绻橀弻鐔虹磼閵忕姵鐏堢紓浣哄Х婵炩偓妤犵偞鐗楀蹇涘礈瑜嶉惌婵嬫⒑缁嬫鍎愰柟鐟版搐閻ｇ柉銇愰幒婵囨櫇濡炪倖甯掗崯鐘诲磻閹捐埖宕夐柕濞垮灩娴?anthropic 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶閸儱绾ч柍鍝勫€荤粻鏍磼缂佹娲存鐐达耿瀵爼骞嬮悩鎻捫犻梻鍌欒兌椤牓顢栭崱娑樼闁煎鍊栧畷鍙夌箾閹寸偠澹橀柡瀣墦閺屾稑鈻庨幇鐢靛姺闂佽桨鐒﹂悧鐘差潖濞差亝鍋￠柡澶嬪浜涙俊鐐€栭崹鐢杆囨导鏉戠疄闁靛ň鏅涢悞鍨亜閹烘垵顏柣鎾崇箻閺屾盯顢曢敐鍥╃暫闂佹寧绋戠粔鐢垫崲濞戙垹绀傞柣鎾抽椤秶绱撴担铏瑰笡闁烩晩鍨堕悰顔锯偓锝庡枟閸婄兘鏌℃径瀣伌闁诲海鍎ょ换婵嗏枔閸喗鐏堥梺鎸庢穿缁犳捇鐛繝鍥х缂備焦蓱濞呭洭姊洪棃娑氱疄闁稿鍊濆顐﹀炊椤掍胶鍘棅顐㈡储閸庡磭澹曢崸妤佺厱婵☆垵顕ф慨宥夋煙椤旂瓔娈滈柣娑卞櫍瀹曞綊顢欓悡搴經濠电姷鏁搁崑娑㈡儑娴兼潙绀夐柟瀛樼箥閸ゆ洟鏌熼梻瀵割槮缁炬儳銈搁弻锝夊箛椤掑娈堕梺鍛婏耿娴滆泛螞閸涙惌鏁冮柕蹇娾偓鎰佹П婵犳鍠栭敃銈夆€﹂悜钘夌畾閻忕偛澧界弧鈧梺鍛婃处閸橀箖鏁嶅鍫熲拺闂傚牊渚楅悡顓犵磼閻樺啿鐏撮柛鈹垮灲瀵挳濮€閿涘嫬骞嶆俊鐐€栭悧妤呮嚌妤ｅ啫姹查柨婵嗘礌閸嬫挾鎲撮崟顒傤槶闂佸摜濮甸悧鐘诲Υ娓氣偓瀵挳濮€閻欌偓濞煎﹪姊洪崘鑼邯闁哄懏绋掔粋宥嗐偅閸愨晝鍘炬繝娈垮枟閸旀洟鍩€椤掍緡娈樺?
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

// 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偞鐗犻、鏇㈡晝閳ь剟鎮块鈧弻锝呂旈埀顒勬偋婵犲洤鐭楅煫鍥ㄧ⊕閻撶喖鏌曡箛瀣労闁绘帡绠栭弻锝夊Χ婢跺瞼鏆梺鍝勬湰濞叉绮╅悢纰辨晬婵﹩鍓︽禒濂告⒑缂佹绠戦柛妤€鍟块～蹇曠磼濡顎撻梺鍛婄缚閸庝即鎼规惔鈽嗘富闁靛牆妫楅悘銉︾箾鐏炲倸鈧繈寮崘顔碱潊闁靛牆鎳愰鎺戭渻閵堝棙鈷掗柡鍜佸亰瀹曘垽鏌嗗鍡╂濡炪倖鍔戦崹鐑樺緞閸曨剚鍙忓┑鐘插亞閻撳ジ鏌熼姘拱缂佺粯绻堝畷鐔碱敇閻欏懐搴婇梻鍌欑窔濞佳団€﹂鐘典笉闁硅揪绠戦悡婵嬫煛閸モ晛袥闁稿鎸鹃幉鎾礋椤掑偆妲伴柣搴″帨閸嬫挾绱掔€ｎ厽纭堕柡鍡畵閺屾稑螖閸愩劋姹楅梺鍝勵儏闁帮綁寮婚悢琛″亾閻㈡鐒鹃崯鎼佹⒑閸涘鎴﹀垂閸噮娼栧┑鐘宠壘绾惧吋绻涢崱妯虹伇缂佹劖顨婂铏圭矙濞嗘儳鍓伴柣搴㈢婵棄螞閻斿吋鈷戞慨鐟版搐閻忣喗銇勯鐐靛ⅲ缂佺粯宀搁弻锝嗘償閿涘嫮鏆涢梺绋块叄娴滄儼妫熸繛鏉戝悑濞兼瑩鎮块悙顑句簻闁圭儤鍨甸顏堟煟閹惧瓨绀冮柕鍥у楠炲洭顢欐径骞垮灲閺岋綀绠涢敐鍛亶婵烇絽娲ら敃顏堝箖濞嗘挸绠ユい鏂垮⒔閸橆垶鏌ｆ惔銏╁晱闁哥姵鐗犻幃銉╂偂鎼达絾娈炬繝闈涘€搁幉锟犲磻閸曨偒娓婚悗锝庝簽閸戣淇婂Δ浣瑰鞍缂佺粯绻勯崰濠冨緞瀹€鈧敍鐔兼⒑缁嬫鍎愰柣鈺婂灠閻ｇ兘骞囬弶鍧楁暅濠德板€愰崑鎾绘煛? 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗ù锝夋交閼板潡姊洪鈧粔瀵稿閸ф鐓忛柛顐ｇ箖婢跺嫰鏌￠崱妯肩煉闁哄苯绉规俊鐑芥晜閻ｅ奔绱橀梻浣告惈椤戝懐绮旇ぐ鎺斿祦闁哄稁鐏旀惔顭戞晢闁逞屽墯娣囧﹥绂掔€ｎ偆鍘遍梺闈涚墕閹锋垿寮ㄦ繝姘厪闁搞儜鍐句紓缂備胶濮甸惄顖炵嵁濮椻偓瀵爼骞嬮悙鍏告倣闂?

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
  // 闂傚倸鍊搁崐鎼佸磹閻戣姤鍊块柨鏇炲€哥粻鏍煕椤愶絾绀€缁炬儳娼″鍫曞醇濞戞ê顬夊┑鐐叉噽婵炩偓闁哄被鍊濋獮渚€骞掗幋婵嗩潥婵犵數鍋涢幊鎰箾閳ь剟鏌＄仦绯曞亾閹颁礁鎮戦柟鑲╄ˉ閳ь剙纾鎴︽⒒娴ｄ警鐒炬い鎴濆暣瀹曟繈骞嬮敃鈧拑鐔兼煥濠靛棭妲哥紒顐㈢Ч閺屾稓浠︾拠娴嬪亾濡ゅ懎绀夐柟闂寸劍閳锋垿鎮归崶顏勭毢缂佺姷澧楃换娑橆啅椤旂厧绫嶅Δ鐘靛仜閸熶即骞夐幘顔肩妞ゆ劦鍋勫▓鍏间繆閻愵亜鈧牠寮婚妸鈺佺妞ゆ劧绠戦悞鍨亜閹哄秶鍔嶇紒鈧€ｎ喗鐓欐い鏃傜摂濞堟粍銇勯姀鈽呰€垮┑顔瑰亾闂佹娊鏁崑鎾绘煙闁垮銇濇慨濠冩そ瀹曨偊宕熼崹顐嶎亜鈹戦悙宸Ч婵炲弶绻堝畷鎰板箻椤旇В鎷绘繛杈剧到濠€鍗烇耿娴犲鐓曞┑鐘插暞缁€鈧柧鑽ゅ仦缁绘繈妫冨☉鍗炲壈闂佸搫顑勯懗鍫曞焵椤掆偓缁犲秹宕曢柆宓ュ洭顢涢悙瀵稿帒闂佹悶鍎崝搴ｅ姬閳ь剚绻濋悽闈浶㈤柛濠傜秺瀹曡櫕绂掔€ｎ偆鍘?antigravity 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬楠炲啫螖閳ь剟鍩ユ径濞炬瀺闁惧浚鍋勯悘锕傛煕閹烘埊韬鐐达耿椤㈡瑩鎸婃径澶婂闂傚倷娴囬鏍储閻ｅ本鏆滈柟鐑樻尫缁诲棗霉閻樺樊鍎愰柣鎾寸洴閺屾盯鍩勯崘鐐暭缂備胶濮撮…鐑藉蓟?
  image_price_1k: null as number | null,
  image_price_2k: null as number | null,
  image_price_4k: null as number | null,
  // Claude Code 闂傚倸鍊搁崐宄懊归崶顒夋晪鐟滃酣銆冮妷鈺佷紶闁靛／鍌滅憹闁诲骸绠嶉崕閬嵥囬鐐插瀭闁稿本绋撶粻鍓р偓鐟板閸犳洜鑺辨繝姘厸闁告洍鏅涢崝婊呯磼缂佹娲存鐐差儔閹瑧鈧潧鎲￠濠氭⒒娴ｅ憡鍟炴い銊ユ閸犲﹤顓兼径濞箓鏌涢弴銊ョ仩闁告劏鍋撴俊鐐€栭崝锕€顭块埀顒佺箾瀹€濠佺盎妞ゎ亜鍟存俊鍫曞川椤撗冨Ψ闂備浇宕甸崯鍧楀疾閻樺樊鍤曢悹鍥ㄧゴ濡插牓鏌曡箛鏇炐ユい鏂匡躬濮婃椽宕崟顒€鍋嶉梺鍛婃煥閻倸顕ｉ弻銉︽櫜闁搞儮鏅濋敍婵囩箾鏉堝墽瀵肩紒顔界懇瀹曨偄煤椤忓懐鍘介梺缁樻礀閸婃悂銆呴鍌滅＜?anthropic 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬楠炲啫螖閳ь剟鍩ユ径濞炬瀺闁惧浚鍋勯悘锕傛煕閹烘埊韬鐐达耿椤㈡瑩鎸婃径澶婂闂傚倷娴囬鏍储閻ｅ本鏆滈柟鐑樻尫缁诲棗霉閻樺樊鍎愰柣鎾寸洴閺屾盯鍩勯崘鐐暭缂備胶濮撮…鐑藉蓟?
  claude_code_only: false,
  fallback_group_id: null as number | null,
  fallback_group_id_on_invalid_request: null as number | null,
  // OpenAI Messages 闂傚倸鍊搁崐宄懊归崶褏鏆﹂柛顭戝亝閸欏繘鏌熺紒銏犳珮闁轰礁瀚伴弻娑樷槈濞嗘劗绋囬梺姹囧€ら崰妤呭Φ閸曨垰绠涢柍杞拌閸嬫捇骞囬弶璺紱闂佽宕橀崺鏍窗閸℃稒鐓曢柡鍥ュ妼娴滄劙鏌＄€ｎ偆鎳囨慨濠冩そ瀹曨偊宕熼纰变純闂備焦瀵уú蹇涘磹濠靛鈧礁螣閼姐倝妾紓浣割儓濞夋洟宕愰悙鐑樷拺闁告劕寮堕幆鍫ユ煕婵犲偆鐓奸柛鈹惧亾濡炪倖甯掗ˇ顖氼嚕椤旇姤鍙忓┑鐘插暞閵囨繃顨ラ悙鏉戝闁诡垱妫冮弫鎰板磼濞戣鲸缍岄梻鍌氬€烽懗鍓佸垝椤栫偑鈧啴宕ㄩ弶鎴犵枃闂佸湱澧楀妯肩矆?openai 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬楠炲啫螖閳ь剟鍩ユ径濞炬瀺闁惧浚鍋勯悘锕傛煕閹烘埊韬鐐达耿椤㈡瑩鎸婃径澶婂闂傚倷娴囬鏍储閻ｅ本鏆滈柟鐑樻尫缁诲棗霉閻樺樊鍎愰柣鎾寸洴閺屾盯鍩勯崘鐐暭缂備胶濮撮…鐑藉蓟?
  allow_messages_dispatch: false,
  default_mapped_model: 'gpt-5.4',
  // 婵犵數濮烽弫鍛婃叏閻戝鈧倿鎸婃竟鈺嬬秮瀹曘劑寮堕幋婵堚偓顓烆渻閵堝懐绠伴柣妤€妫濋幃鐐哄垂椤愮姳绨婚梺鐟版惈濡绂嶉崜褏纾奸柛鎾楀棙顎楅梺鍛婄懃閸熸潙鐣峰ú顏勭劦妞ゆ帊闄嶆禍婊堟煙閻戞ê鐏ユい蹇撶摠娣囧﹪顢曢敐鍛紝闂佸搫鐬奸崰鏍箖閻戣棄绾ф繛鍡楀綖缁辨梹淇婇悙顏勨偓鎴﹀垂閾忓湱鐭欓柟杈捐吂閳ь剨绠撴俊鎼佸煛娴ｄ警妲规俊鐐€栫敮濠勬閿熺姴鐤柛娑樼摠閳锋垿鎮归崶銊ョ祷妞ゆ帇鍨洪妵鍕籍閳ь剟鎮ч悩鑼殾濞村吋娼欑粻铏繆閵堝倸浜剧紒鐐劤椤兘寮婚悢鐓庣鐟滃繒鏁☉銏＄厽?
  model_routing_enabled: false,
  // 闂傚倸鍊搁崐鎼佸磹瀹勬噴褰掑炊椤掆偓杩濋梺閫炲苯澧撮柡灞剧〒閳ь剨缍嗛崑鍛焊娴煎瓨鐓忛柛銉戝喚浼冮悗娈垮枙缁瑦淇婂宀婃Х濠碘剝褰冮悧鎾愁潖閻戞ê顕辨繛鍡楃箳濡诧綁姊洪棃鈺冪Ф缂傚秳绀侀锝嗙節濮橆厽娅滄繝銏ｆ硾璋╅柍鍝勬噺閻撳繐顭跨捄铏瑰闁告梹娼欓湁闁绘宕甸悾娲煛鐏炶鈧繂鐣烽悷鎵虫婵炲棙鍔楄ぐ瀣⒒娴ｅ憡鎯堥柡鍫墴閹嫰顢涘☉妤冪畾闂佸綊妫跨粈浣告暜婵＄偑鍊栧濠氬储瑜庣粩鐔衡偓锝庡枟閳锋帡鏌涚仦鍓ф噮閻犳劒鍗抽弻娑㈡偐閹颁焦鐤佸Δ鐘靛仦閸旀瑥鐣峰鈧幊鐘活敆娴ｈ鍟庡┑鐘愁問閸犳鏁冮埡鍛？闁汇垻顭堢猾宥夋煕鐏炵虎娈斿ù婊堢畺閺屻劌鈹戦崱娑扁偓妤€顭胯婢ф濡甸崟顖涙櫆閻熸瑥瀚悵鏇犵磽?antigravity 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷?
  supported_model_scopes: ['claude', 'gemini_text', 'gemini_image'] as string[],
  // MCP XML 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸弫鎾绘偐椤旂懓浜鹃柛鎰靛枛瀹告繃銇勯弽銊р槈閹兼潙锕ら埞鎴炲箠闁稿﹥娲熼獮蹇曗偓锝庡枛閺嬩礁鈹戦悩鍙夊闁绘挻娲樼换娑㈠箣濠靛棜鍩炲Δ鐘靛仦閸旀瑩寮婚敍鍕ㄥ亾閿濆骸浜炲褏鏁婚弻锛勪沪閻愵剛顦ㄧ紓浣虹帛閻╊垰鐣疯ぐ鎺濇晩闁绘挸瀵掑娑㈡⒑鐠囨彃顒㈡い鏃€鐗犲畷浼村冀椤愩埄妫滃銈嗘尪閸ㄥ綊鎮為崹顐犱簻闁瑰搫绉剁拹浼存煕閻旈绠婚柡灞剧洴閹晠宕橀崣澶庣檨闁诲氦顫夊ú姗€宕归崸妤冨祦婵☆垵鍋愮壕鍏间繆椤栨粌甯舵鐐茬墦濮婄粯绻濇惔鈥茬盎濠电偠顕滅粻鎴犲弲闂佹寧娲栭崐鍝ョ矆婢跺绡€濠电姴鍊归崳鐣岀棯閹佸仮鐎殿喖鐖煎畷鐓庘槈濡警鐎烽梻浣烘嚀閸熻法鈧凹鍠氬Σ?antigravity 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷?
  mcp_xml_inject: true,
  // 濠电姷鏁告慨鐑藉极閹间礁纾绘繛鎴欏焺閺佸銇勯幘璺烘瀾闁告瑥绻橀幃妤€鈽夊▎娆庣返濠电偛鐗呯划娆撳蓟閻斿吋鈷掗悗闈涘濡差噣姊洪幖鐐插闁稿﹤娼￠悰顕€寮介妸锔剧Ф闂佸憡鎸嗛崟顐¤繕缂傚倷鑳堕崑鎾诲磿閹惰棄围闁归棿绀侀拑鐔哥箾閹存瑥鐏╅柛妤佸▕閺屾洘绻涢崹顔煎闂佺厧澹婃禍婊堝煘閹达箑鐒洪柛鎰╁妿缁佸嘲顪冮妶搴″箻闁稿繑锚椤曪絿鎷犲ù瀣潔闂侀潧绻掓慨鐑筋敊婵犲洦鈷戦悷娆忓閸斻倝鏌涢悢绋款嚋闁逛究鍔戝畷銊︾節閸曨厾妲囬梻渚€娼ф蹇曞緤閸撗勫厹濡わ絽鍟崐鍨叏濮楀棗骞楃紒鑸电叀閺?
  copy_accounts_from_group_ids: [] as number[]
})

// 缂傚倸鍊搁崐鎼佸磹閹间礁纾归柣鎴ｅГ閸ゅ嫰鏌ょ粙璺ㄤ粵闁告瑥绻戦妵鍕箻閸楃偟浠肩紒鐐劤椤兘寮婚悢鐓庣鐟滃繒鏁☉銏＄厽闁规儳鐡ㄧ粈鍐ㄇ庨崶褝韬┑鈥崇埣瀹曠喖顢橀悙宸€寸紓鍌氬€风拋鏌ュ磻閹炬剚鐔嗛悹杞拌閻擃剟鏌ｉ弬鎸庮棦闁诡喖缍婂畷顐﹀礋椤掍礁鍓垫俊鐐€曞ù姘跺储妤ｅ啫鐒垫い鎺戝枤濞兼劖绻涘ù瀣珖鐎垫澘锕ラ妶锝夊礂椤栨碍鍠橀柟顔ㄥ洤閱囬柣鏂垮槻婵℃娊姊绘笟鈧褔鎮ч崱娆屽亾濞戞帗娅婇柟椋庡劋瀵板嫰骞囬娑欏濠电偠鎻紞鈧い顐㈩樀瀹曟垿鎮╃紒妯煎幈闂佸搫鍊藉▔鏇㈡倿閸涘浜滈柕蹇ョ磿婢у灚顨ラ悙宸剶闁诡喒鏅犻獮鍥ㄦ媴閸︻厾鈼ユ繝鐢靛Х閺佹悂宕戝☉妯滄稑鈻庨幘宕囶唶婵犵數濮甸懝鍓у婵犳碍鐓忓┑鐐靛亾濞呭棝鏌ｉ幘宕囩闁宠鍨块幃娆撴嚑椤戣儻妾搁梻渚€鈧偛鑻晶顖涖亜閵娿儻韬鐐诧工閳规垹鈧綆鍋勯埀顒勬敱閵囧嫰骞嬮敐鍡欍€婄紓浣哄У閼归箖鈥旈崘顔嘉ч柛鈩冪懃椤囨⒑闂堟稒澶勯柣鈺婂灠閻ｇ兘骞嬮敃鈧粻鑽も偓瑙勬礀濞诧箑鈻撴ィ鍐┾拺闁硅偐鍋涢崝鐢电磼閼艰泛袚缂佸倸绉归獮鍡氼檨婵炴挸顭烽幃妤呮晲鎼存繄鐩庡銈嗗竾閸ㄥ鍩€椤掑喚娼愭繛鍙夌墵钘濆ù鍏兼綑妗呴梺鍛婃处閸ㄤ即宕橀埀顒勬⒑闂堟丹娑㈠川椤栨艾绗掗梻鍌氬€风粈渚€骞栭鈶芥稑鈽夊▎鎴狀啎闂侀€炲苯澧柟渚垮妽缁绘繈宕橀埞澶歌檸婵＄偑鍊戦崹鍝劽洪悢鍛婂弿闁逞屽墴閺屽秹宕崟顐熷亾閹邦垼妯勯梺鍝勭焿缁查箖骞嗛弮鍫濐潊闁绘娅曢崕褔姊绘担鍛婂暈婵﹤缍婂畷褰掑垂椤曞懎娈?
interface SimpleAccount {
  id: number
  name: string
}

// 婵犵數濮烽弫鍛婃叏閻戝鈧倿鎸婃竟鈺嬬秮瀹曘劑寮堕幋婵堚偓顓烆渻閵堝懐绠伴柣妤€妫濋幃鐐哄垂椤愮姳绨婚梺鐟版惈濡绂嶉崜褏纾奸柛鎾楀棙顎楅梺鍛婄懃閸熸潙鐣峰ú顏勭劦妞ゆ帊闄嶆禍婊堟煙閻戞ê鐏ユい蹇撶摠娣囧﹪顢曢敐鍛紝闂佸搫鐬奸崰鏍箖閻戣棄绾ф繛鍡楀綖缁辨梹淇婇悙顏勨偓鎴﹀垂閾忓湱鐭欓柟杈捐吂閳ь剨绠撴俊鎼佸煛娴ｄ警妲规俊鐐€栫敮濠勬閿熺姴鐤柛娑樼摠閳锋帒霉閿濆牊顥夐柛姘秺閺屾盯鎮╅崘鎻掝潚闂佽鍠氶崗妯讳繆閻ゎ垼妲烽梺绋款儐閹告悂锝炲┑瀣亗閹肩补妾ч幏顐︽煟鎼淬値娼愭繛鍙夌墱缁辩偞绻濋崶銉㈠亾娴ｇ硶妲堟慨妤€妫涢崣鍡涙⒑閸涘﹣绶遍柛妯挎閳绘捇顢曢敂瑙ｆ嫼闂佺厧顫曢崐鏇炵摥婵犵數鍋炵粊鎾疾濠靛洨顩茬紒瀣氨閺嬪酣鏌熼幆褏锛嶉柛?

interface ModelRoutingRule {
  pattern: string
  accounts: SimpleAccount[] // 闂傚倸鍊搁崐鎼佸磹閹间礁纾瑰瀣椤愪粙鏌ㄩ悢鍝勑㈢痪鎯ь煼閺屾盯寮撮妸銉р偓顒勬煕閵夘喖澧婚柡浣割儔閺屽秷顧侀柛鎾跺枛閻涱喗绻濋崒妤佺€婚梺瑙勫劤绾绢參顢樺ú顏呪拺闁圭瀛╅ˉ鍡樹繆椤愩垹顏€规洘绮撻、姗€濮€閳锯偓閹锋椽姊洪崨濠勨槈闁挎洏鍎插鍕礋椤栨稓鍘遍梺缁樏壕顓熸櫠閻㈠憡瀵犳繝闈涱儐閻撳啴鏌涘┑鍡楊仾闁革絾妞介弻锝夊箻鐎涙顦ㄧ紓浣虹帛閻╊垰鐣烽崡鐐嶆棃宕橀宥呮暪濠电姴鐥夐妶鍛缂備緡鍣崹鑸典繆閻㈢绀嬫い鏍ㄨ壘瀹撳棝姊洪棃娑辩劸闁稿孩鐓￠幃姗€骞樼紒妯锋嫼闂佽崵鍠撴晶妤呭疮閻愮儤鍋ㄦい鏍ュ€楃弧鈧悗瑙勬礃濡炶棄鐣烽悢纰辨晬婵﹢纭稿Σ浼存⒒娴ｇ鏆遍柟纰卞亰瀹曟垿宕卞☉妯兼焾闂侀€炲苯澧扮紒?
}

// 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸弫宥夊礋椤掍焦顔囨繝寰锋澘鈧洟宕导瀛樺剹婵炲棙鎸婚悡娆撴倵閻㈡鐒鹃崯鍝ョ磼閹冪稏缂侇喗鐟╁濠氭偄閻撳海顔夐梺褰掑亰閸撴盯宕㈤锔解拺缂佸顑欓崕鎰版煥閺囨ê鐏茬€殿喖顭峰鎾偄閾忚鍟庨梻浣稿閻撳牓宕伴弽銊﹀弿閹艰揪绲跨壕钘壝归敐鍡楃祷濞存粓绠栭弻锝嗘償椤栨粎校闂佸摜濮甸〃濠囧箖閿熺姵鍋勯柛蹇氬亹閸橆亝绻濋悽闈涗粶闁诲繑绻堝畷婵嗩潩椤撴粈绨婚棅顐㈡祫缁插ジ鏌囬鐐寸厵闁惧浚鍋勯崫娲煕閳哄绡€鐎规洏鍔戦、娆撳箚瑜嶇粻鐐烘⒒閸屾瑧绐旀繛浣冲洠鈧箓宕奸妷銉э紵濠电偛妯婃禍婊呭閸ф鐓欓柛鎾楀懎绗￠梺缁樻尵閸犳劙濡甸崟顖氱闁告鍋熸导鍕⒑娴兼瑧绋绘俊鐐扮矙瀵鎮㈢喊杈ㄦ櫓闂佸壊鐓堥崯鈺呭箻鐎靛摜顔曢梺缁樻尭濞撮绮荤紒妯镐簻闁靛繆鍓濈粈瀣亜閵忊剝绀嬮柡浣瑰姍瀹曞爼鍩為鎯у姦婵?
const createModelRoutingRules = ref<ModelRoutingRule[]>([])

// 缂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳婀遍幉鎼佹偋閸繄鐟查梺绋款儏椤戝寮婚敐澶婄疀妞ゆ帒鍊风划闈涱渻閵堝棗濮屽┑顔哄€濇俊鐢稿礋椤栨氨鐤€闂佸疇妗ㄧ拋鏌ュ磻閹捐鍐€妞ゆ挾鍋熼敍娑樷攽椤斿浠滈柛瀣崌閺岋紕浠﹂悙顒傤槰缂備胶绮惄顖氱暦瑜版帩鏁婇柣鎾冲瘨濞兼稓绱撻崒姘偓椋庢媼閺屻儱纾婚柟鍓х帛閸婄敻鏌ㄥ┑鍡涱€楅柡瀣枛閺岋綁骞樼捄鐑樼亪闂佸搫鏈ú妯兼崲濞戙垹鍨傛い鏃傚帶椤挻淇婇悙顏勨偓鏇灻瑰顓狀洸妞ゅ繐鐗嗙粻鐘虫叏濡寧纭鹃柣鎾寸洴閺屾稑鈽夐崡鐐寸亾闂佸憡锕㈡禍璺侯潖濞差亝顥堟繛鎴ｄ含閸旀悂姊虹粙鍨劉婵＄偠妫勯悾鐑藉即閵忕姷顔掔紓鍌欑劍椤洭宕㈤挊澶樻富闁靛牆妫欑亸顏堟煕閵娿儺鐓奸柟顔藉劤鐓ゆい蹇撴噽閸橀潧顪冮妶鍡樼叆闁诲繑绻嗛。鎸庣節閻㈤潧浠ч柛妯犲懏宕叉慨妞诲亾闁靛棔绶氬鎾閳哄倹娅囬梻浣瑰缁诲倸煤閵娿儲姣勯梻?
const editModelRoutingRules = ref<ModelRoutingRule[]>([])

// 闂傚倸鍊搁崐宄懊归崶褏鏆﹂柣銏㈩焾缁愭鏌熼幍顔碱暭闁稿绻濋弻鏇熺珶椤栨浜鹃梺绋款儐閹告悂锝炲┑瀣亗閹肩补妾ч幏顐︽煟鎼淬値娼愭繛鍙夌墱缁辩偞绻濋崶銉㈠亾娴ｇ硶妲堟俊顖炴敱閺佺娀姊虹拠鈥崇€婚柛灞惧嚬濡粓姊婚崒娆戝妽闁活亜缍婂畷鐟懊洪宥嗘櫓闂佸搫绋侀崢鑲╃磼閳轰急褰掓偂鎼达絾鎲奸梺缁樻尪閸ㄤ粙寮婚垾宕囨殼妞ゆ梻鍘ч弸銈吤圭亸鏍т壕缂傚倸鍊搁崐椋庢閿熺姴绐楁俊銈呮噺閸嬶繝鏌嶆潪鎷屽厡闁哄棴闄勯幈銊ヮ渻鐠囪弓澹曢梻浣烘嚀绾绢厽绻涢埀顒併亜閵忊槅娈曢柟宄版嚇瀹曠兘顢橀悢?key闂傚倸鍊搁崐鎼佸磹閻戣姤鍊块柨鏃堟暜閸嬫挾绮☉妯诲櫧闁活厽鐟╅弻鐔告綇妤ｅ啯顎嶉梺绋款儏椤戝寮婚悢鍏煎€锋い鎺嶈兌缁嬪洤顪冮妶鍡樼濞存粌鐖煎璇差吋婢跺﹣绱堕梺鍛婃处閸撴瑥鈻嶉敐鍥╃＝濞达絾褰冩禍楣冩⒑缁嬫寧婀板瑙勬礋瀹曟垿骞橀懡銈呯ウ闂佸壊鐓堥崰鏍ㄦ叏鎼淬劍鐓熼幖娣灩閸ゎ剟鏌涢幘璺烘灈鐎规洘妞芥慨鈧柨婵嗘川閺夌鈹戦悙鏉戠仸闁荤啙鍥у偍?index 闂傚倸鍊搁崐宄懊归崶顒夋晪鐟滃繘鎳為柆宥嗗殐闁宠桨鑳剁粵蹇曠磽閸屾瑧鍔嶆い顓炴喘閹敻宕奸弴鐔哄幈濡炪倖鍔楁慨鎾礉濮樿埖鐓曟慨姗嗗墯鐠愶繝鏌熼崣澶嬪€愮€殿噮鍣ｅ畷鐓庘攽閸垺姣囬梻鍌欑閸熷潡骞栭锕€绠犻煫鍥ㄧ☉閸氬綊鎮楅敐搴″婵☆偒鍨堕弻锝呂旈埀顒勬偂椤愶富鏁傞柛娑卞灱濞叉悂姊虹拠鈥崇€诲ù锝嗗絻娴滈箖鏌熼悜妯虹劸婵炲皷鏅犻弻銊モ攽閸℃ê娅㈡繝銏ｎ潐濞茬喎顫忕紒妯诲闁兼亽鍎埀顒€鍟扮槐鎺楀焵椤掍胶鐟归柍褜鍓熷顐﹀礃椤斿槈銊╂煃鏉炴媽鍏屾い鏂挎濮婃椽宕ㄦ繝浣虹箒闂佸憡鐟ラ柊锝呯暦閵壯€鍋撻敐搴℃灓闁告瑦鎹囬弻娑㈠Ψ閿濆懎顬夌紓浣插亾闁割偆鍠嗘禍?
const resolveCreateRuleKey = createStableObjectKeyResolver<ModelRoutingRule>('create-rule')
const resolveEditRuleKey = createStableObjectKeyResolver<ModelRoutingRule>('edit-rule')

const getCreateRuleRenderKey = (rule: ModelRoutingRule) => resolveCreateRuleKey(rule)
const getEditRuleRenderKey = (rule: ModelRoutingRule) => resolveEditRuleKey(rule)

const getCreateRuleSearchKey = (rule: ModelRoutingRule) => `create-${resolveCreateRuleKey(rule)}`
const getEditRuleSearchKey = (rule: ModelRoutingRule) => `edit-${resolveEditRuleKey(rule)}`

const getRuleSearchKey = (rule: ModelRoutingRule, isEdit: boolean = false) => {
  return isEdit ? getEditRuleSearchKey(rule) : getCreateRuleSearchKey(rule)
}

// 闂傚倸鍊搁崐宄懊归崶褏鏆﹂柛顭戝亝閸欏繘鏌ｉ姀銏╃劸缂佲偓婢跺本鍠愰煫鍥ㄦ礀閸ㄦ繂鈹戦悩瀹犲缂佺媴缍侀弻銊モ攽閸℃娈ㄥ┑顔款潐椤ㄥ﹤顫忛搹鍦煋闁糕剝顨呴瀛樼箾閸喐鈷掔紒杈ㄥ浮閹晜绗熼娑氱Х缂傚倷鑳剁划顖滄崲閸愵亝宕叉繝闈涱儏绾惧吋绻涢幋鐐嗘垿顢旈悩缁樷拻濞达絼璀﹂悞鐐亜閹存繃鍣界紒顔芥閵囨劙骞掗幋锝嗘啺闂備焦瀵х换鍌炈囬鐐村亗婵炴垯鍨洪悡鏇㈡煙闁箑澧柍閿嬫閺屾盯寮埀顒勫箖閸屾凹娼栨繛宸簻瀹告繂鈹戦悩鑼嚬缂佹墎鏅犲娲箹閻愭彃濮岄梺鍛婃煥閺堫剛绮嬪鍫涗汗闁圭儤鎸撮幏娲⒑缂佹ɑ灏悗娑掓櫊楠炲宕ㄦ繛澶哥盎闂佹寧绻傜€氼噣鎯屽▎鎴斿亾?
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

// 闂傚倸鍊搁崐鎼佸磹閻戣姤鍤勯柛顐ｆ礀缁犵娀鏌熼幑鎰靛殭閻熸瑱绠撻幃妤呮晲鎼粹€愁潻闂佹悶鍔嶇换鍫ョ嵁閺嶎灔搴敆閳ь剚淇婇懖鈺冩／闁诡垎浣镐划闂佸搫鏈ú妯兼崲濞戞粠妲婚柣搴㈠嚬閸撶喖銆侀弮鍫濈濞达絽鍘滈幏娲⒑閸︻収鐒鹃悗娑掓櫇婢规洟鎮介崨濠勫幈闁诲函缍嗛崑鍛焊閻㈠憡鐓欓柛娆忣槹閸婃劗鈧鍠栭悥濂哥嵁閹邦厽鍎熼柍銉ュ级濡﹪姊婚崒娆愮グ婵℃ぜ鍔庣划鍫熸媴閾忓湱鐓嬮梺鑽ゅ枑婢瑰棝寮抽敃鍌涚厽闁规儳鍟块惁銊╂煟閹惧崬鍔﹂柡灞剧☉閳藉宕￠悙鍏稿寲闂備礁鎲¤摫閻㈩垪鈧剚娼?anthropic 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷?
const searchAccounts = (key: string) => {
  accountSearchRunner.trigger(key, accountSearchKeyword.value[key] || '')
}

const searchAccountsByRule = (rule: ModelRoutingRule, isEdit: boolean = false) => {
  searchAccounts(getRuleSearchKey(rule, isEdit))
}

// 闂傚倸鍊搁崐鎼佸磹閹间礁纾瑰瀣椤愪粙鏌ㄩ悢鍝勑㈢痪鎯ь煼閺屾盯寮撮妸銉р偓顒勬煕閵夘喖澧紒鐘劜閵囧嫰寮崒娑樻畬婵炲瓨绮庨崑鎾诲箞閵娿儙鐔虹矙閸喖顫撳┑鐘殿暯閳ь剝娅曢崑銉╂煛瀹€鈧崰搴ㄦ偩閿熺姴绠ユい鏃€瀚庨妸鈺傗拺閺夌偞澹嗛ˇ锕傛煕閻斿憡灏︾€规洘妞介崺鈧い鎺嶉檷娴滄粓鏌熼悜妯虹仴闁哄棙鏌ㄩ湁?

const selectAccount = (rule: ModelRoutingRule, account: SimpleAccount, isEdit: boolean = false) => {
  if (!rule) return

  // 婵犵數濮烽弫鍛婃叏閻戝鈧倿鎸婃竟鈺嬬秮瀹曘劑寮堕幋鐙呯幢闂備線鈧偛鑻晶鎾煛鐏炲墽銆掗柍褜鍓ㄧ紞鍡涘磻閸涱厾鏆︾€光偓閸曨剛鍘搁悗鍏夊亾閻庯綆鍓涢敍鐔哥箾鐎电顎撳┑鈥虫喘楠炲繘鎮╃拠鑼唽闂佸湱鍎ら崺鍫濐焽閵夈儮鏀介柣妯活問閺嗩垶鏌嶈閸撴瑩宕捄銊ф／鐟滄棃寮婚悢纰辨晩闁绘挸绨堕崑鎾诲箹娴ｇ懓浠奸梺缁樺灱濡嫬鏁梻浣稿暱閹碱偊宕愰悷鎵虫瀺闁搞儺鍓氶埛鎴︽⒑椤愩倕浠滈柤娲诲灡閺呭墎鈧數纭堕崑鎾舵喆閸曨剛顦ㄩ梺鑽ゅ暱閺呯姴顕ｉ锕€绠荤紓浣姑禍褰掓倵鐟欏嫭绀€婵炲眰鍊栭幈銊ョ暋閹佃櫕鏂€闂佺粯锕╅崰鏍倶鏉堛劎绠惧璺侯儑濞插瓨銇勯姀锛勫⒌鐎规洏鍔戦獮鏍敇閻斿憡鐝?

  if (!rule.accounts.some(a => a.id === account.id)) {
    rule.accounts.push(account)
  }

  // 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柟闂寸绾惧鏌ｉ幇顒佹儓缂佺姳鍗抽弻鐔兼⒒鐎靛壊妲紓浣哄Х婵炩偓闁哄瞼鍠栭幃褔宕奸悢鍝勫殥闂備胶绮幐濠氭晝閵忋倕钃熼柍銉﹀墯閸氬骞栫划鍏夊亾瀹曞浂鍟囩紓鍌氬€烽懗鑸垫叏妤ｅ喛缍栧璺烘湰瀹曞弶绻濋棃娑欏窛缂佲檧鍋撻梻浣侯焾閺堫剛绮欓幋婵冩瀺闁挎繂顦伴埛?

  const key = getRuleSearchKey(rule, isEdit)
  accountSearchKeyword.value[key] = ''
  showAccountDropdown.value[key] = false
}

// 缂傚倸鍊搁崐鎼佸磹閹间礁纾瑰瀣捣閻棗銆掑锝呬壕闁芥ɑ绻冮妵鍕冀椤愵澀姹楅梺閫炲苯澧剧紒鐘冲灴閹箖鏁撻悩鑼吋闂佹儳娴氶崑鍡涘焵椤掍焦銇濋柡宀€鍠栭幊鏍煛閸曞﹤顦版穱濠囧箵閹烘挸娈楅悗瑙勬处閸ㄨ泛鐣烽妸褉鍋撳☉娆樼劷闁告ɑ鎮傞弻锝堢疀閺囩偘鍝楀銈嗘肠閸パ咁槷濠德板€曢幊蹇涙偂閺囥垺鍊甸柨婵嗛娴滄繈鎮樿箛鏂款棆闁逞屽墮閻忔艾顭垮Ο灏栧亾濮樼厧寮鐐叉瀵噣宕煎┑鍫濆绩闂備胶纭堕崜婵嬧€﹂崟顖涘亜闁稿繐鐨烽幏?
const removeSelectedAccount = (rule: ModelRoutingRule, accountId: number, _isEdit: boolean = false) => {
  if (!rule) return

  rule.accounts = rule.accounts.filter(a => a.id !== accountId)
}

// 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸弫宥夊礋椤掍焦顔囬梻浣告贡閸庛倝宕靛顑炴椽顢旈崨顔界彇闂備線鈧偛鑻晶鎾煙椤曗偓缁犳牠骞冨鍫熷癄濠㈣泛瀛╅幉浼存⒒娓氣偓濞佳囁囬銏犵？闁硅泛澹欓崶顒夋晝闁冲灈鏅滈弬鈧梻浣虹帛閸旀牕顭囧▎鎾村€堕柨鏂款潟娴滄粍銇勯幘璺轰沪鐎瑰憡绻冩穱濠囧矗婢跺﹤顫掗梺杞扮閸熸潙鐣烽幒鎴旀婵☆垳鎳撻幗瀣攽閻樺灚鏆╅柛瀣仱瀹曞綊宕奸弴鐔蜂罕闂佸搫娲ㄩ崰宥囨崲閸℃稒鐓犳繛鏉戭儐閺夊綊鏌涚€ｎ偅灏甸柟鍙夋尦瀹曠喖顢楅崒銈喰ら梻鍌欑閹芥粍鎱ㄩ悽鍛婂亱闁绘宕靛畵渚€鏌涢埄鍐槈缂佲偓閸愨斂浜滈柡鍐ㄦ搐娴滆銇勯敂鑺ョ凡妞ゎ亜鍟存俊鍫曞幢濡ゅ啩鎮ｉ梻浣告憸婵绮旂捄琛℃瀻闁靛繒濯鈺傘亜閹烘垵鈧懓鈻撴ィ鍐┾拺闂傚牃鏅涢惁婊堟煕濡粯鍊愮€规洘鍨块獮妯肩磼濡粯鐝抽梺纭呭亹鐞涖儵鍩€椤掍焦鐏遍柟濂夊亰濮婄粯鎷呮笟顖滃姼缂備胶绮崝鏇犲弲濡炪倖鎸堕崹褰掓嫅閻斿摜绠鹃柟瀛樼懃閻忊晝绱掗悩鍐测枙闁哄瞼鍠栧畷褰掝敊閸忓吋顔夐梻浣圭湽閸婃挾娆㈠顒夋綎婵炲樊浜濋悞濠氭煟閹邦垰钄奸悗姘緲椤儻顧侀柛鐘愁殜婵＄敻宕熼娑欐珕闂佽姤锚椤﹁棄螞閸℃稒鈷?

const toggleCreateScope = (scope: string) => {
  const idx = createForm.supported_model_scopes.indexOf(scope)
  if (idx === -1) {
    createForm.supported_model_scopes.push(scope)
  } else {
    createForm.supported_model_scopes.splice(idx, 1)
  }
}

// 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸弫宥夊礋椤掍焦顔囬梻浣告贡閸庛倝宕靛顑炴椽顢旈崨顔界彇闂備線鈧偛鑻晶鎾煙椤曗偓缁犳牠骞冨鍫熷殟闁靛鍎查缁樹繆閻愵亜鈧牜鏁幒妤€纾瑰┑鐘崇閺咁剚绻濋棃娑卞剱闁抽攱鍨块弻娑樷攽閸℃浼€闂佸疇顕чˇ鐢稿蓟濞戙垹绠婚悗鍨偠閳ь剚甯￠弻鈩冪瑹閸パ勭彎閻庤娲栧畷顒勫煡婢跺ň鏋庨煫鍥ч閹藉鈹戦悩鍨毄闁稿鍋ゅ畷褰掑醇閺囩喎浜遍梺鍝勬川閸犲秶鎹㈤崱娑欑厾婵炴潙顑嗛弶褰掓煕鐎ｎ偅灏甸柟鍙夋尦瀹曠喖顢楅崒銈喰ら梻鍌欑閹芥粍鎱ㄩ悽鍛婂亱闁绘宕靛畵渚€鏌涢埄鍐槈缂佲偓閸愨斂浜滈柡鍐ㄦ搐娴滆銇勯敂鑺ョ凡妞ゎ亜鍟存俊鍫曞幢濡ゅ啩鎮ｉ梻浣告憸婵绮旂捄琛℃瀻闁靛繒濯鈺傘亜閹烘垵鈧懓鈻撴ィ鍐┾拺闂傚牃鏅涢惁婊堟煕濡粯鍊愮€规洘鍨块獮妯肩磼濡粯鐝抽梺纭呭亹鐞涖儵鍩€椤掍焦鐏遍柟濂夊亰濮婄粯鎷呮笟顖滃姼缂備胶绮崝鏇犲弲濡炪倖鎸堕崹褰掓嫅閻斿摜绠鹃柟瀛樼懃閻忊晝绱掗悩鍐测枙闁哄瞼鍠栧畷褰掝敊閸忓吋顔夐梻浣圭湽閸婃挾娆㈠顒夋綎婵炲樊浜濋悞濠氭煟閹邦垰钄奸悗姘緲椤儻顧侀柛鐘愁殜婵＄敻宕熼娑欐珕闂佽姤锚椤﹁棄螞閸℃稒鈷?

const toggleEditScope = (scope: string) => {
  const idx = editForm.supported_model_scopes.indexOf(scope)
  if (idx === -1) {
    editForm.supported_model_scopes.push(scope)
  } else {
    editForm.supported_model_scopes.splice(idx, 1)
  }
}

// 濠电姷鏁告慨鐑藉极閸涘﹥鍙忓ù鍏兼綑閸ㄥ倿鏌ｉ幘宕囧哺闁哄鐗楃换娑㈠箣閻愯尙鍔伴梺绋款儐閹告悂锝炲┑瀣亗閹兼番鍨昏ぐ搴繆閵堝洤啸闁稿鐩畷顖烆敍濠婂嫬搴婂┑鐘绘涧椤戝懐绮堢€ｎ偁浜滈柟鍝勬娴滄儳顪冮妶鍛闁诡喖鍊搁～蹇撁洪鍕唶闁瑰吋鐣崹濠氭晬閺囥垺鈷戠紒瀣儥閸庢劙鏌熼幖浣虹暫妤犵偛顦甸獮鏍ㄦ媴閸濄儵鐛撻梻浣告贡椤牊顨ラ崨濠勵浄闁哄鍤﹂弮鍫熷亹闂傚牊绋愬▽顏堟⒑缂佹﹩娈曟繛鑼枛閸ㄩ箖鏁冮崒姣尖晠鏌嶆潪鎵槮缂佹劖绋戦—鍐Χ閸℃ê鏆楅梺绋款儐閿曘垽鐛径瀣ㄥ亝闁告劏鏂侀幏娲⒑閸涘﹦绠撻悗姘槻鍗遍柟闂寸劍閻撶喐淇婇妶鍌氫壕闂佹寧娲︽禍顏堟偘椤曗偓楠炲洭寮剁捄顭戝晣濠电偠鎻徊浠嬪箺濠婂牊鍋傞柟杈鹃檮閳锋垹绱掗娑欑濠⒀冨级閵囧嫰濡搁妷褏顔囬梻鍥ь樀閺屸€愁吋鎼粹€崇闂佺粯鎸诲ú鐔煎蓟瀹ュ鐓涘ù锝呮啞濞堝爼姊虹紒妯绘儎闁搞劌纾Σ鎰板箳濡ゅ﹥鏅╅梺缁樺姦閸撴艾袙閸儲鈷?
const onAccountSearchFocus = (rule: ModelRoutingRule, isEdit: boolean = false) => {
  const key = getRuleSearchKey(rule, isEdit)
  showAccountDropdown.value[key] = true
  // 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴濐潟閳ь剙鍊圭粋鎺斺偓锝庝簽閸旓箑顪冮妶鍡楀潑闁稿鎹囬弻娑㈡偄闁垮浠撮梺绯曟杹閸嬫挸顪冮妶鍡楀潑闁稿鎸剧槐鎾愁吋閸滃啳鍚Δ鐘靛仜閸燁偉鐏掗柣鐘叉穿鐏忔瑧绮ｉ悙瀵哥瘈闁汇垽娼ф禒褔鏌涚€ｎ偅宕岄柟顕嗙節婵偓闁靛牆妫岄幏娲⒑閸濆嫬鈧爼宕曢幓鎺嗘瀺闁告稑鐡ㄩ悡鏇㈡煟閺囨氨顦﹂柣蹇ョ悼缁辨帡顢欑喊杈╁悑濡ょ姷鍋炵敮鎺曠亙婵炶揪绲介幖顐︾嵁瀹ュ鈷掑ù锝勮閻掑墽绱掔紒姗嗘疁鐎规洘鍨块獮鍥偋閸繂澹掗柣搴″帨閸嬫捇鏌涢弴銊ュ闁告﹢浜堕弻锝堢疀閺囩偘鍝楀銈嗘肠閸曨亞绠氶柣鐘充航閸斿海澹曟總绋跨骇闁割偅绋戞俊鍏肩箾閹碱厼鏋熸い銊ｅ劦閹瑩骞栭鐘插Ш闂備線鈧偛鑻晶浼存煛娴ｅ壊鐓肩€殿喗鐓″濠氬Ψ閵壯屾П濠电偛顕慨鎾敄閸℃稑鐓曢柟杈鹃檮閻撴瑧绱掔€ｎ偄顕滈柟鐧哥秮閺屽秹鎸婃径妯烩枅闂佸搫鐭夌紞渚€銆佸Ο琛℃斀闁搞儮鏂傞崑妯讳繆閻愵亜鈧垿宕愰弴銏犵９闁割煈鍣鏍ㄧ箾瀹割喕绨荤€瑰憡绻傞埞鎴︽偐閹绘帩浠鹃梺闈╃秵閸欏啫顫忓ú顏勭闁绘劖褰冮‖鍫㈢磼閸撗嗘闁告﹢绠栧畷姘跺箳濡も偓閻撴盯鏌涘☉鍗炲箻濡ょ姴娲娲偡閹殿喗鎲煎┑顔硷工缂嶅﹪骞冨Δ浣虹瘈婵﹩鍘搁幏娲⒑閸濆嫬鈧爼宕曢幓鎺嗘瀺闁告稑鐡ㄩ悡鏇㈡煟閺囨氨顦﹂柣蹇ュ閳ь剝顫夊ú姗€宕归悽绋跨厴闁硅揪绠戦柋鍥煏韫囷絾绶涚紒?
  if (!accountSearchResults.value[key]?.length) {
    searchAccounts(key)
  }
}

// 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柟闂寸绾惧潡鏌熺€电啸缂佲偓婵犲洦鐓冪憸婊堝礈濮樿泛桅闁告洦鍨伴崡鎶芥煕閳╁喚娈旀い蹇ｄ邯閺屾稑鈻庤箛鏇狀啋闂佸搫鐭夌紞渚€鐛崶顒夋晢濞达絿鎳撻崜鐢电磽閸屾瑨顔夐柛瀣崌閺屾盯鍩勯崘鍓у姺缂備胶濞€缁犳牠寮婚悢琛″亾閻㈡鐒鹃柛鎾讳憾閺岋綁骞掗弮鈧▍鏇犵磼鏉堛劌娴鐐差槺閳ь剨缍嗛崜娑㈠汲閵堝鈷戦悹鍥ｂ偓铏仌濠电偛顦伴惄顖炲春閵夛箑绶為柟閭﹀墮閸炪劑鎮峰鍐ч柍銉︾墵瀹曞ジ濡烽敃鈧埀顒傛暬閹嘲鈻庤箛鎿冧痪缂備讲鍋撻柛顐犲劜閻撴洟鏌ｅΟ铏癸紞濠⒀冪仛椤ㄣ儵鎮欓崣澶樻＆閻庤娲橀敃銏ゃ€佸▎鎾村殐闁冲搫锕ユ晥婵犵绱曢崑鎴﹀磹閺嶎厼钃熼柕濠忛檮濞呯姴霉閻樺樊鍎忛柤绋跨秺閺岀喓绱掗姀鐘崇亶闂佹娊鏀遍崹鍫曞箞閵娾晜鍋￠柣妤€鐗婇幃娆撴⒑鐠囪尙绠哄ù婊庡墰濡叉劙骞掑Δ濠冩櫔闂侀€炲苯澧扮紒顔碱煼瀵粙顢橀悢鍛婄彸闂備胶绮崝鏍ь焽濞嗘垶顐介柣鎰劋閻撴瑩姊洪銊х暠濠⒀呭閵?
const addCreateRoutingRule = () => {
  createModelRoutingRules.value.push({ pattern: '', accounts: [] })
}

// 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸弫宥夊礋椤掍焦顔囬梻浣虹帛閸旀洟顢氶鐘典笉濡わ絽鍟悡鍐喐濠婂牆绀堟慨妯块哺瀹曞弶绻涢幋娆忕仼鐎瑰憡绻冮妵鍕箻閸楃偟浠奸悗娈垮枙閸楁娊骞冨Δ鍐╁枂闁告洦鍓涢ˇ銊╂⒑閻撳海绉虹紒鐘崇墵瀹曟椽濡烽敃鈧欢鐐测攽閻愵亜鐏柛鐘崇墵楠炲﹪寮介鐐靛幐婵炴挻鑹惧ú锕佲吀闂傚倸鍊搁崐宄懊归崶褏鏆﹂柣銏㈩焾绾惧鏌ｉ幇顔芥毄闁活厽鐟╅悡顐﹀炊閵娧€妲堢紓浣插亾濠㈣埖鍔栭崐鍫曟煟閹邦剛浠涙繛鍛礋閺岋繝宕遍鐘垫殼闂佸搫鐭夌紞浣割嚕椤掑嫬鍨傛い鏃囨閳ь剦鍨跺铏圭矙閸ф鈧绱掓径濠勭Ш鐎殿喛顕ч埥澶愬閻樻彃绁梻渚€娼ч…鍫ュ磿閺屻儲鍊堕柟鍓х帛閳锋垹绱撴担鑲℃垿鎮￠妷鈺傜厵闁惧浚鍋撻懓璺ㄢ偓娈垮枛椤兘寮幇顓炵窞濠电姴瀛╃紞鍌炴⒒娴ｇ懓顕滅紒璇插€哥叅闁靛ě鍛槱闂佸憡顨堥崕鎰€掓繝姘厪闁割偅绻堥妤€霉濠婂嫮鐭嬮柕鍥у閺佸啴宕掑☉娆撴暘闂?
const removeCreateRoutingRule = (rule: ModelRoutingRule) => {
  const index = createModelRoutingRules.value.indexOf(rule)
  if (index === -1) return

  const key = getCreateRuleSearchKey(rule)
  accountSearchRunner.clearKey(key)
  clearAccountSearchStateByKey(key)
  createModelRoutingRules.value.splice(index, 1)
}

// 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柟闂寸绾惧潡鏌熺€电啸缂佲偓婵犲洦鐓冪憸婊堝礈濮樿泛桅闁告洦鍨伴崡鎶芥煕閳╁喚娈旀い蹇ｄ邯閺屾稑鈻庤箛鏇狀啋闂佸搫鐭夌紞渚€鐛崶顒夋晩闁兼亽鍎查惁搴ｇ磽娴ｇ懓鍔ょ憸鎵仧閸掓帒鈻庨幘瀹犳憰闂佺偨鍎查崜姘跺触鐎ｎ喖绠圭紒顔煎帨閸嬫捇骞囨担褰掔崕闂傚倷鐒﹂惇褰掑春閸曨垰鍨傞柡澶庮嚦濞戞ǚ鏀介悗锝庝簽閻ｆ椽鏌ｉ悩鑽ょ窗闁靛棌鍋撳┑鈩冨絻閻楁捇寮婚垾宕囨殼妞ゆ梻鍘ч弸銈夋煟鎼淬垹鈻曢柡宀嬬稻閹棃濡舵惔銏㈢Х婵犵數鍋涘鍓佸垝閹惧磭鏆﹀ù鍏兼綑閸愨偓濡炪倖鎸鹃崑鐔兼偩妤ｅ啯鈷戦柛娑橈工婵箓鏌涢悩宕囧⒈缂侇喖鐗忛埀顒婄秵閸嬩焦绂嶅鍫熺厸闁稿本姘ㄦ禒銏ゆ煃闁垮顥堥柡灞剧洴閹瑧鈧稒锚閸炲鈹戦纭峰伐闁圭⒈鍋呴弲銉╂⒑閹肩偛鍔€闁告侗鍓涚粈鍐磽閸屾艾鈧娆㈤敓鐘查棷闁挎繂娲ㄦ稉宥夋煛瀹ュ骸骞楅柡鍛叀閺岋綁骞囬鐔虹▏缂備礁澧庨崑銈夊蓟閻斿吋鐒介柨鏇楀亾濠⒀呭閵?
const addEditRoutingRule = () => {
  editModelRoutingRules.value.push({ pattern: '', accounts: [] })
}

// 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸弫宥夊礋椤掍焦顔囬梻浣虹帛閸旀洟顢氶鐘典笉濡わ絽鍟悡鍐喐濠婂牆绀堟慨妯块哺瀹曞弶绻涢幋娆忕仼鐎瑰憡绻冮妵鍕箻鐠虹儤鐎鹃悗瑙勬礀绾绢厾妲愰幘瀛樺濞寸姴顑呴幗鐢告⒑閸︻厽鍤€婵炲眰鍊濋、姘舵晲閸℃瑯娴勯柣搴到閻忔岸寮查姀銈嗏拺缂佸瀵у﹢鐗堟叏濡濮傛い銏℃閵囨劙骞掗幘顖涘闂備胶鍘ч～鏇㈠磹閺囩喓顩烽柨鏂垮⒔绾惧ジ鎮楅敐搴濈敖缂佺姾宕甸埀顒冾潐濞叉鎹㈤崟顒傤浄闁挎洖鍊归崑鈺呮倶閻愰潧浜炬い鏃€娲熷缁樻媴閾忕懓绗￠梺鍝勮閸旀垵鐣烽妷褉鍋撻敐搴℃灈缁炬崘娉曢埀顒€绠嶉崕閬嵥囨导瀛樺亗闁哄洢鍨洪悡娑㈡煕閵夛絽鍔氬┑鈥炽偢閺岋紕鈧絻鍔岄埀顒佺箞瀵鏁愭径濠冾棟闂佸湱顭堟绋库枔濡ゅ懏鈷戦柛婵嗗閻掕法绱撳鍕妽闁逛究鍔戦崺鈧い鎺戝閻撱儲绻濋棃娑欘棡鐎瑰憡绻勭槐鎾诲醇濠靛棜鍩炲銈庝簻閸熷瓨淇婇懜鍨劅闁炽儴灏欓惄搴繆閻愵亜鈧呯不閹炬剚鐒界憸鏃堝箖?
const removeEditRoutingRule = (rule: ModelRoutingRule) => {
  const index = editModelRoutingRules.value.indexOf(rule)
  if (index === -1) return

  const key = getEditRuleSearchKey(rule)
  accountSearchRunner.clearKey(key)
  clearAccountSearchStateByKey(key)
  editModelRoutingRules.value.splice(index, 1)
}

// 闂?UI 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偞鐗犻、鏇㈠Χ閸モ晝鍘犻梻浣稿閸嬪懎煤閺嶎厼纾奸柕濞у嫬鏋戦棅顐㈡处閹峰綊鏁愭径濠勭杸闂佺粯顨呴悧濠傗枍閵忋倖鈷戠紓浣光棨椤忓牆鐤柛褎顨呯粈鍫熴亜閹惧崬鐏柍閿嬪灴閺屾稑鈽夊鍫熸暰婵犮垼娉涚€氫即寮诲☉銏″亹闁告劖褰冮幗鐢告⒑鐠団€虫灁闁搞劏妫勯悾鐑藉Ω瑜夐崑鎾斥槈濞嗘鍔烽梺鍛娒肩划娆忣潖濞差亝顥堟繛鎴欏灮瀹曨亞绱撴担钘夎敿婵炲娲熼幊鐐烘焼瀹ュ懐顔囬柟鐓庣摠缁嬫劗鑺辩拠宸富闁靛牆妫楁慨鍌炴煕閳轰礁顏╅摶锝夋煃瑜滈崜鐔煎箖濡ゅ啯鍠嗛柛鏇ㄥ墰椤︺劑姊洪悡搴ｇШ缂佺姵鐗犲畷娲閳╁啫鍔呴梺闈涱焾閸庢娊顢欐繝鍥ㄢ拺閻熸瑥瀚粈鍐磽閸粌宓嗙€规洘鍨垮浠嬵敇閻斿弶瀚藉┑鐐舵彧缁茶偐鎷冮敃鍌氱哗闁绘鐗勬禍婊堟煛閸モ晛鏋斿褜浜濋妵鍕敇閻愭潙浠撮梺绯曟杹閸嬫挸顪冮妶鍡楃瑨閻庢凹鍙冨畷?API 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偞鐗犻、鏇㈠Χ閸モ晝鍘犻梻浣稿閸嬪懎煤閺嶎厼纾奸柕濞у嫬鏋戦棅顐㈡处閹?

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

// 闂?API 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偞鐗犻、鏇㈠Χ閸モ晝鍘犻梻浣稿閸嬪懎煤閺嶎厼纾奸柕濞у嫬鏋戦棅顐㈡处閹峰綊鏁愭径濠勭杸闂佺粯顨呴悧濠傗枍閵忋倖鈷戠紓浣光棨椤忓牆鐤柛褎顨呯粈鍫熴亜閹惧崬鐏柍閿嬪灴閺屾稑鈽夊鍫熸暰婵犮垼娉涚€氫即寮诲☉銏″亹闁告劖褰冮幗鐢告⒑鐠団€虫灁闁搞劏妫勯悾鐑藉Ω瑜夐崑鎾斥槈濞嗘鍔烽梺鍛娒肩划娆忣潖濞差亝顥堟繛鎴欏灮瀹曨亞绱撴担钘夎敿婵炲娲熼幊鐐烘焼瀹ュ懐顔囬柟鐓庣摠缁嬫劗鑺辩拠宸富闁靛牆妫楁慨鍌炴煕閳轰礁顏╅摶锝夋煃瑜滈崜鐔煎箖濡ゅ啯鍠嗛柛鏇ㄥ墰椤︺劑姊洪悡搴ｇШ缂佺姵鐗犲畷娲閳╁啫鍔呴梺闈涱焾閸庢娊顢欐繝鍥ㄢ拺閻熸瑥瀚粈鍐磽閸粌宓嗙€规洘鍨垮浠嬵敇閻斿弶瀚藉┑鐐舵彧缁茶偐鎷冮敃鍌氱哗闁绘鐗勬禍婊堟煛閸モ晛鏋斿褜浜濋妵鍕敇閻愭潙浠撮梺绯曟杹閸嬫挸顪冮妶鍡楃瑨閻庢凹鍙冨畷?UI 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偞鐗犻、鏇㈠Χ閸モ晝鍘犻梻浣稿閸嬪懎煤閺嶎厼纾奸柕濞у嫬鏋戦棅顐㈡处閹峰綊鏁愭径濠勭杸闂佺粯顨呴悧濠傗枍閵忋倖鈷戠紓浣癸供閻掔偓绻涢崨顔界闁宠绉瑰浠嬵敇閻斿弶瀚奸梻浣告啞閹告槒銇愰崘鈺冾洸闁哄稁鍘介悡鏇㈠箹缁厜鍋撳畷鍥崜缂傚倷绀侀崐鍦暜閹烘缍栨繝濠傜墕閻掑灚銇勯幒鎴濐仼缂佺姵鐗犻弻锝夊箣閿濆棭妫勯梺鍛婎殙濞呮洟骞堥妸銉建闁逞屽墰濞戠數绮欑拠鎾呯秮椤㈡瑩鎮惧畝鈧惁鍫濃攽閻愯尙澧曢柣蹇旂箞瀵悂濡舵径瀣幈闁诲函缍嗛崜娆愮濠婂嫨浜滄い鎾跺仦閸犳ɑ銇勯姀鈩冾棃鐎规洜鍘ч埞鎴﹀醇椤掑倷瑕嗛梻鍌氬€烽懗鍓佸垝椤栫偛钃熼柕濞炬杺閳ь剙鍟幆鏃堝焾閵夘垳绉柡灞芥椤撳ジ宕辫箛鏂款伖闂傚倷绀侀幉锛勭矙閹达附鏅濋柨鏇炲€哥粻鐘绘煕閹伴潧鏋熼柣鎾存礋閺屾洝绠涢妷褏锛熼梺鍛婄憿閸嬫挻淇婇妶鍥ラ柛瀣洴瀹曨垶寮堕幋顓炴闂佸湱铏庨崰妤呭磻閹扮増鐓熼柕蹇嬪灪閺嗏晠鏌涘Ο鐓庘枅婵﹤鎼晥闁搞儜鍛Ф闂備焦鎮堕崝宀勫Χ缁嬭法鏆﹂柡澶嬵儥濞尖晠寮堕崼姘珖闁?

const convertApiFormatToRoutingRules = async (apiFormat: Record<string, number[]> | null): Promise<ModelRoutingRule[]> => {
  if (!apiFormat) return []

  const rules: ModelRoutingRule[] = []
  for (const [pattern, accountIds] of Object.entries(apiFormat)) {
    // 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸崹楣冨箛娴ｅ湱绋佺紓鍌氬€烽悞锕佹懌闂佸憡鐟ョ换姗€寮婚悢纰辨晬闁挎繂娲ｅЧ妤呮偡濠婂懎顣奸悽顖涘浮閹瑦绻濋崶銊у帗闂佸憡绻傜€氼剟鍩€椤掍焦鍊愭い銏℃楠炴牗鎷呴崗澶嬪闂備礁婀遍…鍫⑩偓娑掓櫇婢规洟鎮介崨濠勫幈闁诲函缍嗛崑鍛焊閻㈠憡鐓欓柛娆忣槹閸婃劙鏌熼銊ユ搐闁卞洦绻濋崹顐㈠婵炲牊绻勭槐鎾诲磼濞嗘埈妲梺娲诲幖閸婂湱绮嬪澶婇唶闁哄洨鍋炲Σ顒勬⒑閸涘﹥澶勯柛銊﹀缁牓宕橀鍛瀾闂婎偄娲﹀ú婊堟儗閸℃稒鐓?
    const accounts: SimpleAccount[] = []
    for (const id of accountIds) {
      try {
        const account = await adminAPI.accounts.getById(id)
        accounts.push({ id: account.id, name: account.name })
      } catch {
        // 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴濐潟閳ь剙鍊圭粋鎺斺偓锝庝簽閸旓箑顪冮妶鍡楀潑闁稿鎹囬弻娑㈡偄闁垮浠撮梺绯曟杹閸嬫挸顪冮妶鍡楀潑闁稿鎸剧槐鎾愁吋閸滃啳鍚Δ鐘靛仜閸燁偉鐏掗柣鐘叉穿鐏忔瑧绮ｉ悙鐑樼厽闊洦娲栨禒锕傛煕鎼淬垹鈻曟い銏℃楠炴牗鎷呴崗澶嬪闂備礁婀遍…鍫⑩偓娑掓櫇婢规洟鎮介崨濠勫幈闁诲函缍嗛崑鍛焊閻㈠憡鐓欓柛娆忣槹閸婃劙鏌熼銊ユ搐闁卞洦绻濋崹顐㈠婵炲牊绻勭槐鎾诲磼濞嗘埈妲銈嗗灥閹虫妫熷銈嗙墱閸庢劙寮崶褉鏀介柛灞剧矤閻掑墽鈧懓鎲＄换鍐Φ閸曨垰鍐€闁靛ě鈧慨鍥⒑閸濄儱校闁绘濮撮～蹇曠磼濡偐鎳濋梺閫炲苯澧い顓炴喘閹筹繝濡堕崒姘闂傚倸鐗婄粙鎾绘倿閹间焦鐓欐い鏃傛嚀婢ц尙鈧灚婢樼€氼喗绂掗敂鍓х＜闁靛骏绱曠粙鎰攽閻樺灚鏆╁┑鐐╁亾濠电偘鍖犻崗鐘虫そ婵℃悂鍩炴惔鎾充壕闁告侗鍣Σ褰掑箹鐎涙◤顏呯閻愵剦娈介柣鎰皺娴犮垽鏌涢弮鈧喊宥囨崲濞戞矮娌柛灞惧焹閸嬫捇寮介鐐电杽闂侀潧顭悡鍫澪熼崟顖涒拺闁告繂瀚悞璺ㄧ磼缂佹绠撻柣锝囧厴閹垻鍠婃潏銊︽珖闂備礁鍟块悘鍫ュ疾濠婂應鍋撻崹顐ゅ⒌婵?ID
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
  // 闂傚倸鍊搁崐鎼佸磹閻戣姤鍊块柨鏇炲€哥粻鏍煕椤愶絾绀€缁炬儳娼″鍫曞醇濞戞ê顬夊┑鐐叉噽婵炩偓闁哄被鍊濋獮渚€骞掗幋婵嗩潥婵犵數鍋涢幊鎰箾閳ь剟鏌＄仦绯曞亾閹颁礁鎮戦柟鑲╄ˉ閳ь剙纾鎴︽⒒娴ｄ警鐒炬い鎴濆暣瀹曟繈骞嬮敃鈧拑鐔兼煥濠靛棭妲哥紒顐㈢Ч閺屾稓浠︾拠娴嬪亾濡ゅ懎绀夐柟闂寸劍閳锋垿鎮归崶顏勭毢缂佺姷澧楃换娑橆啅椤旂厧绫嶅Δ鐘靛仜閸熶即骞夐幘顔肩妞ゆ劦鍋勫▓鍏间繆閻愵亜鈧牠寮婚妸鈺佺妞ゆ劧绠戦悞鍨亜閹哄秶鍔嶇紒鈧€ｎ喗鐓欐い鏃傜摂濞堟粍銇勯姀鈽呰€垮┑顔瑰亾闂佹娊鏁崑鎾绘煙闁垮銇濇慨濠冩そ瀹曨偊宕熼崹顐嶎亜鈹戦悙宸Ч婵炲弶绻堝畷鎰板箻椤旇В鎷绘繛杈剧到濠€鍗烇耿娴犲鐓曞┑鐘插暞缁€鈧柧鑽ゅ仦缁绘繈妫冨☉鍗炲壈闂佸搫顑勯懗鍫曞焵椤掆偓缁犲秹宕曢柆宓ュ洭顢涢悙瀵稿帒闂佹悶鍎崝搴ｅ姬閳ь剚绻濋悽闈浶㈤柛濠傜秺瀹曡櫕绂掔€ｎ偆鍘?antigravity 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬楠炲啫螖閳ь剟鍩ユ径濞炬瀺闁惧浚鍋勯悘锕傛煕閹烘埊韬鐐达耿椤㈡瑩鎸婃径澶婂闂傚倷娴囬鏍储閻ｅ本鏆滈柟鐑樻尫缁诲棗霉閻樺樊鍎愰柣鎾寸洴閺屾盯鍩勯崘鐐暭缂備胶濮撮…鐑藉蓟?
  image_price_1k: null as number | null,
  image_price_2k: null as number | null,
  image_price_4k: null as number | null,
  // Claude Code 闂傚倸鍊搁崐宄懊归崶顒夋晪鐟滃酣銆冮妷鈺佷紶闁靛／鍌滅憹闁诲骸绠嶉崕閬嵥囬鐐插瀭闁稿本绋撶粻鍓р偓鐟板閸犳洜鑺辨繝姘厸闁告洍鏅涢崝婊呯磼缂佹娲存鐐差儔閹瑧鈧潧鎲￠濠氭⒒娴ｅ憡鍟炴い銊ユ閸犲﹤顓兼径濞箓鏌涢弴銊ョ仩闁告劏鍋撴俊鐐€栭崝锕€顭块埀顒佺箾瀹€濠佺盎妞ゎ亜鍟存俊鍫曞川椤撗冨Ψ闂備浇宕甸崯鍧楀疾閻樺樊鍤曢悹鍥ㄧゴ濡插牓鏌曡箛鏇炐ユい鏂匡躬濮婃椽宕崟顒€鍋嶉梺鍛婃煥閻倸顕ｉ弻銉︽櫜闁搞儮鏅濋敍婵囩箾鏉堝墽瀵肩紒顔界懇瀹曨偄煤椤忓懐鍘介梺缁樻礀閸婃悂銆呴鍌滅＜?anthropic 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬楠炲啫螖閳ь剟鍩ユ径濞炬瀺闁惧浚鍋勯悘锕傛煕閹烘埊韬鐐达耿椤㈡瑩鎸婃径澶婂闂傚倷娴囬鏍储閻ｅ本鏆滈柟鐑樻尫缁诲棗霉閻樺樊鍎愰柣鎾寸洴閺屾盯鍩勯崘鐐暭缂備胶濮撮…鐑藉蓟?
  claude_code_only: false,
  fallback_group_id: null as number | null,
  fallback_group_id_on_invalid_request: null as number | null,
  // OpenAI Messages 闂傚倸鍊搁崐宄懊归崶褏鏆﹂柛顭戝亝閸欏繘鏌熺紒銏犳珮闁轰礁瀚伴弻娑樷槈濞嗘劗绋囬梺姹囧€ら崰妤呭Φ閸曨垰绠涢柍杞拌閸嬫捇骞囬弶璺紱闂佽宕橀崺鏍窗閸℃稒鐓曢柡鍥ュ妼娴滄劙鏌＄€ｎ偆鎳囨慨濠冩そ瀹曨偊宕熼纰变純闂備焦瀵уú蹇涘磹濠靛鈧礁螣閼姐倝妾紓浣割儓濞夋洟宕愰悙鐑樷拺闁告劕寮堕幆鍫ユ煕婵犲偆鐓奸柛鈹惧亾濡炪倖甯掗ˇ顖氼嚕椤旇姤鍙忓┑鐘插暞閵囨繃顨ラ悙鏉戝闁诡垱妫冮弫鎰板磼濞戣鲸缍岄梻鍌氬€烽懗鍓佸垝椤栫偑鈧啴宕ㄩ弶鎴犵枃闂佸湱澧楀妯肩矆?openai 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬楠炲啫螖閳ь剟鍩ユ径濞炬瀺闁惧浚鍋勯悘锕傛煕閹烘埊韬鐐达耿椤㈡瑩鎸婃径澶婂闂傚倷娴囬鏍储閻ｅ本鏆滈柟鐑樻尫缁诲棗霉閻樺樊鍎愰柣鎾寸洴閺屾盯鍩勯崘鐐暭缂備胶濮撮…鐑藉蓟?
  allow_messages_dispatch: false,
  default_mapped_model: '',
  // 婵犵數濮烽弫鍛婃叏閻戝鈧倿鎸婃竟鈺嬬秮瀹曘劑寮堕幋婵堚偓顓烆渻閵堝懐绠伴柣妤€妫濋幃鐐哄垂椤愮姳绨婚梺鐟版惈濡绂嶉崜褏纾奸柛鎾楀棙顎楅梺鍛婄懃閸熸潙鐣峰ú顏勭劦妞ゆ帊闄嶆禍婊堟煙閻戞ê鐏ユい蹇撶摠娣囧﹪顢曢敐鍛紝闂佸搫鐬奸崰鏍箖閻戣棄绾ф繛鍡楀綖缁辨梹淇婇悙顏勨偓鎴﹀垂閾忓湱鐭欓柟杈捐吂閳ь剨绠撴俊鎼佸煛娴ｄ警妲规俊鐐€栫敮濠勬閿熺姴鐤柛娑樼摠閳锋垿鎮归崶銊ョ祷妞ゆ帇鍨洪妵鍕籍閳ь剟鎮ч悩鑼殾濞村吋娼欑粻铏繆閵堝倸浜剧紒鐐劤椤兘寮婚悢鐓庣鐟滃繒鏁☉銏＄厽?
  model_routing_enabled: false,
  // 闂傚倸鍊搁崐鎼佸磹瀹勬噴褰掑炊椤掆偓杩濋梺閫炲苯澧撮柡灞剧〒閳ь剨缍嗛崑鍛焊娴煎瓨鐓忛柛銉戝喚浼冮悗娈垮枙缁瑦淇婂宀婃Х濠碘剝褰冮悧鎾愁潖閻戞ê顕辨繛鍡楃箳濡诧綁姊洪棃鈺冪Ф缂傚秳绀侀锝嗙節濮橆厽娅滄繝銏ｆ硾璋╅柍鍝勬噺閻撳繐顭跨捄铏瑰闁告梹娼欓湁闁绘宕甸悾娲煛鐏炶鈧繂鐣烽悷鎵虫婵炲棙鍔楄ぐ瀣⒒娴ｅ憡鎯堥柡鍫墴閹嫰顢涘☉妤冪畾闂佸綊妫跨粈浣告暜婵＄偑鍊栧濠氬储瑜庣粩鐔衡偓锝庡枟閳锋帡鏌涚仦鍓ф噮閻犳劒鍗抽弻娑㈡偐閹颁焦鐤佸Δ鐘靛仦閸旀瑥鐣峰鈧幊鐘活敆娴ｈ鍟庡┑鐘愁問閸犳鏁冮埡鍛？闁汇垻顭堢猾宥夋煕鐏炵虎娈斿ù婊堢畺閺屻劌鈹戦崱娑扁偓妤€顭胯婢ф濡甸崟顖涙櫆閻熸瑥瀚悵鏇犵磽?antigravity 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷?
  supported_model_scopes: ['claude', 'gemini_text', 'gemini_image'] as string[],
  // MCP XML 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸弫鎾绘偐椤旂懓浜鹃柛鎰靛枛瀹告繃銇勯弽銊р槈閹兼潙锕ら埞鎴炲箠闁稿﹥娲熼獮蹇曗偓锝庡枛閺嬩礁鈹戦悩鍙夊闁绘挻娲樼换娑㈠箣濠靛棜鍩炲Δ鐘靛仦閸旀瑩寮婚敍鍕ㄥ亾閿濆骸浜炲褏鏁婚弻锛勪沪閻愵剛顦ㄧ紓浣虹帛閻╊垰鐣疯ぐ鎺濇晩闁绘挸瀵掑娑㈡⒑鐠囨彃顒㈡い鏃€鐗犲畷浼村冀椤愩埄妫滃銈嗘尪閸ㄥ綊鎮為崹顐犱簻闁瑰搫绉剁拹浼存煕閻旈绠婚柡灞剧洴閹晠宕橀崣澶庣檨闁诲氦顫夊ú姗€宕归崸妤冨祦婵☆垵鍋愮壕鍏间繆椤栨粌甯舵鐐茬墦濮婄粯绻濇惔鈥茬盎濠电偠顕滅粻鎴犲弲闂佹寧娲栭崐鍝ョ矆婢跺绡€濠电姴鍊归崳鐣岀棯閹佸仮鐎殿喖鐖煎畷鐓庘槈濡警鐎烽梻浣烘嚀閸熻法鈧凹鍠氬Σ?antigravity 婵犵數濮撮惀澶愬级鎼存挸浜炬俊銈勭劍閸欏繘鏌ｉ幋锝嗩棄缁惧墽绮换娑㈠箣濞嗗繒浠奸梺姹囧€ら崳锝夊蓟閵堝绠涘ù锝呮憸娴犳粍绻涚€涙鐭婄紓宥咃躬瀵鎮㈤搹鍦紲闂侀潧绻掓慨鐢告倶瀹ュ鈷?
  mcp_xml_inject: true,
  // 濠电姷鏁告慨鐑藉极閹间礁纾绘繛鎴欏焺閺佸銇勯幘璺烘瀾闁告瑥绻橀幃妤€鈽夊▎娆庣返濠电偛鐗呯划娆撳蓟閻斿吋鈷掗悗闈涘濡差噣姊洪幖鐐插闁稿﹤娼￠悰顕€寮介妸锔剧Ф闂佸憡鎸嗛崟顐¤繕缂傚倷鑳堕崑鎾诲磿閹惰棄围闁归棿绀侀拑鐔哥箾閹存瑥鐏╅柛妤佸▕閺屾洘绻涢崹顔煎闂佺厧澹婃禍婊堝煘閹达箑鐒洪柛鎰╁妿缁佸嘲顪冮妶搴″箻闁稿繑锚椤曪絿鎷犲ù瀣潔闂侀潧绻掓慨鐑筋敊婵犲洦鈷戦悷娆忓閸斻倝鏌涢悢绋款嚋闁逛究鍔戝畷銊︾節閸曨厾妲囬梻渚€娼ф蹇曞緤閸撗勫厹濡わ絽鍟崐鍨叏濮楀棗骞楃紒鑸电叀閺?
  copy_accounts_from_group_ids: [] as number[]
})

// 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偞鐗犻、鏇㈠Χ閸モ晝鍘犻梻浣虹帛閸ㄥ爼寮搁懡銈囩闁哄诞宀€鍞甸柣鐘烘鐏忋劑宕濋悢鍏肩厸闁糕剝鍔曢埀顒佹礋婵℃挳骞掗幋顓熷兊闂佹寧绻傞幊宥嗙珶閺囥垺鐓熼幖娣€ゅ鎰箾閸欏顏堟偩濠靛牏鐭欓悹鎭掑妽濞堥箖姊洪崨濠庢畼闁稿鍔欏鍛婃償閵婏妇鍘甸柣搴ｆ暩椤牆鏆╅梻浣告啞钃遍柛銊ユ贡濡叉劙骞樼拠鑼紲濠电偛妫欓崹鑲╃玻濡ゅ懏鈷戠紒瀣儥閸庡繑銇勯敂鐐毈闁靛棔绀侀～婵嬪箛娴ｅ厜鏋岄梻鍌欐祰椤曟牠宕板Δ鍛偓鍐╃節閸パ嗘憰闂侀潧臎閳ь剟宕戦幘缁樻櫜閹肩补鈧尙鏆楁繝娈垮枟閿曨偆绮婚幘鎰佹綎婵炲樊浜滄导鐘绘煕閺囥劌浜愰柛瀣崌瀹曠兘顢樺☉妯瑰缂備焦顨嗙粙鎴﹀箠閹扮増鍎楅柛鈩冪⊕閻撳啰鎲稿鍫濈婵椴稿畷鍙夌箾閹存瑥鐏╃紒鐙呯秮閺岋綁骞囬鐔虹▏缂佺偓婢樼粔鍓佹閹惧瓨濯撮柛鎾冲级鏁堥梻浣烘嚀閹测剝绻涙繝鍥ㄥ仒妞ゆ洍鍋撶€规洘锕㈤、娆撴嚃閳哄﹤鏅紓鍌氬€搁崐鐑芥倿閿曞倹鏅梻浣告啞閹尖晠宕ｉ崘顭戞綎婵炲樊浜滃婵嗏攽閻樻彃鏆欐い锔规櫊濮婅櫣绮欏▎鎯у壈闁诲孩鐭崡鎶界嵁閸愵煈娼ㄩ柍褜鍓熷畷娲晸閻樻彃绐涘銈嗘椤鈧矮绮欏缁樻媴閸涘﹤鏆堝┑鐐村絻缁夌懓顕ｉ幓鎺嗘闁靛繒濮烽崝锕€顪冮妶鍡楃瑐缂佸灈鈧枼鏋旀繝濠傜墛閻撴洟鏌曟繝蹇曠暠闁告柣鍊濋幗鍫曟晲婢跺鍘遍棅顐㈡处濞叉牜鏁懜鐐逛簻閹兼番鍨诲ú瀛樻叏婵犲啯銇濇俊顐㈠暙閳藉顫濋澶嬫瘒濠电姷顣藉Σ鍛村磻閸涱垳鐭欓柟杈剧畱濮规煡鏌嶉崫鍕櫤闁?
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
    // 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偞鐗犻、鏇㈡晜缂佹ɑ娅堥梻浣告啞娓氭宕归鍫濈；闁圭偓鍓氬鈺呮煟閹炬娊顎楁い顐亞缁辨挻鎷呮禒瀣懙闂佸湱鎳撳ú顓㈡偘椤旂⒈娼ㄩ柍褜鍓熼獮濠傤煥閸噥妫冨┑鐐村灦閼归箖鏌婇柆宥嗏拻闁稿本鑹鹃埀顒勵棑缁牊绗熼埀顒勫箖濡　鏀介悗锝呯仛閺呯偤姊洪幖鐐插妧鐎广儱鐗冮崑鎾诲锤濡や胶鍘电紓鍌欓檷閸ㄥ綊寮搁悢鍏肩厽闁规儳顕埥澶愭婢舵劖鐓ユ繝闈涙閸ｆ椽鏌熼姘卞闁靛洤瀚伴弫鎰板醇濠靛懏鐫忛梻渚€鈧偛鑻晶鍙夈亜椤愩埄妯€妤犵偞鍔栭ˇ鐗堟償閵忊晛浠烘繝娈垮枟椤牆鈻斿☉銏犲嚑濞达綀銆€閸嬫捇鐛崹顔煎濠电偠顕滅徊浠嬪煝娴犲鏁傞柛顐ゅ枔閸欏啫鈹戦埥鍡楃仧閻犫偓閿曗偓鍗遍柣鎴灻肩换鍡樸亜閺嶃劎鎳呯紒鎰⒐閹便劍绻濋崘鈹夸虎閻庤娲忛崝宥囨崲濠靛纾兼慨姗嗗幘濡叉姊婚崒娆掑厡妞ゎ厼鐗撻、鏍川閺夋垹鐤囬梺瑙勫礃椤曆囨偪椤斿浜滈柡宥庡亜娴狅箓鏌嶉柨瀣伌闁哄瞼鍠栭幊鏍煛娴ｉ鎹曞┑鐘殿暯閳ь剙鍟块幃鎴︽煏閸パ冾伃妤犵偛顑呴埞鎴﹀醇濠靛啫寰嶉梻鍌欐祰椤曟牠宕规导鏉戝珘妞ゆ帒瀚洿闂佹寧绻傞ˇ顖滅矆閸愵喗鐓冮弶鐐靛椤﹂绱撻崒娑欏磳婵﹤顭峰畷鎺戔枎閹达絿鐛ラ梻浣告啞閹稿鎮疯閸掓帡寮崒妤€浜鹃梻鍫熺⊕閸熺偤鏌￠崱妯肩煂闁逞屽墲椤煤閺嶎灐娲Ω瑜嶉ˉ?
    const requestData = {
      ...createForm,
      priority: normalizeGroupPriority(createForm.priority),
      daily_limit_usd: normalizeOptionalLimit(createForm.daily_limit_usd as number | string | null),
      weekly_limit_usd: normalizeOptionalLimit(createForm.weekly_limit_usd as number | string | null),
      monthly_limit_usd: normalizeOptionalLimit(createForm.monthly_limit_usd as number | string | null),
      model_routing: convertRoutingRulesToApiFormat(createModelRoutingRules.value)
    }
    // v-model.number 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柟闂寸绾惧鏌ｉ幇顒佹儓缂佺姳鍗抽弻鐔兼⒒鐎靛壊妲紓浣哄Х婵炩偓闁哄瞼鍠栭幃褔宕奸悢鍝勫殥闂備胶绮幐濠氭晝閵忋倕钃熼柍銉﹀墯閸氬鏌涢幇鈺佸妞ゎ剙顦辩槐鎾诲磼濮樻瘷銏＄箾瀹割喖寮€殿喖顭烽幃銏ゆ偂鎼达絿鏆伴柣鐔哥矊缁夌懓顕ｉ搹顐ｇ秶闁宠鍎虫禍鐐箾閸繄浠㈤柡瀣堕檮缁绘盯宕ㄩ鐣岊槰缂備礁鐭佹ご鍝ユ崲濠靛纾婚柤鎭掑劜椤ュ牓鏌涢埞鎯т壕婵＄偑鍊栧濠氬磻閹剧粯鎳氶柣鎰嚟缁犻箖鏌涢埄鍐炬畷缂佸倸顑嗙换娑㈠箻閹颁胶鍚嬮梺鍝勬湰缁嬫挻绂掗敃鍌氱鐟滃鍩€椤掍礁绗氶柕鍥у閸╋繝宕熼銈嗩嚄闂備胶鎳撳鍫曟偤閺囩姷涓嶆繛鎴炃氬Σ鍫熶繆椤栨繂鍚归柍?""闂傚倸鍊搁崐鎼佸磹閻戣姤鍊块柨鏃堟暜閸嬫挾绮☉妯诲櫧闁活厽鐟╅弻鐔告綇妤ｅ啯顎嶉梺绋垮椤ㄥ懘婀侀梺鎸庣箓閻楁粓宕垫径灞惧枑闁绘鐗嗙粭鎺擃殽閻愵亜鐏ǎ鍥э躬椤㈡稑顫濇潏銊ф闂?null 闂傚倸鍊搁崐宄懊归崶褏鏆﹂柛顭戝亝閸欏繒鈧娲栧ú锔藉垔婵傚憡鐓涢悘鐐额嚙閸旀岸鏌ｉ妶鍥т壕缂佺粯鐩獮瀣枎韫囨洑鎮ｇ紓鍌欒兌婵敻鎮уΔ浣衡攳濠电姴娲ゅ洿闂佸憡渚楅崢钘夆枔閸洘鈷戦柛娑橆煬濞堟洜鐥弶璺ㄐу┑锛勬暬瀹曠喖顢涘顒€鏁ら梻渚€娼ц噹闁告粈绀侀幃鎴︽⒒閸屾瑧顦﹂柟鑺ョ矒瀹曠増鎯旈姀鈺傤啍闂佸綊妫跨粈渚€鎮￠垾鎰佺唵闁兼悂娼ф慨鍥╃磼閻樺樊鐓奸柟顔肩秺瀹曨偊宕熼浣稿壍濠电姷顣介埀顒冩珪閹牓鏌嶇憴鍕伌闁糕斂鍎靛畷鍗烆渻閸撗勫瘻闂傚倷鑳堕…鍫ユ晝閵壯勫床闁稿瞼鍎戠紞鏍叓閸ャ劍鈷掓い鈺冨厴閹鏁愭惔婵堟晼闂佹眹鍊愰崑鎾斥攽閿涘嫬浜奸柛濠冪墪椤斿繑绻濆顒傦紱闂佺懓澧界划顖炴偂?

    const emptyToNull = (v: any) => v === '' ?
 null : v
    requestData.daily_limit_usd = emptyToNull(requestData.daily_limit_usd)
    requestData.weekly_limit_usd = emptyToNull(requestData.weekly_limit_usd)
    requestData.monthly_limit_usd = emptyToNull(requestData.monthly_limit_usd)
    await adminAPI.groups.create(requestData)
    appStore.showSuccess(t('admin.groups.groupCreated'))
    closeCreateModal()
    loadGroups()
    // Only advance tour if active, on submit step, and creation succeeded
    if (onboardingStore.isCurrentStep('[data-tour="group-form-submit"]')) {
      onboardingStore.nextStep(500)
    }
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.groups.failedToCreate'))
    console.error('Error creating group:', error)
    // Don't advance tour on error
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
  editForm.claude_code_only = group.claude_code_only || false
  editForm.fallback_group_id = group.fallback_group_id
  editForm.fallback_group_id_on_invalid_request = group.fallback_group_id_on_invalid_request
  editForm.allow_messages_dispatch = group.allow_messages_dispatch || false
  editForm.default_mapped_model = group.default_mapped_model || ''
  editForm.model_routing_enabled = group.model_routing_enabled || false
  editForm.supported_model_scopes = group.supported_model_scopes || ['claude', 'gemini_text', 'gemini_image']
  editForm.mcp_xml_inject = group.mcp_xml_inject ?? true
  editForm.copy_accounts_from_group_ids = [] // 濠电姷鏁告慨鐑藉极閸涘﹥鍙忓ù鍏兼綑閸ㄥ倿鏌ｉ幘宕囧哺闁哄鐗楃换娑㈠箣閻愬娈ら梺娲诲幗閹瑰洭寮婚悢铏圭＜闁靛繒濮甸悘鍫ユ⒑閸濆嫬顏ラ柛搴″级缁岃鲸绻濋崶顬囨煕濞戝崬骞楁繛鍫熺叀閺岋絾鎯旈姀锝咁棟濡炪値鍘鹃崗妯侯嚕婵犳艾惟闁宠桨绀佸畵鍡椻攽鎺抽崐鎰板磻閹捐秮鐟邦煥閸垻鏆梺璇″枛閸㈡煡鈥﹂妸鈺佺妞ゆ劦鍋€閸嬫挸顭ㄩ崟顓狀啎婵犮垼娉涢鍡欑矆鐎ｎ喗鐓曢柍瑙勫劤娴滅偓淇婇悙顏勨偓鏍暜婵犲洦鍤勯柛顐ｆ礀閻撯偓闂佸搫娲ㄩ崰鍡樼濠婂牊鐓忓┑鐐茬仢閸旀粍銇勯妷锝呯仾闁靛洤瀚版慨鈧柨娑樺閸ｎ喗绻涚€电顎撶紒鐘虫尭閻ｇ兘鏁愭径妯绘櫇闂佹寧绻傞幊鎰邦敄閸曨厾纾藉ù锝呮惈鏍￠梺鐑╂櫓閸ㄤ即顢氶敐澶樻晪闁逞屽墴閻涱喚鈧綆浜栭弨浠嬫煕濞戞瑥顥嬬紒鐘活棑缁辨捇宕掑顑藉亾閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳婀遍幉鎼佹偋閸繄鐟查梺绋款儏椤戝寮婚敐澶婄疀妞ゆ帒鍊风划闈涱渻閵堝棗濮屽┑顔哄€濇俊鐢稿礋椤栨氨鐤€闂佸憡鎸烽懗鍫曞汲閻樼數纾藉〒姘搐娴滄粎绱掓径濠勭Ш鐎殿喛顕ч埥澶愬閳哄倹娅囬梻浣瑰缁诲倸螞濞戔懞鍥Ψ閵夈垺鏂€闂佺粯鍔曞鍫曞闯閻戣姤鐓曢柕濞垮妼閸氳淇婇崣澶婂妤犵偞顭囬幏鐘绘嚑椤掑袨闂傚倷娴囬鏍垂鎼淬劌绀嬫い鎾寸箘閺佹牠姊婚崒娆愮グ鐎规洖鐏氶幈銊╂偨缁嬭法顦┑掳鍊曢幊搴ｇ玻濡ゅ懏鐓欓柟瑙勫姈绾箖鏌?  // 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸崹楣冨箛娴ｅ湱绋佺紓鍌氬€烽悞锕佹懌闂佸憡鐟ョ换姗€寮婚悢纰辨晬闁挎繂娲ｅЧ妤呮偡濠婂懎顣奸悽顖涘浮閹瑦绻濋崘锔跨盎濡炪倖甯掗ˇ顖涙櫠娴煎瓨鐓曢柣妯虹－婢х數鈧娲栫紞濠囩嵁閸℃凹妲鹃梺鍝勬４缁犳捇寮诲☉銏犖ㄦい鏍ㄧ矊閸╁本绻涚€涙鐭嬮柛搴㈠▕閳ユ棃宕橀浣镐壕闁挎繂绨肩花缁樸亜閹哄鐏柍褜鍓濋～澶娒洪弽顓勫洦瀵奸弶鎳筹箓鏌涢弴銊ョ仩闁告劏鍋撻梻渚€娼ц噹闁告侗鍨扮敮鎺楁⒒閸屾瑧顦﹂柟纰卞亜铻炴繛鎴炵懅缁犻箖鏌嶈閸撶喖寮婚悢纰辨晩闁绘挸瀛╀簺闂備礁鐤囬褍顫濋敂鍓т笉婵炴垯鍨圭粻濂告煕閹板苯瀚禍婵嬫⒒閸屾艾鈧绮堟担鍦彾濠电姴娲﹂崑鍌炴煕椤愮姴鍔氶柛銊ュ€块弻娑㈩敃閿濆棛顦ラ梺鍝勵儎閼冲爼鍩€椤掆偓缁犲秹宕曢柆宓ュ洭顢涢悙瀵稿帒闂佹悶鍎崝搴ｅ姬閳ь剟姊婚崒姘卞缂佸鎸婚弲鍫曞閵堝棛鍙嗗┑鐐村灦閻熝呯不閹剧粯鐓冮悷娆忓閻忓瓨銇勯姀锛勬噰闁诡喗鍎抽悾婵嬪焵椤掑倹顫曞ù鐓庣摠閳锋垹绱掔€ｎ偄顕滈柛鐐差槸椤啰鈧稒蓱閸婃劖顨ラ悙鑼ⅵ濠碘剝鎮傞崺锟犲磼濮橈絽浜鹃柣鎴ｅГ閸婂灚绻涢幋鐑嗕痪妞ゅ繐鐗炵紞鏍煥閻斿搫校闁抽攱鍨圭槐鎺斺偓锝庝簻閻繝鏌涢弮鍌氭灈闁哄本鐩幃鈺呭蓟閵夈儱鍙婇梺鑺ド戠换鍫ュ蓟瀹ュ浼犻柛鏇ㄥ墮濞呫倝姊虹紒妯诲鞍婵炲弶顭囬幑銏犫槈閵忕姴鑰垮┑掳鍊曢崯浼搭敊閸パ€鏀介柍鈺佸暙缁茬粯銇勯鐘插幋鐎殿喖顭烽幃銏㈠枈鏉堛劍娅撻梻浣稿悑娴滀粙宕曢幎鍓垮洭顢橀悩鐢碉紳婵炴挻鑹惧ú銈夊几閻斿憡鍙忓┑鐘插亞閻撹偐鈧鍣崑濠囥€佸璺虹劦妞ゆ帒瀚ㄩ埀顑跨閳诲酣骞樺畷鍥舵Ф闁荤喐绮嶇划灞藉祫?
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
}

const handleUpdateGroup = async () => {
  if (!editingGroup.value) return
  if (!editForm.name.trim()) {
    appStore.showError(t('admin.groups.nameRequired'))
    return
  }

  submitting.value = true
  try {
    // 闂傚倸鍊搁崐椋庣矆娓氣偓楠炴牠顢曚綅閸ヮ剚鐒肩€广儱鎳愰敍鐔兼⒑閸︻厼顣兼繝銏☆焽缁牓宕奸悢绋垮伎濠殿喗顨呭Λ妤佹櫠娴煎瓨鐓?fallback_group_id: null -> 0 (闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸弫鎾绘偐閸愬弶鐤勯梻浣筋嚃閸ㄥジ鎮橀幇顖樹汗闁圭儤鎸搁埀顒冨吹缁辨帒鈽夊鍡楀壉闂佸搫鎳忕划宀勨€旈崘顔嘉ч柛鈩冾殘閻熸劙姊虹紒妯洪嚋闂傚嫬瀚板畷姘節閸パ咁啋濡炪倖妫佸Λ鍕椤栫偞鈷戦悹鍥ｂ偓鍐茬闁汇埄鍨辩敮妤佺┍婵犲啰顩烽悗锝庡亞閸?0 闂傚倸鍊搁崐宄懊归崶褏鏆﹂柣銏㈩焾绾惧鏌ｉ幇顔芥毄闁活厽鐟╅悡顐﹀炊閵娧€妲堢紓浣插亾濠㈣埖鍔曠粻瑙勭箾閿濆骸澧┑锛勫帶椤╁ジ宕ㄩ娑欐杸濡炪倖姊归弸缁樼瑹濞戙垺鐓曢煫鍥ㄦ⒒閹冲洭鏌涢埞鎯т壕婵＄偑鍊栫敮濠勭矆娴ｈ褰掝敊闁款垰浜鹃悷娆忓绾炬悂鏌涢弬鍧楀弰闁糕斁鍋撳銈嗗笂閼冲爼鍩婇弴鐔翠簻妞ゆ挾鍋炵粚鍧楁煏?
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
    // v-model.number 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柟闂寸绾惧鏌ｉ幇顒佹儓缂佺姳鍗抽弻鐔兼⒒鐎靛壊妲紓浣哄Х婵炩偓闁哄瞼鍠栭幃褔宕奸悢鍝勫殥闂備胶绮幐濠氭晝閵忋倕钃熼柍銉﹀墯閸氬鏌涢幇鈺佸妞ゎ剙顦辩槐鎾诲磼濮樻瘷銏＄箾瀹割喖寮€殿喖顭烽幃銏ゆ偂鎼达絿鏆伴柣鐔哥矊缁夌懓顕ｉ搹顐ｇ秶闁宠鍎虫禍鐐箾閸繄浠㈤柡瀣堕檮缁绘盯宕ㄩ鐣岊槰缂備礁鐭佹ご鍝ユ崲濠靛纾婚柤鎭掑劜椤ュ牓鏌涢埞鎯т壕婵＄偑鍊栧濠氬磻閹剧粯鎳氶柣鎰嚟缁犻箖鏌涢埄鍐炬畷缂佸倸顑嗙换娑㈠箻閹颁胶鍚嬮梺鍝勬湰缁嬫挻绂掗敃鍌氱鐟滃鍩€椤掍礁绗氶柕鍥у閸╋繝宕熼銈嗩嚄闂備胶鎳撳鍫曟偤閺囩姷涓嶆繛鎴炃氬Σ鍫熶繆椤栨繂鍚归柍?""闂傚倸鍊搁崐鎼佸磹閻戣姤鍊块柨鏃堟暜閸嬫挾绮☉妯诲櫧闁活厽鐟╅弻鐔告綇妤ｅ啯顎嶉梺绋垮椤ㄥ懘婀侀梺鎸庣箓閻楁粓宕垫径灞惧枑闁绘鐗嗙粭鎺擃殽閻愵亜鐏ǎ鍥э躬椤㈡稑顫濇潏銊ф闂?null 闂傚倸鍊搁崐宄懊归崶褏鏆﹂柛顭戝亝閸欏繒鈧娲栧ú锔藉垔婵傚憡鐓涢悘鐐额嚙閸旀岸鏌ｉ妶鍥т壕缂佺粯鐩獮瀣枎韫囨洑鎮ｇ紓鍌欒兌婵敻鎮уΔ浣衡攳濠电姴娲ゅ洿闂佸憡渚楅崢钘夆枔閸洘鈷戦柛娑橆煬濞堟洜鐥弶璺ㄐу┑锛勬暬瀹曠喖顢涘顒€鏁ら梻渚€娼ц噹闁告粈绀侀幃鎴︽⒒閸屾瑧顦﹂柟鑺ョ矒瀹曠増鎯旈姀鈺傤啍闂佸綊妫跨粈渚€鎮￠垾鎰佺唵闁兼悂娼ф慨鍥╃磼閻樺樊鐓奸柟顔肩秺瀹曨偊宕熼浣稿壍濠电姷顣介埀顒冩珪閹牓鏌嶇憴鍕伌闁糕斂鍎靛畷鍗烆渻閸撗勫瘻闂傚倷鑳堕…鍫ユ晝閵壯勫床闁稿瞼鍎戠紞鏍叓閸ャ劍鈷掓い鈺冨厴閹鏁愭惔婵堟晼闂佹眹鍊愰崑鎾斥攽閿涘嫬浜奸柛濠冪墪椤斿繑绻濆顒傦紱闂佺懓澧界划顖炴偂?

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

// 闂傚倸鍊搁崐鎼佸磹閻戣姤鍤勯柛顐ｆ礀绾惧潡鏌ｉ姀銏╃劸闁汇倗鍋撶换婵囩節閸屾粌顣洪梺鎼炲妼閸婂潡寮婚敐澶婎潊闁靛繆鏅濋崝鍝ョ磽?subscription_type 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸弫鎾绘偐閸愯弓鐢婚梻浣瑰濞叉牠宕愯ぐ鎺戠柧妞ゅ繐鐗婇埛鎴︽偣閸ャ劎鍙€妞ゅ孩顨婇弻锝堢疀閺傚灝鎽甸梺璇″櫙缁绘繈寮幘缁樺亹闁告瑥顦遍埀顒夊墴閹鐛崹顔煎闂佺娅曢崝娆忣嚕閹惰棄骞㈡繛鎴炵懅閸橀亶姊鸿ぐ鎺戜喊闁告ê銈稿畷婵嬪Χ婢跺鍘靛銈嗘煥閸氬宕甸崶銊﹀弿濠电姴鍋嗛悡鑲┾偓瑙勬礃鐢帡锝炲┑瀣垫晢濞达綀顕栭崬鏌ユ⒒閸屾瑨鍏岄柟铏崌瀹曨垳鎹勯妸锕€搴婂┑掳鍊曢幊搴ㄦ偪椤斿浜滈柡宥庡亜娴狅箓鏌涙繝鍌滀粵缂佺粯鐩畷鐓庘攽閸粏妾搁梻浣告惈椤戝棛绮欓幘鑸殿潟闁圭儤姊荤壕鍏间繆椤栨繃顏犲ù鐘虫綑椤?is_exclusive 濠电姷鏁告慨鐢割敊閺嶎厼绐楁俊銈呭暞閺嗘粍淇婇妶鍛殶闁活厽鐟╅弻鐔兼倻濡晲绮堕梺閫炲苯澧剧紒鐘虫尭閻ｉ攱绺界粙璇俱劍銇勯弮鍥撴繛鍛Ф缁辨捇宕掑顑藉亾閻戣姤鍊块柨鏇炲€甸埀顒婄畵瀹曞爼顢楅埀顒勬偂?true
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
  }
)

// 闂傚倸鍊搁崐鎼佸磹閻戣姤鍤勯柛鎾茬閸ㄦ繃銇勯弽顐粶缂佲偓婢跺绻嗛柕鍫濇噺閸ｅ湱绱掗悩闈涒枅闁哄瞼鍠栭獮鎴﹀箛闂堟稒顔勯梻浣告啞娣囨椽锝炴径鎰﹂柛鏇ㄥ灠缁秹鏌涢妷顔惧帥婵☆偄瀚槐鎾存媴缁嬫鏆㈤梺绋款儍閸婃繂顕ｆ繝姘伋闁归绀侀幃鎴︽煙閼测晞藟闁逞屽墮绾绢參顢欓幇鐗堚拻濞达絽鎲￠崯鐐烘煛瀹€瀣М鐎规洘娲熼幃鐣岀矙閼愁垱鎲版繝鐢靛仦閸ㄥ墎鍒掓惔銊﹀仾闁逞屽墴濮婃椽宕崟顐熷亾閸濄儴濮抽梺顒€绉撮崙鐘崇箾閸℃绠氶柡鈧禒瀣厓闁芥ê顦伴ˉ婊兠瑰鍕煁闁靛洤瀚板鎾Ω閵夈儳鐫勬俊銈囧Х閸嬫稓鎹㈠Δ鍛﹂柟鐗堟緲缁犳娊鏌熼幆褏鎽犻柣婵囩墵濮婄粯绗熼埀顒勫焵椤掑倸浠滈柤娲诲灡閺呭爼寮婚妷锔惧帾闂佹悶鍎崝灞炬叏瀹ュ洠鍋撶憴鍕妞ゎ厼鐗愰悘鍐⒑闁偛鑻晶鎵磼椤斿灝鍚归柟鍙夋尦瀹曠喖顢橀悩鍨潓婵犵數濮烽弫鍛婃叏閹绢喖纾圭憸鐗堝笒闂傤垳鎲搁悧鍫濈瑲闁绘挻娲熼幃妤呮晲鎼存繄鏁栭悗娑欑箞濮婅櫣鈧湱濮甸ˉ澶嬨亜閿旂偓鏆柣娑卞枛椤粓鍩€椤掑嫮宓侀柛銉ｅ妽婵绱掔€ｎ亞浠㈡い銉ュ缁绘繈鎮介棃娴讹絾銇勯弮鈧悧鐘茬暦鐟欏嫬顕遍柡澶嬪灩椤旀劕顪冮妶鍡楀Ё閻犳劗鍠栭崺鈧?
const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  // 婵犵數濮烽弫鍛婃叏閻戝鈧倿鎸婃竟鈺嬬秮瀹曘劑寮堕幋鐙呯幢闂備線鈧偛鑻晶鎾煛鐏炲墽銆掗柍褜鍓ㄧ紞鍡涘磻閸涱厾鏆︾€光偓閸曨剛鍘搁悗鍏夊亾閻庯綆鍓涢敍鐔哥箾鐎电顎撳┑鈥虫喘楠炲繘鎮╃拠鑼唽闂佸湱鍎ら崺鍫濐焽閵夈儮鏀介柣妯活問閺嗩垶鏌嶈閸撴瑩宕捄銊ф／鐟滄棃寮婚悢纰辨晩闁绘挸绨堕崑鎾诲箹娴ｇ懓浠奸梺缁樺灱濡嫬鏁梻浣稿暱閹碱偊宕愰悷鎵虫瀺闁糕剝绋掗埛鎴︽偣閸ワ絺鍋撳畷鍥ｅ亾婵犳碍鐓曢悘鐐村礃婢规ɑ銇勯弴鍡楁处閳锋帡鏌涚仦鍓ф噮妞わ讣绠撻弻鐔烘嫚瑜忕弧鈧悗瑙勬礉椤缂撴禒瀣闁告瑥顦板▍鏃堟煟閻斿摜鐭婄紒澶婄秺瀵偊宕橀纰辨綂闂侀潧鐗嗙€涒晠寮插鍫熲拺闁告稑锕ｇ欢閬嶆煕椤垳閭€殿喗鎮傚浠嬵敇閻斿搫骞楅梺鐟板悑閹矂宕瑰畷鍥╃煋闁汇垹鎲￠悡鏇㈡倵閿濆簼绨绘い蹇ｅ幗閵囧嫰濮€閳╁啫纰嶉梺瀹狀嚙闁帮綁鐛崶顒夋晜闁糕剝鐟ч弳鏉库攽閻樿尙妫勯柡澶婄氨閸嬫捇骞囬弶璺紱闂佺懓澧界换婵堟崲閸℃ǜ浜滈柟鏉垮閹ジ鏌熼崘鍙夊窛闁逞屽墲椤煤閺嶎偆绀婂┑鐘插暕缁诲棝鏌熼梻瀵稿妽闁哄懏绮撻弻锝夋晲閸涱垳浼囬柤鍙夌墵濮婂宕掑▎鎰偘濡炪値鍋勯ˇ闈涚暦閺囥垹绠柦妯侯槺閺屽牓姊洪崜鑼帥闁革綆鍣ｅ畷鐢稿即閵忥紕鍙勯棅顐㈡祫缁茶姤绂嶅Δ鍐＜閻犲洤寮堕ˉ鐐睬庨崶褝韬┑鈥崇埣瀹曟帒顫濋銏╂婵犵數濮烽。钘壩ｉ崨鏉戠；闁告稒鐣埀顒€鍟换婵嬪磼濠婂嫭顔?

  if (!target.closest('.account-search-container')) {
    Object.keys(showAccountDropdown.value).forEach(key => {
      showAccountDropdown.value[key] = false
    })
  }
}

// 闂傚倸鍊搁崐鎼佸磹閻戣姤鍤勯柛顐ｆ礀缁犵娀鏌熼崜褏甯涢柛濠呭煐閹便劌螣閹稿海銆愮紓浣哄С閸楁娊寮诲☉妯锋斀闁告洦鍋勬慨鏇烆渻閵堝骸浜濇繛鑼枛瀵鎮㈤崨濠勭Ф闂佸憡鎸嗛崨顔筋啅缂傚倸鍊烽懗鑸垫叏閸偆绠惧┑鐘叉搐閽冪喖鏌ｉ弬鍨倯闁稿﹦绮穱濠囶敍濠靛浂浠╂繛瀵稿Ь閸嬫劗妲愰幒妤佸€锋い鎺嶈兌閸戯紕绱撴担鍓叉Ц闁绘牕銈稿畷娲焵椤掍降浜滈柟鍝勭Ф鐠愪即鏌涢悢椋庣闁哄本鐩幃鈺佺暦閸パ€鍚傞梻浣瑰濞插繘宕规禒瀣摕闁糕剝顨忛崥瀣煕閳╁啨浠︾紒?

const openSortModal = async () => {
  try {
    // 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾鐎规洏鍎抽埀顒婄秵閸犳牜澹曢崸妤佺厵闁诡垳澧楅ˉ澶愬箹閺夋埊韬柡灞诲€濋幊婵嬪箥椤旇偐澧┑鐐茬摠缁瞼绱炴繝鍥ц摕婵炴垯鍨瑰敮濡炪倖姊婚崢褔锝為埡鍐＝濞达絽鎼ⅷ闂佺娴烽弫璇差嚕鐠囧樊鍚嬮柛顐亝椤庡洭姊绘担鍛婂暈闁规悂绠栧畷鐗堟償椤垶鏅梺鎸庣箓椤︻垶宕归崒娑栦簻闁规壋鏅涢悘顔剧磼閹邦喖浠︾紒缁樼箞閹粙妫冨ù韬插灲閺屾稒绻濋崒銈囧悑闂佺硶鏂侀崑鎾愁渻閵堝棗绗掗柛鐕佸亰瀹曘儵宕烽鐘碉紲闁哄鐗勯崝宥囦焊娴煎瓨鐓欑€瑰嫮澧楅崳鐣岀磼椤旂晫鎳冮柍璇查叄瀹曪綁濡疯琚濈紓鍌氬€搁崐鎼佸磹瀹勬噴褰掑炊閵婏絼绮撻梺褰掓？閻掞箓宕戦敓鐘崇叆闁哄倸鐏濋埛鏃堟煟椤撶喓鎳冩い顓℃硶閹瑰嫭绗熼婊冨絾闂備礁鎲￠懝楣冾敄婢舵劕钃熸繛鎴炃氬Σ鍫ユ煕濡ゅ啫浠滅紒鐘差煼濮婃椽鏌呴悙鑼跺濠⒀冨级閵囧嫰寮撮崱妤佺闁逞屽墯濡啫鐣峰鈧、娆撳床婢诡垰娲﹂悡鏇㈡煏婢跺牆濡奸柣鎾村姈缁绘盯宕奸悢濂夋缂?

    const allGroups = await adminAPI.groups.getAll()
    // 闂?sort_order 闂傚倸鍊搁崐鎼佸磹妞嬪海鐭嗗ù锝夋交閼板潡姊洪鈧粔瀵稿閸ф鐓忛柛顐ｇ箖婢跺嫰鏌￠崱妯肩煉闁哄瞼鍠栭幃婊兾熷ú缁樷挅濠?
    sortableGroups.value = [...allGroups].sort((a, b) => a.sort_order - b.sort_order)
    showSortModal.value = true
  } catch (error) {
    appStore.showError(t('admin.groups.failedToLoad'))
    console.error('Error loading groups for sorting:', error)
  }
}

// 闂傚倸鍊搁崐鎼佸磹閻戣姤鍤勯柛顐ｆ磵閳ь剨绠撳畷濂稿閳ュ啿绨ラ梻浣烘嚀椤曨參宕戦悢铏逛笉闁诡垎鈧弨浠嬫煟濡灝绱﹂弶鈺勵潐缁绘盯骞嬪▎蹇曚患缂備胶濮风划顖炲Φ閸曨垰绫嶉柛銉戝倹鐫忔俊鐐€栭弻銊╂晝椤忓嫷娼栨繛宸簻閹硅埖銇勯幘璺轰粶濠碘剝妞藉娲箹閻愭彃顬堝┑鐘亾闂侇剙绉寸粣妤€鈹戦悩鎻掍簽闁告艾顑夐弻娑㈠Ψ椤栨粌鍩屾繛瀛樼矌閸嬫挾鎹㈠☉銏犻唶婵犻潧鐗呴搹搴ㄦ⒑閸濆嫷鍎愮紒瀣笧閹广垹鈹戠€ｎ亞鍔﹀銈嗗笒鐎氼參鎮為懖鈹惧亾楠炲灝鍔氭俊顐ｎ殕缁?

const closeSortModal = () => {
  showSortModal.value = false
  sortableGroups.value = []
}

// 濠电姷鏁告慨鐑藉极閹间礁纾块柟瀵稿Т缁躲倝鏌﹀Ο渚＆婵炲樊浜濋弲婊堟煟閹伴潧澧幖鏉戯躬濮婅櫣绮欑捄銊т紘闂佺顑囬崑銈呯暦閹达箑围濠㈣泛顑囬崢顏呯節閻㈤潧浠ч柛瀣尭閳诲秹宕ㄩ婊咃紲闂佺粯锚閸熷潡鍩㈤弴銏＄厸鐎光偓閳ь剟宕伴弽顓炵畺婵犲﹤鍚橀悢鍏兼優闂侇偅绋掗崑鍛磽?

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
