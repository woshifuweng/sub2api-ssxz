<template>
  <component :is="pageShell" v-bind="pageShellProps">
    <div class="channel-status-workbench">
      <MonitorHero
        :overall-status="overallStatus"
        :interval-seconds="DEFAULT_INTERVAL_SECONDS"
        :window="currentWindow"
        :loading="loading"
        :auto-refresh="autoRefresh"
        @update:window="handleWindowChange"
        @refresh="manualReload"
      />

      <p class="channel-status-disclaimer">
        {{ t("channelStatus.disclaimer") }}
      </p>

      <MonitorCardGrid
        v-if="loading || items.length > 0"
        :items="items"
        :window="currentWindow"
        :countdown-seconds="countdown"
        :loading="loading"
        :detail-cache="detailCache"
        :empty-description="emptyDescription"
        @card-click="openDetail"
      />
      <div v-else class="f0-card channel-status-empty">
        <div class="channel-status-empty__icon">
          <Icon name="inbox" size="lg" />
        </div>
        <strong>{{ t("channelStatus.empty.title") }}</strong>
        <span>{{ emptyDescription }}</span>
        <LiquidButton
          v-if="!channelMonitorDisabled"
          type="button"
          @click="manualReload"
          size="sm"
        >
          <Icon name="refresh" size="sm" />
          {{ t("common.refresh", "刷新") }}
        </LiquidButton>
      </div>
    </div>

    <MonitorDetailDialog
      :show="showDetail"
      :monitor-id="detailTarget?.id ?? null"
      :title="detailTitle"
      @close="closeDetail"
    />
  </component>
</template>

<script setup lang="ts">
import LiquidButton from "@/components/common/LiquidButton.vue";
import {
  ref,
  reactive,
  computed,
  onMounted,
  onBeforeUnmount,
  watch,
} from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { useAppStore } from "@/stores/app";
import { extractApiErrorMessage } from "@/utils/apiError";
import {
  list as listChannelMonitorViews,
  status as fetchChannelMonitorDetail,
  type UserMonitorView,
  type UserMonitorDetail,
} from "@/api/channelMonitor";
import AppLayout from "@/components/layout/AppLayout.vue";
import AppSectionShell from "@/components/user/AppSectionShell.vue";
import MonitorHero, {
  type MonitorWindow,
  type OverallStatus,
} from "@/components/user/monitor/MonitorHero.vue";
import MonitorCardGrid from "@/components/user/monitor/MonitorCardGrid.vue";
import MonitorDetailDialog from "@/components/user/MonitorDetailDialog.vue";
import Icon from "@/components/icons/Icon.vue";
import {
  DEFAULT_INTERVAL_SECONDS,
  STATUS_OPERATIONAL,
} from "@/constants/channelMonitor";
import { useAutoRefresh } from "@/composables/useAutoRefresh";

const { t } = useI18n();
const route = useRoute();
const appStore = useAppStore();

const useWorkbenchShell = computed(() => route.path === "/app/channel-status");
const pageShell = computed(() =>
  useWorkbenchShell.value ? AppSectionShell : AppLayout,
);
const pageShellProps = computed(() =>
  useWorkbenchShell.value
    ? {
        title: t("channelStatus.title"),
        subtitle: t("channelStatus.description"),
        eyebrow: t("channelStatus.eyebrow"),
        icon: "chartBar",
      }
    : {},
);

// ── State ──
const items = ref<UserMonitorView[]>([]);
const loading = ref(false);
const currentWindow = ref<MonitorWindow>("7d");
const detailCache = reactive<Record<number, UserMonitorDetail>>({});
const showDetail = ref(false);
const detailTarget = ref<UserMonitorView | null>(null);

let abortController: AbortController | null = null;

const autoRefresh = useAutoRefresh({
  storageKey: "channel-status-auto-refresh",
  intervals: [30, 60, 120] as const,
  defaultInterval: DEFAULT_INTERVAL_SECONDS,
  onRefresh: () => reload(true),
  shouldPause: () => document.hidden || loading.value,
});
const countdown = autoRefresh.countdown;

// ── Computed ──
const overallStatus = computed<OverallStatus>(() => {
  if (items.value.length === 0) return "unknown";
  for (const it of items.value) {
    if (it.primary_status === "failed" || it.primary_status === "error")
      return "degraded";
    if (it.primary_status !== STATUS_OPERATIONAL) return "degraded";
  }
  return "operational";
});

const emptyDescription = computed(() => {
  if (appStore.cachedPublicSettings?.channel_monitor_enabled === false) {
    return t("channelStatus.empty.disabledDescription");
  }
  return t("channelStatus.empty.description");
});

const detailTitle = computed(() => {
  return detailTarget.value?.name || t("channelStatus.detailTitle");
});
const channelMonitorDisabled = computed(
  () => appStore.cachedPublicSettings?.channel_monitor_enabled === false,
);

// ── Loaders ──
async function reload(silent = false) {
  if (abortController) abortController.abort();
  const ctrl = new AbortController();
  abortController = ctrl;
  if (!silent) loading.value = true;
  try {
    const res = await listChannelMonitorViews({ signal: ctrl.signal });
    if (ctrl.signal.aborted || abortController !== ctrl) return;
    items.value = res.items || [];
  } catch (err: unknown) {
    const e = err as { name?: string; code?: string };
    if (e?.name === "AbortError" || e?.code === "ERR_CANCELED") return;
    appStore.showError(
      extractApiErrorMessage(err, t("channelStatus.loadError")),
    );
  } finally {
    if (abortController === ctrl) {
      if (!silent) loading.value = false;
      countdown.value = DEFAULT_INTERVAL_SECONDS;
      abortController = null;
    }
  }
}

async function manualReload() {
  await reload(false);
  // After base reload, refresh any cached detail records so non-7d availability
  // values stay in sync without forcing the user to switch tabs again.
  if (currentWindow.value !== "7d") {
    await Promise.all(items.value.map((it) => loadDetail(it.id, true)));
  }
}

async function loadDetail(id: number, force = false) {
  if (!force && detailCache[id]) return;
  try {
    detailCache[id] = await fetchChannelMonitorDetail(id);
  } catch (err: unknown) {
    appStore.showError(
      extractApiErrorMessage(err, t("channelStatus.detailLoadError")),
    );
  }
}

async function ensureDetailsForWindow() {
  if (currentWindow.value === "7d") return;
  await Promise.all(items.value.map((it) => loadDetail(it.id)));
}

// ── Handlers ──
async function handleWindowChange(value: MonitorWindow) {
  currentWindow.value = value;
  await ensureDetailsForWindow();
}

function openDetail(row: UserMonitorView) {
  detailTarget.value = row;
  showDetail.value = true;
}

function closeDetail() {
  showDetail.value = false;
  detailTarget.value = null;
}

watch(items, () => {
  void ensureDetailsForWindow();
});

watch(
  () => appStore.cachedPublicSettings?.channel_monitor_enabled,
  (enabled) => {
    if (enabled === false) {
      if (abortController) abortController.abort();
      items.value = [];
      autoRefresh.stop();
      return;
    }
    if (autoRefresh.enabled.value) autoRefresh.start();
  },
);

onMounted(() => {
  if (channelMonitorDisabled.value) {
    items.value = [];
    autoRefresh.stop();
    return;
  }
  void reload(false);
  autoRefresh.setEnabled(true);
});

onBeforeUnmount(() => {
  if (abortController) abortController.abort();
});
</script>

<style scoped>
.channel-status-workbench {
  display: grid;
  gap: 1.5rem;
}

.channel-status-disclaimer {
  margin: -0.5rem 0 0;
  color: var(--ssxz-text-muted);
  font-size: 0.75rem;
  line-height: 1.6;
}

.channel-status-workbench :deep(.channel-monitor-toolbar) {
  margin-bottom: 0;
  background: var(--ssxz-surface-raised);
}

.channel-status-workbench :deep(.channel-monitor-empty) {
  min-height: 18rem;
}

.channel-status-empty {
  display: grid;
  min-height: 18rem;
  place-items: center;
  align-content: center;
  gap: 0.65rem;
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
  padding: 2rem;
  text-align: center;
}

.channel-status-empty__icon {
  display: grid;
  width: 3.5rem;
  height: 3.5rem;
  place-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-primary-soft);
  color: var(--ssxz-action);
}

.channel-status-empty strong {
  color: var(--ssxz-text-primary);
  font-size: 1rem;
}

.channel-status-empty span {
  max-width: 44rem;
  color: var(--ssxz-text-muted);
  font-size: 0.84rem;
  line-height: 1.6;
}
</style>
