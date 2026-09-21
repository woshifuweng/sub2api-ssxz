// Local-only visual QA fixture. Never contacts production or generates content.
import { fileURLToPath, pathToFileURL } from 'node:url'
import { resolve } from 'node:path'
const root = fileURLToPath(new URL('../../frontend/', import.meta.url))
process.chdir(root)
const { createServer, preview } = await import(pathToFileURL(resolve(root, 'node_modules/vite/dist/node/index.js')).href)
const vue = (await import(pathToFileURL(resolve(root, 'node_modules/@vitejs/plugin-vue/dist/index.mjs')).href)).default
const group = { id: 23, name: '企业 GPT 综合分组 · 七折 · GPT-6', platform: 'openai', status: 'active', rate_multiplier: .7, subscription_type: 'standard', is_exclusive: false }
const user = { id: 999, username: '本地验收账号', email: 'qa@example.invalid', role: 'admin', status: 'active', balance: 5, concurrency: 2, allowed_groups: [23] }
const settings = { site_name: 'SSXZ AI', registration_enabled: false, subscription_enabled: true, allow_user_view_error_requests: true, affiliate_enabled: false, turnstile_enabled: false, email_verify_enabled: false, custom_menu_items: [], site_logo: '', docs_url: '' }
const key = { id: 7, name: '企业生产用密钥 · 长名称验收', key: 'sk-local-fixture-not-valid', status: 'active', user_id: 999, group_id: 23, group_ids: [23], group, groups: [group], quota: 100, quota_used: 4.17, created_at: '2026-09-21T08:00:00Z', expires_at: null }
const records = Array.from({ length: 6 }, (_, i) => ({ id: i + 1, user_id: 999, api_key_id: 7, api_key: key, group_id: 23, group, model: 'gpt-6-astra', reasoning_effort: 'high', request_id: `local-qa-request-${i + 1}`, inbound_endpoint: '/v1/responses', ip_address: '', input_tokens: 15341, output_tokens: 8220, cache_creation_tokens: 0, cache_read_tokens: 9500, input_cost: .03, output_cost: .07, cache_read_cost: .001, cache_creation_cost: 0, total_cost: .144286, actual_cost: i ? .101 : .00384, rate_multiplier: .7, billing_type: 0, billing_mode: 'token', request_type: 'stream', stream: true, duration_ms: 119999, first_token_ms: 1300, created_at: '2026-09-21T08:00:00Z' }))
const stats = { total_requests: 6, total_input_tokens: 92046, total_output_tokens: 49320, total_tokens: 141366, total_cache_tokens: 57000, total_cache_creation_tokens: 0, total_cache_read_tokens: 57000, total_actual_cost: .50884, total_cost: .73, average_duration_ms: 119999, endpoints: [{ endpoint: '/v1/responses', requests: 6, total_tokens: 141366, actual_cost: .50884, cost: .73 }] }
const paged = (items) => ({ items, total: items.length, pages: 1, page: 1, page_size: 20 })
const adminStats = { ...stats, total_users: 12, today_new_users: 1, active_users: 2, hourly_active_users: 1, stats_updated_at: '2026-09-21T08:00:00Z', stats_stale: false, total_api_keys: 16, active_api_keys: 14, total_accounts: 4, normal_accounts: 4, error_accounts: 0, ratelimit_accounts: 0, overload_accounts: 0, today_requests: 6, today_input_tokens: 92046, today_output_tokens: 49320, today_cache_creation_tokens: 0, today_cache_read_tokens: 57000, today_tokens: 141366, today_cost: .73, today_actual_cost: .50884, uptime: 3600, rpm: 0, tpm: 0 }
const responses = {
  '/settings/public': settings, '/auth/me': user, '/user/groups': [group], '/groups/available': [group],
  '/groups': [group], '/keys': paged([key]), '/keys/group-rates': {}, '/subscriptions/active': [], '/subscriptions': [],
  '/usage': paged(records), '/usage/stats': stats, '/usage/dashboard/stats': stats,
  '/usage/dashboard/models': { models: [{ model: 'gpt-6-astra', requests: 6, total_tokens: 141366, cost: .73, actual_cost: .50884 }] },
  '/usage/dashboard/snapshot-v2': { trend: [{ date: '2026-09-21', requests: 6, input_tokens: 92046, output_tokens: 49320, total_tokens: 141366, actual_cost: .50884 }], groups: [{ group_id: 23, group_name: group.name, requests: 6, total_tokens: 141366, actual_cost: .50884, cost: .73 }] },
  '/usage/errors': paged([{ id: 9, created_at: '2026-09-21T08:00:00Z', key_name: key.name, model: 'gpt-6-astra', endpoint: '/v1/responses', status_code: 429, category: 'rate_limit', message: '本地验收：限流提示，未发起真实请求', request_id: 'local-qa-error', group_name: group.name }]),
  '/usage/errors/9': { id: 9, status_code: 429, category: 'rate_limit', message: '本地模拟限流', request_id: 'local-qa-error' },
  '/user/reseller/role': { role: 'none', reseller_role: 'none', is_agent: false, is_manager: false },
  '/groups/rates': {},
  '/announcements': paged([]), '/announcements/unread': [], '/user/announcements': [],
  '/admin/dashboard/stats': stats, '/admin/groups': paged([group]), '/admin/accounts': paged([]),
  '/admin/dashboard/snapshot-v2': { stats: adminStats, trend: [], models: [], groups: [] },
  '/admin/dashboard/operations-summary': { start_date: '2026-09-01', end_date: '2026-09-21', new_customers: 1, customer_actual_cost: .50884, invitee_recharge_amount: 0, rebate_pending: 0, rebate_available: 0, rebate_transferred: 0, active_customers: 1, active_api_keys: 1, top_customers: [] },
  '/admin/dashboard/users-trend': { trend: [] },
  '/admin/dashboard/users-ranking': { ranking: [], total_actual_cost: .50884, total_requests: 6, total_tokens: 141366 },
  '/admin/settings': settings,
  '/admin/system/version': { version: '0.2.7-ssxz', build_type: 'source' },
  '/admin/system/check-updates': { current_version: '0.2.7-ssxz', latest_version: '0.2.7', has_update: false, build_type: 'source', release_info: { html_url: 'https://github.com/Wei-Shaw/sub2api/releases/tag/v0.2.7', tag_name: 'v0.2.7', name: 'Sub2 v0.2.7' } },
}
const fixturePlugin = {
  name: 'local-only-fixture-api', configureServer(vite) {
    vite.middlewares.use((req, res, next) => {
      const url = new URL(req.url, 'http://127.0.0.1')
      if (!url.pathname.startsWith('/api/') && !url.pathname.startsWith('/setup/')) return next()
      res.setHeader('Content-Type', 'application/json')
      res.setHeader('Cache-Control', 'no-store')
      const path = url.pathname.replace('/api/v1', '')
      let data = responses[path]
      if (path === '/auth/login') data = { access_token: 'local-fixture-only', expires_in: 86400, user }
      if (path === '/setup/status') data = { needs_setup: false }
      if (path === '/usage/dashboard/api-keys-usage') data = { stats: { 7: { total_actual_cost: 4.17, today_actual_cost: .50884 } } }
      if (data === undefined) { console.log('UNMOCKED', req.method, path); res.statusCode = 404; res.end(JSON.stringify({ code: 404, message: 'Unmocked local QA request' })); return }
      if (req.method !== 'GET' && !['/auth/login', '/usage/dashboard/api-keys-usage'].includes(path)) { res.statusCode = 405; res.end(JSON.stringify({ code: 405, message: 'QA fixture is read only' })); return }
      res.end(JSON.stringify({ code: 0, data }))
    })
  }
}
fixturePlugin.configurePreviewServer = fixturePlugin.configureServer
const config = { root, configFile: false, resolve: { alias: { '@': resolve(root, 'src'), 'vue-i18n': 'vue-i18n/dist/vue-i18n.esm-bundler.js' } }, define: { __INTLIFY_JIT_COMPILATION__: true }, plugins: [vue(), fixturePlugin], build: { outDir: resolve(root, '../backend/internal/web/dist') }, server: { host: '127.0.0.1', port: 4317, strictPort: true }, preview: { host: '127.0.0.1', port: 4317, strictPort: true } }
const production = process.argv.includes('--production')
const server = production ? await preview(config) : await createServer(config)
if (!production) await server.listen()
console.log(`Local-only SSXZ ${production ? 'compiled' : 'development'} UI fixture: http://127.0.0.1:4317/app/usage`)
