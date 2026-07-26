import { describe, expect, it } from 'vitest'
import {
  decidePaymentLaunch,
  getVisibleMethods,
  type StripeVisibleMethod,
} from '../paymentFlow'
import type { CreateOrderResult, MethodLimit } from '@/types/payment'

function methodLimit(overrides: Partial<MethodLimit> = {}): MethodLimit {
  return {
    daily_limit: 0,
    daily_used: 0,
    daily_remaining: 0,
    single_min: 0,
    single_max: 0,
    fee_rate: 0,
    available: true,
    ...overrides,
  }
}

function createOrderResult(overrides: Partial<CreateOrderResult> = {}): CreateOrderResult {
  return {
    order_id: 1,
    amount: 10,
    pay_amount: 10,
    fee_rate: 0,
    expires_at: '2099-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('getVisibleMethods', () => {
  it('normalizes provider aliases and keeps stripe as a top-level method', () => {
    const visible = getVisibleMethods({
      alipay_direct: methodLimit({ single_min: 5 }),
      wxpay: methodLimit({ single_max: 100 }),
      stripe: methodLimit({ fee_rate: 3 }),
      easypay: methodLimit({ fee_rate: 9 }),
    })

    expect(visible).toEqual({
      alipay: methodLimit({ single_min: 5 }),
      wxpay: methodLimit({ single_max: 100 }),
      stripe: methodLimit({ fee_rate: 3 }),
    })
  })
})

describe('decidePaymentLaunch', () => {
  it('routes dedicated Stripe button clicks to the full Payment Element without a preselected sub-method', () => {
    const decision = decidePaymentLaunch(createOrderResult({
      client_secret: 'cs_test',
    }), {
      visibleMethod: 'stripe',
      orderType: 'balance',
      isMobile: false,
      stripeRouteUrl: '/payment/stripe?client_secret=cs_test',
    })

    expect(decision.kind).toBe('stripe_route')
    expect(decision.stripeMethod).toBeUndefined()
    expect(decision.paymentState.payUrl).toBe('/payment/stripe?client_secret=cs_test')
  })

  it.each([
    ['alipay', 'stripe_popup', 'alipay'],
    ['wxpay', 'stripe_route', 'wechat_pay'],
  ] as const)('preselects Stripe sub-method for %s compatibility', (visibleMethod, kind, stripeMethod) => {
    const decision = decidePaymentLaunch(createOrderResult({
      client_secret: 'cs_test',
    }), {
      visibleMethod,
      orderType: 'balance',
      isMobile: false,
      stripePopupUrl: '/payment/stripe?method=alipay',
      stripeRouteUrl: '/payment/stripe?method=wechat_pay',
    })

    expect(decision.kind).toBe(kind)
    expect(decision.stripeMethod).toBe(stripeMethod as StripeVisibleMethod)
  })

  it('launches the Alipay app for a mobile precreate order', () => {
    const decision = decidePaymentLaunch(createOrderResult({
      qr_code: 'https://qr.alipay.com/dynamic-order-101',
      alipay_mobile_precreate_deep_link: true,
    }), {
      visibleMethod: 'alipay',
      orderType: 'balance',
      isMobile: true,
    })

    expect(decision.kind).toBe('alipay_deep_link')
    expect(decision.paymentState.qrCode).toBe('https://qr.alipay.com/dynamic-order-101')
    expect(decision.paymentState.alipayMobilePrecreateDeepLink).toBe(true)
  })

  it('keeps the desktop Alipay QR flow when a precreate marker is present', () => {
    const decision = decidePaymentLaunch(createOrderResult({
      qr_code: 'https://qr.alipay.com/dynamic-order-102',
      alipay_mobile_precreate_deep_link: true,
    }), {
      visibleMethod: 'alipay',
      orderType: 'balance',
      isMobile: false,
    })

    expect(decision.kind).toBe('qr_waiting')
  })
})
