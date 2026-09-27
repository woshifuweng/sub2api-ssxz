(() => {
  const showStartupFailure = () => {
    const root = document.getElementById('ssxz-startup')
    const title = document.getElementById('ssxz-startup-title')
    const status = document.getElementById('ssxz-startup-status')
    const detail = document.getElementById('ssxz-startup-detail')
    const retry = document.getElementById('ssxz-startup-retry')
    if (!root || !status || !retry) return
    root.dataset.state = 'error'
    if (title) title.textContent = '页面暂时没有加载完成'
    status.textContent = '网页资源没有成功启动，请重新加载一次。'
    if (detail) detail.textContent = '如果问题持续出现，请检查网络或关闭拦截网页资源的浏览器插件。'
    retry.hidden = false
    retry.href = window.location.href
  }

  window.addEventListener('error', (event) => {
    if (event.target instanceof HTMLScriptElement) showStartupFailure()
  }, true)
  window.addEventListener('unhandledrejection', showStartupFailure)
})()
