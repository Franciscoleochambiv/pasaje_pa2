<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { loginWithGoogle } from '../api/client'

declare global {
  interface Window {
    google: any
  }
}

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const error = ref('')

function getRedirectPath(): string {
  const redirect = route.query.redirect as string
  if (redirect) {
    try { return decodeURIComponent(redirect) } catch { return redirect }
  }
  return '/viajes'
}

async function handleGoogleCallback(response: { credential: string }) {
  loading.value = true
  error.value = ''
  try {
    const result = await loginWithGoogle(response.credential)
    localStorage.setItem('customer_token', result.token)
    localStorage.setItem('customer_user', JSON.stringify(result.user))
    router.push(getRedirectPath())
  } catch (e: any) {
    error.value = e.message || 'Error al iniciar sesion con Google'
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  const clientId = import.meta.env.VITE_GOOGLE_CLIENT_ID || ''
  if (!clientId) {
    error.value = 'Google Client ID no configurado'
    return
  }

  // Wait for DOM to be fully rendered
  await nextTick()

  const initGoogle = () => {
    if (!window.google?.accounts?.id) return false
    const btnEl = document.getElementById('google-signin-btn')
    if (!btnEl) return false
    try {
      window.google.accounts.id.initialize({
        client_id: clientId,
        callback: handleGoogleCallback,
      })
      window.google.accounts.id.renderButton(btnEl, {
        theme: 'outline',
        size: 'large',
        width: 350,
        text: 'continue_with',
        locale: 'es',
      })
      return true
    } catch {
      return false
    }
  }

  // Retry until Google SDK loads (async script)
  if (!initGoogle()) {
    let attempts = 0
    const interval = setInterval(() => {
      attempts++
      if (initGoogle() || attempts > 30) {
        clearInterval(interval)
        if (attempts > 30) {
          error.value = 'No se pudo cargar Google Sign-In. Recarga la pagina.'
        }
      }
    }, 200)
  }
})
</script>

<template>
  <div class="customer-login-page">
    <div class="login-container">
      <!-- Decorative top bar -->
      <div class="card-accent"></div>

      <div class="login-card">
        <!-- Icon -->
        <div class="login-icon">
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M8 6v6"/>
            <path d="M15 6v6"/>
            <path d="M2 12h19.6"/>
            <path d="M18 18h3s.5-1.7.8-2.8c.1-.4.2-.8.2-1.2 0-.4-.1-.8-.2-1.2l-1.4-5C20.1 6.8 19.1 6 18 6H4a2 2 0 0 0-2 2v10h3"/>
            <circle cx="7" cy="18" r="2"/>
            <path d="M9 18h5"/>
            <circle cx="16" cy="18" r="2"/>
          </svg>
        </div>

        <h1 class="login-title">Inicia sesion para reservar tu pasaje</h1>
        <p class="login-subtitle">
          Accede con tu cuenta de Google para una experiencia rapida y segura
        </p>

        <!-- Error message -->
        <div v-if="error" class="alert alert-error">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" x2="12" y1="8" y2="12"/>
            <line x1="12" x2="12.01" y1="16" y2="16"/>
          </svg>
          {{ error }}
        </div>

        <!-- Loading spinner -->
        <div v-if="loading" class="loading-state">
          <div class="spinner"></div>
          <span>Iniciando sesion...</span>
        </div>

        <!-- Google Sign-In button container -->
        <div class="google-btn-wrapper">
          <div id="google-signin-btn"></div>
        </div>

        <!-- Divider -->
        <div class="divider">
          <span>Informacion</span>
        </div>

        <!-- Info note -->
        <div class="info-note">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/>
            <path d="M12 16v-4"/>
            <path d="M12 8h.01"/>
          </svg>
          <p>
            Al iniciar sesion, podras reservar y comprar pasajes, ver tu historial
            y recibir confirmaciones por email.
          </p>
        </div>

        <!-- Back link -->
        <div class="back-link">
          <router-link to="/viajes">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="m15 18-6-6 6-6"/>
            </svg>
            Volver a buscar viajes
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.customer-login-page {
  min-height: calc(100vh - var(--nav-height) - 80px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem 1rem;
  background:
    linear-gradient(135deg, var(--brand-50) 0%, var(--color-background) 50%, var(--accent-50) 100%);
}

.login-container {
  width: 100%;
  max-width: 440px;
  position: relative;
}

.card-accent {
  height: 5px;
  border-radius: var(--radius-lg) var(--radius-lg) 0 0;
  background: linear-gradient(90deg, var(--brand-500), var(--accent-500));
}

.login-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-top: none;
  border-radius: 0 0 var(--radius-lg) var(--radius-lg);
  padding: 2.5rem 2rem 2rem;
  box-shadow: var(--shadow-xl);
  text-align: center;
}

.login-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--brand-50), var(--accent-50));
  color: var(--brand-600);
  margin-bottom: 1.25rem;
}

.login-title {
  font-size: 1.35rem;
  font-weight: 800;
  color: var(--color-heading);
  letter-spacing: -0.02em;
  line-height: 1.3;
  margin-bottom: 0.5rem;
}

.login-subtitle {
  font-size: 0.9rem;
  color: var(--color-text-muted);
  line-height: 1.5;
  margin-bottom: 1.75rem;
}

.alert {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border-radius: var(--radius-md);
  font-size: 0.88rem;
  margin-bottom: 1rem;
  text-align: left;
}

.alert-error {
  background: var(--danger-50);
  color: var(--danger-700);
  border: 1px solid var(--danger-100);
}

.loading-state {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem 0;
  justify-content: center;
  color: var(--color-text-muted);
  font-size: 0.9rem;
}

.spinner {
  width: 22px;
  height: 22px;
  border: 3px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.google-btn-wrapper {
  display: flex;
  justify-content: center;
  margin-bottom: 1.5rem;
  min-height: 44px;
}

.divider {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1.25rem;
}

.divider::before,
.divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--color-border);
}

.divider span {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.info-note {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 1rem;
  background: var(--info-50);
  border: 1px solid var(--info-100);
  border-radius: var(--radius-md);
  text-align: left;
  margin-bottom: 1.5rem;
}

.info-note svg {
  flex-shrink: 0;
  color: var(--info-500);
  margin-top: 1px;
}

.info-note p {
  font-size: 0.85rem;
  color: var(--slate-600);
  line-height: 1.5;
}

.back-link {
  margin-top: 0.5rem;
}

.back-link a {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.88rem;
  font-weight: 500;
  color: var(--color-text-muted);
  text-decoration: none;
  transition: color 0.2s ease;
}

.back-link a:hover {
  color: var(--color-primary);
}

@media (prefers-color-scheme: dark) {
  .customer-login-page {
    background:
      linear-gradient(135deg, rgba(27, 85, 245, 0.05) 0%, var(--color-background) 50%, rgba(6, 201, 170, 0.05) 100%);
  }

  .info-note p {
    color: var(--slate-300);
  }
}

@media (max-width: 480px) {
  .login-card {
    padding: 2rem 1.25rem 1.5rem;
  }

  .login-title {
    font-size: 1.2rem;
  }
}
</style>
