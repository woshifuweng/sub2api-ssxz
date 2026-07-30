<template>
  <section class="model-catalog" aria-labelledby="model-catalog-title">
    <div class="model-catalog__tabs" role="tablist" :aria-label="t('availableChannels.providerTabsLabel')">
      <button
        v-for="provider in providerTabs"
        :key="provider.value"
        type="button"
        class="model-catalog__tab"
        :class="{ 'model-catalog__tab--active': activeProvider === provider.value }"
        role="tab"
        :aria-selected="activeProvider === provider.value"
        :data-provider-tab="provider.value"
        @click="activeProvider = provider.value"
      >
        {{ provider.label }}
        <span>{{ provider.count }}</span>
      </button>
    </div>

    <div class="f0-card model-catalog__frame">
      <div class="model-catalog__scroll">
        <table class="f0-table model-catalog__table">
          <thead>
            <tr>
              <th>{{ t('availableChannels.columns.model') }}</th>
              <th>{{ t('availableChannels.columns.provider') }}</th>
              <th>{{ t('availableChannels.columns.group') }}</th>
              <th>{{ t('availableChannels.columns.input') }}</th>
              <th>{{ t('availableChannels.columns.output') }}</th>
              <th>{{ t('availableChannels.columns.cache') }}</th>
              <th>{{ t('availableChannels.columns.image') }}</th>
              <th>{{ t('availableChannels.columns.context') }}</th>
            </tr>
          </thead>
          <tbody v-if="loading">
            <tr>
              <td colspan="8" class="model-catalog__empty">
                <Icon name="refresh" size="lg" class="inline-block animate-spin" />
              </td>
            </tr>
          </tbody>
          <tbody v-else-if="visibleRows.length === 0">
            <tr>
              <td colspan="8" class="model-catalog__empty">
                <span class="model-catalog__empty-icon"><Icon name="inbox" size="lg" /></span>
                <strong>{{ emptyLabel }}</strong>
                <small>{{ t('availableChannels.emptyDescription') }}</small>
              </td>
            </tr>
          </tbody>
          <tbody v-else>
            <tr
              v-for="row in visibleRows"
              :key="row.key"
              data-testid="model-row"
              :data-model="row.name"
            >
              <td class="model-catalog__model">
                <strong>{{ row.name }}</strong>
              </td>
              <td>
                <span :class="['model-catalog__provider', providerToneClass(row.platform)]">
                  <span class="model-catalog__provider-mark">
                    <ModelIcon :model="providerIconModel(row.platform)" size="18px" />
                  </span>
                  {{ providerLabel(row.platform) }}
                </span>
              </td>
              <td class="model-catalog__group-cell">
                <Select
                  :model-value="selectedOption(row)?.value"
                  :options="row.options"
                  value-key="value"
                  label-key="label"
                  searchable
                  :search-placeholder="t('availableChannels.groupSearchPlaceholder')"
                  :empty-text="t('availableChannels.groupSearchEmpty')"
                  :data-testid="`group-select-${row.key}`"
                  @update:model-value="selectPlatformGroup(row.platform, $event)"
                >
                  <template #selected="{ option }">
                    <span v-if="option" class="model-group-selected">
                      <span class="model-group-selected__name">{{ asModelGroupOption(option).group.name }}</span>
                      <span :class="['model-group-rate', rateToneClass(asModelGroupOption(option).effectiveRate)]">{{ formatRate(asModelGroupOption(option).effectiveRate) }}</span>
                    </span>
                  </template>
                  <template #option="{ option, selected }">
                    <div class="model-group-option">
                      <span class="model-catalog__provider-mark model-catalog__provider-mark--small">
                        <ModelIcon :model="providerIconModel(asModelGroupOption(option).group.platform)" size="16px" />
                      </span>
                      <span class="model-group-option__copy">
                        <span class="model-group-option__title">
                          <strong>{{ asModelGroupOption(option).group.name }}</strong>
                          <span :class="['model-group-rate', rateToneClass(asModelGroupOption(option).effectiveRate)]">{{ formatRate(asModelGroupOption(option).effectiveRate) }}</span>
                        </span>
                        <small>{{ asModelGroupOption(option).group.description || t('availableChannels.groupDescriptionFallback') }}</small>
                      </span>
                      <Icon v-if="selected" name="check" size="sm" class="model-group-option__check" />
                    </div>
                  </template>
                </Select>
              </td>
              <td class="model-catalog__price">{{ formatTokenPrice(selectedOption(row)?.pricing.input_price, selectedOption(row)?.effectiveRate) }}</td>
              <td class="model-catalog__price">{{ formatTokenPrice(selectedOption(row)?.pricing.output_price, selectedOption(row)?.effectiveRate) }}</td>
              <td class="model-catalog__price model-catalog__price--stacked">
                <span>{{ t('availableChannels.cacheWriteShort') }} {{ formatTokenPrice(selectedOption(row)?.pricing.cache_write_price, selectedOption(row)?.effectiveRate) }}</span>
                <span>{{ t('availableChannels.cacheReadShort') }} {{ formatTokenPrice(selectedOption(row)?.pricing.cache_read_price, selectedOption(row)?.effectiveRate) }}</span>
              </td>
              <td class="model-catalog__price">{{ formatImagePrice(selectedOption(row)) }}</td>
              <td class="model-catalog__context">
                <strong>{{ formatTokenLimit(row.contextLength) }}</strong>
                <small v-if="row.maxOutputTokens">{{ t('availableChannels.maxOutputShort') }} {{ formatTokenLimit(row.maxOutputTokens) }}</small>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type {
  UserAvailableChannel,
  UserAvailableGroup,
  UserSupportedModelPricing
} from '@/api/channels'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

interface ModelGroupOption extends Record<string, unknown> {
  value: number
  label: string
  description: string
  group: UserAvailableGroup
  pricing: UserSupportedModelPricing
  effectiveRate: number
}

const asModelGroupOption = (option: Record<string, unknown>): ModelGroupOption => option as ModelGroupOption

interface ModelCatalogRow {
  key: string
  name: string
  platform: string
  contextLength: number | null
  maxOutputTokens: number | null
  options: ModelGroupOption[]
}

const props = defineProps<{
  rows: UserAvailableChannel[]
  loading: boolean
  emptyLabel: string
  searchQuery: string
  userGroupRates: Record<number, number>
}>()

const { t } = useI18n()
const activeProvider = ref('all')
// Keyed by platform (e.g. 'anthropic', 'openai'). Selecting a group on any row
// updates all rows of the same platform — page-level selection, not per-row.
const selectedGroupByPlatform = ref<Record<string, number>>({})

const flatRows = computed<ModelCatalogRow[]>(() => {
  const models = new Map<string, ModelCatalogRow>()

  for (const channel of props.rows) {
    for (const section of channel.platforms) {
      for (const model of section.supported_models) {
        if (!model.pricing) continue
        const platform = (model.platform || section.platform).toLowerCase()
        const key = `${platform}:${model.name.toLowerCase()}`
        let row = models.get(key)
        if (!row) {
          row = {
            key,
            name: model.name,
            platform,
            contextLength: model.context_length ?? null,
            maxOutputTokens: model.max_output_tokens ?? null,
            options: []
          }
          models.set(key, row)
        } else {
          row.contextLength ??= model.context_length ?? null
          row.maxOutputTokens ??= model.max_output_tokens ?? null
        }

        for (const group of section.groups) {
          const effectiveRate = resolveEffectiveRate(group)
          const existing = row.options.find((option) => option.value === group.id)
          if (existing) continue
          row.options.push({
            value: group.id,
            label: group.name,
            description: group.description || '',
            group,
            pricing: model.pricing,
            effectiveRate
          })
        }
      }
    }
  }

  return Array.from(models.values())
    .filter((row) => row.options.length > 0)
    .map((row) => ({ ...row, options: [...row.options].sort(compareGroupOptions) }))
    .sort((a, b) => {
      const providerOrder = providerSortOrder(a.platform) - providerSortOrder(b.platform)
      return providerOrder || a.name.localeCompare(b.name, undefined, { numeric: true })
    })
})

watch(flatRows, (rows) => {
  const next = { ...selectedGroupByPlatform.value }
  const validPlatforms = new Set(rows.map((row) => row.platform))
  for (const platform of Object.keys(next)) {
    if (!validPlatforms.has(platform)) delete next[platform]
  }
  for (const platform of validPlatforms) {
    const options = rows
      .filter((row) => row.platform === platform)
      .flatMap((row) => row.options)
      .sort(compareGroupOptions)
    if (!options.some((option) => option.value === next[platform])) {
      next[platform] = options[0]?.value
    }
  }
  selectedGroupByPlatform.value = next
}, { immediate: true })

const providerTabs = computed(() => {
  const counts = new Map<string, number>()
  for (const row of flatRows.value) counts.set(row.platform, (counts.get(row.platform) || 0) + 1)
  const providers = Array.from(counts.entries())
    .sort(([a], [b]) => providerSortOrder(a) - providerSortOrder(b) || a.localeCompare(b))
    .map(([value, count]) => ({ value, label: providerLabel(value), count }))
  return [{ value: 'all', label: t('availableChannels.allProviders'), count: flatRows.value.length }, ...providers]
})

const visibleRows = computed(() => {
  const query = props.searchQuery.trim().toLowerCase()
  return flatRows.value.filter((row) => {
    if (activeProvider.value !== 'all' && row.platform !== activeProvider.value) return false
    if (!query) return true
    return row.name.toLowerCase().includes(query)
      || providerLabel(row.platform).toLowerCase().includes(query)
      || row.options.some((option) => `${option.group.name} ${option.group.description || ''}`.toLowerCase().includes(query))
  })
})

function resolveEffectiveRate(group: UserAvailableGroup): number {
  const override = props.userGroupRates[group.id]
  if (Number.isFinite(override) && override > 0) return override
  return Number.isFinite(group.rate_multiplier) && group.rate_multiplier > 0 ? group.rate_multiplier : 1
}

function compareGroupOptions(a: ModelGroupOption, b: ModelGroupOption): number {
  if (a.effectiveRate !== b.effectiveRate) return a.effectiveRate - b.effectiveRate
  if (a.group.is_exclusive !== b.group.is_exclusive) return a.group.is_exclusive ? -1 : 1
  return a.group.name.localeCompare(b.group.name) || a.group.id - b.group.id
}

function selectedOption(row: ModelCatalogRow): ModelGroupOption | undefined {
  const groupId = selectedGroupByPlatform.value[row.platform]
  return row.options.find((option) => option.value === groupId) || row.options[0]
}

function selectPlatformGroup(platform: string, value: string | number | boolean | null) {
  if (typeof value !== 'number') return
  selectedGroupByPlatform.value[platform] = value
}

function formatRate(rate: number): string {
  return `${new Intl.NumberFormat('en-US', { maximumFractionDigits: 3 }).format(rate)}x`
}

function formatTokenPrice(price: number | null | undefined, rate = 1): string {
  if (price == null || !Number.isFinite(price)) return '—'
  const perMillion = price * rate * 1_000_000
  return `$${new Intl.NumberFormat('en-US', { maximumFractionDigits: 6 }).format(perMillion)} / 1M token`
}

function formatImagePrice(option: ModelGroupOption | undefined): string {
  if (!option) return '—'
  if (option.pricing.image_output_price != null) {
    return formatTokenPrice(option.pricing.image_output_price, option.effectiveRate)
  }
  if (option.pricing.per_request_price != null) {
    const price = option.pricing.per_request_price * option.effectiveRate
    return `$${new Intl.NumberFormat('en-US', { maximumFractionDigits: 6 }).format(price)} / ${t('availableChannels.perRequestShort')}`
  }
  return '—'
}

function formatTokenLimit(value: number | null | undefined): string {
  if (!value) return '—'
  if (value % 1000 === 0 && value < 1_000_000) return `${value / 1000}K`
  return new Intl.NumberFormat('en-US').format(value)
}

function providerSortOrder(platform: string): number {
  const order: Record<string, number> = { anthropic: 0, openai: 1, gemini: 2, google: 2 }
  return order[platform] ?? 50
}

function providerLabel(platform: string): string {
  const labels: Record<string, string> = {
    anthropic: 'Anthropic',
    openai: 'OpenAI',
    gemini: 'Google',
    google: 'Google'
  }
  return labels[platform] || platform.charAt(0).toUpperCase() + platform.slice(1)
}

function providerToneClass(platform: string): string {
  if (platform === 'anthropic') return 'model-catalog__provider--anthropic'
  if (platform === 'openai') return 'model-catalog__provider--openai'
  if (platform === 'gemini' || platform === 'google') return 'model-catalog__provider--google'
  return 'model-catalog__provider--neutral'
}

function rateToneClass(rate: number): string {
  if (rate < 1) return 'model-group-rate--discount'
  if (rate > 1) return 'model-group-rate--premium'
  return 'model-group-rate--standard'
}

function providerIconModel(platform: string): string {
  const models: Record<string, string> = {
    anthropic: 'claude',
    openai: 'gpt-5',
    gemini: 'gemini',
    google: 'gemini'
  }
  return models[platform] || platform
}
</script>

<style scoped>
.model-catalog {
  display: grid;
  gap: 1rem;
}

.model-catalog__tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.model-catalog__tab {
  display: inline-flex;
  min-height: 2.25rem;
  align-items: center;
  gap: 0.5rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: transparent;
  color: var(--ssxz-text-secondary);
  font-size: 0.78rem;
  font-weight: 650;
  padding: 0.45rem 0.85rem;
  transition: border-color 160ms ease, background-color 160ms ease, color 160ms ease;
}

.model-catalog__tab span {
  color: var(--ssxz-text-muted);
  font-size: 0.7rem;
  font-variant-numeric: tabular-nums;
}

.model-catalog__tab:hover {
  border-color: var(--ssxz-border-strong);
  color: var(--ssxz-text-primary);
}

.model-catalog__tab--active {
  border-color: var(--ssxz-action);
  background: var(--ssxz-action);
  color: var(--ssxz-action-text);
}

.model-catalog__tab--active span { color: currentColor; opacity: 0.72; }

.model-catalog__frame {
  overflow: hidden;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.75rem;
  background: transparent;
}

.model-catalog__scroll {
  max-height: min(68vh, 54rem);
  overflow: auto;
}

.model-catalog__table {
  min-width: 70.5rem;
  table-layout: fixed;
}

.model-catalog__table th {
  position: sticky;
  z-index: 2;
  top: 0;
  background: var(--ssxz-surface-raised);
  color: var(--ssxz-text-muted);
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0;
  text-align: left;
}

.model-catalog__table th:nth-child(1) { width: 10rem; }
.model-catalog__table th:nth-child(2) { width: 6.5rem; }
.model-catalog__table th:nth-child(3) { width: 14rem; }
.model-catalog__table th:nth-child(4),
.model-catalog__table th:nth-child(5) { width: 8.25rem; }
.model-catalog__table th:nth-child(6) { width: 9.5rem; }
.model-catalog__table th:nth-child(7) { width: 7rem; }
.model-catalog__table th:nth-child(8) { width: 7rem; }

.model-catalog__table td {
  height: 4.75rem;
  border-color: var(--ssxz-border);
  color: var(--ssxz-text-secondary);
  font-size: 0.78rem;
  padding: 0.75rem 1rem;
  vertical-align: middle;
}

.model-catalog__table tbody tr:hover td {
  background: color-mix(in srgb, var(--ssxz-surface-raised) 68%, transparent);
}

.model-catalog__model strong {
  color: var(--ssxz-text-primary);
  font-family: var(--ssxz-font-mono, ui-monospace, monospace);
  font-size: 0.8rem;
  font-weight: 650;
}

.model-catalog__provider,
.model-catalog__provider-mark {
  display: inline-flex;
  align-items: center;
}

.model-catalog__provider {
  gap: 0.45rem;
  border: 1px solid transparent;
  border-radius: 999px;
  color: var(--ssxz-text-primary);
  font-weight: 650;
  padding: 0.2rem 0.55rem 0.2rem 0.28rem;
}

.model-catalog__provider--anthropic {
  border-color: rgba(212, 168, 67, 0.45);
  background: rgba(212, 168, 67, 0.1);
  color: #d4a843;
}

.model-catalog__provider--openai {
  border-color: rgba(16, 163, 127, 0.45);
  background: rgba(16, 163, 127, 0.1);
  color: #10a37f;
}

.model-catalog__provider--google {
  border-color: rgba(66, 133, 244, 0.45);
  background: rgba(66, 133, 244, 0.1);
  color: #4285f4;
}

.model-catalog__provider--neutral {
  border-color: var(--ssxz-border);
}

.model-catalog__provider-mark {
  width: 1.8rem;
  height: 1.8rem;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 0.5rem;
  background: transparent;
}

.model-catalog__provider-mark--small { width: 1.65rem; height: 1.65rem; }

.model-catalog__group-cell :deep(.select-trigger) {
  min-height: 2.5rem;
  border-color: var(--ssxz-border);
  background: transparent;
  transition: border-color 160ms ease, background-color 160ms ease;
}

.model-catalog__group-cell :deep(.select-trigger:hover) {
  border-color: var(--ssxz-border-strong);
  background: color-mix(in srgb, var(--ssxz-surface-raised) 38%, transparent);
}

.model-group-selected,
.model-group-option__title {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.model-group-selected__name {
  overflow: hidden;
  color: var(--ssxz-text-primary);
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-group-rate {
  flex: 0 0 auto;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  color: var(--ssxz-text-secondary);
  font-size: 0.68rem;
  font-variant-numeric: tabular-nums;
  padding: 0.15rem 0.42rem;
}

.model-group-rate--discount {
  border-color: #10A37F;
  background: rgba(16, 163, 127, 0.15);
  color: #10A37F;
}

.model-group-rate--premium {
  border-color: #D4A843;
  background: rgba(212, 168, 67, 0.15);
  color: #D4A843;
}

.model-group-rate--standard {
  border-color: var(--ssxz-border);
  background: transparent;
  color: var(--ssxz-text-secondary);
}

.model-group-option {
  display: grid;
  width: 100%;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.65rem;
}

.model-group-option__copy { display: grid; min-width: 0; gap: 0.2rem; }
.model-group-option__copy strong { color: var(--ssxz-text-primary); font-size: 0.78rem; }
.model-group-option__copy small { overflow: hidden; color: var(--ssxz-text-muted); font-size: 0.68rem; text-overflow: ellipsis; white-space: nowrap; }
.model-group-option__check { color: var(--ssxz-action); }

.model-catalog__price,
.model-catalog__context {
  color: var(--ssxz-text-primary) !important;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.model-catalog__price--stacked,
.model-catalog__context { display: grid; gap: 0.25rem; }
.model-catalog__price--stacked span { color: var(--ssxz-text-secondary); }
.model-catalog__context strong { font-size: 0.8rem; font-weight: 650; }
.model-catalog__context small { color: var(--ssxz-text-muted); font-size: 0.68rem; }

.model-catalog__empty {
  height: 16rem !important;
  text-align: center;
}

.model-catalog__empty > * { display: block; margin-inline: auto; }
.model-catalog__empty-icon {
  display: grid;
  width: 3.25rem;
  height: 3.25rem;
  place-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.75rem;
  color: var(--ssxz-action);
}
.model-catalog__empty strong { margin-top: 0.75rem; color: var(--ssxz-text-primary); }
.model-catalog__empty small { max-width: 32rem; margin-top: 0.35rem; color: var(--ssxz-text-muted); }

@media (max-width: 640px) {
  .model-catalog__tabs { flex-wrap: nowrap; overflow-x: auto; padding-bottom: 0.25rem; }
  .model-catalog__tab { flex: 0 0 auto; }
  .model-catalog__scroll { max-height: none; }
}
</style>
