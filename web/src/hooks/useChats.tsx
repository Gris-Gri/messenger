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
  reloadChats: () => Promise<void>
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

async function loadPeerNames(
  items: ChatListItem[],
  currentUserId: number,
): Promise<Record<number, string>> {
  const directChats = items.filter((chat) => chat.type === 'direct' && !chat.title)
  if (directChats.length === 0) {
    return {}
  }

  const entries = await Promise.all(
    directChats.map(async (chat) => {
      try {
        const members = await fetchChatMembers(chat.id)
        const peer = members.find((member) => member.user_id !== currentUserId)
        return peer ? ([chat.id, peer.login] as const) : null
      } catch {
        return null
      }
    }),
  )

  const next: Record<number, string> = {}
  for (const entry of entries) {
    if (entry) {
      next[entry[0]] = entry[1]
    }
  }
  return next
}

export function ChatsProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated, currentUser } = useAuth()
  const [chats, setChats] = useState<ChatListItem[]>([])
  const [peerNames, setPeerNames] = useState<Record<number, string>>({})
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const reloadChats = useCallback(async () => {
    if (!isAuthenticated || !currentUser) {
      setChats([])
      setPeerNames({})
      return
    }

    setLoading(true)
    setError(null)

    try {
      const items = await fetchChats()
      setChats(items)
      setPeerNames(await loadPeerNames(items, currentUser.id))
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить чаты')
    } finally {
      setLoading(false)
    }
  }, [currentUser, isAuthenticated])

  useEffect(() => {
    if (!isAuthenticated) {
      setChats([])
      setPeerNames({})
      setLoading(false)
      setError(null)
      return
    }

    let cancelled = false

    void (async () => {
      setLoading(true)
      setError(null)
      try {
        const items = await fetchChats()
        if (cancelled) {
          return
        }
        setChats(items)
        if (currentUser) {
          setPeerNames(await loadPeerNames(items, currentUser.id))
        } else {
          setPeerNames({})
        }
      } catch (err: unknown) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Не удалось загрузить чаты')
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    })()

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
      reloadChats,
    }),
    [chats, error, loading, peerNames, reloadChats, updateChatPreview],
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
