export type Theme = 'dark' | 'light'

export const THEME_STORAGE_KEY = 'theme'

export function readStoredTheme(): Theme {
  try {
    const value = localStorage.getItem(THEME_STORAGE_KEY)
    return value === 'light' ? 'light' : 'dark'
  } catch {
    return 'dark'
  }
}

export function getAppliedTheme(): Theme {
  return document.documentElement.getAttribute('data-theme') === 'light'
    ? 'light'
    : 'dark'
}

export function applyTheme(theme: Theme): void {
  document.documentElement.setAttribute('data-theme', theme)
  try {
    localStorage.setItem(THEME_STORAGE_KEY, theme)
  } catch {
    /* private mode / blocked storage — attribute still applies for this session */
  }
}

export function toggleTheme(): Theme {
  const next: Theme = getAppliedTheme() === 'light' ? 'dark' : 'light'
  applyTheme(next)
  return next
}
