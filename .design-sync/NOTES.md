# design-sync notes (herd)

- herd's UI is **Vue 3**; Claude Design renders React only. So this is a **tokens-only** sync: herd's look (themes, base classes, fonts), no components. Decided with the user on 2026-11-14.
- `.design-sync/theme/` is a stand-in package for the converter: an empty `index.js`/`index.d.ts` (no exports, so the converter goes tokens-only) and `build.mjs`, which writes `herd.css` from `frontend/src/themes/` (`fleet-dark.css`, `flatwhite.css`, `base.css`). `herd.css` is generated and gitignored: run `buildCmd` before every build.
- `build.mjs` changes two things for design pages: Fleet Dark also applies on `:root` (a page without `data-theme` still gets colours), and `html`/`body` scroll again (herd's window sets `overflow: hidden`).
- Fonts: `extraFonts` points at the fontsource `latin-400/500/600.css` files of Geist and Geist Mono in `frontend/node_modules` (run `npm ci` in `frontend/` first). Only the latin subset ships.
- Converter deps live in `.ds-sync/node_modules` (esbuild, ts-morph, @types/react, react, react-dom, playwright); pass it as `--node-modules`. Chromium for the render check is in `~/.cache/ms-playwright`.
- Build: `node .design-sync/theme/build.mjs`, then `node .ds-sync/resync.mjs --config .design-sync/config.json --node-modules .ds-sync/node_modules --entry .design-sync/theme/index.js --out ./ds-bundle [--remote .design-sync/.cache/remote-sync.json]`.

## Known build warns

- `[DTS_REACT] @types/react not found`: harmless, there are no components to type.
- `[ZERO_MATCH] no component exports — treating as tokens-only DS`: expected.

## Re-sync risks

- `conventions.md` names tokens and classes from `frontend/src/themes/`; renaming one there makes the header lie. Re-check it against `ds-bundle/_ds_bundle.css` on every sync.
- `build.mjs` finds Fleet Dark by its exact selector `[data-theme='fleet-dark'] {`; it throws if that changes.
- Component-scoped styles in `.vue` files (e.g. `.chip`, the todo list) are not synced: only the global theme and `base.css`.
