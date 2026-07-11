import { useEffect, useMemo, useState } from 'react'
import { fetchChatMembers } from '../api/members'
import type { ChatType, User } from '../types/domain'

export function useMemberNames(
  chatId: number | null,
  chatType: ChatType | null,
  currentUser: User | null,
): Record<number, string> {
  const [fetchedNames, setFetchedNames] = useState<Record<number, string>>({})

  useEffect(() => {
    if (chatId === null || chatType !== 'group') {
      setFetchedNames({})
      return
    }

    let cancelled = false

    fetchChatMembers(chatId)
      .then((members) => {
        if (cancelled) {
          return
        }

        const next: Record<number, string> = {}
        for (const member of members) {
          next[member.user_id] = member.login
        }
        setFetchedNames(next)
      })
      .catch(() => {
        if (!cancelled) {
          setFetchedNames({})
        }
      })

    return () => {
      cancelled = true
    }
  }, [chatId, chatType])

  return useMemo(() => {
    if (chatType !== 'group') {
      return {}
    }

    const names = { ...fetchedNames }
    if (currentUser) {
      names[currentUser.id] = currentUser.login
    }
    return names
  }, [chatType, currentUser, fetchedNames])
}
