import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('@/components/user/AppSectionShell.vue', () => ({
  default: {
    name: 'AppSectionShell',
    props: ['title', 'subtitle', 'eyebrow', 'icon'],
    template: '<main data-testid="app-section-shell"><slot /></main>'
  }
}))

vi.mock('../PaymentCheckoutContent.vue', () => ({
  default: {
    name: 'PaymentCheckoutContent',
    props: ['variant'],
    template: '<section data-testid="payment-checkout-content" :data-variant="variant" />'
  }
}))

import AppPurchaseView from '../AppPurchaseView.vue'

describe('AppPurchaseView', () => {
  it('wraps the shared checkout content in the user workspace shell', () => {
    const wrapper = mount(AppPurchaseView)

    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    const checkout = wrapper.find('[data-testid="payment-checkout-content"]')
    expect(checkout.exists()).toBe(true)
    expect(checkout.attributes('data-variant')).toBe('workspace')
  })
})
