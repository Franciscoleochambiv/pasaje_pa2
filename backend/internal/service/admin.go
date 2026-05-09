package service

import (
	"context"
	"time"

	"pasaje/backend/internal/domain"
	"pasaje/backend/internal/repository"
)

// AdminService lógica de negocio para el panel de administración.
type AdminService struct {
	RouteRepo        *repository.RouteRepository
	VehicleRepo      *repository.VehicleRepository
	TripTemplateRepo *repository.TripTemplateRepository
	TripRepo         *repository.TripRepository
	StatsRepo        *repository.StatsRepository
	BusLayoutRepo    *repository.BusLayoutRepository
}

// --- Rutas ---

// CreateRoute crea una nueva ruta.
func (s *AdminService) CreateRoute(ctx context.Context, route *domain.Route) (int64, error) {
	return s.RouteRepo.Create(ctx, route)
}

// UpdateRoute actualiza una ruta.
func (s *AdminService) UpdateRoute(ctx context.Context, route *domain.Route) error {
	return s.RouteRepo.Update(ctx, route)
}

// DeactivateRoute desactiva una ruta.
func (s *AdminService) DeactivateRoute(ctx context.Context, id int64) error {
	return s.RouteRepo.Deactivate(ctx, id)
}

// --- Vehículos ---

// ListVehicles devuelve todos los vehículos.
func (s *AdminService) ListVehicles(ctx context.Context) ([]domain.Vehicle, error) {
	return s.VehicleRepo.List(ctx)
}

// CreateVehicle crea un vehículo con sus asientos. Si layoutID > 0, ignora `seats`
// y clona la geometría completa desde la plantilla.
func (s *AdminService) CreateVehicle(ctx context.Context, v *domain.Vehicle, seats []domain.VehicleSeat, layoutID int64) (int64, error) {
	if layoutID > 0 {
		layout, err := s.BusLayoutRepo.Get(ctx, layoutID)
		if err != nil {
			return 0, err
		}
		v.Floors = layout.Floors
		v.LayoutCols = layout.LayoutCols
		v.SeatType = layout.SeatType
	}
	id, err := s.VehicleRepo.Create(ctx, v)
	if err != nil {
		return 0, err
	}
	if layoutID > 0 {
		if err := s.BusLayoutRepo.CloneToVehicle(ctx, layoutID, id); err != nil {
			return 0, err
		}
		return id, nil
	}
	for i := range seats {
		seats[i].VehicleID = id
		if _, err := s.VehicleRepo.CreateSeat(ctx, &seats[i]); err != nil {
			return 0, err
		}
	}
	return id, nil
}

// UpdateVehicle actualiza un vehículo. Si layoutID > 0 y difiere del layout
// actual, re-clona la plantilla al vehículo (borra y recrea vehicle_seats y
// vehicle_layout_elements). layoutID == 0 conserva el layout actual.
func (s *AdminService) UpdateVehicle(ctx context.Context, v *domain.Vehicle, layoutID int64) error {
	// Si el caller pasó layoutID, primero leemos el actual para saber si cambió.
	var needClone bool
	if layoutID > 0 {
		current, err := s.VehicleRepo.Get(ctx, v.ID)
		if err != nil {
			return err
		}
		if current.LayoutID == nil || *current.LayoutID != layoutID {
			needClone = true
			// Al re-aplicar plantilla, sincronizamos floors/layout_cols/seat_type
			// con la plantilla nueva — si no, los grids quedarían inconsistentes.
			layout, err := s.BusLayoutRepo.Get(ctx, layoutID)
			if err != nil {
				return err
			}
			v.Floors = layout.Floors
			v.LayoutCols = layout.LayoutCols
			v.SeatType = layout.SeatType
		}
	}
	if err := s.VehicleRepo.Update(ctx, v); err != nil {
		return err
	}
	if needClone {
		if err := s.BusLayoutRepo.CloneToVehicle(ctx, layoutID, v.ID); err != nil {
			return err
		}
	}
	return nil
}

// DeactivateVehicle marca el vehículo como inactivo sin tocar el resto de campos.
func (s *AdminService) DeactivateVehicle(ctx context.Context, id int64) error {
	return s.VehicleRepo.Deactivate(ctx, id)
}

// ListVehicleSeats devuelve los asientos de un vehículo.
func (s *AdminService) ListVehicleSeats(ctx context.Context, vehicleID int64) ([]domain.VehicleSeat, error) {
	return s.VehicleRepo.ListSeats(ctx, vehicleID)
}

// --- Plantillas de viaje ---

// ListTripTemplates devuelve todas las plantillas.
func (s *AdminService) ListTripTemplates(ctx context.Context) ([]domain.TripTemplate, error) {
	return s.TripTemplateRepo.List(ctx)
}

// CreateTripTemplate crea una plantilla de viaje.
func (s *AdminService) CreateTripTemplate(ctx context.Context, t *domain.TripTemplate) (int64, error) {
	return s.TripTemplateRepo.Create(ctx, t)
}

// UpdateTripTemplate actualiza una plantilla de viaje.
func (s *AdminService) UpdateTripTemplate(ctx context.Context, t *domain.TripTemplate) error {
	return s.TripTemplateRepo.Update(ctx, t)
}

// --- Instancias de viaje ---

// ListTripInstances devuelve todas las instancias de viaje.
func (s *AdminService) ListTripInstances(ctx context.Context, from time.Time) ([]domain.TripInstanceAdmin, error) {
	return s.TripRepo.ListAllInstances(ctx, from)
}

// CreateTripInstance crea una instancia con inventario de asientos.
func (s *AdminService) CreateTripInstance(ctx context.Context, tripTemplateID int64, departureAt time.Time) (int64, error) {
	return s.TripRepo.CreateInstance(ctx, tripTemplateID, departureAt)
}

// UpdateTripInstanceStatus actualiza el estado de una instancia.
func (s *AdminService) UpdateTripInstanceStatus(ctx context.Context, id int64, status string) error {
	return s.TripRepo.UpdateInstanceStatus(ctx, id, status)
}

// UpdateTripInstanceDeparture actualiza la hora de salida de una instancia.
func (s *AdminService) UpdateTripInstanceDeparture(ctx context.Context, id int64, departureAt time.Time) error {
	return s.TripRepo.UpdateInstanceDeparture(ctx, id, departureAt)
}

// --- Paradas ---

func (s *AdminService) ListStops(ctx context.Context, routeID int64) ([]domain.Stop, error) {
	return s.RouteRepo.ListStops(ctx, routeID)
}

func (s *AdminService) CreateStop(ctx context.Context, stop *domain.Stop) (int64, error) {
	return s.RouteRepo.CreateStop(ctx, stop)
}

func (s *AdminService) UpdateStop(ctx context.Context, stop *domain.Stop) error {
	return s.RouteRepo.UpdateStop(ctx, stop)
}

func (s *AdminService) DeleteStop(ctx context.Context, id int64) error {
	return s.RouteRepo.DeleteStop(ctx, id)
}

// --- Segmentos ---

func (s *AdminService) ListSegments(ctx context.Context, routeID int64) ([]domain.RouteSegment, error) {
	return s.RouteRepo.ListSegments(ctx, routeID)
}

func (s *AdminService) UpsertSegment(ctx context.Context, seg *domain.RouteSegment) (int64, error) {
	return s.RouteRepo.UpsertSegment(ctx, seg)
}

func (s *AdminService) DeleteSegment(ctx context.Context, id int64) error {
	return s.RouteRepo.DeleteSegment(ctx, id)
}

// --- Stats ---

// GetDashboardStats devuelve estadísticas del dashboard.
func (s *AdminService) GetDashboardStats(ctx context.Context) (*repository.DashboardStats, error) {
	return s.StatsRepo.GetDashboardStats(ctx)
}

// --- Bus Layouts (plantillas reutilizables) ---

func (s *AdminService) ListBusLayouts(ctx context.Context) ([]domain.BusLayout, error) {
	return s.BusLayoutRepo.List(ctx)
}

func (s *AdminService) GetBusLayoutFull(ctx context.Context, id int64) (*domain.BusLayoutFull, error) {
	return s.BusLayoutRepo.GetFull(ctx, id)
}

func (s *AdminService) CreateBusLayout(ctx context.Context, l *domain.BusLayout) (int64, error) {
	return s.BusLayoutRepo.Create(ctx, l)
}

func (s *AdminService) UpdateBusLayout(ctx context.Context, l *domain.BusLayout) error {
	return s.BusLayoutRepo.Update(ctx, l)
}

func (s *AdminService) DeleteBusLayout(ctx context.Context, id int64) error {
	return s.BusLayoutRepo.Delete(ctx, id)
}

// SaveBusLayoutFull actualiza metadata + reemplaza asientos y elementos.
func (s *AdminService) SaveBusLayoutFull(ctx context.Context, l *domain.BusLayout, seats []domain.BusLayoutSeat, elements []domain.BusLayoutElement) error {
	if err := s.BusLayoutRepo.Update(ctx, l); err != nil {
		return err
	}
	return s.BusLayoutRepo.SaveFull(ctx, l.ID, seats, elements)
}

// SaveVehicleAsLayout crea una plantilla nueva basada en un vehículo existente.
func (s *AdminService) SaveVehicleAsLayout(ctx context.Context, vehicleID int64, name, brand, model, description, previewImageURL string) (int64, error) {
	return s.BusLayoutRepo.CreateFromVehicle(ctx, vehicleID, name, brand, model, description, previewImageURL)
}
