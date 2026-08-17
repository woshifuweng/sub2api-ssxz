<template>
  <component :is="pageShell" v-bind="pageShellProps">
    <div
      :class="[
        'keys-page-surface',
        { 'keys-page-surface--workbench': useWorkbenchShell },
      ]"
    >
      <BalanceWarningBanner />
      <a
        class="keys-purchase-link"
        href="https://pay.ldxp.cn/shop/VT7XKDFI"
        target="_blank"
        rel="noopener noreferrer"
      >
        <Icon name="creditCard" size="sm" aria-hidden="true" />
        <span>购买余额</span>
        <Icon name="externalLink" size="xs" aria-hidden="true" />
      </a>

      <TablePageLayout
        :class="[
          { 'keys-workbench-layout': useWorkbenchShell },
          {
            'keys-workbench-layout--no-pagination':
              useWorkbenchShell && pagination.total === 0,
          },
        ]"
      >
        <template #filters>
          <div class="space-y-3">
            <div class="keys-access-row">
              <div class="keys-access-base" data-testid="keys-base-url-row">
                <span>{{ t("keys.workbenchGuide.baseUrlLabel") }}</span>
                <code>{{ openAICompatibleBaseUrl }}</code>
                <LiquidButton
                  type="button"
                  class="keys-access-copy"
                  data-testid="keys-guide-copy-base-url"
                  :title="
                    baseUrlCopied
                      ? t('keys.workbenchGuide.baseUrlCopied')
                      : t('keys.workbenchGuide.copyBaseUrl')
                  "
                  @click="copyBaseUrl"
                  variant="plain"
                  size="icon"
                >
                  <Icon
                    v-if="baseUrlCopied"
                    name="check"
                    size="sm"
                    :stroke-width="2"
                  />
                  <Icon v-else name="clipboard" size="sm" />
                </LiquidButton>
              </div>
              <a href="/docs" class="keys-doc-link">
                <Icon name="document" size="sm" aria-hidden="true" />
                <span>{{ t("home.viewDocs") }}</span>
                <Icon name="chevronRight" size="sm" aria-hidden="true" />
              </a>
            </div>
            <div class="keys-filter-row">
              <SearchInput
                v-model="filterSearch"
                :placeholder="t('keys.searchPlaceholder')"
                class="w-full sm:w-64"
                @search="onFilterChange"
              />
              <Select
                :model-value="filterGroupId"
                class="w-40"
                :options="groupFilterOptions"
                @update:model-value="onGroupFilterChange"
              />
              <Select
                :model-value="filterStatus"
                class="w-40"
                :options="statusFilterOptions"
                @update:model-value="onStatusFilterChange"
              />
            </div>
          </div>
        </template>

        <template #actions>
          <div class="flex justify-end gap-3">
            <LiquidButton
              @click="loadApiKeys"
              :disabled="loading"
              :title="t('common.refresh')"
              variant="outline"
              size="icon"
            >
              <Icon
                name="refresh"
                size="md"
                :class="loading ? 'animate-spin' : ''"
              />
            </LiquidButton>
            <div ref="columnDropdownRef" class="relative">
              <LiquidButton
                type="button"
                data-testid="keys-column-settings-trigger"
                :title="t('admin.users.columnSettings')"
                :aria-label="t('admin.users.columnSettings')"
                :aria-expanded="showColumnDropdown"
                @click.stop="showColumnDropdown = !showColumnDropdown"
                variant="outline"
                size="default"
              >
                <Icon name="grid" size="sm" />
                <span>{{ t("admin.users.columnSettings") }}</span>
              </LiquidButton>
              <div
                v-if="showColumnDropdown"
                class="keys-column-menu"
                data-testid="keys-column-settings-menu"
                role="menu"
              >
                <button
                  v-for="column in toggleableColumns"
                  :key="column.key"
                  type="button"
                  class="keys-column-menu__item"
                  role="menuitemcheckbox"
                  :data-column-key="column.key"
                  :aria-checked="isColumnVisible(column.key)"
                  @click="toggleColumn(column.key)"
                >
                  <span>{{ column.label }}</span>
                  <Icon
                    v-if="isColumnVisible(column.key)"
                    name="check"
                    size="sm"
                    :stroke-width="2"
                  />
                </button>
              </div>
            </div>
            <LiquidButton
              @click="openCreateModal"
              :disabled="groups.length === 0"
              data-tour="keys-create-btn"
              :title="
                groups.length === 0
                  ? t('keys.noAvailableGroups')
                  : t('keys.createKey')
              "
              size="default"
            >
              <Icon name="plus" size="md" />
              {{ t("keys.createKey") }}
            </LiquidButton>
          </div>
        </template>

        <template #table>
          <DataTable :columns="columns" :data="apiKeys" :loading="loading">
            <template #cell-key="{ value, row }">
              <div class="flex items-center gap-2">
                <code class="code text-xs">
                  {{ maskKey(value) }}
                </code>
                <LiquidButton
                  @click="copyToClipboard(row)"
                  class="rounded-lg p-1 transition-colors"
                  :class="[
                    'hover:bg-gray-100 dark:hover:bg-dark-700',
                    copiedKeyId === row.id
                      ? 'text-primary-600 dark:text-primary-400'
                      : 'text-gray-400 hover:text-gray-600 dark:hover:text-gray-300',
                  ]"
                  :title="
                    copiedKeyId === row.id
                      ? t('keys.copied')
                      : t('keys.copyToClipboard')
                  "
                  variant="plain"
                  size="icon"
                >
                  <Icon
                    v-if="copiedKeyId === row.id"
                    name="check"
                    size="sm"
                    :stroke-width="2"
                  />
                  <Icon v-else name="clipboard" size="sm" />
                </LiquidButton>
              </div>
            </template>

            <template #cell-name="{ value, row }">
              <div class="flex items-center gap-1.5">
                <span class="font-medium text-gray-900 dark:text-white">{{
                  value
                }}</span>
                <Icon
                  v-if="
                    row.ip_whitelist?.length > 0 || row.ip_blacklist?.length > 0
                  "
                  name="shield"
                  size="sm"
                  class="text-primary-600 dark:text-primary-300"
                  :title="t('keys.ipRestrictionEnabled')"
                />
              </div>
            </template>

            <template #cell-group="{ row }">
              <div class="relative">
                <LiquidButton
                  :ref="(el) => setGroupButtonRef(row.id, el)"
                  @click="editKey(row)"
                  class="-mx-2 -my-1 flex cursor-pointer items-center gap-2 rounded-lg px-2 py-1 transition-all duration-200 hover:bg-gray-100 dark:hover:bg-dark-700"
                  :title="t('common.edit')"
                  variant="plain"
                  size="sm"
                >
                  <div
                    v-if="(row.groups && row.groups.length > 0) || row.group"
                    class="flex flex-wrap items-center gap-1.5"
                  >
                    <GroupBadge
                      v-for="group in row.groups && row.groups.length > 0
                        ? row.groups.slice(0, 2)
                        : row.group
                          ? [row.group]
                          : []"
                      :key="group.id"
                      :name="formatGroupDisplayName(group)"
                      :platform="group.platform"
                      :subscription-type="group.subscription_type"
                      :show-rate="false"
                      variant="outline"
                    />
                    <span
                      v-if="row.groups && row.groups.length > 2"
                      class="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-dark-700 dark:text-gray-300"
                    >
                      +{{ row.groups.length - 2 }}
                    </span>
                  </div>
                  <span
                    v-else
                    class="text-sm text-gray-400 dark:text-dark-500"
                    >{{ t("keys.noGroup") }}</span
                  >
                </LiquidButton>
              </div>
            </template>

            <template #cell-usage="{ row }">
              <div
                class="keys-usage-cell"
                data-testid="keys-usage-cell"
                :title="usageCellTitle(row)"
              >
                <template v-if="row.quota > 0">
                  <span class="keys-usage-bar">
                    <span
                      :class="[
                        'keys-usage-bar__fill',
                        row.quota_used >= row.quota
                          ? 'keys-usage-bar__fill--danger'
                          : '',
                      ]"
                      :style="{
                        width:
                          Math.min(
                            ((row.quota_used || 0) / row.quota) * 100,
                            100,
                          ) + '%',
                      }"
                    />
                  </span>
                  <span
                    :class="[
                      'text-sm tabular-nums',
                      row.quota_used >= row.quota
                        ? 'font-medium text-red-500'
                        : 'text-gray-500 dark:text-gray-400',
                    ]"
                  >
                    {{ formatCurrency(row.quota_used || 0) }} /
                    {{ formatCurrency(row.quota || 0) }}
                  </span>
                </template>
                <span
                  v-else
                  class="text-sm tabular-nums text-gray-500 dark:text-gray-400"
                >
                  {{
                    formatCurrency(usageStats[row.id]?.total_actual_cost ?? 0)
                  }}
                </span>
              </div>
            </template>

            <template #cell-rate_limit="{ row }">
              <div
                v-if="tightestRateWindow(row)"
                class="keys-usage-cell"
                data-testid="keys-rate-limit-cell"
                :title="rateLimitCellTitle(row)"
              >
                <span class="text-xs text-gray-500 dark:text-gray-400">{{
                  tightestRateWindow(row)!.label
                }}</span>
                <span class="keys-usage-bar">
                  <span
                    :class="[
                      'keys-usage-bar__fill',
                      tightestRateWindow(row)!.usage >=
                      tightestRateWindow(row)!.limit
                        ? 'keys-usage-bar__fill--danger'
                        : '',
                    ]"
                    :style="{
                      width:
                        Math.min(
                          (tightestRateWindow(row)!.usage /
                            tightestRateWindow(row)!.limit) *
                            100,
                          100,
                        ) + '%',
                    }"
                  />
                </span>
                <span
                  :class="[
                    'text-sm tabular-nums',
                    tightestRateWindow(row)!.usage >=
                    tightestRateWindow(row)!.limit
                      ? 'font-medium text-red-500'
                      : 'text-gray-500 dark:text-gray-400',
                  ]"
                >
                  {{ formatCurrency(tightestRateWindow(row)!.usage) }} /
                  {{ formatCurrency(tightestRateWindow(row)!.limit) }}
                </span>
                <LiquidButton
                  v-if="
                    row.usage_5h > 0 || row.usage_1d > 0 || row.usage_7d > 0
                  "
                  @click.stop="confirmResetRateLimitFromTable(row)"
                  class="keys-action-button !h-7 !w-7"
                  :title="t('keys.resetRateLimitUsage')"
                  :aria-label="t('keys.resetRateLimitUsage')"
                  variant="plain"
                  size="icon"
                >
                  <Icon name="refresh" size="xs" />
                  <span class="sr-only">{{
                    t("keys.resetRateLimitUsage")
                  }}</span>
                </LiquidButton>
              </div>
              <span v-else class="text-sm text-gray-400 dark:text-dark-500"
                >-</span
              >
            </template>

            <template #cell-expires_at="{ value }">
              <span
                v-if="value"
                :class="[
                  'text-sm',
                  new Date(value) < new Date()
                    ? 'text-red-500 dark:text-red-400'
                    : 'text-gray-500 dark:text-dark-400',
                ]"
              >
                {{ formatDateTime(value) }}
              </span>
              <span v-else class="text-sm text-gray-400 dark:text-dark-500">{{
                t("keys.noExpiration")
              }}</span>
            </template>

            <template #cell-status="{ value }">
              <span class="keys-status-cell" data-testid="keys-status-cell">
                <span
                  :class="[
                    'keys-status-dot',
                    value === 'quota_exhausted' || value === 'expired'
                      ? 'keys-status-dot--danger'
                      : value === 'inactive'
                        ? 'keys-status-dot--muted'
                        : '',
                  ]"
                />
                <span class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t("keys.status." + value) }}
                </span>
              </span>
            </template>

            <template #cell-last_used_at="{ value }">
              <span
                v-if="value"
                class="text-sm text-gray-500 dark:text-dark-400"
              >
                {{ formatDateTime(value) }}
              </span>
              <span v-else class="text-sm text-gray-400 dark:text-dark-500"
                >-</span
              >
            </template>

            <template #cell-created_at="{ value }">
              <span class="text-sm text-gray-500 dark:text-dark-400">{{
                formatDateTime(value)
              }}</span>
            </template>

            <template #cell-actions="{ row }">
              <div class="keys-row-actions">
                <!-- Connection Tutorial Button -->
                <LiquidButton
                  type="button"
                  @click="openUseKeyModal(row)"
                  :title="t('keys.useKey')"
                  :aria-label="t('keys.useKey')"
                  class="keys-action-button"
                  variant="plain"
                  size="icon"
                >
                  <Icon name="terminal" size="sm" />
                  <span class="sr-only">{{ t("keys.useKey") }}</span>
                </LiquidButton>
                <!-- Import to CCS Button -->
                <LiquidButton
                  v-if="!publicSettings?.hide_ccs_import_button"
                  type="button"
                  data-testid="api-key-ccs-import"
                  @click="importToCcswitch(row)"
                  :title="t('keys.importToCcSwitch')"
                  :aria-label="t('keys.importToCcSwitch')"
                  class="keys-action-button"
                  variant="plain"
                  size="icon"
                >
                  <Icon name="upload" size="sm" />
                  <span class="sr-only">{{ t("keys.importToCcSwitch") }}</span>
                </LiquidButton>
                <!-- Edit Button -->
                <LiquidButton
                  @click="editKey(row)"
                  :title="t('common.edit')"
                  :aria-label="t('common.edit')"
                  class="keys-action-button"
                  variant="plain"
                  size="icon"
                >
                  <Icon name="edit" size="sm" />
                  <span class="sr-only">{{ t("common.edit") }}</span>
                </LiquidButton>
                <!-- Toggle Status Button -->
                <LiquidButton
                  @click="confirmToggleKeyStatus(row)"
                  :disabled="statusUpdatingKeyId === row.id"
                  :title="
                    row.status === 'active'
                      ? t('keys.disable')
                      : t('keys.enable')
                  "
                  :aria-label="
                    row.status === 'active'
                      ? t('keys.disable')
                      : t('keys.enable')
                  "
                  :class="[
                    'keys-action-button',
                    statusUpdatingKeyId === row.id
                      ? 'cursor-not-allowed opacity-50'
                      : '',
                    row.status === 'active'
                      ? ''
                      : 'keys-action-button--primary',
                  ]"
                  variant="plain"
                  size="icon"
                >
                  <Icon v-if="row.status === 'active'" name="ban" size="sm" />
                  <Icon v-else name="checkCircle" size="sm" />
                  <span class="sr-only">{{
                    row.status === "active"
                      ? t("keys.disable")
                      : t("keys.enable")
                  }}</span>
                </LiquidButton>
                <!-- Delete Button -->
                <LiquidButton
                  @click="confirmDelete(row)"
                  :title="t('common.delete')"
                  :aria-label="t('common.delete')"
                  class="keys-action-button keys-action-button--danger"
                  variant="plain"
                  size="icon"
                >
                  <Icon name="trash" size="sm" />
                  <span class="sr-only">{{ t("common.delete") }}</span>
                </LiquidButton>
              </div>
            </template>

            <template #empty>
              <EmptyState
                :title="t('keys.noKeysYet')"
                :description="t('keys.createFirstKey')"
                :action-text="t('keys.createKey')"
                @action="openCreateModal"
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
    </div>

    <!-- Create/Edit Modal -->
    <BaseDialog
      :show="showCreateModal || showEditModal"
      :title="showEditModal ? t('keys.editKey') : t('keys.createKey')"
      width="normal"
      @close="closeModals"
    >
      <form id="key-form" @submit.prevent="handleSubmit" class="space-y-5">
        <div>
          <label class="input-label">{{ t("keys.nameLabel") }}</label>
          <input
            v-model="formData.name"
            type="text"
            required
            class="input"
            :placeholder="t('keys.namePlaceholder')"
            data-tour="key-form-name"
          />
        </div>

        <div>
          <label class="input-label">{{ t("keys.groupLabel") }}</label>
          <p class="mb-2 text-xs leading-5 text-gray-500 dark:text-gray-400">
            {{ t("keys.groupClientHint") }}
          </p>
          <div
            v-if="groups.length === 0"
            class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm leading-6 text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-100"
            data-testid="key-form-no-groups"
          >
            {{ t("keys.noAvailableGroups") }}
          </div>
          <Select
            v-else
            :model-value="selectedFormGroupId"
            :options="groupOptions"
            :placeholder="t('keys.selectGroup')"
            :search-placeholder="t('keys.searchGroup')"
            searchable
            data-tour="key-form-group"
            data-testid="key-form-group-select"
            @update:model-value="selectFormGroup"
          >
            <template #selected="{ option }">
              <div
                v-if="option"
                class="flex min-w-0 flex-1 items-center gap-2.5 text-left"
              >
                <span
                  :class="[
                    'inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md',
                    getGroupProviderIconClass(getGroupOptionPlatform(option)),
                  ]"
                >
                  <PlatformIcon
                    :platform="getGroupOptionPlatform(option)"
                    size="sm"
                  />
                </span>
                <span
                  class="min-w-0 flex-1 truncate text-sm font-semibold text-gray-900 dark:text-white"
                >
                  {{ option.label }}
                </span>
                <span
                  :class="[
                    'shrink-0 rounded-full px-2.5 py-1 text-xs font-semibold',
                    getGroupRateBadgeClass(option),
                  ]"
                >
                  {{ formatGroupRate(option) }}x
                </span>
              </div>
              <span v-else class="text-gray-400 dark:text-gray-500">{{
                t("keys.selectGroup")
              }}</span>
            </template>

            <template #option="{ option, selected }">
              <div
                class="flex w-full min-w-0 items-start justify-between gap-3 py-0.5"
              >
                <div class="flex min-w-0 flex-1 items-start gap-2.5">
                  <span
                    :class="[
                      'mt-0.5 inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md',
                      getGroupProviderIconClass(getGroupOptionPlatform(option)),
                    ]"
                  >
                    <PlatformIcon
                      :platform="getGroupOptionPlatform(option)"
                      size="sm"
                    />
                  </span>
                  <span class="min-w-0 flex-1 text-left">
                    <span class="flex items-center gap-2">
                      <span
                        class="truncate text-sm font-semibold text-gray-900 dark:text-white"
                        >{{ option.label }}</span
                      >
                      <Icon
                        v-if="selected"
                        name="check"
                        size="sm"
                        class="shrink-0 text-primary-500"
                      />
                    </span>
                    <span
                      v-if="option.description"
                      class="mt-1 block truncate text-xs text-gray-500 dark:text-gray-400"
                    >
                      {{ option.description }}
                    </span>
                  </span>
                </div>
                <span
                  :class="[
                    'mt-0.5 shrink-0 rounded-full px-2.5 py-1 text-xs font-semibold',
                    getGroupRateBadgeClass(option),
                  ]"
                >
                  {{ formatGroupRate(option) }}x
                </span>
              </div>
            </template>
          </Select>
        </div>

        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{
                t("admin.accounts.modelRestriction")
              }}</label>
              <p class="mt-1 text-xs text-gray-500">
                {{ t("admin.accounts.selectAllowedModels") }}
              </p>
            </div>
            <Toggle v-model="formData.enable_model_restriction" :aria-label="t('admin.accounts.modelRestriction')" />
          </div>
          <div
            v-if="formData.enable_model_restriction"
            class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-800"
          >
            <ModelWhitelistSelector
              v-model="formData.allowed_models"
              :platforms="selectedGroupPlatforms"
            />
            <p class="mt-2 text-xs text-gray-500">
              {{
                formData.allowed_models.length > 0
                  ? t("admin.accounts.selectedModels", {
                      count: formData.allowed_models.length,
                    })
                  : t("admin.accounts.supportsAllModels")
              }}
            </p>
          </div>
        </div>

        <!-- Custom Key Section (only for create) -->
        <div v-if="!showEditModal" class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{
              t("keys.customKeyLabel")
            }}</label>
            <Toggle v-model="formData.use_custom_key" :aria-label="t('keys.customKeyLabel')" />
          </div>
          <div v-if="formData.use_custom_key">
            <input
              v-model="formData.custom_key"
              type="text"
              class="input font-mono"
              :placeholder="t('keys.customKeyPlaceholder')"
              :class="{ 'border-red-500 dark:border-red-500': customKeyError }"
            />
            <p v-if="customKeyError" class="mt-1 text-sm text-red-500">
              {{ customKeyError }}
            </p>
            <p v-else class="input-hint">{{ t("keys.customKeyHint") }}</p>
          </div>
        </div>

        <div v-if="showEditModal">
          <label class="input-label">{{ t("keys.statusLabel") }}</label>
          <Select
            v-model="formData.status"
            :options="statusOptions"
            :placeholder="t('keys.selectStatus')"
          />
        </div>

        <!-- IP Restriction Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{
              t("keys.ipRestriction")
            }}</label>
            <Toggle v-model="formData.enable_ip_restriction" :aria-label="t('keys.ipRestriction')" />
          </div>

          <div v-if="formData.enable_ip_restriction" class="space-y-4 pt-2">
            <div>
              <label class="input-label">{{ t("keys.ipWhitelist") }}</label>
              <textarea
                v-model="formData.ip_whitelist"
                rows="3"
                class="input font-mono text-sm"
                :placeholder="t('keys.ipWhitelistPlaceholder')"
              />
              <p class="input-hint">{{ t("keys.ipWhitelistHint") }}</p>
            </div>

            <div>
              <label class="input-label">{{ t("keys.ipBlacklist") }}</label>
              <textarea
                v-model="formData.ip_blacklist"
                rows="3"
                class="input font-mono text-sm"
                :placeholder="t('keys.ipBlacklistPlaceholder')"
              />
              <p class="input-hint">{{ t("keys.ipBlacklistHint") }}</p>
            </div>
          </div>
        </div>

        <!-- Quota Limit Section -->
        <div class="space-y-3">
          <label class="input-label">{{ t("keys.quotaLimit") }}</label>

          <div class="space-y-4">
            <div>
              <div class="relative">
                <span
                  class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500"
                  >$</span
                >
                <input
                  v-model.number="formData.quota"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-7"
                  :placeholder="t('keys.quotaAmountPlaceholder')"
                />
              </div>
              <p class="input-hint">{{ t("keys.quotaAmountHint") }}</p>
            </div>

            <!-- Quota used display (only in edit mode) -->
            <div v-if="showEditModal && selectedKey && selectedKey.quota > 0">
              <label class="input-label">{{ t("keys.quotaUsed") }}</label>
              <div class="flex items-center gap-2">
                <div
                  class="flex-1 rounded-lg bg-gray-100 px-3 py-2 dark:bg-dark-700"
                >
                  <span
                    class="font-medium text-gray-900 dark:text-white"
                    :title="formatCurrencyTitle(selectedKey.quota_used || 0)"
                  >
                    {{ formatCurrency(selectedKey.quota_used || 0) }}
                  </span>
                  <span class="mx-2 text-gray-400">/</span>
                  <span class="text-gray-500 dark:text-gray-400">
                    {{ formatCurrency(selectedKey.quota || 0) }}
                  </span>
                </div>
                <LiquidButton
                  type="button"
                  @click="confirmResetQuota"
                  class="text-sm"
                  :title="t('keys.resetQuotaUsed')"
                  variant="outline"
                  size="sm"
                >
                  {{ t("keys.reset") }}
                </LiquidButton>
              </div>
            </div>
          </div>
        </div>

        <!-- Rate Limit Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{
              t("keys.rateLimitSection")
            }}</label>
            <Toggle v-model="formData.enable_rate_limit" :aria-label="t('keys.rateLimitSection')" />
          </div>

          <div v-if="formData.enable_rate_limit" class="space-y-4 pt-2">
            <p class="input-hint -mt-2">{{ t("keys.rateLimitHint") }}</p>
            <!-- 5-Hour Limit -->
            <div>
              <label class="input-label">{{ t("keys.rateLimit5h") }}</label>
              <div class="relative">
                <span
                  class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500"
                  >$</span
                >
                <input
                  v-model.number="formData.rate_limit_5h"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-7"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div
                v-if="
                  showEditModal && selectedKey && selectedKey.rate_limit_5h > 0
                "
                class="mt-2"
              >
                <div class="flex items-center gap-2">
                  <div
                    class="flex-1 rounded-lg bg-gray-100 px-3 py-2 dark:bg-dark-700 text-sm"
                  >
                    <span
                      :class="[
                        'font-medium',
                        selectedKey.usage_5h >= selectedKey.rate_limit_5h
                          ? 'text-red-500'
                          : selectedKey.usage_5h >=
                              selectedKey.rate_limit_5h * 0.8
                            ? 'text-yellow-500'
                            : 'text-gray-900 dark:text-white',
                      ]"
                      :title="formatCurrencyTitle(selectedKey.usage_5h || 0)"
                    >
                      {{ formatCurrency(selectedKey.usage_5h || 0) }}
                    </span>
                    <span class="mx-2 text-gray-400">/</span>
                    <span class="text-gray-500 dark:text-gray-400">
                      {{ formatCurrency(selectedKey.rate_limit_5h || 0) }}
                    </span>
                  </div>
                </div>
                <div
                  class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600"
                >
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h
                        ? 'bg-red-500'
                        : selectedKey.usage_5h >=
                            selectedKey.rate_limit_5h * 0.8
                          ? 'bg-yellow-500'
                          : 'bg-primary-500',
                    ]"
                    :style="{
                      width:
                        Math.min(
                          (selectedKey.usage_5h / selectedKey.rate_limit_5h) *
                            100,
                          100,
                        ) + '%',
                    }"
                  />
                </div>
              </div>
            </div>

            <!-- Daily Limit -->
            <div>
              <label class="input-label">{{ t("keys.rateLimit1d") }}</label>
              <div class="relative">
                <span
                  class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500"
                  >$</span
                >
                <input
                  v-model.number="formData.rate_limit_1d"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-7"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div
                v-if="
                  showEditModal && selectedKey && selectedKey.rate_limit_1d > 0
                "
                class="mt-2"
              >
                <div class="flex items-center gap-2">
                  <div
                    class="flex-1 rounded-lg bg-gray-100 px-3 py-2 dark:bg-dark-700 text-sm"
                  >
                    <span
                      :class="[
                        'font-medium',
                        selectedKey.usage_1d >= selectedKey.rate_limit_1d
                          ? 'text-red-500'
                          : selectedKey.usage_1d >=
                              selectedKey.rate_limit_1d * 0.8
                            ? 'text-yellow-500'
                            : 'text-gray-900 dark:text-white',
                      ]"
                      :title="formatCurrencyTitle(selectedKey.usage_1d || 0)"
                    >
                      {{ formatCurrency(selectedKey.usage_1d || 0) }}
                    </span>
                    <span class="mx-2 text-gray-400">/</span>
                    <span class="text-gray-500 dark:text-gray-400">
                      {{ formatCurrency(selectedKey.rate_limit_1d || 0) }}
                    </span>
                  </div>
                </div>
                <div
                  class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600"
                >
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d
                        ? 'bg-red-500'
                        : selectedKey.usage_1d >=
                            selectedKey.rate_limit_1d * 0.8
                          ? 'bg-yellow-500'
                          : 'bg-primary-500',
                    ]"
                    :style="{
                      width:
                        Math.min(
                          (selectedKey.usage_1d / selectedKey.rate_limit_1d) *
                            100,
                          100,
                        ) + '%',
                    }"
                  />
                </div>
              </div>
            </div>

            <!-- 7-Day Limit -->
            <div>
              <label class="input-label">{{ t("keys.rateLimit7d") }}</label>
              <div class="relative">
                <span
                  class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500"
                  >$</span
                >
                <input
                  v-model.number="formData.rate_limit_7d"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-7"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div
                v-if="
                  showEditModal && selectedKey && selectedKey.rate_limit_7d > 0
                "
                class="mt-2"
              >
                <div class="flex items-center gap-2">
                  <div
                    class="flex-1 rounded-lg bg-gray-100 px-3 py-2 dark:bg-dark-700 text-sm"
                  >
                    <span
                      :class="[
                        'font-medium',
                        selectedKey.usage_7d >= selectedKey.rate_limit_7d
                          ? 'text-red-500'
                          : selectedKey.usage_7d >=
                              selectedKey.rate_limit_7d * 0.8
                            ? 'text-yellow-500'
                            : 'text-gray-900 dark:text-white',
                      ]"
                      :title="formatCurrencyTitle(selectedKey.usage_7d || 0)"
                    >
                      {{ formatCurrency(selectedKey.usage_7d || 0) }}
                    </span>
                    <span class="mx-2 text-gray-400">/</span>
                    <span class="text-gray-500 dark:text-gray-400">
                      {{ formatCurrency(selectedKey.rate_limit_7d || 0) }}
                    </span>
                  </div>
                </div>
                <div
                  class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600"
                >
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d
                        ? 'bg-red-500'
                        : selectedKey.usage_7d >=
                            selectedKey.rate_limit_7d * 0.8
                          ? 'bg-yellow-500'
                          : 'bg-primary-500',
                    ]"
                    :style="{
                      width:
                        Math.min(
                          (selectedKey.usage_7d / selectedKey.rate_limit_7d) *
                            100,
                          100,
                        ) + '%',
                    }"
                  />
                </div>
              </div>
            </div>

            <!-- Reset Rate Limit button (edit mode only) -->
            <div
              v-if="
                showEditModal &&
                selectedKey &&
                (selectedKey.rate_limit_5h > 0 ||
                  selectedKey.rate_limit_1d > 0 ||
                  selectedKey.rate_limit_7d > 0)
              "
            >
              <LiquidButton
                type="button"
                @click="confirmResetRateLimit"
                class="text-sm"
                variant="outline"
                size="sm"
              >
                {{ t("keys.resetRateLimitUsage") }}
              </LiquidButton>
            </div>
          </div>
        </div>

        <!-- Expiration Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t("keys.expiration") }}</label>
            <Toggle v-model="formData.enable_expiration" :aria-label="t('keys.expiration')" />
          </div>

          <div v-if="formData.enable_expiration" class="space-y-4 pt-2">
            <!-- Quick select buttons (for both create and edit mode) -->
            <div class="flex flex-wrap gap-2">
              <LiquidButton
                v-for="days in ['7', '30', '90']"
                :key="days"
                type="button"
                @click="setExpirationDays(parseInt(days))"
                :class="[
                  'rounded-lg px-3 py-1.5 text-sm transition-colors',
                  formData.expiration_preset === days
                    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-400 dark:hover:bg-dark-600',
                ]"
                variant="plain"
                size="sm"
              >
                {{
                  showEditModal
                    ? t("keys.extendDays", { days })
                    : t("keys.expiresInDays", { days })
                }}
              </LiquidButton>
              <LiquidButton
                type="button"
                @click="formData.expiration_preset = 'custom'"
                :class="[
                  'rounded-lg px-3 py-1.5 text-sm transition-colors',
                  formData.expiration_preset === 'custom'
                    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-400 dark:hover:bg-dark-600',
                ]"
                variant="plain"
                size="sm"
              >
                {{ t("keys.customDate") }}
              </LiquidButton>
            </div>

            <!-- Date picker (always show for precise adjustment) -->
            <div>
              <label class="input-label">{{ t("keys.expirationDate") }}</label>
              <input
                v-model="formData.expiration_date"
                type="datetime-local"
                class="input"
              />
              <p class="input-hint">{{ t("keys.expirationDateHint") }}</p>
            </div>

            <!-- Current expiration display (only in edit mode) -->
            <div
              v-if="showEditModal && selectedKey?.expires_at"
              class="text-sm"
            >
              <span class="text-gray-500 dark:text-gray-400"
                >{{ t("keys.currentExpiration") }}:
              </span>
              <span class="font-medium text-gray-900 dark:text-white">
                {{ formatDateTime(selectedKey.expires_at) }}
              </span>
            </div>
          </div>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <LiquidButton
            @click="closeModals"
            type="button"
            variant="outline"
            size="sm"
          >
            {{ t("common.cancel") }}
          </LiquidButton>
          <LiquidButton
            form="key-form"
            type="submit"
            :disabled="submitting || formData.group_ids.length === 0"
            data-tour="key-form-submit"
            size="default"
          >
            <svg
              v-if="submitting"
              class="-ml-1 h-4 w-4 animate-spin"
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
            {{
              submitting
                ? t("keys.saving")
                : showEditModal
                  ? t("common.update")
                  : t("common.create")
            }}
          </LiquidButton>
        </div>
      </template>
    </BaseDialog>

    <!-- Created Key One-Time Reveal Dialog -->
    <BaseDialog
      :show="!!createdKeyToReveal"
      :title="t('keys.createdKeyReveal.title')"
      width="wide"
      :close-on-escape="false"
      :close-on-click-outside="false"
      @close="acknowledgeCreatedKey"
    >
      <div
        v-if="createdKeyToReveal"
        class="space-y-4"
        data-testid="created-key-reveal"
      >
        <div
          class="rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-100"
        >
          <p class="font-semibold">
            {{ t("keys.createdKeyReveal.warningTitle") }}
          </p>
          <p class="mt-1 leading-6">
            {{ t("keys.createdKeyReveal.warningDescription") }}
          </p>
        </div>

        <div>
          <label class="input-label">{{
            t("keys.createdKeyReveal.apiKeyLabel")
          }}</label>
          <div class="flex gap-2">
            <input
              :value="createdKeyToReveal.key"
              readonly
              class="input font-mono text-sm"
              data-testid="created-key-value"
            />
            <LiquidButton
              type="button"
              class="shrink-0"
              data-testid="created-key-copy"
              @click="copyCreatedKey"
              variant="outline"
              size="sm"
            >
              {{
                createdKeyCopied
                  ? t("keys.createdKeyReveal.copied")
                  : t("keys.createdKeyReveal.copyFullKey")
              }}
            </LiquidButton>
          </div>
        </div>

        <div
          class="rounded-xl border border-gray-200 bg-gray-50 p-4 text-sm text-gray-700 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300"
        >
          <p class="font-medium text-gray-900 dark:text-white">
            {{ t("keys.createdKeyReveal.connectionTitle") }}
          </p>
          <div class="mt-2 flex flex-wrap items-center gap-2">
            <span class="text-gray-500 dark:text-gray-400">{{
              t("keys.workbenchGuide.baseUrlLabel")
            }}</span>
            <code
              class="rounded-lg bg-white px-2 py-1 text-xs dark:bg-dark-900"
              >{{ openAICompatibleBaseUrl }}</code
            >
            <LiquidButton
              type="button"
              class="rounded-lg p-1 text-gray-500 transition-colors hover:bg-white hover:text-gray-700 dark:hover:bg-dark-900 dark:hover:text-gray-200"
              data-testid="created-key-base-url-copy"
              :title="
                baseUrlCopied
                  ? t('keys.workbenchGuide.baseUrlCopied')
                  : t('keys.workbenchGuide.copyBaseUrl')
              "
              @click="copyBaseUrl"
              variant="plain"
              size="icon"
            >
              <Icon
                v-if="baseUrlCopied"
                name="check"
                size="sm"
                :stroke-width="2"
              />
              <Icon v-else name="clipboard" size="sm" />
            </LiquidButton>
          </div>
          <!-- P2-15: expose root URL (no /v1) for tools that append the path themselves -->
          <div class="mt-1 flex flex-wrap items-center gap-2">
            <span class="text-gray-500 dark:text-gray-400">{{
              t("keys.workbenchGuide.baseUrlRootLabel")
            }}</span>
            <code
              class="rounded-lg bg-white px-2 py-1 text-xs dark:bg-dark-900"
              >{{ apiBaseRoot }}</code
            >
            <LiquidButton
              type="button"
              class="rounded-lg p-1 text-gray-500 transition-colors hover:bg-white hover:text-gray-700 dark:hover:bg-dark-900 dark:hover:text-gray-200"
              data-testid="created-key-base-url-root-copy"
              :title="
                baseUrlRootCopied
                  ? t('keys.workbenchGuide.baseUrlRootCopied')
                  : t('keys.workbenchGuide.copyBaseUrlRoot')
              "
              @click="copyBaseUrlRoot"
              variant="plain"
              size="icon"
            >
              <Icon
                v-if="baseUrlRootCopied"
                name="check"
                size="sm"
                :stroke-width="2"
              />
              <Icon v-else name="clipboard" size="sm" />
            </LiquidButton>
          </div>
          <div class="mt-2 flex flex-wrap items-start gap-2">
            <span class="text-gray-500 dark:text-gray-400">{{
              t("keys.createdKeyReveal.modelLabel")
            }}</span>
            <span class="leading-6">{{
              t("keys.createdKeyReveal.modelHint")
            }}</span>
          </div>
          <p class="mt-2 leading-6" data-testid="created-key-readiness-hint">
            {{ t("keys.createdKeyReveal.readinessHint") }}
          </p>
          <p class="mt-2 leading-6">
            {{ t("keys.createdKeyReveal.connectionDescription") }}
          </p>
          <p
            v-if="!publicSettings?.hide_ccs_import_button"
            class="mt-2 text-xs font-semibold text-primary-700 dark:text-primary-300"
          >
            {{ t("keys.createdKeyReveal.primaryActionHint") }}
          </p>
        </div>

        <div
          class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-800"
          data-testid="created-key-quick-start"
        >
          <p class="text-sm font-medium text-gray-900 dark:text-white">
            {{ t("keys.createdKeyReveal.quickStart.title") }}
          </p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t("keys.createdKeyReveal.quickStart.description") }}
          </p>

          <div class="mt-2 border-b border-gray-200 dark:border-dark-600">
            <nav
              class="-mb-px flex space-x-4"
              aria-label="Quick start examples"
            >
              <LiquidButton
                v-for="tab in createdKeyExampleTabs"
                :key="tab.id"
                type="button"
                :data-testid="`created-key-example-tab-${tab.id}`"
                :class="[
                  'whitespace-nowrap border-b-2 px-1 py-2 text-sm font-medium transition-colors',
                  createdKeyExampleTab === tab.id
                    ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                    : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300',
                ]"
                @click="createdKeyExampleTab = tab.id"
                variant="plain"
                size="sm"
              >
                {{ tab.label }}
              </LiquidButton>
            </nav>
          </div>

          <div class="mt-3 flex justify-end">
            <LiquidButton
              type="button"
              class="shrink-0"
              data-testid="created-key-example-copy"
              @click="copyCreatedKeyExample"
              variant="outline"
              size="sm"
            >
              <Icon
                v-if="createdKeyExampleCopied"
                name="check"
                size="sm"
                :stroke-width="2"
              />
              <Icon v-else name="clipboard" size="sm" />
              {{
                createdKeyExampleCopied
                  ? t("keys.createdKeyReveal.quickStart.copied")
                  : t("keys.createdKeyReveal.quickStart.copyAll")
              }}
            </LiquidButton>
          </div>
          <pre
            class="mt-2 max-h-56 overflow-auto whitespace-pre rounded-lg bg-gray-900 p-4 font-mono text-xs leading-5 text-gray-100 dark:bg-dark-950"
            data-testid="created-key-example-code"
            >{{ activeCreatedKeyExample }}</pre
          >
        </div>
      </div>

      <template #footer>
        <div class="flex justify-end gap-3">
          <LiquidButton
            type="button"
            data-testid="created-key-ack"
            @click="acknowledgeCreatedKey"
            variant="outline"
            size="sm"
          >
            {{ t("keys.createdKeyReveal.acknowledge") }}
          </LiquidButton>
          <LiquidButton
            v-if="createdKeyToReveal && !publicSettings?.hide_ccs_import_button"
            type="button"
            data-testid="created-key-ccs-import"
            @click="importToCcswitch(createdKeyToReveal)"
            size="sm"
          >
            {{ t("keys.importToCcSwitch") }}
          </LiquidButton>
        </div>
      </template>
    </BaseDialog>

    <!-- Delete Confirmation Dialog -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('keys.deleteKey')"
      :message="t('keys.deleteConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="handleDelete"
      @cancel="showDeleteDialog = false"
    />

    <!-- Status Change Confirmation Dialog -->
    <ConfirmDialog
      :show="showStatusDialog"
      :title="
        pendingStatusAction?.status === 'active'
          ? t('keys.enableKeyTitle')
          : t('keys.disableKeyTitle')
      "
      :message="
        pendingStatusAction?.status === 'active'
          ? t('keys.enableConfirmMessage', {
              name: pendingStatusAction?.key.name,
            })
          : t('keys.disableConfirmMessage', {
              name: pendingStatusAction?.key.name,
            })
      "
      :confirm-text="
        pendingStatusAction?.status === 'active'
          ? t('keys.enable')
          : t('keys.disable')
      "
      :cancel-text="t('common.cancel')"
      :danger="pendingStatusAction?.status === 'inactive'"
      @confirm="handleToggleKeyStatus"
      @cancel="cancelToggleKeyStatus"
    />

    <!-- Reset Quota Confirmation Dialog -->
    <ConfirmDialog
      :show="showResetQuotaDialog"
      :title="t('keys.resetQuotaTitle')"
      :message="
        t('keys.resetQuotaConfirmMessage', {
          name: selectedKey?.name,
          used: selectedKey?.quota_used?.toFixed(4),
        })
      "
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetQuotaUsed"
      @cancel="showResetQuotaDialog = false"
    />

    <!-- Reset Rate Limit Confirmation Dialog -->
    <ConfirmDialog
      :show="showResetRateLimitDialog"
      :title="t('keys.resetRateLimitTitle')"
      :message="
        t('keys.resetRateLimitConfirmMessage', { name: selectedKey?.name })
      "
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetRateLimitUsage"
      @cancel="showResetRateLimitDialog = false"
    />

    <!-- Use Key Modal -->
    <UseKeyModal
      :show="showUseKeyModal"
      :api-key="useKeyPlaintext"
      :base-url="apiBaseUrl"
      :platform="selectedKey?.group?.platform || null"
      :allowed-models="selectedKey?.allowed_models || []"
      :allow-messages-dispatch="
        selectedKey?.group?.allow_messages_dispatch || false
      "
      :key-status="selectedKey?.status"
      @close="closeUseKeyModal"
    />

    <!-- CCS Client Selection Dialog for Antigravity -->
    <BaseDialog
      :show="showCcsClientSelect"
      :title="t('keys.ccsClientSelect.title')"
      width="narrow"
      @close="closeCcsClientSelect"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-400">
          {{ t("keys.ccsClientSelect.description") }}
        </p>
        <div class="grid grid-cols-2 gap-3">
          <LiquidButton
            @click="handleCcsClientSelect('claude')"
            class="flex flex-col items-center gap-2 p-4 rounded-xl border-2 border-gray-200 dark:border-dark-600 hover:border-primary-500 dark:hover:border-primary-500 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-all"
            variant="plain"
            size="sm"
          >
            <Icon
              name="terminal"
              size="xl"
              class="text-gray-600 dark:text-gray-400"
            />
            <span class="font-medium text-gray-900 dark:text-white">{{
              t("keys.ccsClientSelect.claudeCode")
            }}</span>
            <span class="text-xs text-gray-500 dark:text-gray-400">{{
              t("keys.ccsClientSelect.claudeCodeDesc")
            }}</span>
          </LiquidButton>
          <LiquidButton
            @click="handleCcsClientSelect('gemini')"
            class="flex flex-col items-center gap-2 p-4 rounded-xl border-2 border-gray-200 dark:border-dark-600 hover:border-primary-500 dark:hover:border-primary-500 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-all"
            variant="plain"
            size="sm"
          >
            <Icon
              name="sparkles"
              size="xl"
              class="text-gray-600 dark:text-gray-400"
            />
            <span class="font-medium text-gray-900 dark:text-white">{{
              t("keys.ccsClientSelect.geminiCli")
            }}</span>
            <span class="text-xs text-gray-500 dark:text-gray-400">{{
              t("keys.ccsClientSelect.geminiCliDesc")
            }}</span>
          </LiquidButton>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end">
          <LiquidButton
            @click="closeCcsClientSelect"
            variant="outline"
            size="sm"
          >
            {{ t("common.cancel") }}
          </LiquidButton>
        </div>
      </template>
    </BaseDialog>

    <!-- Group Selector Dropdown (Teleported to body to avoid overflow clipping) -->
    <Teleport to="body">
      <div
        v-if="groupSelectorKeyId !== null && dropdownPosition"
        ref="dropdownRef"
        class="keys-group-dropdown animate-in fade-in slide-in-from-top-2 fixed z-[100000020] w-max min-w-[380px] overflow-hidden rounded-xl shadow-lg ring-1 ring-black/5 duration-200 dark:ring-white/10"
        style="pointer-events: auto !important"
        :style="{
          top:
            dropdownPosition.top !== undefined
              ? dropdownPosition.top + 'px'
              : undefined,
          bottom:
            dropdownPosition.bottom !== undefined
              ? dropdownPosition.bottom + 'px'
              : undefined,
          left: dropdownPosition.left + 'px',
        }"
      >
        <!-- Search box -->
        <div class="border-b border-gray-100 p-2 dark:border-dark-700">
          <div class="relative">
            <svg
              class="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
              />
            </svg>
            <input
              v-model="groupSearchQuery"
              type="text"
              class="w-full rounded-lg border border-gray-200 bg-gray-50 py-1.5 pl-8 pr-3 text-sm text-gray-900 placeholder-gray-400 outline-none focus:border-primary-300 focus:ring-1 focus:ring-primary-300 dark:border-dark-600 dark:bg-dark-700 dark:text-white dark:placeholder-gray-500 dark:focus:border-primary-600 dark:focus:ring-primary-600"
              :placeholder="t('keys.searchGroup')"
              @click.stop
            />
          </div>
        </div>
        <!-- Group list -->
        <div class="max-h-80 overflow-y-auto p-1.5">
          <LiquidButton
            v-for="option in filteredGroupOptions"
            :key="option.value ?? 'null'"
            @click="changeGroup(selectedKeyForGroup!, option.value)"
            :class="[
              'flex w-full items-center justify-between rounded-lg px-3 py-2.5 text-sm transition-colors',
              'border-b border-gray-100 last:border-0 dark:border-dark-700',
              selectedKeyForGroup?.group_id === option.value ||
              (!selectedKeyForGroup?.group_id && option.value === null)
                ? 'bg-primary-50 dark:bg-primary-900/20'
                : 'hover:bg-gray-100 dark:hover:bg-dark-700',
            ]"
            :title="option.description || undefined"
            variant="plain"
            size="icon"
          >
            <GroupOptionItem
              :name="option.label"
              :platform="option.platform"
              :subscription-type="option.subscriptionType"
              :description="option.description"
              :selected="
                selectedKeyForGroup?.group_id === option.value ||
                (!selectedKeyForGroup?.group_id && option.value === null)
              "
            />
          </LiquidButton>
          <!-- Empty state when search has no results -->
          <div
            v-if="filteredGroupOptions.length === 0"
            class="py-4 text-center text-sm text-gray-400 dark:text-gray-500"
          >
            {{ t("keys.noGroupFound") }}
          </div>
        </div>
      </div>
    </Teleport>
  </component>
</template>

<script setup lang="ts">
import LiquidButton from "@/components/common/LiquidButton.vue";
import Toggle from "@/components/common/Toggle.vue";
import {
  ref,
  reactive,
  computed,
  onMounted,
  onUnmounted,
  type ComponentPublicInstance,
} from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { useAppStore } from "@/stores/app";
import { useOnboardingStore } from "@/stores/onboarding";
import { useClipboard } from "@/composables/useClipboard";
import { getPersistedPageSize } from "@/composables/usePersistedPageSize";

const { t } = useI18n();
import { keysAPI, authAPI, usageAPI, userGroupsAPI, userChannelsAPI } from "@/api";
import AppLayout from "@/components/layout/AppLayout.vue";
import AppSectionShell from "@/components/user/AppSectionShell.vue";
import BalanceWarningBanner from "@/components/user/BalanceWarningBanner.vue";
import TablePageLayout from "@/components/layout/TablePageLayout.vue";
import DataTable from "@/components/common/DataTable.vue";
import Pagination from "@/components/common/Pagination.vue";
import BaseDialog from "@/components/common/BaseDialog.vue";
import ConfirmDialog from "@/components/common/ConfirmDialog.vue";
import EmptyState from "@/components/common/EmptyState.vue";
import Select from "@/components/common/Select.vue";
import SearchInput from "@/components/common/SearchInput.vue";
import Icon from "@/components/icons/Icon.vue";
import UseKeyModal from "@/components/keys/UseKeyModal.vue";
import GroupBadge from "@/components/common/GroupBadge.vue";
import GroupOptionItem from "@/components/common/GroupOptionItem.vue";
import PlatformIcon from "@/components/common/PlatformIcon.vue";
import ModelWhitelistSelector from "@/components/account/ModelWhitelistSelector.vue";
import type { ApiKey, Group, PublicSettings } from "@/types";
import type { UserAvailableGroup } from "@/api/channels";
import type { Column } from "@/components/common/types";
import type { BatchApiKeyUsageStats } from "@/api/usage";
import {
  formatCurrency,
  formatCurrencyTitle,
  formatDateTime,
} from "@/utils/format";
import { DEFAULT_SITE_NAME, normalizeSiteName } from "@/utils/brand";

// Helper to format date for datetime-local input
const formatDateTimeLocal = (isoDate: string): string => {
  const date = new Date(isoDate);
  const pad = (n: number) => n.toString().padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
};

const appStore = useAppStore();
const onboardingStore = useOnboardingStore();
const { copyToClipboard: clipboardCopy } = useClipboard();
const route = useRoute();
const useWorkbenchShell = computed(() => route.path.startsWith("/app/"));
const pageShell = computed(() =>
  useWorkbenchShell.value ? AppSectionShell : AppLayout,
);
const pageShellProps = computed(() =>
  useWorkbenchShell.value
    ? {
        title: "API Key",
        subtitle: "创建和管理用于客户端接入的 API Key。",
        eyebrow: "",
        icon: "key",
      }
    : {},
);

const allColumns = computed<Column[]>(() => [
  { key: "name", label: t("common.name"), sortable: true },
  { key: "key", label: t("keys.apiKey"), sortable: false },
  { key: "group", label: t("keys.group"), sortable: false },
  { key: "usage", label: t("keys.usage"), sortable: false },
  { key: "rate_limit", label: t("keys.rateLimitColumn"), sortable: false },
  { key: "status", label: t("common.status"), sortable: true },
  { key: "expires_at", label: t("keys.expiresAt"), sortable: true },
  { key: "last_used_at", label: t("keys.lastUsedAt"), sortable: true },
  { key: "created_at", label: t("keys.created"), sortable: true },
  {
    key: "actions",
    label: t("common.actions"),
    sortable: false,
    class: "keys-actions-column",
  },
]);

const DEFAULT_HIDDEN_COLUMNS = ["rate_limit", "last_used_at"];
const HIDDEN_COLUMNS_STORAGE_KEY = "ssxz-api-key-hidden-columns-v1";
const hiddenColumns = reactive<Set<string>>(new Set());

const toggleableColumns = computed(() =>
  allColumns.value.filter(
    (column) => !["name", "key", "actions"].includes(column.key),
  ),
);

const columns = computed<Column[]>(() =>
  allColumns.value.filter((column) => !hiddenColumns.has(column.key)),
);

const loadSavedColumns = () => {
  hiddenColumns.clear();
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_STORAGE_KEY);
    const columnKeys = new Set(allColumns.value.map((column) => column.key));
    const parsed = saved ? JSON.parse(saved) : DEFAULT_HIDDEN_COLUMNS;
    if (!Array.isArray(parsed))
      throw new Error("Invalid API key column settings");
    parsed
      .filter(
        (key): key is string => typeof key === "string" && columnKeys.has(key),
      )
      .forEach((key) => hiddenColumns.add(key));
  } catch {
    DEFAULT_HIDDEN_COLUMNS.forEach((key) => hiddenColumns.add(key));
  }
};

const saveColumns = () => {
  localStorage.setItem(
    HIDDEN_COLUMNS_STORAGE_KEY,
    JSON.stringify([...hiddenColumns]),
  );
};

const toggleColumn = (key: string) => {
  if (hiddenColumns.has(key)) hiddenColumns.delete(key);
  else hiddenColumns.add(key);
  saveColumns();
};

const isColumnVisible = (key: string) => !hiddenColumns.has(key);

const apiKeys = ref<ApiKey[]>([]);
const groups = ref<Group[]>([]);
const loading = ref(false);
const submitting = ref(false);
const now = ref(new Date());
let resetTimer: ReturnType<typeof setInterval> | null = null;
let createdKeyCopyTimer: ReturnType<typeof setTimeout> | null = null;
let baseUrlCopyTimer: ReturnType<typeof setTimeout> | null = null;
let baseUrlRootCopyTimer: ReturnType<typeof setTimeout> | null = null;
let createdKeyExampleCopyTimer: ReturnType<typeof setTimeout> | null = null;
const usageStats = ref<Record<string, BatchApiKeyUsageStats>>({});
const userGroupRates = ref<Record<number, number>>({});

const pagination = ref({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0,
});

// Filter state
const filterSearch = ref("");
const filterStatus = ref("");
const filterGroupId = ref<string | number>("");

const showCreateModal = ref(false);
const showEditModal = ref(false);
const showDeleteDialog = ref(false);
const showStatusDialog = ref(false);
const showResetQuotaDialog = ref(false);
const showResetRateLimitDialog = ref(false);
const showUseKeyModal = ref(false);
const useKeyPlaintext = ref("");
const showCcsClientSelect = ref(false);
const pendingCcsRow = ref<ApiKey | null>(null);
const pendingStatusAction = ref<{
  key: ApiKey;
  status: "active" | "inactive";
} | null>(null);
const selectedKey = ref<ApiKey | null>(null);
const copiedKeyId = ref<number | null>(null);
const createdKeyToReveal = ref<ApiKey | null>(null);
const createdKeyCopied = ref(false);
const baseUrlCopied = ref(false);
const baseUrlRootCopied = ref(false);
type CreatedKeyExampleTabId = "curl" | "python" | "cherry";
const createdKeyExampleTab = ref<CreatedKeyExampleTabId>("curl");
const createdKeyExampleCopied = ref(false);
const statusUpdatingKeyId = ref<number | null>(null);
const groupSelectorKeyId = ref<number | null>(null);
const publicSettings = ref<PublicSettings | null>(null);
const dropdownRef = ref<HTMLElement | null>(null);
const columnDropdownRef = ref<HTMLElement | null>(null);
const showColumnDropdown = ref(false);
const dropdownPosition = ref<{
  top?: number;
  bottom?: number;
  left: number;
} | null>(null);
const groupButtonRefs = ref<Map<number, HTMLElement>>(new Map());
let abortController: AbortController | null = null;
const apiBaseUrl = computed(() =>
  resolveUserFacingApiBaseUrl(publicSettings.value?.api_base_url),
);
const openAICompatibleBaseUrl = computed(() => {
  const trimmed = apiBaseUrl.value.replace(/\/+$/, "");
  return trimmed.endsWith("/v1") ? trimmed : `${trimmed}/v1`;
});
// Raw base URL without /v1 — for tools that append the path themselves (Claude Code, Codex CLI)
const apiBaseRoot = computed(() => apiBaseUrl.value.replace(/\/+$/, ""));

const createdKeyExampleTabs: Array<{
  id: CreatedKeyExampleTabId;
  label: string;
}> = [
  { id: "curl", label: "curl" },
  { id: "python", label: "Python" },
  { id: "cherry", label: "Cherry Studio" },
];

const createdKeyExampleModel = computed(() => {
  const allowed = createdKeyToReveal.value?.allowed_models;
  const first = Array.isArray(allowed)
    ? allowed.find((model) => !!model?.trim())
    : undefined;
  return first?.trim() || "gpt-5.5";
});

const createdKeyCurlExample = computed(() => {
  const base = openAICompatibleBaseUrl.value;
  const key = createdKeyToReveal.value?.key ?? "";
  return [
    `curl -X POST "${base}/chat/completions" \\`,
    '  -H "Content-Type: application/json" \\',
    `  -H "Authorization: Bearer ${key}" \\`,
    "  -d '{",
    `    "model": "${createdKeyExampleModel.value}",`,
    '    "messages": [{"role": "user", "content": "Hello"}]',
    "  }'",
  ].join("\n");
});

const createdKeyPythonExample = computed(() => {
  const base = openAICompatibleBaseUrl.value;
  const key = createdKeyToReveal.value?.key ?? "";
  return [
    "from openai import OpenAI",
    "",
    "client = OpenAI(",
    `    base_url="${base}",`,
    `    api_key="${key}"`,
    ")",
    "",
    "response = client.chat.completions.create(",
    `    model="${createdKeyExampleModel.value}",`,
    '    messages=[{"role": "user", "content": "Hello"}]',
    ")",
    "print(response.choices[0].message.content)",
  ].join("\n");
});

const createdKeyCherryExample = computed(() => {
  const base = openAICompatibleBaseUrl.value;
  const key = createdKeyToReveal.value?.key ?? "";
  return [
    t("keys.createdKeyReveal.quickStart.cherryHeading"),
    t("keys.createdKeyReveal.quickStart.cherryProvider"),
    `${t("keys.createdKeyReveal.quickStart.cherryBaseUrl")}${base}`,
    `${t("keys.createdKeyReveal.quickStart.cherryApiKey")}${key}`,
    `${t("keys.createdKeyReveal.quickStart.cherryModel")}${createdKeyExampleModel.value}`,
  ].join("\n");
});

const activeCreatedKeyExample = computed(() => {
  switch (createdKeyExampleTab.value) {
    case "python":
      return createdKeyPythonExample.value;
    case "cherry":
      return createdKeyCherryExample.value;
    default:
      return createdKeyCurlExample.value;
  }
});

function resolveUserFacingApiBaseUrl(configuredUrl?: string | null) {
  const currentOrigin =
    typeof window !== "undefined" ? window.location.origin : "";
  const trimmed = configuredUrl?.trim();
  const shouldUseConfigured =
    trimmed &&
    !isLocalhostBaseUrl(trimmed) &&
    !isPrivateLoopbackBaseUrl(trimmed);

  return shouldUseConfigured ? trimmed : currentOrigin;
}

function isLocalhostBaseUrl(value: string) {
  try {
    const hostname = new URL(value).hostname.toLowerCase();
    return (
      hostname === "localhost" ||
      hostname === "127.0.0.1" ||
      hostname === "::1" ||
      hostname === "[::1]"
    );
  } catch {
    return /(^|\/\/)(localhost|127\.0\.0\.1)(?::|\/|$)/i.test(value);
  }
}

function isPrivateLoopbackBaseUrl(value: string) {
  try {
    const hostname = new URL(value).hostname.toLowerCase();
    return hostname.startsWith("127.") || hostname === "0.0.0.0";
  } catch {
    return false;
  }
}

// Get the currently selected key for group change
const selectedKeyForGroup = computed(() => {
  if (groupSelectorKeyId.value === null) return null;
  return apiKeys.value.find((k) => k.id === groupSelectorKeyId.value) || null;
});

const setGroupButtonRef = (
  keyId: number,
  el: Element | ComponentPublicInstance | null,
) => {
  if (el instanceof HTMLElement) {
    groupButtonRefs.value.set(keyId, el);
  } else {
    groupButtonRefs.value.delete(keyId);
  }
};

const formData = ref({
  name: "",
  group_ids: [] as number[],
  enable_model_restriction: false,
  allowed_models: [] as string[],
  status: "active" as "active" | "inactive",
  use_custom_key: false,
  custom_key: "",
  enable_ip_restriction: false,
  ip_whitelist: "",
  ip_blacklist: "",
  // Quota settings (empty = unlimited)
  enable_quota: false,
  quota: null as number | null,
  // Rate limit settings
  enable_rate_limit: false,
  rate_limit_5h: null as number | null,
  rate_limit_1d: null as number | null,
  rate_limit_7d: null as number | null,
  enable_expiration: false,
  expiration_preset: "30" as "7" | "30" | "90" | "custom",
  expiration_date: "",
});

const getDefaultCreateGroupIds = (): number[] => {
  const firstGroup = groups.value[0];
  return firstGroup ? [firstGroup.id] : [];
};

const resetFormData = (groupIds: number[] = []) => {
  formData.value = {
    name: "",
    group_ids: groupIds,
    enable_model_restriction: false,
    allowed_models: [],
    status: "active",
    use_custom_key: false,
    custom_key: "",
    enable_ip_restriction: false,
    ip_whitelist: "",
    ip_blacklist: "",
    enable_quota: false,
    quota: null,
    enable_rate_limit: false,
    rate_limit_5h: null,
    rate_limit_1d: null,
    rate_limit_7d: null,
    enable_expiration: false,
    expiration_preset: "30",
    expiration_date: "",
  };
};

const openCreateModal = () => {
  selectedKey.value = null;
  resetFormData(getDefaultCreateGroupIds());
  showCreateModal.value = true;
};

// 自定义Key验证
const customKeyError = computed(() => {
  if (!formData.value.use_custom_key || !formData.value.custom_key) {
    return "";
  }
  const key = formData.value.custom_key;
  if (key.length < 16) {
    return t("keys.customKeyTooShort");
  }
  // 检查字符：只允许字母、数字、下划线、连字符
  if (!/^[a-zA-Z0-9_-]+$/.test(key)) {
    return t("keys.customKeyInvalidChars");
  }
  return "";
});

const statusOptions = computed(() => [
  { value: "active", label: t("common.active") },
  { value: "inactive", label: t("common.inactive") },
]);

function formatGroupDisplayName(group: Pick<Group, "name">) {
  return group.name?.trim() || "";
}

function formatGroupDescription(group: Pick<Group, "description">) {
  return group.description?.trim() || "";
}

interface GroupSelectOption extends Record<string, unknown> {
  value: number;
  label: string;
  description: string;
  subscriptionType: Group["subscription_type"];
  platform: Group["platform"];
  rateMultiplier: number;
  userRateMultiplier: number | null;
}

function asGroupSelectOption(option: Record<string, unknown>) {
  return option as GroupSelectOption;
}

function getGroupOptionPlatform(option: Record<string, unknown>) {
  return asGroupSelectOption(option).platform;
}

function getEffectiveGroupRate(option: Record<string, unknown>) {
  const groupOption = asGroupSelectOption(option);
  return groupOption.userRateMultiplier ?? groupOption.rateMultiplier;
}

function formatGroupRate(option: Record<string, unknown>) {
  return Number(getEffectiveGroupRate(option).toFixed(4)).toString();
}

function getGroupRateBadgeClass(option: Record<string, unknown>) {
  const platform = getGroupOptionPlatform(option);
  if (platform === "openai") {
    return "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300";
  }
  if (platform === "anthropic") {
    return "bg-orange-50 text-orange-700 dark:bg-orange-950/40 dark:text-orange-300";
  }
  return "bg-gray-100 text-gray-700 dark:bg-dark-600 dark:text-gray-200";
}

function getGroupProviderIconClass(platform: Group["platform"]) {
  if (platform === "openai") {
    return "bg-emerald-50 text-emerald-600 dark:bg-emerald-950/50 dark:text-emerald-300";
  }
  if (platform === "anthropic") {
    return "bg-orange-50 text-orange-600 dark:bg-orange-950/50 dark:text-orange-300";
  }
  return "bg-gray-100 text-gray-600 dark:bg-dark-600 dark:text-gray-300";
}

// Filter dropdown options
const groupFilterOptions = computed(() => [
  { value: "", label: t("keys.allGroups") },
  { value: 0, label: t("keys.noGroup") },
  ...groups.value.map((g) => ({
    value: g.id,
    label: formatGroupDisplayName(g),
  })),
]);

const statusFilterOptions = computed(() => [
  { value: "", label: t("keys.allStatus") },
  { value: "active", label: t("keys.status.active") },
  { value: "inactive", label: t("keys.status.inactive") },
  { value: "quota_exhausted", label: t("keys.status.quota_exhausted") },
  { value: "expired", label: t("keys.status.expired") },
]);

const onFilterChange = () => {
  pagination.value.page = 1;
  loadApiKeys();
};

const onGroupFilterChange = (value: string | number | boolean | null) => {
  filterGroupId.value = value as string | number;
  onFilterChange();
};

const onStatusFilterChange = (value: string | number | boolean | null) => {
  filterStatus.value = value as string;
  onFilterChange();
};

// Convert groups to Select options format with rate multiplier and subscription type
const groupOptions = computed<GroupSelectOption[]>(() =>
  groups.value.map((group) => ({
    value: group.id,
    label: formatGroupDisplayName(group),
    description: formatGroupDescription(group),
    subscriptionType: group.subscription_type,
    platform: group.platform,
    rateMultiplier: group.rate_multiplier,
    userRateMultiplier: userGroupRates.value[group.id] ?? null,
  })),
);

const selectedFormGroupId = computed(() => formData.value.group_ids[0] ?? null);

const selectFormGroup = (value: string | number | boolean | null) => {
  formData.value.group_ids = typeof value === "number" ? [value] : [];
};

const selectedGroupPlatforms = computed(() => {
  const platforms = new Set<string>();
  for (const group of groups.value) {
    if (formData.value.group_ids.includes(group.id) && group.platform) {
      platforms.add(group.platform);
    }
  }
  return Array.from(platforms);
});

// Group dropdown search
const groupSearchQuery = ref("");
const filteredGroupOptions = computed(() => {
  const query = groupSearchQuery.value.trim().toLowerCase();
  if (!query) return groupOptions.value;
  return groupOptions.value.filter((opt) => {
    return (
      opt.label.toLowerCase().includes(query) ||
      (opt.description && opt.description.toLowerCase().includes(query))
    );
  });
});

const maskKey = (key: string): string => {
  if (key.length <= 12) return key;
  return `${key.slice(0, 8)}...${key.slice(-4)}`;
};

const isMaskedApiKey = (key?: string | null): boolean => {
  if (!key) return true;
  return key === "[redacted]" || key.includes("...");
};

type CcsImportPlatform = "openai" | "gemini" | "anthropic" | "antigravity";

const CCS_CLAUDE_MODELS = {
  fable: "claude-fable-5",
  haiku: "claude-3-5-haiku",
  sonnet: "claude-sonnet-5",
  opus: "claude-opus-4-8",
} as const;

const normalizeCcsImportPlatform = (
  value?: string | null,
): CcsImportPlatform | null => {
  const platform = value?.trim().toLowerCase();
  if (
    platform === "openai" ||
    platform === "gemini" ||
    platform === "anthropic" ||
    platform === "antigravity"
  ) {
    return platform;
  }
  return null;
};

const resolveCcsImportPlatform = (row: ApiKey): CcsImportPlatform | null => {
  const candidates = [
    normalizeCcsImportPlatform(row.group?.platform),
    ...(row.groups || []).map((group) =>
      normalizeCcsImportPlatform(group.platform),
    ),
  ].filter((platform): platform is CcsImportPlatform => Boolean(platform));
  const uniquePlatforms = Array.from(new Set(candidates));
  return uniquePlatforms.length === 1 ? uniquePlatforms[0] : null;
};

const resolvePlaintextApiKey = async (row: ApiKey): Promise<string | null> => {
  if (!isMaskedApiKey(row.key)) return row.key;

  try {
    const revealed = await keysAPI.reveal(row.id);
    if (!revealed.key || isMaskedApiKey(revealed.key)) {
      throw new Error("API key reveal returned no plaintext key");
    }
    return revealed.key;
  } catch {
    appStore.showError(t("keys.failedToReveal"));
    return null;
  }
};

const copyToClipboard = async (row: ApiKey) => {
  const plaintextKey = await resolvePlaintextApiKey(row);
  if (!plaintextKey) return;

  const success = await clipboardCopy(plaintextKey, t("keys.copied"));
  if (success) {
    copiedKeyId.value = row.id;
    setTimeout(() => {
      copiedKeyId.value = null;
    }, 800);
  }
};

const copyCreatedKey = async () => {
  const key = createdKeyToReveal.value?.key;
  if (!key) return;

  const success = await clipboardCopy(
    key,
    t("keys.createdKeyReveal.fullKeyCopied"),
  );
  if (success) {
    createdKeyCopied.value = true;
    if (createdKeyCopyTimer) clearTimeout(createdKeyCopyTimer);
    createdKeyCopyTimer = setTimeout(() => {
      createdKeyCopied.value = false;
      createdKeyCopyTimer = null;
    }, 1200);
  }
};

const copyBaseUrl = async () => {
  const success = await clipboardCopy(
    openAICompatibleBaseUrl.value,
    t("keys.workbenchGuide.baseUrlCopied"),
  );
  if (success) {
    baseUrlCopied.value = true;
    if (baseUrlCopyTimer) clearTimeout(baseUrlCopyTimer);
    baseUrlCopyTimer = setTimeout(() => {
      baseUrlCopied.value = false;
      baseUrlCopyTimer = null;
    }, 1200);
  }
};

const copyBaseUrlRoot = async () => {
  const success = await clipboardCopy(
    apiBaseRoot.value,
    t("keys.workbenchGuide.baseUrlRootCopied"),
  );
  if (success) {
    baseUrlRootCopied.value = true;
    if (baseUrlRootCopyTimer) clearTimeout(baseUrlRootCopyTimer);
    baseUrlRootCopyTimer = setTimeout(() => {
      baseUrlRootCopied.value = false;
      baseUrlRootCopyTimer = null;
    }, 1200);
  }
};

const copyCreatedKeyExample = async () => {
  const text = activeCreatedKeyExample.value;
  if (!createdKeyToReveal.value || !text) return;

  const success = await clipboardCopy(
    text,
    t("keys.createdKeyReveal.quickStart.exampleCopied"),
  );
  if (success) {
    createdKeyExampleCopied.value = true;
    if (createdKeyExampleCopyTimer) clearTimeout(createdKeyExampleCopyTimer);
    createdKeyExampleCopyTimer = setTimeout(() => {
      createdKeyExampleCopied.value = false;
      createdKeyExampleCopyTimer = null;
    }, 1200);
  }
};

const acknowledgeCreatedKey = () => {
  createdKeyToReveal.value = null;
  createdKeyCopied.value = false;
  createdKeyExampleTab.value = "curl";
  createdKeyExampleCopied.value = false;
  if (createdKeyCopyTimer) {
    clearTimeout(createdKeyCopyTimer);
    createdKeyCopyTimer = null;
  }
  if (createdKeyExampleCopyTimer) {
    clearTimeout(createdKeyExampleCopyTimer);
    createdKeyExampleCopyTimer = null;
  }
};

const isAbortError = (error: unknown) => {
  if (!error || typeof error !== "object") return false;
  const { name, code } = error as { name?: string; code?: string };
  return name === "AbortError" || code === "ERR_CANCELED";
};

const loadApiKeys = async () => {
  abortController?.abort();
  const controller = new AbortController();
  abortController = controller;
  const { signal } = controller;
  loading.value = true;
  try {
    // Build filters
    const filters: {
      search?: string;
      status?: string;
      group_id?: number | string;
    } = {};
    if (filterSearch.value) filters.search = filterSearch.value;
    if (filterStatus.value) filters.status = filterStatus.value;
    if (filterGroupId.value !== "") filters.group_id = filterGroupId.value;

    const response = await keysAPI.list(
      pagination.value.page,
      pagination.value.page_size,
      filters,
      {
        signal,
      },
    );
    if (signal.aborted) return;
    apiKeys.value = response.items;
    pagination.value.total = response.total;
    pagination.value.pages = response.pages;

    // Load usage stats for all API keys in the list
    if (response.items.length > 0) {
      const keyIds = response.items.map((k) => k.id);
      try {
        const usageResponse = await usageAPI.getDashboardApiKeysUsage(keyIds, {
          signal,
        });
        if (signal.aborted) return;
        usageStats.value = usageResponse.stats ?? {};
      } catch (e) {
        if (!isAbortError(e)) {
          console.error("Failed to load usage stats:", e);
        }
      }
    }
  } catch (error) {
    if (isAbortError(error)) {
      return;
    }
    appStore.showError(t("keys.failedToLoad"));
  } finally {
    if (abortController === controller) {
      loading.value = false;
    }
  }
};

const loadGroups = async () => {
  try {
    const channels = await userChannelsAPI.getAvailable();
    const availableGroups = new Map<number, UserAvailableGroup>();
    for (const channel of channels) {
      for (const platform of channel.platforms) {
        for (const group of platform.groups) {
          if (!availableGroups.has(group.id)) {
            availableGroups.set(group.id, group);
          }
        }
      }
    }
    groups.value = Array.from(availableGroups.values()).map((group) => ({
      id: group.id,
      name: group.name,
      description: group.description || null,
      platform: group.platform as Group["platform"],
      rate_multiplier: group.rate_multiplier,
      is_exclusive: group.is_exclusive,
      status: "active",
      subscription_type: group.subscription_type as Group["subscription_type"],
    })) as Group[];
    if (
      showCreateModal.value &&
      !showEditModal.value &&
      formData.value.group_ids.length === 0
    ) {
      formData.value.group_ids = getDefaultCreateGroupIds();
    }
  } catch (error) {
    console.error("Failed to load groups:", error);
  }
};

const loadUserGroupRates = async () => {
  try {
    userGroupRates.value = await userGroupsAPI.getUserGroupRates();
  } catch (error) {
    console.error("Failed to load user group rates:", error);
  }
};

const loadPublicSettings = async () => {
  try {
    publicSettings.value = await authAPI.getPublicSettings();
  } catch (error) {
    console.error("Failed to load public settings:", error);
  }
};

const openUseKeyModal = async (key: ApiKey) => {
  selectedKey.value = key;
  useKeyPlaintext.value = "";
  showUseKeyModal.value = true;
  // Resolve plaintext async; guard against stale updates if modal was closed or key changed
  const plaintext = await resolvePlaintextApiKey(key);
  if (plaintext && showUseKeyModal.value && selectedKey.value?.id === key.id) {
    useKeyPlaintext.value = plaintext;
  }
};

const closeUseKeyModal = () => {
  showUseKeyModal.value = false;
  selectedKey.value = null;
  useKeyPlaintext.value = "";
};

const handlePageChange = (page: number) => {
  pagination.value.page = page;
  loadApiKeys();
};

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.page_size = pageSize;
  pagination.value.page = 1;
  loadApiKeys();
};

const editKey = (key: ApiKey) => {
  selectedKey.value = key;
  const hasIPRestriction =
    key.ip_whitelist?.length > 0 || key.ip_blacklist?.length > 0;
  const hasExpiration = !!key.expires_at;
  const selectedGroupId = key.group_id ?? key.group_ids?.[0] ?? null;
  formData.value = {
    name: key.name,
    group_ids: selectedGroupId === null ? [] : [selectedGroupId],
    enable_model_restriction: (key.allowed_models?.length || 0) > 0,
    allowed_models: [...(key.allowed_models || [])],
    status:
      key.status === "quota_exhausted" || key.status === "expired"
        ? "inactive"
        : key.status,
    use_custom_key: false,
    custom_key: "",
    enable_ip_restriction: hasIPRestriction,
    ip_whitelist: (key.ip_whitelist || []).join("\n"),
    ip_blacklist: (key.ip_blacklist || []).join("\n"),
    enable_quota: key.quota > 0,
    quota: key.quota > 0 ? key.quota : null,
    enable_rate_limit:
      key.rate_limit_5h > 0 || key.rate_limit_1d > 0 || key.rate_limit_7d > 0,
    rate_limit_5h: key.rate_limit_5h || null,
    rate_limit_1d: key.rate_limit_1d || null,
    rate_limit_7d: key.rate_limit_7d || null,
    enable_expiration: hasExpiration,
    expiration_preset: "custom",
    expiration_date: key.expires_at ? formatDateTimeLocal(key.expires_at) : "",
  };
  showEditModal.value = true;
};

const confirmToggleKeyStatus = (key: ApiKey) => {
  pendingStatusAction.value = {
    key,
    status: key.status === "active" ? "inactive" : "active",
  };
  showStatusDialog.value = true;
};

const cancelToggleKeyStatus = () => {
  showStatusDialog.value = false;
  pendingStatusAction.value = null;
};

const handleToggleKeyStatus = async () => {
  const action = pendingStatusAction.value;
  if (!action) return;

  showStatusDialog.value = false;
  statusUpdatingKeyId.value = action.key.id;
  try {
    await keysAPI.toggleStatus(action.key.id, action.status);
    appStore.showSuccess(
      action.status === "active"
        ? t("keys.keyEnabledSuccess")
        : t("keys.keyDisabledSuccess"),
    );
    loadApiKeys();
  } catch (error) {
    appStore.showError(t("keys.failedToUpdateStatus"));
  } finally {
    statusUpdatingKeyId.value = null;
    pendingStatusAction.value = null;
  }
};

const changeGroup = async (key: ApiKey, newGroupId: number | null) => {
  groupSelectorKeyId.value = null;
  dropdownPosition.value = null;
  if (key.group_id === newGroupId) return;

  try {
    await keysAPI.update(key.id, {
      group_id: newGroupId,
      group_ids: newGroupId ? [newGroupId] : [],
    });
    appStore.showSuccess(t("keys.groupChangedSuccess"));
    loadApiKeys();
  } catch (error) {
    appStore.showError(t("keys.failedToChangeGroup"));
  }
};

const closeGroupSelector = (event: MouseEvent) => {
  const target = event.target as HTMLElement;
  // Check if click is inside the dropdown or the trigger button
  if (
    !target.closest(".group\\/dropdown") &&
    !dropdownRef.value?.contains(target)
  ) {
    groupSelectorKeyId.value = null;
    dropdownPosition.value = null;
  }
  if (!columnDropdownRef.value?.contains(target)) {
    showColumnDropdown.value = false;
  }
};

const confirmDelete = (key: ApiKey) => {
  selectedKey.value = key;
  showDeleteDialog.value = true;
};

function createApiKeyCreateIdempotencyKey() {
  if (
    typeof crypto !== "undefined" &&
    typeof crypto.randomUUID === "function"
  ) {
    return `api-key-create-${crypto.randomUUID()}`;
  }
  return `api-key-create-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

const handleSubmit = async () => {
  if (submitting.value) return;

  if (formData.value.group_ids.length === 0) {
    appStore.showError(t("keys.groupRequired"));
    return;
  }

  // Validate custom key if enabled
  if (!showEditModal.value && formData.value.use_custom_key) {
    if (!formData.value.custom_key) {
      appStore.showError(t("keys.customKeyRequired"));
      return;
    }
    if (customKeyError.value) {
      appStore.showError(customKeyError.value);
      return;
    }
  }

  // Parse IP lists only if IP restriction is enabled
  const parseIPList = (text: string): string[] =>
    text
      .split("\n")
      .map((ip) => ip.trim())
      .filter((ip) => ip.length > 0);
  const ipWhitelist = formData.value.enable_ip_restriction
    ? parseIPList(formData.value.ip_whitelist)
    : [];
  const ipBlacklist = formData.value.enable_ip_restriction
    ? parseIPList(formData.value.ip_blacklist)
    : [];

  // Calculate quota value (null/empty/0 = unlimited, stored as 0)
  const quota =
    formData.value.quota && formData.value.quota > 0 ? formData.value.quota : 0;
  const allowedModels = formData.value.enable_model_restriction
    ? Array.from(
        new Set(
          formData.value.allowed_models
            .map((model) => model.trim())
            .filter(Boolean),
        ),
      )
    : [];

  // Calculate expiration
  let expiresInDays: number | undefined;
  let expiresAt: string | null | undefined;
  if (formData.value.enable_expiration && formData.value.expiration_date) {
    if (!showEditModal.value) {
      // Create mode: calculate days from date
      const expDate = new Date(formData.value.expiration_date);
      const now = new Date();
      const diffDays = Math.ceil(
        (expDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24),
      );
      expiresInDays = diffDays > 0 ? diffDays : 1;
    } else {
      // Edit mode: use custom date directly
      expiresAt = new Date(formData.value.expiration_date).toISOString();
    }
  } else if (showEditModal.value) {
    // Edit mode: if expiration disabled or date cleared, send empty string to clear
    expiresAt = "";
  }

  // Calculate rate limit values (send 0 when toggle is off)
  const rateLimitData = formData.value.enable_rate_limit
    ? {
        rate_limit_5h:
          formData.value.rate_limit_5h && formData.value.rate_limit_5h > 0
            ? formData.value.rate_limit_5h
            : 0,
        rate_limit_1d:
          formData.value.rate_limit_1d && formData.value.rate_limit_1d > 0
            ? formData.value.rate_limit_1d
            : 0,
        rate_limit_7d:
          formData.value.rate_limit_7d && formData.value.rate_limit_7d > 0
            ? formData.value.rate_limit_7d
            : 0,
      }
    : { rate_limit_5h: 0, rate_limit_1d: 0, rate_limit_7d: 0 };

  submitting.value = true;
  try {
    if (showEditModal.value && selectedKey.value) {
      await keysAPI.update(selectedKey.value.id, {
        name: formData.value.name,
        group_id: formData.value.group_ids[0] ?? null,
        group_ids: formData.value.group_ids,
        allowed_models: allowedModels,
        status: formData.value.status,
        ip_whitelist: ipWhitelist,
        ip_blacklist: ipBlacklist,
        quota: quota,
        expires_at: expiresAt,
        rate_limit_5h: rateLimitData.rate_limit_5h,
        rate_limit_1d: rateLimitData.rate_limit_1d,
        rate_limit_7d: rateLimitData.rate_limit_7d,
      });
      appStore.showSuccess(t("keys.keyUpdatedSuccess"));
    } else {
      const customKey = formData.value.use_custom_key
        ? formData.value.custom_key
        : undefined;
      const createdKey = await keysAPI.create(
        formData.value.name,
        formData.value.group_ids[0] ?? null,
        formData.value.group_ids,
        allowedModels,
        customKey,
        ipWhitelist,
        ipBlacklist,
        quota,
        expiresInDays,
        rateLimitData,
        { idempotencyKey: createApiKeyCreateIdempotencyKey() },
      );
      if (createdKey?.key) {
        createdKeyToReveal.value = createdKey;
      }
      appStore.showSuccess(t("keys.keyCreatedSuccess"));
      // Only advance tour if active, on submit step, and creation succeeded
      if (onboardingStore.isCurrentStep('[data-tour="key-form-submit"]')) {
        onboardingStore.nextStep(500);
      }
    }
    closeModals();
    loadApiKeys();
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t("keys.failedToSave");
    appStore.showError(errorMsg);
    // Don't advance tour on error
  } finally {
    submitting.value = false;
  }
};

/**
 * 处理删除 API Key 的操作
 * 优化：错误处理改进，优先显示后端返回的具体错误消息（如权限不足等），
 * 若后端未返回消息则显示默认的国际化文本
 */
const handleDelete = async () => {
  if (!selectedKey.value) return;

  try {
    await keysAPI.delete(selectedKey.value.id);
    appStore.showSuccess(t("keys.keyDeletedSuccess"));
    showDeleteDialog.value = false;
    loadApiKeys();
  } catch (error: any) {
    // 优先使用后端返回的错误消息，提供更具体的错误信息给用户
    const errorMsg = error?.message || t("keys.failedToDelete");
    appStore.showError(errorMsg);
  }
};

const closeModals = () => {
  showCreateModal.value = false;
  showEditModal.value = false;
  selectedKey.value = null;
  resetFormData();
};

// Show reset quota confirmation dialog
const confirmResetQuota = () => {
  showResetQuotaDialog.value = true;
};

// Set expiration date based on quick select days
const setExpirationDays = (days: number) => {
  formData.value.expiration_preset = days.toString() as "7" | "30" | "90";
  const expDate = new Date();
  expDate.setDate(expDate.getDate() + days);
  formData.value.expiration_date = formatDateTimeLocal(expDate.toISOString());
};

// Reset quota used for an API key
const resetQuotaUsed = async () => {
  if (!selectedKey.value) return;
  const keyId = selectedKey.value.id;
  showResetQuotaDialog.value = false;
  try {
    await keysAPI.update(keyId, { reset_quota: true });
    appStore.showSuccess(t("keys.quotaResetSuccess"));
    // Refresh key data so backend-calculated status and quota fields stay authoritative.
    await loadApiKeys();
    const refreshedKey = apiKeys.value.find((k) => k.id === keyId);
    if (refreshedKey) {
      selectedKey.value = refreshedKey;
    }
  } catch (error: any) {
    const errorMsg =
      error.response?.data?.detail || t("keys.failedToResetQuota");
    appStore.showError(errorMsg);
  }
};

// Show reset rate limit confirmation dialog (from edit modal)
const confirmResetRateLimit = () => {
  showResetRateLimitDialog.value = true;
};

// Show reset rate limit confirmation dialog (from table row)
const confirmResetRateLimitFromTable = (row: ApiKey) => {
  selectedKey.value = row;
  showResetRateLimitDialog.value = true;
};

// Reset rate limit usage for an API key
const resetRateLimitUsage = async () => {
  if (!selectedKey.value) return;
  showResetRateLimitDialog.value = false;
  try {
    await keysAPI.update(selectedKey.value.id, {
      reset_rate_limit_usage: true,
    });
    appStore.showSuccess(t("keys.rateLimitResetSuccess"));
    // Refresh key data
    await loadApiKeys();
    // Update the editing key with fresh data
    const refreshedKey = apiKeys.value.find(
      (k) => k.id === selectedKey.value!.id,
    );
    if (refreshedKey) {
      selectedKey.value = refreshedKey;
    }
  } catch (error: any) {
    const errorMsg =
      error.response?.data?.detail || t("keys.failedToResetRateLimit");
    appStore.showError(errorMsg);
  }
};

const importToCcswitch = async (row: ApiKey) => {
  const platform = resolveCcsImportPlatform(row);
  if (!platform) {
    appStore.showError(t("keys.noGroupFound"));
    return;
  }

  const plaintextKey = await resolvePlaintextApiKey(row);
  if (!plaintextKey) return;
  const importRow = { ...row, key: plaintextKey };

  // For antigravity platform, show client selection dialog
  if (platform === "antigravity") {
    pendingCcsRow.value = importRow;
    showCcsClientSelect.value = true;
    return;
  }

  // For other platforms, execute directly
  executeCcsImport(importRow, platform === "gemini" ? "gemini" : "claude");
};

const resolveCcsDefaultModel = (
  row: ApiKey,
  platform: CcsImportPlatform,
  clientType: "claude" | "gemini",
): string => {
  const preferred =
    platform === "openai"
      ? "gpt-5.5"
      : platform === "anthropic" ||
          (platform === "antigravity" && clientType === "claude")
        ? CCS_CLAUDE_MODELS.fable
        : "";
  const allowedModels = (row.allowed_models || [])
    .map((model) => model.trim())
    .filter(Boolean);
  if (
    !preferred ||
    allowedModels.length === 0 ||
    allowedModels.includes(preferred)
  )
    return preferred;
  return allowedModels[0] || "";
};

const resolveCcsClaudeModels = (
  row: ApiKey,
  platform: CcsImportPlatform,
  clientType: "claude" | "gemini",
) => {
  if (
    clientType !== "claude" ||
    (platform !== "anthropic" && platform !== "antigravity")
  ) {
    return null;
  }

  const allowedModels = (row.allowed_models || [])
    .map((model) => model.trim())
    .filter(Boolean);
  const isAvailable = (model: string) =>
    allowedModels.length === 0 || allowedModels.includes(model);
  const defaultModel =
    [
      CCS_CLAUDE_MODELS.fable,
      CCS_CLAUDE_MODELS.sonnet,
      CCS_CLAUDE_MODELS.opus,
      CCS_CLAUDE_MODELS.haiku,
    ].find(isAvailable) || allowedModels[0] || "";

  return {
    defaultModel,
    fableModel: isAvailable(CCS_CLAUDE_MODELS.fable)
      ? CCS_CLAUDE_MODELS.fable
      : "",
    haikuModel: isAvailable(CCS_CLAUDE_MODELS.haiku)
      ? CCS_CLAUDE_MODELS.haiku
      : "",
    sonnetModel: isAvailable(CCS_CLAUDE_MODELS.sonnet)
      ? CCS_CLAUDE_MODELS.sonnet
      : "",
    opusModel: isAvailable(CCS_CLAUDE_MODELS.opus)
      ? CCS_CLAUDE_MODELS.opus
      : "",
  };
};

const executeCcsImport = (row: ApiKey, clientType: "claude" | "gemini") => {
  const baseUrl = apiBaseUrl.value;
  const platform = resolveCcsImportPlatform(row);
  if (!platform) {
    appStore.showError(t("keys.noGroupFound"));
    return;
  }

  // Determine app name and endpoint based on platform and client type
  let app: string;
  let endpoint: string;

  if (platform === "antigravity") {
    // Antigravity always uses /antigravity suffix
    app = clientType === "gemini" ? "gemini" : "claude";
    endpoint = `${baseUrl}/antigravity`;
  } else {
    switch (platform) {
      case "openai":
        app = "codex";
        endpoint = openAICompatibleBaseUrl.value;
        break;
      case "gemini":
        app = "gemini";
        endpoint = baseUrl;
        break;
      default: // anthropic
        app = "claude";
        endpoint = baseUrl;
    }
  }

  const usageScript = `({
    request: {
      url: "{{baseUrl}}/v1/usage",
      method: "GET",
      headers: { "Authorization": "Bearer {{apiKey}}" }
    },
    extractor: function(response) {
      const remaining = response?.remaining ?? response?.quota?.remaining ?? response?.balance;
      const unit = response?.unit ?? response?.quota?.unit ?? "USD";
      return {
        isValid: response?.is_active ?? response?.isValid ?? true,
        remaining,
        unit
      };
    }
  })`;
  const providerName = normalizeSiteName(
    publicSettings.value?.site_name || DEFAULT_SITE_NAME,
  );
  const claudeModels = resolveCcsClaudeModels(row, platform, clientType);
  const defaultModel =
    claudeModels?.defaultModel ||
    resolveCcsDefaultModel(row, platform, clientType);

  const params = new URLSearchParams({
    resource: "provider",
    app: app,
    name: providerName,
    homepage: baseUrl,
    endpoint: endpoint,
    apiKey: row.key,
    enabled: "true",
    configFormat: "json",
    usageEnabled: "true",
    usageBaseUrl: baseUrl,
    usageScript: btoa(usageScript),
    usageAutoInterval: "30",
  });
  if (defaultModel) params.set("model", defaultModel);
  if (claudeModels) {
    if (claudeModels.haikuModel) {
      params.set("haikuModel", claudeModels.haikuModel);
    }
    if (claudeModels.sonnetModel) {
      params.set("sonnetModel", claudeModels.sonnetModel);
    }
    if (claudeModels.opusModel) {
      params.set("opusModel", claudeModels.opusModel);
    }
    if (claudeModels.fableModel) {
      params.set(
        "config",
        btoa(
          JSON.stringify({
            env: {
              ANTHROPIC_DEFAULT_FABLE_MODEL: claudeModels.fableModel,
            },
          }),
        ),
      );
    }
  }
  const deeplink = `ccswitch://v1/import?${params.toString()}`;

  try {
    window.open(deeplink, "_self");

    // Check if the protocol handler worked by detecting if we're still focused
    setTimeout(() => {
      if (document.hasFocus()) {
        // Still focused means the protocol handler likely failed
        appStore.showError(t("keys.ccSwitchNotInstalled"));
      }
    }, 100);
  } catch (error) {
    appStore.showError(t("keys.ccSwitchNotInstalled"));
  }
};

const handleCcsClientSelect = (clientType: "claude" | "gemini") => {
  if (pendingCcsRow.value) {
    executeCcsImport(pendingCcsRow.value, clientType);
  }
  showCcsClientSelect.value = false;
  pendingCcsRow.value = null;
};

const closeCcsClientSelect = () => {
  showCcsClientSelect.value = false;
  pendingCcsRow.value = null;
};

function formatResetTime(resetAt: string | null): string {
  if (!resetAt) return "";
  const diff = new Date(resetAt).getTime() - now.value.getTime();
  if (diff <= 0) return t("keys.resetNow");
  const days = Math.floor(diff / 86400000);
  const hours = Math.floor((diff % 86400000) / 3600000);
  const mins = Math.floor((diff % 3600000) / 60000);
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${mins}m`;
  return `${mins}m`;
}

const usageCellTitle = (row: ApiKey): string => {
  const stats = usageStats.value[row.id];
  const parts = [
    `${t("keys.today")} ${formatCurrencyTitle(stats?.today_actual_cost ?? 0)}`,
    `${t("keys.total")} ${formatCurrencyTitle(stats?.total_actual_cost ?? 0)}`,
  ];
  if (row.quota > 0) {
    parts.push(
      `${t("keys.quota")} ${formatCurrency(row.quota_used || 0)} / ${formatCurrency(row.quota || 0)}`,
    );
  }
  return parts.join(" · ");
};

interface RateWindow {
  label: string;
  usage: number;
  limit: number;
  resetAt: string | null;
}

const rateLimitWindows = (row: ApiKey): RateWindow[] => {
  const windows: RateWindow[] = [];
  if (row.rate_limit_5h > 0) {
    windows.push({
      label: "5h",
      usage: row.usage_5h || 0,
      limit: row.rate_limit_5h,
      resetAt: row.reset_5h_at || null,
    });
  }
  if (row.rate_limit_1d > 0) {
    windows.push({
      label: "1d",
      usage: row.usage_1d || 0,
      limit: row.rate_limit_1d,
      resetAt: row.reset_1d_at || null,
    });
  }
  if (row.rate_limit_7d > 0) {
    windows.push({
      label: "7d",
      usage: row.usage_7d || 0,
      limit: row.rate_limit_7d,
      resetAt: row.reset_7d_at || null,
    });
  }
  return windows;
};

const tightestRateWindow = (row: ApiKey): RateWindow | null => {
  const windows = rateLimitWindows(row);
  if (windows.length === 0) return null;
  return windows.reduce((tightest, current) =>
    current.usage / current.limit > tightest.usage / tightest.limit
      ? current
      : tightest,
  );
};

const rateLimitCellTitle = (row: ApiKey): string =>
  rateLimitWindows(row)
    .map((window) => {
      const reset = window.resetAt ? formatResetTime(window.resetAt) : "";
      const base = `${window.label} ${formatCurrency(window.usage)} / ${formatCurrency(window.limit)}`;
      return reset ? `${base} (⟳ ${reset})` : base;
    })
    .join(" · ");

onMounted(() => {
  loadSavedColumns();
  loadApiKeys();
  loadGroups();
  loadUserGroupRates();
  loadPublicSettings();
  document.addEventListener("click", closeGroupSelector);
  resetTimer = setInterval(() => {
    now.value = new Date();
  }, 60000);
});

onUnmounted(() => {
  document.removeEventListener("click", closeGroupSelector);
  if (resetTimer) clearInterval(resetTimer);
  if (createdKeyCopyTimer) clearTimeout(createdKeyCopyTimer);
  if (baseUrlCopyTimer) clearTimeout(baseUrlCopyTimer);
  if (baseUrlRootCopyTimer) clearTimeout(baseUrlRootCopyTimer);
});
</script>

<style scoped>
.keys-page-surface {
  width: 100%;
}

.keys-purchase-link {
  display: inline-flex;
  width: fit-content;
  align-items: center;
  gap: 0.4rem;
  margin: 0.75rem 0 1rem;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  color: var(--ssxz-text-secondary);
  background: var(--ssxz-surface-raised);
  padding: 0.45rem 0.7rem;
  font-size: 0.78rem;
  font-weight: 600;
  text-decoration: none;
  transition:
    border-color 160ms ease,
    color 160ms ease,
    background 160ms ease;
}

.keys-purchase-link:hover {
  border-color: var(--ssxz-border-strong);
  color: var(--ssxz-text-primary);
  background: var(--ssxz-surface-muted);
}

.keys-page-surface--workbench {
  margin-inline: auto;
  max-width: 100%;
}

.keys-access-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  min-width: 0;
  padding: 0.1rem 0.15rem 0.15rem;
}

.keys-access-base {
  display: flex;
  min-width: 0;
  max-width: 100%;
  align-items: center;
  gap: 0.55rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: color-mix(in srgb, var(--ssxz-input) 88%, transparent);
  padding: 0.42rem 0.65rem;
}

.keys-access-base > span {
  flex: 0 0 auto;
  color: var(--ssxz-text-muted);
  font-size: 0.76rem;
  font-weight: 650;
}

.keys-access-base code {
  min-width: 0;
  overflow: hidden;
  color: var(--ssxz-text-primary);
  font-size: 0.78rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.keys-access-copy {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 1.75rem;
  height: 1.75rem;
  border-radius: 999px;
  color: var(--ssxz-text-muted);
  transition:
    background 0.16s ease,
    color 0.16s ease;
}

.keys-access-copy:hover {
  background: var(--ssxz-surface);
  color: var(--ssxz-text-primary);
}

.keys-doc-link {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 0.4rem;
  color: var(--ssxz-action);
  font-size: 0.8125rem;
  font-weight: 650;
  transition: color 0.16s ease;
}

.keys-doc-link:hover {
  color: var(--ssxz-text-primary);
}

.keys-filter-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
}

.keys-page-surface--workbench :deep(.table-page-layout) {
  height: auto;
  gap: 1rem;
}

.keys-page-surface--workbench :deep(.layout-section-fixed) {
  border: 1px solid var(--ssxz-border);
  border-radius: 1.25rem;
  background: color-mix(in srgb, var(--ssxz-surface-raised) 88%, transparent);
  box-shadow: var(--ssxz-shadow-sm);
  padding: 1rem;
}

.keys-page-surface--workbench :deep(.layout-section-fixed:first-child) {
  display: flex;
  justify-content: flex-end;
  padding: 0;
  border: 0;
  background: transparent;
  box-shadow: none;
}

.keys-page-surface--workbench
  :deep(
    .keys-workbench-layout--no-pagination > .layout-section-fixed:last-child
  ) {
  display: none;
}

.keys-page-surface--workbench :deep(.table-scroll-container) {
  height: auto;
  min-height: 22rem;
  border-color: var(--ssxz-border);
  border-radius: 1.25rem;
  background: color-mix(in srgb, var(--ssxz-surface-raised) 90%, transparent);
  box-shadow: var(--ssxz-shadow-sm);
}

.keys-page-surface--workbench :deep(table) {
  width: 100%;
  min-width: 100%;
  table-layout: fixed;
}

.keys-page-surface--workbench :deep(.table-scroll-container .table-wrapper) {
  max-height: none;
  overflow-x: hidden;
}

.keys-page-surface--workbench :deep(.table-scroll-container th) {
  background: color-mix(in srgb, var(--ssxz-surface-muted) 72%, transparent);
  color: var(--ssxz-subtle);
  font-size: 0.78rem;
  letter-spacing: 0;
  white-space: nowrap;
}

.keys-page-surface--workbench :deep(.table-scroll-container td) {
  height: 3.5rem;
  padding-top: 0.5rem;
  padding-bottom: 0.5rem;
  border-color: color-mix(in srgb, var(--ssxz-border) 62%, transparent);
  color: var(--ssxz-body);
  white-space: normal;
}

.keys-page-surface--workbench :deep(.keys-actions-column) {
  /* Five 2.15rem action buttons plus gaps and cell padding need this width. */
  width: 14.5rem;
  min-width: 14.5rem;
}

.keys-row-actions {
  display: flex;
  min-height: 100%;
  align-items: center;
  justify-content: flex-end;
  gap: 0.35rem;
  white-space: nowrap;
}

.keys-usage-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  white-space: nowrap;
}

.keys-usage-bar {
  display: inline-block;
  width: 3.5rem;
  height: 0.25rem;
  flex: 0 0 auto;
  overflow: hidden;
  border-radius: 999px;
  background: color-mix(in srgb, var(--ssxz-border) 80%, transparent);
}

.keys-usage-bar__fill {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: var(--ssxz-action);
  transition: width 0.2s ease;
}

.keys-usage-bar__fill--danger {
  background: hsl(var(--destructive));
}

.keys-status-cell {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  white-space: nowrap;
}

.keys-status-dot {
  width: 0.5rem;
  height: 0.5rem;
  flex: 0 0 auto;
  border-radius: 999px;
  background: var(--ssxz-text-muted);
}

.keys-status-dot--muted {
  background: var(--ssxz-border-strong);
}

.keys-status-dot--danger {
  background: hsl(var(--destructive));
}

.keys-action-button {
  display: inline-flex;
  width: 2.15rem;
  height: 2.15rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 0.65rem;
  color: var(--ssxz-text-muted);
  cursor: pointer;
  transition:
    border-color 0.16s ease,
    background 0.16s ease,
    color 0.16s ease,
    transform 0.12s ease;
}

.keys-action-button:hover {
  border-color: var(--ssxz-border-strong);
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-text-primary);
}

.keys-action-button:active {
  transform: translateY(1px);
}

.keys-action-button:focus-visible {
  outline: 2px solid var(--ssxz-action);
  outline-offset: 2px;
}

.keys-action-button:disabled {
  cursor: not-allowed;
}

.keys-action-button--primary {
  color: var(--ssxz-action);
}

.keys-action-button--primary:hover {
  border-color: var(--ssxz-action);
  background: hsl(var(--ssxz-action-rgb) / 0.08);
  color: var(--ssxz-action);
}

.keys-action-button--danger:hover {
  border-color: hsl(var(--destructive) / 0.3);
  background: hsl(var(--destructive) / 0.08);
  color: hsl(var(--destructive));
}

.keys-page-surface--workbench :deep(.empty-state) {
  padding: 3.5rem 1rem;
}

.keys-page-surface--workbench :deep(.empty-state-visual) {
  border-color: color-mix(in srgb, var(--ssxz-action) 24%, var(--ssxz-border));
  background: color-mix(
    in srgb,
    var(--ssxz-action-soft) 72%,
    var(--ssxz-surface-muted)
  );
}

.keys-page-surface--workbench :deep(.empty-state-icon) {
  color: var(--ssxz-action);
}

.keys-page-surface--workbench :deep(.empty-state-title) {
  color: var(--ssxz-text-primary);
  font-size: 1rem;
  font-weight: 700;
}

.keys-page-surface--workbench :deep(.empty-state-description) {
  max-width: 24rem;
  margin-inline: auto;
  color: var(--ssxz-text-secondary);
}

.keys-page-surface--workbench :deep(.btn-primary) {
  border-color: transparent;
  background: var(--ssxz-action);
  color: var(--ssxz-action-text);
}

.keys-page-surface--workbench :deep(.btn-secondary) {
  border-color: var(--ssxz-border);
  background: color-mix(in srgb, var(--ssxz-surface-raised) 86%, transparent);
  color: var(--ssxz-body);
}

.keys-column-menu {
  position: absolute;
  top: calc(100% + 0.45rem);
  right: 0;
  z-index: 50;
  width: 13rem;
  overflow: hidden;
  padding: 0.4rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.85rem;
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-dialog);
}

.keys-column-menu__item {
  display: flex;
  width: 100%;
  min-height: 2.35rem;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.5rem 0.65rem;
  border-radius: 0.6rem;
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
  text-align: left;
  transition:
    background-color 140ms ease,
    color 140ms ease;
}

.keys-column-menu__item:hover,
.keys-column-menu__item:focus-visible {
  background: var(--ssxz-primary-soft);
  color: var(--ssxz-text);
  outline: none;
}

.keys-page-surface--workbench :deep(input),
.keys-page-surface--workbench :deep(select) {
  background: var(--ssxz-input);
  color: var(--ssxz-text);
}

.keys-group-dropdown {
  background: var(--ssxz-surface-raised);
  border: 1px solid var(--ssxz-border-strong);
  color: var(--ssxz-text);
  box-shadow: var(--ssxz-shadow-dialog);
}

.keys-group-dropdown > div:first-child {
  border-color: var(--ssxz-border);
}

.keys-group-dropdown input {
  border-color: var(--ssxz-border);
  background: var(--ssxz-input);
  color: var(--ssxz-text);
}

.keys-group-dropdown input:focus {
  border-color: var(--ssxz-primary);
  box-shadow: var(--ssxz-focus-ring);
}

.keys-group-dropdown button {
  color: var(--ssxz-text-secondary);
}

.keys-group-dropdown button:hover,
.keys-group-dropdown button.bg-primary-50,
.keys-group-dropdown button.dark\:bg-primary-900\/20 {
  background: var(--ssxz-primary-soft);
  color: var(--ssxz-text);
}

@media (max-width: 767px) {
  .keys-access-row {
    align-items: flex-start;
    flex-direction: column;
    gap: 0.75rem;
    padding-inline: 0;
  }

  .keys-access-base {
    width: 100%;
  }

  .keys-doc-link {
    padding-inline: 0.15rem;
  }

  .keys-page-surface--workbench :deep(.layout-section-fixed) {
    border-radius: 1rem;
    padding: 0.85rem;
  }

  .keys-page-surface--workbench :deep(.layout-section-fixed:first-child) {
    padding: 0;
  }

  .keys-page-surface--workbench :deep(.table-scroll-container) {
    min-height: 16rem;
    border-radius: 1rem;
  }
}
</style>
