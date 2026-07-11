import type { Message } from '../types/domain'
import { apiClient } from './client'

export function searchMessages(chatId: number, query: string): Promise<Message[]> {
  const params = new URLSearchParams({ q: query })
  return apiClient<Message[]>(`/chats/${chatId}/search?${params.toString()}`)
}
