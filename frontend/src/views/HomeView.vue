<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getRoutes, type Route } from '../api/client'
import { useScrollReveal } from '../composables/useScrollReveal'

const router = useRouter()
const { reveal } = useScrollReveal()
const routes = ref<Route[]>([])
const selectedOrigin = ref('')
const selectedDestination = ref('')
const travelDate = ref(new Date().toISOString().slice(0, 10))

const apiStatus = ref<{ status: string; error?: string } | null>(null)
const apiBase = import.meta.env.VITE_API_URL ?? ''
const heroLoaded = ref(false)

async function checkHealth() {
  try {
    const res = await fetch(`${apiBase}/health/ready`)
    const data = await res.json()
    apiStatus.value = res.ok ? { status: 'ok' } : { status: 'error', error: data.error || res.statusText }
  } catch (e) {
    apiStatus.value = { status: 'error', error: (e as Error).message }
  }
}

async function loadRoutes() {
  try {
    routes.value = await getRoutes()
  } catch {
    routes.value = []
  }
}

function searchTrips() {
  const r = routes.value[0]
  if (r) {
    router.push(`/viajes/ruta/${r.id}`)
  } else {
    router.push('/viajes')
  }
}

function goToRoute(r: Route) {
  router.push(`/viajes/ruta/${r.id}`)
}

onMounted(() => {
  checkHealth()
  loadRoutes()
  // Trigger hero entrance animations after mount
  setTimeout(() => { heroLoaded.value = true }, 100)
})
</script>

<template>
  <div class="home">
    <!-- ═══════════════════════════════════════════════════════
         HERO SECTION — Modern Cinematic with Video Background
         ═══════════════════════════════════════════════════════ -->
    <section class="hero">
      <!-- Video Background -->
      <div class="hero-video-wrapper">
        <video
          class="hero-video"
          autoplay
          muted
          loop
          playsinline
          poster="https://images.pexels.com/photos/1173777/pexels-photo-1173777.jpeg?auto=compress&cs=tinysrgb&w=1920"
        >
          <source
            src="https://videos.pexels.com/video-files/2435463/2435463-uhd_2560_1440_24fps.mp4"
            type="video/mp4"
          />
        </video>
        <!-- Dark cinematic overlay -->
        <div class="hero-overlay"></div>
        <!-- Animated light streaks -->
        <div class="hero-light-streaks">
          <div class="streak streak-1"></div>
          <div class="streak streak-2"></div>
          <div class="streak streak-3"></div>
          <div class="streak streak-4"></div>
        </div>
        <!-- Floating particles -->
        <div class="hero-particles">
          <span v-for="n in 20" :key="n" class="particle" :style="`--i:${n}`"></span>
        </div>
        <!-- Vignette effect -->
        <div class="hero-vignette"></div>
      </div>

      <div class="container hero-content" :class="{ loaded: heroLoaded }">
        <!-- Animated bus icon -->
        <div class="hero-bus-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M8 6v6"/><path d="M15 6v6"/><path d="M2 12h19.6"/>
            <path d="M18 18h3s.5-1.7.8-2.8c.1-.4.2-.8.2-1.2 0-.4-.1-.8-.2-1.2l-1.4-5C20.1 6.8 19.1 6 18 6H4a2 2 0 0 0-2 2v10h3"/>
            <circle cx="7" cy="18" r="2"/><path d="M9 18h5"/><circle cx="16" cy="18" r="2"/>
          </svg>
        </div>

        <!-- Main headline with reveal animation -->
        <h1 class="hero-title">
          <span class="hero-title-line">Tu viaje comienza</span>
          <span class="hero-title-line hero-title-gradient">aqui y ahora</span>
        </h1>

        <p class="hero-subtitle">
          Reserva tu pasaje interprovincial entre Arequipa, Cusco y mas destinos.
          Asientos en tiempo real, sin filas, sin complicaciones.
        </p>

        <!-- Trust badges -->
        <div class="hero-badges">
          <span class="hero-badge">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/></svg>
            Pago seguro
          </span>
          <span class="hero-badge">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
            Asiento garantizado
          </span>
          <span class="hero-badge">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
            Tiempo real
          </span>
        </div>

        <!-- Search Widget — Premium Glassmorphism -->
        <div class="search-widget" :class="{ loaded: heroLoaded }">
          <div class="search-row">
            <div class="search-field">
              <label class="search-label">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="10" r="3"/><path d="M12 21.7C17.3 17 20 13 20 10a8 8 0 1 0-16 0c0 3 2.7 6.9 8 11.7z"/></svg>
                Origen
              </label>
              <select v-model="selectedOrigin" class="search-select">
                <option value="">Selecciona origen</option>
                <option value="arequipa">Arequipa</option>
                <option value="cusco">Cusco</option>
              </select>
            </div>

            <button class="swap-btn" title="Intercambiar">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="m7 16 4 4 4-4"/><path d="M11 20V4"/><path d="m17 8-4-4-4 4"/><path d="M13 4v16"/>
              </svg>
            </button>

            <div class="search-field">
              <label class="search-label">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="10" r="3"/><path d="M12 21.7C17.3 17 20 13 20 10a8 8 0 1 0-16 0c0 3 2.7 6.9 8 11.7z"/></svg>
                Destino
              </label>
              <select v-model="selectedDestination" class="search-select">
                <option value="">Selecciona destino</option>
                <option value="cusco">Cusco</option>
                <option value="arequipa">Arequipa</option>
              </select>
            </div>

            <div class="search-field">
              <label class="search-label">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="4" rx="2" ry="2"/><line x1="16" x2="16" y1="2" y2="6"/><line x1="8" x2="8" y1="2" y2="6"/><line x1="3" x2="21" y1="10" y2="10"/></svg>
                Fecha de viaje
              </label>
              <input type="date" v-model="travelDate" class="search-date" :min="new Date().toISOString().slice(0, 10)" />
            </div>

            <button class="search-btn" @click="searchTrips">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
              Buscar pasajes
            </button>
          </div>
        </div>

        <!-- Scroll indicator -->
        <div class="scroll-indicator">
          <div class="scroll-mouse">
            <div class="scroll-wheel"></div>
          </div>
          <span>Descubre mas</span>
        </div>
      </div>
    </section>

    <!-- Routes Section -->
    <section class="routes-section" v-if="routes.length > 0">
      <div class="container">
        <h2 class="section-title">Rutas disponibles</h2>
        <p class="section-subtitle">Elige tu destino y encuentra los mejores horarios</p>
        <div class="route-grid">
          <button v-for="(r, idx) in routes" :key="r.id" class="route-card" :style="`--delay:${idx * 0.1}s`" @click="goToRoute(r)">
            <div class="route-card__top">
              <div class="route-cities">
                <span class="route-city">{{ r.name.split(' - ')[0] || r.name }}</span>
                <div class="route-arrow">
                  <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M5 12h14"/><path d="m12 5 7 7-7 7"/>
                  </svg>
                </div>
                <span class="route-city">{{ r.name.split(' - ')[1] || '' }}</span>
              </div>
              <span class="route-badge">{{ r.code }}</span>
            </div>
            <div class="route-card__bottom">
              <span class="route-price">Desde S/ {{ r.price_per_seat.toFixed(2) }}</span>
              <span class="route-cta">
                Ver salidas
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>
              </span>
            </div>
          </button>
        </div>
      </div>
    </section>

    <!-- Benefits Section -->
    <section class="benefits-section">
      <div class="container">
        <h2 class="section-title">Viaja con confianza</h2>
        <p class="section-subtitle">Todo lo que necesitas para un viaje seguro y comodo</p>
        <div class="benefits-grid">
          <div class="benefit-card" style="--delay:0s">
            <div class="benefit-icon benefit-icon--blue">
              <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect width="18" height="18" x="3" y="3" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/>
              </svg>
            </div>
            <h3 class="benefit-title">Elige tu asiento</h3>
            <p class="benefit-text">Selecciona el asiento que prefieras viendo la disponibilidad en tiempo real del bus.</p>
          </div>
          <div class="benefit-card" style="--delay:0.12s">
            <div class="benefit-icon benefit-icon--emerald">
              <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/><path d="m9 12 2 2 4-4"/>
              </svg>
            </div>
            <h3 class="benefit-title">Compra segura</h3>
            <p class="benefit-text">Tu asiento queda reservado al instante. Sin doble venta, sin sorpresas.</p>
          </div>
          <div class="benefit-card" style="--delay:0.24s">
            <div class="benefit-icon benefit-icon--amber">
              <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M8 6v6"/><path d="M15 6v6"/><path d="M2 12h19.6"/>
                <path d="M18 18h3s.5-1.7.8-2.8c.1-.4.2-.8.2-1.2 0-.4-.1-.8-.2-1.2l-1.4-5C20.1 6.8 19.1 6 18 6H4a2 2 0 0 0-2 2v10h3"/>
                <circle cx="7" cy="18" r="2"/><path d="M9 18h5"/><circle cx="16" cy="18" r="2"/>
              </svg>
            </div>
            <h3 class="benefit-title">Flota moderna</h3>
            <p class="benefit-text">Buses de 1 y 2 pisos con servicio Semi-Cama y Cama para tu comodidad.</p>
          </div>
          <div class="benefit-card" style="--delay:0.36s">
            <div class="benefit-icon benefit-icon--green">
              <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M2 9a3 3 0 0 1 0 6v2a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-2a3 3 0 0 1 0-6V7a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2Z"/>
                <path d="M13 5v2"/><path d="M13 17v2"/><path d="M13 11v2"/>
              </svg>
            </div>
            <h3 class="benefit-title">Boleto digital</h3>
            <p class="benefit-text">Recibe tu boleto electronico al instante por email. Sin colas, sin papel.</p>
          </div>
        </div>
      </div>
    </section>

    <!-- How It Works Section -->
    <section class="steps-section">
      <div class="container">
        <h2 class="section-title">Como comprar tu pasaje</h2>
        <p class="section-subtitle">En solo 4 pasos tendras tu boleto listo</p>
        <div class="steps-track">
          <div class="step-item" style="--delay:0s">
            <div class="step-number">1</div>
            <div class="step-line"></div>
            <h3 class="step-title">Busca tu viaje</h3>
            <p class="step-text">Selecciona origen, destino y fecha</p>
          </div>
          <div class="step-item" style="--delay:0.15s">
            <div class="step-number">2</div>
            <div class="step-line"></div>
            <h3 class="step-title">Elige tu asiento</h3>
            <p class="step-text">Selecciona del mapa interactivo del bus</p>
          </div>
          <div class="step-item" style="--delay:0.30s">
            <div class="step-number">3</div>
            <div class="step-line"></div>
            <h3 class="step-title">Paga en linea</h3>
            <p class="step-text">Confirma tus datos y realiza el pago seguro</p>
          </div>
          <div class="step-item" style="--delay:0.45s">
            <div class="step-number">4</div>
            <h3 class="step-title">Viaja tranquilo</h3>
            <p class="step-text">Recibe tu boleto y presentalo al abordar</p>
          </div>
        </div>
      </div>
    </section>

    <!-- CTA Final -->
    <section class="cta-section">
      <div class="container">
        <div class="cta-card">
          <div class="cta-content">
            <h2 class="cta-title">Listo para viajar?</h2>
            <p class="cta-text">Reserva tu asiento ahora y viaja con total tranquilidad. Paga con tarjeta, Yape o en efectivo.</p>
          </div>
          <router-link to="/viajes" class="cta-btn">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
            Buscar viajes
          </router-link>
        </div>
      </div>
    </section>

    <!-- System Status Bar -->
    <section class="status-section" v-if="apiStatus">
      <div class="container">
        <div class="status-bar" :class="apiStatus.status === 'ok' ? 'status-ok' : 'status-err'">
          <div class="status-dot" :class="apiStatus.status === 'ok' ? 'dot-ok' : 'dot-err'"></div>
          <span v-if="apiStatus.status === 'ok'">Sistema operativo — Venta habilitada</span>
          <span v-else>Sistema en mantenimiento: {{ apiStatus.error }}</span>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
/* ═══════════════════════════════════════════════════════
   HERO SECTION — Cinematic Video Background
   ═══════════════════════════════════════════════════════ */
.hero {
  position: relative;
  padding: 5rem 0 4rem;
  overflow: hidden;
  min-height: 100vh;
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Video background wrapper */
.hero-video-wrapper {
  position: absolute;
  inset: 0;
  z-index: 0;
  background: #0a0e1a;
}

.hero-video {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0.85;
  animation: videoFadeIn 2s ease-out forwards;
}

@keyframes videoFadeIn {
  from { opacity: 0; }
  to { opacity: 0.85; }
}

/* Cinematic overlay */
.hero-overlay {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(180deg, rgba(10,14,26,0.4) 0%, rgba(10,14,26,0.2) 40%, rgba(10,14,26,0.6) 100%),
    linear-gradient(135deg, rgba(15,23,42,0.5) 0%, transparent 50%);
  z-index: 1;
}

/* Animated light streaks (speed lines) */
.hero-light-streaks {
  position: absolute;
  inset: 0;
  z-index: 1;
  overflow: hidden;
  pointer-events: none;
}

.streak {
  position: absolute;
  height: 2px;
  background: linear-gradient(90deg, transparent, rgba(96,165,250,0.4), transparent);
  border-radius: 2px;
  animation: streakMove linear infinite;
}

.streak-1 {
  top: 20%;
  left: -100%;
  width: 60%;
  animation-duration: 4s;
  animation-delay: 0s;
}

.streak-2 {
  top: 35%;
  left: -100%;
  width: 40%;
  animation-duration: 5s;
  animation-delay: 1.5s;
}

.streak-3 {
  top: 60%;
  left: -100%;
  width: 50%;
  animation-duration: 3.5s;
  animation-delay: 0.8s;
}

.streak-4 {
  top: 75%;
  left: -100%;
  width: 45%;
  animation-duration: 4.5s;
  animation-delay: 2.2s;
}

@keyframes streakMove {
  0% { transform: translateX(0); opacity: 0; }
  10% { opacity: 1; }
  90% { opacity: 1; }
  100% { transform: translateX(400vw); opacity: 0; }
}

/* Floating particles */
.hero-particles {
  position: absolute;
  inset: 0;
  z-index: 1;
  overflow: hidden;
  pointer-events: none;
}

.particle {
  position: absolute;
  width: 3px;
  height: 3px;
  background: rgba(255,255,255,0.4);
  border-radius: 50%;
  bottom: -10px;
  left: calc(var(--i) * 5%);
  animation: particleFloat calc(4s + var(--i) * 0.3s) ease-in-out infinite;
  animation-delay: calc(var(--i) * 0.2s);
  box-shadow: 0 0 6px rgba(96,165,250,0.5);
}

@keyframes particleFloat {
  0% { transform: translateY(0) scale(1); opacity: 0; }
  10% { opacity: 1; }
  90% { opacity: 0.6; }
  100% { transform: translateY(-100vh) scale(0.3); opacity: 0; }
}

/* Vignette */
.hero-vignette {
  position: absolute;
  inset: 0;
  z-index: 2;
  box-shadow: inset 0 0 150px rgba(0,0,0,0.5);
  pointer-events: none;
}

/* Hero content */
.hero-content {
  position: relative;
  z-index: 3;
  text-align: center;
}

/* Bus icon */
.hero-bus-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: rgba(255,255,255,0.08);
  border: 2px solid rgba(255,255,255,0.15);
  color: #60A5FA;
  margin-bottom: 1.5rem;
  animation: floatBus 3s ease-in-out infinite, iconGlow 3s ease-in-out infinite alternate;
  backdrop-filter: blur(8px);
}

@keyframes floatBus {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-8px); }
}

@keyframes iconGlow {
  from { box-shadow: 0 0 20px rgba(96,165,250,0.15); }
  to { box-shadow: 0 0 40px rgba(96,165,250,0.35); }
}

/* Title with staggered animation */
.hero-title {
  font-size: 3.5rem;
  font-weight: 800;
  color: #ffffff;
  line-height: 1.1;
  letter-spacing: -0.03em;
  margin-bottom: 1.25rem;
  text-shadow: 0 4px 30px rgba(0,0,0,0.3);
}

.hero-title-line {
  display: block;
  opacity: 0;
  transform: translateY(20px);
  animation: lineReveal 0.8s cubic-bezier(0.22, 1, 0.36, 1) forwards;
}

.hero-title-line:first-child {
  animation-delay: 0.3s;
}

.hero-title-line:last-child {
  animation-delay: 0.5s;
}

@keyframes lineReveal {
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.hero-title-gradient {
  background: linear-gradient(135deg, #60A5FA 0%, #34D399 50%, #A78BFA 100%);
  background-size: 200% 200%;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  animation: gradientShift 4s ease infinite;
}

@keyframes gradientShift {
  0%, 100% { background-position: 0% 50%; }
  50% { background-position: 100% 50%; }
}

.hero-subtitle {
  font-size: 1.15rem;
  color: rgba(255,255,255,0.75);
  max-width: 540px;
  margin: 0 auto 2.5rem;
  line-height: 1.7;
  font-weight: 400;
  text-shadow: 0 2px 10px rgba(0,0,0,0.2);
}

/* Badges */
.hero-badges {
  display: flex;
  justify-content: center;
  gap: 1rem;
  margin-bottom: 2.5rem;
  flex-wrap: wrap;
}

.hero-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.4rem 1rem;
  background: rgba(255,255,255,0.06);
  border: 1px solid rgba(255,255,255,0.12);
  border-radius: var(--radius-full);
  font-size: 0.78rem;
  font-weight: 600;
  color: rgba(255,255,255,0.85);
  letter-spacing: 0.02em;
  backdrop-filter: blur(8px);
  transition: all 0.3s ease;
}

.hero-badge:hover {
  background: rgba(255,255,255,0.12);
  border-color: rgba(255,255,255,0.2);
  transform: translateY(-2px);
}

.hero-badge svg {
  color: #34D399;
}

/* ═══════════════════════════════════════════════════════
   SEARCH WIDGET — Premium Glassmorphism
   ═══════════════════════════════════════════════════════ */
.search-widget {
  max-width: 940px;
  margin: 0 auto;
  background: rgba(255,255,255,0.08);
  backdrop-filter: blur(32px) saturate(180%);
  -webkit-backdrop-filter: blur(32px) saturate(180%);
  border: 1px solid rgba(255,255,255,0.15);
  border-radius: var(--radius-xl);
  padding: 1.75rem 2rem;
  box-shadow:
    0 32px 64px rgba(0,0,0,0.3),
    inset 0 1px 0 rgba(255,255,255,0.1),
    0 0 0 1px rgba(255,255,255,0.05);
  position: relative;
  overflow: hidden;
}

.search-widget::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 50%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(255,255,255,0.06), transparent);
  animation: shimmer 5s ease-in-out infinite;
}

@keyframes shimmer {
  0%, 100% { left: -100%; }
  50% { left: 150%; }
}

.search-row {
  display: flex;
  align-items: flex-end;
  gap: 0.875rem;
  position: relative;
  z-index: 1;
}

.search-field {
  flex: 1;
  min-width: 0;
}

.search-label {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.7rem;
  font-weight: 700;
  color: rgba(255,255,255,0.7);
  text-transform: uppercase;
  letter-spacing: 0.07em;
  margin-bottom: 0.5rem;
  padding-left: 0.2rem;
}

.search-label svg {
  color: #60A5FA;
}

.search-select,
.search-date {
  width: 100%;
  padding: 0.8rem 1rem;
  border: 2px solid rgba(255,255,255,0.12);
  border-radius: var(--radius-md);
  font-size: 0.92rem;
  font-family: inherit;
  font-weight: 600;
  color: #ffffff;
  background: rgba(0,0,0,0.2);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  appearance: none;
  -webkit-appearance: none;
}

.search-select:focus,
.search-date:focus {
  outline: none;
  border-color: rgba(96,165,250,0.5);
  background: rgba(0,0,0,0.35);
  box-shadow: 0 0 0 4px rgba(96,165,250,0.12), 0 0 20px rgba(96,165,250,0.1);
}

.search-select {
  background-image: url("data:image/svg+xml,%3Csvg width='12' height='8' viewBox='0 0 12 8' fill='none' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M1 1.5L6 6.5L11 1.5' stroke='rgba(255,255,255,0.6)' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 0.85rem center;
  padding-right: 2.5rem;
  cursor: pointer;
}

.search-select option {
  background: #1e293b;
  color: #ffffff;
}

.search-date::-webkit-calendar-picker-indicator {
  filter: invert(1) opacity(0.6);
  cursor: pointer;
}

.swap-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border: 2px solid rgba(255,255,255,0.15);
  border-radius: 50%;
  background: rgba(255,255,255,0.06);
  color: #60A5FA;
  cursor: pointer;
  flex-shrink: 0;
  margin-bottom: 0.1rem;
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

.swap-btn:hover {
  border-color: rgba(96,165,250,0.5);
  background: rgba(96,165,250,0.15);
  transform: rotate(180deg);
  box-shadow: 0 0 20px rgba(96,165,250,0.2);
}

.search-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.85rem 2rem;
  background: linear-gradient(135deg, #10B981 0%, #059669 50%, #10B981 100%);
  background-size: 200% 200%;
  color: #ffffff;
  border: none;
  border-radius: var(--radius-md);
  font-size: 0.95rem;
  font-weight: 700;
  cursor: pointer;
  white-space: nowrap;
  flex-shrink: 0;
  font-family: inherit;
  letter-spacing: -0.01em;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow:
    0 4px 20px rgba(16,185,129,0.4),
    0 0 40px rgba(16,185,129,0.1);
  animation: gradientShift 3s ease infinite;
}

.search-btn:hover {
  transform: translateY(-2px);
  box-shadow:
    0 8px 30px rgba(16,185,129,0.5),
    0 0 60px rgba(16,185,129,0.2);
}

.search-btn:active {
  transform: translateY(0);
}

/* Scroll indicator */
.scroll-indicator {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  margin-top: 3rem;
  color: rgba(255,255,255,0.5);
  font-size: 0.75rem;
  font-weight: 500;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.scroll-mouse {
  width: 22px;
  height: 36px;
  border: 2px solid rgba(255,255,255,0.3);
  border-radius: 12px;
  position: relative;
}

.scroll-wheel {
  width: 4px;
  height: 8px;
  background: rgba(255,255,255,0.5);
  border-radius: 2px;
  position: absolute;
  top: 6px;
  left: 50%;
  transform: translateX(-50%);
  animation: scrollWheel 2s ease-in-out infinite;
}

@keyframes scrollWheel {
  0%, 100% { top: 6px; opacity: 1; }
  50% { top: 18px; opacity: 0.3; }
}

@keyframes fadeInUp {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

/* ═══════════════════════════════════════════════════════
   SECTION TRANSITION DIVIDER
   ═══════════════════════════════════════════════════════ */
.hero-divider {
  position: relative;
  height: 120px;
  background: linear-gradient(180deg, transparent 0%, #0a0e1a 100%);
  z-index: 2;
  margin-top: -120px;
  pointer-events: none;
}

/* ═══════════════════════════════════════════════════════
   ROUTES SECTION — Dark premium
   ═══════════════════════════════════════════════════════ */
.routes-section {
  padding: 4.5rem 0;
  background: var(--color-background-soft);
}

.section-title {
  font-size: 1.85rem;
  font-weight: 800;
  color: var(--color-heading);
  text-align: center;
  margin-bottom: 0.4rem;
  letter-spacing: -0.025em;
}

.section-subtitle {
  font-size: 1rem;
  color: var(--color-text-muted);
  text-align: center;
  margin-bottom: 2.5rem;
  font-weight: 400;
}

.route-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 1.25rem;
}

.route-card {
  display: flex;
  flex-direction: column;
  padding: 1.5rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  font-family: inherit;
  color: inherit;
  text-align: left;
  width: 100%;
}

.route-card:hover {
  transform: translateY(-3px);
  box-shadow: var(--shadow-lg);
  border-color: var(--slate-300);
}

.route-card__top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.route-cities {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.route-city {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--color-heading);
}

.route-arrow {
  color: var(--slate-400);
  display: flex;
  flex-shrink: 0;
}

.route-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.2rem 0.65rem;
  background: var(--slate-100);
  color: var(--slate-600);
  border-radius: var(--radius-full);
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  flex-shrink: 0;
}

.route-card__bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 1rem;
  border-top: 1px solid var(--color-border);
}

.route-price {
  font-size: 1.05rem;
  font-weight: 800;
  color: var(--accent-600);
}

.route-cta {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  color: var(--slate-500);
  font-size: 0.85rem;
  font-weight: 600;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.route-card:hover .route-cta {
  gap: 0.5rem;
  color: var(--brand-600);
}

/* ═══════════════════════════════════════════════════════
   BENEFITS SECTION — Dark premium
   ═══════════════════════════════════════════════════════ */
.benefits-section {
  padding: 4.5rem 0;
  background: #ffffff;
  position: relative;
}

.benefits-section::before {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(ellipse at 50% 0%, rgba(59,130,246,0.06) 0%, transparent 60%);
  pointer-events: none;
}

.benefits-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1.5rem;
}

.benefit-card {
  padding: 2rem 1.5rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  text-align: center;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.benefit-card:hover {
  transform: translateY(-3px);
  box-shadow: var(--shadow-lg);
}

.benefit-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 60px;
  height: 60px;
  border-radius: var(--radius-lg);
  margin-bottom: 1.15rem;
}

.benefit-icon--blue {
  background: linear-gradient(135deg, #3B82F6, #60A5FA);
  color: #ffffff;
  box-shadow: 0 4px 14px rgba(59,130,246,0.3);
}

.benefit-icon--emerald {
  background: linear-gradient(135deg, #10B981, #34D399);
  color: #ffffff;
  box-shadow: 0 4px 14px rgba(16,185,129,0.3);
}

.benefit-icon--amber {
  background: linear-gradient(135deg, #F59E0B, #FBBF24);
  color: #ffffff;
  box-shadow: 0 4px 14px rgba(245,158,11,0.3);
}

.benefit-icon--green {
  background: linear-gradient(135deg, #059669, #10B981);
  color: #ffffff;
  box-shadow: 0 4px 14px rgba(5,150,105,0.3);
}

.benefit-title {
  font-size: 1.05rem;
  font-weight: 700;
  color: var(--color-heading);
  margin-bottom: 0.5rem;
}

.benefit-text {
  font-size: 0.88rem;
  color: var(--color-text-muted);
  line-height: 1.6;
}

/* ═══════════════════════════════════════════════════════
   HOW IT WORKS — STEPS SECTION
   ═══════════════════════════════════════════════════════ */
.steps-section {
  padding: 4.5rem 0;
  background: var(--color-background-soft);
}

.steps-track {
  display: flex;
  align-items: flex-start;
  justify-content: center;
  max-width: 900px;
  margin: 0 auto;
  position: relative;
}

.step-item {
  flex: 1;
  text-align: center;
  padding: 0 1rem;
  position: relative;
}

.step-number {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  border-radius: 50%;
  background: linear-gradient(135deg, #3B82F6 0%, #60A5FA 100%);
  color: #ffffff;
  font-size: 1.2rem;
  font-weight: 800;
  margin-bottom: 1rem;
  position: relative;
  z-index: 1;
  box-shadow: 0 6px 20px rgba(59,130,246,0.3);
}

.step-line {
  position: absolute;
  top: 26px;
  left: calc(50% + 32px);
  width: calc(100% - 64px);
  height: 2px;
  background: repeating-linear-gradient(
    90deg,
    var(--brand-200) 0px,
    var(--brand-200) 6px,
    transparent 6px,
    transparent 12px
  );
  z-index: 0;
}

.step-title {
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--color-heading);
  margin-bottom: 0.3rem;
}

.step-text {
  font-size: 0.82rem;
  color: var(--color-text-muted);
  line-height: 1.5;
}

/* ═══════════════════════════════════════════════════════
   CTA FINAL SECTION
   ═══════════════════════════════════════════════════════ */
.cta-section {
  padding: 3rem 0 4rem;
  background: #ffffff;
}

.cta-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 2rem;
  padding: 2.5rem 3rem;
  background: linear-gradient(135deg, #0F172A 0%, #1E3A8A 100%);
  border-radius: var(--radius-xl);
  box-shadow: 0 20px 40px rgba(15,23,42,0.25);
  position: relative;
  overflow: hidden;
}

.cta-card::before {
  content: '';
  position: absolute;
  top: -50%;
  right: -10%;
  width: 300px;
  height: 300px;
  border-radius: 50%;
  background: rgba(96,165,250,0.15);
  filter: blur(60px);
}

.cta-content {
  position: relative;
}

.cta-title {
  font-size: 1.75rem;
  font-weight: 800;
  color: white;
  margin-bottom: 0.5rem;
}

.cta-text {
  font-size: 0.95rem;
  color: rgba(255,255,255,0.7);
  max-width: 420px;
}

.cta-btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.9rem 2rem;
  background: linear-gradient(135deg, #10B981, #34D399);
  color: white;
  border: none;
  border-radius: var(--radius-lg);
  font-size: 1rem;
  font-weight: 700;
  font-family: inherit;
  text-decoration: none;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 4px 16px rgba(16,185,129,0.4);
  white-space: nowrap;
  flex-shrink: 0;
}

.cta-btn:hover {
  background: linear-gradient(135deg, #34D399, #6EE7B7);
  box-shadow: 0 6px 24px rgba(16,185,129,0.5);
  transform: translateY(-2px);
  color: white;
}

/* ═══════════════════════════════════════════════════════
   STATUS BAR
   ═══════════════════════════════════════════════════════ */
.status-section {
  padding: 1.5rem 0 3.5rem;
  background: var(--color-background-soft);
}

.status-bar {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.65rem 1.15rem;
  border-radius: var(--radius-full);
  font-size: 0.82rem;
  font-weight: 500;
  max-width: 420px;
  margin: 0 auto;
}

.status-ok {
  background: var(--success-50);
  color: var(--success-700);
  border: 1px solid var(--success-100);
}

.status-err {
  background: var(--danger-50);
  color: var(--danger-700);
  border: 1px solid var(--danger-100);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.dot-ok {
  background: var(--success-500);
  animation: statusPulse 2s ease-in-out infinite;
}

.dot-err {
  background: var(--danger-500);
}

@keyframes statusPulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(16,185,129,0.4); }
  50% { box-shadow: 0 0 0 5px rgba(16,185,129,0.08); }
}

/* ═══════════════════════════════════════════════════════
   RESPONSIVE
   ═══════════════════════════════════════════════════════ */
@media (max-width: 768px) {
  .hero {
    padding: 4rem 0 3rem;
    min-height: auto;
  }

  .hero-title {
    font-size: 2.25rem;
  }

  .hero-subtitle {
    font-size: 1rem;
    margin-bottom: 2rem;
  }

  .search-widget {
    padding: 1.25rem;
  }

  .search-row {
    flex-direction: column;
    gap: 0.75rem;
  }

  .swap-btn {
    align-self: center;
  }

  .search-btn {
    width: 100%;
    justify-content: center;
  }

  .scroll-indicator {
    display: none;
  }

  .route-grid {
    grid-template-columns: 1fr;
  }

  .benefits-grid {
    grid-template-columns: 1fr;
    gap: 1rem;
  }

  .steps-track {
    flex-direction: column;
    gap: 1.5rem;
    align-items: center;
  }

  .step-line {
    display: none;
  }

  .section-title {
    font-size: 1.5rem;
  }

  .cta-card {
    flex-direction: column;
    text-align: center;
    padding: 2rem 1.5rem;
  }

  .cta-text { max-width: 100%; }
  .cta-btn { width: 100%; justify-content: center; }
}

@media (min-width: 769px) and (max-width: 1024px) {
  .hero-title {
    font-size: 2.75rem;
  }

  .benefits-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .route-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

/* ═══════════════════════════════════════════════════════
   ENHANCED CARD EFFECTS
   ═══════════════════════════════════════════════════════ */
.route-card {
  position: relative;
  overflow: hidden;
}

.route-card::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  padding: 1.5px;
  background: linear-gradient(135deg, transparent 40%, rgba(96,165,250,0.3) 50%, transparent 60%);
  -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  opacity: 0;
  transition: opacity 0.5s ease;
  pointer-events: none;
}

.route-card:hover::before {
  opacity: 1;
}

.route-card:hover {
  transform: translateY(-6px) scale(1.01);
  box-shadow: 0 20px 40px rgba(0,0,0,0.12), 0 0 0 1px rgba(96,165,250,0.1);
}

.benefit-card {
  position: relative;
  overflow: hidden;
}

.benefit-card::after {
  content: '';
  position: absolute;
  top: -50%;
  left: -50%;
  width: 200%;
  height: 200%;
  background: radial-gradient(circle, rgba(59,130,246,0.06) 0%, transparent 60%);
  opacity: 0;
  transition: opacity 0.5s ease;
  pointer-events: none;
}

.benefit-card:hover::after {
  opacity: 1;
}

.benefit-card:hover {
  transform: translateY(-6px);
  box-shadow: 0 20px 40px rgba(0,0,0,0.1);
}

/* Step number pulse animation when revealed */
.step-item.reveal.is-revealed .step-number {
  animation: stepPop 0.5s cubic-bezier(0.22, 1, 0.36, 1) var(--delay, 0s) both;
}

@keyframes stepPop {
  0% { transform: scale(0.5); opacity: 0; }
  60% { transform: scale(1.1); }
  100% { transform: scale(1); opacity: 1; }
}

/* Section title underline animation */
.section-title {
  position: relative;
  display: inline-block;
  left: 50%;
  transform: translateX(-50%);
}

.section-title::after {
  content: '';
  display: block;
  width: 40px;
  height: 3px;
  background: var(--slate-300);
  border-radius: 2px;
  margin: 0.6rem auto 0;
  transition: width 0.5s cubic-bezier(0.22, 1, 0.36, 1);
}

.reveal.is-revealed .section-title::after {
  width: 64px;
}

/* CTA card enhanced glow */
.cta-card {
  transition: transform 0.4s cubic-bezier(0.22, 1, 0.36, 1), box-shadow 0.4s ease;
}

.cta-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 30px 60px rgba(15,23,42,0.35);
}

/* Status bar enhanced */
.status-bar {
  transition: all 0.4s ease;
}

.status-bar:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0,0,0,0.08);
}
</style>
