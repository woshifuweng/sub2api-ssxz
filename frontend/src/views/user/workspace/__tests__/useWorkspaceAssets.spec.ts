import { beforeEach, describe, expect, it, vi } from 'vitest'

const api = vi.hoisted(() => ({
  registerAsset: vi.fn()
}))

vi.mock('@/api/chatWorkspace', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/chatWorkspace')>()
  return {
    ...actual,
    registerAsset: api.registerAsset
  }
})

import { useWorkspaceAssets } from '../useWorkspaceAssets'

describe('useWorkspaceAssets', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.registerAsset.mockResolvedValue({
      id: 9,
      asset_kind: 'image',
      source_type: 'user_upload',
      asset_role: 'attachment',
      storage_provider: 'pending',
      storage_key: '',
      url: 'data:image/png;base64,abc',
      original_name: 'sample.png',
      content_type: 'image/png',
      byte_size: 10,
      status: 'registered',
      created_at: 'a',
      updated_at: 'b'
    })
  })

  it('accepts image files and rejects non-images', async () => {
    const assets = useWorkspaceAssets()
    const image = new File(['abc'], 'sample.png', { type: 'image/png' })
    const text = new File(['abc'], 'notes.txt', { type: 'text/plain' })

    await assets.addFiles([image, text])

    expect(assets.previews.value).toHaveLength(1)
    expect(assets.rejectedFiles.value[0].name).toBe('notes.txt')
  })

  it('registers pending image assets for a conversation', async () => {
    const assets = useWorkspaceAssets()
    const image = new File(['abc'], 'sample.png', { type: 'image/png' })

    await assets.addFiles([image])
    const registered = await assets.registerPendingAssets(10)

    expect(api.registerAsset).toHaveBeenCalledWith(expect.objectContaining({
      conversation_id: 10,
      asset_kind: 'image',
      source_type: 'user_upload',
      asset_role: 'attachment',
      original_name: 'sample.png',
      content_type: 'image/png'
    }))
    expect(registered[0].asset?.id).toBe(9)
  })

  it('removes previews and clears state', async () => {
    const assets = useWorkspaceAssets()
    const image = new File(['abc'], 'sample.png', { type: 'image/png' })

    await assets.addFiles([image])
    const id = assets.previews.value[0].id
    assets.removePreview(id)

    expect(assets.previews.value).toHaveLength(0)
  })
})
