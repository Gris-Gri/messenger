import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { useAuth } from './AuthContext'
import type { CachedUser, ChatMember } from '../types/domain'

export type UserUpsert = {
  user_id: number
  login?: string
  online?: boolean
  last_seen_at?: string | null
}

type UsersContextValue = {
  users: Record<number, CachedUser>
  upsertUsers: (entries: UserUpsert[]) => void
  updateLogin: (userId: number, login: string) => void
  updatePresence: (
    userId: number,
    status: 'online' | 'offline',
    lastSeenAt?: string | null,
  ) => void
  getLogin: (userId: number, fallback?: string) => string
}

const UsersContext = createContext<UsersContextValue | null>(null)

function mergeUser(prev: CachedUser | undefined, patch: UserUpsert): CachedUser {
  return {
    login: patch.login?.trim() || prev?.login || '',
    online: patch.online ?? prev?.online ?? false,
    last_seen_at:
      patch.last_seen_at !== undefined
        ? patch.last_seen_at
        : (prev?.last_seen_at ?? null),
  }
}

export function UsersProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated, currentUser } = useAuth()
  const [users, setUsers] = useState<Record<number, CachedUser>>({})

  useEffect(() => {
    if (!isAuthenticated) {
      setUsers({})
      return
    }
    if (currentUser) {
      setUsers((prev) => ({
        ...prev,
        [currentUser.id]: mergeUser(prev[currentUser.id], {
          user_id: currentUser.id,
          login: currentUser.login,
        }),
      }))
    }
  }, [currentUser, isAuthenticated])

  const upsertUsers = useCallback((entries: UserUpsert[]) => {
    if (entries.length === 0) {
      return
    }
    setUsers((prev) => {
      const next = { ...prev }
      for (const entry of entries) {
        next[entry.user_id] = mergeUser(prev[entry.user_id], entry)
      }
      return next
    })
  }, [])

  const updateLogin = useCallback((userId: number, login: string) => {
    const trimmed = login.trim()
    if (!trimmed) {
      return
    }
    setUsers((prev) => ({
      ...prev,
      [userId]: mergeUser(prev[userId], { user_id: userId, login: trimmed }),
    }))
  }, [])

  const updatePresence = useCallback(
    (userId: number, status: 'online' | 'offline', lastSeenAt?: string | null) => {
      setUsers((prev) => ({
        ...prev,
        [userId]: mergeUser(prev[userId], {
          user_id: userId,
          online: status === 'online',
          last_seen_at:
            status === 'offline'
              ? (lastSeenAt ?? prev[userId]?.last_seen_at ?? null)
              : prev[userId]?.last_seen_at ?? null,
        }),
      }))
    },
    [],
  )

  const getLogin = useCallback(
    (userId: number, fallback = '') => users[userId]?.login || fallback,
    [users],
  )

  const value = useMemo(
    () => ({
      users,
      upsertUsers,
      updateLogin,
      updatePresence,
      getLogin,
    }),
    [getLogin, updateLogin, updatePresence, upsertUsers, users],
  )

  return <UsersContext.Provider value={value}>{children}</UsersContext.Provider>
}

export function useUsers(): UsersContextValue {
  const ctx = useContext(UsersContext)
  if (!ctx) {
    throw new Error('useUsers must be used within UsersProvider')
  }
  return ctx
}

export function membersToUpserts(members: ChatMember[]): UserUpsert[] {
  return members.map((member) => ({
    user_id: member.user_id,
    login: member.login,
    online: member.online,
    last_seen_at: member.last_seen_at ?? null,
  }))
}
