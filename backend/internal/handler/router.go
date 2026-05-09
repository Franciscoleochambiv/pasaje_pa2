package handler

import (
	"net/http"

	"pasaje/backend/internal/service"

	"github.com/go-chi/chi/v5"
)

// Router returns the HTTP handler with all routes.
func Router(health *HealthHandler, routes *RoutesHandler, admin *AdminHandler, reservations *ReservationHandler, auth *AuthHandler, users *UserHandler, authSvc *service.AuthService, googleAuth *GoogleAuthHandler, wsHandler *WSHandler, billing *BillingHandler, billingConfig *BillingConfigHandler, payment *PaymentHandler, settings *SettingsHandler, yapePayment *YapePaymentHandler, parcels *ParcelHandler) http.Handler {
	r := chi.NewRouter()
	r.Use(CORS)

	// WebSocket routes
	r.Get("/ws/trips/{id}/seats", wsHandler.HandleTripSeats)
	r.Get("/ws/reservations/{code}/status", wsHandler.HandleReservationStatus)
	r.Get("/ws/parcels/{code}/status", wsHandler.HandleReservationStatus) // reutiliza mismo mecanismo

	r.Get("/health/live", health.Live)
	r.Get("/health/ready", health.Ready)

	r.Route("/api", func(r chi.Router) {
		r.Get("/routes", routes.ListRoutes)
		r.Get("/routes/{id}/trips", routes.ListTripsByRoute)
		r.Get("/routes/{id}/stops", routes.ListStops)
		r.Get("/trips/{id}/seats", routes.GetTripSeats)

		// Reservaciones públicas
		r.Post("/reservations", reservations.Create)
		r.Post("/reservations/{code}/confirm", reservations.Confirm)
		r.Get("/reservations/{code}", reservations.GetByCode)

		// Billing / DNI-RUC lookup (público)
		r.Get("/billing/dni/{dni}", billing.LookupDNI)
		r.Get("/billing/ruc/{ruc}", billing.LookupRUC)
		r.Post("/billing/sale", billing.CreateBillingSale)
		r.Get("/billing/pdf/{ventaId}", billing.GetPDF)
		r.Post("/billing/send-email", billing.SendReceiptEmail)

		// Payment / Culqi (público)
		r.Get("/payment/config", payment.GetConfig)
		r.Post("/payment/charge", payment.CreateCharge)

		// Yape directo (público)
		r.Get("/payment/yape-config", yapePayment.GetYapeConfig)
		r.Post("/payment/yape-direct", yapePayment.CreateYapeDirectPayment)
		r.Get("/payment/yape-direct/{code}/status", yapePayment.GetPaymentStatus)

		// Tracking de encomiendas (público)
		r.Get("/parcels/track/{code}", parcels.TrackParcel)

		// Settings públicos (precio, empresa)
		r.Get("/settings/public", settings.GetPublic)

		// Auth (público)
		r.Post("/auth/login", auth.Login)
		r.Post("/auth/google", googleAuth.LoginWithGoogle)

		// Auth (requiere token)
		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(authSvc))
			r.Get("/auth/me", auth.Me)
		})

		// Admin (requiere token)
		r.Route("/admin", func(r chi.Router) {
			r.Use(AuthMiddleware(authSvc))

			// Rutas
			r.Post("/routes", admin.CreateRoute)
			r.Put("/routes/{id}", admin.UpdateRoute)
			r.Delete("/routes/{id}", admin.DeleteRoute)

			// Paradas y segmentos de ruta
			r.Get("/routes/{id}/stops", admin.ListStops)
			r.Post("/routes/{id}/stops/reorder", admin.ReorderStops)
			r.Post("/routes/{id}/stops", admin.CreateStop)
			r.Put("/stops/{id}", admin.UpdateStop)
			r.Delete("/stops/{id}", admin.DeleteStop)
			r.Get("/routes/{id}/segments", admin.ListSegments)
			r.Post("/routes/{id}/segments/generate", admin.GenerateSegments)
			r.Post("/routes/{id}/segments", admin.UpsertSegment)
			r.Delete("/segments/{id}", admin.DeleteSegment)

			// Vehículos
			r.Get("/vehicles", admin.ListVehicles)
			r.Post("/vehicles", admin.CreateVehicle)
			r.Put("/vehicles/{id}", admin.UpdateVehicle)
			r.Delete("/vehicles/{id}", admin.DeleteVehicle)
			r.Get("/vehicles/{id}/seats", admin.ListVehicleSeats)
			r.Post("/vehicles/{id}/save-as-layout", admin.SaveVehicleAsLayout)

			// Plantillas de bus (layouts reutilizables)
			r.Get("/bus-layouts", admin.ListBusLayouts)
			r.Post("/bus-layouts", admin.CreateBusLayout)
			r.Put("/bus-layouts/{id}", admin.UpdateBusLayout)
			r.Delete("/bus-layouts/{id}", admin.DeleteBusLayout)
			r.Get("/bus-layouts/{id}/full", admin.GetBusLayoutFull)
			r.Put("/bus-layouts/{id}/full", admin.SaveBusLayoutFull)

			// Plantillas de viaje
			r.Get("/trip-templates", admin.ListTripTemplates)
			r.Post("/trip-templates", admin.CreateTripTemplate)
			r.Put("/trip-templates/{id}", admin.UpdateTripTemplate)
			r.Delete("/trip-templates/{id}", admin.DeleteTripTemplate)

			// Instancias de viaje
			r.Get("/trip-instances", admin.ListTripInstances)
			r.Post("/trip-instances", admin.CreateTripInstance)
			r.Put("/trip-instances/{id}", admin.UpdateTripInstance)
			r.Delete("/trip-instances/{id}", admin.DeleteTripInstance)

			// Dashboard stats
			r.Get("/stats", admin.GetStats)

			// Hoja de ruta (manifiesto)
			r.Get("/trips/{id}/manifest", parcels.GetTripManifest)

			// Billing config
			r.Get("/billing-config", billingConfig.GetConfig)
			r.Put("/billing-config", billingConfig.UpdateConfig)
			r.Get("/billing-config/test", billingConfig.TestConnection)
			r.Get("/billing-config/tenants", billingConfig.ListTenants)
			r.Post("/billing-config/sync-empresa", billingConfig.SyncEmpresa)
			r.Get("/billing-config/subscription", billingConfig.GetSubscription)

			// Settings
			r.Get("/settings", settings.GetAll)
			r.Put("/settings", settings.Update)
			r.Post("/settings/sync", settings.SyncFromVenta)

			// Reservas admin (POS)
			r.Post("/reservations", reservations.CreateAdmin)
			r.Get("/reservations", reservations.ListAdmin)

			// Voucher verification
			r.Get("/vouchers/pending", yapePayment.ListPendingVouchers)
			r.Get("/vouchers/{id}/image", yapePayment.ServeVoucherImage)
			r.Post("/vouchers/{id}/approve", yapePayment.ApproveVoucher)
			r.Post("/vouchers/{id}/reject", yapePayment.RejectVoucher)
			r.Post("/payments/{id}/retry-comprobante", yapePayment.RetryComprobante)
			r.Get("/trips/{tripId}/seats/{seatInventoryId}/holder", yapePayment.GetSeatHolder)
			r.Post("/reservations/{code}/void", yapePayment.VoidReservation)

			// Encomiendas
			r.Get("/parcels", parcels.ListParcels)
			r.Get("/parcels/{id}", parcels.GetParcel)
			r.Post("/parcels", parcels.CreateParcel)
			r.Put("/parcels/{id}", parcels.UpdateParcel)
			r.Post("/parcels/{id}/status", parcels.UpdateParcelStatus)
			r.Post("/parcels/{id}/pay", parcels.PayParcel)
			r.Post("/parcels/{id}/cancel", parcels.CancelParcel)
			r.Post("/parcels/{id}/retry-billing", parcels.RetryParcelBilling)
			r.Get("/parcels/by-trip/{tripId}", parcels.ListParcelsByTrip)

			// Usuarios
			r.Get("/users", users.ListUsers)
			r.Post("/users", users.CreateUser)
			r.Put("/users/{id}", users.UpdateUser)
			r.Put("/users/{id}/password", users.ChangePassword)
			r.Delete("/users/{id}", users.DeactivateUser)
		})
	})

	return r
}
