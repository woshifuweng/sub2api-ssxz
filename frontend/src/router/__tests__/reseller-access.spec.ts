import { describe, expect, it } from 'vitest'
import { hasRequiredResellerRole, resellerAccessFallback } from '@/router/resellerAccess'

describe('reseller route access', () => {
  it('allows agents and managers into agent pages', () => {
    expect(hasRequiredResellerRole('agent', 'agent')).toBe(true)
    expect(hasRequiredResellerRole('agent', 'agent_manager')).toBe(true)
  })

  it('only allows managers into manager pages', () => {
    expect(hasRequiredResellerRole('agent_manager', 'agent')).toBe(false)
    expect(hasRequiredResellerRole('agent_manager', 'agent_manager')).toBe(true)
  })

  it('returns users without a reseller role to the native dashboard', () => {
    expect(resellerAccessFallback(null)).toBe('/app/dashboard')
    expect(resellerAccessFallback('agent')).toBe('/app/reseller')
  })
})
