# Wave 1 Usage And Pricing Implementation Plan

> **For Codex:** Execute continuously with focused tests and spec/code review gates.

**Goal:** Bring usage records and model pricing onto the accepted F0 visual system without changing backend or billing behavior.

**Architecture:** Keep the current workbench routes and API clients. Compose existing controls and DTOs inside the current views; style them with shared F0 tokens. Preserve provider identity through existing LobeHub model icon support.

**Tech Stack:** Vue 3, TypeScript, Vitest, F0 CSS tokens, existing API clients.

---

### Task 1: Usage workbench

- Modify `frontend/src/views/user/AppUsageView.vue`.
- Extend `frontend/src/views/user/__tests__/AppUsageView.spec.ts` first for real filter params, pagination, reset, safe export, and preserved workbench states.
- Add only needed locale strings in `frontend/src/i18n/locales/zh.ts` and `frontend/src/i18n/locales/en.ts`.
- Run focused tests.

### Task 2: Model pricing

- Modify `frontend/src/views/user/AvailableChannelsView.vue`, `frontend/src/components/channels/AvailableChannelsTable.vue`, and `frontend/src/components/channels/SupportedModelChip.vue`.
- Extend `frontend/src/views/user/__tests__/AvailableChannelsView.spec.ts` and add a focused table test if needed.
- Keep provider colors and pricing values unchanged; use `ModelIcon` for model identities.
- Run focused tests.

### Task 3: Verification and visual evidence

- Run `pnpm typecheck`, `pnpm lint:check`, focused Vitest suites, and `pnpm build`.
- Capture required mock-data screenshots, calculate B-R samples and SHA-256 sums, and sync evidence to the collaboration workspace.
- Review route/API/lifecycle/disabled-state behavior and inspect the final diff for forbidden scope.

### Task 4: Commit and deploy

- Commit only the planned files.
- Build Linux release with `-tags embed`.
- Create root-only binary/config/database snapshot and rollback script.
- Confirm `rust.sidecar.responses_ws_enabled=false` with no true environment override.
- Deploy staging, smoke, deploy production, external smoke; rollback immediately on any failure.
- Prepend the full receipt to `CODEX_TO_CLAUDE.md`.
