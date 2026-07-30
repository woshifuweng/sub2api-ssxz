# Toggle Visual Consistency Design

## Goal

Separate theme switching from business boolean controls:

- The application header uses a compact, dedicated moon/sun theme selector.
- Business settings use one visually consistent switch with an unambiguous on/off state.
- Existing values, events, API payloads, and save behavior remain unchanged.

## Components

### ThemeToggle

`frontend/src/components/common/ThemeToggle.vue` remains responsible only for light/dark mode.
It uses the compact 64x32 two-icon control: moon and sun are both visible, with the selected
side highlighted. It keeps the current keyboard, ARIA, document-class, and persistence behavior.

### Toggle

`frontend/src/components/common/Toggle.vue` is the canonical business switch. It keeps the
existing `v-model` contract and adds disabled-state handling. The control is 44x24, uses a
neutral track when off and the existing primary token when on, and provides position plus
icon contrast so state is understandable without relying on color alone.

## Migration Scope

1. All existing `<Toggle>` consumers inherit the new business-switch appearance.
2. Payment `ToggleSwitch.vue` delegates its button rendering to the common `Toggle`.
3. The four legacy `.toggle-slider` controls in `SettingsView.vue` become common `Toggle`
   instances.
4. Hand-written 44x24 switches in account modals, group configuration, quota controls, and
   payment-plan editing become common `Toggle` instances.
5. Theme controls outside the authenticated application are not changed in this batch.

## Non-Goals

- No settings field, API request, default value, permission check, or save flow changes.
- No new UI dependency.
- No color family outside existing neutral and primary tokens.
- No conversion of menu items, segmented controls, or ordinary action buttons.

## Verification

- Component tests lock down ARIA, click, keyboard, disabled, and theme persistence behavior.
- Static scans verify no legacy `.toggle-slider` or known hand-written switch signature remains.
- `pnpm typecheck`, targeted Vitest tests, and `pnpm build` must pass.

