# Codex Workflow

This is the project-specific Codex workflow for the SSXZ/Sub2API AI relay and unified `/app` workspace. It is intentionally lightweight: it adds guidance and a read-only preflight script, but it does not install hooks, background jobs, or auto-updaters.

## Project Compass

This project is not a DeepSeek-only demo, a single image page, or a greenfield chat site. It is a multi-provider, multi-model, multi-capability AI relay built on the existing sub2/sub2api foundation.

Keep these product boundaries in mind:

- `/app` is the unified workspace.
- Models must come from backend channel/account/group/model availability, not frontend guesses.
- Real image models must derive from real channel `supported_models`; gate env vars are filters, not catalog sources.
- Fake models are allowed only for explicit staging/test validation and must be marked fake/test-only.
- Billing, usage, ledger, provider routing, account selection, and pricing are existing sub2 capabilities. Reuse them instead of rebuilding them casually.
- Image generation and image editing are core product capabilities, not side features.

## Before Editing

Start with discovery before changing files:

1. Check `git status --short --branch`.
2. Identify the current branch, target PR, and whether unrelated changes already exist.
3. Read the relevant source files and tests.
4. Read `AGENTS.md` plus any relevant docs under `docs/security/` or `docs/operations/`.
5. State the intended file scope before editing when the task is risky or cross-cutting.

For model/provider/image work, explicitly identify:

- provider
- model
- capability
- channel/account/group source
- pricing source
- gate/allowlist/cap behavior
- audit/usage/ledger impact

## During Editing

Prefer small PRs with one behavioral goal. Keep the diff close to the requested scope.

Do not:

- hardcode a real provider as the long-term architecture
- treat env allowlists as model catalog sources
- expose provider keys, base URLs, Authorization headers, cookies, or secrets to the browser
- bypass backend permission, billing, usage, or provider/account/channel checks
- make production behavior depend on fake/test models
- add independent image pages when the task belongs in `/app`
- rewrite billing, ledger, payment, provider routing, or quota systems unless explicitly requested

When adding a helper, include tests that prove it is provider-agnostic or clearly test-only.

## Verification

Run the smallest meaningful checks first, then broaden when the change crosses package boundaries.

Common backend checks:

```powershell
cd backend
go test ./internal/service ./internal/handler ./internal/server/routes ./cmd/server
```

Common frontend checks:

```powershell
cd frontend
pnpm run typecheck
pnpm run build
```

Useful repository check:

```powershell
git diff --check
```

Use `scripts/codex-preflight.ps1` when you want a read-only context snapshot:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\codex-preflight.ps1
```

## Handoff

Final reports should include:

- branch and HEAD
- files changed
- what changed
- tests run and results
- security-sensitive areas touched or explicitly untouched
- remaining blockers or risk
- whether provider calls, deployments, billing/ledger/payment, Nginx, or production were touched

For reviews, lead with findings and file/line references. Keep summaries short.

