import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import VersionBadge from '../VersionBadge.vue'

const { app, auth, update, restart, rollback } = vi.hoisted(() => ({
  app: { currentVersion: '0.2.7-ssxz', latestVersion: '0.2.7', hasUpdate: false, versionLoading: false, buildType: 'source', releaseInfo: { html_url: 'https://github.com/Wei-Shaw/sub2api/releases/tag/v0.2.7' }, fetchVersion: vi.fn() },
  auth: { isAdmin: true }, update: vi.fn(), restart: vi.fn(), rollback: vi.fn(),
}))
vi.mock('@/stores', () => ({ useAppStore: () => app, useAuthStore: () => auth }))
vi.mock('@/api/admin/system', () => ({ performUpdate: update, restartService: restart, rollback }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const wrappers: ReturnType<typeof mount>[] = []

describe('managed SSXZ release information', () => {
  beforeEach(() => { vi.clearAllMocks(); app.hasUpdate = false; app.buildType = 'source' })
  afterEach(() => { wrappers.splice(0).forEach(wrapper => wrapper.unmount()) })
  it.each([false, true])('keeps release notes discoverable with hasUpdate=%s without enabling production actions', async hasUpdate => {
    app.hasUpdate = hasUpdate
    const wrapper = mount(VersionBadge, { props: { runtimeActionsEnabled: false }, global: { stubs: { Icon: true, ConfirmDialog: true } } })
    wrappers.push(wrapper)
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(wrapper.get('a').attributes('href')).toBe(app.releaseInfo.html_url)
    expect(wrapper.get('a').attributes('rel')).toBe('noopener noreferrer')
    expect(wrapper.find('[data-testid="version-update-action"]').exists()).toBe(false)
    for (const action of ['update', 'rollback', 'restart']) {
      expect(wrapper.get(`[data-testid="managed-${action}-action"]`).attributes('disabled')).toBeDefined()
    }
    expect(update).not.toHaveBeenCalled()
    expect(restart).not.toHaveBeenCalled()
    expect(rollback).not.toHaveBeenCalled()
  })
})
