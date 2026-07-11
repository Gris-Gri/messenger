import { createContext, useContext, useMemo, useState } from 'react'

type ActiveChatContextValue = {
  activeChatId: number | null
  setActiveChatId: (chatId: number) => void
}

const ActiveChatContext = createContext<ActiveChatContextValue | null>(null)

export function ActiveChatProvider({ children }: { children: React.ReactNode }) {
  const [activeChatId, setActiveChatId] = useState<number | null>(null)

  const value = useMemo(
    () => ({
      activeChatId,
      setActiveChatId,
    }),
    [activeChatId],
  )

  return (
    <ActiveChatContext.Provider value={value}>{children}</ActiveChatContext.Provider>
  )
}

export function useActiveChat(): ActiveChatContextValue {
  const ctx = useContext(ActiveChatContext)
  if (!ctx) {
    throw new Error('useActiveChat must be used within ActiveChatProvider')
  }
  return ctx
}
