import type { Message, ReactionSummary, ReactionType } from '../types/domain'
import { apiClient } from './client'

const DEFAULT_LIMIT = 50

export type FetchMessagesParams = {
  beforeId?: number
  limit?: number
}

export function fetchMessages(
  chatId: number,
  params: FetchMessagesParams = {},
): Promise<Message[]> {
  const limit = params.limit ?? DEFAULT_LIMIT
  const search = new URLSearchParams({ limit: String(limit) })

  if (params.beforeId !== undefined && params.beforeId > 0) {
    search.set('before_id', String(params.beforeId))
  }

  return apiClient<Message[]>(`/chats/${chatId}/messages?${search.toString()}`)
}

export function editMessage(
  chatId: number,
  messageId: number,
  body: string,
): Promise<Message> {
  return apiClient<Message>(`/chats/${chatId}/messages/${messageId}`, {
    method: 'PATCH',
    body: JSON.stringify({ body }),
  })
}

export function setMessageReaction(
  chatId: number,
  messageId: number,
  reaction: ReactionType,
): Promise<ReactionSummary> {
  return apiClient<ReactionSummary>(
    `/chats/${chatId}/messages/${messageId}/reactions`,
    {
      method: 'POST',
      body: JSON.stringify({ reaction }),
    },
  )
}
