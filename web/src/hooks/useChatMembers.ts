import { useCallback, useEffect, useMemo, useState } from 'react'
import { addChatMember, fetchChatMembers, removeChatMember } from '../api/members'
import { ApiError } from '../api/errors'
import { membersToUpserts, useUsers } from '../context/UsersContext'
import type { ChatMember } from '../types/domain'

function forbiddenMessage(err: unknown): string | null {
  if (err instanceof ApiError && err.status === 403) {
    return err.message || 'Недостаточно прав'
  }
  return null
}

export function useChatMembers(
  chatId: number | null,
  currentUserId: number | null,
  enabled: boolean,
) {
  const { upsertUsers } = useUsers()
  const [members, setMembers] = useState<ChatMember[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [actionLoading, setActionLoading] = useState(false)

  const isAdmin = useMemo(
    () => members.some((member) => member.user_id === currentUserId && member.role === 'admin'),
    [currentUserId, members],
  )

  const refresh = useCallback(async () => {
    if (chatId === null) {
      setMembers([])
      return
    }

    setLoading(true)
    setError(null)

    try {
      const next = await fetchChatMembers(chatId)
      upsertUsers(membersToUpserts(next))
      setMembers(next)
    } catch (err: unknown) {
      setMembers([])
      setError(err instanceof Error ? err.message : 'Не удалось загрузить участников')
    } finally {
      setLoading(false)
    }
  }, [chatId, upsertUsers])

  useEffect(() => {
    if (!enabled || chatId === null) {
      return
    }

    void refresh()
  }, [chatId, enabled, refresh])

  const addMember = useCallback(
    async (userId: number) => {
      if (chatId === null) {
        return false
      }

      setActionLoading(true)
      setActionError(null)

      try {
        await addChatMember(chatId, userId)
        await refresh()
        return true
      } catch (err: unknown) {
        setActionError(
          forbiddenMessage(err) ??
            (err instanceof Error ? err.message : 'Не удалось добавить участника'),
        )
        return false
      } finally {
        setActionLoading(false)
      }
    },
    [chatId, refresh],
  )

  const removeMember = useCallback(
    async (userId: number) => {
      if (chatId === null) {
        return false
      }

      setActionLoading(true)
      setActionError(null)

      try {
        await removeChatMember(chatId, userId)
        await refresh()
        return true
      } catch (err: unknown) {
        setActionError(
          forbiddenMessage(err) ??
            (err instanceof Error ? err.message : 'Не удалось удалить участника'),
        )
        return false
      } finally {
        setActionLoading(false)
      }
    },
    [chatId, refresh],
  )

  return {
    members,
    loading,
    error,
    isAdmin,
    actionError,
    actionLoading,
    addMember,
    removeMember,
    refresh,
  }
}
