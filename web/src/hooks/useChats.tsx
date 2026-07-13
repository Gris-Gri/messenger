import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { fetchChats } from '../api/chats'
import { toUserMessage } from '../api/errors'
import { fetchChatMembers } from '../api/members'
import { useAuth } from '../context/AuthContext'
import { membersToUpserts, useUsers } from '../context/UsersContext'
import type { Chat, ChatListItem } from '../types/domain'

type ReloadChatsOptions = {
  /** Не показывать «Загрузка…» в сайдбаре (фоновый синк по WS). */
  silent?: boolean
}

type ChatsContextValue = {
  chats: ChatListItem[]
  /** chatId → peer userId для direct-чатов (логин/аватар из Users store). */
  peerUserIds: Record<number, number>
  loading: boolean
  error: string | null
  /** Обновляет превью; `false`, если чата ещё нет в локальном списке. */
  updateChatPreview: (
    chatId: number,
    body: string,
    createdAt: string,
    lastMessageId?: number,
  ) => boolean
  /** Если messageId — последнее в чате, обновляет last_message_body (после edit). */
  patchLastMessageBodyIfMatch: (
    chatId: number,
    messageId: number,
    body: string,
  ) => void
  /** Локально обновляет title группового чата (PATCH / WS chat_updated). */
  setChatTitle: (chatId: number, title: string) => void
  /** Поднимает my_last_read_message_id (GREATEST), чтобы снять unread в сайдбаре. */
  advanceMyReadCursor: (chatId: number, messageId: number) => void
  /** Сразу вставляет ответ POST /chats в начало списка (без ожидания WS/рефетча). */
  upsertCreatedChat: (
    chat: Chat,
    peer?: { login: string; userId: number },
  ) => void
  /**
   * Если `chat_id` из `new_message` отсутствует в списке — точечный рефетч GET /chats.
   * Иначе no-op.
   */
  ensureChatFromMessage: (chatId: number) => Promise<void>
  reloadChats: (options?: ReloadChatsOptions) => Promise<void>
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

function chatToListItem(chat: Chat): ChatListItem {
  return {
    id: chat.id,
    type: chat.type,
    title: chat.title,
    last_message_id: null,
    last_message_body: null,
    last_message_at: null,
    my_last_read_message_id: 0,
  }
}

async function loadPeerUserIds(
  items: ChatListItem[],
  currentUserId: number,
): Promise<{
  userIds: Record<number, number>
  membersByChat: Awaited<ReturnType<typeof fetchChatMembers>>[]
}> {
  const directChats = items.filter((chat) => chat.type === 'direct' && !chat.title)
  if (directChats.length === 0) {
    return { userIds: {}, membersByChat: [] }
  }

  const results = await Promise.all(
    directChats.map(async (chat) => {
      try {
        const members = await fetchChatMembers(chat.id)
        const peer = members.find((member) => member.user_id !== currentUserId)
        return { chatId: chat.id, peer, members }
      } catch {
        return null
      }
    }),
  )

  const userIds: Record<number, number> = {}
  const membersByChat: Awaited<ReturnType<typeof fetchChatMembers>>[] = []
  for (const entry of results) {
    if (!entry) {
      continue
    }
    membersByChat.push(entry.members)
    if (entry.peer) {
      userIds[entry.chatId] = entry.peer.user_id
    }
  }
  return { userIds, membersByChat }
}

export function ChatsProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated, currentUser } = useAuth()
  const { upsertUsers } = useUsers()
  const [chats, setChats] = useState<ChatListItem[]>([])
  const [peerUserIds, setPeerUserIds] = useState<Record<number, number>>({})
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const chatsRef = useRef(chats)
  const refreshInFlightRef = useRef<Promise<void> | null>(null)
  const upsertUsersRef = useRef(upsertUsers)

  chatsRef.current = chats
  upsertUsersRef.current = upsertUsers

  const applyPeers = useCallback(
    async (items: ChatListItem[], userId: number) => {
      const peers = await loadPeerUserIds(items, userId)
      setPeerUserIds((prev) => ({ ...prev, ...peers.userIds }))
      for (const members of peers.membersByChat) {
        upsertUsersRef.current(membersToUpserts(members))
      }
    },
    [],
  )

  const reloadChats = useCallback(
    async (options?: ReloadChatsOptions) => {
      if (!isAuthenticated || !currentUser) {
        setChats([])
        setPeerUserIds({})
        return
      }

      if (refreshInFlightRef.current) {
        return refreshInFlightRef.current
      }

      const silent = options?.silent === true
      const run = async () => {
        if (!silent) {
          setLoading(true)
        }
        setError(null)

        try {
          const items = await fetchChats()
          setChats(items)
          await applyPeers(items, currentUser.id)
        } catch (err: unknown) {
          setError(toUserMessage(err, 'Не удалось загрузить чаты'))
        } finally {
          if (!silent) {
            setLoading(false)
          }
        }
      }

      refreshInFlightRef.current = run().finally(() => {
        refreshInFlightRef.current = null
      })
      return refreshInFlightRef.current
    },
    [applyPeers, currentUser, isAuthenticated],
  )

  useEffect(() => {
    if (!isAuthenticated) {
      setChats([])
      setPeerUserIds({})
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
          await applyPeers(items, currentUser.id)
        } else {
          setPeerUserIds({})
        }
      } catch (err: unknown) {
        if (!cancelled) {
          setError(toUserMessage(err, 'Не удалось загрузить чаты'))
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
  }, [applyPeers, currentUser, isAuthenticated])

  const updateChatPreview = useCallback(
    (
      chatId: number,
      body: string,
      createdAt: string,
      lastMessageId?: number,
    ): boolean => {
      if (!chatsRef.current.some((chat) => chat.id === chatId)) {
        return false
      }

      setChats((prev) =>
        sortChatsByLastMessage(
          prev.map((chat) => {
            if (chat.id !== chatId) {
              return chat
            }

            const next: ChatListItem = { ...chat }
            if (body) {
              next.last_message_body = body
            }
            if (createdAt) {
              next.last_message_at = createdAt
            }
            if (lastMessageId != null && lastMessageId > 0) {
              next.last_message_id = Math.max(chat.last_message_id ?? 0, lastMessageId)
            }
            return next
          }),
        ),
      )
      return true
    },
    [],
  )

  const patchLastMessageBodyIfMatch = useCallback(
    (chatId: number, messageId: number, body: string) => {
      setChats((prev) =>
        prev.map((chat) => {
          if (chat.id !== chatId || chat.last_message_id !== messageId) {
            return chat
          }
          return { ...chat, last_message_body: body }
        }),
      )
    },
    [],
  )

  const setChatTitle = useCallback((chatId: number, title: string) => {
    setChats((prev) =>
      prev.map((chat) => (chat.id === chatId ? { ...chat, title } : chat)),
    )
  }, [])

  const advanceMyReadCursor = useCallback((chatId: number, messageId: number) => {
    if (messageId <= 0) {
      return
    }

    setChats((prev) =>
      prev.map((chat) => {
        if (chat.id !== chatId) {
          return chat
        }
        const current = chat.my_last_read_message_id ?? 0
        if (messageId <= current) {
          return chat
        }
        return { ...chat, my_last_read_message_id: messageId }
      }),
    )
  }, [])

  const upsertCreatedChat = useCallback(
    (chat: Chat, peer?: { login: string; userId: number }) => {
      const item = chatToListItem(chat)
      setChats((prev) => {
        if (prev.some((existing) => existing.id === chat.id)) {
          return prev
        }
        return [item, ...prev]
      })

      if (peer) {
        setPeerUserIds((prev) => ({ ...prev, [chat.id]: peer.userId }))
        upsertUsersRef.current([
          { user_id: peer.userId, login: peer.login },
        ])
        return
      }

      if (chat.type === 'direct' && currentUser) {
        void applyPeers([item], currentUser.id)
      }
    },
    [applyPeers, currentUser],
  )

  const ensureChatFromMessage = useCallback(
    async (chatId: number) => {
      if (chatsRef.current.some((chat) => chat.id === chatId)) {
        return
      }
      await reloadChats({ silent: true })
    },
    [reloadChats],
  )

  const value = useMemo(
    () => ({
      chats,
      peerUserIds,
      loading,
      error,
      updateChatPreview,
      patchLastMessageBodyIfMatch,
      setChatTitle,
      advanceMyReadCursor,
      upsertCreatedChat,
      ensureChatFromMessage,
      reloadChats,
    }),
    [
      advanceMyReadCursor,
      chats,
      error,
      ensureChatFromMessage,
      loading,
      patchLastMessageBodyIfMatch,
      peerUserIds,
      reloadChats,
      setChatTitle,
      updateChatPreview,
      upsertCreatedChat,
    ],
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
