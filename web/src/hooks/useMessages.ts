import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type UIEventHandler,
} from 'react'
import { fetchMessages } from '../api/messages'
import type { DisplayMessage, Message } from '../types/domain'
import type { ChatMessageHandlers } from './useWebSocket'

const PAGE_SIZE = 50
const SCROLL_TOP_THRESHOLD = 48

type ScrollAdjust = {
  prevScrollHeight: number
  prevScrollTop: number
}

function messageKey(message: DisplayMessage): string {
  return message.client_msg_id ?? String(message.id)
}

export function useMessages(
  chatId: number | null,
  registerChatHandlers?: (handlers: ChatMessageHandlers | null) => void,
) {
  const [messages, setMessages] = useState<DisplayMessage[]>([])
  const [loading, setLoading] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [hasMore, setHasMore] = useState(false)

  const listRef = useRef<HTMLUListElement>(null)
  const pendingScrollAdjust = useRef<ScrollAdjust | null>(null)
  const shouldScrollToBottom = useRef(false)
  const pendingByClientId = useRef<Map<string, DisplayMessage>>(new Map())
  const messagesRef = useRef<DisplayMessage[]>([])
  const hasMoreRef = useRef(false)
  const [highlightedMessageId, setHighlightedMessageId] = useState<number | null>(null)

  useEffect(() => {
    messagesRef.current = messages
  }, [messages])

  useEffect(() => {
    hasMoreRef.current = hasMore
  }, [hasMore])

  const addOptimisticMessage = useCallback((message: DisplayMessage) => {
    if (message.client_msg_id) {
      pendingByClientId.current.set(message.client_msg_id, message)
    }
    shouldScrollToBottom.current = true
    setMessages((prev) => {
      if (
        message.client_msg_id &&
        prev.some((item) => item.client_msg_id === message.client_msg_id)
      ) {
        return prev
      }
      return [...prev, message]
    })
  }, [])

  const markAcked = useCallback((clientMsgId: string, serverId: number) => {
    pendingByClientId.current.delete(clientMsgId)
    setMessages((prev) =>
      prev.map((message) =>
        message.client_msg_id === clientMsgId
          ? {
              ...message,
              id: serverId,
              delivery_status: 'acked',
            }
          : message,
      ),
    )
  }, [])

  const addIncomingMessage = useCallback((message: Message) => {
    setMessages((prev) => {
      if (prev.some((item) => item.id === message.id && item.id > 0)) {
        return prev
      }
      shouldScrollToBottom.current = true
      return [...prev, message]
    })
  }, [])

  useEffect(() => {
    if (!registerChatHandlers || chatId === null) {
      registerChatHandlers?.(null)
      return
    }

    registerChatHandlers({
      chatId,
      addOptimisticMessage,
      markAcked,
      addIncomingMessage,
    })

    return () => {
      registerChatHandlers(null)
    }
  }, [
    addIncomingMessage,
    addOptimisticMessage,
    chatId,
    markAcked,
    registerChatHandlers,
  ])

  useEffect(() => {
    if (chatId === null) {
      setMessages([])
      setLoading(false)
      setLoadingMore(false)
      setError(null)
      setHasMore(false)
      pendingScrollAdjust.current = null
      shouldScrollToBottom.current = false
      pendingByClientId.current.clear()
      return
    }

    let cancelled = false
    setLoading(true)
    setLoadingMore(false)
    setError(null)
    setHasMore(false)
    pendingScrollAdjust.current = null
    shouldScrollToBottom.current = true

    fetchMessages(chatId, { limit: PAGE_SIZE })
      .then((page) => {
        if (cancelled) {
          return
        }

        const chronological: DisplayMessage[] = [...page].reverse()
        const pending = [...pendingByClientId.current.values()].filter(
          (message) => message.client_msg_id,
        )

        setMessages(() => {
          const merged = [...chronological]
          for (const pendingMessage of pending) {
            if (
              pendingMessage.client_msg_id &&
              merged.some((item) => item.client_msg_id === pendingMessage.client_msg_id)
            ) {
              continue
            }
            if (pendingMessage.id > 0 && merged.some((item) => item.id === pendingMessage.id)) {
              continue
            }
            merged.push(pendingMessage)
          }
          return merged
        })
        setHasMore(page.length === PAGE_SIZE)
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Не удалось загрузить сообщения')
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
  }, [chatId])

  useLayoutEffect(() => {
    const list = listRef.current
    if (!list) {
      return
    }

    if (pendingScrollAdjust.current) {
      const { prevScrollHeight, prevScrollTop } = pendingScrollAdjust.current
      list.scrollTop = list.scrollHeight - prevScrollHeight + prevScrollTop
      pendingScrollAdjust.current = null
      return
    }

    if (shouldScrollToBottom.current) {
      list.scrollTop = list.scrollHeight
      shouldScrollToBottom.current = false
    }
  }, [messages])

  const loadMore = useCallback(async () => {
    if (chatId === null || loading || loadingMore || !hasMore || messages.length === 0) {
      return
    }

    const oldestId = messages.find((message) => message.id > 0)?.id
    if (!oldestId) {
      return
    }

    const list = listRef.current
    if (list) {
      pendingScrollAdjust.current = {
        prevScrollHeight: list.scrollHeight,
        prevScrollTop: list.scrollTop,
      }
    }

    setLoadingMore(true)
    setError(null)

    try {
      const page = await fetchMessages(chatId, {
        beforeId: oldestId,
        limit: PAGE_SIZE,
      })

      if (page.length === 0) {
        pendingScrollAdjust.current = null
        setHasMore(false)
        return
      }

      const older = [...page].reverse()
      setMessages((prev) => [...older, ...prev])
      setHasMore(page.length === PAGE_SIZE)
    } catch (err: unknown) {
      pendingScrollAdjust.current = null
      setError(err instanceof Error ? err.message : 'Не удалось загрузить сообщения')
    } finally {
      setLoadingMore(false)
    }
  }, [chatId, hasMore, loading, loadingMore, messages])

  const handleScroll: UIEventHandler<HTMLUListElement> = useCallback(
    (event) => {
      if (event.currentTarget.scrollTop <= SCROLL_TOP_THRESHOLD) {
        void loadMore()
      }
    },
    [loadMore],
  )

  const ensureMessageLoaded = useCallback(
    async (messageId: number): Promise<boolean> => {
      if (chatId === null) {
        return false
      }

      if (messagesRef.current.some((message) => message.id === messageId)) {
        return true
      }

      while (hasMoreRef.current) {
        const oldestId = messagesRef.current.find((message) => message.id > 0)?.id
        if (!oldestId) {
          return false
        }

        const list = listRef.current
        if (list) {
          pendingScrollAdjust.current = {
            prevScrollHeight: list.scrollHeight,
            prevScrollTop: list.scrollTop,
          }
        }

        try {
          const page = await fetchMessages(chatId, {
            beforeId: oldestId,
            limit: PAGE_SIZE,
          })

          if (page.length === 0) {
            pendingScrollAdjust.current = null
            hasMoreRef.current = false
            setHasMore(false)
            return false
          }

          const older = [...page].reverse()
          setMessages((prev) => {
            const next = [...older, ...prev]
            messagesRef.current = next
            return next
          })
          const nextHasMore = page.length === PAGE_SIZE
          hasMoreRef.current = nextHasMore
          setHasMore(nextHasMore)

          if (older.some((message) => message.id === messageId)) {
            return true
          }
        } catch (err: unknown) {
          pendingScrollAdjust.current = null
          setError(err instanceof Error ? err.message : 'Не удалось загрузить сообщения')
          return false
        }
      }

      return messagesRef.current.some((message) => message.id === messageId)
    },
    [chatId],
  )

  const scrollToMessage = useCallback(
    async (messageId: number): Promise<boolean> => {
      const loaded = await ensureMessageLoaded(messageId)
      if (!loaded) {
        return false
      }

      setHighlightedMessageId(messageId)

      requestAnimationFrame(() => {
        const element = listRef.current?.querySelector(
          `[data-message-id="${messageId}"]`,
        )
        element?.scrollIntoView({ block: 'center' })
      })

      return true
    },
    [ensureMessageLoaded],
  )

  useEffect(() => {
    if (highlightedMessageId === null) {
      return
    }

    const timer = window.setTimeout(() => {
      setHighlightedMessageId(null)
    }, 2000)

    return () => {
      window.clearTimeout(timer)
    }
  }, [highlightedMessageId])

  return {
    messages,
    loading,
    loadingMore,
    error,
    hasMore,
    listRef,
    handleScroll,
    messageKey,
    scrollToMessage,
    highlightedMessageId,
  }
}
