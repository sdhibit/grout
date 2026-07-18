package esde

import (
	"encoding/json"
	"testing"
)

// mirrors gabagool's serialized mapping shape closely enough to validate the
// bindings we care about.
type testMapping struct {
	ControllerButtonMap map[string]int `json:"controller_button_map"`
	JoystickAxisMap     map[string]struct {
		PositiveButton int `json:"positive_button"`
		Threshold      int `json:"threshold"`
	} `json:"joystick_axis_map"`
}

// Virtual button values from gabagool constants (VirtualButton iota).
const (
	vbA     = 5
	vbB     = 6
	vbX     = 7
	vbY     = 8
	vbStart = 13
	vbMenu  = 15
)

// SDL game controller button indices.
const (
	sdlButtonA         = "0"
	sdlButtonB         = "1"
	sdlButtonX         = "2"
	sdlButtonY         = "3"
	sdlButtonStart     = "6"
	sdlButtonLeftStick = "7"
)

// SDL game controller axis indices.
const sdlAxisTriggerLeft = "4"

func TestInputMappingParses(t *testing.T) {
	data, err := GetInputMappingBytes()
	if err != nil {
		t.Fatalf("GetInputMappingBytes() error: %v", err)
	}

	var m testMapping
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("input mapping is not valid JSON: %v", err)
	}

	// Face buttons must be DIRECT (Xbox/Steam Deck layout), not Nintendo-swapped:
	// physical A (bottom) -> confirm (VirtualButtonA).
	direct := map[string]int{
		sdlButtonA: vbA,
		sdlButtonB: vbB,
		sdlButtonX: vbX,
		sdlButtonY: vbY,
	}
	for sdlBtn, want := range direct {
		if got := m.ControllerButtonMap[sdlBtn]; got != want {
			t.Errorf("controller_button_map[%s] = %d, want %d (direct face mapping)", sdlBtn, got, want)
		}
	}

	// Start must remain usable (platform-mapping confirm).
	if got := m.ControllerButtonMap[sdlButtonStart]; got != vbStart {
		t.Errorf("controller_button_map[START] = %d, want %d", got, vbStart)
	}

	// Menu must be reachable without the Steam (GUIDE) button: L2 trigger and
	// the left-stick click both open it, so BIOS download works on the Deck.
	if got := m.JoystickAxisMap[sdlAxisTriggerLeft].PositiveButton; got != vbMenu {
		t.Errorf("joystick_axis_map[TRIGGERLEFT].positive_button = %d, want %d (Menu)", got, vbMenu)
	}
	if m.JoystickAxisMap[sdlAxisTriggerLeft].Threshold <= 0 {
		t.Errorf("TRIGGERLEFT threshold must be positive, got %d", m.JoystickAxisMap[sdlAxisTriggerLeft].Threshold)
	}
	if got := m.ControllerButtonMap[sdlButtonLeftStick]; got != vbMenu {
		t.Errorf("controller_button_map[LEFTSTICK] = %d, want %d (Menu fallback)", got, vbMenu)
	}
}
