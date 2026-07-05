import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { appStore, push } = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: {
      payment_enabled: true
    }
  },
  push: vi.fn()
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push })
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    props: ['name'],
    template: '<span class="icon-stub" :data-icon="name" />'
  }
}))

import UserDashboardQuickActions from '../UserDashboardQuickActions.vue'

function mountView() {
  return mount(UserDashboardQuickActions)
}

describe('UserDashboardQuickActions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    appStore.cachedPublicSettings = {
      payment_enabled: true
    }
  })

  it('routes to purchase when online payment is enabled', async () => {
    const wrapper = mountView()

    expect(wrapper.text()).toContain('充值余额')
    expect(wrapper.text()).toContain('购买套餐或查看订单')

    await wrapper.findAll('button')[2].trigger('click')

    expect(push).toHaveBeenCalledWith('/app/purchase')
  })

  it('routes to orders instead of purchase when online payment is disabled', async () => {
    appStore.cachedPublicSettings = {
      payment_enabled: false
    }
    const wrapper = mountView()

    expect(wrapper.text()).toContain('订单记录')
    expect(wrapper.text()).toContain('查看订单和到账状态')
    expect(wrapper.text()).not.toContain('充值余额')

    await wrapper.findAll('button')[2].trigger('click')

    expect(push).toHaveBeenCalledWith('/app/orders')
  })
})
