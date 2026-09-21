<template>
  <div
    class="admin-b5-outline-panel flex h-full min-h-0 flex-col bg-white dark:bg-dark-900"
  >
    <!-- Loading State -->
    <div v-if="loading" class="flex flex-1 items-center justify-center py-10">
      <div
        class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"
      ></div>
    </div>

    <!-- Table Container -->
    <div v-else class="flex min-h-0 flex-1 flex-col">
      <div
        class="min-h-0 flex-1 overflow-auto border-b border-gray-200 dark:border-dark-700"
      >
        <table class="w-full border-separate border-spacing-0">
          <thead class="sticky top-0 z-10 bg-gray-50 dark:bg-dark-800">
            <tr>
              <th v-for="column in columns" :key="column.key"
                class="border-b border-gray-200 px-4 py-2.5 text-left text-[11px] font-bold uppercase tracking-wider text-gray-500 dark:border-dark-700 dark:text-dark-400"
                :class="{ 'cursor-pointer': column.sortable }"
                :aria-sort="column.sortable ? (sortKey === column.key ? (sortOrder === 'asc' ? 'ascending' : 'descending') : 'none') : undefined"
                @click="column.sortable && sortColumn(column.key)"
              >
                <button v-if="column.sortable" type="button" class="text-left" @click.stop="sortColumn(column.key)">
                  {{ column.label }}
                  <svg v-if="sortKey === column.key" aria-hidden="true" class="inline h-3 w-3" :class="{ 'rotate-180': sortOrder === 'desc' }" viewBox="0 0 12 12" fill="currentColor"><path d="M6 2 2 7h8z" /></svg>
                </button>
                <span v-else>{{ column.label }}</span>
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-if="rows.length === 0">
              <td
                :colspan="Math.max(1, columns.length)"
                class="py-12 text-center text-sm text-gray-400 dark:text-dark-500"
              >
                {{ t("admin.ops.errorLog.noErrors") }}
              </td>
            </tr>

            <tr
              v-for="log in rows"
              :key="log.id"
              class="group cursor-pointer transition-colors hover:bg-gray-50/80 dark:hover:bg-dark-800/50"
              @click="emit('openErrorDetail', log.id)"
            >
              <template v-for="column in columns" :key="column.key">
              <!-- Time -->
              <td v-if="column.key === 'created_at'" class="whitespace-nowrap px-4 py-2">
                <el-tooltip
                  :content="log.request_id || log.client_request_id"
                  placement="top"
                  :show-after="500"
                >
                  <span
                    class="font-mono text-xs font-medium text-gray-900 dark:text-gray-200"
                  >
                    {{ formatDateTime(log.created_at).split(" ")[1] }}
                  </span>
                </el-tooltip>
              </td>

              <!-- Type -->
              <td v-if="column.key === 'type'" class="whitespace-nowrap px-4 py-2">
                <span
                  :class="[
                    'inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-bold ring-1 ring-inset',
                    getTypeBadge(log).className,
                  ]"
                >
                  {{ getTypeBadge(log).label }}
                </span>
              </td>

              <!-- Endpoint -->
              <td v-if="column.key === 'endpoint'" class="px-4 py-2">
                <div class="max-w-[160px]">
                  <el-tooltip
                    v-if="log.inbound_endpoint"
                    :content="formatEndpointTooltip(log)"
                    placement="top"
                    :show-after="500"
                  >
                    <span
                      class="truncate font-mono text-[11px] text-gray-700 dark:text-gray-300"
                    >
                      {{ log.inbound_endpoint }}
                    </span>
                  </el-tooltip>
                  <span v-else class="text-xs text-gray-400">-</span>
                </div>
              </td>

              <!-- Platform -->
              <td v-if="column.key === 'platform'" class="whitespace-nowrap px-4 py-2">
                <span
                  class="inline-flex items-center rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-bold uppercase text-gray-600 dark:bg-dark-700 dark:text-gray-300"
                >
                  {{ log.platform || "-" }}
                </span>
              </td>

              <!-- Model -->
              <td v-if="column.key === 'model'" class="px-4 py-2">
                <div class="max-w-[160px]">
                  <template v-if="hasModelMapping(log)">
                    <el-tooltip
                      :content="modelMappingTooltip(log)"
                      placement="top"
                      :show-after="500"
                    >
                      <span
                        class="flex items-center gap-1 truncate font-mono text-[11px] text-gray-700 dark:text-gray-300"
                      >
                        <span class="truncate">{{ log.requested_model }}</span>
                        <span class="flex-shrink-0 text-gray-400">→</span>
                        <span
                          class="truncate text-primary-600 dark:text-primary-400"
                          >{{ log.upstream_model }}</span
                        >
                      </span>
                    </el-tooltip>
                  </template>
                  <template v-else>
                    <span
                      v-if="displayModel(log)"
                      class="truncate font-mono text-[11px] text-gray-700 dark:text-gray-300"
                      :title="displayModel(log)"
                    >
                      {{ displayModel(log) }}
                    </span>
                    <span v-else class="text-xs text-gray-400">-</span>
                  </template>
                </div>
              </td>

              <!-- Group -->
              <td v-if="column.key === 'group'" class="px-4 py-2">
                <el-tooltip
                  v-if="log.group_id"
                  :content="t('admin.ops.errorLog.id') + ' ' + log.group_id"
                  placement="top"
                  :show-after="500"
                >
                  <span
                    class="max-w-[100px] truncate text-xs font-medium text-gray-900 dark:text-gray-200"
                  >
                    {{ log.group_name || "-" }}
                  </span>
                </el-tooltip>
                <span v-else class="text-xs text-gray-400">-</span>
              </td>

              <!-- User -->
              <td v-if="column.key === 'user'" class="px-4 py-2">
                <el-tooltip
                  v-if="log.user_id"
                  :content="t('admin.ops.errorLog.userId') + ' ' + log.user_id"
                  placement="top"
                  :show-after="500"
                >
                  <button
                    v-if="userClickable && log.user_email"
                    type="button"
                    class="max-w-[100px] truncate text-xs font-medium text-primary-600 dark:text-primary-400"
                    :title="t('admin.usage.clickToViewBalance')"
                    @click.stop="emit('userClick', log.user_id, log.user_email)"
                  >{{ log.user_email }}</button>
                  <span v-else class="max-w-[100px] truncate text-xs font-medium text-gray-900 dark:text-gray-200">
                    {{ log.user_email || '#' + log.user_id }}
                  </span>
                </el-tooltip>
                <span v-else class="text-xs text-gray-400">-</span>
              </td>

              <!-- API Key -->
              <td v-if="column.key === 'api_key'" class="px-4 py-2">
                <template v-if="log.api_key_id || log.api_key_name">
                  <span class="max-w-[140px] truncate text-xs font-medium text-gray-900 dark:text-gray-200">
                    {{ log.api_key_name || "#" + log.api_key_id }}
                  </span>
                  <span
                    v-if="log.api_key_deleted"
                    class="ml-1 inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight text-rose-600 ring-1 ring-inset ring-rose-200 dark:text-rose-400 dark:ring-rose-500/30"
                  >
                    {{ t("admin.ops.errorLog.keyDeletedBadge") }}
                  </span>
                </template>
                <span v-else class="text-xs text-gray-400">-</span>
              </td>

              <!-- Account -->
              <td v-if="column.key === 'account'" class="px-4 py-2">
                <el-tooltip
                  v-if="log.account_id"
                  :content="t('admin.ops.errorLog.accountId') + ' ' + log.account_id"
                  placement="top"
                  :show-after="500"
                >
                  <span
                    class="max-w-[140px] truncate text-xs font-medium text-gray-900 dark:text-gray-200"
                  >
                    {{ log.account_name || "#" + log.account_id }}
                  </span>
                </el-tooltip>
                <span v-else class="text-xs text-gray-400">-</span>
              </td>

              <!-- Status -->
              <td v-if="column.key === 'status'" class="whitespace-nowrap px-4 py-2">
                <div class="flex items-center gap-1.5">
                  <span
                    :class="[
                      'inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-bold ring-1 ring-inset',
                      getStatusClass(log.status_code),
                    ]"
                  >
                    {{ log.status_code }}
                  </span>
                  <span
                    v-if="log.severity"
                    :class="[
                      'rounded px-1.5 py-0.5 text-[10px] font-bold',
                      getSeverityClass(log.severity),
                    ]"
                  >
                    {{ log.severity }}
                  </span>
                  <span
                    v-if="log.request_type != null && log.request_type > 0"
                    class="rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-bold text-gray-600 dark:bg-dark-700 dark:text-gray-300"
                  >
                    {{ formatRequestType(log.request_type) }}
                  </span>
                </div>
              </td>

              <!-- Message (Response Content) -->
              <td v-if="column.key === 'message'" class="px-4 py-2">
                <div class="max-w-[200px]">
                  <p
                    class="truncate text-[11px] font-medium text-gray-600 dark:text-gray-400"
                    :title="log.message"
                  >
                    {{ formatSmartMessage(log.message) || "-" }}
                  </p>
                </div>
              </td>

              <!-- Actions -->
              <td v-if="column.key === 'actions'" class="whitespace-nowrap px-4 py-2 text-right" @click.stop>
                <div class="flex items-center justify-end gap-3">
                  <LiquidButton
                    type="button"
                    class="text-primary-600 hover:text-primary-700 dark:text-primary-400 text-xs font-bold"
                    @click="emit('openErrorDetail', log.id)"
                    variant="plain"
                    size="sm"
                  >
                    {{ t("admin.ops.errorLog.details") }}
                  </LiquidButton>
                </div>
              </td>
              <td v-if="column.key === 'category'" class="px-4 py-2 text-xs">
                {{ t('usage.errors.categories.' + mapErrorCategory(log.phase, log.type)) }}
              </td>
              <td v-if="column.key === 'user_agent'" class="max-w-[320px] truncate px-4 py-2 text-xs" :title="log.user_agent || ''">{{ log.user_agent || '-' }}</td>
              <td v-if="column.key === 'client_ip'" class="px-4 py-2 font-mono text-xs">{{ log.client_ip || '-' }}</td>
              </template>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div class="bg-gray-50/50 dark:bg-dark-800/50">
        <Pagination
          v-if="total > 0"
          :total="total"
          :page="page"
          :page-size="pageSize"
          @update:page="emit('update:page', $event)"
          @update:pageSize="emit('update:pageSize', $event)"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { mapErrorSortKey } from "@/utils/errorBadges";
import { mapErrorCategory } from "@/utils/errorCategory";
import LiquidButton from "@/components/common/LiquidButton.vue";
import { useI18n } from "vue-i18n";
import Pagination from "@/components/common/Pagination.vue";
import type { OpsErrorLog } from "@/api/admin/ops";
import { getSeverityClass, formatDateTime } from "../utils/opsFormatters";

const { t } = useI18n();

function isUpstreamRow(log: OpsErrorLog): boolean {
  const phase = String(log.phase || "").toLowerCase();
  const owner = String(log.error_owner || "").toLowerCase();
  return phase === "upstream" && owner === "provider";
}

function formatEndpointTooltip(log: OpsErrorLog): string {
  const parts: string[] = [];
  if (log.inbound_endpoint) parts.push(`Inbound: ${log.inbound_endpoint}`);
  if (log.upstream_endpoint) parts.push(`Upstream: ${log.upstream_endpoint}`);
  return parts.join("\n") || "";
}

function hasModelMapping(log: OpsErrorLog): boolean {
  const requested = String(log.requested_model || "").trim();
  const upstream = String(log.upstream_model || "").trim();
  return !!requested && !!upstream && requested !== upstream;
}

function modelMappingTooltip(log: OpsErrorLog): string {
  const requested = String(log.requested_model || "").trim();
  const upstream = String(log.upstream_model || "").trim();
  if (!requested && !upstream) return "";
  if (requested && upstream) return `${requested} → ${upstream}`;
  return upstream || requested;
}

function displayModel(log: OpsErrorLog): string {
  const upstream = String(log.upstream_model || "").trim();
  if (upstream) return upstream;
  const requested = String(log.requested_model || "").trim();
  if (requested) return requested;
  return String(log.model || "").trim();
}

function formatRequestType(type: number | null | undefined): string {
  switch (type) {
    case 1:
      return t("admin.ops.errorLog.requestTypeSync");
    case 2:
      return t("admin.ops.errorLog.requestTypeStream");
    case 3:
      return t("admin.ops.errorLog.requestTypeWs");
    default:
      return "";
  }
}

function getTypeBadge(log: OpsErrorLog): { label: string; className: string } {
  const phase = String(log.phase || "").toLowerCase();
  const owner = String(log.error_owner || "").toLowerCase();

  if (isUpstreamRow(log)) {
    return {
      label: t("admin.ops.errorLog.typeUpstream"),
      className:
        "bg-red-50 text-red-700 ring-red-600/20 dark:bg-red-900/30 dark:text-red-400 dark:ring-red-500/30",
    };
  }
  if (phase === "request" && owner === "client") {
    return {
      label: t("admin.ops.errorLog.typeRequest"),
      className:
        "bg-amber-50 text-amber-700 ring-amber-600/20 dark:bg-amber-900/30 dark:text-amber-400 dark:ring-amber-500/30",
    };
  }
  if (phase === "auth" && owner === "client") {
    return {
      label: t("admin.ops.errorLog.typeAuth"),
      className:
        "bg-blue-50 text-blue-700 ring-blue-600/20 dark:bg-blue-900/30 dark:text-blue-400 dark:ring-blue-500/30",
    };
  }
  if (phase === "routing" && owner === "platform") {
    return {
      label: t("admin.ops.errorLog.typeRouting"),
      className:
        "bg-purple-50 text-purple-700 ring-purple-600/20 dark:bg-purple-900/30 dark:text-purple-400 dark:ring-purple-500/30",
    };
  }
  if (phase === "internal" && owner === "platform") {
    return {
      label: t("admin.ops.errorLog.typeInternal"),
      className:
        "bg-gray-100 text-gray-800 ring-gray-600/20 dark:bg-dark-700 dark:text-gray-200 dark:ring-dark-500/40",
    };
  }

  const fallback = phase || owner || t("common.unknown");
  return {
    label: fallback,
    className:
      "bg-gray-50 text-gray-700 ring-gray-600/10 dark:bg-dark-900 dark:text-gray-300 dark:ring-dark-700",
  };
}

interface Props {
  rows: OpsErrorLog[];
  total: number;
  loading: boolean;
  page: number;
  pageSize: number;
  visibleColumnKeys?: string[];
  columnOrder?: string[];
  summaryFirst?: boolean;
  userClickable?: boolean;
  flat?: boolean;
}

interface Emits {
  (e: "sort", sortBy: string, order: "asc" | "desc"): void;
  (e: "userClick", id: number, email?: string): void;
  (e: "openErrorDetail", id: number): void;
  (e: "update:page", value: number): void;
  (e: "update:pageSize", value: number): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const allColumns = computed(() => [
  { key: 'user', label: t('admin.ops.errorLog.user') },
  { key: 'api_key', label: t('admin.ops.errorLog.apiKey') },
  { key: 'account', label: t('admin.ops.errorLog.account') },
  { key: 'platform', label: t('admin.ops.errorLog.platform') },
  { key: 'model', label: t('admin.ops.errorLog.model'), sortable: true },
  { key: 'endpoint', label: t('admin.ops.errorLog.endpoint') },
  { key: 'group', label: t('admin.ops.errorLog.group') },
  { key: 'type', label: t('admin.ops.errorLog.type') },
  { key: 'category', label: t('usage.errors.category') },
  { key: 'status', label: t('admin.ops.errorLog.status'), sortable: true },
  { key: 'message', label: t('admin.ops.errorLog.message') },
  { key: 'created_at', label: t('admin.ops.errorLog.time'), sortable: true },
  { key: 'user_agent', label: t('usage.userAgent') },
  { key: 'client_ip', label: t('admin.ops.errorLog.ip') },
  { key: 'actions', label: t('admin.ops.errorLog.action') },
]);
const columns = computed(() => {
  const visible = allColumns.value.filter((column) => !props.visibleColumnKeys || props.visibleColumnKeys.includes(column.key));
  const order = props.columnOrder ?? (props.summaryFirst ? ['created_at', 'message'] : []);
  const keys = [...new Set([...order, ...visible.map((column) => column.key)])];
  return keys.flatMap((key) => visible.filter((column) => column.key === key));
});
const sortKey = ref('created_at');
const sortOrder = ref<'asc' | 'desc'>('desc');
function sortColumn(key: string) {
  sortOrder.value = sortKey.value === key && sortOrder.value === 'asc' ? 'desc' : 'asc';
  sortKey.value = key;
  emit('sort', mapErrorSortKey(key), sortOrder.value);
}

function getStatusClass(code: number): string {
  if (code >= 500)
    return "bg-red-50 text-red-700 ring-red-600/20 dark:bg-red-900/30 dark:text-red-400 dark:ring-red-500/30";
  if (code === 429)
    return "bg-purple-50 text-purple-700 ring-purple-600/20 dark:bg-purple-900/30 dark:text-purple-400 dark:ring-purple-500/30";
  if (code >= 400)
    return "bg-amber-50 text-amber-700 ring-amber-600/20 dark:bg-amber-900/30 dark:text-amber-400 dark:ring-amber-500/30";
  return "bg-gray-50 text-gray-700 ring-gray-600/20 dark:bg-gray-900/30 dark:text-gray-400 dark:ring-gray-500/30";
}

function formatSmartMessage(msg: string): string {
  if (!msg) return "";

  if (msg.startsWith("{") || msg.startsWith("[")) {
    try {
      const obj = JSON.parse(msg);
      if (obj?.error?.message) return String(obj.error.message);
      if (obj?.message) return String(obj.message);
      if (obj?.detail) return String(obj.detail);
      if (typeof obj === "object") return JSON.stringify(obj).substring(0, 150);
    } catch {
      // ignore parse error
    }
  }

  if (msg.includes("context deadline exceeded"))
    return t("admin.ops.errorLog.commonErrors.contextDeadlineExceeded");
  if (msg.includes("connection refused"))
    return t("admin.ops.errorLog.commonErrors.connectionRefused");
  if (msg.toLowerCase().includes("rate limit"))
    return t("admin.ops.errorLog.commonErrors.rateLimit");

  return msg.length > 200 ? msg.substring(0, 200) + "..." : msg;
}
</script>
