package domain

import "testing"

func TestValidParcelTransitions(t *testing.T) {
	tests := []struct {
		from    string
		to      string
		allowed bool
	}{
		// Happy path: full lifecycle
		{"registered", "boarded", true},
		{"boarded", "in_transit", true},
		{"in_transit", "arrived", true},
		{"arrived", "ready_for_pickup", true},
		{"ready_for_pickup", "delivered", true},

		// Cancellation from any pre-delivered state
		{"registered", "cancelled", true},
		{"boarded", "cancelled", true},
		{"in_transit", "cancelled", true},
		{"arrived", "cancelled", true},
		{"ready_for_pickup", "cancelled", true},

		// Invalid: skip states
		{"registered", "in_transit", false},
		{"registered", "arrived", false},
		{"registered", "delivered", false},
		{"boarded", "arrived", false},
		{"boarded", "delivered", false},
		{"in_transit", "delivered", false},

		// Invalid: go backwards
		{"boarded", "registered", false},
		{"in_transit", "boarded", false},
		{"arrived", "in_transit", false},
		{"delivered", "arrived", false},

		// Invalid: delivered cannot transition
		{"delivered", "cancelled", false},
		{"delivered", "registered", false},

		// Invalid: cancelled cannot transition
		{"cancelled", "registered", false},
		{"cancelled", "boarded", false},
	}

	for _, tt := range tests {
		t.Run(tt.from+"->"+tt.to, func(t *testing.T) {
			valid := false
			for _, v := range ValidParcelTransitions[tt.from] {
				if v == tt.to {
					valid = true
					break
				}
			}
			if valid != tt.allowed {
				t.Errorf("transition %s -> %s: got allowed=%v, want %v", tt.from, tt.to, valid, tt.allowed)
			}
		})
	}
}

func TestParcelStatusLabels(t *testing.T) {
	expectedStatuses := []string{
		"registered", "boarded", "in_transit", "arrived",
		"ready_for_pickup", "delivered", "cancelled",
	}

	for _, status := range expectedStatuses {
		label, ok := ParcelStatusLabels[status]
		if !ok {
			t.Errorf("missing label for status %q", status)
		}
		if label == "" {
			t.Errorf("empty label for status %q", status)
		}
	}
}

func TestAllTransitionableStatesHaveEntries(t *testing.T) {
	// Verify all states that should have transitions do have them
	statesWithTransitions := []string{
		"registered", "boarded", "in_transit", "arrived", "ready_for_pickup",
	}
	for _, s := range statesWithTransitions {
		transitions, ok := ValidParcelTransitions[s]
		if !ok || len(transitions) == 0 {
			t.Errorf("status %q should have transitions but has none", s)
		}
	}

	// Verify terminal states have no transitions
	terminalStates := []string{"delivered", "cancelled"}
	for _, s := range terminalStates {
		transitions := ValidParcelTransitions[s]
		if len(transitions) > 0 {
			t.Errorf("terminal status %q should have no transitions but has %v", s, transitions)
		}
	}
}
