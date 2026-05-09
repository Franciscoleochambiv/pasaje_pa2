import { reactive, computed } from 'vue'

interface User {
  id: number
  email: string
  name: string
  role: string
}

interface AuthState {
  token: string | null
  user: User | null
}

const state = reactive<AuthState>({
  token: localStorage.getItem('auth_token'),
  user: JSON.parse(localStorage.getItem('auth_user') || 'null'),
})

export function useAuth() {
  const isAuthenticated = computed(() => !!state.token)
  const user = computed(() => state.user)
  const token = computed(() => state.token)

  function setAuth(token: string, user: User) {
    state.token = token
    state.user = user
    localStorage.setItem('auth_token', token)
    localStorage.setItem('auth_user', JSON.stringify(user))
  }

  function logout() {
    state.token = null
    state.user = null
    localStorage.removeItem('auth_token')
    localStorage.removeItem('auth_user')
  }

  return { isAuthenticated, user, token, setAuth, logout }
}
