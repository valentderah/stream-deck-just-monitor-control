package streamdeck

import "encoding/json"

type Event struct {
	Event   string          `json:"event"`
	Action  string          `json:"action"`
	Context string          `json:"context"`
	Device  string          `json:"device"`
	Payload json.RawMessage `json:"payload"`
}
