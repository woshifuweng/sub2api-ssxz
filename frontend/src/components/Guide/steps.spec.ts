import { describe, expect, it } from 'vitest'
import { getAdminSteps, getUserSteps, normalizeTourCopy } from './steps'

const translate = (key: string) => {
  if (key.endsWith('.title')) return '👋 Tour title 🎯'
  if (key.endsWith('.description')) {
    return '<p style="color: #10b981; background: #f0fdf4; border-left: 3px solid #3b82f6">📦 Tour copy</p>'
  }
  return 'Start 🚀'
}

describe('onboarding tour presentation', () => {
  it('removes decorative emoji and maps inline brand colors to F0 tokens', () => {
    const result = normalizeTourCopy('👋 Ready 🎯 <span style="color: #10b981; background: #eff6ff">Go</span>')

    expect(result).not.toMatch(/\p{Extended_Pictographic}/u)
    expect(result).not.toContain('#10b981')
    expect(result).not.toContain('#eff6ff')
    expect(result).toContain('var(--ssxz-primary)')
    expect(result).toContain('var(--ssxz-primary-soft)')
  })

  it('keeps the established admin and customer step counts', () => {
    expect(getAdminSteps(translate)).toHaveLength(21)
    expect(getUserSteps(translate)).toHaveLength(6)
  })

  it('normalizes every generated tour copy field without changing its step targets', () => {
    const adminSteps = getAdminSteps(translate)
    const userSteps = getUserSteps(translate)
    const allSteps = [...adminSteps, ...userSteps]

    for (const step of allSteps) {
      const popover = step.popover
      expect(popover?.title).not.toMatch(/\p{Extended_Pictographic}/u)
      expect(popover?.description).not.toMatch(/\p{Extended_Pictographic}/u)
      expect(popover?.title).not.toContain('#10b981')
      expect(popover?.description).not.toContain('#3b82f6')
    }

    expect(adminSteps[1]?.element).toBe('#sidebar-group-manage')
    expect(userSteps[1]?.element).toBe('[data-tour="sidebar-my-keys"]')
  })
})
