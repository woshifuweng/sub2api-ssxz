import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useWorkspaceAssets } from '../useWorkspaceAssets'

const objectUrls = vi.hoisted(() => ({
  create: vi.fn((file: File) => `blob:workspace-preview-${file.name}`),
  revoke: vi.fn()
}))

describe('useWorkspaceAssets', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.stubGlobal('URL', {
      createObjectURL: objectUrls.create,
      revokeObjectURL: objectUrls.revoke
    })
  })

  it('accepts image files as local previews and rejects non-images', async () => {
    const assets = useWorkspaceAssets()
    const image = new File(['abc'], 'sample.png', { type: 'image/png' })
    const text = new File(['abc'], 'notes.txt', { type: 'text/plain' })

    await assets.addFiles([image, text])

    expect(assets.previews.value).toHaveLength(1)
    expect(assets.previews.value[0].url).toBe('blob:workspace-preview-sample.png')
    expect(assets.previews.value[0].url).not.toContain('data:image')
    expect(assets.rejectedFiles.value[0].name).toBe('notes.txt')
  })

  it('returns local attachments without asset records or base64 payloads', async () => {
    const assets = useWorkspaceAssets()
    const image = new File(['abc'], 'sample.png', { type: 'image/png' })

    await assets.addFiles([image])
    const attachments = assets.getLocalAttachments()

    expect(attachments).toEqual([
      expect.objectContaining({
        name: 'sample.png',
        type: 'image',
        url: 'blob:workspace-preview-sample.png'
      })
    ])
    expect(attachments[0].asset).toBeUndefined()
    expect(JSON.stringify(attachments)).not.toContain('data:image')
  })

  it('revokes preview object URLs when removing or clearing previews', async () => {
    const assets = useWorkspaceAssets()
    const first = new File(['abc'], 'first.png', { type: 'image/png' })
    const second = new File(['abc'], 'second.png', { type: 'image/png' })

    await assets.addFiles([first, second])
    assets.removePreview(assets.previews.value[0].id)
    expect(objectUrls.revoke).toHaveBeenCalledWith('blob:workspace-preview-first.png')

    assets.clearPreviews()
    expect(objectUrls.revoke).toHaveBeenCalledWith('blob:workspace-preview-second.png')
    expect(assets.previews.value).toHaveLength(0)
  })
})
