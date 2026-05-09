package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pasaje/backend/internal/config"
	"pasaje/backend/internal/domain"
	"pasaje/backend/internal/handler"
	"pasaje/backend/internal/repository"
	"pasaje/backend/internal/service"
	"pasaje/backend/internal/ws"
	"pasaje/backend/pkg/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	health := &handler.HealthHandler{Pool: pool}

	routeRepo := &repository.RouteRepository{Pool: pool}
	tripRepo := &repository.TripRepository{Pool: pool}
	vehicleRepo := &repository.VehicleRepository{Pool: pool}
	tripTemplateRepo := &repository.TripTemplateRepository{Pool: pool}
	reservationRepo := &repository.ReservationRepository{Pool: pool}
	statsRepo := &repository.StatsRepository{Pool: pool}
	userRepo := &repository.UserRepository{Pool: pool}
	busLayoutRepo := &repository.BusLayoutRepository{Pool: pool}

	routesSvc := &service.RoutesService{RouteRepo: routeRepo, TripRepo: tripRepo}
	adminSvc := &service.AdminService{
		RouteRepo:        routeRepo,
		VehicleRepo:      vehicleRepo,
		TripTemplateRepo: tripTemplateRepo,
		TripRepo:         tripRepo,
		StatsRepo:        statsRepo,
		BusLayoutRepo:    busLayoutRepo,
	}
	reservationSvc := &service.ReservationService{ReservationRepo: reservationRepo}
	authSvc := &service.AuthService{
		UserRepo:  userRepo,
		JWTSecret: cfg.JWTSecret,
	}
	googleAuthSvc := &service.GoogleAuthService{
		Pool:            pool,
		GoogleClientID:  cfg.GoogleClientID,
		GoogleClientIDs: cfg.GoogleClientIDs,
		JWTSecret:       cfg.JWTSecret,
	}

	// Seed default admin user if none exists
	seedDefaultAdmin(ctx, userRepo, cfg.DefaultAdminPassword)

	// WebSocket hub and notifier
	hub := ws.NewHub()
	notifier := &ws.Notifier{Hub: hub, Routes: routesSvc}

	// Start expiry worker to release expired seat holds every 30 seconds
	expirySvc := &service.ExpiryService{Pool: pool, Notifier: notifier}
	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()
	expirySvc.StartExpiryWorker(workerCtx, 30*time.Second)

	// Billing: connect to tenant DB for client lookup/creation
	var billingDBPool *pgxpool.Pool
	if cfg.BillingDBPassword != "" {
		billingDBPool, err = pgxpool.New(ctx, cfg.BillingDSN())
		if err != nil {
			log.Printf("WARN: billing tenant DB not available: %v (client lookup will use defaults)", err)
		} else {
			log.Printf("billing: connected to tenant_%s DB", cfg.BillingTenantSlug)
		}
	}

	tenantAPIURL := fmt.Sprintf("https://%s.facturame.online", cfg.BillingTenantSlug)
	billingSvc := &service.BillingService{
		GoServiceURL:   cfg.BillingGoServiceURL,
		TenantSlug:     cfg.BillingTenantSlug,
		APIPeruURL:     cfg.BillingAPIPeruURL,
		APIPeruToken:   cfg.BillingAPIPeruToken,
		TenantDBPool:   billingDBPool,
		TenantAPIURL:   tenantAPIURL,
		TenantEmail:    cfg.BillingTenantEmail,
		TenantPassword: cfg.BillingTenantPass,
	}

	routes := &handler.RoutesHandler{Routes: routesSvc}
	admin := &handler.AdminHandler{Admin: adminSvc}
	settingsRepo := &repository.SettingsRepository{Pool: pool}
	reservations := &handler.ReservationHandler{Reservations: reservationSvc, ReservationRepo: reservationRepo, Notifier: notifier, SettingsRepo: settingsRepo}
	auth := &handler.AuthHandler{Auth: authSvc}
	users := &handler.UserHandler{UserRepo: userRepo}
	googleAuth := &handler.GoogleAuthHandler{GoogleAuth: googleAuthSvc}
	wsHandler := &handler.WSHandler{Hub: hub, Routes: routesSvc}
	settingsHandler := &handler.SettingsHandler{Repo: settingsRepo, Billing: billingSvc}
	billingCfgHandler := &handler.BillingConfigHandler{Billing: billingSvc, Settings: settingsRepo}

	// Email service
	var emailSvc *service.EmailService
	if cfg.MailUsername != "" {
		emailSvc = &service.EmailService{
			Host:     cfg.MailHost,
			Port:     cfg.MailPort,
			Username: cfg.MailUsername,
			Password: cfg.MailPassword,
			FromAddr: cfg.MailFromAddr,
			FromName: cfg.MailFromName,
		}
		log.Printf("email: configured with %s", cfg.MailUsername)
	}

	billingHandler := &handler.BillingHandler{Billing: billingSvc, Email: emailSvc, ReservationRepo: reservationRepo}

	// Culqi payment gateway
	paymentSvc := &service.PaymentService{PrivateKey: cfg.CulqiPrivateKey}
	paymentHandler := &handler.PaymentHandler{
		PublicKey:       cfg.CulqiPublicKey,
		Payment:         paymentSvc,
		Reservation:     reservationSvc,
		Billing:         billingSvc,
		Notifier:        notifier,
		Email:           emailSvc,
		ReservationRepo: reservationRepo,
	}

	// Yape direct payment handler
	voucherSvc := &service.VoucherService{
		StoragePath: cfg.VoucherStoragePath,
		MaxSizeMB:   cfg.VoucherMaxSizeMB,
	}
	comprobanteSvc := &service.ComprobanteRetryService{
		ReservationRepo: reservationRepo,
		Billing:         billingSvc,
		Email:           emailSvc,
	}
	comprobanteSvc.StartWorker(workerCtx, 60*time.Second)

	yapePaymentHandler := &handler.YapePaymentHandler{
		Voucher:         voucherSvc,
		ReservationSvc:  reservationSvc,
		ReservationRepo: reservationRepo,
		Billing:         billingSvc,
		Notifier:        notifier,
		Email:           emailSvc,
		Settings:        settingsRepo,
		Comprobante:     comprobanteSvc,
		HoldMinutes:     cfg.YapeHoldMinutes,
		YapeNumber:      cfg.YapeBusinessNumber,
		WhatsAppPhone:   cfg.WhatsAppNotifyPhone,
	}

	// Parcel (encomiendas) handler
	parcelRepo := &repository.ParcelRepository{Pool: pool}
	parcelSvc := &service.ParcelService{
		ParcelRepo: parcelRepo,
		Billing:    billingSvc,
		Email:      emailSvc,
	}
	parcelHandler := &handler.ParcelHandler{
		Parcel:   parcelSvc,
		Notifier: notifier,
		Billing:  billingSvc,
		Routes:   routesSvc,
	}

	portStr := fmt.Sprintf(":%d", cfg.APIPort)
	srv := &http.Server{
		Addr:    portStr,
		Handler: handler.Router(health, routes, admin, reservations, auth, users, authSvc, googleAuth, wsHandler, billingHandler, billingCfgHandler, paymentHandler, settingsHandler, yapePaymentHandler, parcelHandler),
	}

	go func() {
		log.Printf("server listening on %s", portStr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("server stopped")
}

// seedDefaultAdmin creates a default admin user if no admin user exists.
func seedDefaultAdmin(ctx context.Context, userRepo *repository.UserRepository, password string) {
	existing, err := userRepo.GetByEmail(ctx, "admin@pasaje.pe")
	if err == nil && existing != nil {
		return // admin already exists
	}

	hash, err := service.HashPassword(password)
	if err != nil {
		log.Printf("WARN: could not hash default admin password: %v", err)
		return
	}

	user := &domain.User{
		Email:        "admin@pasaje.pe",
		Name:         "Administrador",
		Role:         "admin",
		PasswordHash: hash,
	}

	id, err := userRepo.Create(ctx, user)
	if err != nil {
		log.Printf("WARN: could not create default admin user: %v", err)
		return
	}
	log.Printf("INFO: created default admin user admin@pasaje.pe (id=%d) - change the password!", id)
}
