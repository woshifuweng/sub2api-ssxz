import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { appStore, authStore, apiClient, soraApi, userChannelsApi, userGroupsApi } = vi.hoisted(() => ({
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
  userChannelsApi: {
    getAvailable: vi.fn(),
  },
  userGroupsApi: {
    getAvailable: vi.fn(),
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

vi.mock('@/api', () => ({
  userChannelsAPI: userChannelsApi,
  userGroupsAPI: userGroupsApi,
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

function stubReferencePreviewUrl(result: 'success' | 'error' = 'success') {
  const createObjectURL = vi.fn((file: File) => {
    if (result === 'error') {
      throw new Error('preview failed')
    }
    return `blob:reference-preview-${file.name}`
  })
  const revokeObjectURL = vi.fn()

  vi.stubGlobal('URL', {
    createObjectURL,
    revokeObjectURL,
  })

  return { createObjectURL, revokeObjectURL }
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
    userChannelsApi.getAvailable.mockReset()
    userGroupsApi.getAvailable.mockReset()
    soraApi.listGenerations.mockResolvedValue({
      data: [],
      total: 0,
      page: 1,
    })
    userChannelsApi.getAvailable.mockResolvedValue([])
    userGroupsApi.getAvailable.mockResolvedValue([])
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

  it('shows image-capable models from the user catalog without making them chat models', async () => {
    userChannelsApi.getAvailable.mockResolvedValue([
      {
        name: 'Image Channel',
        description: '',
        platforms: [
          {
            platform: 'image-provider',
            groups: [],
            supported_models: [
              {
                name: 'gpt-image-2',
                platform: 'image-provider',
                pricing: null,
                capabilities: ['image_generation'],
                provider_label: 'workspace-openai-compatible-image-staging',
                model_catalog_source: 'real_channel',
                pricing_status: 'configured',
              },
              {
                name: 'gemini-2.5-flash-image',
                platform: 'gemini',
                pricing: null,
                capabilities: ['image_generation'],
                provider_label: 'gemini',
                model_catalog_source: 'real_channel',
                pricing_status: 'configured',
              },
              {
                name: 'deepseek-v4-flash',
                platform: 'openai',
                pricing: null,
                capabilities: ['text_chat'],
                provider_label: 'openai',
                model_catalog_source: 'real_channel',
                pricing_status: 'configured',
              },
            ],
          },
        ],
      },
    ])

    const wrapper = mountImageStudio()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('图片模型')
    expect(text).toContain('Gpt-Image-2')
    expect(text).toContain('Gemini-2.5-Flash-Image')
    expect(text).not.toContain('Deepseek-V4-Flash')
    const optionValues = wrapper.findAll('.hero-model-select option').map((option) => option.attributes('value'))
    expect(optionValues).toContain('gpt-image-2')
    expect(optionValues).toContain('gemini-2.5-flash-image')

    wrapper.unmount()
  })

  it('hides non-real image-capable models from the image selector', async () => {
    userChannelsApi.getAvailable.mockResolvedValue([
      {
        name: 'Image Channel',
        description: '',
        platforms: [
          {
            platform: 'image-provider',
            groups: [],
            supported_models: [
              {
                name: 'gpt-image-2',
                platform: 'image-provider',
                pricing: null,
                capabilities: ['image_generation'],
                provider_label: 'workspace-openai-compatible-image-staging',
                model_catalog_source: 'real_channel',
                pricing_status: 'configured',
              },
              {
                name: 'gpt-image-env',
                platform: 'image-provider',
                pricing: null,
                capabilities: ['image_generation'],
                model_catalog_source: 'env_gate',
              },
              {
                name: 'workspace-image-fake-model',
                platform: 'workspace-image-fake',
                pricing: null,
                capabilities: ['image_generation'],
                provider_label: 'workspace-image-fake',
                model_catalog_source: 'fake_gate',
                fake: true,
                test_only: true,
              },
            ],
          },
        ],
      },
    ])

    const wrapper = mountImageStudio()
    await flushPromises()

    const optionValues = wrapper.findAll('.hero-model-select option').map((option) => option.attributes('value'))
    expect(optionValues).toEqual(['gpt-image-2'])
    expect(wrapper.text()).not.toContain('Gpt-Image-Env')
    expect(wrapper.text()).not.toContain('Workspace-Image-Fake-Model')

    wrapper.unmount()
  })

  it('defaults image generation to the verified gpt-image-2 model when available', async () => {
    userChannelsApi.getAvailable.mockResolvedValue([
      {
        name: 'Image Channel',
        description: '',
        platforms: [
          {
            platform: 'image-provider',
            groups: [],
            supported_models: [
              {
                name: 'gemini-3.1-flash-image-preview',
                platform: 'gemini',
                pricing: null,
                capabilities: ['image_generation'],
                provider_label: 'gemini',
                model_catalog_source: 'real_channel',
                pricing_status: 'configured',
              },
              {
                name: 'gpt-image-2',
                platform: 'image-provider',
                pricing: null,
                capabilities: ['image_generation'],
                provider_label: 'workspace-openai-compatible-image-staging',
                model_catalog_source: 'real_channel',
                pricing_status: 'configured',
              },
            ],
          },
        ],
      },
    ])

    const wrapper = mountImageStudio()
    await flushPromises()

    const select = wrapper.find('.hero-model-select').element as HTMLSelectElement
    expect(select.value).toBe('gpt-image-2')

    wrapper.unmount()
  })

  it('falls back to the first image model when the verified model is unavailable', async () => {
    userChannelsApi.getAvailable.mockResolvedValue([
      {
        name: 'Image Channel',
        description: '',
        platforms: [
          {
            platform: 'image-provider',
            groups: [],
            supported_models: [
              {
                name: 'gemini-3.1-flash-image-preview',
                platform: 'gemini',
                pricing: null,
                capabilities: ['image_generation'],
                provider_label: 'gemini',
                model_catalog_source: 'real_channel',
                pricing_status: 'configured',
              },
            ],
          },
        ],
      },
    ])

    const wrapper = mountImageStudio()
    await flushPromises()

    const select = wrapper.find('.hero-model-select').element as HTMLSelectElement
    expect(select.value).toBe('gemini-3.1-flash-image-preview')

    wrapper.unmount()
  })

  it('submits the selected image model to the image-studio API', async () => {
    userChannelsApi.getAvailable.mockResolvedValue([
      {
        name: 'Image Channel',
        description: '',
        platforms: [
          {
            platform: 'image-provider',
            groups: [],
            supported_models: [
              {
                name: 'gpt-image-2',
                platform: 'image-provider',
                pricing: null,
                capabilities: ['image_generation'],
                provider_label: 'workspace-openai-compatible-image-staging',
                model_catalog_source: 'real_channel',
                pricing_status: 'configured',
              },
              {
                name: 'gemini-2.5-flash-image',
                platform: 'gemini',
                pricing: null,
                capabilities: ['image_generation'],
                provider_label: 'gemini',
                model_catalog_source: 'real_channel',
                pricing_status: 'configured',
              },
            ],
          },
        ],
      },
    ])
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
    await flushPromises()

    await wrapper.find('.hero-model-select').setValue('gemini-2.5-flash-image')
    await wrapper.find('textarea').setValue('minimal skincare product cover')
    await wrapper.find('button.generate-button').trigger('click')
    await flushPromises()

    const form = apiClient.post.mock.calls[0][1] as FormData
    expect(form.get('model')).toBe('gemini-2.5-flash-image')

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

  it('opens a focused preview for recent image works', async () => {
    soraApi.listGenerations.mockResolvedValue({
      data: [
        {
          id: 12,
          user_id: 7,
          model: 'gpt-image-2',
          prompt: 'skincare cover',
          media_type: 'image',
          status: 'completed',
          storage_type: 'upstream',
          media_url: 'https://cdn.example.com/preview.png',
          media_urls: ['https://cdn.example.com/preview.png'],
          s3_object_keys: [],
          file_size_bytes: 0,
          error_message: '',
          created_at: '2026-06-20T00:00:00Z',
        },
      ],
      total: 1,
      page: 1,
    })

    const wrapper = mountImageStudio()
    await flushPromises()

    await wrapper.find('.recent-thumb-button').trigger('click')

    const dialog = wrapper.find('.recent-preview-dialog')
    expect(dialog.exists()).toBe(true)
    expect(dialog.attributes('role')).toBe('dialog')
    expect(dialog.text()).toContain('图片作品')
    expect(dialog.find('img[alt="preview-work-12"]').attributes('src')).toBe('https://cdn.example.com/preview.png')
    expect(wrapper.find('.recent-thumb-hint').text()).toBe('点击预览')

    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    await wrapper.find('.recent-preview-actions .secondary-button').trigger('click')
    expect(click).toHaveBeenCalledTimes(1)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()
    expect(wrapper.find('.recent-preview-dialog').exists()).toBe(false)

    await wrapper.find('.recent-thumb-button').trigger('click')
    expect(wrapper.find('.recent-preview-dialog').exists()).toBe(true)
    await wrapper.find('.recent-preview-close').trigger('click')
    expect(wrapper.find('.recent-preview-dialog').exists()).toBe(false)

    await wrapper.find('.recent-card-actions .secondary-button').trigger('click')
    expect(wrapper.find('.recent-preview-dialog').exists()).toBe(true)

    click.mockRestore()
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
          {
            url: 'https://cdn.example.com/result-3.png',
          },
        ],
      },
    })

    const wrapper = mountImageStudio()
    const countButtons = wrapper.findAll('.count-chip')
    expect(countButtons).toHaveLength(3)
    await countButtons[2].trigger('click')
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
    expect(form.get('count')).toBe('3')
    expect(String(form.get('selling_points'))).toContain('高级感电商广告图')
    expect(String(form.get('selling_points'))).toContain('创作用途：商品主图')
    expect(String(form.get('selling_points'))).toContain('商用安全')
    expect(wrapper.find('img[alt="result-1"]').attributes('src')).toBe('https://cdn.example.com/result.png')
    expect(wrapper.findAll('.thumbnail-button')).toHaveLength(3)
    expect(wrapper.findAll('.result-actions .secondary-button')[2].attributes('disabled')).toBeUndefined()
    expect(authStore.refreshUser).toHaveBeenCalledTimes(1)
    expect(appStore.showSuccess).toHaveBeenCalledWith('图片生成完成')

    wrapper.unmount()
  })

  it('renders an uploaded reference image preview and submits the original file', async () => {
    const previewUrl = stubReferencePreviewUrl()
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

    const preview = wrapper.find('.asset-thumb img')
    expect(preview.exists()).toBe(true)
    expect(preview.attributes('src')).toBe('blob:reference-preview-reference.png')
    expect(previewUrl.createObjectURL).toHaveBeenCalledWith(file)
    expect(wrapper.text()).toContain('参考图 1')
    expect(wrapper.text()).not.toContain('预览失败')

    await wrapper.find('input[placeholder*="无线耳机"]').setValue('护肤品封面')
    await wrapper.find('button.generate-button').trigger('click')
    await flushPromises()

    const form = apiClient.post.mock.calls[0][1] as FormData
    expect(form.get('image')).toBe(file)

    wrapper.unmount()
    expect(previewUrl.revokeObjectURL).toHaveBeenCalledWith('blob:reference-preview-reference.png')
  })

  it('accepts a dragged reference image through the same preview and submit path', async () => {
    const previewUrl = stubReferencePreviewUrl()
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
    const file = new File(['dragged-image'], 'dragged-reference.webp', { type: 'image/webp' })
    const dropZone = wrapper.get('.asset-drop')

    await dropZone.trigger('dragenter')
    expect(dropZone.classes()).toContain('dragging')

    ;(wrapper.vm as unknown as {
      handleReferenceDrop: (event: DragEvent) => void
    }).handleReferenceDrop({
      dataTransfer: {
        files: [file],
      },
    } as unknown as DragEvent)
    await wrapper.vm.$nextTick()
    await flushPromises()

    expect(previewUrl.createObjectURL).toHaveBeenCalledWith(file)
    const preview = wrapper.find('.asset-thumb img')
    expect(preview.exists()).toBe(true)
    expect(preview.attributes('src')).toBe('blob:reference-preview-dragged-reference.webp')

    await wrapper.find('textarea').setValue('minimal skincare product cover')
    await wrapper.find('button.generate-button').trigger('click')
    await flushPromises()

    const form = apiClient.post.mock.calls[0][1] as FormData
    expect(form.get('image')).toBe(file)

    wrapper.unmount()
    expect(previewUrl.revokeObjectURL).toHaveBeenCalledWith('blob:reference-preview-dragged-reference.webp')
  })

  it('clears the reference image when browser preview loading fails', async () => {
    const previewUrl = stubReferencePreviewUrl()
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
    expect(wrapper.find('.asset-thumb img').exists()).toBe(true)

    await wrapper.find('.asset-thumb img').trigger('error')
    await flushPromises()

    expect(wrapper.find('.asset-thumb img').exists()).toBe(false)
    expect(wrapper.text()).toContain('参考图预览加载失败，请重新上传 JPG / PNG / WEBP 图片。')
    expect(appStore.showError).toHaveBeenCalledWith('参考图预览加载失败，请重新上传 JPG / PNG / WEBP 图片。')
    expect(previewUrl.revokeObjectURL).toHaveBeenCalledWith('blob:reference-preview-reference.png')

    await wrapper.find('textarea').setValue('minimal skincare product cover')
    await wrapper.find('button.generate-button').trigger('click')
    await flushPromises()

    const form = apiClient.post.mock.calls[0][1] as FormData
    expect(form.get('image')).toBeNull()

    wrapper.unmount()
  })

  it('shows a clear reference preview failure instead of a broken image state', async () => {
    stubReferencePreviewUrl('error')

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
    const previewUrl = stubReferencePreviewUrl()

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
    expect(previewUrl.revokeObjectURL).toHaveBeenCalledWith('blob:reference-preview-reference.png')

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
    expect(text).toContain('本次没有生成成功作品')
    expect(text).toContain('不会保存到历史')
    expect(text).toContain('未成功返回结果不会扣生成费用')
    expect(text).toContain('可以调整提示词后重试')
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
    expect(text).toContain('本次没有生成成功作品')
    expect(text).toContain('不会保存到历史')
    expect(text).toContain('未成功返回结果不会扣生成费用')
    expect(text).toContain('可以调整提示词后重试')
    expect(text).not.toContain('Request failed with status code 502')
    expect(appStore.showError).toHaveBeenCalledWith('图片生成服务暂不可用，请稍后重试或联系管理员。')
    expect(authStore.refreshUser).not.toHaveBeenCalled()
    expect(consoleError).toHaveBeenCalledTimes(1)

    consoleError.mockRestore()
    wrapper.unmount()
  })
})
