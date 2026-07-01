import { describe, expect, it } from 'vitest'
import { DEFAULT_AUTH_REDIRECT, resolveAuthRedirect, resolveRouteAuthRedirect } from '../authRedirect'

describe('auth redirect resolution', () => {
  it('defaults regular users to the operating dashboard', () => {
    expect(DEFAULT_AUTH_REDIRECT).toBe('/app/dashboard')
    expect(resolveAuthRedirect(undefined)).toBe('/app/dashboard')
    expect(resolveRouteAuthRedirect({})).toBe('/app/dashboard')
  })

  it('keeps the chat helper route as an allowed return target', () => {
    expect(resolveAuthRedirect('/app/chat')).toBe('/app/chat')
  })

  it('maps legacy user return targets into the workbench shell', () => {
    expect(resolveAuthRedirect('/keys')).toBe('/app/keys')
    expect(resolveAuthRedirect('/usage')).toBe('/app/usage')
    expect(resolveAuthRedirect('/purchase')).toBe('/app/purchase')
    expect(resolveAuthRedirect('/orders')).toBe('/app/orders')
    expect(resolveAuthRedirect('/subscriptions')).toBe('/app/purchase')
    expect(resolveAuthRedirect('/redeem')).toBe('/app/redeem')
    expect(resolveAuthRedirect('/available-channels')).toBe('/app/available-channels')
    expect(resolveAuthRedirect('/monitor')).toBe('/app/channel-status')
    expect(resolveAuthRedirect('/profile')).toBe('/app/profile')
  })

  it('keeps API Key as an allowed third-party client workbench return target', () => {
    expect(resolveAuthRedirect('/app/keys')).toBe('/app/keys')
  })

  it('maps generic workspace entrypoints to the operating dashboard', () => {
    expect(resolveAuthRedirect('/dashboard')).toBe('/app/dashboard')
    expect(resolveAuthRedirect('/app')).toBe('/app/dashboard')
    expect(resolveAuthRedirect('/app/dashboard')).toBe('/app/dashboard')
    expect(resolveAuthRedirect('/home')).toBe('/app/dashboard')
    expect(resolveAuthRedirect('/apps')).toBe('/app/dashboard')
  })

  it('keeps image-specific legacy entrypoints on the image beta page', () => {
    expect(resolveAuthRedirect('/app/image')).toBe('/app/image')
    expect(resolveAuthRedirect('/image-studio')).toBe('/app/image')
  })

  it('keeps the legacy Sora route available as a direct compatibility target', () => {
    expect(resolveAuthRedirect('/sora')).toBe('/sora')
  })

  it('does not allow hidden or external return targets', () => {
    expect(resolveAuthRedirect('https://example.com/app')).toBe('/app/dashboard')
    expect(resolveAuthRedirect('//example.com/app')).toBe('/app/dashboard')
  })
})
