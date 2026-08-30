<template>
  <component :is="pageShell" v-bind="pageShellProps">
    <TablePageLayout>
      <template #filters>
        <div class="pricing-toolbar">
          <div class="pricing-actions">
            <div class="pricing-search">
              <Icon name="search" size="md" class="pricing-search__icon" />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('availableChannels.searchPlaceholder')"
                class="f0-input-control f0-input-control--leading"
              />
            </div>
            <LiquidButton
              @click="loadChannels"
              :disabled="loading"
              :title="t('common.refresh', 'Refresh')"
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
        </div>
      </template>

      <template #table>
        <AvailableChannelsTable
          :rows="channels"
          :loading="loading"
          :user-group-rates="userGroupRates"
          :search-query="searchQuery"
          :empty-label="emptyLabel"
        />
      </template>
    </TablePageLayout>
  </component>
</template>

<script setup lang="ts">
import LiquidButton from "@/components/common/LiquidButton.vue";
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import AppLayout from "@/components/layout/AppLayout.vue";
import AppSectionShell from "@/components/user/AppSectionShell.vue";
import TablePageLayout from "@/components/layout/TablePageLayout.vue";
import Icon from "@/components/icons/Icon.vue";
import AvailableChannelsTable from "@/components/channels/AvailableChannelsTable.vue";
import userChannelsAPI, { type UserAvailableChannel } from "@/api/channels";
import userGroupsAPI from "@/api/groups";
import { useAppStore } from "@/stores/app";
import { extractApiErrorMessage } from "@/utils/apiError";

const { t } = useI18n();
const route = useRoute();
const appStore = useAppStore();

const useWorkbenchShell = computed(
  () => route.path === "/app/available-channels",
);
const pageShell = computed(() =>
  useWorkbenchShell.value ? AppSectionShell : AppLayout,
);
const pageShellProps = computed(() =>
  useWorkbenchShell.value
    ? {
        title: t("availableChannels.title"),
        subtitle: t("availableChannels.description"),
        eyebrow: t("availableChannels.eyebrow"),
        icon: "server",
      }
    : {},
);

const channels = ref<UserAvailableChannel[]>([]);
const userGroupRates = ref<Record<number, number>>({});
const loading = ref(false);
const searchQuery = ref("");

const emptyLabel = computed(() =>
  appStore.cachedPublicSettings?.available_channels_enabled === true
    ? t("availableChannels.empty")
    : t("availableChannels.emptyDisabled"),
);

async function loadChannels() {
  loading.value = true;
  try {
    // 渠道列表和用户专属倍率并发拉取。专属倍率失败不阻塞模型目录展示，
    // 价格会降级为按分组默认倍率计算。
    const [list, rates] = await Promise.all([
      userChannelsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates().catch((err: unknown) => {
        console.error("Failed to load user group rates:", err);
        return {} as Record<number, number>;
      }),
    ]);
    channels.value = list;
    userGroupRates.value = rates;
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t("common.error")));
  } finally {
    loading.value = false;
  }
}

onMounted(loadChannels);
</script>

<style scoped>
.pricing-toolbar {
  display: block;
}

.pricing-actions {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
}

:deep(.table-page-layout) {
  gap: 1.5rem;
  height: auto;
}

:deep(.layout-section-scrollable) {
  min-height: 18rem;
}

:deep(.table-scroll-container) {
  overflow: visible;
  height: auto;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.pricing-search {
  position: relative;
  width: min(100%, 24rem);
}

.pricing-search__icon {
  position: absolute;
  z-index: 1;
  top: 50%;
  left: 0.75rem;
  color: var(--ssxz-text-muted);
  transform: translateY(-50%);
}

@media (max-width: 640px) {
  .pricing-actions,
  .pricing-search {
    width: 100%;
  }
}
</style>
