# Wave 1 Usage And Pricing Design

## Scope

Reskin `/app/usage` and `/app/available-channels` with the existing F0 foundation. Preserve their routes, API contracts, billing values, authentication, and user-visible behavior.

## Usage page

- Keep the current balance, monthly actual cost, billing explanation, monthly trend, detailed billing/performance/support-code columns, and genuine failure/empty states.
- Restore the existing real controls from the legacy usage view in the workbench page: API-key filter, model filter, date range, pagination, refresh, reset, and safe CSV export.
- Scope API calls only with parameters accepted by the current user usage endpoint. The user endpoint does not accept a group filter, so this batch must not add a fake group selector or broaden the backend contract.
- Use F0 cards, controls, table frame, buttons, and neutral chart treatment. Green remains status-only.

## Model pricing page

- Keep channel/model/group data, search, refresh, exact pricing values, pricing popovers, and genuine disabled/empty states.
- Replace legacy page/table surfaces with F0 cards, controls, and table treatment.
- Keep provider identity color and render model identity through the existing LobeHub-backed `ModelIcon`; no pricing or filtering semantics change.

## Verification

- Extend focused component tests for filters, pagination/export, empty/error behavior, search, and model identity rendering.
- Run frontend typecheck, lint check, focused tests, and production build.
- Capture synthetic-data dark/light/mobile, data/empty, and open-control states under `artifacts/ui-refactor/usage-pricing/`, with SHA-256 manifest and neutral-surface samples.
- Deploy through snapshot, staging smoke, production smoke, and rollback-on-failure gates.
