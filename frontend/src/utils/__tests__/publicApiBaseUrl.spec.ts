import { describe, expect, it } from 'vitest'
import { resolvePublicApiBaseUrl } from '@/utils/publicApiBaseUrl'

describe('resolvePublicApiBaseUrl', () => {
  it('preserves a configured public API URL', () => {
    expect(resolvePublicApiBaseUrl(' https://gateway.example/v1 ', 'https://current.example')).toBe(
      'https://gateway.example/v1'
    )
  })

  it('derives the fallback from the current site origin', () => {
    expect(resolvePublicApiBaseUrl('', 'https://current.example/')).toBe('https://current.example/v1')
  })

  it('uses a relative fallback during server-side rendering', () => {
    expect(resolvePublicApiBaseUrl(undefined, '')).toBe('/v1')
  })
})
