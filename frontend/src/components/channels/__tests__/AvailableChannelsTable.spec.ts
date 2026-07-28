import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { UserAvailableChannel } from '@/api/channels'
import AvailableChannelsTable from '../AvailableChannelsTable.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

const rows: UserAvailableChannel[] = [
  {
    name: 'Claude channel A',
    description: '',
    platforms: [{
      platform: 'anthropic',
      groups: [{
        id: 11,
        name: 'Claude 满血池',
        description: '满血高质量',
        platform: 'anthropic',
        subscription_type: 'standard',
        rate_multiplier: 1.2,
        is_exclusive: true
      }],
      supported_models: [{
        name: 'claude-opus-4-8',
        platform: 'anthropic',
        context_length: 200000,
        max_output_tokens: 128000,
        pricing: {
          billing_mode: 'token',
          input_price: 0.000005,
          output_price: 0.000025,
          cache_write_price: 0.00000625,
          cache_read_price: 0.0000005,
          image_output_price: null,
          per_request_price: null,
          intervals: []
        }
      }]
    }]
  },
  {
    name: 'Claude channel B',
    description: '',
    platforms: [{
      platform: 'anthropic',
      groups: [{
        id: 12,
        name: 'Kiro 高缓池',
        description: '高缓性价比',
        platform: 'anthropic',
        subscription_type: 'standard',
        rate_multiplier: 0.8,
        is_exclusive: false
      }],
      supported_models: [{
        name: 'claude-opus-4-8',
        platform: 'anthropic',
        context_length: 200000,
        max_output_tokens: 128000,
        pricing: {
          billing_mode: 'token',
          input_price: 0.000005,
          output_price: 0.000025,
          cache_write_price: 0.00000625,
          cache_read_price: 0.0000005,
          image_output_price: null,
          per_request_price: null,
          intervals: []
        }
      }]
    }]
  },
  {
    name: 'OpenAI channel',
    description: '',
    platforms: [{
      platform: 'openai',
      groups: [{
        id: 10,
        name: 'GPT Pro 池',
        description: '稳定满血',
        platform: 'openai',
        subscription_type: 'standard',
        rate_multiplier: 1,
        is_exclusive: false
      }],
      supported_models: [{
        name: 'gpt-5.5',
        platform: 'openai',
        pricing: {
          billing_mode: 'token',
          input_price: 0.000005,
          output_price: 0.00003,
          cache_write_price: null,
          cache_read_price: null,
          image_output_price: null,
          per_request_price: null,
          intervals: []
        }
      }]
    }]
  }
]

afterEach(() => {
  document.body.innerHTML = ''
})

describe('AvailableChannelsTable', () => {
  it('deduplicates a model across channels and defaults to the lowest effective group rate', () => {
    const wrapper = mount(AvailableChannelsTable, {
      props: {
        rows,
        loading: false,
        emptyLabel: '暂无模型',
        searchQuery: '',
        userGroupRates: { 11: 0.7 }
      }
    })

    expect(wrapper.findAll('[data-testid="model-row"]')).toHaveLength(2)
    const claudeRow = wrapper.find('[data-model="claude-opus-4-8"]')
    expect(claudeRow.text()).toContain('Claude 满血池')
    expect(claudeRow.text()).toContain('$3.5')
    expect(claudeRow.text()).toContain('200K')
    expect(claudeRow.text()).toContain('128K')
    expect(claudeRow.find('.model-catalog__provider').classes()).toContain('model-catalog__provider--anthropic')
    expect(claudeRow.find('.model-group-rate').classes()).toContain('model-group-rate--discount')
  })

  it('updates displayed prices when the selected group changes', async () => {
    const wrapper = mount(AvailableChannelsTable, {
      attachTo: document.body,
      props: {
        rows,
        loading: false,
        emptyLabel: '暂无模型',
        searchQuery: '',
        userGroupRates: {}
      }
    })

    const claudeRow = wrapper.find('[data-model="claude-opus-4-8"]')
    expect(claudeRow.text()).toContain('Kiro 高缓池')
    expect(claudeRow.text()).toContain('$4')

    await claudeRow.find('.select-trigger').trigger('click')
    const fullPoolOption = Array.from(document.body.querySelectorAll<HTMLElement>('.select-option'))
      .find((element) => element.textContent?.includes('Claude 满血池'))
    expect(fullPoolOption).toBeTruthy()
    expect(document.body.textContent).toContain('满血高质量')
    fullPoolOption?.click()
    await wrapper.vm.$nextTick()

    expect(claudeRow.text()).toContain('Claude 满血池')
    expect(claudeRow.text()).toContain('$6')
    expect(claudeRow.find('.model-group-rate').classes()).toContain('model-group-rate--premium')
  })

  it('filters rows with dynamic provider tabs and leaves unknown context as a dash', async () => {
    const wrapper = mount(AvailableChannelsTable, {
      props: {
        rows,
        loading: false,
        emptyLabel: '暂无模型',
        searchQuery: '',
        userGroupRates: {}
      }
    })

    await wrapper.find('[data-provider-tab="openai"]').trigger('click')

    expect(wrapper.findAll('[data-testid="model-row"]')).toHaveLength(1)
    expect(wrapper.find('[data-model="gpt-5.5"]').text()).toContain('—')
    expect(wrapper.find('[data-model="claude-opus-4-8"]').exists()).toBe(false)
  })
})

const rows: UserAvailableChannel[] = [
  {
    name: 'Primary channel',
    description: 'Fast and reliable access',
    platforms: [
      {
        platform: 'anthropic',
        groups: [
          {
            id: 1,
            name: 'Exclusive Pro',
            platform: 'anthropic',
            subscription_type: 'standard',
            rate_multiplier: 1.2,
            peak_rate_enabled: true,
            peak_start: '08:00',
            peak_end: '10:00',
            peak_rate_multiplier: 1.5,
            is_exclusive: true,
          },
          {
            id: 2,
            name: 'Public',
            platform: 'anthropic',
            subscription_type: 'standard',
            rate_multiplier: 1,
            peak_rate_enabled: false,
            peak_start: '',
            peak_end: '',
            peak_rate_multiplier: 1,
            is_exclusive: false,
          },
        ],
        supported_models: [{ name: 'claude-test', platform: 'anthropic', pricing: null }],
      },
    ],
  },
]

const baseProps = {
  columns: {
    name: 'Channel',
    description: 'Description',
    platform: 'Platform',
    groups: 'Groups and rates',
    supportedModels: 'Models and pricing',
  },
  rows,
  loading: false,
  pricingKeyPrefix: 'availableChannels.pricing',
  noPricingLabel: 'No pricing',
  noModelsLabel: 'No models',
  emptyLabel: 'No channels',
  userGroupRates: { 1: 0.8 },
}

function mountTable(props = {}) {
  return mount(AvailableChannelsTable, {
    props: { ...baseProps, ...props },
    global: {
      plugins: [createPinia()],
      stubs: {
        Icon: { props: ['name'], template: '<i :data-icon="name" />' },
        PlatformIcon: { template: '<i data-platform-icon />' },
        GroupBadge: {
          props: ['name', 'rateMultiplier', 'userRateMultiplier'],
          template:
            '<span data-group-badge>{{ name }}:{{ rateMultiplier }}:{{ userRateMultiplier }}</span>',
        },
        SupportedModelChip: {
          props: ['model', 'noPricingLabel'],
          template: '<span data-model-chip>{{ model.name }}:{{ noPricingLabel }}</span>',
        },
      },
    },
  })
}

describe('AvailableChannelsTable responsive surfaces', () => {
  it('keeps the five-column table as the desktop-only surface', () => {
    const wrapper = mountTable()
    const desktop = wrapper.get('[data-testid="desktop-channels"]')

    // TablePageLayout has a more-specific `display: table` rule for descendants,
    // so these display utilities must remain important at both breakpoints.
    expect(desktop.classes()).toContain('!hidden')
    expect(desktop.classes()).toContain('lg:!table')
    expect(desktop.findAll('thead th')).toHaveLength(5)
    expect(desktop.text()).toContain('Primary channel')
    expect(desktop.text()).toContain('Fast and reliable access')
    expect(desktop.findAll('[data-group-badge]')).toHaveLength(2)
    expect(desktop.get('[data-model-chip]').text()).toContain('claude-test:No pricing')
  })

  it('renders a mobile-only readable surface with groups, rates, peaks, and model pricing chips', () => {
    const wrapper = mountTable()
    const mobile = wrapper.get('[data-testid="mobile-channels"]')

    expect(mobile.classes()).toContain('lg:hidden')
    expect(mobile.classes()).toContain('overflow-x-hidden')
    expect(mobile.text()).toContain('Primary channel')
    expect(mobile.text()).toContain('Fast and reliable access')
    expect(mobile.text()).toContain('Groups and rates')
    expect(mobile.text()).toContain('Models and pricing')
    expect(mobile.text()).toContain('availableChannels.exclusive')
    expect(mobile.text()).toContain('availableChannels.public')
    expect(mobile.get('[data-group-badge]').text()).toBe('Exclusive Pro:1.2:0.8')
    expect(mobile.findAll('[data-group-badge]')).toHaveLength(2)
    expect(mobile.get('[data-icon="clock"]')).toBeTruthy()
    expect(mobile.text()).toContain('08:00')
    expect(mobile.text()).toContain('10:00')
    expect(mobile.text()).toContain('×1.5')
    expect(mobile.get('[data-model-chip]').text()).toBe('claude-test:No pricing')
    expect(mobile.findAll('.max-w-full')).not.toHaveLength(0)
  })

  it('keeps the mobile placeholders when a platform has no groups or models', () => {
    const wrapper = mountTable({
      rows: [
        {
          name: 'Fallback channel',
          description: '',
          platforms: [{ platform: 'openai', groups: [], supported_models: [] }],
        },
      ],
    })
    const mobile = wrapper.get('[data-testid="mobile-channels"]')

    expect(mobile.text()).toContain('Fallback channel')
    expect(mobile.text()).toContain('openai')
    expect(mobile.text()).toContain('No models')
    expect(mobile.findAll('dd')[0].text()).toBe('-')
  })

  it('provides loading and empty states on both responsive surfaces', async () => {
    const wrapper = mountTable({ loading: true, rows: [] })

    expect(wrapper.get('[data-testid="desktop-channels"] [data-icon="refresh"]')).toBeTruthy()
    expect(wrapper.get('[data-testid="mobile-loading"] [data-icon="refresh"]')).toBeTruthy()

    await wrapper.setProps({ loading: false })

    expect(wrapper.get('[data-testid="desktop-channels"]').text()).toContain('No channels')
    expect(wrapper.get('[data-testid="mobile-empty"]').text()).toContain('No channels')
  })
})
