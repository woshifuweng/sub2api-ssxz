import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const { authState, appState, systemApi } = vi.hoisted(() => ({
  authState: {
    isAdmin: true
  },
  appState: {
    versionLoading: false,
    currentVersion: '1.2.3',
    latestVersion: '1.2.4',
    hasUpdate: true,
    buildType: 'release',
    releaseInfo: null,
    fetchVersion: vi.fn().mockResolvedValue(null),
    clearVersionCache: vi.fn()
  },
  systemApi: {
    performUpdate: vi.fn().mockResolvedValue({ message: 'updated', need_restart: false }),
    rollback: vi.fn().mockResolvedValue({ message: 'rolled back', need_restart: false }),
    restartService: vi.fn().mockResolvedValue({ message: 'restarting' })
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => authState,
  useAppStore: () => appState
}))

vi.mock('@/api/admin/system', () => systemApi)

import VersionBadge from '../VersionBadge.vue'

const ConfirmDialogStub = defineComponent({
  name: 'ConfirmDialog',
  props: {
    show: Boolean,
    title: String,
    message: String,
    confirmText: String,
    danger: Boolean
  },
  emits: ['confirm', 'cancel'],
  template: `
    <div v-if="show" data-testid="confirm-dialog">
      <span>{{ title }}</span>
      <button data-testid="confirm-runtime-action" @click="$emit('confirm')">confirm</button>
      <button @click="$emit('cancel')">cancel</button>
    </div>
  `
})

function mountBadge(runtimeActionsEnabled = false) {
  return mount(VersionBadge, {
    props: { runtimeActionsEnabled },
    global: {
      stubs: {
        ConfirmDialog: ConfirmDialogStub
      }
    }
  })
}

async function openBadge(wrapper: ReturnType<typeof mountBadge>) {
  await wrapper.get('button').trigger('click')
}

describe('VersionBadge', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    authState.isAdmin = true
    appState.versionLoading = false
    appState.currentVersion = '1.2.3'
    appState.latestVersion = '1.2.4'
    appState.hasUpdate = true
    appState.buildType = 'release'
    appState.releaseInfo = null
    appState.fetchVersion.mockClear()
    appState.clearVersionCache.mockClear()
    systemApi.performUpdate.mockClear()
    systemApi.rollback.mockClear()
    systemApi.restartService.mockClear()
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
  })

  it('shows check updates but disables runtime actions for the managed deployment', async () => {
    const wrapper = mountBadge()
    await openBadge(wrapper)

    expect(wrapper.text()).toContain('version.checkUpdates')
    expect(wrapper.text()).toContain('version.managedDeployment')
    expect(wrapper.get('[data-testid="managed-update-action"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-testid="managed-rollback-action"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-testid="managed-restart-action"]').attributes('disabled')).toBeDefined()

    await wrapper.get('[data-testid="managed-update-action"]').trigger('click')
    expect(systemApi.performUpdate).not.toHaveBeenCalled()
    expect(systemApi.rollback).not.toHaveBeenCalled()
    expect(systemApi.restartService).not.toHaveBeenCalled()
  })

  it.each([
    ['update', 'version-update-action', 'confirmUpdateTitle', systemApi.performUpdate],
    ['rollback', 'version-rollback-action', 'confirmRollbackTitle', systemApi.rollback],
    ['restart', 'version-restart-action', 'confirmRestartTitle', systemApi.restartService]
  ])('requires confirmation before the %s action', async (_action, testId, titleKey, apiCall) => {
    const wrapper = mountBadge(true)
    await openBadge(wrapper)

    await wrapper.get(`[data-testid="${testId}"]`).trigger('click')
    expect(apiCall).not.toHaveBeenCalled()
    expect(wrapper.get('[data-testid="confirm-dialog"]').text()).toContain(`version.${titleKey}`)

    await wrapper.get('[data-testid="confirm-runtime-action"]').trigger('click')
    await flushPromises()
    expect(apiCall).toHaveBeenCalledTimes(1)

    wrapper.unmount()
  })

  it('does not load or render the admin control for non-admin users', () => {
    authState.isAdmin = false
    const wrapper = mountBadge()

    expect(wrapper.text()).toBe('')
    expect(appState.fetchVersion).not.toHaveBeenCalled()
  })
})
