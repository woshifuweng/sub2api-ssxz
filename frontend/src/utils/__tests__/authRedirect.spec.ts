import { describe, expect, it } from 'vitest'
import { DEFAULT_AUTH_REDIRECT, resolveAuthRedirect, resolveRouteAuthRedirect } from '../authRedirect'

describe('auth redirect resolution', () => {
  it('defaults regular users to the lightweight image workspace', () => {
    expect(DEFAULT_AUTH_REDIRECT).toBe('/sora')
    expect(resolveAuthRedirect(undefined)).toBe('/sora')
    expect(resolveRouteAuthRedirect({})).toBe('/sora')
  })

  it('keeps the chat helper route as an allowed return target', () => {
    expect(resolveAuthRedirect('/app/chat')).toBe('/app/chat')
  })

  it('keeps API Key as an allowed third-party client return target', () => {
    expect(resolveAuthRedirect('/keys')).toBe('/keys')
  })

  it('maps heavy or legacy workspace entrypoints back to Sora', () => {
    expect(resolveAuthRedirect('/dashboard')).toBe('/sora')
    expect(resolveAuthRedirect('/app')).toBe('/sora')
    expect(resolveAuthRedirect('/app/image')).toBe('/sora')
    expect(resolveAuthRedirect('/image-studio')).toBe('/sora')
  })

  it('does not allow hidden or external return targets', () => {
    expect(resolveAuthRedirect('https://example.com/app')).toBe('/sora')
    expect(resolveAuthRedirect('//example.com/app')).toBe('/sora')
  })
})
