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
