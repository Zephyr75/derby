import { onBeforeUnmount, onUpdated, ref, watch, type Ref } from 'vue'

/** The left margin of a .label, the name next to an icon. */
export const LABEL_GAP = 6

/**
 * useCompact tells when buttons should drop their names (.label) and keep
 * only their icons: when the children of one of groups(root) no longer fit
 * on one line. The component hides them with .compact .label (max-width: 0,
 * no margin, overflow hidden), which keeps their natural width measurable,
 * so it knows when they fit again. A .swap, an icon shown only while compact,
 * stands in for its label (and its LABEL_GAP): it doesn't count once they fit.
 */
export function useCompact(root: Ref<HTMLElement | undefined>, groups: (root: HTMLElement) => HTMLElement[] = (el) => [el]) {
  const compact = ref(false)

  function need(g: HTMLElement): number {
    const s = getComputedStyle(g)
    const kids = [...g.children] as HTMLElement[]
    let w = kids.reduce((n, k) => n + k.offsetWidth, 0) + (parseFloat(s.columnGap) || 0) * Math.max(0, kids.length - 1)
    w += (parseFloat(s.paddingLeft) || 0) + (parseFloat(s.paddingRight) || 0)
    if (compact.value) {
      for (const l of g.querySelectorAll<HTMLElement>('.label')) w += l.scrollWidth + LABEL_GAP
      for (const i of g.querySelectorAll('.swap')) w -= i.getBoundingClientRect().width + LABEL_GAP
    }
    return w
  }
  function fit() {
    const el = root.value
    if (!el) return
    const over = groups(el).map((g) => need(g) - g.clientWidth)
    // a few pixels of slack each way so rounding can't make it flicker
    const next = over.some((o) => o > (compact.value ? -6 : -2))
    if (next !== compact.value) compact.value = next
  }

  const sizes = new ResizeObserver(fit)
  watch(
    root,
    (el, old) => {
      if (old) sizes.unobserve(old)
      if (el) sizes.observe(el)
    },
    { flush: 'post' },
  )
  onUpdated(fit)
  document.fonts?.ready.then(fit)
  onBeforeUnmount(() => sizes.disconnect())
  return compact
}
