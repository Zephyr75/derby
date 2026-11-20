// Syntax highlighting with tree-sitter, in the webview: the grammars (wasm)
// and their highlight queries ship in src/grammars and load on first use, a
// language at a time. A token gets the class of the theme key Zed would pick
// for its capture (themes/syntax.css colours them per theme).
import { Language, Parser, Query, type Node, type Range } from 'web-tree-sitter'
import core from 'web-tree-sitter/web-tree-sitter.wasm?url'

const wasms = import.meta.glob('../grammars/*.wasm', { query: '?url', import: 'default', eager: true }) as Record<string, string>
const queries = import.meta.glob('../grammars/*.scm', { query: '?raw', import: 'default' }) as Record<string, () => Promise<string>>

// A language: its grammar and its query file in src/grammars. TypeScript's
// query holds JavaScript's too, TSX's also JSX's, C++'s C's.
const langs: Record<string, [grammar: string, query: string]> = Object.assign(Object.create(null), {
  go: ['go', 'go'],
  python: ['python', 'python'],
  javascript: ['javascript', 'javascript'],
  typescript: ['typescript', 'typescript'],
  tsx: ['tsx', 'tsx'],
  bash: ['bash', 'bash'],
  c: ['c', 'c'],
  cpp: ['cpp', 'cpp'],
  html: ['html', 'html'],
  json: ['json', 'json'],
  toml: ['toml', 'toml'],
  yaml: ['yaml', 'yaml'],
  make: ['make', 'make'],
  sql: ['sql', 'sql'],
  markdown: ['markdown', 'markdown'],
  'markdown-inline': ['markdown_inline', 'markdown-inline'],
})

const exts: Record<string, string> = Object.assign(Object.create(null), {
  go: 'go',
  py: 'python', pyi: 'python',
  js: 'javascript', mjs: 'javascript', cjs: 'javascript', jsx: 'javascript',
  ts: 'typescript', mts: 'typescript', cts: 'typescript',
  tsx: 'tsx',
  sh: 'bash', bash: 'bash', zsh: 'bash',
  c: 'c', h: 'c',
  cc: 'cpp', cpp: 'cpp', cxx: 'cpp', hh: 'cpp', hpp: 'cpp', hxx: 'cpp',
  html: 'html', htm: 'html',
  json: 'json', jsonc: 'json',
  toml: 'toml',
  yaml: 'yaml', yml: 'yaml',
  mk: 'make',
  sql: 'sql',
  md: 'markdown', markdown: 'markdown',
})

/** langOf is the language of a file, by its name; '' when derby has none for it. */
export function langOf(path: string): string {
  const name = path.split('/').pop() ?? ''
  if (/^(GNUmakefile|[Mm]akefile)$/.test(name)) return 'make'
  const dot = name.lastIndexOf('.')
  return dot > 0 ? (exts[name.slice(dot + 1).toLowerCase()] ?? '') : ''
}

// The theme keys syntax.css styles: themes/syntax.css is generated for these,
// so the two change together.
const keys = ['attribute', 'boolean', 'comment', 'constant', 'constant.builtin', 'constant.macro', 'constructor', 'emphasis', 'emphasis.strong', 'function', 'function.builtin', 'function.call', 'function.method', 'function.special', 'keyword', 'keyword.conditional', 'keyword.exception', 'keyword.function', 'keyword.import', 'keyword.operator', 'keyword.repeat', 'label', 'link_text', 'link_uri', 'number', 'operator', 'property', 'punctuation.bracket', 'punctuation.delimiter', 'punctuation.special', 'string', 'string.escape', 'string.regex', 'string.special', 'tag', 'tag.doctype', 'text.literal', 'title', 'type', 'type.builtin', 'variable', 'variable.builtin', 'variable.parameter']
// Older capture names (Markdown's, nvim-treesitter's) as Zed's. (No prototype:
// captures and extensions like constructor are plain names here.)
const aliases: Record<string, string> = Object.assign(Object.create(null), {
  'text.title': 'title', 'text.emphasis': 'emphasis', 'text.strong': 'emphasis.strong', 'text.uri': 'link_uri', 'text.reference': 'link_text',
  repeat: 'keyword.repeat', conditional: 'keyword.conditional', include: 'keyword.import', exception: 'keyword.exception',
  parameter: 'variable.parameter', escape: 'string.escape', delimiter: 'punctuation.delimiter', 'string.special.key': 'property',
})

// Zed's rule: the key with the most parts, each found in the capture's parts in order.
const classes = new Map<string, string>()
function classOf(capture: string): string {
  let c = classes.get(capture)
  if (c !== undefined) return c
  const parts = (aliases[capture] ?? capture).split('.')
  let best = ''
  let bestLen = 0
  for (const k of keys) {
    const kps = k.split('.')
    let at = 0
    const ok = kps.every((kp) => {
      const i = parts.indexOf(kp, at)
      at = i + 1
      return i >= 0
    })
    if (ok && kps.length > bestLen) (best = k), (bestLen = kps.length)
  }
  c = best ? 'hl-' + best.replace(/\./g, '-') : ''
  classes.set(capture, c)
  return c
}

let ready: Promise<void> | null = null
const loaded = new Map<string, Promise<{ lang: Language; query: Query } | null>>()
function load(id: string) {
  let p = loaded.get(id)
  if (!p) {
    const [grammar, query] = langs[id]
    ready ??= Parser.init({ locateFile: () => core })
    p = ready
      .then(async () => {
        const lang = await Language.load(wasms[`../grammars/tree-sitter-${grammar}.wasm`])
        return { lang, query: new Query(lang, await queries[`../grammars/${query}.scm`]()) }
      })
      .catch((e) => {
        console.warn(`derby: no highlighting for ${id}:`, e)
        return null
      })
    loaded.set(id, p)
  }
  return p
}

/** A run of a line in one style ('' for none). */
export interface Seg {
  t: string
  c: string
}

const maxChars = 400_000

/** highlight splits text, in the language of path, into its lines' styled
 *  runs; null when the language has no grammar here. */
export async function highlight(path: string, text: string): Promise<Seg[][] | null> {
  const id = langOf(path)
  if (!id || text.length > maxChars) return null
  const l = await load(id)
  if (!l) return null
  const paint = new Int16Array(text.length).fill(-1)
  const names: string[] = []
  const parser = new Parser()
  try {
    parser.setLanguage(l.lang)
    const tree = parser.parse(text)
    if (!tree) return null
    apply(l.query.captures(tree.rootNode), paint, names)
    // Markdown's inline text (emphasis, code, links) is a grammar of its own.
    if (id === 'markdown') {
      const inline = tree.rootNode.descendantsOfType('inline')
      const il = await load('markdown-inline')
      if (il && inline.length) {
        parser.setLanguage(il.lang)
        const ranges: Range[] = inline.map((n) => ({ startIndex: n.startIndex, endIndex: n.endIndex, startPosition: n.startPosition, endPosition: n.endPosition }))
        const it = parser.parse(text, null, { includedRanges: ranges })
        if (it) apply(il.query.captures(it.rootNode), paint, names)
        it?.delete()
      }
    }
    tree.delete()
  } finally {
    parser.delete()
  }
  return lines(text, paint, names)
}

// apply paints each capture's class on its characters: outer nodes first, so
// nested ones show over them, and of the captures of one node the query's
// first pattern last, so it wins, as tree-sitter's highlighter does.
function apply(caps: { name: string; node: Node; patternIndex: number }[], paint: Int16Array, names: string[]) {
  const ix = new Map<string, number>(names.map((n, i) => [n, i]))
  const order = caps
    .map((c) => ({ c, cls: classOf(c.name) }))
    .filter((x) => x.cls)
    .sort((a, b) => a.c.node.startIndex - b.c.node.startIndex || b.c.node.endIndex - a.c.node.endIndex || b.c.patternIndex - a.c.patternIndex)
  for (const { c, cls } of order) {
    let i = ix.get(cls)
    if (i === undefined) {
      i = names.push(cls) - 1
      ix.set(cls, i)
    }
    paint.fill(i, c.node.startIndex, c.node.endIndex)
  }
}

function lines(text: string, paint: Int16Array, names: string[]): Seg[][] {
  const out: Seg[][] = []
  let start = 0
  for (const line of text.split('\n')) {
    const segs: Seg[] = []
    let from = 0
    for (let i = 1; i <= line.length; i++) {
      if (i === line.length || paint[start + i] !== paint[start + from]) {
        const p = paint[start + from]
        segs.push({ t: line.slice(from, i), c: p >= 0 ? names[p] : '' })
        from = i
      }
    }
    out.push(segs)
    start += line.length + 1
  }
  return out
}
