// Derby's themes, each a dark and a light variant (a [data-theme] block in
// themes/*.css). Settings > Appearance lists one row per theme, with Dark,
// Light and System (follows the OS).
import type { ThemeMode, ThemeName } from './types'

export interface ThemeDef {
  id: ThemeName
  name: string
  note: string
  dark: string // data-theme of the dark variant
  light: string
  darkName: string
  lightName: string
}

export const themes: ThemeDef[] = [
  { id: 'fleet', name: 'Fleet', note: 'Zed’s Fleet Dark and Flatwhite', dark: 'fleet-dark', light: 'flatwhite', darkName: 'Fleet Dark', lightName: 'Flatwhite' },
  { id: 'catppuccin', name: 'Catppuccin', note: 'Mocha and Latte', dark: 'mocha', light: 'latte', darkName: 'Mocha', lightName: 'Latte' },
  { id: 'copilot', name: 'GitHub Copilot', note: 'The GitHub Copilot app’s colours', dark: 'copilot-dark', light: 'copilot-light', darkName: 'Copilot Dark', lightName: 'Copilot Light' },
]

export const modes: ThemeMode[] = ['dark', 'light', 'system']

export function themeDef(id: string | undefined): ThemeDef {
  return themes.find((t) => t.id === id) ?? themes[0]
}
