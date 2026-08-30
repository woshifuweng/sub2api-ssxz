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
  it('renders the eight-column catalog and provider tab', () => {
    const wrapper = mountTable()
    expect(wrapper.findAll('thead th')).toHaveLength(8)
    expect(wrapper.get('[data-provider-tab="anthropic"]').text()).toContain('1')
    expect(wrapper.get('[data-testid="model-row"]').attributes('data-model')).toBe('claude-test')
    expect(wrapper.text()).toContain('Exclusive Pro')
    expect(wrapper.text()).toContain('200K')
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
