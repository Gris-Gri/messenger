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
import { getAccessToken, subscribeAuth } from '../api/auth'
import {
  MessengerWebSocket,
  type OutgoingMessage,
  type WsAckFrame,
  type WsChatUpdatedFrame,
  type WsMessageEditedFrame,
  type WsNewMessageFrame,
  type WsPresenceFrame,
  type WsReactionUpdatedFrame,
  type WsReadFrame,
  type WsStatus,
  type WsUserUpdatedFrame,
} from '../api/ws'
import { useActiveChat } from '../context/ActiveChatContext'
import { useAuth } from '../context/AuthContext'
import { useUsers } from '../context/UsersContext'
import type { DisplayMessage, ReactionCounts } from '../types/domain'
import { createClientMsgId } from '../utils/clientMsgId'

export type ChatMessageHandlers = {
  chatId: number
  addOptimisticMessage: (message: DisplayMessage) => void
  markAcked: (clientMsgId: string, serverId: number) => void
  addIncomingMessage: (message: DisplayMessage) => void
  applyMessageEdited: (messageId: number, body: string, editedAt: string) => void
  applyReactionUpdated: (messageId: number, counts: ReactionCounts) => void
}

type WebSocketContextValue = {
  status: WsStatus
  sendMessage: (chatId: number, body: string) => string | null
  registerChatHandlers: (handlers: ChatMessageHandlers | null) => void
  registerReadHandler: (
    handler: ((userId: number, lastReadMessageId: number) => void) | null,
  ) => void
  updateChatPreview: (
    chatId: number,
    body: string,
    createdAt: string,
    lastMessageId?: number,
  ) => boolean
}

const WebSocketContext = createContext<WebSocketContextValue | null>(null)

type WebSocketProviderProps = {
  children: ReactNode
  updateChatPreview: (
    chatId: number,
    body: string,
    createdAt: string,
    lastMessageId?: number,
  ) => boolean
  patchLastMessageBodyIfMatch: (
    chatId: number,
    messageId: number,
    body: string,
  ) => void
  setChatTitle: (chatId: number, title: string) => void
  advanceMyReadCursor: (chatId: number, messageId: number) => void
  ensureChatFromMessage: (chatId: number) => Promise<void>
}

export function WebSocketProvider({
  children,
  updateChatPreview,
  patchLastMessageBodyIfMatch,
  setChatTitle,
  advanceMyReadCursor,
  ensureChatFromMessage,
}: WebSocketProviderProps) {
  const { isAuthenticated, currentUser } = useAuth()
  const { activeChatId } = useActiveChat()
  const { updateLogin, updatePresence } = useUsers()
  const [status, setStatus] = useState<WsStatus>('reconnecting')
  const clientRef = useRef<MessengerWebSocket | null>(null)
  const chatHandlersRef = useRef<ChatMessageHandlers | null>(null)
  const readHandlerRef = useRef<
    ((userId: number, lastReadMessageId: number) => void) | null
  >(null)
  const activeChatIdRef = useRef(activeChatId)
  const updateChatPreviewRef = useRef(updateChatPreview)
  const patchLastMessageBodyIfMatchRef = useRef(patchLastMessageBodyIfMatch)
  const setChatTitleRef = useRef(setChatTitle)
  const advanceMyReadCursorRef = useRef(advanceMyReadCursor)
  const ensureChatFromMessageRef = useRef(ensureChatFromMessage)
  const updateLoginRef = useRef(updateLogin)
  const updatePresenceRef = useRef(updatePresence)

  activeChatIdRef.current = activeChatId
  updateChatPreviewRef.current = updateChatPreview
  patchLastMessageBodyIfMatchRef.current = patchLastMessageBodyIfMatch
  setChatTitleRef.current = setChatTitle
  advanceMyReadCursorRef.current = advanceMyReadCursor
  ensureChatFromMessageRef.current = ensureChatFromMessage
  updateLoginRef.current = updateLogin
  updatePresenceRef.current = updatePresence

  const registerChatHandlers = useCallback((handlers: ChatMessageHandlers | null) => {
    chatHandlersRef.current = handlers
  }, [])

  const registerReadHandler = useCallback(
    (handler: ((userId: number, lastReadMessageId: number) => void) | null) => {
      readHandlerRef.current = handler
    },
    [],
  )

  const sendMessage = useCallback(
    (chatId: number, body: string): string | null => {
      const trimmed = body.trim()
      if (!trimmed || !currentUser) {
        return null
      }

      const clientMsgId = createClientMsgId()
      const optimistic: DisplayMessage = {
        id: 0,
        sender_id: currentUser.id,
        body: trimmed,
        created_at: new Date().toISOString(),
        client_msg_id: clientMsgId,
        delivery_status: 'pending',
      }

      if (chatHandlersRef.current?.chatId === chatId) {
        chatHandlersRef.current.addOptimisticMessage(optimistic)
      }

      updateChatPreviewRef.current(chatId, trimmed, optimistic.created_at)

      const payload: OutgoingMessage = {
        chatId,
        clientMsgId,
        body: trimmed,
      }

      clientRef.current?.sendMessage(payload)
      return clientMsgId
    },
    [currentUser],
  )

  useEffect(() => {
    if (!isAuthenticated || !getAccessToken()) {
      clientRef.current?.disconnect()
      clientRef.current = null
      setStatus('reconnecting')
      return
    }

    const client = new MessengerWebSocket()
    clientRef.current = client

    client.setHandlers({
      onStatusChange: setStatus,
      onAck: (frame: WsAckFrame, chatId?: number) => {
        chatHandlersRef.current?.markAcked(frame.client_msg_id, frame.server_id)
        if (chatId != null) {
          // Собственное сообщение: поднимаем last_message_id и свой курсор,
          // чтобы сайдбар не помечал чат непрочитанным из‑за своего же сообщения.
          updateChatPreviewRef.current(chatId, '', '', frame.server_id)
          advanceMyReadCursorRef.current(chatId, frame.server_id)
        }
      },
      onNewMessage: (frame: WsNewMessageFrame) => {
        const updated = updateChatPreviewRef.current(
          frame.chat_id,
          frame.message.body,
          frame.message.created_at,
          frame.message.id,
        )
        if (!updated) {
          void ensureChatFromMessageRef.current(frame.chat_id)
        }

        const openChatId = activeChatIdRef.current
        if (openChatId === frame.chat_id) {
          chatHandlersRef.current?.addIncomingMessage(frame.message)
        }
      },
      onRead: (frame: WsReadFrame) => {
        if (activeChatIdRef.current !== frame.chat_id) {
          return
        }
        readHandlerRef.current?.(frame.user_id, frame.last_read_message_id)
      },
      onChatUpdated: (frame: WsChatUpdatedFrame) => {
        setChatTitleRef.current(frame.chat_id, frame.title)
      },
      onUserUpdated: (frame: WsUserUpdatedFrame) => {
        updateLoginRef.current(frame.user_id, frame.login)
      },
      onPresence: (frame: WsPresenceFrame) => {
        updatePresenceRef.current(
          frame.user_id,
          frame.status,
          frame.last_seen_at ?? null,
        )
      },
      onMessageEdited: (frame: WsMessageEditedFrame) => {
        patchLastMessageBodyIfMatchRef.current(
          frame.chat_id,
          frame.message_id,
          frame.body,
        )

        const handlers = chatHandlersRef.current
        if (handlers && handlers.chatId === frame.chat_id) {
          handlers.applyMessageEdited(
            frame.message_id,
            frame.body,
            frame.edited_at,
          )
        }
      },
      onReactionUpdated: (frame: WsReactionUpdatedFrame) => {
        const handlers = chatHandlersRef.current
        if (handlers && handlers.chatId === frame.chat_id) {
          handlers.applyReactionUpdated(frame.message_id, frame.reactions)
        }
      },
    })

    client.connect()

    return () => {
      client.disconnect()
      if (clientRef.current === client) {
        clientRef.current = null
      }
    }
  }, [isAuthenticated])

  useEffect(() => {
    const unsubscribe = subscribeAuth(() => {
      if (!getAccessToken()) {
        clientRef.current?.disconnect()
        clientRef.current = null
        setStatus('reconnecting')
      }
    })
    return unsubscribe
  }, [])

  const value = useMemo<WebSocketContextValue>(
    () => ({
      status,
      sendMessage,
      registerChatHandlers,
      registerReadHandler,
      updateChatPreview,
    }),
    [registerChatHandlers, registerReadHandler, sendMessage, status, updateChatPreview],
  )

  return (
    <WebSocketContext.Provider value={value}>{children}</WebSocketContext.Provider>
  )
}

export function useWebSocket(): WebSocketContextValue {
  const ctx = useContext(WebSocketContext)
  if (!ctx) {
    throw new Error('useWebSocket must be used within WebSocketProvider')
  }
  return ctx
}
