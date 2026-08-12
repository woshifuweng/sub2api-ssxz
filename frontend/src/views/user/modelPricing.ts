import type {
  UserAvailableChannel,
  UserAvailableGroup,
  UserSupportedModel,
} from '@/api/channels'

export type PricingPlatform = 'anthropic' | 'openai'

export interface ModelPricingRow {
  key: string
  name: string
  platform: PricingPlatform
  providerLabel: string
  model: UserSupportedModel
  groups: UserAvailableGroup[]
}

const ANTHROPIC_PREFIX = /^claude-/i
const OPENAI_PREFIXES = [/^gpt-/i, /^o1(?:$|-)/i, /^o3(?:$|-)/i, /^deepseek-/i]

export function classifyModelPlatform(modelName: string, sourcePlatform?: string): PricingPlatform | null {
  const normalizedName = modelName.trim()
  if (ANTHROPIC_PREFIX.test(normalizedName)) return 'anthropic'
  if (OPENAI_PREFIXES.some((prefix) => prefix.test(normalizedName))) return 'openai'

  const normalizedPlatform = sourcePlatform?.trim().toLowerCase()
  if (normalizedPlatform === 'anthropic') return 'anthropic'
  if (normalizedPlatform === 'openai') return 'openai'
  return null
}

export function flattenPricingRows(channels: UserAvailableChannel[]): ModelPricingRow[] {
  const rows = new Map<string, ModelPricingRow>()

  for (const channel of channels) {
    for (const section of channel.platforms) {
      for (const model of section.supported_models) {
        const platform = classifyModelPlatform(model.name, model.platform || section.platform)
        if (!platform) continue

        const key = `${platform}:${model.name}`
        const existing = rows.get(key)
        if (existing) {
          const groupIds = new Set(existing.groups.map((group) => group.id))
          existing.groups = [
            ...existing.groups,
            ...section.groups.filter((group) => !groupIds.has(group.id)),
          ]
          continue
        }

        rows.set(key, {
          key,
          name: model.name,
          platform,
          providerLabel: platform === 'anthropic' ? 'Anthropic' : 'OpenAI',
          model,
          groups: [...section.groups],
        })
      }
    }
  }

  return [...rows.values()].sort((left, right) => left.name.localeCompare(right.name))
}

export function filterPricingRows(rows: ModelPricingRow[], query: string): ModelPricingRow[] {
  const normalizedQuery = query.trim().toLowerCase()
  if (!normalizedQuery) return rows
  return rows.filter((row) => (
    row.name.toLowerCase().includes(normalizedQuery) ||
    row.providerLabel.toLowerCase().includes(normalizedQuery) ||
    row.groups.some((group) => group.name.toLowerCase().includes(normalizedQuery))
  ))
}

export function pricePerMillion(basePrice: number | null | undefined, multiplier: number): number | null {
  if (basePrice === null || basePrice === undefined || !Number.isFinite(basePrice)) return null
  return basePrice * multiplier * 1_000_000
}

export function formatPrice(basePrice: number | null | undefined, multiplier: number): string {
  const value = pricePerMillion(basePrice, multiplier)
  if (value === null) return '-'
  return `$${value.toLocaleString('en-US', {
    maximumFractionDigits: value >= 1 ? 4 : 8,
  })} / 1M token`
}

export function formatContext(model: UserSupportedModel): string {
  const extendedModel = model as UserSupportedModel & {
    context_length?: number | null
    context_window?: number | null
  }
  const context = extendedModel.context_length ?? extendedModel.context_window
  if (!context || !Number.isFinite(context)) return '-'
  return context.toLocaleString('en-US')
}
