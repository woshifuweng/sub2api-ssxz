import { mount, RouterLinkStub } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import PaymentFlowShell from '../PaymentFlowShell.vue'

describe('PaymentFlowShell', () => {
  it('returns users to the add-balance entry without old recharge-center copy', () => {
    const wrapper = mount(PaymentFlowShell, {
      global: {
        stubs: {
          RouterLink: RouterLinkStub
        }
      },
      slots: {
        default: '<p>payment body</p>'
      }
    })

    const text = wrapper.text()
    expect(text).toContain('返回补充额度')
    expect(text).toContain('支付处理中')
    expect(text).not.toContain('返回充值中心')
    expect(wrapper.get('.payment-flow-brand [data-testid="brand-logo"]').exists()).toBe(true)
    expect(wrapper.find('.payment-flow-brand-mark').text()).not.toBe('S')

    const links = wrapper.findAllComponents(RouterLinkStub)
    expect(links).toHaveLength(2)
    expect(links.every((link) => link.props('to') === '/app/purchase')).toBe(true)
  })
})
