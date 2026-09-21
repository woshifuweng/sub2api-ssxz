import { afterEach, describe, expect, it, vi } from 'vitest'
import { showPageLoadFailure } from '../pageLoadRecovery'

afterEach(() => { document.body.replaceChildren(); vi.restoreAllMocks() })

describe('page loading recovery', () => {
  it('replaces the startup placeholder with an actionable error', () => {
    document.body.innerHTML = '<div id="app"><section id="ssxz-startup"><h1 id="ssxz-startup-title">Loading</h1><p id="ssxz-startup-status">Loading</p><p id="ssxz-startup-detail"></p><a id="ssxz-startup-retry" hidden>重新加载页面</a></section></div>'
    showPageLoadFailure({ stage: '资源加载' })
    expect(document.querySelector('#ssxz-startup-title')?.textContent).toContain('页面暂时没有加载完成')
    expect(document.querySelector('#ssxz-startup-status')?.textContent).toContain('资源加载')
    expect(document.querySelector('#ssxz-startup-retry')?.textContent).toContain('重新加载')
    expect(document.querySelector('[role="alert"]')).not.toBeNull()
  })

  it('keeps the mounted app intact and offers one manual retry, not automatic reloads', () => {
    document.body.innerHTML = '<div id="app"><form>unsaved work</form></div>'
    const timer = vi.spyOn(window, 'setTimeout')
    for (let i = 0; i < 4; i++) showPageLoadFailure()
    expect(document.querySelector('#app')?.textContent).toBe('unsaved work')
    expect(document.querySelectorAll('#ssxz-page-load-error')).toHaveLength(1)
    expect(document.querySelector('#ssxz-page-load-error a')?.textContent).toContain('重新加载')
    expect(timer).not.toHaveBeenCalled()
  })
})
