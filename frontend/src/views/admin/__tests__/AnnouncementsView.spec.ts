import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { listAnnouncements, getGroups, showError } = vi.hoisted(() => ({
  listAnnouncements: vi.fn(),
  getGroups: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    announcements: {
      list: listAnnouncements,
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
      getReadStatus: vi.fn()
    },
    groups: {
      getAll: getGroups
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess: vi.fn()
  })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => ({
      'admin.announcements.failedToLoad': '加载公告失败',
      'admin.announcements.retryHint': '公告暂时无法加载，请重试。',
      'admin.announcements.retry': '重试',
      'admin.announcements.noAnnouncements': '暂无公告',
      'admin.announcements.noAnnouncementsDescription': '创建您的第一条公告，向用户发布重要信息。',
      'admin.announcements.createFirstAnnouncement': '创建您的第一条公告',
      'admin.announcements.createAnnouncement': '创建公告',
      'admin.announcements.title': '公告管理',
      'admin.announcements.description': '发布与管理站内公告',
      'admin.announcements.searchAnnouncements': '搜索公告',
      'admin.announcements.allStatus': '全部状态',
      'admin.announcements.columns.title': '标题',
      'admin.announcements.columns.status': '状态',
      'admin.announcements.columns.notifyMode': '通知方式',
      'admin.announcements.columns.targeting': '展示条件',
      'admin.announcements.columns.timeRange': '有效期',
      'admin.announcements.columns.createdAt': '创建时间',
      'admin.announcements.columns.actions': '操作',
      'admin.announcements.statusLabels.draft': '草稿',
      'admin.announcements.statusLabels.active': '展示中',
      'admin.announcements.statusLabels.archived': '已归档',
      'admin.announcements.notifyModeLabels.silent': '静默',
      'admin.announcements.notifyModeLabels.popup': '弹窗',
      'admin.announcements.targetingSummaryAll': '全部用户',
      'admin.announcements.targetingSummaryCustom': '自定义（{groups} 组）',
      'admin.announcements.timeImmediate': '立即',
      'admin.announcements.timeNever': '永久',
      'admin.announcements.readStatus': '已读情况',
      'admin.announcements.form.startsAt': '开始时间',
      'admin.announcements.form.endsAt': '结束时间',
      'admin.announcements.editAnnouncement': '编辑公告',
      'admin.announcements.deleteAnnouncement': '删除公告',
      'admin.announcements.deleteConfirm': '确定删除？',
      'common.refresh': '刷新',
      'common.edit': '编辑',
      'common.delete': '删除',
      'common.cancel': '取消',
      'common.saving': '保存中',
      'common.save': '保存',
      'common.success': '成功'
      })[key] ?? key
    })
  }
})

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: { template: '<div><slot /></div>' }
}))

vi.mock('@/components/admin/AdminPageHeader.vue', () => ({
  default: { template: '<header><slot name="actions" /></header>' }
}))

vi.mock('@/components/layout/TablePageLayout.vue', () => ({
  default: {
    template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
  }
}))

vi.mock('@/components/common/DataTable.vue', () => ({
  default: {
    props: ['data', 'loading'],
    template: '<div data-testid="announcement-table"><slot name="empty" v-if="!loading && data.length === 0" /></div>'
  }
}))

vi.mock('@/components/common/EmptyState.vue', () => ({
  default: {
    props: ['title', 'description', 'actionText'],
    emits: ['action'],
    template: '<section data-testid="announcement-empty"><h2>{{ title }}</h2><p>{{ description }}</p><button v-if="actionText" data-testid="announcement-empty-action" @click="$emit(\'action\')">{{ actionText }}</button></section>'
  }
}))

vi.mock('@/components/common/Select.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/components/common/Pagination.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/components/common/BaseDialog.vue', () => ({ default: { template: '<div><slot /><slot name="footer" /></div>' } }))
vi.mock('@/components/common/ConfirmDialog.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/components/icons/Icon.vue', () => ({ default: { template: '<span />' } }))
vi.mock('@/components/admin/announcements/AnnouncementTargetingEditor.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/components/admin/announcements/AnnouncementReadStatusDialog.vue', () => ({ default: { template: '<div />' } }))

import AnnouncementsView from '../AnnouncementsView.vue'

function emptyResponse() {
  return { items: [], total: 0, pages: 0, page: 1, page_size: 20 }
}

describe('admin AnnouncementsView empty and error states', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getGroups.mockResolvedValue([])
  })

  it('shows a creation empty state when the list loads successfully with no announcements', async () => {
    listAnnouncements.mockResolvedValue(emptyResponse())

    const wrapper = mount(AnnouncementsView)
    await flushPromises()

    expect(wrapper.get('[data-testid="announcement-empty"]').text()).toContain('暂无公告')
    expect(wrapper.get('[data-testid="announcement-empty-action"]').text()).toBe('创建您的第一条公告')
    expect(wrapper.text()).not.toContain('加载公告失败')
  })

  it('shows a retryable error state when the list request fails', async () => {
    listAnnouncements.mockRejectedValueOnce(new Error('network failure'))

    const wrapper = mount(AnnouncementsView)
    await flushPromises()

    expect(wrapper.get('[data-testid="announcement-empty"]').text()).toContain('加载公告失败')
    expect(wrapper.get('[data-testid="announcement-empty-action"]').text()).toBe('重试')

    listAnnouncements.mockResolvedValueOnce(emptyResponse())
    await wrapper.get('[data-testid="announcement-empty-action"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="announcement-empty"]').text()).toContain('暂无公告')
    expect(showError).toHaveBeenCalledTimes(1)
  })
})
