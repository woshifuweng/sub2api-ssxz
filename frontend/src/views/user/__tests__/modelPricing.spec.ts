import { describe, expect, it } from 'vitest'
import type { UserAvailableChannel } from '@/api/channels'
import {
  classifyModelPlatform,
  filterPricingRows,
  flattenPricingRows,
  formatPrice,
  pricePerMillion,
} from '../modelPricing'

function channel(overrides: Partial<UserAvailableChannel> = {}): UserAvailableChannel {
  return {
    name: 'open ai',
    description: '',
    platforms: [],
    ...overrides,
  }
}

describe('model pricing helpers', () => {
  it('classifies model names before using the backend platform label', () => {
    expect(classifyModelPlatform('claude-opus-4-1', 'openai')).toBe('anthropic')
    expect(classifyModelPlatform('gpt-5.5', 'anthropic')).toBe('openai')
    expect(classifyModelPlatform('o3-mini')).toBe('openai')
    expect(classifyModelPlatform('unknown-model', 'anthropic')).toBe('anthropic')
    expect(classifyModelPlatform('unknown-model', 'kiro')).toBeNull()
  })

  it('flattens duplicate model entries while merging group access', () => {
    const model = {
      name: 'gpt-5.5',
      platform: 'openai',
      pricing: { input_price: 0.000001, output_price: 0.000002 },
    } as never
    const firstGroup = { id: 1, name: 'GPT Plus', platform: 'openai', subscription_type: 'standard', rate_multiplier: 0.04, is_exclusive: false }
    const secondGroup = { ...firstGroup, id: 2, name: 'GPT Pro', rate_multiplier: 0.08 }
    const rows = flattenPricingRows([
      channel({ platforms: [{ platform: 'openai', groups: [firstGroup], supported_models: [model] }] }),
      channel({ platforms: [{ platform: 'openai', groups: [secondGroup], supported_models: [model] }] }),
    ])

    expect(rows).toHaveLength(1)
    expect(rows[0].groups.map((group) => group.id)).toEqual([1, 2])
  })

  it('filters by model, provider, and group name', () => {
    const rows = flattenPricingRows([
      channel({ platforms: [{
        platform: 'anthropic',
        groups: [{ id: 11, name: 'Claude 满血池', platform: 'anthropic', subscription_type: 'standard', rate_multiplier: 1.3, is_exclusive: false }],
        supported_models: [{ name: 'claude-sonnet-4-5', platform: 'anthropic', pricing: null } as never],
      }] }),
    ])
    expect(filterPricingRows(rows, 'sonnet')).toHaveLength(1)
    expect(filterPricingRows(rows, 'anthropic')).toHaveLength(1)
    expect(filterPricingRows(rows, '满血')).toHaveLength(1)
    expect(filterPricingRows(rows, 'gpt')).toHaveLength(0)
  })

  it('scales per-token prices by the group multiplier for 1M-token display', () => {
    expect(pricePerMillion(0.0000006, 1.3)).toBeCloseTo(0.78)
    expect(formatPrice(0.0000006, 1.3)).toBe('$0.78 / 1M token')
    expect(formatPrice(null, 1.3)).toBe('-')
  })
})
