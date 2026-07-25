import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, string>) => {
      const messages: Record<string, string> = {
        'profile.avatar.dialogTitle': '更换头像',
        'profile.avatar.chooseHint': '请选择一张图片',
        'profile.avatar.uploadTitle': '选择头像图片',
        'profile.avatar.uploadButton': '选择图片',
        'profile.avatar.uploadHint': '支持 JPEG、PNG、WebP，最大 5MB；也可以拖入图片',
        'profile.avatar.selectedFile': `已选择：${values?.name || ''}`,
        'profile.avatar.zoomLabel': '缩放',
        'profile.avatar.cropHint': '头像会以图片中心为基准自动居中裁剪，并在浏览器中重新编码后上传。',
        'common.cancel': '取消',
        'profile.avatar.saving': '保存中...',
        'profile.avatar.save': '保存头像'
      }
      return messages[key] || key
    }
  })
}))

import AvatarCropDialog from '../AvatarCropDialog.vue'

function mountDialog() {
  return mount(AvatarCropDialog, {
    props: { show: true },
    global: {
      stubs: {
        BaseDialog: {
          template: '<div><slot /><slot name="footer" /></div>'
        }
      }
    }
  })
}

describe('AvatarCropDialog', () => {
  it('replaces the native file control with a custom upload zone', () => {
    const wrapper = mountDialog()

    expect(wrapper.find('.avatar-upload-zone').exists()).toBe(true)
    expect(wrapper.find('.avatar-upload-button').text()).toContain('选择图片')
    expect(wrapper.find('input[type="file"]').classes()).toContain('avatar-file-input')
    expect(wrapper.find('input[type="file"]').classes()).not.toContain('input')
    expect(wrapper.text()).toContain('支持 JPEG、PNG、WebP')
  })

  it('opens the hidden file picker from the custom button', async () => {
    const wrapper = mountDialog()
    const click = vi.spyOn(HTMLInputElement.prototype, 'click')

    await wrapper.find('.avatar-upload-button').trigger('click')

    expect(click).toHaveBeenCalledOnce()
    click.mockRestore()
  })

  it('describes the existing center crop behavior accurately', () => {
    const wrapper = mountDialog()

    expect(wrapper.text()).toContain('自动居中裁剪')
    expect(wrapper.text()).not.toContain('裁剪为圆形')
  })
})
