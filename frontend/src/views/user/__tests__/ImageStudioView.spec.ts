import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { appStore, authStore, apiClient, soraApi } = vi.hoisted(() => ({
  appStore: {
    showError: vi.fn(),
    showInfo: vi.fn(),
    showSuccess: vi.fn(),
  },
  authStore: {
    user: {
      balance: 1,
    },
    refreshUser: vi.fn(),
  },
  apiClient: {
    post: vi.fn(),
  },
  soraApi: {
    listGenerations: vi.fn(),
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore,
}))

vi.mock('@/api/client', () => ({
  apiClient,
}))

vi.mock('@/components/user/AppSectionShell.vue', () => ({
  default: {
    name: 'AppSectionShell',
    props: ['title', 'subtitle', 'eyebrow', 'icon'],
    template: '<main><header>{{ title }} {{ subtitle }} {{ eyebrow }}</header><slot /></main>',
  },
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    props: ['name', 'size'],
    template: '<span class="icon-stub" />',
  },
}))

vi.mock('@/api/sora', () => ({
  default: soraApi,
}))

import ImageStudioView from '../ImageStudioView.vue'

function mountImageStudio() {
  return mount(ImageStudioView, {
    global: {
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>',
        },
      },
    },
  })
}

function stubFileReader(result: 'success' | 'error' = 'success') {
  class MockFileReader {
    result: string | ArrayBuffer | null = null
    error: DOMException | null = result === 'error'
      ? new DOMException('preview failed')
      : null
    onload: (() => void) | null = null
    onerror: (() => void) | null = null

    readAsDataURL(file: File) {
      if (result === 'error') {
        this.onerror?.()
        return
      }
      this.result = `data:${file.type};base64,ZmFrZS1wcmV2aWV3`
      this.onload?.()
    }
  }

  vi.stubGlobal('FileReader', MockFileReader)
}

describe('ImageStudioView workbench', () => {
  beforeEach(() => {
    authStore.user.balance = 1
    authStore.refreshUser.mockReset()
    appStore.showError.mockReset()
    appStore.showInfo.mockReset()
    appStore.showSuccess.mockReset()
    apiClient.post.mockReset()
    soraApi.listGenerations.mockReset()
    soraApi.listGenerations.mockResolvedValue({
      data: [],
      total: 0,
      page: 1,
    })
    vi.unstubAllGlobals()
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      callback(0)
      return 0
    })
  })

  it('restores the old-site style image workbench instead of the discarded console page', () => {
    const wrapper = mountImageStudio()
    const text = wrapper.text()

    expect(text).toContain('AI 作图')
    expect(text).toContain('图片生成工作台')
    expect(text).toContain('把想法整理成可交付的视觉作品')
    expect(text).toContain('用途、比例、风格和参考图是创作方向，不是固定人设')
    expect(text).toContain('先用对话整理想法')
    expect(text).toContain('作图是主流程，对话可以帮你把随口需求整理成更清楚的画面描述')
    expect(text).toContain('对话辅助保留')
    expect(text).toContain('先聊清楚，再去作图')
    expect(text).toContain('选择创作目标')
    expect(text).toContain('商品主图')
    expect(text).toContain('营销海报')
    expect(text).toContain('社媒封面')
    expect(text).toContain('详情页配图')
    expect(text).toContain('自定义画幅')
    expect(text).toContain('生成张数')
    expect(text).toContain('多张结果会在右侧以缩略图切换')
    expect(text).toContain('上传商品图或风格参考图')
    expect(text).toContain('你的作品将在这里呈现')
    expect(text).toContain('普通话说需求')
    expect(text).toContain('对话辅助润色')
    expect(text).toContain('最近作品')
    expect(text).not.toContain('商品图、海报和灵感图，一页完成')
    expect(text).not.toContain('Sora')
    const resultActionButtons = wrapper.findAll('.result-actions .secondary-button')
    expect(resultActionButtons[1].attributes('disabled')).toBeDefined()
    expect(resultActionButtons[2].attributes('disabled')).toBeDefined()

    wrapper.unmount()
  })

  it('loads recent image works from existing history', async () => {
    soraApi.listGenerations.mockResolvedValue({
      data: [
        {
          id: 11,
          user_id: 7,
          model: 'gpt-image-2',
          prompt: 'product poster',
          media_type: 'image',
          status: 'completed',
          storage_type: 'upstream',
          media_url: 'https://cdn.example.com/work.png',
          media_urls: ['https://cdn.example.com/work.png'],
          s3_object_keys: [],
          file_size_bytes: 0,
          error_message: '',
          created_at: '2026-06-18T00:00:00Z',
        },
      ],
      total: 1,
      page: 1,
    })

    const wrapper = mountImageStudio()
    await flushPromises()

    expect(soraApi.listGenerations).toHaveBeenCalledWith({
      status: 'completed',
      media_type: 'image',
      page: 1,
      page_size: 8,
    })
    expect(wrapper.text()).toContain('最近作品')
    expect(wrapper.text()).toContain('图片作品')
    expect(wrapper.text()).not.toContain('gpt-image-2')
    expect(wrapper.find('img[alt="work-11"]').attributes('src')).toBe('https://cdn.example.com/work.png')

    wrapper.unmount()
  })

  it('submits the restored workbench through the current image-studio API', async () => {
    apiClient.post.mockResolvedValue({
      data: {
        data: [
          {
            url: 'https://cdn.example.com/result.png',
          },
          {
            url: 'https://cdn.example.com/result-2.png',
          },
        ],
      },
    })

    const wrapper = mountImageStudio()
    const countButtons = wrapper.findAll('.count-chip')
    expect(countButtons).toHaveLength(3)
    await countButtons[1].trigger('click')
    await wrapper.find('input[placeholder*="无线耳机"]').setValue('玫瑰花束')
    await wrapper.find('textarea').setValue('高级感电商广告图，黑色包装纸，红玫瑰主体')
    await wrapper.find('input[placeholder*="晨光办公桌"]').setValue('柔和棚拍')
    await wrapper.find('button.generate-button').trigger('click')
    await flushPromises()

    expect(apiClient.post).toHaveBeenCalledTimes(1)
    expect(apiClient.post).toHaveBeenCalledWith(
      '/image-studio/generate',
      expect.any(FormData),
      { timeout: 120000 },
    )
    const form = apiClient.post.mock.calls[0][1] as FormData
    expect(form.get('template_id')).toBe('white')
    expect(form.get('product_name')).toBe('玫瑰花束')
    expect(form.get('size')).toBe('1024x1024')
    expect(form.get('count')).toBe('2')
    expect(String(form.get('selling_points'))).toContain('高级感电商广告图')
    expect(String(form.get('selling_points'))).toContain('创作用途：商品主图')
    expect(String(form.get('selling_points'))).toContain('商用安全')
    expect(wrapper.find('img[alt="result-1"]').attributes('src')).toBe('https://cdn.example.com/result.png')
    expect(wrapper.findAll('.thumbnail-button')).toHaveLength(2)
    expect(wrapper.findAll('.result-actions .secondary-button')[2].attributes('disabled')).toBeUndefined()
    expect(authStore.refreshUser).toHaveBeenCalledTimes(1)
    expect(appStore.showSuccess).toHaveBeenCalledWith('图片生成完成')

    wrapper.unmount()
  })

  it('renders an uploaded reference image preview and submits the original file', async () => {
    stubFileReader()
    apiClient.post.mockResolvedValue({
      data: {
        data: [
          {
            url: 'https://cdn.example.com/result.png',
          },
        ],
      },
    })

    const wrapper = mountImageStudio()
    const file = new File(['fake-image'], 'reference.png', { type: 'image/png' })
    const fileInput = wrapper.find('input[type="file"]')
    Object.defineProperty(fileInput.element, 'files', {
      value: [file],
      configurable: true,
    })

    await fileInput.trigger('change')
    await flushPromises()

    const preview = wrapper.find('img[alt="参考素材预览"]')
    expect(preview.exists()).toBe(true)
    expect(preview.attributes('src')).toBe('data:image/png;base64,ZmFrZS1wcmV2aWV3')
    expect(wrapper.text()).toContain('参考图 1')
    expect(wrapper.text()).not.toContain('预览失败')

    await wrapper.find('input[placeholder*="无线耳机"]').setValue('护肤品封面')
    await wrapper.find('button.generate-button').trigger('click')
    await flushPromises()

    const form = apiClient.post.mock.calls[0][1] as FormData
    expect(form.get('image')).toBe(file)

    wrapper.unmount()
  })

  it('shows a clear reference preview failure instead of a broken image state', async () => {
    stubFileReader('error')

    const wrapper = mountImageStudio()
    const file = new File(['fake-image'], 'reference.png', { type: 'image/png' })
    const fileInput = wrapper.find('input[type="file"]')
    Object.defineProperty(fileInput.element, 'files', {
      value: [file],
      configurable: true,
    })

    await fileInput.trigger('change')
    await flushPromises()

    expect(wrapper.find('img[alt="参考素材预览"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('参考图预览失败，请重新上传 JPG / PNG / WEBP 图片。')
    expect(appStore.showError).toHaveBeenCalledWith('参考图预览失败，请重新上传 JPG / PNG / WEBP 图片。')

    wrapper.unmount()
  })

  it('clears the previous reference preview when an unsupported file is selected', async () => {
    stubFileReader()

    const wrapper = mountImageStudio()
    const fileInput = wrapper.find('input[type="file"]')
    const validFile = new File(['fake-image'], 'reference.png', { type: 'image/png' })
    Object.defineProperty(fileInput.element, 'files', {
      value: [validFile],
      configurable: true,
    })

    await fileInput.trigger('change')
    await flushPromises()

    expect(wrapper.find('img[alt="参考素材预览"]').exists()).toBe(true)

    const invalidFile = new File(['not-an-image'], 'reference.txt', { type: 'text/plain' })
    Object.defineProperty(fileInput.element, 'files', {
      value: [invalidFile],
      configurable: true,
    })

    await fileInput.trigger('change')
    await flushPromises()

    expect(wrapper.find('img[alt="参考素材预览"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('请上传 JPG / PNG / WEBP 图片。')
    expect(appStore.showError).toHaveBeenCalledWith('请上传 JPG / PNG / WEBP 图片。')

    wrapper.unmount()
  })

  it('keeps upstream errors generic instead of hard-coding a model name', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
    apiClient.post.mockRejectedValue({
      response: {
        data: {
          error: {
            message: 'account does not support OpenAI Images API',
          },
        },
      },
    })

    const wrapper = mountImageStudio()
    await wrapper.find('input[placeholder*="无线耳机"]').setValue('护肤精华')
    await wrapper.find('button.generate-button').trigger('click')
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('当前账号暂不支持图片生成/改图接口')
    expect(text).toContain('支持图片生成的模型或上游账号')
    expect(text).not.toContain('gpt-image-2')
    expect(authStore.refreshUser).not.toHaveBeenCalled()
    expect(consoleError).toHaveBeenCalledTimes(1)

    consoleError.mockRestore()
    wrapper.unmount()
  })

  it('hides technical HTTP failures from image generation users', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
    apiClient.post.mockRejectedValue({
      response: {
        status: 502,
        data: {
          message: 'Request failed with status code 502',
        },
      },
    })

    const wrapper = mountImageStudio()
    await wrapper.find('textarea').setValue('minimal skincare product cover')
    await wrapper.find('button.generate-button').trigger('click')
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('图片生成服务暂不可用，请稍后重试或联系管理员。')
    expect(text).not.toContain('Request failed with status code 502')
    expect(appStore.showError).toHaveBeenCalledWith('图片生成服务暂不可用，请稍后重试或联系管理员。')
    expect(authStore.refreshUser).not.toHaveBeenCalled()
    expect(consoleError).toHaveBeenCalledTimes(1)

    consoleError.mockRestore()
    wrapper.unmount()
  })
})
