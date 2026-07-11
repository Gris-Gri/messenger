import type { ChatListItem } from '../types/domain'
import { apiClient } from './client'

export function fetchChats(): Promise<ChatListItem[]> {
  return apiClient<ChatListItem[]>('/chats')
}

export function getChatDisplayName(
  chat: ChatListItem,
  peerNames: Record<number, string> = {},
): string {
  if (chat.title) {
    return chat.title
  }
  if (chat.type === 'direct') {
    return peerNames[chat.id] ?? 'Личный чат'
  }
  return `Чат ${chat.id}`
}
