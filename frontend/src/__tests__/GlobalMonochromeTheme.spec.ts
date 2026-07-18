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
    expect(styles).toContain('--ssxz-primary: #4f6882')
    expect(styles).toContain('--ssxz-primary: #9db6ce')
    expect(styles).toContain('--ssxz-accent: #617991')
    expect(styles).toContain('--ssxz-accent: #a8bfd5')
    expect(styles).toContain('--ssxz-success: #22c55e')
    expect(styles).toContain('--ssxz-warning: #f59e0b')
    expect(styles).toContain('--ssxz-error: #ef4444')
    expect(styles).toContain('--ssxz-bg: #0b0b0d')
    expect(styles).toContain('--ssxz-surface: transparent')
    expect(styles).toContain('--ssxz-surface-raised: #0b0b0d')
    expect(styles).toContain('--ssxz-surface-muted: #111115')
    expect(styles).toContain('--ssxz-border: #2a2a30')
    expect(styles).toContain('--ssxz-border-strong: #33333a')
    expect(styles).toContain('--ssxz-text-secondary: #70707a')
    expect(styles).toContain('--ssxz-text-subtle: #55555d')
    expect(styles).toContain('--ssxz-action: #f4f4f5')
    expect(styles).toContain('--ssxz-action-text: #0b0b0d')
    expect(styles).toContain('--ssxz-placeholder-border: #e4e4e7')
    expect(styles).toContain('--ssxz-sidebar-width: 208px')
    expect(styles).toContain('--ssxz-sidebar-collapsed-width: 60px')
    expect(styles).toContain('--ssxz-content-max: 1540px')
    expect(styles).toContain('"PingFang SC"')
    expect(styles).toContain('"Noto Sans SC"')
    expect(styles).toContain('font-variant-numeric: tabular-nums')
    expect(styles).toMatch(/body::before\s*\{[\s\S]*opacity:\s*0\.026;/)
    expect(styles).toMatch(/\.sidebar\s*\{[\s\S]*transition:\s*width 200ms ease/)
    expect(styles).toContain('--ssxz-shadow-button:')
    expect(styles).toContain('--ssxz-shadow-card:')
    expect(styles).toContain('background: var(--ssxz-surface-muted);')
    expect(styles).toContain('background: rgb(0 0 0 / 0.72)')
    expect(styles).not.toMatch(/#(?:0f0f0e|141412|1a1a18|22221f|2a2a27|353532|4a4945)/)

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

  it('keeps the account shell scrim neutral instead of blue tinted', () => {
    const shell = readSource('src/components/user/AppSectionShell.vue')

    expect(shell).toContain('background: rgb(0 0 0 / 0.58)')
    expect(shell).toContain('box-shadow: 18px 0 50px rgb(0 0 0 / 0.35)')
    expect(shell).not.toContain('rgb(2 6 23')
  })
})
