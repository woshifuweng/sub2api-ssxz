import { describe, expect, it } from 'vitest'
import { resolveDocumentTitle } from '@/router/title'

describe('payment route document titles', () => {
  it('can use a route-level site name override without changing the configured site name', () => {
    expect(resolveDocumentTitle('Stripe Payment', 'SSXZ API', undefined, 'SSXZ AI')).toBe('Stripe Payment - SSXZ AI')
  })
})
