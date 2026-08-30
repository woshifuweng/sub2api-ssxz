export default {
  docs: {
    title: 'Documentation',
    description: 'Follow client setup steps, screenshots, and troubleshooting guidance.',
    publicDescription: 'Follow the real interface step by step; no sign-in is required.',
    eyebrow: 'Integration help',
    quickStart: 'Quick start',
    clientAccess: 'Client setup',
    ccSwitch: 'CC Switch',
    ccSwitchHint: 'Connect in three steps',
    apiKeys: 'API keys',
    apiKeysHint: 'Create or manage a key',
    backHome: 'Back home',
    login: 'Sign in',
    start: 'Get started',
    guide: {
      title: 'Connect SSXZ with CC Switch',
      summary: 'Connect in three steps without manually entering an endpoint or key.',
      prepareTitle: 'Before you start: install CC Switch',
      prepareBody: 'Download only from the official CC Switch GitHub. Choose Windows.msi on Windows or macOS.dmg on Mac, then open the app once.',
      officialLink: 'Official CC Switch GitHub',
      warning: 'CC Switch is free. Do not pay a website pretending to be official.',
      step1Title: 'Step 1: Select “Import to CCS” in SSXZ',
      step1Body: 'Sign in, open API keys, find the key you want, and select “Import to CCS”. Create a key first if the list is empty.',
      step2Title: 'Step 2: Confirm the import',
      step2Body: 'Allow the browser to open CC Switch. Verify the provider, endpoint, masked key, and model, then import.',
      step3Title: 'Step 3: Switch to SSXZ and start',
      step3Body: 'Make sure SSXZ is the active provider. Fully restart Claude Code if it was already open.',
      finishTitle: 'Done',
      finishBody: 'You can switch providers in CC Switch without editing configuration again. Never share a complete API key.',
      images: {
        download: 'Official CC Switch download page',
        window: 'CC Switch main window',
        importButton: 'Import to CCS in SSXZ',
        browserPrompt: 'Browser prompt to open CC Switch',
        confirmation: 'CC Switch import confirmation',
        selected: 'SSXZ selected in CC Switch',
        success: 'Claude Code responding through SSXZ'
      }
    }
  }
}
