<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { login } from '../api/client'
import { useAuth } from '../stores/auth'

const router = useRouter()
const route = useRoute()
const { setAuth, isAuthenticated } = useAuth()

const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

// If already authenticated, redirect
if (isAuthenticated.value) {
  router.replace('/admin')
}

async function handleLogin() {
  if (!email.value.trim() || !password.value.trim()) {
    error.value = 'Por favor ingresa tu email y contrasena.'
    return
  }
  loading.value = true
  error.value = ''
  try {
    const result = await login(email.value, password.value)
    setAuth(result.token, result.user)
    const redirect = (route.query.redirect as string) || '/admin'
    router.replace(redirect)
  } catch (e) {
    error.value = 'Credenciales invalidas. Verifica tu email y contrasena.'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-header">
        <div class="login-logo">
          <div class="login-logo-icon">
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M8 6v6"/>
              <path d="M15 6v6"/>
              <path d="M2 12h19.6"/>
              <path d="M18 18h3s.5-1.7.8-2.8c.1-.4.2-.8.2-1.2 0-.4-.1-.8-.2-1.2l-1.4-5C20.1 6.8 19.1 6 18 6H4a2 2 0 0 0-2 2v10h3"/>
              <circle cx="7" cy="18" r="2"/>
              <path d="M9 18h5"/>
              <circle cx="16" cy="18" r="2"/>
            </svg>
          </div>
          <h1 class="login-brand">Pasaje</h1>
        </div>
        <p class="login-subtitle">Panel de Administracion</p>
      </div>

      <form class="login-form" @submit.prevent="handleLogin">
        <div v-if="error" class="login-error">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" x2="12" y1="8" y2="12"/>
            <line x1="12" x2="12.01" y1="16" y2="16"/>
          </svg>
          <span>{{ error }}</span>
        </div>

        <div class="form-group">
          <label class="form-label" for="login-email">Correo electronico</label>
          <input
            id="login-email"
            v-model="email"
            type="email"
            class="form-input"
            placeholder="admin@ejemplo.com"
            autocomplete="email"
            autofocus
          />
        </div>

        <div class="form-group">
          <label class="form-label" for="login-password">Contrasena</label>
          <input
            id="login-password"
            v-model="password"
            type="password"
            class="form-input"
            placeholder="Tu contrasena"
            autocomplete="current-password"
          />
        </div>

        <button type="submit" class="login-btn" :disabled="loading">
          <div v-if="loading" class="spinner-sm"></div>
          <span v-else>Iniciar Sesion</span>
        </button>
      </form>

      <div class="login-footer">
        <router-link to="/" class="login-back">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="m12 19-7-7 7-7"/>
            <path d="M19 12H5"/>
          </svg>
          Volver al sitio
        </router-link>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--brand-50) 0%, var(--brand-100) 50%, var(--slate-100) 100%);
  padding: 1.5rem;
}

@media (prefers-color-scheme: dark) {
  .login-page {
    background: linear-gradient(135deg, var(--slate-950) 0%, var(--slate-900) 50%, var(--brand-950) 100%);
  }
}

.login-card {
  width: 100%;
  max-width: 420px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-xl);
  padding: 2.5rem 2rem;
}

.login-header {
  text-align: center;
  margin-bottom: 2rem;
}

.login-logo {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  margin-bottom: 0.5rem;
}

.login-logo-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  background: linear-gradient(135deg, var(--brand-600), var(--brand-700));
  color: white;
}

.login-brand {
  font-size: 1.6rem;
  font-weight: 800;
  color: var(--color-heading);
  letter-spacing: -0.03em;
}

.login-subtitle {
  font-size: 0.9rem;
  color: var(--color-text-muted);
  margin-top: 0.25rem;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 1.15rem;
}

.login-error {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.75rem 1rem;
  border-radius: var(--radius-md);
  background: var(--danger-50);
  color: var(--danger-700);
  border: 1px solid var(--danger-100);
  font-size: 0.88rem;
}

.login-error svg {
  flex-shrink: 0;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.form-label {
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--color-heading);
}

.form-input {
  padding: 0.65rem 0.9rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-background);
  color: var(--color-text);
  font-size: 0.92rem;
  font-family: inherit;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
  outline: none;
}

.form-input:focus {
  border-color: var(--color-border-focus);
  box-shadow: 0 0 0 3px rgba(27, 85, 245, 0.1);
}

.form-input::placeholder {
  color: var(--color-text-muted);
  opacity: 0.6;
}

.login-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  width: 100%;
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: var(--radius-sm);
  background: linear-gradient(135deg, var(--brand-600), var(--brand-700));
  color: white;
  font-size: 0.95rem;
  font-weight: 600;
  font-family: inherit;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: var(--shadow-sm);
  margin-top: 0.25rem;
}

.login-btn:hover:not(:disabled) {
  background: linear-gradient(135deg, var(--brand-500), var(--brand-600));
  box-shadow: var(--shadow-md);
}

.login-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.spinner-sm {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.login-footer {
  text-align: center;
  margin-top: 1.75rem;
  padding-top: 1.25rem;
  border-top: 1px solid var(--color-border);
}

.login-back {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.85rem;
  color: var(--color-text-muted);
  text-decoration: none;
  transition: color 0.2s ease;
}

.login-back:hover {
  color: var(--color-primary);
}

@media (max-width: 480px) {
  .login-card {
    padding: 2rem 1.5rem;
    border-radius: var(--radius-lg);
  }
}
</style>
