import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    post,
  },
}))

import { resellerAPI } from '@/api/reseller'

describe('reseller api', () => {
  beforeEach(() => {
    post.mockReset()
    post.mockResolvedValue({ data: {} })
  })

  it('uses the plural withdrawal route and forwards the idempotency key', async () => {
    await resellerAPI.requestBalanceConversion(5, { idempotencyKey: 'withdraw-1' })

    expect(post).toHaveBeenCalledWith(
      '/user/reseller/withdrawals',
      { amount: 5 },
      { headers: { 'Idempotency-Key': 'withdraw-1' } }
    )
  })
})
