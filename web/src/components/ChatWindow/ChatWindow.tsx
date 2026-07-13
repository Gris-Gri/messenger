import { Info, Menu, Pencil, Search, Send } from 'lucide-react'
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
  type TouchEvent,
} from 'react'
import { getChatDisplayName } from '../../api/chats'
import { toUserMessage } from '../../api/errors'
import { editMessage } from '../../api/messages'
import { useActiveChat } from '../../context/ActiveChatContext'
import { useAuth } from '../../context/AuthContext'
import { useSidebar } from '../../context/SidebarContext'
import { useUsers } from '../../context/UsersContext'
import { useChats } from '../../hooks/useChats'
import { useMemberNames } from '../../hooks/useMemberNames'
import { useMessages } from '../../hooks/useMessages'
import { useReadState } from '../../hooks/useReadState'
import { useWebSocket } from '../../hooks/useWebSocket'
import type { DisplayMessage, ReactionType } from '../../types/domain'
import { resolveOwnDeliveryStatus } from '../../utils/deliveryStatus'
import { formatMessageTime } from '../../utils/formatMessageTime'
import {
  hasAnyReaction,
  normalizeReactions,
  REACTION_EMOJI,
  REACTION_TYPES,
} from '../../utils/reactions'
import { Avatar } from '../Avatar/Avatar'
import { EmptyState } from '../EmptyState/EmptyState'
import { MembersPanel } from '../MembersPanel/MembersPanel'
import {
  MessageStatus,
  toMessageStatusKind,
} from '../MessageStatus/MessageStatus'
import { SearchPanel } from '../SearchPanel/SearchPanel'
import { MessageListSkeletonItems } from '../Skeleton/Skeleton'
import styles from './ChatWindow.module.css'

type ChatWindowProps = {
  chatId: number | null
  chatTitle: string | null
  chatType: 'direct' | 'group' | null
  avatarUserId: number | null
}

const LONG_PRESS_MS = 480

function resolveSenderName(
  senderId: number,
  chatType: 'direct' | 'group' | null,
  currentUserId: number | null,
  getLogin: (userId: number, fallback?: string) => string,
): string | undefined {
  if (chatType !== 'group' || currentUserId === senderId) {
    return undefined
  }

  const login = getLogin(senderId).trim()
  return login || undefined
}

function resizeTextarea(element: HTMLTextAreaElement): void {
  element.style.height = 'auto'
  element.style.height = `${Math.min(element.scrollHeight, 120)}px`
}

function resizeEditTextarea(element: HTMLTextAreaElement): void {
  element.style.height = 'auto'
  element.style.height = `${Math.min(element.scrollHeight, 160)}px`
}

export function ChatWindow({ chatId, chatTitle, chatType, avatarUserId }: ChatWindowProps) {
  const { currentUser } = useAuth()
  const { users, getLogin } = useUsers()
  const { membersPanelRequest } = useActiveChat()
  const { isNarrow, toggleSidebar } = useSidebar()
  const { advanceMyReadCursor, patchLastMessageBodyIfMatch } = useChats()
  const { sendMessage, registerChatHandlers, registerReadHandler } = useWebSocket()
  const {
    messages,
    loading,
    loadingMore,
    error,
    listRef,
    handleScroll,
    messageKey,
    scrollToMessage,
    highlightedMessageId,
    applyMessageEdited,
    toggleReaction,
  } = useMessages(chatId, registerChatHandlers)
  const { readCursors } = useReadState({
    chatId,
    messages,
    loading,
    currentUserId: currentUser?.id ?? null,
    advanceMyReadCursor,
    registerReadHandler,
  })
  const memberNames = useMemberNames(chatId, chatType, currentUser)
  const [draft, setDraft] = useState('')
  const [membersOpen, setMembersOpen] = useState(false)
  const [searchOpen, setSearchOpen] = useState(false)
  const [editingMessageId, setEditingMessageId] = useState<number | null>(null)
  const [editDraft, setEditDraft] = useState('')
  const [editSaving, setEditSaving] = useState(false)
  const [editError, setEditError] = useState<string | null>(null)
  const [actionsVisibleId, setActionsVisibleId] = useState<number | null>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const editTextareaRef = useRef<HTMLTextAreaElement>(null)
  const lastMembersRequestRef = useRef(0)
  const longPressTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const headerTitle = chatTitle ?? 'Выберите чат'
  const headerLogin =
    avatarUserId != null
      ? users[avatarUserId]?.login || chatTitle || ''
      : chatTitle || ''
  const canSend = chatId !== null && draft.trim().length > 0
  const canSaveEdit = editDraft.trim().length > 0 && !editSaving

  useEffect(() => {
    setSearchOpen(false)
    setEditingMessageId(null)
    setEditDraft('')
    setEditError(null)
    setActionsVisibleId(null)
    if (membersPanelRequest > lastMembersRequestRef.current) {
      lastMembersRequestRef.current = membersPanelRequest
      setMembersOpen(true)
      return
    }
    setMembersOpen(false)
  }, [chatId, membersPanelRequest])

  useEffect(() => {
    const textarea = textareaRef.current
    if (textarea) {
      resizeTextarea(textarea)
    }
  }, [draft, chatId])

  useEffect(() => {
    if (editingMessageId === null) {
      return
    }
    const textarea = editTextareaRef.current
    if (!textarea) {
      return
    }
    resizeEditTextarea(textarea)
    textarea.focus()
    textarea.setSelectionRange(textarea.value.length, textarea.value.length)
  }, [editingMessageId])

  const clearLongPressTimer = useCallback(() => {
    if (longPressTimerRef.current !== null) {
      clearTimeout(longPressTimerRef.current)
      longPressTimerRef.current = null
    }
  }, [])

  useEffect(() => {
    return () => {
      clearLongPressTimer()
    }
  }, [clearLongPressTimer])

  const openMembers = useCallback(() => {
    setSearchOpen(false)
    setMembersOpen(true)
  }, [])

  const handleSearchSelect = useCallback(
    async (messageId: number) => {
      await scrollToMessage(messageId)
    },
    [scrollToMessage],
  )

  const handleSend = useCallback(() => {
    if (chatId === null || !draft.trim()) {
      return
    }

    sendMessage(chatId, draft)
    setDraft('')
    const textarea = textareaRef.current
    if (textarea) {
      textarea.style.height = 'auto'
    }
  }, [chatId, draft, sendMessage])

  const handleKeyDown = useCallback(
    (event: KeyboardEvent<HTMLTextAreaElement>) => {
      if (event.key === 'Enter' && !event.shiftKey) {
        event.preventDefault()
        handleSend()
      }
    },
    [handleSend],
  )

  const startEdit = useCallback((message: DisplayMessage) => {
    if (message.id <= 0) {
      return
    }
    setEditingMessageId(message.id)
    setEditDraft(message.body)
    setEditError(null)
    setActionsVisibleId(null)
  }, [])

  const cancelEdit = useCallback(() => {
    setEditingMessageId(null)
    setEditDraft('')
    setEditError(null)
    setEditSaving(false)
  }, [])

  const saveEdit = useCallback(async () => {
    if (chatId === null || editingMessageId === null) {
      return
    }
    const trimmed = editDraft.trim()
    if (!trimmed || editSaving) {
      return
    }

    setEditSaving(true)
    setEditError(null)
    try {
      const updated = await editMessage(chatId, editingMessageId, trimmed)
      applyMessageEdited(
        updated.id,
        updated.body,
        updated.edited_at ?? new Date().toISOString(),
      )
      patchLastMessageBodyIfMatch(chatId, updated.id, updated.body)
      setEditingMessageId(null)
      setEditDraft('')
    } catch (err: unknown) {
      setEditError(toUserMessage(err, 'Не удалось сохранить сообщение'))
    } finally {
      setEditSaving(false)
    }
  }, [
    applyMessageEdited,
    chatId,
    editDraft,
    editSaving,
    editingMessageId,
    patchLastMessageBodyIfMatch,
  ])

  const handleEditKeyDown = useCallback(
    (event: KeyboardEvent<HTMLTextAreaElement>) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        cancelEdit()
        return
      }
      if (event.key === 'Enter' && !event.shiftKey) {
        event.preventDefault()
        void saveEdit()
      }
    },
    [cancelEdit, saveEdit],
  )

  const handleReactionClick = useCallback(
    (messageId: number, reaction: ReactionType) => {
      void toggleReaction(messageId, reaction)
    },
    [toggleReaction],
  )

  const handleMessageTouchStart = useCallback(
    (messageId: number) => {
      clearLongPressTimer()
      longPressTimerRef.current = setTimeout(() => {
        setActionsVisibleId(messageId)
        longPressTimerRef.current = null
      }, LONG_PRESS_MS)
    },
    [clearLongPressTimer],
  )

  const handleMessageTouchEnd = useCallback(() => {
    clearLongPressTimer()
  }, [clearLongPressTimer])

  const handleMessageTouchMove = useCallback(
    (_event: TouchEvent<HTMLLIElement>) => {
      clearLongPressTimer()
    },
    [clearLongPressTimer],
  )

  return (
    <section className={styles.chatWindow}>
      <div className={styles.chatMain}>
        <header className={styles.header}>
          {isNarrow && (
            <button
              type="button"
              className={styles.menuBtn}
              aria-label="Список чатов"
              onClick={toggleSidebar}
            >
              <Menu size={18} strokeWidth={1.75} aria-hidden />
            </button>
          )}
          <button
            type="button"
            className={styles.headerTitleBtn}
            onClick={chatId !== null ? openMembers : undefined}
            disabled={chatId === null}
          >
            {chatId !== null && avatarUserId !== null && headerLogin && (
              <Avatar userId={avatarUserId} login={headerLogin} size="sm" />
            )}
            <h1 className={styles.headerTitle}>{headerTitle}</h1>
          </button>
          <div className={styles.headerActions}>
            <button
              type="button"
              className={`${styles.headerBtn} ${searchOpen ? styles.headerBtnActive : ''}`}
              aria-label="Поиск по чату"
              disabled={chatId === null}
              onClick={() => {
                setMembersOpen(false)
                setSearchOpen((open) => !open)
              }}
            >
              <Search size={16} strokeWidth={1.75} aria-hidden />
            </button>
            <button
              type="button"
              className={`${styles.headerBtn} ${membersOpen ? styles.headerBtnActive : ''}`}
              aria-label="Сведения о чате"
              disabled={chatId === null}
              onClick={() => {
                setSearchOpen(false)
                setMembersOpen((open) => !open)
              }}
            >
              <Info size={16} strokeWidth={1.75} aria-hidden />
            </button>
          </div>
        </header>

        {chatId === null ? (
          <div className={styles.emptyState}>
            <EmptyState variant="selectChat" title="Выберите чат в списке слева" />
          </div>
        ) : (
          <>
            {searchOpen && (
              <SearchPanel
                chatId={chatId}
                memberNames={memberNames}
                onClose={() => setSearchOpen(false)}
                onSelectMessage={handleSearchSelect}
              />
            )}

            {error && <div className={styles.errorBanner}>{error}</div>}

            <div className={styles.feedArea}>
              <ul
                ref={listRef}
                className={styles.messageList}
                onScroll={handleScroll}
                onClick={() => {
                  if (actionsVisibleId !== null && editingMessageId === null) {
                    setActionsVisibleId(null)
                  }
                }}
              >
                {loadingMore && (
                  <li className={styles.loadMoreHint}>Загрузка…</li>
                )}
                {loading && messages.length === 0 && <MessageListSkeletonItems />}
                {!loading && !error && messages.length === 0 && (
                  <li className={styles.stateMessage}>Нет сообщений</li>
                )}
                {messages.map((msg, index) => {
                  const isOwn = currentUser?.id === msg.sender_id
                  const prev = index > 0 ? messages[index - 1] : null
                  const isFirstInGroup =
                    !isOwn && (!prev || prev.sender_id !== msg.sender_id)
                  const senderName = isFirstInGroup
                    ? resolveSenderName(
                        msg.sender_id,
                        chatType,
                        currentUser?.id ?? null,
                        getLogin,
                      )
                    : undefined
                  const senderLogin =
                    getLogin(msg.sender_id) ||
                    memberNames[msg.sender_id] ||
                    `#${msg.sender_id}`
                  const deliveryStatus =
                    isOwn && currentUser
                      ? resolveOwnDeliveryStatus(msg, currentUser.id, readCursors)
                      : null
                  const showDelivery = isOwn && (msg.delivery_status != null || msg.id > 0)
                  const isHighlighted = msg.id > 0 && msg.id === highlightedMessageId
                  const isEditing = isOwn && editingMessageId === msg.id
                  const canEdit = isOwn && msg.id > 0 && !isEditing
                  const isEdited = Boolean(msg.edited_at)
                  const canReact = msg.id > 0 && !isEditing
                  const reactions = normalizeReactions(msg.reactions)
                  const showReactions = hasAnyReaction(reactions)

                  return (
                    <li
                      key={messageKey(msg)}
                      data-message-id={msg.id > 0 ? msg.id : undefined}
                      className={`${styles.bubbleWrap} ${isOwn ? styles.bubbleWrapOwn : styles.bubbleWrapOther} ${isHighlighted ? styles.bubbleWrapHighlight : ''} ${actionsVisibleId === msg.id ? styles.actionsVisible : ''}`}
                      onTouchStart={
                        canReact
                          ? () => {
                              handleMessageTouchStart(msg.id)
                            }
                          : undefined
                      }
                      onTouchEnd={canReact ? handleMessageTouchEnd : undefined}
                      onTouchMove={canReact ? handleMessageTouchMove : undefined}
                      onTouchCancel={canReact ? handleMessageTouchEnd : undefined}
                    >
                      {!isOwn && (
                        <div className={styles.avatarSlot}>
                          {isFirstInGroup ? (
                            <Avatar userId={msg.sender_id} login={senderLogin} size="sm" />
                          ) : (
                            <span className={styles.avatarSpacer} aria-hidden="true" />
                          )}
                        </div>
                      )}
                      <div className={styles.bubbleColumn}>
                        {senderName && (
                          <span className={styles.senderName}>{senderName}</span>
                        )}
                        <div className={styles.bubbleRow}>
                          {canEdit && (
                            <button
                              type="button"
                              className={styles.editBtn}
                              aria-label="Редактировать"
                              onClick={(event) => {
                                event.stopPropagation()
                                startEdit(msg)
                              }}
                            >
                              <Pencil size={14} strokeWidth={1.75} aria-hidden />
                            </button>
                          )}
                          {canReact && (
                            <div
                              className={styles.reactionPicker}
                              role="group"
                              aria-label="Реакции"
                              onClick={(event) => event.stopPropagation()}
                            >
                              {REACTION_TYPES.map((reaction) => (
                                <button
                                  key={reaction}
                                  type="button"
                                  className={`${styles.reactionPickerBtn} ${
                                    reactions.my_reaction === reaction
                                      ? styles.reactionPickerBtnActive
                                      : ''
                                  }`}
                                  aria-label={reaction}
                                  aria-pressed={reactions.my_reaction === reaction}
                                  onClick={() => handleReactionClick(msg.id, reaction)}
                                >
                                  {REACTION_EMOJI[reaction]}
                                </button>
                              ))}
                            </div>
                          )}
                          {isEditing ? (
                            <div
                              className={`${styles.bubble} ${styles.bubbleOwn} ${styles.bubbleEditing}`}
                            >
                              <textarea
                                ref={editTextareaRef}
                                className={styles.editTextarea}
                                rows={1}
                                value={editDraft}
                                disabled={editSaving}
                                onChange={(e) => {
                                  setEditDraft(e.target.value)
                                  resizeEditTextarea(e.currentTarget)
                                }}
                                onKeyDown={handleEditKeyDown}
                                onClick={(e) => e.stopPropagation()}
                              />
                              <div className={styles.editActions}>
                                <button
                                  type="button"
                                  className={styles.editCancelBtn}
                                  disabled={editSaving}
                                  onClick={(e) => {
                                    e.stopPropagation()
                                    cancelEdit()
                                  }}
                                >
                                  Отмена
                                </button>
                                <button
                                  type="button"
                                  className={styles.editSaveBtn}
                                  disabled={!canSaveEdit}
                                  onClick={(e) => {
                                    e.stopPropagation()
                                    void saveEdit()
                                  }}
                                >
                                  Сохранить
                                </button>
                              </div>
                              {editError && (
                                <p className={styles.editError}>{editError}</p>
                              )}
                            </div>
                          ) : (
                            <div
                              className={`${styles.bubble} ${isOwn ? styles.bubbleOwn : styles.bubbleOther}`}
                            >
                              <span className={styles.bubbleBody}>{msg.body}</span>
                            </div>
                          )}
                        </div>
                        <div className={styles.messageFooter}>
                          {showReactions && (
                            <div
                              className={styles.reactionRow}
                              onClick={(event) => event.stopPropagation()}
                            >
                              {REACTION_TYPES.filter(
                                (reaction) => reactions[reaction] > 0,
                              ).map((reaction) => (
                                <button
                                  key={reaction}
                                  type="button"
                                  className={`${styles.reactionChip} ${
                                    reactions.my_reaction === reaction
                                      ? styles.reactionChipMine
                                      : ''
                                  }`}
                                  aria-label={`${REACTION_EMOJI[reaction]} ${reactions[reaction]}`}
                                  aria-pressed={reactions.my_reaction === reaction}
                                  onClick={() => handleReactionClick(msg.id, reaction)}
                                >
                                  <span aria-hidden>{REACTION_EMOJI[reaction]}</span>
                                  <span className={styles.reactionCount}>
                                    {reactions[reaction]}
                                  </span>
                                </button>
                              ))}
                            </div>
                          )}
                          <div className={styles.meta}>
                            {isEdited && (
                              <span className={styles.editedMark}>ред.</span>
                            )}
                            <span className={styles.timestamp}>
                              {formatMessageTime(msg.created_at)}
                            </span>
                            {showDelivery && deliveryStatus && (
                              <MessageStatus status={toMessageStatusKind(deliveryStatus)} />
                            )}
                          </div>
                        </div>
                      </div>
                    </li>
                  )
                })}
              </ul>
            </div>
          </>
        )}

        <div className={`chromeBar ${styles.inputArea}`}>
          <textarea
            ref={textareaRef}
            className={styles.textarea}
            rows={1}
            placeholder="Сообщение…"
            value={draft}
            onChange={(e) => {
              setDraft(e.target.value)
              resizeTextarea(e.currentTarget)
            }}
            onKeyDown={handleKeyDown}
            disabled={chatId === null}
          />
          <button
            type="button"
            className={styles.sendBtn}
            disabled={!canSend}
            aria-label="Отправить"
            onClick={handleSend}
          >
            <Send size={18} strokeWidth={1.75} aria-hidden />
          </button>
        </div>
      </div>

      {chatId !== null && (
        <MembersPanel
          chatId={chatId}
          open={membersOpen}
          onClose={() => setMembersOpen(false)}
        />
      )}
    </section>
  )
}

export function ConnectedChatWindow() {
  const { chats, peerUserIds } = useChats()
  const { users } = useUsers()
  const { activeChatId } = useActiveChat()

  const activeChat = useMemo(
    () => chats.find((chat) => chat.id === activeChatId) ?? null,
    [activeChatId, chats],
  )

  const peerLogin =
    activeChat?.type === 'direct' && peerUserIds[activeChat.id] != null
      ? users[peerUserIds[activeChat.id]!]?.login
      : undefined

  const chatTitle = activeChat ? getChatDisplayName(activeChat, peerLogin) : null
  const avatarUserId = activeChat
    ? activeChat.type === 'direct' && peerUserIds[activeChat.id] != null
      ? peerUserIds[activeChat.id]!
      : activeChat.id
    : null

  return (
    <ChatWindow
      chatId={activeChat?.id ?? null}
      chatTitle={chatTitle}
      chatType={activeChat?.type ?? null}
      avatarUserId={avatarUserId}
    />
  )
}
