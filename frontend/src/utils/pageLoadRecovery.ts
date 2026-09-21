export interface PageLoadFailureOptions {
  stage?: string
  retryHref?: string
}

/** Also works when locale/router initialization has not completed. Never retries paid work. */
export function showPageLoadFailure(options: PageLoadFailureOptions = {}): void {
  const stage = options.stage || '网页资源'
  const retryHref = options.retryHref || ''
  const startup = document.getElementById('ssxz-startup')
  const status = document.getElementById('ssxz-startup-status')
  const title = document.getElementById('ssxz-startup-title')
  const detail = document.getElementById('ssxz-startup-detail')
  const retry = document.getElementById('ssxz-startup-retry') as HTMLAnchorElement | null
  if (startup && status) {
    startup.setAttribute('data-state', 'error')
    if (title) title.textContent = '页面暂时没有加载完成'
    status.textContent = `${stage}没有成功启动，请重新加载一次。`
    status.setAttribute('role', 'alert')
    if (detail) detail.textContent = '当前没有执行任何付费操作。若问题持续出现，请检查网络或关闭资源拦截插件。'
    if (retry) {
      retry.hidden = false
      retry.href = retryHref
    }
    return
  }
  if (document.getElementById('ssxz-page-load-error')) return
  const notice = document.createElement('section')
  notice.id = 'ssxz-page-load-error'
  notice.setAttribute('role', 'alert')
  notice.style.cssText = 'position:fixed;bottom:24px;left:16px;right:16px;z-index:99999;padding:18px;background:#111827;color:#f8fafc;border:1px solid rgba(148,163,184,.35);border-radius:14px;font:14px/1.6 system-ui;max-width:560px;margin:auto;box-shadow:0 20px 55px rgba(0,0,0,.35)'
  const message = document.createElement('p')
  message.textContent = `${stage}加载失败，当前内容已保留。请先保存未提交的内容，再手动刷新。`
  const retryLink = document.createElement('a')
  // Empty relative URL reloads this document without copying sensitive query strings into logs.
  retryLink.href = retryHref
  retryLink.textContent = '重新加载页面'
  retryLink.style.cssText = 'display:inline-flex;margin-top:10px;padding:7px 12px;border:1px solid rgba(148,163,184,.35);border-radius:8px;color:#f8fafc;text-decoration:none;background:rgba(148,163,184,.1)'
  notice.append(message, retryLink)
  document.body.append(notice)
}
