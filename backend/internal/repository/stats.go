package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StatsRepository acceso a estadísticas del dashboard.
type StatsRepository struct {
	Pool *pgxpool.Pool
}

// DashboardStats contiene los conteos para el dashboard admin.
type DashboardStats struct {
	TotalRoutes       int64 `json:"total_routes"`
	TotalVehicles     int64 `json:"total_vehicles"`
	UpcomingTrips     int64 `json:"upcoming_trips"`
	ReservationsToday int64 `json:"reservations_today"`

	// Extended stats for charts
	SeatsAvailable   int64   `json:"seats_available"`
	SeatsSold        int64   `json:"seats_sold"`
	SeatsHeld        int64   `json:"seats_held"`
	SeatsBlocked     int64   `json:"seats_blocked"`
	TotalSalesAmount float64 `json:"total_sales_amount"`
	SalesToday       float64 `json:"sales_today"`
	SalesWeek        float64 `json:"sales_week"`
	SalesMonth       float64 `json:"sales_month"`
	ConfirmedToday   int64   `json:"confirmed_today"`
	PendingToday     int64   `json:"pending_today"`
	ExpiredToday     int64   `json:"expired_today"`
	CancelledToday   int64   `json:"cancelled_today"`
}

// GetDashboardStats devuelve las estadísticas del dashboard.
func (r *StatsRepository) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	var s DashboardStats

	r.Pool.QueryRow(ctx, `SELECT count(*) FROM routes WHERE active = true`).Scan(&s.TotalRoutes)
	r.Pool.QueryRow(ctx, `SELECT count(*) FROM vehicles WHERE active = true`).Scan(&s.TotalVehicles)
	r.Pool.QueryRow(ctx, `SELECT count(*) FROM trip_instances WHERE status = 'scheduled' AND departure_at >= now()`).Scan(&s.UpcomingTrips)
	r.Pool.QueryRow(ctx, `SELECT count(*) FROM reservations WHERE created_at::date = CURRENT_DATE`).Scan(&s.ReservationsToday)

	// Seat stats (for upcoming trips only)
	r.Pool.QueryRow(ctx, `SELECT COALESCE(count(*),0) FROM trip_seat_inventory tsi JOIN trip_instances ti ON ti.id = tsi.trip_instance_id WHERE ti.status = 'scheduled' AND ti.departure_at >= now() AND tsi.status = 'available'`).Scan(&s.SeatsAvailable)
	r.Pool.QueryRow(ctx, `SELECT COALESCE(count(*),0) FROM trip_seat_inventory tsi JOIN trip_instances ti ON ti.id = tsi.trip_instance_id WHERE ti.status = 'scheduled' AND ti.departure_at >= now() AND tsi.status = 'sold'`).Scan(&s.SeatsSold)
	r.Pool.QueryRow(ctx, `SELECT COALESCE(count(*),0) FROM trip_seat_inventory tsi JOIN trip_instances ti ON ti.id = tsi.trip_instance_id WHERE ti.status = 'scheduled' AND ti.departure_at >= now() AND tsi.status = 'held'`).Scan(&s.SeatsHeld)
	r.Pool.QueryRow(ctx, `SELECT COALESCE(count(*),0) FROM trip_seat_inventory tsi JOIN trip_instances ti ON ti.id = tsi.trip_instance_id WHERE ti.status = 'scheduled' AND ti.departure_at >= now() AND tsi.status = 'blocked'`).Scan(&s.SeatsBlocked)

	// Sales amounts
	r.Pool.QueryRow(ctx, `SELECT COALESCE(SUM(amount_cents),0)/100.0 FROM payments WHERE status IN ('completed','approved')`).Scan(&s.TotalSalesAmount)
	r.Pool.QueryRow(ctx, `SELECT COALESCE(SUM(amount_cents),0)/100.0 FROM payments WHERE status IN ('completed','approved') AND created_at::date = CURRENT_DATE`).Scan(&s.SalesToday)
	r.Pool.QueryRow(ctx, `SELECT COALESCE(SUM(amount_cents),0)/100.0 FROM payments WHERE status IN ('completed','approved') AND created_at >= now() - interval '7 days'`).Scan(&s.SalesWeek)
	r.Pool.QueryRow(ctx, `SELECT COALESCE(SUM(amount_cents),0)/100.0 FROM payments WHERE status IN ('completed','approved') AND created_at >= now() - interval '30 days'`).Scan(&s.SalesMonth)

	// Reservation status breakdown today
	r.Pool.QueryRow(ctx, `SELECT count(*) FROM reservations WHERE status = 'confirmed' AND created_at::date = CURRENT_DATE`).Scan(&s.ConfirmedToday)
	r.Pool.QueryRow(ctx, `SELECT count(*) FROM reservations WHERE status IN ('pending','pending_verification') AND created_at::date = CURRENT_DATE`).Scan(&s.PendingToday)
	r.Pool.QueryRow(ctx, `SELECT count(*) FROM reservations WHERE status = 'expired' AND created_at::date = CURRENT_DATE`).Scan(&s.ExpiredToday)
	r.Pool.QueryRow(ctx, `SELECT count(*) FROM reservations WHERE status = 'cancelled' AND created_at::date = CURRENT_DATE`).Scan(&s.CancelledToday)

	return &s, nil
}
