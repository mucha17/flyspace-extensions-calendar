# Template extension

The base scaffold for a FlySpace extension, emitted by `flyspace-ext init`.

## Layout

- `flyspace-extension.json` — the manifest (Ed25519-signed at publish time).
- `federation.config.js` — Native Federation config; exposes `./routes` and the four component
  artifacts.
- `src/main-view/` — the routed surface mounted under `/apps/<id>` (MainView + child routes).
- `src/thread-renderer/` — renders a thread on the dashboard.
- `src/dashboard-widget/` — a dashboard widget.
- `src/settings-panel/` — optional custom settings UI (omit it to rely on the auto-generated form).
- `src/config.user.schema.json` — the per-user config schema.
- `src/i18n/{en,pl}.json` — bilingual translations (`en` is the required fallback).

## Build

```bash
pnpm install
pnpm build       # ng build + Native Federation + integrity hash
pnpm lint        # design-token lint
flyspace-ext validate
flyspace-ext sign
```

`ng serve` runs a standalone dev playground (`src/main.ts`) that mounts the extension's own routes
with a stub SDK, so the extension can be developed without the full shell.
