import { useState, type FormEvent } from 'react'
import { useAuth } from '../../context/AuthContext'
import { useChatMembers } from '../../hooks/useChatMembers'
import styles from './MembersPanel.module.css'

type MembersPanelProps = {
  chatId: number
  open: boolean
  onClose: () => void
}

export function MembersPanel({ chatId, open, onClose }: MembersPanelProps) {
  const { currentUser } = useAuth()
  const {
    members,
    loading,
    error,
    isAdmin,
    actionError,
    actionLoading,
    addMember,
    removeMember,
  } = useChatMembers(chatId, currentUser?.id ?? null, open)
  const [userIdInput, setUserIdInput] = useState('')

  if (!open) {
    return null
  }

  const handleAdd = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    const userId = Number.parseInt(userIdInput.trim(), 10)
    if (!Number.isFinite(userId) || userId <= 0) {
      return
    }

    const ok = await addMember(userId)
    if (ok) {
      setUserIdInput('')
    }
  }

  const bannerError = actionError ?? error

  return (
    <aside className={styles.membersPanel} aria-label="Участники чата">
      <div className={styles.header}>
        <h2 className={styles.title}>Участники</h2>
        <button
          type="button"
          className={styles.closeBtn}
          aria-label="Закрыть"
          onClick={onClose}
        >
          ×
        </button>
      </div>

      {bannerError && <div className={styles.errorBanner}>{bannerError}</div>}

      {isAdmin && (
        <form className={styles.addForm} onSubmit={(event) => void handleAdd(event)}>
          <input
            className={styles.addInput}
            type="text"
            inputMode="numeric"
            placeholder="user_id"
            value={userIdInput}
            onChange={(event) => setUserIdInput(event.target.value)}
            disabled={actionLoading}
          />
          <button
            type="submit"
            className={styles.addBtn}
            disabled={actionLoading || !userIdInput.trim()}
          >
            Добавить
          </button>
        </form>
      )}

      <ul className={styles.memberList}>
        {loading && members.length === 0 && (
          <li className={styles.stateMessage}>Загрузка…</li>
        )}
        {!loading && members.length === 0 && !error && (
          <li className={styles.stateMessage}>Нет участников</li>
        )}
        {members.map((member) => (
          <li key={member.user_id} className={styles.memberItem}>
            <div className={styles.memberInfo}>
              <span className={styles.memberLogin}>{member.login}</span>
              <span className={styles.memberRole}>{member.role}</span>
            </div>
            {isAdmin && member.user_id !== currentUser?.id && (
              <button
                type="button"
                className={styles.removeBtn}
                disabled={actionLoading}
                onClick={() => void removeMember(member.user_id)}
              >
                Удалить
              </button>
            )}
          </li>
        ))}
      </ul>
    </aside>
  )
}
