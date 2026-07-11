import { type FormEvent, useState } from 'react'
import { mapAuthError, useAuth } from '../../context/AuthContext'
import styles from './Login.module.css'

type Mode = 'login' | 'register'

export function Login() {
  const { login, register } = useAuth()
  const [mode, setMode] = useState<Mode>('login')
  const [loginValue, setLoginValue] = useState('')
  const [password, setPassword] = useState('')
  const [errors, setErrors] = useState<{ login?: string; password?: string }>({})
  const [submitting, setSubmitting] = useState(false)

  function validate(): boolean {
    const next: { login?: string; password?: string } = {}
    if (!loginValue.trim()) next.login = 'Введите логин'
    if (!password.trim()) next.password = 'Введите пароль'
    setErrors(next)
    return Object.keys(next).length === 0
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!validate()) return

    setSubmitting(true)
    setErrors({})

    try {
      const trimmedLogin = loginValue.trim()
      if (mode === 'login') {
        await login(trimmedLogin, password)
      } else {
        await register(trimmedLogin, password)
      }
    } catch (err) {
      setErrors(mapAuthError(err, mode))
    } finally {
      setSubmitting(false)
    }
  }

  function switchMode(next: Mode) {
    setMode(next)
    setErrors({})
  }

  return (
    <div className={styles.screen}>
      <div className={styles.card}>
        <h1 className={styles.title}>
          {mode === 'login' ? 'Вход' : 'Регистрация'}
        </h1>
        <p className={styles.subtitle}>
          {mode === 'login'
            ? 'Войдите в аккаунт мессенджера'
            : 'Создайте новый аккаунт'}
        </p>

        <form className={styles.form} onSubmit={handleSubmit} noValidate>
          <div className={styles.field}>
            <label className={styles.label} htmlFor="login">
              Логин
            </label>
            <input
              id="login"
              className={`${styles.input} ${errors.login ? styles.inputError : ''}`}
              type="text"
              autoComplete="username"
              value={loginValue}
              onChange={(e) => setLoginValue(e.target.value)}
              disabled={submitting}
            />
            {errors.login && <p className={styles.error}>{errors.login}</p>}
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="password">
              Пароль
            </label>
            <input
              id="password"
              className={`${styles.input} ${errors.password ? styles.inputError : ''}`}
              type="password"
              autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={submitting}
            />
            {errors.password && <p className={styles.error}>{errors.password}</p>}
          </div>

          <button type="submit" className={styles.submit} disabled={submitting}>
            {submitting
              ? 'Подождите…'
              : mode === 'login'
                ? 'Войти'
                : 'Зарегистрироваться'}
          </button>
        </form>

        <p className={styles.switch}>
          {mode === 'login' ? 'Нет аккаунта?' : 'Уже есть аккаунт?'}{' '}
          <button
            type="button"
            className={styles.switchBtn}
            onClick={() => switchMode(mode === 'login' ? 'register' : 'login')}
            disabled={submitting}
          >
            {mode === 'login' ? 'Зарегистрироваться' : 'Войти'}
          </button>
        </p>
      </div>
    </div>
  )
}
