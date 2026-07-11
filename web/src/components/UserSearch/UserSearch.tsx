import { useEffect, useMemo, useState } from 'react'
import { searchUsers } from '../../api/users'
import type { User } from '../../types/domain'
import styles from './UserSearch.module.css'

const DEBOUNCE_MS = 300
const MIN_QUERY_LEN = 2

type UserSearchProps = {
  onSelect: (user: User) => void | Promise<void>
  disabled?: boolean
  excludeUserIds?: readonly number[]
  placeholder?: string
  autoFocus?: boolean
}

export function UserSearch({
  onSelect,
  disabled = false,
  excludeUserIds = [],
  placeholder = 'Логин…',
  autoFocus = false,
}: UserSearchProps) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<User[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [selectingId, setSelectingId] = useState<number | null>(null)

  const excludeKey = useMemo(
    () => [...excludeUserIds].sort((a, b) => a - b).join(','),
    [excludeUserIds],
  )

  useEffect(() => {
    const trimmed = query.trim()
    if (trimmed.length < MIN_QUERY_LEN) {
      setResults([])
      setError(null)
      setLoading(false)
      return
    }

    let cancelled = false
    setLoading(true)
    setError(null)

    const excluded = new Set(
      excludeKey
        ? excludeKey.split(',').map((id) => Number.parseInt(id, 10))
        : [],
    )

    const timer = window.setTimeout(() => {
      searchUsers(trimmed, 20)
        .then((users) => {
          if (!cancelled) {
            setResults(users.filter((user) => !excluded.has(user.id)))
          }
        })
        .catch((err: unknown) => {
          if (!cancelled) {
            setResults([])
            setError(err instanceof Error ? err.message : 'Ошибка поиска')
          }
        })
        .finally(() => {
          if (!cancelled) {
            setLoading(false)
          }
        })
    }, DEBOUNCE_MS)

    return () => {
      cancelled = true
      window.clearTimeout(timer)
    }
  }, [excludeKey, query])

  const trimmed = query.trim()
  const showResults = trimmed.length >= MIN_QUERY_LEN

  const handleSelect = async (user: User) => {
    setSelectingId(user.id)
    try {
      await onSelect(user)
      setQuery('')
      setResults([])
    } finally {
      setSelectingId(null)
    }
  }

  return (
    <div className={styles.userSearch}>
      <input
        className={styles.input}
        type="search"
        placeholder={placeholder}
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        disabled={disabled || selectingId !== null}
        autoFocus={autoFocus}
        aria-label={placeholder}
      />

      {trimmed.length > 0 && trimmed.length < MIN_QUERY_LEN && (
        <p className={styles.hint}>Введите минимум {MIN_QUERY_LEN} символа</p>
      )}

      {error && <div className={styles.error}>{error}</div>}

      {showResults && (
        <ul className={styles.results} role="listbox">
          {loading && <li className={styles.stateMessage}>Поиск…</li>}
          {!loading && results.length === 0 && !error && (
            <li className={styles.stateMessage}>Никого не найдено</li>
          )}
          {!loading &&
            results.map((user) => (
              <li key={user.id}>
                <button
                  type="button"
                  className={styles.resultItem}
                  role="option"
                  disabled={disabled || selectingId !== null}
                  onClick={() => void handleSelect(user)}
                >
                  {user.login}
                </button>
              </li>
            ))}
        </ul>
      )}
    </div>
  )
}
