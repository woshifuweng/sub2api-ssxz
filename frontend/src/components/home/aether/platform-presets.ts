import { Apple, Box, Monitor, Terminal } from '@lucide/vue'
import type { Component } from 'vue'

export interface PlatformOption {
  value: string
  label: string
  hint: string
  icon: Component
  command: string
}

export const defaultPlatformOptions: PlatformOption[] = [
  { value: 'mac', label: 'macOS / Linux', hint: '终端', icon: Terminal, command: '' },
  { value: 'windows', label: 'Windows', hint: 'PowerShell', icon: Monitor, command: '' }
]

export const platformPresets = {
  default: {
    options: defaultPlatformOptions,
    defaultValue: 'mac'
  },
  claude: {
    options: [
      { value: 'mac', label: 'macOS / Linux', hint: '终端', icon: Terminal, command: 'curl -fsSL https://claude.ai/install.sh | bash' },
      { value: 'windows', label: 'Windows', hint: 'PowerShell', icon: Monitor, command: 'irm https://claude.ai/install.ps1 | iex' },
      { value: 'nodejs', label: 'Node.js', hint: 'npm', icon: Box, command: 'npm install -g @anthropic-ai/claude-code' },
      { value: 'homebrew', label: 'macOS', hint: 'Homebrew', icon: Apple, command: 'brew install --cask claude-code' }
    ] as PlatformOption[],
    defaultValue: 'mac'
  },
  codex: {
    options: [
      { value: 'nodejs', label: 'Node.js', hint: 'npm', icon: Box, command: 'npm install -g @openai/codex' },
      { value: 'homebrew', label: 'macOS', hint: 'Homebrew', icon: Apple, command: 'brew install --cask codex' }
    ] as PlatformOption[],
    defaultValue: 'nodejs'
  },
  gemini: {
    options: [
      { value: 'nodejs', label: 'Node.js', hint: 'npm', icon: Box, command: 'npm install -g @google/gemini-cli' },
      { value: 'homebrew', label: 'macOS', hint: 'Homebrew', icon: Apple, command: 'brew install gemini-cli' }
    ] as PlatformOption[],
    defaultValue: 'nodejs'
  }
} as const

export function getInstallCommand(preset: keyof typeof platformPresets, value: string): string {
  const config = platformPresets[preset]
  return config.options.find(option => option.value === value)?.command ?? ''
}
