<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '../stores/auth'
import { getPendingVouchers } from '../api/client'

const router = useRouter()
const { user, logout } = useAuth()

const sidebarOpen = ref(false)
const pendingVouchersCount = ref(0)

let voucherPollTimer: ReturnType<typeof setInterval> | null = null

async function loadPendingCount() {
  try {
    const vouchers = await getPendingVouchers()
    pendingVouchersCount.value = vouchers.length
  } catch { /* ignore */ }
}

onMounted(() => {
  loadPendingCount()
  voucherPollTimer = setInterval(loadPendingCount, 15000)
})

onUnmounted(() => {
  if (voucherPollTimer) clearInterval(voucherPollTimer)
})

function toggleSidebar() {
  sidebarOpen.value = !sidebarOpen.value
}

function closeSidebar() {
  sidebarOpen.value = false
}

function handleLogout() {
  logout()
  router.push('/login')
}
</script>

<template>
  <div class="admin-layout">
    <!-- Mobile overlay -->
    <div v-if="sidebarOpen" class="sidebar-overlay" @click="closeSidebar"></div>

    <!-- Sidebar -->
    <aside class="sidebar" :class="{ open: sidebarOpen }">
      <div class="sidebar-header">
        <router-link to="/admin" class="sidebar-brand" @click="closeSidebar">
          <div class="sidebar-brand-icon">
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M8 6v6"/>
              <path d="M15 6v6"/>
              <path d="M2 12h19.6"/>
              <path d="M18 18h3s.5-1.7.8-2.8c.1-.4.2-.8.2-1.2 0-.4-.1-.8-.2-1.2l-1.4-5C20.1 6.8 19.1 6 18 6H4a2 2 0 0 0-2 2v10h3"/>
              <circle cx="7" cy="18" r="2"/>
              <path d="M9 18h5"/>
              <circle cx="16" cy="18" r="2"/>
            </svg>
          </div>
          <div class="sidebar-brand-text">
            <span class="sidebar-brand-name">Pasaje</span>
            <span class="sidebar-brand-label">Panel Admin</span>
          </div>
        </router-link>
      </div>

      <nav class="sidebar-nav">
        <router-link to="/admin" class="sidebar-link" exact-active-class="active" @click="closeSidebar">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect width="7" height="9" x="3" y="3" rx="1"/>
            <rect width="7" height="5" x="14" y="3" rx="1"/>
            <rect width="7" height="9" x="14" y="12" rx="1"/>
            <rect width="7" height="5" x="3" y="16" rx="1"/>
          </svg>
          <span>Dashboard</span>
        </router-link>

        <router-link to="/admin/venta" class="sidebar-link" active-class="active" @click="closeSidebar">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M2 9a3 3 0 0 1 0 6v2a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-2a3 3 0 0 1 0-6V7a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2Z"/>
            <path d="M13 5v2"/><path d="M13 17v2"/><path d="M13 11v2"/>
          </svg>
          <span>Punto de Venta</span>
        </router-link>

        <router-link to="/admin/vouchers" class="sidebar-link" active-class="active" @click="closeSidebar">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M9 12h6"/><path d="M12 9v6"/><circle cx="12" cy="12" r="10"/>
          </svg>
          <span>Verificar Pagos</span>
          <span v-if="pendingVouchersCount > 0" class="sidebar-badge">{{ pendingVouchersCount }}</span>
        </router-link>

        <router-link to="/admin/encomiendas" class="sidebar-link" active-class="active" @click="closeSidebar">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z"/>
            <path d="m3.3 7 8.7 5 8.7-5"/><path d="M12 22V12"/>
          </svg>
          <span>Encomiendas</span>
        </router-link>

        <router-link to="/admin/rutas" class="sidebar-link" active-class="active" @click="closeSidebar">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="6" cy="19" r="3"/>
            <path d="M9 19h8.5a3.5 3.5 0 0 0 0-7h-11a3.5 3.5 0 0 1 0-7H15"/>
            <circle cx="18" cy="5" r="3"/>
          </svg>
          <span>Rutas</span>
        </router-link>

        <router-link to="/admin/paradas" class="sidebar-link sidebar-link-sub" active-class="active" @click="closeSidebar">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z"/>
            <circle cx="12" cy="10" r="3"/>
          </svg>
          <span>Paradas</span>
        </router-link>

        <router-link to="/admin/vehiculos" class="sidebar-link" active-class="active" @click="closeSidebar">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M8 6v6"/>
            <path d="M15 6v6"/>
            <path d="M2 12h19.6"/>
            <path d="M18 18h3s.5-1.7.8-2.8c.1-.4.2-.8.2-1.2 0-.4-.1-.8-.2-1.2l-1.4-5C20.1 6.8 19.1 6 18 6H4a2 2 0 0 0-2 2v10h3"/>
            <circle cx="7" cy="18" r="2"/>
            <path d="M9 18h5"/>
            <circle cx="16" cy="18" r="2"/>
          </svg>
          <span>Vehiculos</span>
        </router-link>

        <router-link to="/admin/plantillas-bus" class="sidebar-link sidebar-link-sub" active-class="active" @click="closeSidebar">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect width="7" height="7" x="3" y="3" rx="1"/>
            <rect width="7" height="7" x="14" y="3" rx="1"/>
            <rect width="7" height="7" x="14" y="14" rx="1"/>
            <rect width="7" height="7" x="3" y="14" rx="1"/>
          </svg>
          <span>Plantillas de Bus</span>
        </router-link>

        <router-link to="/admin/plantillas" class="sidebar-link" active-class="active" @click="closeSidebar">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
            <line x1="16" y1="2" x2="16" y2="6"/>
            <line x1="8" y1="2" x2="8" y2="6"/>
            <line x1="3" y1="10" x2="21" y2="10"/>
          </svg>
          <span>Salidas Programadas</span>
        </router-link>

        <router-link to="/admin/viajes" class="sidebar-link" active-class="active" @click="closeSidebar">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect width="18" height="18" x="3" y="4" rx="2" ry="2"/>
            <line x1="16" x2="16" y1="2" y2="6"/>
            <line x1="8" x2="8" y1="2" y2="6"/>
            <line x1="3" x2="21" y1="10" y2="10"/>
          </svg>
          <span>Viajes</span>
        </router-link>

        <router-link v-if="user?.role === 'admin'" to="/admin/usuarios" class="sidebar-link" active-class="active" @click="closeSidebar">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/>
            <circle cx="9" cy="7" r="4"/>
            <path d="M22 21v-2a4 4 0 0 0-3-3.87"/>
            <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
          </svg>
          <span>Usuarios</span>
        </router-link>
        <router-link v-if="user?.role === 'admin'" to="/admin/facturacion" class="sidebar-link" active-class="active" @click="closeSidebar">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/>
            <circle cx="12" cy="12" r="3"/>
          </svg>
          <span>Facturacion</span>
        </router-link>

        <router-link v-if="user?.role === 'admin'" to="/admin/configuracion" class="sidebar-link" active-class="active" @click="closeSidebar">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/>
            <circle cx="12" cy="12" r="3"/>
          </svg>
          <span>Configuracion</span>
        </router-link>
      </nav>

      <div class="sidebar-footer">
        <!-- User info -->
        <div v-if="user" class="sidebar-user">
          <div class="sidebar-user-avatar">
            {{ user.name.charAt(0).toUpperCase() }}
          </div>
          <div class="sidebar-user-info">
            <span class="sidebar-user-name">{{ user.name }}</span>
            <span class="sidebar-user-role">{{ user.role }}</span>
          </div>
        </div>

        <button class="sidebar-link sidebar-logout" @click="handleLogout">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
            <polyline points="16 17 21 12 16 7"/>
            <line x1="21" x2="9" y1="12" y2="12"/>
          </svg>
          <span>Cerrar Sesion</span>
        </button>

        <router-link to="/" class="sidebar-link sidebar-back">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="m12 19-7-7 7-7"/>
            <path d="M19 12H5"/>
          </svg>
          <span>Volver al sitio</span>
        </router-link>
      </div>
    </aside>

    <!-- Main Area -->
    <div class="admin-main">
      <!-- Top bar -->
      <header class="admin-topbar">
        <button class="topbar-toggle" @click="toggleSidebar" aria-label="Menu">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="4" x2="20" y1="12" y2="12"/>
            <line x1="4" x2="20" y1="6" y2="6"/>
            <line x1="4" x2="20" y1="18" y2="18"/>
          </svg>
        </button>
        <div class="topbar-spacer"></div>
      </header>

      <!-- Content -->
      <main class="admin-content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<style scoped>
.admin-layout {
  display: flex;
  min-height: 100vh;
  background: var(--slate-50);
}

/* ── Sidebar (White - Venta style) ── */
.sidebar {
  width: 256px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: white;
  border-right: 1px solid var(--slate-200);
  box-shadow: var(--shadow-sm);
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  z-index: 200;
  transition: transform 0.25s ease;
}

.sidebar-header {
  padding: 1.25rem 1rem;
  border-bottom: 1px solid var(--slate-200);
}

.sidebar-brand {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  text-decoration: none;
  color: var(--slate-900);
  padding: 0.75rem;
  background: linear-gradient(135deg, var(--brand-50) 0%, var(--accent-50) 100%);
  border-radius: var(--radius-lg);
  transition: all 0.3s ease;
}

.sidebar-brand:hover {
  color: var(--slate-900);
}

.sidebar-brand-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  background: linear-gradient(135deg, var(--brand-400), var(--brand-300));
  color: white;
  flex-shrink: 0;
  box-shadow: var(--shadow-md);
}

.sidebar-brand-text {
  display: flex;
  flex-direction: column;
  line-height: 1.2;
}

.sidebar-brand-name {
  font-size: 1.1rem;
  font-weight: 800;
  color: var(--slate-900);
  letter-spacing: -0.02em;
}

.sidebar-brand-label {
  font-size: 0.68rem;
  color: var(--slate-500);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  margin-top: 0.1rem;
}

/* ── Nav Links (Venta style) ── */
.sidebar-nav {
  flex: 1;
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  overflow-y: auto;
}

.sidebar-link {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border-radius: var(--radius-md);
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--slate-700);
  text-decoration: none;
  transition: all 0.3s ease;
  border: none;
  background: none;
  cursor: pointer;
  width: 100%;
  font-family: inherit;
}

.sidebar-link:hover {
  color: var(--brand-500);
  background: var(--brand-50);
}

.sidebar-link.active {
  color: white;
  background: linear-gradient(135deg, var(--brand-400), var(--brand-300));
  box-shadow: var(--shadow-md);
}

.sidebar-link.active svg {
  color: white;
}

.sidebar-link-sub {
  padding-left: 2.5rem;
  font-size: 0.82rem;
  color: var(--slate-500);
}

.sidebar-link-sub svg {
  width: 18px;
  height: 18px;
}

.sidebar-badge {
  margin-left: auto;
  background: var(--danger-500);
  color: white;
  font-size: 0.7rem;
  font-weight: 700;
  min-width: 20px;
  height: 20px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 5px;
  animation: pulse-badge 2s ease-in-out infinite;
}

@keyframes pulse-badge {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

.sidebar-footer {
  padding: 0.75rem;
  border-top: 1px solid var(--slate-200);
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.sidebar-back {
  color: var(--slate-500);
  font-size: 0.82rem;
}

.sidebar-back:hover {
  color: var(--slate-700);
  background: var(--slate-100);
}

.sidebar-logout {
  color: var(--slate-500);
  font-size: 0.82rem;
}

.sidebar-logout:hover {
  color: var(--danger-600);
  background: var(--danger-50);
}

/* ── User Info ── */
.sidebar-user {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  padding: 0.65rem 0.85rem;
  margin-bottom: 0.35rem;
}

.sidebar-user-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: var(--radius-full);
  background: linear-gradient(135deg, var(--brand-400), var(--accent-400));
  color: white;
  font-size: 0.82rem;
  font-weight: 700;
  flex-shrink: 0;
  box-shadow: var(--shadow-sm);
}

.sidebar-user-info {
  display: flex;
  flex-direction: column;
  line-height: 1.25;
  min-width: 0;
}

.sidebar-user-name {
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--slate-900);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sidebar-user-role {
  font-size: 0.68rem;
  font-weight: 500;
  color: var(--slate-500);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* ── Overlay ── */
.sidebar-overlay {
  display: none;
}

/* ── Main Area ── */
.admin-main {
  flex: 1;
  margin-left: 256px;
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.admin-topbar {
  display: none;
  align-items: center;
  height: 56px;
  padding: 0 1.25rem;
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
  position: sticky;
  top: 0;
  z-index: 100;
}

.topbar-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
}

.topbar-toggle:hover {
  background: var(--color-background-mute);
}

.topbar-spacer {
  flex: 1;
}

.admin-content {
  flex: 1;
  padding: 1.75rem 2rem 2.5rem;
}

/* ── Mobile ── */
@media (max-width: 1024px) {
  .sidebar {
    transform: translateX(-100%);
  }

  .sidebar.open {
    transform: translateX(0);
  }

  .sidebar-overlay {
    display: block;
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    z-index: 199;
  }

  .admin-main {
    margin-left: 0;
  }

  .admin-topbar {
    display: flex;
  }

  .admin-content {
    padding: 1.25rem 1rem 2rem;
  }
}
</style>
