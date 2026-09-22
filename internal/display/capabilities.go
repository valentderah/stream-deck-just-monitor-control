package display

import (
	"strconv"
	"strings"
)

// DefaultInputPorts used when MCCS capabilities cannot be read.
func DefaultInputPorts() []uint32 {
	return []uint32{0x0F, 0x10, 0x11, 0x12}
}

// InputsFromCapabilities parses VCP 0x60 values or returns DefaultInputPorts.
func InputsFromCapabilities(caps string) []uint32 {
	vals, err := ParseVCPValues(caps, 0x60)
	if err != nil || len(vals) == 0 {
		return DefaultInputPorts()
	}
	return vals
}

// ParseVCPValues extracts allowed values for a VCP code from an MCCS capabilities string.
// Looks for patterns like 60(01 03 0F 11) inside a vcp(...) block (case-insensitive hex).
func ParseVCPValues(caps string, code byte) ([]uint32, error) {
	lower := strings.ToLower(caps)
	codeHex := strings.ToLower(strconv.FormatUint(uint64(code), 16))
	if len(codeHex) == 1 {
		codeHex = "0" + codeHex
	}
	// Prefer exact two-digit form; also accept single digit for codes < 0x10
	needle := codeHex + "("
	idx := strings.Index(lower, needle)
	if idx < 0 && len(codeHex) == 2 && codeHex[0] == '0' {
		needle = string(codeHex[1]) + "("
		idx = strings.Index(lower, needle)
	}
	if idx < 0 {
		return nil, errCapabilitiesMiss
	}
	start := idx + len(needle)
	end := strings.IndexByte(lower[start:], ')')
	if end < 0 {
		return nil, errCapabilitiesMiss
	}
	body := strings.Fields(lower[start : start+end])
	out := make([]uint32, 0, len(body))
	for _, tok := range body {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		v, err := strconv.ParseUint(tok, 16, 32)
		if err != nil {
			return nil, err
		}
		out = append(out, uint32(v))
	}
	if len(out) == 0 {
		return nil, errCapabilitiesMiss
	}
	return out, nil
}

var errCapabilitiesMiss = errString("display: vcp values not found in capabilities")

type errString string

func (e errString) Error() string { return string(e) }
