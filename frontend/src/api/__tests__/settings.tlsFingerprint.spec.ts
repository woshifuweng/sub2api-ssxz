import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, put, del } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  del: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { get, post, put, delete: del },
}))

import {
  createTLSFingerprintProfile,
  deleteTLSFingerprintProfile,
  getTLSFingerprintSettings,
  updateTLSFingerprintProfile,
  updateTLSFingerprintSettings,
} from '@/api/admin/settings'

const profile = {
  profile_id: 'chrome',
  name: 'Chrome',
  enabled: true,
  enable_grease: true,
  cipher_suites: [],
  curves: [],
  point_formats: [],
}

describe('admin TLS fingerprint API', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    put.mockReset()
    del.mockReset()
  })

  it('uses the backend tls-fingerprint-profiles contract for list and mutations', async () => {
    get.mockResolvedValueOnce({ data: { enabled: true, items: [] } })
    put.mockResolvedValueOnce({ data: { enabled: true } })
    post.mockResolvedValueOnce({ data: profile })
    put.mockResolvedValueOnce({ data: profile })
    del.mockResolvedValueOnce({ data: undefined })

    await getTLSFingerprintSettings()
    await updateTLSFingerprintSettings({ enabled: true })
    await createTLSFingerprintProfile(profile)
    await updateTLSFingerprintProfile('chrome', { ...profile, name: 'Chrome 2' })
    await deleteTLSFingerprintProfile('chrome')

    expect(get).toHaveBeenCalledWith('/admin/tls-fingerprint-profiles')
    expect(put).toHaveBeenNthCalledWith(1, '/admin/tls-fingerprint-profiles', { enabled: true })
    expect(post).toHaveBeenCalledWith('/admin/tls-fingerprint-profiles', profile)
    expect(put).toHaveBeenNthCalledWith(
      2,
      '/admin/tls-fingerprint-profiles/chrome',
      { ...profile, name: 'Chrome 2' },
    )
    expect(del).toHaveBeenCalledWith('/admin/tls-fingerprint-profiles/chrome')
  })
})
