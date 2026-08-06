<template>
  <div class="f0-card pricing-table-frame">
    <table class="f0-table pricing-table">
      <thead>
        <tr>
          <th class="w-[180px] px-4 py-3 text-center">{{ columns.name }}</th>
          <th class="w-[200px] px-4 py-3 text-left">{{ columns.description }}</th>
          <th class="w-[140px] px-4 py-3 text-left">{{ columns.platform }}</th>
          <th class="px-4 py-3 text-left">{{ columns.groups }}</th>
          <th class="px-4 py-3 text-left">{{ columns.supportedModels }}</th>
        </tr>
      </thead>
      <tbody v-if="loading">
        <tr>
          <td colspan="5" class="pricing-empty">
            <Icon name="refresh" size="lg" class="inline-block animate-spin" />
          </td>
        </tr>
      </tbody>
      <tbody v-else-if="rows.length === 0">
        <tr>
          <td colspan="5" class="pricing-empty">
            <Icon name="inbox" size="xl" />
            <p>{{ emptyLabel }}</p>
          </td>
        </tr>
      </tbody>
      <!-- 每个渠道一个 tbody：首行 td rowspan 渠道名，后续行只渲染其余三列。
           tbody 之间强分隔线表达"渠道边界"，tbody 内部用淡分隔线区分平台。 -->
      <tbody
        v-else
        v-for="(channel, chIdx) in rows"
        :key="`${channel.name}-${chIdx}`"
        class="channel-group"
      >
        <tr
          v-for="(section, secIdx) in channel.platforms"
          :key="`${channel.name}-${section.platform}`"
          :class="{ 'channel-section': secIdx > 0 }"
        >
          <!-- 渠道名：只在第一行渲染并用 rowspan 纵向合并 -->
          <td
            v-if="secIdx === 0"
            :rowspan="channel.platforms.length"
            class="channel-name"
          >
            {{ channel.name }}
          </td>

          <!-- 描述：独立一列，同样用 rowspan 纵向合并 -->
          <td
            v-if="secIdx === 0"
            :rowspan="channel.platforms.length"
            class="channel-description"
          >
            <template v-if="channel.description">{{ channel.description }}</template>
            <span v-else>-</span>
          </td>

          <!-- 平台徽章 -->
          <td class="align-top px-4 py-3">
            <span
              :class="[
                'inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-[11px] font-medium uppercase',
                platformBadgeClass(canonicalPlatform(section.platform)),
              ]"
            >
              <PlatformIcon :platform="canonicalPlatform(section.platform) as GroupPlatform" size="xs" />
              {{ canonicalPlatform(section.platform) }}
            </span>
          </td>

          <!-- 分组：专属分组在前（紫色 shield 行），公开分组在后（灰色 globe 行）。 -->
          <td class="align-top px-4 py-3">
            <div class="flex flex-col gap-1.5">
              <div
                v-if="exclusiveGroups(section).length > 0"
                class="flex flex-wrap items-center gap-1.5"
              >
                <span
                  class="inline-flex items-center gap-0.5 text-[10px] font-medium uppercase text-primary-600 dark:text-primary-400"
                  :title="t('availableChannels.exclusiveTooltip')"
                >
                  <Icon name="shield" size="xs" class="h-3 w-3" />
                  {{ t('availableChannels.exclusive') }}
                </span>
                <GroupBadge
                  v-for="g in exclusiveGroups(section)"
                  :key="`ex-${g.id}`"
                  :name="g.name"
                  :platform="canonicalPlatform(g.platform) as GroupPlatform"
                  :subscription-type="(g.subscription_type || 'standard') as SubscriptionType"
                  :rate-multiplier="g.rate_multiplier"
                  :user-rate-multiplier="userGroupRates[g.id] ?? null"
                  always-show-rate
                />
              </div>
              <div
                v-if="publicGroups(section).length > 0"
                class="flex flex-wrap items-center gap-1.5"
              >
                <span
                  class="inline-flex items-center gap-0.5 text-[10px] font-medium uppercase text-gray-500 dark:text-gray-400"
                  :title="t('availableChannels.publicTooltip')"
                >
                  <Icon name="globe" size="xs" class="h-3 w-3" />
                  {{ t('availableChannels.public') }}
                </span>
                <GroupBadge
                  v-for="g in publicGroups(section)"
                  :key="`pub-${g.id}`"
                  :name="g.name"
                  :platform="canonicalPlatform(g.platform) as GroupPlatform"
                  :subscription-type="(g.subscription_type || 'standard') as SubscriptionType"
                  :rate-multiplier="g.rate_multiplier"
                  :user-rate-multiplier="userGroupRates[g.id] ?? null"
                  always-show-rate
                />
              </div>
              <span v-if="section.groups.length === 0" class="text-xs text-gray-400">-</span>
            </div>
          </td>

          <!-- 支持模型 -->
          <td class="align-top px-4 py-3">
            <div class="flex flex-wrap gap-1">
              <SupportedModelChip
                v-for="m in section.supported_models"
                :key="`${section.platform}-${m.name}`"
                :model="m"
                :pricing-key-prefix="pricingKeyPrefix"
                :no-pricing-label="noPricingLabel"
                :show-platform="false"
                :platform-hint="canonicalPlatform(section.platform)"
              />
              <span v-if="section.supported_models.length === 0" class="text-xs text-gray-400">
                {{ noModelsLabel }}
              </span>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import SupportedModelChip from './SupportedModelChip.vue'
import type { UserAvailableChannel, UserAvailableGroup, UserChannelPlatformSection } from '@/api/channels'
import type { GroupPlatform, SubscriptionType } from '@/types'
import { platformBadgeClass } from '@/utils/platformColors'

const props = defineProps<{
  columns: {
    name: string
    description: string
    platform: string
    groups: string
    supportedModels: string
  }
  rows: UserAvailableChannel[]
  loading: boolean
  pricingKeyPrefix: string
  noPricingLabel: string
  noModelsLabel: string
  emptyLabel: string
  /** 用户专属倍率（group_id → multiplier）；无专属时由 GroupBadge 仅显示默认倍率。 */
  userGroupRates: Record<number, number>
}>()

// Suppress unused warning — props is accessed via template automatically but
// the explicit reference here keeps the linter from flagging userGroupRates.
void props.userGroupRates

const { t } = useI18n()

function exclusiveGroups(section: UserChannelPlatformSection): UserAvailableGroup[] {
  return section.groups.filter((g) => g.is_exclusive)
}

function publicGroups(section: UserChannelPlatformSection): UserAvailableGroup[] {
  return section.groups.filter((g) => !g.is_exclusive)
}

// API data is normally lowercase, but normalize defensively so a legacy or
// manually-created platform value still reaches the same icon/color mapping.
function canonicalPlatform(platform: string): string {
  return platform.trim().toLowerCase()
}
</script>

<style scoped>
.pricing-table-frame {
  overflow: auto;
}

.pricing-table {
  min-width: 64rem;
}

.pricing-table th:first-child {
  width: 11rem;
  text-align: center;
}

.pricing-table th:nth-child(2) { width: 13rem; }
.pricing-table th:nth-child(3) { width: 9rem; }

.channel-group:not(:last-child) tr:last-child td {
  border-bottom-color: var(--ssxz-border-strong);
}

.channel-section td {
  border-top: 1px solid color-mix(in srgb, var(--ssxz-border) 72%, transparent);
}

.channel-name {
  color: var(--ssxz-text-primary);
  font-weight: 700;
  text-align: center;
  vertical-align: middle;
}

.channel-description {
  color: var(--ssxz-text-muted);
  font-size: 0.78rem;
  line-height: 1.5;
  vertical-align: middle;
}

.pricing-empty {
  height: 12rem;
  color: var(--ssxz-text-muted);
  text-align: center;
}

.pricing-empty svg {
  display: block;
  margin: 0 auto 0.75rem;
}

.pricing-empty p { margin: 0; }
</style>
