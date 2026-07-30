# Toggle Visual Consistency Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give every business boolean setting one consistent switch while restoring the compact moon/sun header theme selector.

**Architecture:** Keep theme selection and business booleans as separate components. Upgrade the existing common `Toggle.vue`, route legacy wrappers and hand-written switches through it, and leave all model values and event handlers intact.

**Tech Stack:** Vue 3, TypeScript, Tailwind CSS, Lucide Vue, Vitest, Vue Test Utils.

---

### Task 1: Lock Down Common Toggle Behavior

**Files:**
- Create: `frontend/src/components/common/__tests__/Toggle.spec.ts`
- Modify: `frontend/src/components/common/Toggle.vue`

- [ ] **Step 1: Add failing tests**

Test that clicking emits the inverted model value, `aria-checked` mirrors the value, and a disabled switch does not emit.

- [ ] **Step 2: Run the focused test**

Run: `pnpm test:run src/components/common/__tests__/Toggle.spec.ts`

Expected: FAIL until disabled behavior and the new DOM contract exist.

- [ ] **Step 3: Implement the business switch**

Keep `modelValue` and `update:modelValue`; add optional `disabled` and `ariaLabel` props. Render one native button with `role="switch"`, a stable 44x24 track, existing primary/neutral tokens, and a non-color-only state indicator.

- [ ] **Step 4: Re-run the focused test**

Run: `pnpm test:run src/components/common/__tests__/Toggle.spec.ts`

Expected: PASS.

### Task 2: Restore the Dedicated Header Theme Selector

**Files:**
- Modify: `frontend/src/components/common/ThemeToggle.vue`
- Modify: `frontend/src/components/common/__tests__/ThemeToggle.spec.ts`

- [ ] **Step 1: Extend tests**

Assert that the control exposes both moon and sun icons while preserving click and keyboard theme switching.

- [ ] **Step 2: Replace only the visual layer**

Use the compact 64x32 dual-icon selector. Preserve the existing document `dark` class, safe-storage key, translated label, keyboard handling, and ARIA state.

- [ ] **Step 3: Run tests**

Run: `pnpm test:run src/components/common/__tests__/ThemeToggle.spec.ts`

Expected: PASS.

### Task 3: Consolidate Existing Switch Implementations

**Files:**
- Modify: `frontend/src/components/payment/ToggleSwitch.vue`
- Modify: `frontend/src/views/admin/SettingsView.vue`
- Modify: `frontend/src/components/account/BulkEditAccountModal.vue`
- Modify: `frontend/src/components/account/CreateAccountModal.vue`
- Modify: `frontend/src/components/account/EditAccountModal.vue`
- Modify: `frontend/src/components/account/QuotaLimitCard.vue`
- Modify: `frontend/src/views/admin/GroupsView.vue`
- Modify: `frontend/src/views/admin/orders/PlanEditDialog.vue`

- [ ] **Step 1: Delegate payment toggles**

Keep `checked`, `label`, and `toggle` as the public API, but render `Toggle` internally.

- [ ] **Step 2: Replace Settings legacy controls**

Replace the four checkbox/`.toggle-slider` blocks with `Toggle v-model` bindings to the same fields.

- [ ] **Step 3: Replace known hand-written switches**

For every known `relative inline-flex h-6 w-11` switch, replace its button/knob block with `Toggle`, binding the same boolean and preserving disabled and ARIA labels.

- [ ] **Step 4: Verify migration completeness**

Run:

```powershell
rg -n "toggle-slider|relative inline-flex h-6 w-11" frontend/src --glob "*.vue"
```

Expected: no legacy business-switch implementations remain except the canonical `Toggle.vue` signature itself.

### Task 4: Full Verification

**Files:**
- No additional files.

- [ ] **Step 1: Run focused tests**

Run:

```powershell
pnpm test:run src/components/common/__tests__/Toggle.spec.ts src/components/common/__tests__/ThemeToggle.spec.ts
```

Expected: PASS.

- [ ] **Step 2: Typecheck**

Run: `pnpm typecheck`

Expected: PASS or only explicitly documented pre-existing failures unrelated to modified files.

- [ ] **Step 3: Production build**

Run: `pnpm build`

Expected: PASS.

- [ ] **Step 4: Review diff scope**

Confirm no backend file, API payload, settings key, or save handler changed.

