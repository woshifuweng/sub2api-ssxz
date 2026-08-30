import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'

import ModelPricingView from '../ModelPricingView.vue'

const { getAvailable, getUserGroupRates, showError } = vi.hoisted(() => ({
  getAvailable: vi.fn(),
  getUserGroupRates: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/channels', () => ({
  default: { getAvailable },
}))

vi.mock('@/api/groups', () => ({
  default: { getUserGroupRates },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

const GroupOptionItemStub = defineComponent({
  name: 'GroupOptionItem',
  props: {
    subscriptionType: String,
  },
  template: '<div />',
})

function availableChannels(subscriptionType: string) {
  return [{
    name: 'Test channel',
    description: '',
    platforms: [{
      platform: 'anthropic',
      groups: [{
        id: 1,
        name: 'Test group',
        platform: 'anthropic',
        subscription_type: subscriptionType,
        rate_multiplier: 1,
        is_exclusive: false,
      }],
      supported_models: [{
        name: 'claude-test',
        platform: 'anthropic',
        pricing: null,
      }],
    }],
  }]
}

async function renderedSubscriptionType(subscriptionType: string) {
  getAvailable.mockResolvedValue(availableChannels(subscriptionType))
  const wrapper = mount(ModelPricingView, {
    global: {
      stubs: {
        AppSectionShell: { template: '<section><slot /></section>' },
        GroupBadge: true,
        GroupOptionItem: GroupOptionItemStub,
        Icon: true,
        PlatformIcon: true,
      },
    },
  })

  await flushPromises()
  await wrapper.get('.group-picker__trigger').trigger('click')

  return wrapper.getComponent(GroupOptionItemStub).props('subscriptionType')
}

describe('ModelPricingView subscription type', () => {
  beforeEach(() => {
    getAvailable.mockReset()
    getUserGroupRates.mockReset()
    showError.mockReset()
    getUserGroupRates.mockResolvedValue({})
  })

  it('preserves subscription groups', async () => {
    expect(await renderedSubscriptionType('subscription')).toBe('subscription')
  })

  it('falls back to standard for unknown subscription types', async () => {
    expect(await renderedSubscriptionType('legacy')).toBe('standard')
  })
})
