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
                class="f0-button f0-button--outline f0-button--icon"
                :title="t('common.refresh', 'Refresh')"
              >
                <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
              </button>
          </div>
        </div>
      </template>

      <template #table>
        <AvailableChannelsTable
          :columns="columnLabels"
          :rows="filteredChannels"
          :loading="loading"
          :user-group-rates="userGroupRates"
          pricing-key-prefix="availableChannels.pricing"
          :no-pricing-label="t('availableChannels.noPricing')"
          :no-models-label="t('availableChannels.noModels')"
          :empty-label="emptyLabel"
        />
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
  name: t('availableChannels.columns.name'),
  description: t('availableChannels.columns.description'),
  platform: t('availableChannels.columns.platform'),
  groups: t('availableChannels.columns.groups'),
  supportedModels: t('availableChannels.columns.supportedModels'),
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
  gap: 1rem;
}

.pricing-guide {
  display: flex;
  align-items: flex-start;
  gap: 0.8rem;
  padding: 1rem;
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
