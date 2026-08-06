import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('@/components/user/AppSectionShell.vue', () => ({
  default: {
    name: 'AppSectionShell',
    props: ['title', 'subtitle', 'eyebrow', 'icon'],
    template: `
      <main data-testid="app-section-shell">
        <h1>{{ title }}</h1>
        <p>{{ subtitle }}</p>
        <slot />
      </main>
    `,
  },
}))

import AppPurchaseView from '../AppPurchaseView.vue'

describe('AppPurchaseView', () => {
  it('embeds the hosted recharge shop inside the shared user shell', () => {
    const wrapper = mount(AppPurchaseView)

    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    const frame = wrapper.find('iframe[title="充值中心"]')
    expect(frame.exists()).toBe(true)
    expect(frame.attributes('src')).toBe('https://pay.ldxp.cn/shop/VT7XKDFI')
    expect(frame.attributes('referrerpolicy')).toBe('no-referrer')
  })

  it('keeps the customer-facing recharge title and does not mount legacy checkout content', () => {
    const wrapper = mount(AppPurchaseView)
    const text = wrapper.text()

    expect(text).toContain('补充额度 / 订阅')
    expect(text).toContain('充值中心')
    expect(wrapper.findComponent({ name: 'PaymentCheckoutContent' }).exists()).toBe(false)
  })
})
