import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

type ActiveChatContextValue = {
  activeChatId: number | null
  setActiveChatId: (chatId: number) => void
  /** Счётчик запросов открыть MembersPanel (после создания группы). */
  membersPanelRequest: number
  requestOpenMembersPanel: () => void
}

const ActiveChatContext = createContext<ActiveChatContextValue | null>(null)

export function ActiveChatProvider({ children }: { children: ReactNode }) {
  const [activeChatId, setActiveChatId] = useState<number | null>(null)
  const [membersPanelRequest, setMembersPanelRequest] = useState(0)

  const requestOpenMembersPanel = useCallback(() => {
    setMembersPanelRequest((n) => n + 1)
  }, [])

  const value = useMemo(
    () => ({
      activeChatId,
      setActiveChatId,
      membersPanelRequest,
      requestOpenMembersPanel,
    }),
    [activeChatId, membersPanelRequest, requestOpenMembersPanel],
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
