import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: {
    name: 'AppLayout',
    template: '<main data-testid="legacy-app-layout"><slot /></main>'
  }
}))

vi.mock('../PaymentCheckoutContent.vue', () => ({
  default: {
    name: 'PaymentCheckoutContent',
    props: ['variant'],
    template: '<section data-testid="payment-checkout-content" :data-variant="variant" />'
  }
}))

import PaymentView from '../PaymentView.vue'

describe('PaymentView', () => {
  it('keeps the legacy purchase route wrapped in the existing AppLayout shell', () => {
    const wrapper = mount(PaymentView)

    expect(wrapper.find('[data-testid="legacy-app-layout"]').exists()).toBe(true)
    const checkout = wrapper.find('[data-testid="payment-checkout-content"]')
    expect(checkout.exists()).toBe(true)
    expect(checkout.attributes('data-variant')).toBeUndefined()
  })
})
