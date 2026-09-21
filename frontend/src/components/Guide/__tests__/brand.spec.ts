import { describe, expect, it } from 'vitest'
import { normalizeTourCopy, getAdminSteps, getUserSteps } from '../steps'

describe('SSXZ upstream guide overlay', () => {
  it('keeps upstream guidance with SSXZ branding and theme colors', () => {
    expect(normalizeTourCopy('👋 Welcome to Sub2API')).toBe('Welcome to SSXZ AI')
    expect(normalizeTourCopy('<p style="color: #10b981">Sub2API</p>'))
      .toBe('<p style="color: var(--ssxz-primary)">SSXZ AI</p>')
  })
  it('retains admin and user onboarding steps', () => {
    const translate = (key: string) => key
    expect(getAdminSteps(translate).length).toBeGreaterThan(10)
    expect(getUserSteps(translate).length).toBeGreaterThan(3)
  })
})
