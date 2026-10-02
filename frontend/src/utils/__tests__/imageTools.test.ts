import { describe, expect, it } from 'vitest'
import { reactive } from 'vue'
import { imageToolInit, imageToolKeyName, isImageToolMessage, isUsableImageToolKey, supportsStationImages } from '../imageTools'

const key = {
  name: imageToolKeyName(10), user_id: 7, status: 'active' as const, group_id: 10,
  group_ids: [10], allowed_models: ['gpt-image-2'], expires_at: null,
  quota: 0, quota_used: 0, key: 'test-image-key',
}

describe('station image-tool boundary', () => {
  it('makes reactive input cloneable for the real browser message protocol', () => {
    const profile = reactive({ name: '图片组', baseUrl: 'https://api.ssxzapi.com/v1', apiKey: 'test-image-key', model: 'gpt-image-2' as const })
    const message = imageToolInit(7, 'session', profile)
    expect(() => structuredClone(profile)).toThrow()
    expect(structuredClone(message)).toEqual(message)
  })

  it('reuses only the current owner, group and model-scoped native key', () => {
    expect(isUsableImageToolKey(key, 7, 10)).toBe(true)
    expect(isUsableImageToolKey(key, 8, 10)).toBe(false)
    expect(isUsableImageToolKey(key, 7, 11)).toBe(false)
    expect(isUsableImageToolKey({ ...key, allowed_models: [] }, 7, 10)).toBe(false)
    expect(isUsableImageToolKey({ ...key, allowed_models: ['gpt-image-2', 'gpt-text'] }, 7, 10)).toBe(false)
    expect(isUsableImageToolKey({ ...key, group_ids: [10, 20] }, 7, 10)).toBe(false)
  })

  it('rejects revoked, expired, exhausted and masked keys', () => {
    expect(isUsableImageToolKey({ ...key, status: 'disabled' }, 7, 10)).toBe(false)
    expect(isUsableImageToolKey({ ...key, expires_at: '2020-01-01T00:00:00Z' }, 7, 10)).toBe(false)
    expect(isUsableImageToolKey({ ...key, expires_at: 'invalid' }, 7, 10)).toBe(false)
    expect(isUsableImageToolKey({ ...key, quota: 1, quota_used: 1 }, 7, 10)).toBe(false)
    expect(isUsableImageToolKey({ ...key, key: 'sk-***' }, 7, 10)).toBe(false)
  })

  it('offers only image-enabled OpenAI groups returned by the native user API', () => {
    const group = { status: 'active' as const, platform: 'openai' as const, allow_image_generation: true }
    expect(supportsStationImages(group)).toBe(true)
    expect(supportsStationImages({ ...group, status: 'inactive' })).toBe(false)
    expect(supportsStationImages({ ...group, platform: 'anthropic' })).toBe(false)
    expect(supportsStationImages({ ...group, allow_image_generation: false })).toBe(false)
  })

  it('checks both exact iframe source and origin before delivering credentials', () => {
    const event = { source: window, origin: 'https://image.ssxzapi.com', data: { type: 'ssxz:image-tool:ready', version: 1 } }
    expect(isImageToolMessage(event, window, event.origin, 'ready')).toBe(true)
    expect(isImageToolMessage(event, null, event.origin, 'ready')).toBe(false)
    expect(isImageToolMessage({ ...event, source: null }, window, event.origin, 'ready')).toBe(false)
    expect(isImageToolMessage({ ...event, origin: 'https://attacker.example' }, window, event.origin, 'ready')).toBe(false)
    expect(isImageToolMessage({ ...event, data: null }, window, event.origin, 'ready')).toBe(false)
    expect(isImageToolMessage({ ...event, data: { ...event.data, version: 2 } }, window, event.origin, 'ready')).toBe(false)
  })

  it('sends only a dedicated image profile, never site login credentials', () => {
    const result = imageToolInit(7, 'session', { name: '图片分组', baseUrl: 'https://api.ssxzapi.com/v1', apiKey: 'test-image-key', model: 'gpt-image-2' })
    expect(Object.keys(result).sort()).toEqual(['accountId', 'profile', 'sessionId', 'type', 'version'])
    expect(result.accountId).toBe('7')
    expect(result.profile.model).toBe('gpt-image-2')
  })
})
