package streamdeck

import "encoding/json"

const (
	ControllerKeypad  = "Keypad"
	ControllerEncoder = "Encoder"
)

type Event struct {
	Event   string          `json:"event"`
	Action  string          `json:"action"`
	Context string          `json:"context"`
	Device  string          `json:"device"`
	Payload json.RawMessage `json:"payload"`
}

type eventPayload struct {
	Controller string `json:"controller"`
	Ticks      int    `json:"ticks"`
}

func (e Event) payload() eventPayload {
	var p eventPayload
	if len(e.Payload) > 0 {
		_ = json.Unmarshal(e.Payload, &p)
	}
	return p
}

// Controller reports where the action sits. Stream Deck omits it on older devices, which only have keys.
func (e Event) Controller() string {
	if e.payload().Controller == ControllerEncoder {
		return ControllerEncoder
	}
	return ControllerKeypad
}

// Ticks is the signed rotation of a dialRotate event.
func (e Event) Ticks() int {
	return e.payload().Ticks
}
