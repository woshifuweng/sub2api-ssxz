import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { redirectLegacyRoute } from '@/router/legacyRedirect'

const source = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../index.ts'), 'utf8')

describe('remaining product routes', () => {
  it.each([
    '/app/reseller',
    '/app/reseller/withdrawals',
    '/app/reseller/recruits',
    '/app/reseller/commission',
    '/app/reseller/invite',
    '/app/reseller/manager',
    '/admin/reseller/agents',
    '/admin/reseller/withdrawals',
    '/app/docs',
    '/docs',
    '/app/channel-status',
  ])('registers %s', (path) => {
    expect(source).toContain(`path: '${path}'`)
  })

  it('keeps /monitor on the admin monitor and the app route on the user view', () => {
    expect(source).toMatch(/path: '\/monitor',[\s\S]*?AdminChannelMonitorLegacy[\s\S]*?ChannelMonitorView\.vue[\s\S]*?requiresAdmin: true/)
    expect(source).toMatch(/path: '\/app\/channel-status',[\s\S]*?ChannelStatusView\.vue[\s\S]*?requiresAdmin: false/)
  })

  it('preserves query and hash for native redirect groups', () => {
    expect(redirectLegacyRoute('/admin/reseller/agents')({
      query: { status: 'pending' },
      hash: '#queue',
    } as never)).toEqual({
      path: '/admin/reseller/agents',
      query: { status: 'pending' },
      hash: '#queue',
    })

    expect(source).toContain("redirect: redirectLegacyRoute('/admin/reseller/agents')")
  })

  it('registers the retained chat workspace', () => {
    expect(source).toContain("path: '/app/chat'")
  })
})
