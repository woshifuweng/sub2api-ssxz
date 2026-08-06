<template>
  <component :is="pageShell" v-bind="pageShellProps">
    <TablePageLayout>
      <template #filters>
        <div class="pricing-toolbar">
          <div class="pricing-guide f0-card">
            <div class="pricing-guide__icon"><Icon name="server" size="md" /></div>
            <div>
              <p>{{ t('availableChannels.userGuideTitle') }}</p>
              <span>{{ t('availableChannels.userGuideDescription') }}</span>
            </div>
          </div>

          <div class="pricing-actions">
              <div class="pricing-search">
                <Icon
                  name="search"
                  size="md"
                  class="pricing-search__icon"
                />
                <input
                  v-model="searchQuery"
                  type="text"
                  :placeholder="t('availableChannels.searchPlaceholder')"
                  class="f0-input-control f0-input-control--leading"
                />
              </div>
              <button
                @click="loadChannels"
                :disabled="loading"
                class="btn btn-secondary btn-icon"
                :title="t('common.refresh', 'Refresh')"
              >
                <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
              </button>
          </div>
        </div>
      </template>

      <template #table>
        <AvailableChannelsTable
          v-show="loading || filteredChannels.length > 0"
          :columns="columnLabels"
          :rows="filteredChannels"
          :loading="loading"
          :user-group-rates="userGroupRates"
          pricing-key-prefix="availableChannels.pricing"
          :no-pricing-label="t('availableChannels.noPricing')"
          :no-models-label="t('availableChannels.noModels')"
          :empty-label="emptyLabel"
        />
        <div v-if="!loading && filteredChannels.length === 0" class="f0-card pricing-empty-state">
          <div class="pricing-empty-state__icon"><Icon name="inbox" size="lg" /></div>
          <strong>{{ emptyLabel }}</strong>
          <span>{{ t('availableChannels.emptyDescription', '当前账号暂时没有可用模型，可稍后刷新或检查 API Key 所属分组。') }}</span>
          <button type="button" class="btn btn-primary btn-sm" @click="loadChannels">
            <Icon name="refresh" size="sm" />
            {{ t('common.refresh', '刷新') }}
          </button>
        </div>
      </template>
    </TablePageLayout>
  </component>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import AvailableChannelsTable from '@/components/channels/AvailableChannelsTable.vue'
import userChannelsAPI, { type UserAvailableChannel } from '@/api/channels'
import userGroupsAPI from '@/api/groups'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()

const useWorkbenchShell = computed(() => route.path === '/app/available-channels')
const pageShell = computed(() => useWorkbenchShell.value ? AppSectionShell : AppLayout)
const pageShellProps = computed(() => useWorkbenchShell.value
  ? {
      title: t('availableChannels.title'),
      subtitle: t('availableChannels.description'),
      eyebrow: t('availableChannels.eyebrow'),
      icon: 'server'
    }
  : {})

const channels = ref<UserAvailableChannel[]>([])
const userGroupRates = ref<Record<number, number>>({})
const loading = ref(false)
const searchQuery = ref('')

const columnLabels = computed(() => ({
  name: t('availableChannels.columns.name', '服务'),
  description: t('availableChannels.columns.description', '说明'),
  platform: t('availableChannels.columns.platform', '类型'),
  groups: t('availableChannels.columns.groups', '适用范围'),
  supportedModels: t('availableChannels.columns.supportedModels', '可选模型'),
}))

const emptyLabel = computed(() =>
  appStore.cachedPublicSettings?.available_channels_enabled === true
    ? t('availableChannels.empty')
    : t('availableChannels.emptyDisabled')
)

/**
 * 搜索过滤：
 * - 命中渠道名/描述 → 整个渠道（所有 platforms）都保留
 * - 否则按 platform/group/model 维度在 sections 里过滤，保留有匹配的 section
 * - 所有 sections 都不匹配时，渠道本身被过滤掉
 */
const filteredChannels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return channels.value
  return channels.value
    .map((ch) => {
      const nameHit = ch.name.toLowerCase().includes(q)
      const descHit = (ch.description || '').toLowerCase().includes(q)
      if (nameHit || descHit) return ch
      const matchingSections = ch.platforms.filter(
        (p) =>
          p.platform.toLowerCase().includes(q) ||
          p.groups.some((g) => g.name.toLowerCase().includes(q)) ||
          p.supported_models.some((m) => m.name.toLowerCase().includes(q)),
      )
      if (matchingSections.length === 0) return null
      return { ...ch, platforms: matchingSections }
    })
    .filter((ch): ch is UserAvailableChannel => ch !== null)
})

async function loadChannels() {
  loading.value = true
  try {
    // 渠道列表和用户专属倍率并发拉取。专属倍率失败不阻塞渠道展示——
    // 失败时只是无法渲染专属倍率角标，降级为仅显示默认倍率。
    const [list, rates] = await Promise.all([
      userChannelsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates().catch((err: unknown) => {
        console.error('Failed to load user group rates:', err)
        return {} as Record<number, number>
      }),
    ])
    channels.value = list
    userGroupRates.value = rates
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(loadChannels)
</script>

<style scoped>
.pricing-toolbar {
  display: grid;
  gap: 1.5rem;
}

.pricing-guide {
  display: flex;
  align-items: flex-start;
  gap: 0.8rem;
  padding: 1.25rem;
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
}

.pricing-guide__icon {
  display: grid;
  width: 2.25rem;
  height: 2.25rem;
  flex: 0 0 auto;
  place-items: center;
  border-radius: var(--ssxz-radius-button);
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-action);
}

.pricing-guide p {
  margin: 0;
  color: var(--ssxz-text-primary);
  font-weight: 700;
}

.pricing-guide span {
  display: block;
  margin-top: 0.25rem;
  color: var(--ssxz-text-secondary);
  font-size: 0.85rem;
  line-height: 1.55;
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

:deep(.pricing-table-frame) {
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
}

:deep(.pricing-empty) {
  height: 16rem;
  color: var(--ssxz-text-muted);
}

:deep(.pricing-empty svg) {
  width: 3.5rem;
  height: 3.5rem;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-primary-soft);
  color: var(--ssxz-action);
  padding: 0.85rem;
}

.pricing-empty-state {
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

.pricing-empty-state__icon {
  display: grid;
  width: 3.5rem;
  height: 3.5rem;
  place-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-primary-soft);
  color: var(--ssxz-action);
}

.pricing-empty-state strong {
  color: var(--ssxz-text-primary);
  font-size: 1rem;
}

.pricing-empty-state span {
  max-width: 36rem;
  color: var(--ssxz-text-muted);
  font-size: 0.84rem;
  line-height: 1.6;
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
