# Wave 1 Batch 4: Recharge and Orders Visual Design

## Scope

Reskin the customer recharge and order-history pages with the accepted F0 visual system. Preserve all payment, pricing, order, authentication, and persistence behavior.

## Behaviour boundary

- Keep checkout loading, amount presets/custom input, method selection, plan selection, fee preview, order creation, QR/popup waiting, payment recovery, and result handling unchanged.
- Keep order listing, status filter, pagination, cancellation, refund eligibility, and refund request calls unchanged.
- Do not change API modules, stores, payment-flow helpers, backend code, route definitions, prices, amount calculations, or order status mappings.
- Visual evidence uses synthetic responses and a non-payable sample QR. It must never create a real order.

## Visual system

- Reuse `AppSectionShell` and the accepted `--ssxz-*` F0 tokens.
- Surfaces use neutral background, raised surface, border, and shadow tokens. Dark surfaces must remain B-R <= 3.
- Steel blue is limited to primary, selected, focus, and active states.
- Amounts and balances are neutral. Green remains only for successful/connected states.
- Payment-provider icons retain their own brand colors, while selected containers use the F0 action treatment.
- Order status badges retain restrained semantic colors without tinting full rows.

## Responsive behaviour

- Desktop checkout uses a clear amount/method/summary hierarchy.
- Narrow screens stack controls and keep primary actions full width.
- Order tables remain horizontally scrollable rather than dropping columns.
- Dialogs use the shared neutral overlay and surface.

## Evidence

Capture recharge and orders in light, dark, and 390px mobile states; recharge method selection and safe QR waiting; order populated, empty, and detail/refund dialog states. Generate SHA256SUMS and record sampled dark RGB values.
