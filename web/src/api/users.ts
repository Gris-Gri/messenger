import type { User } from '../types/domain'
import { apiClient } from './client'

export function searchUsers(login: string, limit = 20): Promise<User[]> {
  const params = new URLSearchParams({
    login,
    limit: String(limit),
  })
  return apiClient<User[]>(`/users/search?${params.toString()}`)
}
