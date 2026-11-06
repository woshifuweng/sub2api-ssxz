import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import type { UserAvailableChannel } from '@/api/channels'
import AvailableChannelsTable from '../AvailableChannelsTable.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const rows: UserAvailableChannel[] = [{
  name: 'Primary channel',
  description: 'Fast and reliable access',
  platforms: [{
    platform: 'anthropic',
    groups: [{
      id: 1,
      name: 'Exclusive Pro',
      description: 'Priority group',
      platform: 'anthropic',
      subscription_type: 'standard',
      rate_multiplier: 1.2,
      is_exclusive: true,
    }],
    supported_models: [{
      name: 'claude-test',
      platform: 'anthropic',
      context_length: 200000,
      max_output_tokens: 8192,
      pricing: {
        billing_mode: 'token',
        input_price: 1,
        output_price: 2,
        cache_write_price: 0.5,
        cache_read_price: 0.1,
        image_output_price: null,
        per_request_price: null,
        intervals: [],
      },
    }],
  }],
}]

function mountTable(props: Record<string, unknown> = {}) {
  return mount(AvailableChannelsTable, {
    props: {
      rows,
      loading: false,
      emptyLabel: 'No channels',
      userGroupRates: { 1: 0.8 },
      ...props,
    },
    global: {
      stubs: {
        Icon: { props: ['name'], template: '<i :data-icon="name" />' },
        ModelIcon: true,
        Select: {
          props: ['modelValue', 'options'],
          template: '<div data-select>{{ options?.[0]?.label }}</div>',
        },
      },
    },
  })
}

describe('AvailableChannelsTable model catalog', () => {
  it('shows image per-request prices before legacy image_output_price and honors independent image rates', () => {
    const images = structuredClone(rows)
    Object.assign(images[0].platforms[0].groups[0], { image_rate_independent: true, image_rate_multiplier: 1 })
    Object.assign(images[0].platforms[0].supported_models[0].pricing!, { billing_mode: 'image', per_request_price: .15, image_output_price: .15 })
    const wrapper = mountTable({ rows: images })
    expect(wrapper.text()).toContain('$0.15 / availableChannels.perRequestShort')
    expect(wrapper.text()).not.toContain('$120,000')
  })

  it('shows each image size price', () => {
    const images = structuredClone(rows)
    Object.assign(images[0].platforms[0].supported_models[0].pricing!, { billing_mode: 'image', intervals: [
      { tier_label: '1K', per_request_price: .15 }, { tier_label: '4K', per_request_price: .3 },
    ] })
    const wrapper = mountTable({ rows: images, userGroupRates: { 1: 1 } })
    expect(wrapper.text()).toContain('1K $0.15 · 4K $0.3')
  })
  it('respects an explicitly configured zero personal multiplier', () => {
    const wrapper = mountTable({ userGroupRates: { 1: 0 } })
    expect(wrapper.text()).toContain('$0 / 1M token')
    expect(wrapper.text()).not.toContain('$1,200,000')
  })
  it('keeps an unpriced model visible without advertising it as free', () => {
    const unpriced = structuredClone(rows)
    unpriced[0].platforms[0].supported_models[0].pricing = null
    const wrapper = mountTable({ rows: unpriced })
    expect(wrapper.findAll('[data-testid="model-row"]')).toHaveLength(1)
    expect(wrapper.text()).toContain('availableChannels.pricingPending')
    expect(wrapper.text()).not.toContain('$0')
  })
  it('renders the eight-column catalog and provider tab', () => {
    const wrapper = mountTable()
    expect(wrapper.findAll('thead th')).toHaveLength(8)
    expect(wrapper.get('[data-provider-tab="anthropic"]').text()).toContain('1')
    expect(wrapper.get('[data-testid="model-row"]').attributes('data-model')).toBe('claude-test')
    expect(wrapper.text()).toContain('Exclusive Pro')
    expect(wrapper.text()).toContain('200K')
  })

  it('classifies Grok-named models under the Grok provider tab', () => {
    const grokChannels: UserAvailableChannel[] = [{
      name: 'Grok channel',
      description: '',
      platforms: [
        {
          platform: 'openai',
          groups: [{
            id: 10,
            name: 'Grok via OpenAI-compatible pool',
            platform: 'openai',
            subscription_type: 'standard',
            rate_multiplier: 1,
            is_exclusive: false,
          }],
          supported_models: [{ name: 'grok-4.6', platform: 'openai', pricing: null }],
        },
        {
          platform: 'grok',
          groups: [{
            id: 11,
            name: 'Grok native pool',
            platform: 'grok',
            subscription_type: 'standard',
            rate_multiplier: 1,
            is_exclusive: false,
          }],
          supported_models: [{ name: 'grok-4.5', platform: 'grok', pricing: null }],
        },
      ],
    }]

    const wrapper = mountTable({ rows: grokChannels })

    expect(wrapper.find('[data-provider-tab="openai"]').exists()).toBe(false)
    expect(wrapper.get('[data-provider-tab="grok"]').text()).toContain('2')
    expect(wrapper.findAll('.model-catalog__provider')).toHaveLength(2)
    expect(wrapper.findAll('.model-catalog__provider').every((node) => node.text().includes('Grok'))).toBe(true)
  })

  it('filters models with the optional search query', async () => {
    const wrapper = mountTable()
    expect(wrapper.findAll('[data-testid="model-row"]')).toHaveLength(1)
    await wrapper.setProps({ searchQuery: 'missing-model' })
    expect(wrapper.findAll('[data-testid="model-row"]')).toHaveLength(0)
    expect(wrapper.text()).toContain('No channels')
  })

  it('provides loading and empty states', async () => {
    const wrapper = mountTable({ loading: true, rows: [] })
    expect(wrapper.get('[data-icon="refresh"]')).toBeTruthy()
    await wrapper.setProps({ loading: false })
    expect(wrapper.text()).toContain('No channels')
    expect(wrapper.get('[data-icon="inbox"]')).toBeTruthy()
  })
})
