package service

import (
	"context"
	"time"

	"pasaje/backend/internal/domain"
	"pasaje/backend/internal/repository"
)

// RoutesService lógica de negocio de rutas y viajes.
type RoutesService struct {
	RouteRepo *repository.RouteRepository
	TripRepo  *repository.TripRepository
}

// ListRoutes devuelve rutas activas.
func (s *RoutesService) ListRoutes(ctx context.Context) ([]domain.Route, error) {
	return s.RouteRepo.List(ctx)
}

// ListTripsByRoute devuelve salidas de una ruta desde hoy.
func (s *RoutesService) ListTripsByRoute(ctx context.Context, routeID int64, fromDate time.Time) ([]domain.TripInstance, error) {
	if fromDate.IsZero() {
		fromDate = time.Now().Truncate(24 * time.Hour)
	}
	return s.TripRepo.ListByRoute(ctx, routeID, fromDate)
}

// GetTripSeats devuelve asientos de una salida (solo lectura).
func (s *RoutesService) GetTripSeats(ctx context.Context, tripID int64) ([]domain.SeatInfo, error) {
	if _, err := s.TripRepo.GetByID(ctx, tripID); err != nil {
		return nil, err
	}
	return s.TripRepo.ListSeatsByTripID(ctx, tripID)
}

// GetTripSeatResponse devuelve asientos con metadata del vehículo para el diagrama.
func (s *RoutesService) GetTripSeatResponse(ctx context.Context, tripID int64) (*domain.TripSeatResponse, error) {
	return s.TripRepo.GetTripSeatResponse(ctx, tripID)
}

// ListStops devuelve las paradas de una ruta.
func (s *RoutesService) ListStops(ctx context.Context, routeID int64) ([]domain.Stop, error) {
	return s.RouteRepo.ListStops(ctx, routeID)
}
