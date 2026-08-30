import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

import { redirectLegacyRoute } from '@/router/legacyRedirect'

const routerSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../index.ts'),
  'utf8',
)

describe('usage workspace route compatibility', () => {
  it('preserves query and hash when redirecting the old usage route', () => {
    const redirect = redirectLegacyRoute('/app/usage')

    expect(redirect({
      query: { model: 'gpt-5.4', page: '2' },
      hash: '#records',
    } as never)).toEqual({
      path: '/app/usage',
      query: { model: 'gpt-5.4', page: '2' },
      hash: '#records',
    })
  })

  it.each([
    ['/dashboard', '/app/dashboard'],
    ['/chat', '/app/chat'],
    ['/keys', '/app/keys'],
    ['/batch-image', '/app/image'],
    ['/usage', '/app/usage'],
    ['/purchase', '/app/purchase'],
    ['/orders', '/app/orders'],
    ['/redeem', '/app/redeem'],
    ['/affiliate', '/app/affiliate'],
    ['/available-channels', '/app/available-channels'],
    ['/profile', '/app/profile'],
  ])('maps legacy %s to native workspace route %s', (legacyPath, workspacePath) => {
    expect(routerSource).toContain(`path: '${workspacePath}'`)
    expect(routerSource).toContain(
      `path: '${legacyPath}',\n    redirect: redirectLegacyRoute('${workspacePath}')`,
    )
  })

  it('registers the restored chat, docs and channel status routes', () => {
    expect(routerSource).toContain("path: '/app/chat'")
    expect(routerSource).toContain("path: '/app/docs'")
    expect(routerSource).toContain("path: '/app/channel-status'")
  })
})
