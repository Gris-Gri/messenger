import { useEffect, useMemo, useState } from 'react'
import { fetchChatMembers } from '../api/members'
import { membersToUpserts, useUsers } from '../context/UsersContext'
import type { ChatType, User } from '../types/domain'

/**
 * Имена участников группы для ленты/поиска.
 * Источник истины — Users store; локально держим только список user_id чата.
 */
export function useMemberNames(
  chatId: number | null,
  chatType: ChatType | null,
  currentUser: User | null,
): Record<number, string> {
  const { users, upsertUsers } = useUsers()
  const [memberIds, setMemberIds] = useState<number[]>([])

  useEffect(() => {
    if (chatId === null || chatType !== 'group') {
      setMemberIds([])
      return
    }

    let cancelled = false

    fetchChatMembers(chatId)
      .then((members) => {
        if (cancelled) {
          return
        }
        upsertUsers(membersToUpserts(members))
        setMemberIds(members.map((member) => member.user_id))
      })
      .catch(() => {
        if (!cancelled) {
          setMemberIds([])
        }
      })

    return () => {
      cancelled = true
    }
  }, [chatId, chatType, upsertUsers])

  return useMemo(() => {
    if (chatType !== 'group') {
      return {}
    }

    const names: Record<number, string> = {}
    for (const id of memberIds) {
      const login = users[id]?.login
      if (login) {
        names[id] = login
      }
    }
    if (currentUser) {
      names[currentUser.id] = users[currentUser.id]?.login || currentUser.login
    }
    return names
  }, [chatType, currentUser, memberIds, users])
}
