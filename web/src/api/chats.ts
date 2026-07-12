import type { Chat, ChatListItem } from '../types/domain'
import { apiClient } from './client'

export function fetchChats(): Promise<ChatListItem[]> {
  return apiClient<ChatListItem[]>('/chats')
}

export function createDirectChat(userId: number): Promise<Chat> {
  return apiClient<Chat>('/chats', {
    method: 'POST',
    body: JSON.stringify({ type: 'direct', user_id: userId }),
  })
}

export function createGroupChat(title: string): Promise<Chat> {
  return apiClient<Chat>('/chats', {
    method: 'POST',
    body: JSON.stringify({ type: 'group', title }),
  })
}

export function updateChatTitle(chatId: number, title: string): Promise<Chat> {
  return apiClient<Chat>(`/chats/${chatId}`, {
    method: 'PATCH',
    body: JSON.stringify({ title }),
  })
}

export function getChatDisplayName(
  chat: ChatListItem,
  peerLogin?: string | null,
): string {
  if (chat.title) {
    return chat.title
  }
  if (chat.type === 'direct') {
    const login = peerLogin?.trim()
    return login || 'Личный чат'
  }
  return `Чат ${chat.id}`
}
