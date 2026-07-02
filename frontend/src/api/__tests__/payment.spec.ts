import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { CreateOrderResult, PublicPaymentOrder } from '@/types/payment'

const get = vi.fn()
const post = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post
  }
}))

describe('payment api public order lookup', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('uses narrow public order responses for anonymous lookup endpoints', async () => {
    const publicOrder: PublicPaymentOrder = {
      id: 101,
      amount: 10,
      pay_amount: 10,
      fee_rate: 0,
      payment_type: 'alipay',
      out_trade_no: 'sub2_public_lookup',
      status: 'PENDING',
      order_type: 'balance',
      created_at: '2026-06-26T00:00:00Z',
      expires_at: '2026-06-26T00:15:00Z'
    }
    post.mockResolvedValue({ data: publicOrder })

    const { paymentAPI } = await import('../payment')
    const byOutTradeNo = await paymentAPI.verifyOrderPublic(publicOrder.out_trade_no)
    const byResumeToken = await paymentAPI.resolveOrderPublicByResumeToken('signed-resume-token')

    expect(post).toHaveBeenNthCalledWith(1, '/payment/public/orders/verify', {
      out_trade_no: publicOrder.out_trade_no
    })
    expect(post).toHaveBeenNthCalledWith(2, '/payment/public/orders/resolve', {
      resume_token: 'signed-resume-token'
    })
    expect(byOutTradeNo.data).toEqual(publicOrder)
    expect(byResumeToken.data).toEqual(publicOrder)
    expect('user_id' in byOutTradeNo.data).toBe(false)
    expect('user_id' in byResumeToken.data).toBe(false)
    expect('refund_requested_by' in byOutTradeNo.data).toBe(false)
    expect('refund_request_reason' in byOutTradeNo.data).toBe(false)
    expect('plan_id' in byOutTradeNo.data).toBe(false)
  })

  it('sends Idempotency-Key when creating a payment order with a key', async () => {
    const result: CreateOrderResult = {
      order_id: 7,
      amount: 10,
      pay_amount: 10,
      fee_rate: 0,
      status: 'PENDING',
      payment_type: 'alipay',
      expires_at: '2026-06-26T00:15:00Z'
    }
    post.mockResolvedValue({ data: result })

    const { paymentAPI } = await import('../payment')
    await paymentAPI.createOrder(
      { amount: 10, payment_type: 'alipay', order_type: 'balance' },
      { idempotencyKey: 'payment-order-once' }
    )

    expect(post).toHaveBeenCalledWith(
      '/payment/orders',
      { amount: 10, payment_type: 'alipay', order_type: 'balance' },
      { headers: { 'Idempotency-Key': 'payment-order-once' } }
    )
  })
})
