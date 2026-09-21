/** Also works when locale/router initialization has not completed. Never retries paid work. */
export function showPageLoadFailure(): void {
  const status = document.getElementById('ssxz-startup-status')
  if (status) {
    status.textContent = '页面加载失败，请检查网络后刷新。Page loading failed; please refresh.'
    status.setAttribute('role', 'alert')
    return
  }
  if (document.getElementById('ssxz-page-load-error')) return
  const notice = document.createElement('section')
  notice.id = 'ssxz-page-load-error'
  notice.setAttribute('role', 'alert')
  notice.style.cssText = 'position:fixed;bottom:24px;left:16px;right:16px;z-index:99999;padding:18px;background:#18181b;color:#fafafa;border:1px solid #71717a;border-radius:12px;font:14px/1.6 system-ui;max-width:560px;margin:auto'
  const message = document.createElement('p')
  message.textContent = '页面资源加载失败，当前内容已保留。请先保存未提交的内容，再手动刷新。Page loading failed; save your work before refreshing.'
  const retry = document.createElement('a')
  // Empty relative URL reloads this document without copying sensitive query strings into logs.
  retry.href = ''
  retry.textContent = '刷新页面 / Refresh'
  retry.style.cssText = 'color:#fff;text-decoration:underline'
  notice.append(message, retry)
  document.body.append(notice)
}
