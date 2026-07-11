import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { fetchChats } from '../api/chats'
import { fetchChatMembers } from '../api/members'
import { useAuth } from '../context/AuthContext'
import type { ChatListItem } from '../types/domain'

type ChatsContextValue = {
  chats: ChatListItem[]
  peerNames: Record<number, string>
  loading: boolean
  error: string | null
  updateChatPreview: (chatId: number, body: string, createdAt: string) => void
}

const ChatsContext = createContext<ChatsContextValue | null>(null)

function sortChatsByLastMessage(chats: ChatListItem[]): ChatListItem[] {
  return [...chats].sort((a, b) => {
    const aTime = a.last_message_at ? Date.parse(a.last_message_at) : 0
    const bTime = b.last_message_at ? Date.parse(b.last_message_at) : 0
    if (aTime !== bTime) {
      return bTime - aTime
    }
    return b.id - a.id
  })
}

export function ChatsProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated, currentUser } = useAuth()
  const [chats, setChats] = useState<ChatListItem[]>([])
  const [peerNames, setPeerNames] = useState<Record<number, string>>({})
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!isAuthenticated) {
      setChats([])
      setPeerNames({})
      setLoading(false)
      setError(null)
      return
    }

    let cancelled = false
    setLoading(true)
    setError(null)

    fetchChats()
      .then(async (items) => {
        if (cancelled) {
          return
        }

        setChats(items)

        const directChats = items.filter((chat) => chat.type === 'direct' && !chat.title)
        if (directChats.length === 0 || !currentUser) {
          setPeerNames({})
          return
        }

        const entries = await Promise.all(
          directChats.map(async (chat) => {
            try {
              const members = await fetchChatMembers(chat.id)
              const peer = members.find((member) => member.user_id !== currentUser.id)
              return peer ? ([chat.id, peer.login] as const) : null
            } catch {
              return null
            }
          }),
        )

        if (!cancelled) {
          const next: Record<number, string> = {}
          for (const entry of entries) {
            if (entry) {
              next[entry[0]] = entry[1]
            }
          }
          setPeerNames(next)
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Не удалось загрузить чаты')
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [currentUser, isAuthenticated])

  const updateChatPreview = useCallback(
    (chatId: number, body: string, createdAt: string) => {
      setChats((prev) =>
        sortChatsByLastMessage(
          prev.map((chat) =>
            chat.id === chatId
              ? {
                  ...chat,
                  last_message_body: body,
                  last_message_at: createdAt,
                }
              : chat,
          ),
        ),
      )
    },
    [],
  )

  const value = useMemo(
    () => ({
      chats,
      peerNames,
      loading,
      error,
      updateChatPreview,
    }),
    [chats, error, loading, peerNames, updateChatPreview],
  )

  return <ChatsContext.Provider value={value}>{children}</ChatsContext.Provider>
}

export function useChats(): ChatsContextValue {
  const ctx = useContext(ChatsContext)
  if (!ctx) {
    throw new Error('useChats must be used within ChatsProvider')
  }
  return ctx
}
