import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

import AmountInput from '../AmountInput.vue'
import PaymentMethodSelector from '../PaymentMethodSelector.vue'

describe('recharge controls', () => {
  it('keeps preset and custom amount values unchanged while applying F0 selection styling', async () => {
    const wrapper = mount(AmountInput, {
      props: { modelValue: null, amounts: [10, 50], min: 0, max: 0 }
    })

    await wrapper.findAll('button')[1].trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([50])

    await wrapper.setProps({ modelValue: 50 })
    expect(wrapper.findAll('button')[1].classes()).toContain('amount-choice-selected')

    await wrapper.get('input').setValue('12.34')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([12.34])
  })

  it('keeps the selected payment method value and provider icon while using the neutral F0 container', async () => {
    const wrapper = mount(PaymentMethodSelector, {
      props: {
        selected: 'alipay',
        methods: [
          { type: 'alipay', fee_rate: 0, available: true },
          { type: 'wxpay', fee_rate: 0, available: true }
        ]
      }
    })

    const methods = wrapper.findAll('button')
    expect(methods[0].classes()).toContain('payment-method-selected')
    expect(methods[0].find('img').attributes('src')).toContain('alipay')

    await methods[1].trigger('click')
    expect(wrapper.emitted('select')?.at(-1)).toEqual(['wxpay'])
  })

  it('keeps custom payment methods within the responsive grid and shows their configured name', () => {
    const wrapper = mount(PaymentMethodSelector, {
      props: {
        selected: 'ldc',
        methods: [{ type: 'ldc', display_name: 'LDC Pay', fee_rate: 0, available: true }]
      }
    })

    expect(wrapper.get('[data-testid="payment-method-grid"]').classes()).toEqual(
      expect.arrayContaining(['grid', 'sm:grid-cols-3', 'lg:grid-cols-4'])
    )
    expect(wrapper.get('button').attributes('title')).toBe('LDC Pay')
    expect(wrapper.get('[data-testid="payment-method-label"]').text()).toBe('LDC Pay')
    expect(wrapper.get('button').classes()).toContain('min-w-0')
  })
})
