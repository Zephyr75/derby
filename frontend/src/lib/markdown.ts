// Artifact preview: markdown-it, with ```mermaid fences rendered by mermaid
// and diagram pages (![title](PLAN.html), drawn with archify) embedded.

import MarkdownIt from 'markdown-it'

const md = new MarkdownIt({ html: false, linkify: true, typographer: false })

const defaultFence = md.renderer.rules.fence!
md.renderer.rules.fence = (tokens, idx, options, env, self) => {
  const tok = tokens[idx]
  if (tok.info.trim().split(/\s+/)[0] === 'mermaid') {
    return `<figure class="mermaid-fig"><pre class="mermaid">${md.utils.escapeHtml(tok.content)}</pre><figcaption>mermaid · ${md.utils.escapeHtml(firstLine(tok.content))}</figcaption></figure>`
  }
  return defaultFence(tokens, idx, options, env, self)
}

// A diagram page in the run directory: embedded, sandboxed (no access to the
// app), with a button to open it full size.
const defaultImage = md.renderer.rules.image!
md.renderer.rules.image = (tokens, idx, options, env, self) => {
  const tok = tokens[idx]
  const file = String(tok.attrGet('src') ?? '')
  const e = env as RenderEnv
  if (!e?.task || !/^[\w.-]+\.html$/.test(file) || file.startsWith('.')) return defaultImage(tokens, idx, options, env, self)
  const title = md.utils.escapeHtml(tok.content || file)
  const f = md.utils.escapeHtml(file)
  return (
    `<figure class="diagram-fig"><iframe src="${md.utils.escapeHtml(diagramURL(e.task, file, !!e.dark, true) + '&v=' + (e.rev ?? 0))}" title="${title}" sandbox="allow-scripts" loading="lazy"></iframe>` +
    `<figcaption><span>${title}</span><button type="button" data-diagram="${f}" title="Open ${f} full size">Open full size</button></figcaption></figure>`
  )
}

/** diagramURL is where jockey serves a task's diagram page to the webview. */
export function diagramURL(task: string, file: string, dark: boolean, embed = false) {
  return `/run/${encodeURIComponent(task)}/${encodeURIComponent(file)}?theme=${dark ? 'dark' : 'light'}${embed ? '&embed=1' : ''}`
}

export interface RenderEnv {
  task?: string
  dark?: boolean
  rev?: number // bumped when a diagram page changes, to reload it
  [k: string]: unknown
}

function firstLine(s: string) {
  return s.trim().split('\n')[0] ?? ''
}

// Links open outside the app.
md.renderer.rules.link_open = (tokens, idx, options, _env, self) => {
  tokens[idx].attrSet('target', '_blank')
  tokens[idx].attrSet('rel', 'noreferrer')
  return self.renderToken(tokens, idx, options)
}

export function render(src: string, env: RenderEnv = {}): string {
  return md.render(src, { ...env })
}

// A report's blocks: paragraphs, lists and fenced code split on blank lines,
// each heading a block of its own.
function blocks(src: string): string[] {
  const out: string[] = []
  let cur: string[] = []
  let fence = ''
  const flush = () => {
    if (cur.length) out.push(cur.join('\n'))
    cur = []
  }
  for (const l of src.replace(/\r/g, '').split('\n')) {
    const f = /^\s*(`{3,}|~{3,})/.exec(l)?.[1]
    if (fence) {
      cur.push(l)
      if (f && f[0] === fence[0] && f.length >= fence.length) fence = ''
    } else if (f) {
      fence = f
      cur.push(l)
    } else if (/^#{1,6}\s/.test(l)) {
      flush()
      out.push(l)
    } else if (!l.trim()) flush()
    else cur.push(l)
  }
  flush()
  return out
}

const sumName = (s: string) => (/^tl;?\s?dr$/i.test(s) ? 'TLDR' : /^eli5$/i.test(s) ? 'ELI5' : '')
const sumLead = /^(?:\*\*|__)\s*(tl;?\s?dr|eli5)\s*[:.]?\s*(?:\*\*|__)\s*[:.]?\s*|^(tl;?\s?dr|eli5)\s*:\s*/i

const heading = (b: string) => /^#{1,6}\s/.test(b)
const led = (b: string) => {
  const m = sumLead.exec(b)
  return m ? sumName(m[1] ?? m[2]) : ''
}
// a block worth showing on its own: no heading, code, rule or bare HTML comment
const prose = (b: string) => !heading(b) && !/^\s*(`{3,}|~{3,})/.test(b) && !/^\s*([-*_])(\s*\1){2,}\s*$/.test(b) && !/^\s*<!--[\s\S]*-->\s*$/.test(b)

/**
 * summary is a report's TLDR, ELI5 and, for a plan, Options, as the Writing prompt asks every
 * artifact to open with: a heading (## TLDR) with its blocks up to the next
 * heading, or a paragraph led by a bold label (**TLDR**: …), with the next
 * paragraph when the label stands alone. Without a TLDR, the report's first
 * block stands in, unlabelled.
 */
export function summary(src: string): { label: string; md: string }[] {
  const bs = blocks(src)
  const found = new Map<string, string>()
  const used = new Set<number>()
  for (let i = 0; i < bs.length; i++) {
    const h = /^#{1,6}\s+(.*?)[\s#]*$/.exec(bs[i])
    if (h) {
      const name = sumName(h[1].replace(/[*_`]/g, '').replace(/[:.]\s*$/, '').trim())
      if (!name || found.has(name)) continue
      const body: string[] = []
      // up to the next heading, or the other label's paragraph (an ELI5 inside the TLDR section)
      while (i + 1 < bs.length && !heading(bs[i + 1]) && !led(bs[i + 1])) body.push(bs[++i])
      if (body.length) found.set(name, body.join('\n\n'))
      continue
    }
    const name = led(bs[i])
    if (!name || found.has(name)) continue
    used.add(i)
    let body = bs[i].slice(sumLead.exec(bs[i])![0].length).trim()
    if (!body && i + 1 < bs.length && !heading(bs[i + 1])) used.add(++i), (body = bs[i])
    if (body) found.set(name, body)
  }
  const out = ['TLDR', 'ELI5'].filter((k) => found.has(k)).map((k) => ({ label: k, md: found.get(k)! }))
  if (!found.has('TLDR')) {
    const first = bs.findIndex((b, i) => prose(b) && !used.has(i))
    if (first >= 0 && (!found.has('ELI5') || first < Math.min(...used))) out.unshift({ label: '', md: bs[first] })
  }
  const opts = options(bs)
  if (opts) out.push({ label: 'Options', md: compactTables(opts) })
  return out
}

/**
 * compactTables turns each markdown table into one small block per row, to fit
 * a narrow panel: the first cell in bold (a size column beside it), then a
 * line for each other column ("Pros: …").
 */
export function compactTables(md: string): string {
  const cells = (l: string) => l.trim().replace(/^\||\|$/g, '').split('|').map((c) => c.trim())
  const lines = md.split('\n')
  const out: string[] = []
  for (let i = 0; i < lines.length; i++) {
    if (!/^\s*\|/.test(lines[i]) || !/^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|?\s*$/.test(lines[i + 1] ?? '')) {
      out.push(lines[i])
      continue
    }
    const head = cells(lines[i])
    const rows: string[][] = []
    for (i += 2; i < lines.length && /^\s*\|/.test(lines[i]); i++) rows.push(cells(lines[i]))
    i--
    const size = (k: number) => /^(size|lines|effort)$/i.test(head[k] ?? '')
    out.push(
      rows
        .map((r) => {
          const side = r.filter((c, k) => k > 0 && size(k) && c).join(', ')
          const rest = r.map((c, k) => (k > 0 && !size(k) && c ? `- **${head[k] || ''}:** ${c}` : '')).filter(Boolean)
          return [`**${r[0]}**${side ? ' · ' + side : ''}`, ...(rest.length ? ['', ...rest] : [])].join('\n')
        })
        .join('\n\n'),
    )
  }
  return out.join('\n')
}

// the Options section of a plan: its heading's blocks up to the next heading
// of the same or a higher level (the options' own subheadings stay in)
function options(bs: string[]): string {
  const level = (b: string) => /^(#{1,6})\s/.exec(b)?.[1].length ?? 0
  const i = bs.findIndex((b) => level(b) && /^(?:\d+[.)]\s*)?options?\s*[:.]?$/i.test(b.replace(/^#+\s+/, '').replace(/[*_`#\s]+$/, '').replace(/[*_`]/g, '')))
  if (i < 0) return ''
  const body: string[] = []
  for (let j = i + 1; j < bs.length && !(level(bs[j]) && level(bs[j]) <= level(bs[i])); j++) body.push(bs[j])
  return body.join('\n\n')
}

/** renderInline renders one line of markdown, without a paragraph. */
export function renderInline(src: string): string {
  return md.renderInline(src)
}

let mermaidMod: typeof import('mermaid').default | null = null

function token(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

/** renderMermaid renders every pending mermaid block inside root. */
export async function renderMermaid(root: HTMLElement, dark: boolean) {
  const nodes = root.querySelectorAll<HTMLElement>('pre.mermaid:not([data-processed])')
  if (!nodes.length) return
  mermaidMod ??= (await import('mermaid')).default
  mermaidMod.initialize({
    startOnLoad: false,
    securityLevel: 'strict',
    theme: 'base',
    fontFamily: 'Geist, system-ui, sans-serif',
    themeVariables: {
      darkMode: dark,
      background: token('--surface'),
      primaryColor: token('--bg'),
      primaryTextColor: token('--text'),
      primaryBorderColor: token('--border'),
      lineColor: token('--faint'),
      secondaryColor: token('--surface'),
      tertiaryColor: token('--surface'),
      fontSize: `${((parseFloat(token('--ui-size')) || 13) * 12.5) / 13}px`, // follows the UI font size, as --fs-12-5
    },
  })
  try {
    await mermaidMod.run({ nodes: Array.from(nodes) })
  } catch {
    // A bad diagram stays as its source text.
  }
}
