import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { authStore, soraApi } = vi.hoisted(() => ({
  authStore: {
    user: {
      balance: 1,
    },
    refreshUser: vi.fn(),
  },
  soraApi: {
    listGenerations: vi.fn(),
  },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore,
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
  return mount(ImageStudioView)
}

describe('ImageStudioView first-screen guidance', () => {
  beforeEach(() => {
    authStore.user.balance = 1
    authStore.refreshUser.mockReset()
    soraApi.listGenerations.mockReset()
    soraApi.listGenerations.mockResolvedValue({
      data: [],
      total: 0,
      page: 1,
    })
    vi.unstubAllGlobals()
  })

  it('frames the page as a general image generation workspace', () => {
    const wrapper = mountImageStudio()
    const text = wrapper.text()

    expect(text).toContain('图片生成工作台')
    expect(text).toContain('商品图、海报和灵感图，一页完成')
    expect(text).toContain('上传参考图会进入改图流程')
    expect(text).toContain('创作设置')
    expect(text).toContain('参考图片')
    expect(text).toContain('文生图模式')
    expect(text).toContain('开始一次创作')
    expect(text).toContain('生成结果会出现在这里，方便预览和下载')
    expect(text).not.toContain('Sora')

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

  it('keeps upstream errors generic instead of hard-coding a model name', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        json: vi.fn().mockResolvedValue({
          error: {
            message: 'account does not support OpenAI Images API',
          },
        }),
      }),
    )

    const wrapper = mountImageStudio()
    await wrapper.get('button.btn-primary').trigger('click')
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
})
