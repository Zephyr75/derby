// Builds derby.css, the stylesheet design-sync uploads to Claude Design:
// derby's own theme and base CSS (frontend/src/themes), with two changes
// for design pages. Run from the repo root: node .design-sync/theme/build.mjs
import { readFileSync, writeFileSync } from 'node:fs';

const themes = 'frontend/src/themes/';
const read = (f) => readFileSync(themes + f, 'utf8');

// Fleet Dark is the default theme: a page without data-theme still gets colours.
const dark = read('fleet-dark.css').replace("[data-theme='fleet-dark'] {", ":root,\n[data-theme='fleet-dark'] {");
if (!dark.includes(':root,')) throw new Error('fleet-dark.css: theme selector not found');

// derby's window never scrolls (its panels do); a design page may.
const pages = `
/* design-sync: pages scroll, unlike derby's window */
html,
body {
  height: auto;
  min-height: 100%;
  overflow: auto;
}
`;

writeFileSync('.design-sync/theme/derby.css', [dark, read('flatwhite.css'), read('base.css'), pages].join('\n'));
