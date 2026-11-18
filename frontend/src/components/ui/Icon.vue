<script setup lang="ts">
// herd's icons, from Lucide (lucide.dev), by the names herd uses.
import { computed, type Component } from 'vue'
import {
  Archive, Check, ChevronDown, ChevronRight, Copy, Diff, ExternalLink, Eye, File, Folder, GitBranch, GitMerge, GitPullRequestArrow, GitPullRequestClosed, GitPullRequestDraft, GripVertical, ListTodo, Maximize2, Menu, Minus,
  PanelLeftClose, PanelLeftOpen, PanelRightClose, PanelRightOpen, Pencil, Play, Plus, RotateCw, Search, Settings, Split, Square, SquareTerminal, TriangleAlert, Undo2, X,
} from '@lucide/vue'

const props = withDefaults(defineProps<{ name: string; size?: number; stroke?: number }>(), { size: 14, stroke: 2 })

const icons: Record<string, Component> = {
  play: Play,
  'play-outline': Play, // the play icon unfilled, beside outline icons
  stop: Square,
  chevron: ChevronDown,
  'chevron-right': ChevronRight,
  folder: Folder,
  branch: GitBranch,
  rebase: GitMerge, // also a merged PR
  search: Search,
  plus: Plus,
  close: X,
  external: ExternalLink,
  file: File,
  terminal: SquareTerminal,
  archive: Archive,
  eye: Eye,
  diff: Diff,
  revert: Undo2,
  pr: GitPullRequestArrow,
  'pr-draft': GitPullRequestDraft,
  'pr-closed': GitPullRequestClosed,
  rerun: RotateCw,
  pencil: Pencil,
  check: Check,
  warn: TriangleAlert,
  copy: Copy,
  worktree: Split, // turned to point right, like Zed's worktree icon
  maximize: Maximize2,
  menu: Menu,
  gear: Settings,
  todo: ListTodo,
  grip: GripVertical,
  // window controls
  'win-min': Minus,
  'win-max': Square,
  'win-restore': Copy,
  'win-close': X,
  // side panels: shown (click hides) or hidden (click shows)
  'panel-left': PanelLeftClose,
  'panel-left-off': PanelLeftOpen,
  'panel-right': PanelRightClose,
  'panel-right-off': PanelRightOpen,
}
const filled = ['play', 'stop'] // solid, like media buttons
const turned = ['worktree']

const icon = computed(() => icons[props.name])
</script>

<template>
  <component :is="icon" v-if="icon" :size="size" :stroke-width="stroke" :fill="filled.includes(name) ? 'currentColor' : 'none'" :style="turned.includes(name) ? { transform: 'rotate(90deg)' } : undefined" aria-hidden="true" />
</template>
