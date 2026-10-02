package display

import "fmt"

type InputPort struct {
	Code uint32 `json:"code"`
	Name string `json:"name"`
}

var knownInputs = []InputPort{
	{Code: 0x0F, Name: "DisplayPort 1"},
	{Code: 0x10, Name: "DisplayPort 2"},
	{Code: 0x11, Name: "HDMI 1"},
	{Code: 0x12, Name: "HDMI 2"},
	{Code: 0x1B, Name: "USB-C / Type-C"},
	{Code: 0x01, Name: "VGA"},
	{Code: 0x03, Name: "DVI 1"},
}

func KnownInputPorts() []InputPort {
	return append([]InputPort(nil), knownInputs...)
}

func InputName(code uint32) string {
	for _, p := range knownInputs {
		if p.Code == code {
			return p.Name
		}
	}
	return fmt.Sprintf("Input 0x%02X", code)
}

func InputPortsFromCapabilities(caps string) ([]InputPort, bool) {
	codes, err := ParseVCPValues(caps, VCPInputSource)
	if err != nil {
		return KnownInputPorts(), false
	}
	seen := make(map[uint32]struct{}, len(codes))
	ports := make([]InputPort, 0, len(codes))
	for _, code := range codes {
		if _, dup := seen[code]; dup {
			continue
		}
		seen[code] = struct{}{}
		ports = append(ports, InputPort{Code: code, Name: InputName(code)})
	}
	return ports, true
}
