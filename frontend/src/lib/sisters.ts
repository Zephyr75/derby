// Sister tasks: the same work in several repositories. The left panel shows
// each family in its own category (jockey.panelFamilies); this names a task's
// sisters for a tooltip.

import { computed } from 'vue'
import { jockey } from './state'

/** groupOf maps each task with sisters to its family's task ids. */
export const groupOf = computed(() => jockey.sisterGroups)

/** sisterTitle names a task's sisters, for a tooltip. */
export function sisterTitle(id: string): string {
  const ids = groupOf.value.get(id)
  if (!ids) return ''
  const names = ids.filter((x) => x !== id).map((x) => jockey.task(x)?.projectName ?? x)
  return 'Sister tasks in ' + names.join(', ')
}
