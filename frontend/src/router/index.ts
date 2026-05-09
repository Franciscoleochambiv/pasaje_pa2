import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import ViajesView from '../views/ViajesView.vue'
import ReservaView from '../views/ReservaView.vue'
import ConsultaView from '../views/ConsultaView.vue'
import LoginView from '../views/LoginView.vue'
import AdminLayout from '../layouts/AdminLayout.vue'
import DashboardView from '../views/admin/DashboardView.vue'
import RoutesAdminView from '../views/admin/RoutesAdminView.vue'
import VehiclesAdminView from '../views/admin/VehiclesAdminView.vue'
import TripTemplatesAdminView from '../views/admin/TripTemplatesAdminView.vue'
import TripInstancesAdminView from '../views/admin/TripInstancesAdminView.vue'
import UsersAdminView from '../views/admin/UsersAdminView.vue'
import PuntoVentaView from '../views/admin/PuntoVentaView.vue'
import ConfigBillingView from '../views/admin/ConfigBillingView.vue'
import SettingsView from '../views/admin/SettingsView.vue'
import VoucherVerificationView from '../views/admin/VoucherVerificationView.vue'
import CustomerLoginView from '../views/CustomerLoginView.vue'
import TrackingView from '../views/TrackingView.vue'
import ParcelsAdminView from '../views/admin/ParcelsAdminView.vue'
import TripManifestView from '../views/admin/TripManifestView.vue'
import ParcelFormView from '../views/admin/ParcelFormView.vue'
import ParcelDetailView from '../views/admin/ParcelDetailView.vue'
import StopsAdminView from '../views/admin/StopsAdminView.vue'
import BusLayoutsAdminView from '../views/admin/BusLayoutsAdminView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', name: 'home', component: HomeView, meta: { title: 'Inicio' } },
    { path: '/viajes', name: 'viajes', component: ViajesView, meta: { title: 'Buscar viajes' } },
    { path: '/viajes/ruta/:routeId', name: 'trips', component: ViajesView, meta: { title: 'Salidas' } },
    { path: '/viajes/salida/:tripId', name: 'seats', component: ViajesView, meta: { title: 'Asientos' } },
    { path: '/viajes/reservar/:tripId', name: 'reservar', component: ReservaView, meta: { title: 'Reservar' } },
    { path: '/consulta', name: 'consulta', component: ConsultaView, meta: { title: 'Consultar Reserva' } },
    { path: '/tracking', name: 'tracking', component: TrackingView, meta: { title: 'Rastrear Encomienda' } },
    { path: '/tracking/:code', name: 'tracking-code', component: TrackingView, meta: { title: 'Rastrear Encomienda' } },
    { path: '/login', name: 'login', component: LoginView, meta: { title: 'Iniciar Sesion' } },
    { path: '/login/cliente', name: 'customer-login', component: CustomerLoginView, meta: { title: 'Iniciar Sesion - Cliente' } },
    {
      path: '/admin',
      component: AdminLayout,
      children: [
        { path: '', name: 'admin-dashboard', component: DashboardView, meta: { title: 'Admin - Dashboard' } },
        { path: 'rutas', name: 'admin-rutas', component: RoutesAdminView, meta: { title: 'Admin - Rutas' } },
        { path: 'paradas', name: 'admin-paradas', component: StopsAdminView, meta: { title: 'Admin - Paradas' } },
        { path: 'rutas/:routeId/paradas', name: 'admin-stops', component: StopsAdminView, meta: { title: 'Admin - Paradas' } },
        { path: 'vehiculos', name: 'admin-vehiculos', component: VehiclesAdminView, meta: { title: 'Admin - Vehiculos' } },
        { path: 'plantillas-bus', name: 'admin-bus-layouts', component: BusLayoutsAdminView, meta: { title: 'Admin - Plantillas de Bus' } },
        { path: 'plantillas', name: 'admin-plantillas', component: TripTemplatesAdminView, meta: { title: 'Admin - Salidas Programadas' } },
        { path: 'viajes', name: 'admin-viajes', component: TripInstancesAdminView, meta: { title: 'Admin - Viajes' } },
        { path: 'venta', name: 'admin-venta', component: PuntoVentaView, meta: { title: 'Admin - Punto de Venta' } },
        { path: 'usuarios', name: 'admin-usuarios', component: UsersAdminView, meta: { title: 'Admin - Usuarios' } },
        { path: 'facturacion', name: 'admin-facturacion', component: ConfigBillingView, meta: { title: 'Admin - Facturacion' } },
        { path: 'vouchers', name: 'admin-vouchers', component: VoucherVerificationView, meta: { title: 'Admin - Verificar Pagos' } },
        { path: 'configuracion', name: 'admin-settings', component: SettingsView, meta: { title: 'Admin - Configuracion' } },
        { path: 'viajes/:tripId/manifiesto', name: 'admin-manifest', component: TripManifestView, meta: { title: 'Admin - Hoja de Ruta' } },
        { path: 'encomiendas', name: 'admin-encomiendas', component: ParcelsAdminView, meta: { title: 'Admin - Encomiendas' } },
        { path: 'encomiendas/nueva', name: 'admin-encomienda-nueva', component: ParcelFormView, meta: { title: 'Admin - Nueva Encomienda' } },
        { path: 'encomiendas/:id', name: 'admin-encomienda-detalle', component: ParcelDetailView, meta: { title: 'Admin - Detalle Encomienda' } },
        { path: 'encomiendas/:id/editar', name: 'admin-encomienda-editar', component: ParcelFormView, meta: { title: 'Admin - Editar Encomienda' } },
      ],
    },
  ],
})

router.beforeEach((to, _from, next) => {
  if (to.path.startsWith('/admin')) {
    const token = localStorage.getItem('auth_token')
    if (!token) {
      next({ path: '/login', query: { redirect: to.fullPath } })
      return
    }
  }
  next()
})

router.afterEach((to) => {
  const title = (to.meta?.title as string) || 'Pasaje'
  document.title = title ? `${title} - Pasaje` : 'Pasaje'
})

export default router
