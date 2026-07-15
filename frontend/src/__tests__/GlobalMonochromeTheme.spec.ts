import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const readSource = (path: string) => readFileSync(resolve(process.cwd(), path), 'utf8')

describe('global monochrome theme', () => {
  it('maps the global light and dark themes to the approved F0 gray scales', () => {
    const styles = readSource('src/style.css')
    const tailwind = readSource('tailwind.config.js')

    expect(styles).toContain('color-scheme: light')
    expect(styles).toContain('.dark {')
    expect(styles).toContain('color-scheme: dark')
    expect(styles).toContain('--ssxz-primary: #181a1e')
    expect(styles).toContain('--ssxz-primary: #f3f4f6')
    expect(styles).toContain('--ssxz-accent: #556273')
    expect(styles).toContain('--ssxz-accent: #aab4c2')
    expect(styles).toContain('--ssxz-success: #22c55e')
    expect(styles).toContain('--ssxz-warning: #f59e0b')
    expect(styles).toContain('--ssxz-error: #ef4444')

    for (const legacyBrandColor of ['#6366f1', '#38bdf8', '99 102 241', '56 189 248']) {
      expect(styles).not.toContain(legacyBrandColor)
      expect(tailwind).not.toContain(legacyBrandColor)
    }
  })

  it('removes hard-coded purple and blue branding from shared customer actions', () => {
    const customerActionSources = [
      'src/components/common/AnnouncementBell.vue',
      'src/components/common/SubscriptionProgressMini.vue',
      'src/components/common/VersionBadge.vue',
      'src/views/user/KeysView.vue',
      'src/views/user/PaymentResultView.vue',
      'src/views/user/RedeemView.vue'
    ].map(readSource)

    for (const source of customerActionSources) {
      expect(source).not.toMatch(/(?:blue|indigo|purple|violet|sky|cyan)-\d/)
    }
  })
})
