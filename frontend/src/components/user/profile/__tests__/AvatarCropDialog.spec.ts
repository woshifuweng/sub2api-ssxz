import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AvatarCropDialog from '@/components/user/profile/AvatarCropDialog.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

const originalImage = globalThis.Image
const originalCreateObjectURL = URL.createObjectURL
const originalRevokeObjectURL = URL.revokeObjectURL
const originalCreateElement = document.createElement.bind(document)

function installImageMock(width = 1200, height = 800) {
  class MockImage {
    naturalWidth = width
    naturalHeight = height
    onload: (() => void) | null = null
    onerror: (() => void) | null = null

    set src(_value: string) {
      this.onload?.()
    }
  }

  globalThis.Image = MockImage as unknown as typeof Image
}

function createCanvasMock() {
  const drawImage = vi.fn()
  const canvas = {
    width: 0,
    height: 0,
    getContext: () => ({ drawImage }),
    toDataURL: () => 'data:image/webp;base64,YXZhdGFy'
  } as unknown as HTMLCanvasElement
  vi.spyOn(document, 'createElement').mockImplementation(((tagName: string, options?: ElementCreationOptions) => {
    if (tagName === 'canvas') return canvas
    return originalCreateElement(tagName, options)
  }) as typeof document.createElement)
  return drawImage
}

function mountDialog() {
  const wrapper = mount(AvatarCropDialog, {
    props: { show: true },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show', 'title'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>'
        },
        Icon: true
      }
    }
  })
  const stage = wrapper.get('[data-testid="avatar-crop-stage"]')
  vi.spyOn(stage.element, 'getBoundingClientRect').mockReturnValue({
    x: 0,
    y: 0,
    top: 0,
    left: 0,
    right: 320,
    bottom: 320,
    width: 320,
    height: 320,
    toJSON: () => ({})
  })
  Object.defineProperty(stage.element, 'setPointerCapture', { value: vi.fn() })
  Object.defineProperty(stage.element, 'releasePointerCapture', { value: vi.fn() })
  return { wrapper, stage }
}

async function chooseFile(wrapper: ReturnType<typeof mountDialog>['wrapper']) {
  const fileInput = wrapper.get('#avatar-file')
  Object.defineProperty(fileInput.element, 'files', {
    value: [new File([new Uint8Array([1])], 'avatar.png', { type: 'image/png' })],
    configurable: true
  })
  await fileInput.trigger('change')
  await flushPromises()
}

describe('AvatarCropDialog', () => {
  beforeEach(() => {
    installImageMock()
    Object.defineProperty(URL, 'createObjectURL', {
      configurable: true,
      writable: true,
      value: vi.fn(() => 'blob:avatar')
    })
    Object.defineProperty(URL, 'revokeObjectURL', {
      configurable: true,
      writable: true,
      value: vi.fn()
    })
  })

  afterEach(() => {
    globalThis.Image = originalImage
    if (originalCreateObjectURL) URL.createObjectURL = originalCreateObjectURL
    else delete (URL as URL & { createObjectURL?: typeof URL.createObjectURL }).createObjectURL
    if (originalRevokeObjectURL) URL.revokeObjectURL = originalRevokeObjectURL
    else delete (URL as URL & { revokeObjectURL?: typeof URL.revokeObjectURL }).revokeObjectURL
    vi.restoreAllMocks()
  })

  it.each(['mouse', 'touch'] as const)('uses the same square crop for %s dragging and zoomed output', async (pointerType) => {
    const { wrapper, stage } = mountDialog()
    const drawImage = createCanvasMock()
    await chooseFile(wrapper)

    await stage.trigger('pointerdown', {
      pointerId: 1,
      pointerType,
      button: 0,
      isPrimary: true,
      clientX: 100,
      clientY: 100
    })
    await stage.trigger('pointermove', {
      pointerId: 1,
      pointerType,
      clientX: 140,
      clientY: 120
    })
    await stage.trigger('pointerup', { pointerId: 1, pointerType })
    await wrapper.get('#avatar-zoom').setValue('2')
    await wrapper.get('button.btn-primary').trigger('click')

    expect(drawImage).toHaveBeenCalledWith(
      expect.anything(),
      expect.any(Number),
      expect.any(Number),
      expect.any(Number),
      expect.any(Number),
      0,
      0,
      256,
      256
    )
    const [, sourceX, sourceY, sourceWidth, sourceHeight] = drawImage.mock.calls[0]
    expect(sourceWidth).toBe(sourceHeight)
    expect(sourceWidth).toBe(400)
    expect(sourceX).toBeLessThan(400)
    expect(sourceY).toBeGreaterThan(0)
    expect(wrapper.emitted('save')).toHaveLength(1)
  })

  it('releases old preview URLs and ignores stale image loads', async () => {
    const { wrapper } = mountDialog()
    const fileInput = wrapper.get('#avatar-file')
    const images: DeferredImage[] = []

    class DeferredImage {
      naturalWidth: number
      naturalHeight: number
      onload: (() => void) | null = null
      onerror: (() => void) | null = null

      constructor() {
        this.naturalWidth = images.length === 0 ? 400 : 800
        this.naturalHeight = this.naturalWidth
        images.push(this)
      }

      set src(_value: string) {
      }
    }

    globalThis.Image = DeferredImage as unknown as typeof Image
    const drawImage = createCanvasMock()
    const files = [
      new File([new Uint8Array([1])], 'first.png', { type: 'image/png' }),
      new File([new Uint8Array([2])], 'second.png', { type: 'image/png' })
    ]
    Object.defineProperty(fileInput.element, 'files', { value: [files[0]], configurable: true })
    await fileInput.trigger('change')
    Object.defineProperty(fileInput.element, 'files', { value: [files[1]], configurable: true })
    await fileInput.trigger('change')

    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:avatar')

    images[0].onload?.()
    await wrapper.get('button.btn-primary').trigger('click')
    expect(wrapper.emitted('save')).toBeUndefined()

    images[1].onload?.()
    await wrapper.get('button.btn-primary').trigger('click')
    expect(drawImage.mock.calls[0][3]).toBe(800)
  })
})
