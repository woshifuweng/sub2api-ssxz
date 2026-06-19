import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('payment locale copy', () => {
  it('provides zh copy for the app purchase checkout shell', () => {
    const visibleCopy = [
      zh.payment.tabTopUp,
      zh.payment.tabSubscribe,
      zh.payment.rechargeAccount,
      zh.payment.currentBalance,
      zh.payment.notAvailable,
      zh.payment.createOrder,
      zh.payment.result.title,
      zh.payment.result.failed,
      zh.payment.qr.cancelled
    ].join(' ')

    expect(visibleCopy).toContain('余额充值')
    expect(visibleCopy).toContain('套餐订阅')
    expect(visibleCopy).toContain('充值账户')
    expect(visibleCopy).not.toContain('payment.')
  })

  it('provides en copy for the app purchase checkout shell', () => {
    const visibleCopy = [
      en.payment.tabTopUp,
      en.payment.tabSubscribe,
      en.payment.rechargeAccount,
      en.payment.currentBalance,
      en.payment.notAvailable,
      en.payment.createOrder,
      en.payment.result.title,
      en.payment.result.failed,
      en.payment.qr.cancelled
    ].join(' ')

    expect(visibleCopy).toContain('Top up balance')
    expect(visibleCopy).toContain('Subscription')
    expect(visibleCopy).toContain('Recharge account')
    expect(visibleCopy).not.toContain('payment.')
  })
})
