import { describe, expect, it } from 'vitest'
import { DEFAULT_AUTH_REDIRECT, resolveAuthRedirect, resolveRouteAuthRedirect } from '../authRedirect'

describe('auth redirect resolution', () => {
  it('defaults regular users to the lightweight image workspace', () => {
    expect(DEFAULT_AUTH_REDIRECT).toBe('/app/image')
    expect(resolveAuthRedirect(undefined)).toBe('/app/image')
    expect(resolveRouteAuthRedirect({})).toBe('/app/image')
  })

  it('keeps the chat helper route as an allowed return target', () => {
    expect(resolveAuthRedirect('/app/chat')).toBe('/app/chat')
  })

  it('keeps API Key as an allowed third-party client return target', () => {
    expect(resolveAuthRedirect('/keys')).toBe('/keys')
  })

  it('maps heavy or legacy workspace entrypoints back to image generation', () => {
    expect(resolveAuthRedirect('/dashboard')).toBe('/app/image')
    expect(resolveAuthRedirect('/app')).toBe('/app/image')
    expect(resolveAuthRedirect('/app/image')).toBe('/app/image')
    expect(resolveAuthRedirect('/image-studio')).toBe('/app/image')
  })

  it('keeps the legacy Sora route available as a direct compatibility target', () => {
    expect(resolveAuthRedirect('/sora')).toBe('/sora')
  })

  it('does not allow hidden or external return targets', () => {
    expect(resolveAuthRedirect('https://example.com/app')).toBe('/app/image')
    expect(resolveAuthRedirect('//example.com/app')).toBe('/app/image')
  })
})
