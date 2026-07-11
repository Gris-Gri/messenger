import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { useMediaQuery } from '../hooks/useMediaQuery'

type SidebarContextValue = {
  isNarrow: boolean
  sidebarOpen: boolean
  openSidebar: () => void
  closeSidebar: () => void
  toggleSidebar: () => void
}

const SidebarContext = createContext<SidebarContextValue | null>(null)

export function SidebarProvider({ children }: { children: ReactNode }) {
  const isNarrow = useMediaQuery('(max-width: 768px)')
  const [sidebarOpen, setSidebarOpen] = useState(() => {
    if (typeof window === 'undefined') {
      return true
    }
    return !window.matchMedia('(max-width: 768px)').matches
  })

  useEffect(() => {
    if (!isNarrow) {
      setSidebarOpen(true)
    }
  }, [isNarrow])

  const openSidebar = useCallback(() => setSidebarOpen(true), [])
  const closeSidebar = useCallback(() => setSidebarOpen(false), [])
  const toggleSidebar = useCallback(() => setSidebarOpen((open) => !open), [])

  const value = useMemo(
    () => ({
      isNarrow,
      sidebarOpen,
      openSidebar,
      closeSidebar,
      toggleSidebar,
    }),
    [closeSidebar, isNarrow, openSidebar, sidebarOpen, toggleSidebar],
  )

  return <SidebarContext.Provider value={value}>{children}</SidebarContext.Provider>
}

export function useSidebar(): SidebarContextValue {
  const ctx = useContext(SidebarContext)
  if (!ctx) {
    throw new Error('useSidebar must be used within SidebarProvider')
  }
  return ctx
}
