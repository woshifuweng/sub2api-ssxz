<template>
  <AppLayout>
    <AdminPageHeader title="推广返利" description="邀请推广数据与返利设置">
      <template #actions>
        <LiquidButton
          :disabled="loading"
          @click="loadEntries"
          variant="outline"
          size="sm"
        >
          <Icon
            name="refresh"
            size="sm"
            :class="loading ? 'animate-spin' : ''"
          />
          刷新
        </LiquidButton>
      </template>
    </AdminPageHeader>

    <div class="space-y-6">
      <section class="card admin-b3-outline-card">
        <div class="grid gap-6 p-6 lg:grid-cols-[minmax(0,1fr)_320px]">
          <div
            class="rounded-lg border border-emerald-100 bg-emerald-50 px-4 py-3 text-sm text-emerald-800 dark:border-emerald-900/40 dark:bg-emerald-950/30 dark:text-emerald-200 lg:col-span-2"
          >
            数据来自已有邀请关系、订单和返利账本；普通用户只看到自己的推广码、邀请记录和可结算额度。
          </div>

          <div class="space-y-4">
            <div>
              <label class="input-label">查找推广用户</label>
              <div class="flex flex-col gap-2 sm:flex-row">
                <input
                  v-model.trim="lookupKeyword"
                  class="input"
                  placeholder="输入邮箱、用户名或用户 ID"
                  @keyup.enter="lookupUsers"
                />
                <LiquidButton
                  data-testid="affiliate-user-search"
                  class="shrink-0"
                  :disabled="lookupLoading || !lookupKeyword"
                  @click="lookupUsers"
                  size="sm"
                >
                  搜索用户
                </LiquidButton>
              </div>
            </div>

            <div
              v-if="lookupResults.length > 0"
              class="divide-y divide-gray-100 rounded-lg border border-gray-200 dark:divide-dark-700 dark:border-dark-700"
            >
              <LiquidButton
                v-for="user in lookupResults"
                :key="user.id"
                class="flex w-full items-center justify-between gap-3 px-4 py-3 text-left transition-colors hover:bg-gray-50 dark:hover:bg-dark-700"
                @click="selectLookupUser(user)"
                variant="plain"
                size="sm"
              >
                <span>
                  <span
                    class="block text-sm font-medium text-gray-900 dark:text-white"
                  >
                    {{ user.email || user.username || `用户 ${user.id}` }}
                  </span>
                  <span class="text-xs text-gray-500 dark:text-gray-400">
                    ID {{ user.id
                    }}{{ user.username ? ` · ${user.username}` : "" }}
                  </span>
                </span>
                <span class="text-xs text-primary-600 dark:text-primary-400"
                  >选择</span
                >
              </LiquidButton>
            </div>

            <div
              v-else-if="lookupSearched"
              class="rounded-lg border border-dashed border-gray-300 px-4 py-3 text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400"
            >
              没有找到匹配用户。
            </div>
          </div>

          <form
            class="space-y-4 rounded-lg border border-gray-200 p-4 dark:border-dark-700"
            @submit.prevent="saveSelectedUser"
          >
            <div>
              <div class="text-sm font-medium text-gray-900 dark:text-white">
                当前配置用户
              </div>
              <div class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                <template v-if="selectedUser">
                  {{
                    selectedUser.email ||
                    selectedUser.username ||
                    `用户 ${selectedUser.id}`
                  }}
                </template>
                <template v-else>先从搜索结果或下方表格选择用户。</template>
              </div>
            </div>

            <div>
              <label class="input-label">推广码</label>
              <input
                v-model.trim="form.aff_code"
                class="input font-mono uppercase"
                placeholder="例如 SSXZ2026"
                :disabled="!selectedUser"
              />
              <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                留空不会修改推广码；后端会校验格式和唯一性。
              </p>
            </div>

            <div>
              <label class="input-label">专属返利比例（%）</label>
              <input
                v-model="form.aff_rebate_rate_percent"
                type="number"
                min="0"
                max="100"
                step="0.01"
                class="input"
                placeholder="留空 = 不设专属比例，跟随全局"
                :disabled="
                  !selectedUser || form.clear_rebate_rate || overviewLoading
                "
              />
              <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                当前状态：{{ selectedRateStateText }}
              </p>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                留空保存 = 清除专属比例，跟随全局{{ globalRateSuffix }}；填 0 =
                该用户返利关闭（0%），不是跟随全局。
              </p>
              <label
                class="mt-2 flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300"
              >
                <input
                  v-model="form.clear_rebate_rate"
                  type="checkbox"
                  class="rounded border-gray-300"
                  :disabled="!selectedUser"
                />
                清除专属比例，改用全局比例
              </label>
            </div>

            <div class="flex flex-wrap gap-2">
              <LiquidButton
                :disabled="saving || !selectedUser"
                type="submit"
                size="default"
              >
                保存设置
              </LiquidButton>
              <LiquidButton
                class="text-red-600 dark:text-red-400"
                type="button"
                :disabled="saving || !selectedUser"
                @click="clearSelectedUser"
                variant="outline"
                size="sm"
              >
                清除自定义
              </LiquidButton>
            </div>
          </form>
        </div>
      </section>

      <TablePageLayout class="admin-b3-outline-scope">
        <template #filters>
          <div
            class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"
          >
            <input
              v-model.trim="searchQuery"
              class="input sm:max-w-xs"
              placeholder="搜索自定义推广用户"
              @keyup.enter="handleSearch"
            />
            <div class="flex gap-2">
              <LiquidButton
                :disabled="loading"
                @click="handleSearch"
                variant="outline"
                size="sm"
                >搜索</LiquidButton
              >
              <LiquidButton
                :disabled="loading"
                @click="resetSearch"
                variant="outline"
                size="sm"
                >重置</LiquidButton
              >
            </div>
          </div>
        </template>

        <template #table>
          <DataTable :columns="columns" :data="entries" :loading="loading">
            <template #cell-user="{ row }">
              <div>
                <div class="text-sm font-medium text-gray-900 dark:text-white">
                  {{ row.email || row.username || `用户 ${row.user_id}` }}
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  ID {{ row.user_id }}
                </div>
              </div>
            </template>

            <template #cell-aff_code="{ value, row }">
              <div class="flex flex-col">
                <code
                  class="font-mono text-sm text-gray-900 dark:text-gray-100"
                  >{{ value }}</code
                >
                <span class="text-xs text-gray-500 dark:text-gray-400">
                  {{ row.aff_code_custom ? "自定义" : "系统生成" }}
                </span>
              </div>
            </template>

            <template #cell-aff_rebate_rate_percent="{ value }">
              <span class="text-sm text-gray-700 dark:text-gray-200">
                {{ formatRate(value) }}
              </span>
            </template>

            <template #cell-aff_count="{ value }">
              <span class="text-sm text-gray-700 dark:text-gray-200"
                >{{ value }} 人</span
              >
            </template>

            <template #cell-invitee_recharge_total="{ value }">
              <span class="text-sm text-gray-700 dark:text-gray-200">{{
                formatQuota(value)
              }}</span>
            </template>

            <template #cell-accrued_rebate_total="{ value }">
              <span class="text-sm text-gray-700 dark:text-gray-200">{{
                formatQuota(value)
              }}</span>
            </template>

            <template #cell-aff_frozen_quota="{ value }">
              <span class="text-sm text-gray-700 dark:text-gray-200">{{
                formatQuota(value)
              }}</span>
            </template>

            <template #cell-aff_quota="{ value }">
              <span class="text-sm text-gray-700 dark:text-gray-200">{{
                formatQuota(value)
              }}</span>
            </template>

            <template #cell-transferred_rebate_total="{ value }">
              <span class="text-sm text-gray-700 dark:text-gray-200">{{
                formatQuota(value)
              }}</span>
            </template>

            <template #cell-actions="{ row }">
              <div class="flex items-center gap-2">
                <LiquidButton
                  @click="copyAffiliateLink(row)"
                  variant="outline"
                  size="sm"
                >
                  复制链接
                </LiquidButton>
                <LiquidButton
                  @click="selectEntry(row)"
                  variant="outline"
                  size="sm"
                  >编辑</LiquidButton
                >
                <LiquidButton
                  class="text-red-600 dark:text-red-400"
                  @click="clearEntry(row)"
                  variant="outline"
                  size="sm"
                >
                  清除
                </LiquidButton>
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
    </div>
  </AppLayout>
</template>

<style scoped>
.admin-b3-outline-card {
  background: transparent !important;
  border: 1px solid var(--ssxz-border) !important;
  box-shadow: none !important;
}

.admin-b3-outline-scope :deep(.table-scroll-container),
.admin-b3-outline-scope :deep(.table-wrapper),
.admin-b3-outline-scope :deep(.table-wrapper table),
.admin-b3-outline-scope :deep(.table-wrapper tbody) {
  background: transparent !important;
  border-color: var(--ssxz-border) !important;
  box-shadow: none !important;
}

.admin-b3-outline-scope :deep(thead),
.admin-b3-outline-scope :deep(.table-header) {
  background: var(--ssxz-surface-raised) !important;
}
</style>

<script setup lang="ts">
import LiquidButton from "@/components/common/LiquidButton.vue";
import { computed, onMounted, reactive, ref } from "vue";
import { useRoute } from "vue-router";
import { adminAPI } from "@/api/admin";
import { useAppStore } from "@/stores/app";
import { getPersistedPageSize } from "@/composables/usePersistedPageSize";
import { useClipboard } from "@/composables/useClipboard";
import type {
  AdminAffiliateEntry,
  AffiliateUserSummary,
  UpdateAffiliateUserRequest,
} from "@/api/admin/affiliate";
import type { Column } from "@/components/common/types";
import AppLayout from "@/components/layout/AppLayout.vue";
import AdminPageHeader from "@/components/admin/AdminPageHeader.vue";
import TablePageLayout from "@/components/layout/TablePageLayout.vue";
import DataTable from "@/components/common/DataTable.vue";
import Pagination from "@/components/common/Pagination.vue";
import Icon from "@/components/icons/Icon.vue";

type SelectedAffiliateUser =
  | AffiliateUserSummary
  | {
      id: number;
      email: string;
      username: string;
    };

const appStore = useAppStore();
const route = useRoute();
const { copyToClipboard } = useClipboard();

const loading = ref(false);
const saving = ref(false);
const lookupLoading = ref(false);
const lookupSearched = ref(false);
const searchQuery = ref("");
const lookupKeyword = ref("");
const entries = ref<AdminAffiliateEntry[]>([]);
const lookupResults = ref<AffiliateUserSummary[]>([]);
const selectedUser = ref<SelectedAffiliateUser | null>(null);

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0,
});

/**
 * The rate field is deliberately `string | number`.
 *
 * Vue's v-model casts the value for any `type="number"` input (the `.number` modifier is
 * not even needed), and an emptied input casts to `''`, not to a number. The old code then
 * did `Number(form.aff_rebate_rate_percent)`, and `Number('')` is `0` — so "I did not set an
 * exclusive rate" was silently submitted as "I set it to 0%". The backend correctly stores
 * that 0 as an explicit override (0% is a valid business value meaning "rebate disabled"),
 * which is a different state from NULL (follow the global rate). Keeping the raw value and
 * normalizing through `rateInputRaw` is what lets us tell the two apart.
 */
const form = reactive({
  aff_code: "",
  aff_rebate_rate_percent: "" as string | number,
  clear_rebate_rate: false,
});

/** Normalizes the rate input to a trimmed string; '' means "no exclusive rate". */
function rateInputRaw() {
  const value = form.aff_rebate_rate_percent;
  return value == null ? "" : String(value).trim();
}

/** Global fallback rate (percent), null while unknown. */
const globalRatePercent = ref<number | null>(null);
/** Whether the selected user's current rate is known, and whether it is an explicit override. */
const rateBaselineKnown = ref(false);
const rateBaselineCustom = ref(false);
const rateBaselineValue = ref<number | null>(null);
const overviewLoading = ref(false);

const columns = computed<Column[]>(() => [
  { key: "user", label: "用户" },
  { key: "aff_code", label: "推广码" },
  { key: "aff_count", label: "邀请人数" },
  { key: "invitee_recharge_total", label: "被邀充值" },
  { key: "accrued_rebate_total", label: "已产生返利" },
  { key: "aff_frozen_quota", label: "待确认" },
  { key: "aff_quota", label: "可结算" },
  { key: "transferred_rebate_total", label: "已转余额" },
  { key: "aff_rebate_rate_percent", label: "专属比例" },
  { key: "actions", label: "操作" },
]);

function formatPercent(value: number) {
  return `${value
    .toFixed(2)
    .replace(/\.00$/, "")
    .replace(/(\.\d)0$/, "$1")}%`;
}

const globalRateText = computed(() =>
  globalRatePercent.value == null
    ? "全局比例"
    : formatPercent(globalRatePercent.value),
);

/** " 5%" when the global rate is known, "" otherwise — keeps hint text readable either way. */
const globalRateSuffix = computed(() =>
  globalRatePercent.value == null
    ? ""
    : ` ${formatPercent(globalRatePercent.value)}`,
);

/**
 * Table cell: NULL (field omitted by the API) means "no exclusive rate, follows the global
 * rate". An explicit 0 means the rebate is switched off for that user. These are different
 * states and must not render the same way.
 */
function formatRate(value: number | null | undefined) {
  if (value == null) return `未设置（跟随全局 ${globalRateText.value}）`;
  if (value === 0) return "0%（已关闭返利）";
  return formatPercent(value);
}

const selectedRateStateText = computed(() => {
  if (!selectedUser.value) return "";
  if (overviewLoading.value) return "正在读取当前配置…";
  if (!rateBaselineKnown.value) return "当前配置未知，留空不会改动专属比例";
  if (!rateBaselineCustom.value)
    return `当前未设置专属比例，跟随全局 ${globalRateText.value}`;
  if (rateBaselineValue.value === 0) return "当前专属比例 0%，该用户返利已关闭";
  return `当前专属比例 ${formatPercent(rateBaselineValue.value ?? 0)}`;
});

function formatQuota(value: number | null | undefined) {
  return `${Number(value ?? 0).toFixed(2)} 额度`;
}

function buildAffiliateLink(code: string) {
  const normalizedCode = String(code || "").trim();
  if (!normalizedCode) return "";
  const path = `/register?aff=${encodeURIComponent(normalizedCode)}`;
  if (typeof window === "undefined") return path;
  return `${window.location.origin}${path}`;
}

async function copyAffiliateLink(row: AdminAffiliateEntry) {
  const link = buildAffiliateLink(row.aff_code);
  if (!link) {
    appStore.showError("该用户暂无可用推广码");
    return;
  }
  await copyToClipboard(link, "推广链接已复制");
}

function resetForm() {
  form.aff_code = "";
  form.aff_rebate_rate_percent = "";
  form.clear_rebate_rate = false;
  rateBaselineKnown.value = false;
  rateBaselineCustom.value = false;
  rateBaselineValue.value = null;
}

/**
 * Record the selected user's current rate state and prefill the input from it.
 *
 * Prefilling matters: an empty input now means "clear the exclusive rate", so an admin who
 * only wants to change the aff code must not silently wipe an existing rate.
 */
function applyRateBaseline(custom: boolean, value: number | null) {
  rateBaselineKnown.value = true;
  rateBaselineCustom.value = custom;
  rateBaselineValue.value = custom ? value : null;
  form.aff_rebate_rate_percent = custom && value != null ? String(value) : "";
  form.clear_rebate_rate = false;
}

const getSingleQueryValue = (
  value: string | null | Array<string | null> | undefined,
): string | undefined => {
  if (Array.isArray(value))
    return value.find(
      (item): item is string => typeof item === "string" && item.length > 0,
    );
  return typeof value === "string" && value.length > 0 ? value : undefined;
};

const getInitialInvestigationKeyword = () =>
  getSingleQueryValue(route.query.search) ||
  getSingleQueryValue(route.query.user) ||
  getSingleQueryValue(route.query.user_id);

function setPagination(response: {
  total: number;
  page: number;
  page_size: number;
  pages: number;
}) {
  pagination.total = response.total;
  pagination.page = response.page;
  pagination.page_size = response.page_size;
  pagination.pages = response.pages;
}

async function loadEntries() {
  loading.value = true;
  try {
    const response = await adminAPI.affiliate.listUsers(
      pagination.page,
      pagination.page_size,
      searchQuery.value,
    );
    entries.value = response.items;
    setPagination(response);
  } catch (error: any) {
    appStore.showError(error?.message || "推广返利列表加载失败");
  } finally {
    loading.value = false;
  }
}

async function lookupUsers() {
  if (!lookupKeyword.value) return;
  lookupLoading.value = true;
  lookupSearched.value = true;
  try {
    lookupResults.value = await adminAPI.affiliate.lookupUsers(
      lookupKeyword.value,
    );
  } catch (error: any) {
    appStore.showError(error?.message || "用户搜索失败");
  } finally {
    lookupLoading.value = false;
  }
}

/**
 * Reads the selected user's current exclusive rate so that an empty input can safely mean
 * "clear the exclusive rate". Without a known baseline, a promo-code-only save would have
 * to guess, and guessing is what caused rates to be silently overwritten.
 */
async function loadRateBaseline(userId: number) {
  overviewLoading.value = true;
  rateBaselineKnown.value = false;
  rateBaselineCustom.value = false;
  rateBaselineValue.value = null;
  try {
    const overview = await adminAPI.affiliate.getUserOverview(userId);
    applyRateBaseline(
      Boolean(overview?.rebate_rate_custom),
      overview?.rebate_rate_percent ?? null,
    );
  } catch (error: any) {
    if (error?.status === 404) {
      // No affiliate row yet: no exclusive rate, follows the global rate.
      applyRateBaseline(false, null);
    } else {
      appStore.showError(error?.message || "读取当前返利配置失败");
    }
  } finally {
    overviewLoading.value = false;
  }
}

function selectLookupUser(user: AffiliateUserSummary) {
  selectedUser.value = user;
  resetForm();
  void loadRateBaseline(user.id);
}

function selectEntry(row: AdminAffiliateEntry) {
  selectedUser.value = {
    id: row.user_id,
    email: row.email,
    username: row.username,
  };
  form.aff_code = row.aff_code || "";
  form.clear_rebate_rate = false;
  // The admin list returns the raw column: omitted/null = no exclusive rate, 0 = disabled.
  applyRateBaseline(
    row.aff_rebate_rate_percent != null,
    row.aff_rebate_rate_percent ?? null,
  );
  lookupKeyword.value = row.email || row.username || String(row.user_id);
}

/** Returns null when the input is invalid (the error has already been surfaced). */
function buildPayload(): UpdateAffiliateUserRequest | null {
  const payload: UpdateAffiliateUserRequest = {};
  if (form.aff_code) {
    payload.aff_code = form.aff_code.toUpperCase();
  }

  const raw = rateInputRaw();
  if (form.clear_rebate_rate || raw === "") {
    // An empty field means "no exclusive rate" -> clear (NULL), never 0.
    // Only send the clear when there is actually an override to remove, so that a
    // promo-code-only save on an unknown baseline leaves the rate untouched.
    if (
      form.clear_rebate_rate ||
      (rateBaselineKnown.value && rateBaselineCustom.value)
    ) {
      payload.clear_rebate_rate = true;
    }
    return payload;
  }

  const parsed = Number(raw);
  if (!Number.isFinite(parsed) || parsed < 0 || parsed > 100) {
    appStore.showError("专属返利比例需在 0 到 100 之间");
    return null;
  }
  // 0 is intentional and is sent as 0: it disables this user's rebate.
  payload.aff_rebate_rate_percent = parsed;
  return payload;
}

async function saveSelectedUser() {
  if (!selectedUser.value) return;
  const payload = buildPayload();
  if (!payload) return;
  if (Object.keys(payload).length === 0) {
    appStore.showError("没有需要保存的改动");
    return;
  }
  saving.value = true;
  try {
    await adminAPI.affiliate.updateUserSettings(selectedUser.value.id, payload);
    // Re-sync the baseline so a second save cannot act on a stale "current rate".
    if (payload.clear_rebate_rate) {
      applyRateBaseline(false, null);
    } else if (payload.aff_rebate_rate_percent != null) {
      applyRateBaseline(true, payload.aff_rebate_rate_percent);
    }
    appStore.showSuccess("推广返利设置已保存");
    await loadEntries();
  } catch (error: any) {
    appStore.showError(error?.message || "保存失败");
  } finally {
    saving.value = false;
  }
}

async function clearSelectedUser() {
  if (!selectedUser.value) return;
  saving.value = true;
  try {
    await adminAPI.affiliate.clearUserSettings(selectedUser.value.id);
    resetForm();
    applyRateBaseline(false, null);
    appStore.showSuccess("推广返利自定义设置已清除");
    await loadEntries();
  } catch (error: any) {
    appStore.showError(error?.message || "清除失败");
  } finally {
    saving.value = false;
  }
}

async function clearEntry(row: AdminAffiliateEntry) {
  saving.value = true;
  try {
    await adminAPI.affiliate.clearUserSettings(row.user_id);
    appStore.showSuccess("推广返利自定义设置已清除");
    await loadEntries();
  } catch (error: any) {
    appStore.showError(error?.message || "清除失败");
  } finally {
    saving.value = false;
  }
}

function handleSearch() {
  pagination.page = 1;
  loadEntries();
}

function resetSearch() {
  searchQuery.value = "";
  pagination.page = 1;
  loadEntries();
}

function handlePageChange(page: number) {
  pagination.page = page;
  loadEntries();
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize;
  pagination.page = 1;
  loadEntries();
}

/**
 * Read the global rebate rate so the UI can spell out what "未设置" actually falls back to.
 * Non-fatal: without it we just render the generic "全局比例" wording.
 */
async function loadGlobalRate() {
  try {
    const settings = await adminAPI.settings.getSettings();
    const rate = Number(settings?.affiliate_rebate_rate);
    globalRatePercent.value = Number.isFinite(rate) ? rate : null;
  } catch {
    globalRatePercent.value = null;
  }
}

onMounted(() => {
  const keyword = getInitialInvestigationKeyword();
  if (keyword) {
    searchQuery.value = keyword;
    lookupKeyword.value = keyword;
    void lookupUsers();
  }
  void loadGlobalRate();
  loadEntries();
});
</script>
