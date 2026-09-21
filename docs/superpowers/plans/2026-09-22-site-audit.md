# Whole-site audit and verified repairs

User authorized finding and repairing known and overlooked defects. Preserve SSXZ UI,
upstream features, customer balances, group grants, prices, and inspection-owned accounts.

## Scope and gates

- [x] Read-only public ingress, runtime, disk, Redis and nginx checks.
- [x] Baseline frontend lint/typecheck: pass; 312 suites / 2464 tests pass.
- [x] Baseline backend: first unconstrained Windows Go compiler crashed; constrained
  `GOMAXPROCS=2 go test -p 1 ./...` passes. This is not a production crash.
- [x] Authenticated route/render audit. HTTP 200 alone is not acceptance of a SPA route.
- [x] Restore error-table details: replace unavailable `el-tooltip` with native titled
  spans. Test real rendered elements without a fake tooltip stub. Keep row/user actions.
- [x] HelpTooltip: fixed coordinates must use boundingClientRect without window scroll.
  Add nonzero scroll regression first; keep existing hover/click behavior.
- [x] Catalog/pricing audit: distinguish advertised models from routable, priced models.
  Never invent image prices or enable Luna to make a catalog test pass.
- [x] Repeat targeted/frontend regression and build; no weakening tests.
- [x] Guarded release only with immutable artifact, rollback and runtime checks.
- [x] Record exact remaining coverage limitations and residual risks.

## Initial evidence

Production `0.2.7-ssxz.3`, service active, NRestarts=0; Redis PONG, nginx active.
Root disk 67%; no systemd priority error entries in prior two hours (does not prove
absence of application JSON errors). Dashboard: one initial connection failure,
then ten consecutive HTTP200; health five HTTP200. Cause of initial failure unknown.
On authenticated admin usage error tab, 95 literal `el-tooltip` elements render.
The package is absent and tests stubbed it, hiding the runtime defect.

The release preflight also exposed two deployment-tooling defects: the candidate
log was root-owned before the service-user process started, and the isolated
environment did not provide a usable migration-table permission set. The release
script now preserves the candidate log for diagnostics and accepts an explicit
preflight config path. The final candidate check used the existing production
database/config with the candidate listening only on port 18081 and background
jobs disabled; the normal service was then switched atomically with health-gated
rollback protection.

## Repair design

Use native `title` on existing content wrappers, not a new UI dependency or global
component shim. This restores hidden detail text without changing routing, money,
layout theme or introducing another tooltip package. For existing HelpTooltip,
remove only the duplicated scroll offset; test hover and scroll repositioning.

## Verified residuals

- The current live catalog exposes GPT-6/Astra, Claude Fable 5.1, Claude Opus 5,
  and image model entries. GPT Image 2.5 has a server-side display fallback, but
  it is not being silently assigned a customer price or enabled in the default
  workbench profile; paid image smoke tests were intentionally not run.
- Build warnings remain for stale Browserslist data, one dynamic/static import,
  and large chunks. They are maintainability/performance follow-ups, not a
  release blocker for this repair.
- No customer balance, group, account, model price, Luna, or inspection-owned
  upstream account was changed by this audit.

## Final live evidence

- Production is serving `0.2.7-ssxz.4` from
  `/opt/sub2api/releases/0.2.7-ssxz.4.20260922.audit`; the deployed binary
  matches SHA-256 `1f6a7a7dc59d8265184f425c33dff2a3d8e741757f6d418199deccec7431c974`.
- `sub2api` and nginx are active, Redis returns `PONG`, `NRestarts=0`, health is
  `{"status":"ok"}`, and no systemd error-level entries were emitted after the
  release window.
- Public routes and authenticated browser routes for dashboard, usage, API keys,
  affiliate, profile, admin dashboard, admin usage, admin keys, groups, and
  accounts rendered with a real `<main>` and no horizontal overflow at 1280px.
  Usage records visibly include group and locator-code fields; the admin error
  table has zero literal `<el-tooltip>` tags.
- The image workbench primary domain, workers.dev fallback, and API health endpoint
  all returned HTTP 200. No paid image-generation smoke request was issued.
