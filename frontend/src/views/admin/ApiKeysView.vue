<template>
  <AppLayout>
    <AdminPageHeader
      title="全站 API Key"
      description="查看全站密钥状态、所属分组与实际消耗"
    />

    <TablePageLayout class="api-key-inventory-page admin-b1-outline-scope">
      <template #actions>
        <div
          class="mb-2 grid grid-cols-2 overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900 xl:grid-cols-5"
          data-testid="api-key-summary"
        >
          <div
            v-for="metric in summaryMetrics"
            :key="metric.key"
            class="border-b border-r border-gray-200 px-4 py-3 even:border-r-0 last:border-b-0 last:border-r-0 dark:border-dark-700 xl:border-b-0 xl:border-r xl:even:border-r xl:last:border-r-0"
            :class="{ 'col-span-2 xl:col-span-1': metric.key === 'cost' }"
            :data-testid="metric.testId"
          >
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">
              {{ metric.label }}
            </p>
            <p class="mt-1 text-xl font-semibold text-gray-950 dark:text-white">
              {{ metric.value }}
            </p>
          </div>
        </div>
      </template>

      <template #filters>
        <div class="flex flex-wrap items-start gap-3">
          <div class="min-w-56 flex-1 lg:max-w-80">
            <SearchInput
              v-model="filters.search"
              :placeholder="t('admin.apiKeyInventory.searchPlaceholder')"
              data-testid="key-search"
              @search="applyFilters"
            />
          </div>

          <div ref="userFilterRef" class="relative w-full sm:w-64">
            <div class="relative">
              <input
                v-model="userKeyword"
                class="input pr-9"
                type="search"
                :placeholder="t('admin.apiKeyInventory.userFilter')"
                data-testid="user-filter"
                @focus="showUserDropdown = true"
                @input="debouncedUserSearch"
              />
              <LiquidButton
                v-if="filters.user_id"
                type="button"
                class="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-gray-400 hover:text-gray-700 dark:hover:text-gray-200"
                :title="t('common.clear')"
                @click="clearUserFilter"
                variant="plain"
                size="icon"
              >
                <Icon name="x" size="sm" />
              </LiquidButton>
            </div>
            <div
              v-if="showUserDropdown && userKeyword.trim()"
              class="absolute z-40 mt-1 max-h-64 w-full overflow-y-auto rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-600 dark:bg-dark-800"
            >
              <LiquidButton
                v-for="user in userResults"
                :key="user.id"
                type="button"
                class="flex w-full items-center justify-between gap-3 px-3 py-2 text-left hover:bg-gray-50 dark:hover:bg-dark-700"
                @click="selectUser(user)"
                variant="plain"
                size="sm"
              >
                <span
                  class="min-w-0 truncate text-sm text-gray-800 dark:text-gray-100"
                  >{{ user.email }}</span
                >
                <span class="shrink-0 text-xs text-gray-400"
                  >#{{ user.id }}</span
                >
              </LiquidButton>
              <p
                v-if="!userSearchLoading && userResults.length === 0"
                class="px-3 py-3 text-sm text-gray-500"
              >
                {{ t("empty.noData") }}
              </p>
            </div>
          </div>

          <Select
            v-model="filters.group_id"
            :options="groupOptions"
            class="w-full sm:w-56"
            searchable
            @change="applyFilters"
          />
          <Select
            v-model="filters.status"
            :options="statusOptions"
            class="w-full sm:w-40"
            @change="applyFilters"
          />
          <Select
            v-model="sortBy"
            :options="sortOptions"
            class="w-full sm:w-48"
            @change="applySortSelection"
          />
          <LiquidButton
            type="button"
            class="px-3"
            :title="
              sortOrder === 'desc'
                ? t('admin.apiKeyInventory.sortDescending')
                : t('admin.apiKeyInventory.sortAscending')
            "
            @click="toggleSortOrder"
            variant="outline"
            size="icon"
          >
            <Icon
              :name="sortOrder === 'desc' ? 'arrowDown' : 'arrowUp'"
              size="md"
            />
          </LiquidButton>

          <LiquidButton
            type="button"
            class="ml-auto px-3"
            :disabled="loading"
            :title="t('common.refresh')"
            @click="loadKeys"
            variant="outline"
            size="icon"
          >
            <Icon
              name="refresh"
              size="md"
              :class="loading ? 'animate-spin' : ''"
            />
          </LiquidButton>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="items"
          :loading="loading"
          row-key="id"
          server-side-sort
          default-sort-key="last_used_at"
          default-sort-order="desc"
          :estimate-row-height="72"
          @sort="handleSort"
        >
          <template #cell-user="{ row }">
            <div class="w-full min-w-0 md:w-[136px] md:max-w-[136px]">
              <div
                class="truncate font-medium text-gray-950 dark:text-white"
                :title="row.user.email"
              >
                {{ row.user.email }}
              </div>
              <div
                class="mt-1.5 flex items-center gap-2 text-[11px] text-gray-400 dark:text-dark-500"
              >
                <span>#{{ row.user.id }}</span>
                <span v-if="row.user.username">{{ row.user.username }}</span>
                <span class="font-medium text-gray-600 dark:text-gray-400">{{
                  formatMoney(row.user.balance)
                }}</span>
              </div>
            </div>
          </template>

          <template #cell-key="{ row }">
            <div class="w-full min-w-0 md:w-[136px] md:max-w-[136px]">
              <div
                class="truncate font-medium text-gray-950 dark:text-white"
                :title="row.name"
              >
                {{ row.name }}
              </div>
              <code
                class="mt-1.5 block text-[11px] text-gray-400 dark:text-dark-500"
                >{{ row.key }}</code
              >
              <div
                class="mt-2.5 flex items-center justify-between gap-3 text-[11px] text-gray-400 dark:text-dark-500"
              >
                <span>{{ t("admin.apiKeyInventory.quota") }}</span>
                <span class="font-medium text-gray-600 dark:text-gray-400">
                  {{
                    row.quota > 0
                      ? `${formatMoney(row.quota_used)} / ${formatMoney(row.quota)}`
                      : t("admin.apiKeyInventory.unlimited")
                  }}
                </span>
              </div>
              <div
                v-if="row.quota > 0"
                class="mt-1 h-1 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700"
              >
                <div
                  class="h-full rounded-full bg-primary-500"
                  :style="{
                    width: `${Math.min(100, (row.quota_used / row.quota) * 100)}%`,
                  }"
                />
              </div>
            </div>
          </template>

          <template #cell-group="{ value }">
            <GroupBadge
              v-if="value"
              :name="value.name"
              :platform="value.platform"
              :rate-multiplier="value.rate_multiplier"
              :title="value.name"
              class="inventory-group-badge w-full max-w-full md:w-56 md:max-w-56"
            />
            <span v-else class="text-sm text-gray-400">{{
              t("admin.apiKeyInventory.noGroup")
            }}</span>
          </template>

          <template #cell-total_actual_cost="{ row }">
            <div
              class="w-full min-w-0 space-y-1.5 text-xs tabular-nums md:w-32"
            >
              <div
                class="flex justify-between gap-3 text-gray-400 dark:text-dark-500"
              >
                <span>{{ t("admin.apiKeyInventory.todayCost") }}</span>
                <span class="font-medium text-gray-800 dark:text-gray-200">{{
                  formatMoney(row.today_actual_cost)
                }}</span>
              </div>
              <div
                class="flex justify-between gap-3 text-gray-400 dark:text-dark-500"
              >
                <span>{{ t("admin.apiKeyInventory.last30DaysCost") }}</span>
                <span class="font-medium text-gray-800 dark:text-gray-200">{{
                  formatMoney(row.last_30_days_actual_cost)
                }}</span>
              </div>
              <div
                class="flex justify-between gap-3 text-gray-500 dark:text-dark-400"
              >
                <span>{{ t("admin.apiKeyInventory.totalCost") }}</span>
                <span class="font-semibold text-gray-950 dark:text-white">{{
                  formatMoney(row.total_actual_cost)
                }}</span>
              </div>
            </div>
          </template>

          <template #cell-last_used_at="{ row }">
            <div class="w-full min-w-0 space-y-2 md:w-32">
              <span
                :class="[
                  'badge inventory-status-badge',
                  statusClass(row.status),
                ]"
                >{{ statusLabel(row.status) }}</span
              >
              <div class="text-[11px] text-gray-400 dark:text-dark-500">
                {{ formatCompactTimestamp(row.last_used_at) }}
              </div>
            </div>
          </template>

          <template #cell-created_at="{ row }">
            <div
              class="w-full min-w-0 space-y-1.5 text-[11px] text-gray-400 dark:text-dark-500 md:w-28"
            >
              <div>
                <span class="mr-2">{{
                  t("admin.apiKeyInventory.createdShort")
                }}</span
                >{{ formatDateOnly(row.created_at) }}
              </div>
              <div>
                <span class="mr-2">{{
                  t("admin.apiKeyInventory.expiresShort")
                }}</span
                >{{
                  row.expires_at
                    ? formatDateOnly(row.expires_at)
                    : t("admin.apiKeyInventory.neverExpires")
                }}
              </div>
            </div>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center justify-end gap-1.5">
              <LiquidButton
                v-if="row.status !== 'expired'"
                type="button"
                class="inline-flex h-9 w-9 items-center justify-center rounded-md border border-gray-200 text-gray-500 transition-colors hover:border-gray-300 hover:bg-gray-50 hover:text-gray-900 disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-600 dark:text-dark-400 dark:hover:bg-dark-700 dark:hover:text-white"
                :data-action="row.status === 'active' ? 'disable' : 'enable'"
                :disabled="isMutating(row.id)"
                :title="
                  row.status === 'active'
                    ? t('admin.apiKeyInventory.disableAction')
                    : t('admin.apiKeyInventory.enableAction')
                "
                :aria-label="
                  row.status === 'active'
                    ? t('admin.apiKeyInventory.disableAction')
                    : t('admin.apiKeyInventory.enableAction')
                "
                @click="setKeyEnabled(row, row.status !== 'active')"
                variant="plain"
                size="icon"
              >
                <Icon
                  :name="row.status === 'active' ? 'xCircle' : 'checkCircle'"
                  size="sm"
                />
              </LiquidButton>
              <LiquidButton
                type="button"
                class="inline-flex h-9 w-9 items-center justify-center rounded-md border border-gray-200 text-gray-500 transition-colors hover:border-gray-300 hover:bg-gray-50 hover:text-gray-900 disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-600 dark:text-dark-400 dark:hover:bg-dark-700 dark:hover:text-white"
                data-action="change-group"
                :disabled="isMutating(row.id)"
                :title="t('admin.apiKeyInventory.changeGroupAction')"
                :aria-label="t('admin.apiKeyInventory.changeGroupAction')"
                @click="openGroupDialog(row)"
                variant="plain"
                size="icon"
              >
                <Icon name="edit" size="sm" />
              </LiquidButton>
              <LiquidButton
                type="button"
                class="inline-flex h-9 w-9 items-center justify-center rounded-md border border-red-200 text-red-500 transition-colors hover:border-red-300 hover:bg-red-50 hover:text-red-700 disabled:cursor-not-allowed disabled:opacity-50 dark:border-red-900/60 dark:text-red-400 dark:hover:bg-red-950/30 dark:hover:text-red-300"
                data-action="delete"
                :disabled="isMutating(row.id)"
                :title="t('admin.apiKeyInventory.deleteAction')"
                :aria-label="t('admin.apiKeyInventory.deleteAction')"
                @click="openDeleteDialog(row)"
                variant="plain"
                size="icon"
              >
                <Icon name="trash" size="sm" />
              </LiquidButton>
            </div>
          </template>

          <template #empty>
            <div class="flex flex-col items-center py-4">
              <Icon
                name="key"
                size="xl"
                class="mb-3 text-gray-300 dark:text-dark-500"
              />
              <p class="font-medium text-gray-900 dark:text-gray-100">
                {{ t("admin.apiKeyInventory.empty") }}
              </p>
            </div>
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
      :show="editingGroupKey !== null"
      :title="t('admin.apiKeyInventory.changeGroupTitle')"
      width="narrow"
      @close="closeGroupDialog"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-400">
          {{ editingGroupKey?.name }} · {{ editingGroupKey?.user.email }}
        </p>
        <Select
          v-model="editingGroupID"
          :options="actionGroupOptions"
          searchable
          data-testid="group-action-select"
        />
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <LiquidButton
            type="button"
            @click="closeGroupDialog"
            variant="outline"
            size="sm"
          >
            {{ t("common.cancel") }}
          </LiquidButton>
          <LiquidButton
            type="button"
            data-testid="save-group-change"
            :disabled="editingGroupKey ? isMutating(editingGroupKey.id) : false"
            @click="saveGroupChange"
            size="default"
          >
            {{ t("common.save") }}
          </LiquidButton>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="deletingKey !== null"
      :title="t('admin.apiKeyInventory.deleteTitle')"
      :message="
        t('admin.apiKeyInventory.deleteConfirm', {
          name: deletingKey?.name || '',
          user: deletingKey?.user.email || '',
        })
      "
      :confirm-text="t('admin.apiKeyInventory.deleteAction')"
      danger
      @confirm="confirmDeleteKey"
      @cancel="deletingKey = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import LiquidButton from "@/components/common/LiquidButton.vue";
import { computed, onMounted, onUnmounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";

import { adminAPI } from "@/api/admin";
import type {
  AdminAPIKeyListItem,
  AdminAPIKeyListSummary,
  AdminAPIKeySort,
  AdminAPIKeyStatus,
} from "@/api/admin/apiKeys";
import type { SimpleUser } from "@/api/admin/usage";
import { useAppStore } from "@/stores/app";
import { getPersistedPageSize } from "@/composables/usePersistedPageSize";
import { formatDateOnly, formatDateTime } from "@/utils/format";
import type { Column } from "@/components/common/types";
import type { SelectOption } from "@/components/common/Select.vue";

import AppLayout from "@/components/layout/AppLayout.vue";
import AdminPageHeader from "@/components/admin/AdminPageHeader.vue";
import BaseDialog from "@/components/common/BaseDialog.vue";
import ConfirmDialog from "@/components/common/ConfirmDialog.vue";
import TablePageLayout from "@/components/layout/TablePageLayout.vue";
import DataTable from "@/components/common/DataTable.vue";
import GroupBadge from "@/components/common/GroupBadge.vue";
import Pagination from "@/components/common/Pagination.vue";
import SearchInput from "@/components/common/SearchInput.vue";
import Select from "@/components/common/Select.vue";
import Icon from "@/components/icons/Icon.vue";

const { t } = useI18n();
const route = useRoute();
const appStore = useAppStore();

const emptySummary = (): AdminAPIKeyListSummary => ({
  total: 0,
  active: 0,
  inactive: 0,
  expired: 0,
  last_30_days_actual_cost: 0,
});

const items = ref<AdminAPIKeyListItem[]>([]);
const summary = ref<AdminAPIKeyListSummary>(emptySummary());
const loading = ref(false);
const filters = reactive<{
  search: string;
  user_id?: number;
  group_id: number | null;
  status: AdminAPIKeyStatus | null;
}>({
  search: "",
  user_id: undefined,
  group_id: null,
  status: null,
});
const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
});
const sortBy = ref<AdminAPIKeySort>("last_used_at");
const sortOrder = ref<"asc" | "desc">("desc");
const mutatingKeyIDs = ref<Set<number>>(new Set());
const editingGroupKey = ref<AdminAPIKeyListItem | null>(null);
const editingGroupID = ref<number>(0);
const deletingKey = ref<AdminAPIKeyListItem | null>(null);
let listAbortController: AbortController | null = null;
let listRequestID = 0;

const userFilterRef = ref<HTMLElement | null>(null);
const userKeyword = ref("");
const userResults = ref<SimpleUser[]>([]);
const userSearchLoading = ref(false);
const showUserDropdown = ref(false);
let userSearchTimer: ReturnType<typeof setTimeout> | null = null;

const groupOptions = ref<SelectOption[]>([
  { value: null, label: t("admin.apiKeyInventory.allGroups") },
]);
const actionGroupOptions = computed<SelectOption[]>(() => [
  { value: 0, label: t("admin.apiKeyInventory.noGroup") },
  ...groupOptions.value.filter((option) => Number(option.value) > 0),
]);
const statusOptions = computed<SelectOption[]>(() => [
  { value: null, label: t("admin.apiKeyInventory.allStatuses") },
  { value: "active", label: t("admin.apiKeyInventory.statusActive") },
  { value: "inactive", label: t("admin.apiKeyInventory.statusInactive") },
  { value: "expired", label: t("admin.apiKeyInventory.statusExpired") },
]);
const sortOptions = computed<SelectOption[]>(() => [
  { value: "last_used_at", label: t("admin.apiKeyInventory.sortLastUsed") },
  {
    value: "today_actual_cost",
    label: t("admin.apiKeyInventory.sortTodayCost"),
  },
  {
    value: "last_30_days_actual_cost",
    label: t("admin.apiKeyInventory.sortLast30DaysCost"),
  },
  {
    value: "total_actual_cost",
    label: t("admin.apiKeyInventory.sortTotalCost"),
  },
  { value: "created_at", label: t("admin.apiKeyInventory.sortCreatedAt") },
]);

const columns = computed<Column[]>(() => [
  {
    key: "user",
    label: t("admin.apiKeyInventory.user"),
    class: "w-[136px] max-w-[136px] !px-2",
  },
  {
    key: "key",
    label: t("admin.apiKeyInventory.key"),
    class: "w-[136px] max-w-[136px] !px-2",
  },
  {
    key: "group",
    label: t("admin.apiKeyInventory.group"),
    class: "w-56 max-w-56 !px-2",
  },
  {
    key: "total_actual_cost",
    label: t("admin.apiKeyInventory.costBreakdown"),
    class: "w-32 max-w-32 !px-2",
    sortable: true,
  },
  {
    key: "last_used_at",
    label: t("admin.apiKeyInventory.activity"),
    class: "w-32 max-w-32 !px-2",
    sortable: true,
  },
  {
    key: "created_at",
    label: t("admin.apiKeyInventory.lifecycle"),
    class: "w-28 max-w-28 !px-2",
    sortable: true,
  },
  {
    key: "actions",
    label: t("admin.apiKeyInventory.actions"),
    class: "w-32 min-w-32 !px-2 text-right",
  },
]);

const formatMoney = (value: number | null | undefined): string => {
  const amount = Number(value || 0);
  return `$${amount.toFixed(amount > 0 && amount < 0.01 ? 6 : 2)}`;
};

const summaryMetrics = computed(() => [
  {
    key: "total",
    label: t("admin.apiKeyInventory.summaryTotal"),
    value: summary.value.total.toLocaleString(),
    testId: "summary-total",
  },
  {
    key: "active",
    label: t("admin.apiKeyInventory.summaryActive"),
    value: summary.value.active.toLocaleString(),
    testId: "summary-active",
  },
  {
    key: "inactive",
    label: t("admin.apiKeyInventory.summaryInactive"),
    value: summary.value.inactive.toLocaleString(),
    testId: "summary-inactive",
  },
  {
    key: "expired",
    label: t("admin.apiKeyInventory.summaryExpired"),
    value: summary.value.expired.toLocaleString(),
    testId: "summary-expired",
  },
  {
    key: "cost",
    label: t("admin.apiKeyInventory.summaryCost"),
    value: formatMoney(summary.value.last_30_days_actual_cost),
    testId: "summary-cost",
  },
]);

const loadKeys = async () => {
  listAbortController?.abort();
  const controller = new AbortController();
  listAbortController = controller;
  const requestID = ++listRequestID;
  loading.value = true;
  try {
    const result = await adminAPI.apiKeys.list(
      {
        page: pagination.page,
        page_size: pagination.page_size,
        search: filters.search.trim() || undefined,
        user_id: filters.user_id,
        group_id: filters.group_id || undefined,
        status: filters.status || undefined,
        sort_by: sortBy.value,
        sort_order: sortOrder.value,
      },
      { signal: controller.signal },
    );
    if (requestID !== listRequestID) return;
    items.value = result.items;
    summary.value = result.summary || emptySummary();
    pagination.total = result.total;
    pagination.page = result.page;
    pagination.page_size = result.page_size;
  } catch (error) {
    if (controller.signal.aborted) return;
    items.value = [];
    summary.value = emptySummary();
    pagination.total = 0;
    appStore.showError(t("admin.apiKeyInventory.loadFailed"));
  } finally {
    if (requestID === listRequestID) loading.value = false;
  }
};

const loadGroups = async () => {
  try {
    const groups = await adminAPI.groups.getAll();
    groupOptions.value = [
      { value: null, label: t("admin.apiKeyInventory.allGroups") },
      ...groups.map((group) => ({ value: group.id, label: group.name })),
    ];
  } catch {
    groupOptions.value = [
      { value: null, label: t("admin.apiKeyInventory.allGroups") },
    ];
  }
};

const applyFilters = () => {
  pagination.page = 1;
  void loadKeys();
};

const debouncedUserSearch = () => {
  filters.user_id = undefined;
  if (userSearchTimer) clearTimeout(userSearchTimer);
  const keyword = userKeyword.value.trim();
  if (!keyword) {
    userResults.value = [];
    applyFilters();
    return;
  }
  userSearchTimer = setTimeout(async () => {
    userSearchLoading.value = true;
    try {
      userResults.value = await adminAPI.usage.searchUsers(keyword);
      showUserDropdown.value = true;
    } catch {
      userResults.value = [];
    } finally {
      userSearchLoading.value = false;
    }
  }, 250);
};

const selectUser = (user: SimpleUser) => {
  filters.user_id = user.id;
  userKeyword.value = user.email;
  showUserDropdown.value = false;
  applyFilters();
};

const clearUserFilter = () => {
  filters.user_id = undefined;
  userKeyword.value = "";
  userResults.value = [];
  showUserDropdown.value = false;
  applyFilters();
};

const handleSort = (key: string, order: "asc" | "desc") => {
  const allowed: AdminAPIKeySort[] = [
    "created_at",
    "last_used_at",
    "today_actual_cost",
    "last_30_days_actual_cost",
    "total_actual_cost",
  ];
  if (!allowed.includes(key as AdminAPIKeySort)) return;
  sortBy.value = key as AdminAPIKeySort;
  sortOrder.value = order;
  pagination.page = 1;
  void loadKeys();
};

const applySortSelection = () => {
  pagination.page = 1;
  void loadKeys();
};

const toggleSortOrder = () => {
  sortOrder.value = sortOrder.value === "desc" ? "asc" : "desc";
  pagination.page = 1;
  void loadKeys();
};

const handlePageChange = (page: number) => {
  pagination.page = page;
  void loadKeys();
};

const handlePageSizeChange = (pageSize: number) => {
  pagination.page_size = pageSize;
  pagination.page = 1;
  void loadKeys();
};

const statusLabel = (status: AdminAPIKeyStatus) =>
  ({
    active: t("admin.apiKeyInventory.statusActive"),
    inactive: t("admin.apiKeyInventory.statusInactive"),
    expired: t("admin.apiKeyInventory.statusExpired"),
  })[status] || status;

const statusClass = (status: AdminAPIKeyStatus) =>
  ({
    active: "badge-success",
    inactive: "badge-gray",
    expired: "badge-danger",
  })[status] || "badge-gray";

const formatCompactTimestamp = (value?: string | null) =>
  value
    ? formatDateTime(value, {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        hour12: false,
      })
    : t("admin.apiKeyInventory.neverUsed");

const isMutating = (keyID: number) => mutatingKeyIDs.value.has(keyID);

const setMutating = (keyID: number, active: boolean) => {
  const next = new Set(mutatingKeyIDs.value);
  if (active) next.add(keyID);
  else next.delete(keyID);
  mutatingKeyIDs.value = next;
};

const setKeyEnabled = async (key: AdminAPIKeyListItem, enabled: boolean) => {
  if (key.status === "expired" || isMutating(key.id)) return;
  setMutating(key.id, true);
  try {
    await adminAPI.apiKeys.setEnabled(key.id, enabled);
    appStore.showSuccess(
      t(
        enabled
          ? "admin.apiKeyInventory.enableSuccess"
          : "admin.apiKeyInventory.disableSuccess",
      ),
    );
    await loadKeys();
  } catch (error: any) {
    appStore.showError(
      error?.message || t("admin.apiKeyInventory.mutationFailed"),
    );
  } finally {
    setMutating(key.id, false);
  }
};

const openGroupDialog = (key: AdminAPIKeyListItem) => {
  editingGroupKey.value = key;
  editingGroupID.value = key.group?.id || 0;
};

const closeGroupDialog = () => {
  if (editingGroupKey.value && isMutating(editingGroupKey.value.id)) return;
  editingGroupKey.value = null;
  editingGroupID.value = 0;
};

const saveGroupChange = async () => {
  const key = editingGroupKey.value;
  if (!key || isMutating(key.id)) return;
  const groupID = Number(editingGroupID.value) || 0;
  setMutating(key.id, true);
  try {
    await adminAPI.apiKeys.updateApiKeyGroup(key.id, groupID || null);
    appStore.showSuccess(t("admin.apiKeyInventory.changeGroupSuccess"));
    editingGroupKey.value = null;
    editingGroupID.value = 0;
    await loadKeys();
  } catch (error: any) {
    appStore.showError(
      error?.message || t("admin.apiKeyInventory.mutationFailed"),
    );
  } finally {
    setMutating(key.id, false);
  }
};

const openDeleteDialog = (key: AdminAPIKeyListItem) => {
  deletingKey.value = key;
};

const confirmDeleteKey = async () => {
  const key = deletingKey.value;
  if (!key || isMutating(key.id)) return;
  setMutating(key.id, true);
  try {
    await adminAPI.apiKeys.deleteApiKey(key.id);
    appStore.showSuccess(t("admin.apiKeyInventory.deleteSuccess"));
    deletingKey.value = null;
    await loadKeys();
  } catch (error: any) {
    appStore.showError(
      error?.message || t("admin.apiKeyInventory.mutationFailed"),
    );
  } finally {
    setMutating(key.id, false);
  }
};

const handleOutsideClick = (event: MouseEvent) => {
  if (
    userFilterRef.value &&
    !userFilterRef.value.contains(event.target as Node)
  ) {
    showUserDropdown.value = false;
  }
};

onMounted(() => {
  const userID = Number(
    Array.isArray(route.query.user_id)
      ? route.query.user_id[0]
      : route.query.user_id,
  );
  if (Number.isSafeInteger(userID) && userID > 0) {
    filters.user_id = userID;
    userKeyword.value = `#${userID}`;
  }
  document.addEventListener("click", handleOutsideClick);
  void Promise.all([loadGroups(), loadKeys()]);
});

onUnmounted(() => {
  listAbortController?.abort();
  if (userSearchTimer) clearTimeout(userSearchTimer);
  document.removeEventListener("click", handleOutsideClick);
});
</script>

<style scoped>
.admin-b1-outline-scope :deep(.card),
.admin-b1-outline-scope :deep(.table-scroll-container),
.admin-b1-outline-scope :deep(.table-wrapper) {
  background: transparent !important;
  border-color: var(--ssxz-border) !important;
  box-shadow: none !important;
}

.api-key-inventory-page :deep(.table-wrapper .table-body > tr + tr) {
  border-top-color: rgb(148 163 184 / 0.1) !important;
}

.api-key-inventory-page :deep(.table-wrapper .table-body td) {
  padding-top: 1.25rem;
  padding-bottom: 1.25rem;
  border-bottom-color: transparent !important;
  line-height: 1.45;
}

.api-key-inventory-page :deep(.table-wrapper .table-body > tr:hover) {
  background: color-mix(in srgb, var(--ssxz-surface-raised) 62%, transparent);
}

.inventory-group-badge {
  min-height: 2rem;
  gap: 0.5rem !important;
  padding: 0.375rem 0.625rem !important;
}

.api-key-inventory-page :deep(.inventory-group-badge > span:last-child) {
  margin-left: 0.125rem;
  padding: 0.25rem 0.5rem !important;
}

.inventory-status-badge {
  min-height: 1.75rem;
  gap: 0.375rem;
  padding: 0.25rem 0.65rem !important;
}
</style>
