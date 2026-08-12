<template>
  <AppSectionShell
    :title="t('availableChannels.title')"
    :subtitle="t('availableChannels.description')"
    :eyebrow="t('availableChannels.eyebrow')"
    icon="server"
  >
    <div class="pricing-page" @click="closeGroupMenu">
      <div class="pricing-controls">
        <div class="pricing-tabs" role="tablist" aria-label="模型供应商">
          <button
            v-for="tab in tabs"
            :key="tab.platform"
            type="button"
            role="tab"
            :aria-selected="activePlatform === tab.platform"
            :class="['pricing-tab', { 'is-active': activePlatform === tab.platform }, tab.platform]"
            @click.stop="activePlatform = tab.platform"
          >
            <span :class="['provider-mark', tab.platform]"><PlatformIcon :platform="tab.platform" size="sm" /></span>
            {{ tab.label }}
          </button>
        </div>

        <div class="pricing-toolbar">
          <label class="pricing-search">
            <span class="sr-only">搜索模型</span>
            <Icon name="search" size="md" class="pricing-search__icon" />
            <input
              v-model="searchQuery"
              type="search"
              class="f0-input-control f0-input-control--leading"
              placeholder="搜索模型或分组..."
            />
          </label>
          <button
            type="button"
            class="btn btn-secondary btn-icon"
            :disabled="loading"
            title="刷新"
            aria-label="刷新模型价格"
            @click.stop="loadChannels"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </div>

      <div class="pricing-summary">
        <span>显示 {{ visibleRows.length }} 个&nbsp; 共 {{ activeRows.length }} 个</span>
        <span v-if="lastUpdated" class="pricing-summary__updated">更新于 {{ lastUpdated }}</span>
      </div>

      <div class="pricing-table-frame">
        <div v-if="loading" class="pricing-state">正在加载可用模型...</div>
        <div v-else-if="visibleRows.length === 0" class="pricing-state">
          <Icon name="inbox" size="lg" />
          <strong>暂无可用模型</strong>
          <span>请检查当前账号的 API Key、余额和分组授权后重试。</span>
        </div>
        <div v-else class="pricing-table-wrap">
          <table class="pricing-table">
            <thead>
              <tr>
                <th scope="col">
                  <button type="button" class="sort-button" @click.stop="toggleSort">
                    模型
                    <Icon name="chevronDown" size="xs" :class="sortDirection === 'desc' ? 'rotate-180' : ''" />
                  </button>
                </th>
                <th scope="col">供应商</th>
                <th scope="col">分组</th>
                <th scope="col">输入</th>
                <th scope="col">输出</th>
                <th scope="col">缓存</th>
                <th scope="col">上下文</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in visibleRows" :key="row.key">
                <td class="model-cell">
                  <code>{{ row.name }}</code>
                </td>
                <td>
                  <span :class="['provider-label', row.platform]">
                    <span :class="['provider-mark', row.platform]"><PlatformIcon :platform="row.platform" size="sm" /></span>
                    {{ row.providerLabel }}
                  </span>
                </td>
                <td class="group-cell">
                  <div v-if="currentGroup(row)" class="group-picker" @click.stop>
                    <button
                      type="button"
                      class="group-picker__trigger"
                      :aria-expanded="openGroupKey === row.key"
                      :aria-label="`选择 ${row.name} 分组`"
                      @click="toggleGroupMenu(row.key)"
                      @keydown.esc="closeGroupMenu"
                    >
                      <GroupBadge
                        :name="currentGroup(row)!.name"
                        :platform="row.platform"
                        :rate-multiplier="effectiveRate(currentGroup(row)!)"
                      />
                      <Icon name="chevronDown" size="xs" :class="['group-picker__chevron', { 'rotate-180': openGroupKey === row.key }]" />
                    </button>
                    <div v-if="openGroupKey === row.key" class="group-picker__menu" @click.stop>
                      <div class="group-picker__search">
                        <Icon name="search" size="sm" />
                        <input v-model="groupQuery" type="search" placeholder="搜索分组..." @keydown.esc="closeGroupMenu" />
                      </div>
                      <button
                        v-for="group in groupsForRow(row)"
                        :key="group.id"
                        type="button"
                        class="group-picker__option"
                        @click="selectGroup(row.key, group.id)"
                      >
                        <GroupOptionItem
                          :name="group.name"
                          :platform="groupPlatform(group, row.platform)"
                          :subscription-type="group.subscription_type as 'standard' | 'subscription'"
                          :rate-multiplier="group.rate_multiplier"
                          :user-rate-multiplier="userGroupRates[group.id] ?? null"
                          :description="group.description"
                          :selected="currentGroup(row)?.id === group.id"
                          show-checkmark
                        />
                      </button>
                      <span v-if="groupsForRow(row).length === 0" class="group-picker__empty">没有匹配分组</span>
                    </div>
                  </div>
                  <span v-else class="muted-value">-</span>
                </td>
                <td>{{ formatPrice(row.model.pricing?.input_price, currentRate(row)) }}</td>
                <td>{{ formatPrice(row.model.pricing?.output_price, currentRate(row)) }}</td>
                <td class="cache-cell">
                  <template v-if="row.model.pricing && (row.model.pricing.cache_write_price !== null || row.model.pricing.cache_read_price !== null)">
                    <span>写入 {{ formatPrice(row.model.pricing.cache_write_price, currentRate(row)) }}</span>
                    <span>读取 {{ formatPrice(row.model.pricing.cache_read_price, currentRate(row)) }}</span>
                  </template>
                  <span v-else class="muted-value">-</span>
                </td>
                <td>{{ formatContext(row.model) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import userChannelsAPI, { type UserAvailableGroup } from '@/api/channels'
import userGroupsAPI from '@/api/groups'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  filterPricingRows,
  flattenPricingRows,
  formatContext,
  formatPrice,
  type ModelPricingRow,
  type PricingPlatform,
} from './modelPricing'
import type { GroupPlatform } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()

const tabs: Array<{ platform: PricingPlatform; label: string }> = [
  { platform: 'anthropic', label: 'Anthropic' },
  { platform: 'openai', label: 'OpenAI' },
]
const activePlatform = ref<PricingPlatform>('anthropic')
const channels = ref<Awaited<ReturnType<typeof userChannelsAPI.getAvailable>>>([])
const userGroupRates = ref<Record<number, number>>({})
const loading = ref(false)
const searchQuery = ref('')
const sortDirection = ref<'asc' | 'desc'>('asc')
const openGroupKey = ref<string | null>(null)
const groupQuery = ref('')
const selectedGroupByRow = ref<Record<string, number>>({})
const lastUpdated = ref('')

const allRows = computed(() => flattenPricingRows(channels.value))
const activeRows = computed(() => allRows.value.filter((row) => row.platform === activePlatform.value))
const visibleRows = computed(() => {
  const filtered = filterPricingRows(activeRows.value, searchQuery.value)
  return [...filtered].sort((left, right) => {
    const result = left.name.localeCompare(right.name)
    return sortDirection.value === 'asc' ? result : -result
  })
})

function currentGroup(row: ModelPricingRow): UserAvailableGroup | null {
  if (row.groups.length === 0) return null
  const selectedId = selectedGroupByRow.value[row.key]
  return row.groups.find((group) => group.id === selectedId) ?? row.groups[0]
}

function effectiveRate(group: UserAvailableGroup): number {
  return userGroupRates.value[group.id] ?? group.rate_multiplier
}

function currentRate(row: ModelPricingRow): number {
  const group = currentGroup(row)
  return group ? effectiveRate(group) : 1
}

function groupsForRow(row: ModelPricingRow): UserAvailableGroup[] {
  const query = groupQuery.value.trim().toLowerCase()
  if (!query) return row.groups
  return row.groups.filter((group) => (
    group.name.toLowerCase().includes(query) ||
    (group.description || '').toLowerCase().includes(query)
  ))
}

function groupPlatform(group: UserAvailableGroup, fallback: PricingPlatform): GroupPlatform {
  const platform = group.platform?.toLowerCase()
  if (platform === 'anthropic' || platform === 'openai' || platform === 'kiro' || platform === 'gemini' || platform === 'sora') {
    return platform
  }
  return fallback
}

function toggleSort() {
  sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
}

function toggleGroupMenu(rowKey: string) {
  if (openGroupKey.value === rowKey) {
    closeGroupMenu()
    return
  }
  openGroupKey.value = rowKey
  groupQuery.value = ''
}

function closeGroupMenu() {
  openGroupKey.value = null
  groupQuery.value = ''
}

function selectGroup(rowKey: string, groupId: number) {
  selectedGroupByRow.value[rowKey] = groupId
  closeGroupMenu()
}

async function loadChannels() {
  loading.value = true
  try {
    const [list, rates] = await Promise.all([
      userChannelsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates().catch((error: unknown) => {
        console.error('Failed to load user group rates:', error)
        return {} as Record<number, number>
      }),
    ])
    channels.value = list
    userGroupRates.value = rates
    lastUpdated.value = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(loadChannels)
</script>

<style scoped>
.pricing-page {
  display: grid;
  gap: 1rem;
}

.pricing-controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.pricing-tabs {
  display: inline-flex;
  gap: 0.3rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.9rem;
  background: var(--ssxz-surface-muted);
  padding: 0.25rem;
}

.pricing-tab {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  min-height: 2.35rem;
  border-radius: 0.65rem;
  color: var(--ssxz-text-muted);
  font-size: 0.85rem;
  font-weight: 700;
  padding: 0 0.85rem;
}

.pricing-tab:hover,
.pricing-tab.is-active {
  background: var(--ssxz-surface-raised);
  color: var(--ssxz-text);
  box-shadow: var(--ssxz-shadow-sm);
}

.pricing-toolbar {
  display: flex;
  align-items: center;
  gap: 0.65rem;
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

.provider-mark {
  display: inline-flex;
  width: 1.6rem;
  height: 1.6rem;
  align-items: center;
  justify-content: center;
  border-radius: 0.5rem;
}

.provider-mark.anthropic {
  background: rgb(139 92 246 / 0.14);
  color: #8b5cf6;
}

.provider-mark.openai {
  background: rgb(16 185 129 / 0.14);
  color: #10b981;
}

.pricing-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: var(--ssxz-text-muted);
  font-size: 0.82rem;
}

.pricing-summary__updated {
  color: var(--ssxz-subtle);
}

.pricing-table-frame {
  overflow: visible;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
}

.pricing-table-wrap {
  overflow-x: auto;
}

.pricing-table {
  width: 100%;
  min-width: 980px;
  border-collapse: collapse;
  text-align: left;
}

.pricing-table th {
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-text-muted);
  font-size: 0.76rem;
  font-weight: 800;
  letter-spacing: 0.02em;
  padding: 0.85rem 1rem;
  white-space: nowrap;
}

.pricing-table td {
  border-top: 1px solid var(--ssxz-border);
  color: var(--ssxz-body);
  font-size: 0.84rem;
  padding: 0.9rem 1rem;
  vertical-align: top;
  white-space: nowrap;
}

.pricing-table tbody tr:hover {
  background: color-mix(in srgb, var(--ssxz-primary) 4%, transparent);
}

.model-cell code {
  color: var(--ssxz-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.82rem;
  font-weight: 750;
}

.provider-label {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  font-weight: 700;
}

.provider-label.anthropic {
  color: #8b5cf6;
}

.provider-label.openai {
  color: #10b981;
}

.sort-button {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  color: inherit;
  font: inherit;
}

.group-cell {
  min-width: 15rem;
}

.group-picker {
  position: relative;
  width: max-content;
  max-width: 22rem;
}

.group-picker__trigger {
  display: inline-flex;
  max-width: 100%;
  align-items: center;
  gap: 0.25rem;
  border-radius: 0.55rem;
  padding: 0.1rem;
}

.group-picker__trigger:hover,
.group-picker__trigger:focus-visible {
  background: var(--ssxz-surface-muted);
}

.group-picker__chevron {
  color: var(--ssxz-text-muted);
  transition: transform 160ms ease;
}

.group-picker__menu {
  position: absolute;
  z-index: 20;
  top: calc(100% + 0.45rem);
  left: 0;
  width: min(25rem, 80vw);
  max-height: 22rem;
  overflow-y: auto;
  border: 1px solid var(--ssxz-border-strong);
  border-radius: 0.85rem;
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow);
  padding: 0.4rem;
}

.group-picker__search {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.6rem;
  color: var(--ssxz-text-muted);
  margin-bottom: 0.35rem;
  padding: 0.45rem 0.6rem;
}

.group-picker__search input {
  min-width: 0;
  width: 100%;
  background: transparent;
  color: var(--ssxz-text);
  outline: none;
}

.group-picker__option {
  display: flex;
  width: 100%;
  border-radius: 0.6rem;
  padding: 0.6rem;
  text-align: left;
}

.group-picker__option:hover,
.group-picker__option:focus-visible {
  background: var(--ssxz-surface-muted);
}

.group-picker__empty,
.muted-value {
  color: var(--ssxz-subtle);
}

.cache-cell {
  display: grid;
  gap: 0.2rem;
  color: var(--ssxz-text-muted) !important;
  font-size: 0.76rem !important;
  line-height: 1.4;
}

.pricing-state {
  display: grid;
  min-height: 18rem;
  place-items: center;
  align-content: center;
  gap: 0.55rem;
  color: var(--ssxz-text-muted);
  padding: 2rem;
  text-align: center;
}

.pricing-state strong {
  color: var(--ssxz-text);
}

@media (max-width: 640px) {
  .pricing-controls,
  .pricing-toolbar,
  .pricing-search {
    width: 100%;
  }

  .pricing-toolbar {
    justify-content: space-between;
  }

  .pricing-search {
    flex: 1;
  }

  .pricing-summary__updated {
    display: none;
  }
}
</style>
