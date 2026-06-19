import { describe, expect, it } from 'vitest'
import router from '@/router'
import { resolveDocumentTitle } from '@/router/title'

describe('payment route document titles', () => {
  it('can use a route-level site name override without changing the configured site name', () => {
    expect(resolveDocumentTitle('Stripe Payment', 'SSXZ API', undefined, 'SSXZ AI')).toBe('Stripe Payment - SSXZ AI')
  })

  it('uses a neutral payment result title instead of a success-only title', () => {
    const route = router.resolve('/payment/result?status=failed')

    expect(route.meta.title).toBe('Payment Result')
    expect(route.meta.titleKey).toBe('payment.result.title')
    expect(resolveDocumentTitle(route.meta.title, 'SSXZ API', undefined, route.meta.titleSiteName)).toBe(
      'Payment Result - SSXZ AI'
    )
  })
})
