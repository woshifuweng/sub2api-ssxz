# Wave 1 Batch 4 Implementation Plan

1. Lock existing payment and order API contracts in focused tests.
2. Move checkout containers, amount controls, method controls, status panels, and order views onto F0 tokens without modifying business script logic.
3. Run focused frontend tests, typecheck, lint, clean build, and relevant backend tests.
4. Capture synthetic-data visual states and verify neutral dark surfaces.
5. Commit only scoped files, build an embedded Linux binary, snapshot production, deploy to staging, smoke test, deploy to production, and repeat smoke tests.
6. Sync evidence to the Claude workspace and prepend the deployment receipt to `CODEX_TO_CLAUDE.md`.
