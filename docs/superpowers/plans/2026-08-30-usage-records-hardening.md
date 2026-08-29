# Usage Records Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make usage records precise, race-safe, export-consistent, and readable without horizontal swiping on a 390px phone.

**Architecture:** Keep the existing `AppUsageView.vue` data flow and one-row expansion model. Add pure formatting helpers, request/export snapshots, and a CSS-only mobile card presentation over the same semantic rows; extend the existing component test suite instead of creating parallel components.

**Tech Stack:** Vue 3 `<script setup>`, TypeScript, Vitest, Vue Test Utils, scoped CSS, Vite.

---

### Task 1: Lock formatting edge cases with failing tests

**Files:**
- Modify: `frontend/src/views/user/__tests__/AppUsageView.spec.ts`

- [x] **Step 1: Add a sub-cent fee and minute-rounding fixture**

Add a third mocked usage row with `actual_cost: 0.00384` and `duration_ms: 119999`. Keep the existing zero-cost row so both zero and non-zero sub-cent behavior are covered.

- [x] **Step 2: Add visible-value assertions**

Assert that the three primary rows include `$0.28`, `$0.00`, `$0.00384`, and `2m`, and that no row contains `1m 60s`.

- [x] **Step 3: Run the focused test and verify failure**

Run:

```bash
pnpm test:run -- src/views/user/__tests__/AppUsageView.spec.ts
```

Expected: the new precision or minute-boundary assertion fails against the current implementation.

### Task 2: Make formatting precise and race-safe

**Files:**
- Modify: `frontend/src/views/user/AppUsageView.vue`
- Modify: `frontend/src/views/user/__tests__/AppUsageView.spec.ts`

- [x] **Step 1: Implement adaptive sub-cent display**

Import `formatCurrencyExact` beside the existing currency helpers and update the local formatter:

```ts
function formatCost(value: number | null | undefined) {
  const amount = Number(value || 0)
  return amount !== 0 && Math.abs(amount) < 0.01
    ? formatCurrencyExact(amount)
    : formatMoney(amount)
}
```

- [x] **Step 2: Correct duration rounding**

For minute-level values, round the total seconds first:

```ts
const roundedTotalSeconds = Math.round(ms / 1000)
const minutes = Math.floor(roundedTotalSeconds / 60)
const seconds = roundedTotalSeconds % 60
```

- [x] **Step 3: Snapshot each page load and ignore stale responses**

Add a module-local counter, increment it at load start, copy page/filter values into local constants, use those constants for detail API calls, and return before mutating refs when the completed request is no longer latest. Keep the monthly summary on its fixed month-to-today range. Only the latest request may clear `loading`. Keep balance refresh separate and deduplicate the in-flight refresh so filter/page changes do not cause unrelated account refreshes.

- [x] **Step 4: Add a deferred-response regression test**

Start two loads with different model filters, resolve the second request before the first, and assert the later filter's rows remain visible after both promises settle.

- [x] **Step 5: Run focused tests**

Run:

```bash
pnpm test:run -- src/views/user/__tests__/AppUsageView.spec.ts
```

Expected: all focused tests pass.

### Task 3: Freeze export inputs and disable conflicting controls

**Files:**
- Modify: `frontend/src/views/user/AppUsageView.vue`
- Modify: `frontend/src/views/user/__tests__/AppUsageView.spec.ts`

- [x] **Step 1: Add one shared busy state**

Create `const controlsDisabled = computed(() => loading.value || exporting.value)` and bind it to the API Key select, model/date inputs, reset, refresh, and pagination buttons. Keep the export button's existing zero-row guard.

- [x] **Step 2: Freeze export values**

At export start, copy `filters.value` and `totalRows.value`. Use only the copied values for every page request and the file name.

- [x] **Step 3: Make download cleanup browser-safe**

Append the generated anchor to `document.body`, click it, remove it, and schedule `URL.revokeObjectURL(url)` with `window.setTimeout(..., 0)`.

- [x] **Step 4: Test filter snapshot and disabled controls**

Hold the first export request open, mutate the visible filters, then resolve it and verify all subsequent export calls still use the original filter snapshot. Assert the filter controls and paging buttons are disabled while export is active.

- [x] **Step 5: Run focused tests**

Run:

```bash
pnpm test:run -- src/views/user/__tests__/AppUsageView.spec.ts
```

Expected: all focused tests pass with no fake timers or skipped assertions.

### Task 4: Replace mobile horizontal scrolling with cards

**Files:**
- Modify: `frontend/src/views/user/AppUsageView.vue`
- Modify: `frontend/src/views/user/__tests__/AppUsageView.spec.ts`

- [x] **Step 1: Add localized data labels to cells**

Bind `data-label` for time, model, usage, duration and fee cells using the existing i18n keys. Keep the chevron button and detail row unchanged.

- [x] **Step 2: Add the 640px mobile card rules**

At `max-width: 640px`, remove the table minimum width, hide the header, display table sections as blocks, render each `.usage-row` as a two-column card, expose labels through `td::before`, keep time/model full-width, and render the detail row as a matching card extension. Ensure `.usage-table-wrap` and `.usage-table` have `max-width: 100%` and no horizontal overflow.

- [x] **Step 3: Add markup contract assertions**

Assert all five data cells expose the expected labels and the stylesheet contains the 640px mobile layout contract.

- [x] **Step 4: Run focused tests**

Run:

```bash
pnpm test:run -- src/views/user/__tests__/AppUsageView.spec.ts
```

Expected: all focused tests pass.

### Task 5: Full verification, visual QA, and commit

**Files:**
- Modify: `design-qa.md` as untracked local QA evidence only
- Do not stage: `backend/sub2api_linux`, screenshots, build output, or unrelated files

- [x] **Step 1: Run static and automated gates**

Run:

```bash
pnpm typecheck
pnpm exec eslint src/views/user/AppUsageView.vue src/views/user/__tests__/AppUsageView.spec.ts
pnpm test:run
pnpm build
git diff --check
```

Expected: all commands pass; existing non-fatal chunk-size warnings may remain.

- [x] **Step 2: Run real-component visual QA**

Render the actual component with representative rows at 1657px, 1024px and 390px. Check default and expanded states, sub-cent fee, `2m` duration, disabled export state, one-open-row behavior, document/table overflow and browser console errors.

- [x] **Step 3: Record QA evidence**

Update `design-qa.md` with source/implementation screenshots, viewport dimensions, measured overflow, interaction results, comparison history and exact final line `final result: passed`.

- [x] **Step 4: Commit only source and tests**

Commit source and tests in small reviewable commits; do not stage QA artifacts, build output, the backend binary, or unrelated files.

```bash
git add frontend/src/views/user/AppUsageView.vue frontend/src/views/user/__tests__/AppUsageView.spec.ts
git commit -m "fix(usage): harden records interactions"
```

- [x] **Step 5: Verify final state**

Run `git status --short`, `git log -3 --oneline`, and `git show --stat HEAD`. Confirm no server, browser or test process remains running and that production was not changed.
