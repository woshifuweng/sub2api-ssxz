# Settings and Startup Recovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复首屏加载失败恢复、设置页 404/误导性加载状态，并提升设置页与公共提示控件的可读性和响应式表现。

**Architecture:** 保留现有 Vue 页面与公共组件，只在 API wrapper、启动恢复层、SettingsView、Toggle、Toast 五个边界做小范围修改。设置页将“必需设置”和“可选设置”分开处理；错误恢复保持手动可控，不自动刷新或改动业务数据。

**Tech Stack:** Vue 3、TypeScript、Axios、Vue Test Utils、Vitest、Tailwind CSS、Vite。

---

### Task 1: 锁定 TLS 指纹 API 契约

**Files:**
- Modify: `frontend/src/api/admin/settings.ts`
- Test: `frontend/src/api/__tests__/settings.tlsFingerprint.spec.ts`

- [ ] **Step 1: 写失败测试，断言所有 TLS 指纹请求使用根级资源。**

```ts
it('uses the backend tls-fingerprint-profiles contract for list and mutations', async () => {
  await settingsAPI.getTLSFingerprintSettings()
  await settingsAPI.updateTLSFingerprintSettings({ enabled: true })
  await settingsAPI.createTLSFingerprintProfile({
    profile_id: 'chrome', name: 'Chrome', enabled: true,
    enable_grease: true, cipher_suites: [], curves: [], point_formats: [],
  })
  await settingsAPI.updateTLSFingerprintProfile('chrome', {
    name: 'Chrome 2', enabled: true, enable_grease: true,
    cipher_suites: [], curves: [], point_formats: [],
  })
  await settingsAPI.deleteTLSFingerprintProfile('chrome')
  expect(get).toHaveBeenCalledWith('/admin/tls-fingerprint-profiles')
  expect(put).toHaveBeenCalledWith('/admin/tls-fingerprint-profiles', { enabled: true })
  expect(post).toHaveBeenCalledWith('/admin/tls-fingerprint-profiles', expect.any(Object))
  expect(put).toHaveBeenCalledWith('/admin/tls-fingerprint-profiles/chrome', expect.any(Object))
  expect(del).toHaveBeenCalledWith('/admin/tls-fingerprint-profiles/chrome')
})
```

- [ ] **Step 2: 运行测试确认当前路径断言失败。**

Run: `pnpm --dir frontend exec vitest run src/api/__tests__/settings.tlsFingerprint.spec.ts`

Expected: FAIL because the current wrapper sends `/admin/settings/tls-fingerprint`.

- [ ] **Step 3: 把五个 wrapper 路径改为 `/admin/tls-fingerprint-profiles`，保留返回类型和调用方不变。**

- [ ] **Step 4: 运行测试确认通过。**

Run: `pnpm --dir frontend exec vitest run src/api/__tests__/settings.tlsFingerprint.spec.ts src/views/admin/__tests__/SettingsView.spec.ts`

Expected: PASS.

- [ ] **Step 5: 提交接口契约修复。**

```bash
git add frontend/src/api/admin/settings.ts frontend/src/api/__tests__/settings.tlsFingerprint.spec.ts
git commit -m "fix(frontend): align tls fingerprint settings API"
```

### Task 2: 让首屏失败状态可读且不循环刷新

**Files:**
- Modify: `frontend/index.html`
- Modify: `frontend/src/utils/pageLoadRecovery.ts`
- Test: `frontend/src/utils/__tests__/pageLoadRecovery.spec.ts`

- [ ] **Step 1: 写测试，覆盖启动占位、脚本错误和重复调用。**

```ts
it('renders a branded retry state without exposing raw exception text', () => {
  document.body.innerHTML = '<div id="app"><section id="ssxz-startup"><p id="ssxz-startup-status">Loading</p></section></div>'
  showPageLoadFailure({ stage: '资源加载', retryHref: '/login' })
  expect(document.querySelector('#ssxz-startup')?.textContent).toContain('页面暂时没有加载完成')
  expect(document.querySelector('#ssxz-page-load-retry')?.textContent).toContain('重新加载')
  expect(document.body.textContent).not.toContain('AxiosError')
})
```

- [ ] **Step 2: 运行测试确认新断言失败。**

Run: `pnpm --dir frontend exec vitest run src/utils/__tests__/pageLoadRecovery.spec.ts`

Expected: FAIL because the helper only changes one plain text node today.

- [ ] **Step 3: 把内联首屏占位改成品牌化、响应式的结构，并让恢复 helper 接收失败阶段。**

实现约束：`showPageLoadFailure` 只更新现有占位或创建一个错误卡片；只提供一次手动重试；不调用 `location.reload()`，不写入业务数据；脚本加载错误只显示用户可理解的状态，不展示原始异常。

- [ ] **Step 4: 运行页面恢复测试和前端类型检查。**

Run: `pnpm --dir frontend exec vitest run src/utils/__tests__/pageLoadRecovery.spec.ts && pnpm --dir frontend typecheck`

Expected: PASS.

- [ ] **Step 5: 提交首屏恢复改动。**

```bash
git add frontend/index.html frontend/src/utils/pageLoadRecovery.ts frontend/src/utils/__tests__/pageLoadRecovery.spec.ts
git commit -m "fix(web): improve startup failure recovery state"
```

### Task 3: 防止设置页失败时显示假数据或误保存

**Files:**
- Modify: `frontend/src/views/admin/SettingsView.vue`
- Test: `frontend/src/views/admin/__tests__/SettingsView.spec.ts`

- [ ] **Step 1: 写测试，断言必需设置失败会阻止保存，可选设置失败只显示分区状态。**

```ts
it('does not allow saving when the required site settings request fails', async () => {
  getSettings.mockRejectedValue({ status: 503, message: 'temporarily unavailable' })
  const wrapper = mountSettingsView()
  await flushPromises()
  expect(wrapper.get('[data-testid="settings-load-error"]').text()).toContain('站点设置暂时无法加载')
  expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeDefined()
})
```

- [ ] **Step 2: 运行测试确认当前页面没有必需加载失败状态。**

Run: `pnpm --dir frontend exec vitest run src/views/admin/__tests__/SettingsView.spec.ts`

Expected: FAIL because `loading` is turned off in `finally` and no load error state is rendered.

- [ ] **Step 3: 引入最小加载状态模型。**

使用 `requiredSettingsLoadState: 'loading' | 'ready' | 'error'` 和 `optionalLoadErrors: Set<string>`；`loadSettings` 在成功时才进入 `ready`，失败时进入 `error`；模板在 error 状态显示重试按钮，form 的 submit 与保存按钮在非 ready 状态禁用。

- [ ] **Step 4: 把并行可选请求包装为带分区名称的失败处理。**

每个 `loadXxx` 保留原有默认 fail-safe 行为，但失败时记录对应分区，不清空已有真实值；页面显示“该分区暂时不可用，点击重试”，不弹出原始 Axios 文本。

- [ ] **Step 5: 运行设置页完整测试。**

Run: `pnpm --dir frontend exec vitest run src/views/admin/__tests__/SettingsView.spec.ts`

Expected: PASS with the existing 38 tests plus the new load-state assertions.

- [ ] **Step 6: 提交设置页安全加载改动。**

```bash
git add frontend/src/views/admin/SettingsView.vue frontend/src/views/admin/__tests__/SettingsView.spec.ts
git commit -m "fix(admin): make settings loading states explicit"
```

### Task 4: 提升公共开关、Toast 和设置页布局

**Files:**
- Modify: `frontend/src/components/common/Toggle.vue`
- Modify: `frontend/src/components/common/Toast.vue`
- Modify: `frontend/src/views/admin/SettingsView.vue`
- Test: `frontend/src/components/common/__tests__/Toggle.spec.ts`

- [ ] **Step 1: 写 Toggle 可访问性测试。**

```ts
it('exposes an accessible name and visible state when requested', async () => {
  const wrapper = mount(Toggle, { props: { modelValue: false, label: '启用设置', showState: true } })
  expect(wrapper.get('[role="switch"]').attributes('aria-label')).toBe('启用设置')
  expect(wrapper.text()).toContain('关')
  await wrapper.get('[role="switch"]').trigger('click')
  expect(wrapper.text()).toContain('开')
})
```

- [ ] **Step 2: 增加可选 `label`、`descriptionId`、`showState` props，扩大焦点区域，保持默认调用兼容。**

关闭态使用更清晰的深灰轨道和白色滑块；开启态保留蓝色；不改变 v-model 事件名。

- [ ] **Step 3: 调整 Toast 的窄屏宽度、标题和错误操作区。**

Toast 使用 `width: min(100% - 2rem, 28rem)`，错误提示优先显示用户可理解的上下文；重试动作通过可选 `action` 传入，不把请求 URL、Token 或后端堆栈展示给用户。

- [ ] **Step 4: 调整 SettingsView 的容器和 tab 响应式规则。**

桌面内容区从 `max-w-4xl` 扩展到 `max-w-6xl`；表单标题与 Toggle 使用 `min-w-0`、`flex-wrap`；移动端切换为纵向布局；tab 继续可滚动但不让正文产生横向溢出。

- [ ] **Step 5: 运行组件、设置页和类型检查。**

Run: `pnpm --dir frontend exec vitest run src/components/common/__tests__/Toggle.spec.ts src/views/admin/__tests__/SettingsView.spec.ts && pnpm --dir frontend typecheck`

Expected: PASS.

- [ ] **Step 6: 提交公共 UI 改动。**

```bash
git add frontend/src/components/common/Toggle.vue frontend/src/components/common/Toast.vue frontend/src/views/admin/SettingsView.vue frontend/src/components/common/__tests__/Toggle.spec.ts
git commit -m "fix(ui): improve settings controls and error feedback"
```

### Task 5: 构建、静态资源检查与发布前验收

**Files:**
- Modify: `backend/internal/web/dist/` generated by the frontend build only
- Test: frontend build and relevant Vitest suites

- [ ] **Step 1: 运行完整前端质量门禁。**

Run: `pnpm --dir frontend lint:check && pnpm --dir frontend test:run && pnpm --dir frontend build`

Expected: lint, tests, typecheck inside build, Vite build, and entry isolation check all PASS.

- [ ] **Step 2: 校验生成产物没有旧 TLS 路径。**

Run: `rg -n --fixed-strings "/admin/settings/tls-fingerprint" backend/internal/web/dist frontend/src`

Expected: no API wrapper occurrence; only migration notes/tests may mention the old path if explicitly documented.

- [ ] **Step 3: 校验每个入口和动态 chunk 均存在。**

Run: `Get-ChildItem backend/internal/web/dist/assets -File | Measure-Object` and verify the generated `index.html` references existing files.

Expected: every referenced JS/CSS file exists; no stale asset name remains.

- [ ] **Step 4: 运行 `git diff --check` 与 Vue review 清单。**

Run: `git diff --check` and inspect disabled save buttons, route guards, async lifecycle, and error handling.

Expected: no whitespace errors, no secrets/logging, no fake success state, no disabled control calling an API.

- [ ] **Step 5: 只在本地构建和静态检查全部通过后，生成发布包并做正式站只读验收。**

验收内容：首页 200、API health 200、设置页资源 200、无 TLS 404、设置页可读、首屏失败卡片可重试。发布前不修改业务数据；若正式发布需要用户明确授权，则停在构建产物等待授权。

- [ ] **Step 6: 提交构建产物和验收记录。**

```bash
git add backend/internal/web/dist docs/superpowers/specs/2026-09-22-settings-startup-recovery.md docs/superpowers/plans/2026-09-22-settings-startup-recovery.md
git commit -m "docs: record settings and startup recovery acceptance"
```
