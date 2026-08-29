# Usage Table Column Alignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Align the user usage table's numeric headers and values on the same right edge without changing data, column order, or interactions.

**Architecture:** Keep the existing semantic table and responsive layout. Add a focused regression assertion for computed alignment, then resolve the scoped CSS specificity conflict so `th.num-cell` and `td.num-cell` share one explicit rule.

**Tech Stack:** Vue 3, TypeScript, scoped CSS, Vitest, Vue Test Utils, Vite.

---

### Task 1: Reproduce the alignment regression

**Files:**
- Modify: `frontend/src/views/user/__tests__/AppUsageView.spec.ts`

- [ ] **Step 1: Add a computed-alignment assertion**

After mounting the page and loading rows, inspect the `Usage`, `Duration`, and `Fee` header/data pairs and require both members of each pair to compute to `text-align: right`.

```ts
const headers = wrapper.findAll('thead th.num-cell')
const firstRowValues = wrapper.get('tr.usage-row').findAll('td.num-cell')

expect(headers).toHaveLength(3)
expect(firstRowValues).toHaveLength(3)
headers.forEach((header, index) => {
  expect(getComputedStyle(header.element).textAlign).toBe('right')
  expect(getComputedStyle(firstRowValues[index].element).textAlign).toBe('right')
})
```

- [ ] **Step 2: Run the focused test and verify the regression is reproduced**

Run: `pnpm test:run -- src/views/user/__tests__/AppUsageView.spec.ts`

Expected: FAIL because numeric data cells compute to `left` while numeric headers compute to `right`.

### Task 2: Apply the minimal alignment fix

**Files:**
- Modify: `frontend/src/views/user/AppUsageView.vue`

- [ ] **Step 1: Make the numeric selector equally explicit for headers and data**

```css
.usage-table th.num-cell,
.usage-table td.num-cell {
  text-align: right;
}
```

Keep existing padding, responsive widths, row expansion, and detail layout unchanged.

- [ ] **Step 2: Run the focused test**

Run: `pnpm test:run -- src/views/user/__tests__/AppUsageView.spec.ts`

Expected: both usage-detail tests and the new alignment test pass.

### Task 3: Verify and save the correction

**Files:**
- Modify only if verification identifies a directly related defect: `frontend/src/views/user/AppUsageView.vue`, `frontend/src/views/user/__tests__/AppUsageView.spec.ts`
- Update: `design-qa.md`

- [ ] **Step 1: Run static and automated checks**

Run `pnpm typecheck`, focused ESLint, `pnpm test:run`, and `pnpm build` from `frontend`.

Expected: all commands exit with code 0.

- [ ] **Step 2: Render the real component and visually verify it**

Capture the default and expanded table at desktop width. Confirm the right edges of each numeric header/data pair match, opening details does not shift columns, and the browser console contains no errors.

- [ ] **Step 3: Commit only the scoped source and test changes**

Stage `AppUsageView.vue` and `AppUsageView.spec.ts`; do not stage QA screenshots, build output, the existing backend binary, or unrelated files.

Use commit message: `fix(usage): align numeric table columns`.
