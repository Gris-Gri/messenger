import type { ChatMember } from '../types/domain'
import { apiClient } from './client'

export function fetchChatMembers(chatId: number): Promise<ChatMember[]> {
  return apiClient<ChatMember[]>(`/chats/${chatId}/members`)
}

export function addChatMember(chatId: number, userId: number): Promise<void> {
  return apiClient<void>(`/chats/${chatId}/members`, {
    method: 'POST',
    body: JSON.stringify({ user_id: userId }),
  })
}

export function removeChatMember(chatId: number, userId: number): Promise<void> {
  return apiClient<void>(`/chats/${chatId}/members/${userId}`, {
    method: 'DELETE',
  })
}
