<script setup lang="ts">
// derby's icons, from Lucide (lucide.dev), by the names derby uses.
import { computed, type Component } from 'vue'
import {
  Archive, Check, ChevronDown, ChevronRight, Copy, Diff, ExternalLink, Eye, File, Folder, FolderOpen, GitBranch, GitMerge, GitPullRequestArrow, GitPullRequestClosed, GitPullRequestDraft, GripVertical, ListTodo, Maximize2, Menu,
  PanelLeftClose, PanelLeftOpen, PanelRightClose, PanelRightOpen, Pencil, Play, Plus, RotateCw, Search, Settings, Split, Square, SquareTerminal, TriangleAlert, Undo2, X,
  ArrowDown, ArrowUp, Bot, Brain, Flag, UsersRound, ChevronUp, Hammer, Lightbulb, ListPlus, SendHorizontal, UserRound, CircleX, Globe, ListChecks, LoaderCircle, MessageSquare, ShieldAlert, Sparkles, Wrench,
  Monitor, Moon, Sun, CircleDot, Code, TextAlignStart, CircleCheck, CircleDashed, CircleMinus, CornerDownRight,
} from '@lucide/vue'

const props = withDefaults(defineProps<{ name: string; size?: number; stroke?: number }>(), { size: 14, stroke: 2 })

const icons: Record<string, Component> = {
  // the chat view
  send: ArrowUp,
  flag: Flag,
  sisters: UsersRound,
  'send-msg': SendHorizontal,
  queue: ListPlus,
  hammer: Hammer,
  think: Lightbulb,
  'chevron-up': ChevronUp,
  'user-up': UserRound,
  'arrow-up': ArrowUp,
  'arrow-down': ArrowDown,
  agent: Bot,
  thinking: Brain,
  error: CircleX,
  globe: Globe,
  todos: ListChecks,
  spinner: LoaderCircle,
  chat: MessageSquare,
  approval: ShieldAlert,
  skill: Sparkles,
  tool: Wrench,
  play: Play,
  'play-outline': Play, // the play icon unfilled, beside outline icons
  stop: Square,
  chevron: ChevronDown,
  'chevron-right': ChevronRight,
  folder: Folder,
  'folder-open': FolderOpen,
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
  issue: CircleDot, // GitHub's open-issue icon
  code: Code, // Zed
  description: TextAlignStart, // a task's own description, when it has no issue
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
  // a PR's checks and comments
  pass: CircleCheck,
  pending: CircleDashed,
  skip: CircleMinus,
  reply: CornerDownRight,
  grip: GripVertical,
  // window controls
  // side panels: shown (click hides) or hidden (click shows)
  'panel-left': PanelLeftClose,
  'panel-left-off': PanelLeftOpen,
  'panel-right': PanelRightClose,
  'panel-right-off': PanelRightOpen,
  // a theme's modes (Settings > Appearance)
  dark: Moon,
  light: Sun,
  system: Monitor,
}
const filled = ['stop'] // solid, like a media button
const turned = ['worktree']

const icon = computed(() => icons[props.name])
</script>

<template>
  <component :is="icon" v-if="icon" :size="size" :stroke-width="stroke" :fill="filled.includes(name) ? 'currentColor' : 'none'" :style="turned.includes(name) ? { transform: 'rotate(90deg)' } : undefined" aria-hidden="true" />
</template>
