# SSXZ / Sub2 feature parity and overlay contract

## Latest execution status — 2026-09-21 22:43 CST

Production is now `0.2.7-ssxz.2`, source `659ea5e23c684b9bf7dd92ce6a4c0a0281644164`.
The HOLD/read-only statements below describe earlier audit checkpoints, not the
current release. See [production acceptance](2026-09-21-v027-production-acceptance.md).
Three requested model families passed paid minimal text calls on the running
release at 22:40 CST. Final rendered-page acceptance is BLOCKED by observed
Cloudflare 520/525 failures from the client (not merely browser-control timeouts).
Server-side public HTML/assets and the three real calls pass. Do not equate these
with all-client availability. Cloudflare account access was requested for read-only
origin/DNS/TLS/error investigation. No SSL/firewall protection has been disabled.

## Product decision

Use upstream functionality as the default, not a frozen SSXZ copy of each page.
New upstream controls must have a discoverable entry and working read/edit/save behavior.
Apply SSXZ branding, layout, typography and spacing to those capabilities. Do not
substitute the upstream brand, hide new capabilities to avoid layout work, or
equate a successful Git merge/version label with acceptance.

Baseline: Sub2 v0.2.7 / `aea725f2ea644d5592d0bbb1d63b607efa7e200a`.
SSXZ source base: `45e6450dff78fe79b0a1803260c81415716385d8`.

## Retained overlays

| Overlay | Reason / boundary |
| --- | --- |
| SSXZ shells, logo, design tokens, shared controls | Brand and responsive layout; must not disable upstream page behavior. |
| Enterprise group grants, low-balance preflight, billing and payment safety | Existing customer contracts and financial correctness, not optional visual customization. |
| Safe external custom pages | Never append login tokens or user IDs to third-party URLs; sanitize Markdown. |
| Managed production update/restart actions | The upstream binary updater would overwrite SSXZ builds; show the managed-release explanation, not a fake actionable button. |
| Existing connected custom providers / workbench entry | Preserve working customer flows; unfinished canvas work is not part of this merge. |
| Reseller data and settlement access | Read-only inventory confirmed active use and unsettled obligations; retain until there is an explicit settlement/transition plan. Do not delete tables or customer balances to simplify UI. |

Removed obsolete overlay: user create/edit `unlimited_concurrency` checkbox and
Sora storage quota editor. Neither field is accepted by this backend's admin user
request schema. Native concurrency editing remains (including zero where supported).
Unsupported Sora/Kiro account creation choices were also removed; legacy data and
history remain intact. Backend-supported native providers, including Grok, remain.

## Functional acceptance matrix

| Area / entry | Restored or corrected behavior | Evidence |
| --- | --- | --- |
| `/app/usage` | Canonical native UsageView inside SSXZ shell; key/model/group/type/compaction/billing filters, errors, sorting, column controls, full CSV, month summary | AppUsageView, UsageView, UsageTable, reasoning-effort integration tests; local desktop/mobile browser |
| `/app/keys` | Bulk editing, multi-group rules, sort, diagnostic/concurrency/IP columns, client import; quota reset preserves user's edited status | KeysView, UseKeyModal, BulkEditKeyModal tests; local long-name/amount browser fixture |
| Admin accounts | Full detail loading before edits, batch refresh failed selection, privacy/TLS controls, provider badges, visible priority, explicit quota/balance query | AccountsView tests including upstream priority and CN provider regressions |
| Account create/edit | Native new fields must survive serialization, including image URL conversion and OAuth/Responses controls | Account component tests and separate parity review |
| Channel/group pricing | Time pricing, tier multipliers, MiniMax/OpenCode composite platform sections, composite Messages dispatch | PricingParity.independent-20260921, TimePricingSection, pricing types tests |
| Model plaza | Effective time/peak/effort pricing and long-context explanation | PlazaModelPricingTable and restored PlazaGroupSection tests |
| Proxy administration | Expired filter, clearing credentials, bracketed IPv6 batch input | ProxiesView credentials and restored ipv6 tests |
| User administration | Native zero-concurrency edit; remove unsupported custom controls | Restored UserEditModal regression test |
| User dashboard | Per-platform consumption and quotas, while retaining SSXZ metric cards | UserDashboardStats tests |
| Custom pages | Markdown/TOC/copy and draggable external link; abort stale loads; do not leak token to iframe | CustomPageView tests including sanitization and abort; upstream external token injection intentionally excluded |
| Navigation/auth | Native payment/subscriptions, optional external shop, available channels, custom menus, audit/plugins, affiliate details, batch images, payment/OAuth callback destinations | Router feature-access, NavigationParity, PluginsView and auth tests |
| Operations | Compaction, upstream request IDs, column controls, opt-in system access logs, TTFT and root-cause error detail | Ops component and admin UsageView tests |
| Branding/layout | SSXZ shells retained; onboarding brand normalized; quota amounts no longer truncated; mobile header wraps instead of overflowing | Guide brand test and local browser at 390/1440 widths |
| Version discovery | Release notes stay visible even when managed deployment disables the binary updater; unsafe direct update/restart/rollback remain disabled | VersionBadge.managed regression tests |

Feature availability still obeys backend settings and permissions. Restoring an
entry does not authorize enabling paid payment providers, plugins, subscriptions,
or customer privileges in production.

Independent review additionally covered stale account-form relay settings, duplicate
submission during token validation/account persistence, Grok system-default routing,
and retention of the original gateway error alongside the upstream root cause.
Mobile browser inspection found the native provider selector overflowing: the
SSXZ form now wraps platform choices and account-type cards instead of clipping
Grok/Antigravity. At 390px the form measured 332px client/scroll width with no
overflowing form descendants; all native supported provider choices remain visible.

## Financial correctness fixes in this candidate

- Positive balance adjustments may reduce existing debt without forgiving it;
  negative adjustments still cannot overdraw. A fresh, local-only PostgreSQL
  cluster verified 11 scenarios, including rollback and concurrent debit safety.
  No production DSN, balance, ledger or customer account was used.
- OpenAI request preflight applies Force Fast then Free Fast consistently with
  forwarding/customer billing, and scopes OpenAI discounts to OpenAI targets.
  The existing output cap, user/group multiplier and reasoning pricing stay in place.
- **Deferred P2:** cumulative recharge tracking needs a source-aware financial
  decision. Generic balance adjustment must not classify admin/free credits as
  customer payments; this candidate deliberately does not increment that field for
  every positive adjustment. It is not a completed historical reconciliation.

## Repeatable merge gates

1. Pin the upstream tag/commit; enumerate changed routes, forms and request/response fields.
2. `pnpm run check:upstream`: no native frontend source/test file silently omitted.
   This is an inventory gate, not proof of functionality. At v0.2.7 it covers
   839 source files including 297 upstream tests.
   CI and release quality workflows fetch the exact reviewed commit and run this
   gate. Advance the pin in both workflows and the script as part of the next
   upstream review; never use a moving latest tag as acceptance evidence.
3. Preserve native regression tests. Add SSXZ route and interaction tests; verify
   load-save roundtrips so invisible fields cannot be accidentally erased.
4. Run locale completeness, typecheck, the full frontend test suite and production build.
5. Run backend unit tests and embedded-server build. Keep tests for enterprise
   grants, low balances, billing idempotency and public DTO field isolation.
6. Use local fake data for browser QA. Verify desktop/mobile layouts, long labels,
   sub-cent charges, slow responses, errors, empty state and disabled settings.
7. Before production: identify the running binary and source modifications,
   review database migrations and preserve operational configuration. Separately
   approve the release, verify page assets plus API health and keep a rollback artifact.

Do not promise zero-work future upgrades. These gates make missing features,
obsolete overlays and visual regressions explicit instead of silently accepting them.

## Production read-only observation (2026-09-21)

Service active, NRestarts=0, local health `status=ok`. Current symlink still has
the September 4 release directory name; executable was built from September 7
base revision above with `vcs.modified=true`. SHA-256:
`5f39fef2acf364e63d20096446a26da17affe34b14567836ba58c9a39e69dfae`.
The directory name does not prove the deployed version. Modified-source provenance
and migration compatibility remain separate release gates. No production mutation
or billable generation request was made during this parity work.

## Migration hazards to resolve before publication

The baseline-to-candidate diff adds 16 SQL migrations. Most add nullable/defaulted
columns, but these are not all safely reversible by switching the executable:

- 235/236 rename `groups.models_list_config` to `model_allowlist`, and the model
  list becomes request admission policy. Audit enabled production lists before
  release, especially GPT-6. Old binaries expect the old column; rollback requires
  a tested schema-compatible plan, not simply switching the symlink.
- 237/238 replace provider CHECK constraints. Verify any existing custom provider
  rows are accepted; never delete incompatible customer rows to make migration pass.
- 238 quota cleanup deletes only rows whose daily/weekly/monthly limits are all
  NULL. Capture the affected count and a recoverable backup before release; do not
  treat this as permission to remove any nonzero limits or balances.
- The upstream request-ID index is concurrent/nontransactional; it must remain
  on the migration runner's nontransactional path.

These migrations were inspected, not run on production in this task.

### Confirmed release blockers, read-only snapshot 2026-09-21 17:52–17:55 CST

1. **Model admission:** 8 enabled legacy lists do not match `gpt-6` or
   `gpt-6-astra`; 6 belong to active groups (5 of the 9 active OpenAI groups).
   Renaming alone would introduce new denials. One GPT-6 usage record in the last
   30 days belongs to an unrestricted group; this does not validate every route.
   Do not blindly enable all models or disable group authorization. Preserve the
   prior admission boundary while explicitly migrating listing-only configuration.
2. **Quota history:** migration 238 would delete 1,352 unlimited rows for 247 users.
   Three rows still contain nonzero usage/window state. Preserve a recoverable
   snapshot and prove the intended treatment on an isolated database before release.
   Provider CHECK changes have no incompatible current rows in the four inspected tables.
3. **Schema rollback:** production still has the old column and target migrations
   have not run. Rollback must account for column names AND migration checksums;
   binary-only rollback is not safe.
4. **Reseller obligations:** 2 active roles, 234 direct invitation links, 17 invited
   users active in 30 days, 1,704 commission entries. Outstanding commission is
   28.21963759 (16.94372531 available, 11.27591228 frozen). Frozen entries have
   passed their unlock time, but were not changed by this audit. Zero withdrawal
   requests does not authorize removing balances, relationships or settlement access.
5. **Binary provenance:** the running binary is byte-identical to local
   `F:/CodexWork/ssxz-purchase-fix-0907prod/backend/sub2api-purchase-fix-0907-csp`
   and the preserved server CSP artifact. Its revision matches the source baseline,
   but `vcs.modified=true` and untracked embedded assets still prevent a claim of
   complete source reproducibility. September 21 local pricing edits are not proof
   of what was embedded in the September 7 executable.

AGENTS.md is absent from the active tree and user workspace on disk; the user-supplied
AGENTS instructions in this task apply. Security review used the repository launch
checklist and unified-workspace security strategy. This is scoped upgrade review,
not a claim of exhaustive penetration testing or live financial reconciliation.

### Proposed migration decision (not executed)

Prefer native semantics over a permanent second legacy authorization system:
archive the old listing-only JSON and retain existing API-key/account/group access
checks; do not silently promote listing preferences into new request restrictions.
For proven legacy-only rows, an explicitly reviewed conversion can leave the new
allowlist disabled until the administrator deliberately configures it. This would
also adopt native model discovery instead of preserving a stale display list.
It is not permission to disable a genuine pre-existing allowlist, add wildcard
authorization to customer keys, or alter account routing. Provenance and exact
affected groups must be checked immediately before a separately approved release.

The alternative is to adopt native restrictive allowlists immediately, after an
explicit per-group approved model list. Merely appending `gpt-6` would not preserve
other previously callable models. Neither alternative has been applied to production.

## Local acceptance, September 21

| Gate | Actual result |
| --- | --- |
| Frontend full suite | 308 files, 2,424 tests passed (`qa-frontend-accepted.log`) |
| Frontend lint | Exit 0, no reported lint errors (`qa-frontend-lint-accepted.log`) |
| Frontend typecheck and production build | `pnpm run build` passed, including locale completeness and `vue-tsc -b` (`qa-frontend-build-accepted.log`) |
| Backend full unit/contract selection | `go test -p 1 -tags unit -json ./...`: 20,151 passing test/subtest events, 16 skipped, zero failed tests/packages (`qa-backend-final.log`) |
| Embedded server build | Linux/amd64, CGO disabled, `go build -p 1 -tags embed ./cmd/server` passed; compiled front-end assets included (`qa-backend-build-final.log`) |
| Real balance SQL | Local PostgreSQL, 11 scenarios passed, no skipped scenarios (`qa-backend-balance-postgres.log`) |
| Upstream file inventory | All 839 frontend sources / 297 upstream test files present |
| Workflow syntax | CI and release YAML parse successfully; remote CI was not triggered |
| Browser | Local fake data only: 390px/1440px usage, keys and admin header; account modal platforms/cards on 390px; release notes visible with managed update actions still disabled |
| Compiled-web preview | Served `backend/internal/web/dist` through `tools/qa/sub2-ui-server.mjs --production`; verified hashed `/assets/index-*.js`, translated labels, group names, `$0.003840`, 2-minute timing, mobile usage/keys with no document overflow, SSXZ admin branding and 56px desktop header |

Skipped backend cases require external credentials/network, Redis/PostgreSQL audit
fixtures, pending OAuth identity storage, or a plugin runtime. Docker-backed full
integration and isolated staging release journeys were **not** run. The local
PostgreSQL balance test uses its own minimal fresh schema, not a production clone.
Some unrelated admin fixture requests deliberately return 404; the local browser
check does not certify every admin feature or payment-provider flow end to end.

Non-blocking build notes: existing chunks over 500kB, stale Browserslist data, and
a Node child-process deprecation warning. Whole-merge whitespace checking reports
11 trailing spaces in the unmodified native `instructions_gpt6_astra.txt`; its Git
blob equals upstream (`0dcbce2fddd604dca9bb5361b0cc92980c96ed4b`). All other changed
files pass `git diff --check`. Do not alter a vendor prompt merely to remove
whitespace without a reason and separate review.

**Historical pre-release checkpoint: HOLD.** Local acceptance is not production acceptance. Complete
the legacy-list migration decision, quota-history preservation and schema rollback
rehearsal, isolated staging gates and separate release approval before any rollout.
No production files, accounts, configuration, database or service were modified;
no paid generation requests were made. Unfinished canvas work and canceled recurring
monitoring remain out of scope.
