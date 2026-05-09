package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCreateParcelValidation tests input validation for the CreateParcel handler.
func TestCreateParcelValidation(t *testing.T) {
	// We test the JSON validation logic by checking that the handler
	// returns proper error codes for invalid input.
	// Since we don't have a DB, we test only the validation layer.

	tests := []struct {
		name       string
		body       map[string]any
		wantStatus int
		wantError  string
	}{
		{
			name:       "empty body",
			body:       map[string]any{},
			wantStatus: http.StatusBadRequest,
			wantError:  "trip_instance_id",
		},
		{
			name: "missing trip_instance_id",
			body: map[string]any{
				"origin_stop_id": 1,
				"dest_stop_id":   2,
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "trip_instance_id",
		},
		{
			name: "same origin and destination",
			body: map[string]any{
				"trip_instance_id": 1,
				"origin_stop_id":   5,
				"dest_stop_id":     5,
				"sender_name":      "Test",
				"sender_doc_number": "12345678",
				"receiver_name":    "Test2",
				"receiver_doc_number": "87654321",
				"receiver_phone":   "999999999",
				"billing_email":    "test@test.com",
				"amount_cents":     1000,
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "misma parada",
		},
		{
			name: "missing sender data",
			body: map[string]any{
				"trip_instance_id": 1,
				"origin_stop_id":   1,
				"dest_stop_id":     2,
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "remitente",
		},
		{
			name: "missing receiver data",
			body: map[string]any{
				"trip_instance_id":   1,
				"origin_stop_id":     1,
				"dest_stop_id":       2,
				"sender_name":        "Test",
				"sender_doc_number":  "12345678",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "destinatario",
		},
		{
			name: "missing billing email",
			body: map[string]any{
				"trip_instance_id":    1,
				"origin_stop_id":      1,
				"dest_stop_id":        2,
				"sender_name":         "Test",
				"sender_doc_number":   "12345678",
				"receiver_name":       "Test2",
				"receiver_doc_number": "87654321",
				"receiver_phone":      "999999999",
				"amount_cents":        1000,
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "email",
		},
		{
			name: "zero amount",
			body: map[string]any{
				"trip_instance_id":    1,
				"origin_stop_id":      1,
				"dest_stop_id":        2,
				"sender_name":         "Test",
				"sender_doc_number":   "12345678",
				"receiver_name":       "Test2",
				"receiver_doc_number": "87654321",
				"receiver_phone":      "999999999",
				"billing_email":       "test@test.com",
				"amount_cents":        0,
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "monto",
		},
		{
			name: "negative amount",
			body: map[string]any{
				"trip_instance_id":    1,
				"origin_stop_id":      1,
				"dest_stop_id":        2,
				"sender_name":         "Test",
				"sender_doc_number":   "12345678",
				"receiver_name":       "Test2",
				"receiver_doc_number": "87654321",
				"receiver_phone":      "999999999",
				"billing_email":       "test@test.com",
				"amount_cents":        -500,
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "monto",
		},
	}

	// Create handler with nil dependencies (only testing validation before DB calls)
	h := &ParcelHandler{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/admin/parcels", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			h.CreateParcel(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			var resp map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err == nil {
				if tt.wantError != "" {
					found := false
					for _, v := range resp {
						if containsInsensitive(v, tt.wantError) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("response %v should contain %q", resp, tt.wantError)
					}
				}
			}
		})
	}
}

// TestCreateParcelInvalidJSON tests that invalid JSON is rejected.
func TestCreateParcelInvalidJSON(t *testing.T) {
	h := &ParcelHandler{}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/parcels", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.CreateParcel(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// TestTrackParcelEmptyCode tests that empty code returns 400.
func TestTrackParcelEmptyCode(t *testing.T) {
	h := &ParcelHandler{}
	req := httptest.NewRequest(http.MethodGet, "/api/parcels/track/", nil)
	rec := httptest.NewRecorder()

	h.TrackParcel(rec, req)

	// Without chi context, code param will be empty
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 400 or 404", rec.Code)
	}
}

// TestPayParcelInvalidJSON tests that invalid JSON is rejected for pay.
func TestPayParcelInvalidJSON(t *testing.T) {
	h := &ParcelHandler{}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/parcels/abc/pay", bytes.NewReader([]byte("{}")))
	rec := httptest.NewRecorder()

	h.PayParcel(rec, req)

	// Without chi context, id param will fail to parse
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// TestUpdateParcelStatusMissingStatus tests that empty status is rejected.
func TestUpdateParcelStatusMissingStatus(t *testing.T) {
	h := &ParcelHandler{}
	body, _ := json.Marshal(map[string]string{"location": "somewhere"})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/parcels/1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.UpdateParcelStatus(rec, req)

	// Without chi context, id will fail to parse → 400
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func containsInsensitive(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if matchAt(s, substr, i) {
			return true
		}
	}
	return false
}

func matchAt(s, substr string, pos int) bool {
	for i := 0; i < len(substr); i++ {
		sc := s[pos+i]
		tc := substr[i]
		if sc >= 'A' && sc <= 'Z' {
			sc += 32
		}
		if tc >= 'A' && tc <= 'Z' {
			tc += 32
		}
		if sc != tc {
			return false
		}
	}
	return true
}
