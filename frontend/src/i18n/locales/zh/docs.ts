export default {
  docs: {
    title: '文档',
    description: '查看客户端接入步骤、真实界面截图和常见问题。',
    publicDescription: '按真实界面一步步完成配置；无需登录即可查看。',
    eyebrow: '接入帮助',
    quickStart: '快速开始',
    clientAccess: '客户端接入',
    ccSwitch: 'CC Switch',
    ccSwitchHint: '三步一键接入',
    apiKeys: 'API 密钥',
    apiKeysHint: '创建或管理 Key',
    backHome: '返回首页',
    login: '登录',
    start: '开始使用',
    guide: {
      title: '用 CC Switch 一键接入 SSXZ',
      summary: '三步完成接入，不用手填地址和密钥。',
      prepareTitle: '准备：先安装 CC Switch',
      prepareBody: '只从 CC Switch 官方 GitHub 下载。Windows 选择 Windows.msi，Mac 选择 macOS.dmg；安装后先打开一次。',
      officialLink: 'CC Switch 官方 GitHub',
      warning: 'CC Switch 完全免费。请勿在仿冒网站付款。',
      step1Title: '第 1 步：在 SSXZ 点“导入到 CCS”',
      step1Body: '登录控制台，打开 API 密钥，找到要使用的 Key 并点击“导入到 CCS”。没有 Key 时请先创建一枚。',
      step2Title: '第 2 步：确认打开并导入',
      step2Body: '允许浏览器打开 CC Switch，确认供应商、API 地址、已遮盖的密钥和模型后点击“导入”。',
      step3Title: '第 3 步：切到 SSXZ，开始使用',
      step3Body: '确认 SSXZ 已成为当前供应商；如果 Claude Code 已打开，请完全关闭后重新打开。',
      finishTitle: '完成',
      finishBody: '以后可直接在 CC Switch 中切换供应商，无需再次修改配置。请勿向任何人发送完整 Key。',
      images: {
        download: 'CC Switch 官方下载页',
        window: 'CC Switch 主界面',
        importButton: 'SSXZ 的导入到 CCS 按钮',
        browserPrompt: '浏览器打开 CC Switch 提示',
        confirmation: 'CC Switch 导入确认',
        selected: 'CC Switch 中已选中的 SSXZ',
        success: 'Claude Code 通过 SSXZ 正常响应'
      }
    }
  }
}
