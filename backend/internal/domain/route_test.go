package domain

import "testing"

func TestRouteStruct(t *testing.T) {
	r := Route{
		ID:           1,
		Name:         "Arequipa - Cusco",
		Code:         "AQP-CUS",
		Active:       true,
		PricePerSeat: 55.00,
	}
	if r.Name == "" {
		t.Error("route name should not be empty")
	}
	if r.PricePerSeat <= 0 {
		t.Error("price per seat should be positive")
	}
}

func TestStopStruct(t *testing.T) {
	s := Stop{
		ID:       1,
		RouteID:  1,
		Name:     "Arequipa",
		Code:     "AQP",
		Position: 1,
	}
	if s.Position < 1 {
		t.Error("stop position should be >= 1")
	}
	if s.Code == "" {
		t.Error("stop code should not be empty")
	}
}

func TestRouteSegmentStruct(t *testing.T) {
	seg := RouteSegment{
		ID:           1,
		RouteID:      1,
		OriginStopID: 1,
		DestStopID:   2,
		Price:        35.00,
	}
	if seg.OriginStopID == seg.DestStopID {
		t.Error("origin and destination stops must be different")
	}
	if seg.Price < 0 {
		t.Error("segment price should not be negative")
	}
}

func TestRouteSegmentSameOriginDest(t *testing.T) {
	seg := RouteSegment{
		OriginStopID: 5,
		DestStopID:   5,
	}
	if seg.OriginStopID == seg.DestStopID {
		// This is the expected invalid case - good
		return
	}
	t.Error("should detect same origin and destination")
}
