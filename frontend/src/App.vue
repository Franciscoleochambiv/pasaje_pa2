<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuth } from './stores/auth'

const router = useRouter()
const route = useRoute()
const { isAuthenticated, user, logout } = useAuth()

const mobileMenuOpen = ref(false)
const scrolled = ref(false)
const scrollProgress = ref(0)

// Always dark navbar on pages other than home
const isHome = computed(() => route.path === '/')

// Customer auth state
const customerUser = ref<{ name: string; email: string; role: string } | null>(null)
const isCustomerLoggedIn = computed(() => {
  return customerUser.value !== null && customerUser.value.role === 'customer'
})

function loadCustomerAuth() {
  const token = localStorage.getItem('customer_token')
  const userStr = localStorage.getItem('customer_user')
  if (token && userStr) {
    try {
      customerUser.value = JSON.parse(userStr)
    } catch {
      customerUser.value = null
    }
  } else {
    customerUser.value = null
  }
}

function handleCustomerLogout() {
  localStorage.removeItem('customer_token')
  localStorage.removeItem('customer_user')
  customerUser.value = null
  mobileMenuOpen.value = false
  router.push('/')
}

function onScroll() {
  const y = window.scrollY
  scrolled.value = y > 40
  const docHeight = document.documentElement.scrollHeight - window.innerHeight
  scrollProgress.value = docHeight > 0 ? (y / docHeight) * 100 : 0
}

onMounted(() => {
  loadCustomerAuth()
  onScroll()
  window.addEventListener('scroll', onScroll, { passive: true })
})

// Force dark navbar on non-home pages immediately
watch(() => route.path, () => {
  if (!isHome.value) {
    scrolled.value = true
  } else {
    onScroll()
  }
}, { immediate: true })

onUnmounted(() => {
  window.removeEventListener('scroll', onScroll)
})

// Reload customer auth on every route change (e.g. after login redirect)
watch(() => route.fullPath, loadCustomerAuth)

function toggleMenu() {
  mobileMenuOpen.value = !mobileMenuOpen.value
}

function handleLogout() {
  logout()
  mobileMenuOpen.value = false
  router.push('/login')
}
</script>

<template>
  <div class="app-layout">
    <!-- Navigation — Smart glassmorphism -->
    <header class="navbar" :class="{ 'navbar--scrolled': scrolled || !isHome, 'navbar--menu-open': mobileMenuOpen }">
      <!-- Scroll progress bar -->
      <div class="scroll-progress" :style="`width: ${scrollProgress}%`"></div>

      <div class="navbar-inner container">
        <router-link to="/" class="navbar-brand">
          <div class="brand-icon">
            <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M8 6v6"/>
              <path d="M15 6v6"/>
              <path d="M2 12h19.6"/>
              <path d="M18 18h3s.5-1.7.8-2.8c.1-.4.2-.8.2-1.2 0-.4-.1-.8-.2-1.2l-1.4-5C20.1 6.8 19.1 6 18 6H4a2 2 0 0 0-2 2v10h3"/>
              <circle cx="7" cy="18" r="2"/>
              <path d="M9 18h5"/>
              <circle cx="16" cy="18" r="2"/>
            </svg>
          </div>
          <div class="brand-text">
            <span class="brand-name">Pasaje</span>
            <span class="brand-tagline">Transporte Interprovincial</span>
          </div>
        </router-link>

        <nav class="navbar-nav" :class="{ open: mobileMenuOpen }">
          <router-link to="/" class="nav-link" @click="mobileMenuOpen = false">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M15 21v-8a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v8"/>
              <path d="M3 10a2 2 0 0 1 .709-1.528l7-5.999a2 2 0 0 1 2.582 0l7 5.999A2 2 0 0 1 21 10v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
            </svg>
            Inicio
          </router-link>
          <router-link to="/viajes" class="nav-link" @click="mobileMenuOpen = false">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M9 18l6-6-6-6"/>
              <path d="M3 12h13"/>
              <path d="M21 5v14"/>
            </svg>
            Buscar Viajes
          </router-link>
          <router-link to="/consulta" class="nav-link" @click="mobileMenuOpen = false">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="11" cy="11" r="8"/>
              <path d="m21 21-4.3-4.3"/>
            </svg>
            Consultar Reserva
          </router-link>
          <router-link to="/tracking" class="nav-link" @click="mobileMenuOpen = false">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M16.5 9.4 7.55 4.24"/>
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
              <path d="m3.3 7 8.7 5 8.7-5"/>
              <path d="M12 22V12"/>
            </svg>
            Rastrear Encomienda
          </router-link>
          <router-link to="/admin" class="nav-link" @click="mobileMenuOpen = false">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/>
              <circle cx="12" cy="12" r="3"/>
            </svg>
            Admin
          </router-link>
          <template v-if="isAuthenticated">
            <div class="nav-divider"></div>
            <span class="nav-user-label">{{ user?.name }}</span>
            <button class="nav-link nav-logout-btn" @click="handleLogout">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
                <polyline points="16 17 21 12 16 7"/>
                <line x1="21" x2="9" y1="12" y2="12"/>
              </svg>
              Salir
            </button>
          </template>
          <template v-if="isCustomerLoggedIn && !isAuthenticated">
            <div class="nav-divider"></div>
            <span class="nav-user-label nav-customer-label">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/>
                <circle cx="12" cy="7" r="4"/>
              </svg>
              {{ customerUser?.name }}
            </span>
            <button class="nav-link nav-logout-btn" @click="handleCustomerLogout">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
                <polyline points="16 17 21 12 16 7"/>
                <line x1="21" x2="9" y1="12" y2="12"/>
              </svg>
              Salir
            </button>
          </template>
        </nav>

        <button class="mobile-toggle" @click="toggleMenu" aria-label="Menu">
          <svg v-if="!mobileMenuOpen" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="4" x2="20" y1="12" y2="12"/>
            <line x1="4" x2="20" y1="6" y2="6"/>
            <line x1="4" x2="20" y1="18" y2="18"/>
          </svg>
          <svg v-else width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M18 6 6 18"/>
            <path d="m6 6 12 12"/>
          </svg>
        </button>
      </div>
    </header>

    <!-- Main Content -->
    <main class="main-content">
      <router-view />
    </main>

    <!-- Footer — Premium dark -->
    <footer class="footer">
      <div class="footer-glow"></div>
      <div class="container">
        <div class="footer-grid">
          <!-- Brand column -->
          <div class="footer-col footer-brand-col">
            <router-link to="/" class="footer-brand">
              <div class="footer-brand-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M8 6v6"/><path d="M15 6v6"/><path d="M2 12h19.6"/>
                  <path d="M18 18h3s.5-1.7.8-2.8c.1-.4.2-.8.2-1.2 0-.4-.1-.8-.2-1.2l-1.4-5C20.1 6.8 19.1 6 18 6H4a2 2 0 0 0-2 2v10h3"/>
                  <circle cx="7" cy="18" r="2"/><path d="M9 18h5"/><circle cx="16" cy="18" r="2"/>
                </svg>
              </div>
              <span class="footer-brand-name">Pasaje</span>
            </router-link>
            <p class="footer-desc">
              Sistema de reserva y venta de pasajes interprovinciales.
              Viaja seguro, comodo y sin complicaciones.
            </p>
            <div class="footer-socials">
              <a href="#" class="social-link" aria-label="Facebook">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M18 2h-3a5 5 0 0 0-5 5v3H7v4h3v8h4v-8h3l1-4h-4V7a1 1 0 0 1 1-1h3z"/></svg>
              </a>
              <a href="#" class="social-link" aria-label="Instagram">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="20" height="20" x="2" y="2" rx="5" ry="5"/><path d="M16 11.37A4 4 0 1 1 12.63 8 4 4 0 0 1 16 11.37z"/><line x1="17.5" x2="17.51" y1="6.5" y2="6.5"/></svg>
              </a>
              <a href="#" class="social-link" aria-label="WhatsApp">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"/></svg>
              </a>
            </div>
          </div>

          <!-- Links column -->
          <div class="footer-col">
            <h4 class="footer-heading">Servicios</h4>
            <ul class="footer-links">
              <li><router-link to="/viajes">Buscar Viajes</router-link></li>
              <li><router-link to="/consulta">Consultar Reserva</router-link></li>
              <li><router-link to="/tracking">Rastrear Encomienda</router-link></li>
              <li><router-link to="/login/cliente">Area de Cliente</router-link></li>
            </ul>
          </div>

          <!-- Admin column -->
          <div class="footer-col">
            <h4 class="footer-heading">Administracion</h4>
            <ul class="footer-links">
              <li><router-link to="/admin">Dashboard</router-link></li>
              <li><router-link to="/admin/venta">Punto de Venta</router-link></li>
              <li><router-link to="/login">Iniciar Sesion</router-link></li>
            </ul>
          </div>

          <!-- Contact column -->
          <div class="footer-col">
            <h4 class="footer-heading">Contacto</h4>
            <ul class="footer-links">
              <li class="footer-contact-item">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0 1 22 16.92z"/></svg>
                +51 999 888 777
              </li>
              <li class="footer-contact-item">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="20" height="16" x="2" y="4" rx="2"/><path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"/></svg>
                hola@pasaje.pe
              </li>
              <li class="footer-contact-item">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z"/><circle cx="12" cy="10" r="3"/></svg>
                Arequipa y Cusco, Peru
              </li>
            </ul>
          </div>
        </div>

        <div class="footer-bottom">
          <p>&copy; 2026 Pasaje — Todos los derechos reservados.</p>
          <p class="footer-credit">Hecho con pasion para el transporte peruano.</p>
        </div>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: var(--color-background-soft);
}

/* ═══════════════════════════════════════════════════════
   NAVBAR — Smart glassmorphism
   ═══════════════════════════════════════════════════════ */
.navbar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 1000;
  height: var(--nav-height);
  display: flex;
  align-items: center;
  transition: all 0.5s cubic-bezier(0.22, 1, 0.36, 1);
  border-bottom: 1px solid transparent;
}

/* Navbar siempre oscuro semi-transparente */
.navbar {
  background: rgba(10, 14, 26, 0.75);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
  border-bottom: 1px solid rgba(255,255,255,0.06);
}

/* En home sin scroll: mas transparente para ver el hero */
.navbar:not(.navbar--scrolled) {
  background: rgba(10, 14, 26, 0.45);
}

/* Al scrollear: mas opaco */
.navbar.navbar--scrolled {
  background: rgba(10, 14, 26, 0.92);
  box-shadow: 0 8px 32px rgba(0,0,0,0.3);
}

/* Colores siempre blancos */
.navbar .navbar-brand {
  color: #ffffff;
}

.navbar .brand-name {
  color: #ffffff;
}

.navbar .brand-tagline {
  color: rgba(255,255,255,0.6);
}

.navbar .nav-link {
  color: rgba(255,255,255,0.75);
}

.navbar .nav-link:hover {
  color: #ffffff;
  background: rgba(255,255,255,0.1);
}

.navbar .mobile-toggle {
  color: #ffffff;
}

/* Scroll progress bar */
.scroll-progress {
  position: absolute;
  bottom: 0;
  left: 0;
  height: 2px;
  background: linear-gradient(90deg, #3B82F6, #60A5FA, #34D399);
  transition: width 0.1s linear;
  z-index: 1001;
}

.navbar-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  height: 100%;
  gap: 2rem;
}

.navbar-brand {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  text-decoration: none;
  flex-shrink: 0;
  transition: color 0.3s ease;
}

.brand-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  background: linear-gradient(135deg, #3B82F6, #60A5FA);
  color: white;
  box-shadow: 0 4px 14px rgba(59,130,246,0.3);
  transition: all 0.3s ease;
}

.navbar-brand:hover .brand-icon {
  transform: scale(1.05) rotate(-3deg);
  box-shadow: 0 6px 20px rgba(59,130,246,0.4);
}

.brand-text {
  display: flex;
  flex-direction: column;
  line-height: 1.2;
}

.brand-name {
  font-size: 1.2rem;
  font-weight: 800;
  letter-spacing: -0.03em;
  transition: color 0.3s ease;
}

.brand-tagline {
  font-size: 0.68rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  transition: color 0.3s ease;
}

.navbar-nav {
  display: flex;
  align-items: center;
  gap: 0.15rem;
}

.nav-link {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.45rem 0.85rem;
  border-radius: var(--radius-md);
  font-size: 0.85rem;
  font-weight: 500;
  text-decoration: none;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  border: none;
  background: none;
  cursor: pointer;
  font-family: inherit;
  position: relative;
}

.nav-link svg {
  opacity: 0.8;
  transition: opacity 0.2s ease;
}

.nav-link:hover svg {
  opacity: 1;
}

/* Active link indicator */
.nav-link.router-link-exact-active,
.nav-link.router-link-active[href="/viajes"],
.nav-link.router-link-active[href="/consulta"],
.nav-link.router-link-active[href="/admin"] {
  color: #60A5FA !important;
  background: rgba(96,165,250,0.1);
}

.nav-link.router-link-exact-active::after,
.nav-link.router-link-active[href="/viajes"]::after,
.nav-link.router-link-active[href="/consulta"]::after,
.nav-link.router-link-active[href="/admin"]::after {
  content: '';
  position: absolute;
  bottom: 2px;
  left: 50%;
  transform: translateX(-50%);
  width: 16px;
  height: 2px;
  background: #60A5FA;
  border-radius: 1px;
}

.nav-divider {
  width: 1px;
  height: 20px;
  background: rgba(255,255,255,0.1);
  margin: 0 0.35rem;
}

.nav-user-label {
  font-size: 0.8rem;
  font-weight: 600;
  padding: 0.45rem 0.5rem;
  white-space: nowrap;
  color: rgba(255,255,255,0.6);
}

.nav-logout-btn {
  font-size: 0.82rem;
}

.nav-logout-btn:hover {
  color: #f87171 !important;
  background: rgba(248,113,113,0.1) !important;
}

.nav-customer-label {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  color: #34D399;
  font-size: 0.8rem;
  font-weight: 600;
}

.mobile-toggle {
  display: none;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border: none;
  border-radius: var(--radius-md);
  background: rgba(255,255,255,0.06);
  cursor: pointer;
  transition: all 0.2s ease;
}

.mobile-toggle:hover {
  background: rgba(255,255,255,0.12);
}

/* ═══════════════════════════════════════════════════════
   MAIN
   ═══════════════════════════════════════════════════════ */
.main-content {
  flex: 1;
  padding-top: var(--nav-height);
}

/* ═══════════════════════════════════════════════════════
   FOOTER — Premium dark
   ═══════════════════════════════════════════════════════ */
.footer {
  position: relative;
  background: linear-gradient(180deg, #0a0e1a 0%, #0f172a 100%);
  border-top: 1px solid rgba(255,255,255,0.06);
  padding: 4rem 0 2rem;
  overflow: hidden;
}

.footer-glow {
  position: absolute;
  top: -100px;
  left: 50%;
  transform: translateX(-50%);
  width: 600px;
  height: 200px;
  background: radial-gradient(ellipse, rgba(59,130,246,0.08) 0%, transparent 70%);
  pointer-events: none;
}

.footer-grid {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1.5fr;
  gap: 3rem;
  margin-bottom: 3rem;
}

.footer-brand-col {
  max-width: 320px;
}

.footer-brand {
  display: inline-flex;
  align-items: center;
  gap: 0.6rem;
  text-decoration: none;
  margin-bottom: 1rem;
}

.footer-brand-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  background: linear-gradient(135deg, #3B82F6, #60A5FA);
  color: white;
}

.footer-brand-name {
  font-size: 1.25rem;
  font-weight: 800;
  color: #ffffff;
  letter-spacing: -0.02em;
}

.footer-desc {
  font-size: 0.85rem;
  color: rgba(255,255,255,0.5);
  line-height: 1.7;
  margin-bottom: 1.25rem;
}

.footer-socials {
  display: flex;
  gap: 0.6rem;
}

.social-link {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: rgba(255,255,255,0.05);
  border: 1px solid rgba(255,255,255,0.08);
  color: rgba(255,255,255,0.6);
  text-decoration: none;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.social-link:hover {
  background: rgba(96,165,250,0.15);
  border-color: rgba(96,165,250,0.3);
  color: #60A5FA;
  transform: translateY(-3px);
  box-shadow: 0 8px 20px rgba(59,130,246,0.2);
}

.footer-heading {
  font-size: 0.8rem;
  font-weight: 700;
  color: rgba(255,255,255,0.85);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  margin-bottom: 1.25rem;
}

.footer-links {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.footer-links a,
.footer-contact-item {
  font-size: 0.85rem;
  color: rgba(255,255,255,0.5);
  text-decoration: none;
  transition: all 0.25s ease;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.footer-links a:hover {
  color: #60A5FA;
  transform: translateX(4px);
}

.footer-contact-item {
  gap: 0.6rem;
}

.footer-contact-item svg {
  color: #60A5FA;
  opacity: 0.7;
  flex-shrink: 0;
}

.footer-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 2rem;
  border-top: 1px solid rgba(255,255,255,0.06);
}

.footer-bottom p {
  font-size: 0.78rem;
  color: rgba(255,255,255,0.35);
  margin: 0;
}

.footer-credit {
  font-weight: 500;
  color: rgba(255,255,255,0.25) !important;
}

/* ═══════════════════════════════════════════════════════
   MOBILE
   ═══════════════════════════════════════════════════════ */
@media (max-width: 768px) {
  .mobile-toggle {
    display: flex;
  }

  .navbar-nav {
    display: none;
    position: absolute;
    top: var(--nav-height);
    left: 0;
    right: 0;
    flex-direction: column;
    background: rgba(10, 14, 26, 0.96);
    backdrop-filter: blur(24px);
    border-bottom: 1px solid rgba(255,255,255,0.08);
    padding: 0.75rem 1rem 1.25rem;
    gap: 0.15rem;
    box-shadow: 0 24px 48px rgba(0,0,0,0.4);
  }

  .navbar-nav.open {
    display: flex;
  }

  .nav-link {
    padding: 0.65rem 1rem;
    border-radius: var(--radius-md);
    width: 100%;
    color: rgba(255,255,255,0.75) !important;
  }

  .nav-link:hover {
    background: rgba(255,255,255,0.08) !important;
    color: #ffffff !important;
  }

  .nav-divider {
    width: 100%;
    height: 1px;
    margin: 0.35rem 0;
    background: rgba(255,255,255,0.08);
  }

  .nav-user-label {
    padding: 0.5rem 1rem;
    color: rgba(255,255,255,0.5) !important;
  }

  .brand-tagline {
    display: none;
  }

  .footer-grid {
    grid-template-columns: 1fr 1fr;
    gap: 2rem;
  }

  .footer-brand-col {
    grid-column: 1 / -1;
    max-width: 100%;
  }

  .footer-bottom {
    flex-direction: column;
    gap: 0.5rem;
    text-align: center;
  }
}

@media (max-width: 480px) {
  .footer-grid {
    grid-template-columns: 1fr;
    gap: 1.5rem;
  }
}
</style>
