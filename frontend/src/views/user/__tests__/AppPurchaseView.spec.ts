import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { appStore } = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: {
      payment_enabled: true as boolean,
      purchase_subscription_enabled: false as boolean,
    },
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
}))

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
    `
  }
}))

vi.mock('../PaymentCheckoutContent.vue', () => ({
  default: {
    name: 'PaymentCheckoutContent',
    props: ['variant'],
    template: '<section data-testid="payment-checkout-content" :data-variant="variant" />'
  }
}))

vi.mock('../PurchaseSubscriptionView.vue', () => ({
  default: {
    name: 'PurchaseSubscriptionView',
    props: {
      embedded: Boolean
    },
    template: '<section data-testid="legacy-purchase-subscription" :data-embedded="String(embedded)" />'
  }
}))

import AppPurchaseView from '../AppPurchaseView.vue'

function mountView() {
  return mount(AppPurchaseView, {
    global: {
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>'
        }
      }
    }
  })
}

describe('AppPurchaseView', () => {
  beforeEach(() => {
    appStore.cachedPublicSettings = {
      payment_enabled: true,
      purchase_subscription_enabled: false,
    }
  })

  it('wraps the shared checkout content in the user workspace shell', () => {
    const wrapper = mountView()

    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    const checkout = wrapper.find('[data-testid="payment-checkout-content"]')
    expect(checkout.exists()).toBe(true)
    expect(checkout.attributes('data-variant')).toBe('workspace')
  })

  it('uses customer-facing billing copy instead of implementation notes', () => {
    const wrapper = mountView()
    const text = wrapper.text()

    expect(text).toContain('补充额度')
    expect(text).toContain('账户变化以账户记录为准')
    expect(text).not.toContain('新版工作台')
    expect(text).not.toContain('支付链路')
    expect(text).not.toContain('账务逻辑')
  })

  it('shows a safe disabled state without mounting checkout when payment is off', () => {
    appStore.cachedPublicSettings = {
      payment_enabled: false,
      purchase_subscription_enabled: false,
    }

    const wrapper = mountView()
    const text = wrapper.text()

    expect(text).toContain('当前可用方式：兑换码')
    expect(text).toContain('充值、兑换和账户调整记录可在账户记录查看')
    expect(text).toContain('去兑换额度')
    expect(text).toContain('查看账户记录')
    const hrefs = wrapper.findAll('a').map((link) => link.attributes('href'))
    expect(hrefs).toContain('/app/redeem')
    expect(hrefs).toContain('/app/orders')
    expect(wrapper.find('[data-testid="payment-checkout-content"]').exists()).toBe(false)
  })

  it('keeps the legacy subscription purchase entry inside the user workspace shell when configured', () => {
    appStore.cachedPublicSettings = {
      payment_enabled: false,
      purchase_subscription_enabled: true,
    }

    const wrapper = mountView()

    const legacyPurchase = wrapper.find('[data-testid="legacy-purchase-subscription"]')
    expect(legacyPurchase.exists()).toBe(true)
    expect(legacyPurchase.attributes('data-embedded')).toBe('true')
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="payment-checkout-content"]').exists()).toBe(false)
  })
})
