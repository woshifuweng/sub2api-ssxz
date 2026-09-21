# Sub2 v0.2.7 / SSXZ production acceptance

Status update 2026-09-21 22:43 CST: **recovery code LIVE; client access incident NOT resolved**.

### Current authoritative deployment

- `.2` active from 22:38:45 CST, source `659ea5e23c684b9bf7dd92ce6a4c0a0281644164`.
- SHA256 `e49fdddc57efca68432c904492809dccca07fd1373357e239022f54548ac15ac`.
- Embedded frontend `index-D2XpKt6e.js`; temporary preview maps and entry rewriting
  retired with backups. Immutable old assets retained for already-open sessions.
- Public origin-host checks: health/home/admin/model/usage/keys=200, all six
  entry scripts/styles=200 and exact SHA256. Service active, restarts0, Redis
  cleanup errors0, fatal/panic0. Isolated acceptance unit stopped.
- Minimal real calls on `.2`, owner test keys only: GPT-6 Astra at 22:40:15,
  Fable5.1 at 22:40:23, Opus5 at 22:40:26; all200/exactOK/correct returned model.
  Ledger charges `.00341950 + .00227500 + .00011700 = .00581150` USD.
  Across all nine successful checks in this release session: `.01604080` USD.
- Recovery dir `/opt/sub2api/backups/v027-catalog-patch-20260921-v4`.
  Preview cleanup backups `/opt/sub2api/backups/v027-preview-retirement-20260921`.

### Still blocking final acceptance

- Browser saw Cloudflare520 at 14:30:56 UTC, Ray `a3e9c2f67c91ddbc` (Hong Kong).
- After `.2`, browser saw Cloudflare525 at 14:40:05 UTC, Ray
  `a3e9d1997a7928ee` (Hong Kong). Screenshot `build/acceptance-20260921/cloudflare-525.png`.
- Origin certificate valid Sep16–Dec15, SAN matches api.ssxzapi.com,443 listening,
  CF allowlist DROP counter0 and provider firewall chain empty. No matching TLS
  failure in default-level nginx log; origin-side TLS/public calls succeed.
  These facts narrow the issue but do NOT prove whether Cloudflare DNS/origin
  selection, a route, hosting network, or another TLS issue caused it.
- Requested login to the managing Cloudflare account for read-only origin/DNS/TLS
  and error analytics. Do not disable encryption/firewall, repeatedly roll back
  application builds, or assert this is just the user's browser.
- No final successful desktop/mobile screenshots yet. Full task NOT complete.

### Guarded packaging failure and fix

Attempt v3 started setup mode because the new directory lacked existing config/data
symlinks; health returned404, traffic stayed behind503 maintenance. Auto-rollback
restored prior runtime22:37:22, without database rewind or setup submission.
The installer now attaches verified existing runtime paths; the release checks
these before any maintenance. With corrected links, v4 succeeded22:38:45.

### Earlier incident notes (historical)

The deployment described below was rolled back at 22:09 CST to initial v0.2.7
`ac2923c5e` (SHA256 `29df07a4cd03482a3058ebd485211e6f6ba9945f9f1b0bda5b3185181d689794`).
Database and newly verified model configuration were not rewound.
Version `.1` is NOT the current runtime. Its tests below are historical evidence.

Recovery candidate `.2`, source `659ea5e23c684b9bf7dd92ce6a4c0a0281644164`,
SHA256 `e49fdddc57efca68432c904492809dccca07fd1373357e239022f54548ac15ac`,
embeds the catalog/Redis fixes plus payment-SDK startup isolation, startup error
fallback, removal of automatic chunk-failure reload loops, and a persistent
catalog/rate-load error. Frontend regressions 12 tests plus locale 3 passed;
lint/typecheck/build and startup-payment isolation guard passed. Server acceptance
and final live browser verification of this candidate are still pending.

Observed incident: module fetch failed for AvailableChannelsView while the exact
asset existed and matched public SHA256. Failures more than 10 seconds apart
triggered repeated automatic reloads. Later the preview rendered successfully.
At 22:33 client home GET=200, health GET timed out; simultaneous origin TLS and
server-to-public edge GETs=200. This does not prove a unique network root cause
or all-client recovery. Browser-control timeouts are not a website success signal.

## Historical `.1` acceptance, before rollback

Status at 2026-09-21 22:05 CST: production published; browser acceptance pending.
Do not report the whole task complete until the final rendered-page check passes.

## Deployed identity

- Upstream: v0.2.7, `aea725f2ea644d5592d0bbb1d63b607efa7e200a`.
- Runtime version: `0.2.7-ssxz.1`; source `d5147b2b2f9633a64896b101ebfa53148940104f`.
- Linux/amd64 embedded frontend, clean source build (`vcs.modified=false`).
- SHA256: `fc0a8a7b903ab90cfd69199dabf7d2f31ac30dbe7702175e890f46f3640df80d`.
- Active since September 21 21:57:44 CST, no automatic restarts at verification.
- Recovery directory: `/opt/sub2api/backups/v027-catalog-patch-20260921-v2`.
- Fresh pre-patch database dump SHA256:
  `c2d54ce62561d55311fe825ad0ab011b461d80b894bc8cf80c047b6753ad88b1`.
- Same-schema rollback targets the previous v0.2.7 release, **not** v0.1.183.
  Never rewind the database after customer activity.

## Product changes

1. Keep SSXZ branding and shared UI while exposing upstream usage/key/account/
   pricing/admin features. The detailed parity matrix remains in the companion plan.
2. The customer model page now projects authorized groups, configured accounts,
   native channel restrictions and the real pricing resolver. Groups without a
   channel are no longer omitted. This read-only view never widens key grants.
3. Preserve unpriced supported rows with an explicit pending-price label, rather
   than hiding them or claiming they are free. Use group overrides and native
   price schedules; show Max-reasoning multipliers and image resolution pricing.
4. Fix image price units: per-image/per-request pricing no longer becomes an
   absurd per-million-token number. Image-specific multipliers are preserved.
5. Add verified GPT-6/Astra pricing and its true alias, and Fable 5.1 pricing plus
   support on the two accounts whose provider catalog actually advertises it.
   Preserve existing customer rates and model restrictions. The mistakenly added
   GPT-5.1 price row was explicitly removed after the user's clarification.
6. Fix upstream concurrency cleanup's Redis 6.2-only query for this Redis 6.0 host.
   The regression reproduced before the fix; final production no longer reports
   the repeated cleanup error.

## Real calls on final release

| Requested model | Result | Actual charge USD |
| --- | --- | ---: |
| gpt-6-astra | 200, completed, exact OK, returned same model | 0.00341950 |
| claude-fable-5-1 | 200, end_turn, exact OK, returned same model | 0.00227500 |
| claude-opus-5 | 200, end_turn, exact OK, returned same model | 0.00011700 |

Final-build calls total $0.00581150; earlier three successful release checks total
$0.00441780; combined $0.01022930. Existing administrator test keys were used.
No customer keys were broadened and no paid image generation was performed by
this verification. Concurrent usage records from other work are not our tests.

## Evidence and boundaries

- 26 provider model catalogs answered 200; 70 distinct advertised model names.
  The timestamped inventory distinguishes disabled accounts, missing account
  mappings, channel restrictions and configured routes. It does not assert every
  advertised model can be billed or generated successfully.
- Public `/v1/models` through existing test keys lists GPT-6/Astra and both new
  Claude names. Native discovery can list more than channel policy permits;
  listing is not a successful-inference certificate.
- Isolated server patch acceptance passed health/home/admin/static/two logins
  and both authenticated catalog responses, with no credential fields returned.
  Separate DB/Redis, no provider egress, production runtime unchanged during QA.
- Final health, homepage, admin, model, usage and key HTML returned 200 and the
  new entry asset name. Public script/style checks passed. These are HTTP/static
  checks, not substitutes for rendered-page screenshots.
- Relevant backend catalog/billing/permission/Redis regressions and frontend
  component/view/locale tests passed; frontend typecheck/build and lint passed.
  Earlier whole-merge full-suite results are separately documented; they were
  not repeated and should not be presented as a fresh final full-suite run.
- The dedicated isolated acceptance unit was stopped after successful release.
- Browser: old live SSXZ usage/model pages were inspected. Final refresh produced
  an empty observed page followed by repeated browser-control timeouts, including
  on a new tab. Browser reconnect requested; **cause and final rendering are not
  yet verified**. Do not claim this is definitely only a browser fault.

## Release procedure findings

The first patch attempt stopped safely before restarting production because
requests did not drain. Investigation showed `sites-enabled` contains ordinary
files, not links to `sites-available`; editing the latter did not close ingress.
The corrected run backed up the actual enabled files, derived a temporary gate
without replacing unrelated configuration, required real HTTPS 503 responses
from all three virtual hosts, drained existing requests, saved a fresh DB dump,
then switched and restored the exact enabled files. Maintenance is now cleared.
This finding also limits earlier claims that an initial release maintenance flag
alone proved a closed-write window; rely on recorded invariants and backups, not
the flag's presence.

Transient isolated test units also disappear after stop (`--collect`). The patch
acceptance launcher now recreates the same isolation properties and uses exact
binary hash guards for the explicitly verified resume path.

## Deliberately unchanged

No customer balances, grants or multipliers changed; no settlement obligations
deleted; no bulk activation of newly advertised/unpriced models. Unfinished
canvas work and canceled recurring monitoring are outside this release.
