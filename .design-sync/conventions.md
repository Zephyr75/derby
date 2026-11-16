# herd: look and feel

herd is a dense, Zed-like desktop app (13px UI, thin borders, small radii). This design system ships **no components**: build with plain React and HTML, styled with herd's tokens and its small class set. Everything comes with `styles.css`.

## Setup

- **Theme:** set `data-theme` on `<html>`: `fleet-dark` (dark, the default, also used with no attribute) or `flatwhite` (light).
- **Fonts:** Geist for the UI (`var(--ui-font)`, weights 400/500/600), Geist Mono for code, ids and keys (`var(--mono)` or class `mono`). Base size `var(--ui-size)` (13px).
- **Never hard-code a colour:** use the `var(--*)` tokens below so both themes work.

## Tokens

| Use | Tokens |
| --- | --- |
| Surfaces | `--bg` (page), `--chrome` (title and status bars), `--surface` (inputs, buttons), `--elev` (popups), `--hover`, `--sel` (selected row), `--overlay` (dimmed backdrop), `--shadow` (box-shadow value) |
| Text | `--text`, `--muted`, `--faint`, `--accent` (links, focus) |
| Lines | `--border` |
| Buttons and toggles | `--button`, `--button-text`, `--toggle-sel`, `--toggle-text` |
| Status: text, plus `-dot` and `-bg` variants | `--running` (blue), `--waiting` (yellow), `--approval` (red: failures), `--idle` (grey), `--done` (green) |
| Other | `--local`, `--local-bg`, `--local-border` (purple tag); `--add`, `--add-bg`, `--del`, `--del-bg` (diffs); `--urgent`; `--decision`; `--code-kw` |

## Classes

| Class | What it is |
| --- | --- |
| `btn`, `btn primary`, `btn ghost`, `btn danger` | 26px buttons |
| `field` | a label stacked over its control |
| `input` (also on `select`), `area` (textarea) | form controls |
| `seg`, `seg on` | segmented choice buttons |
| `pill` with `<button>` children, `on` on the active one | Zed-style toggle group |
| `badge s-running` / `s-waiting` / `s-failed` / `s-idle` / `s-done` / `s-local`, with a `dot` span inside | status badge |
| `kbd` | keyboard key |
| `popup` | elevated panel (popovers, dialogs) |
| `scrim`, `scrim dim` | full-screen backdrop behind a popup |
| `local-chip`, `tool-chip`, `tool-chip denied`, `issue-ref` | small mono chips |
| `sis-0` to `sis-5` | sets `--sis` (a linked-task colour); `issue-ref sis` uses it |
| `muted`, `mono`, `sr-only` | utilities |

Layout glue (flex, gaps, padding) is yours: inline styles or your own classes, with spacing in px (4, 6, 8, 10, 12) and radii of 4 to 8px.

## Where the truth lives

`_ds_bundle.css` (imported by `styles.css`) is herd's whole stylesheet: the `[data-theme=…]` token blocks, then the classes above. Read it before styling.

## Example

```jsx
<div className="popup" style={{ width: 360, padding: 12, display: 'flex', flexDirection: 'column', gap: 10 }}>
  <label className="field">
    Task title
    <input className="input" defaultValue="Retry unlock commands" />
  </label>
  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
    <span className="badge s-running"><span className="dot" />Running</span>
    <span className="kbd">Alt+T</span>
    <span style={{ flex: 1 }} />
    <button className="btn ghost">Cancel</button>
    <button className="btn primary">Create</button>
  </div>
</div>
```
