import type { Component } from 'vue'

export interface FoundationSidebarItem {
  id: string
  label: string
  icon?: Component
  badge?: string
}

export interface FoundationSidebarSection {
  label: string
  items: FoundationSidebarItem[]
}
