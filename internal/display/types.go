package display

// RefreshRate is a DXGI/DisplayConfig-style rational (e.g. 60000/1001 ≈ 59.94 Hz).
type RefreshRate struct {
	Numerator   uint32 `json:"numerator"`
	Denominator uint32 `json:"denominator"`
}

func (r RefreshRate) Hertz() float64 {
	den := r.Denominator
	if den == 0 {
		den = 1
	}
	return float64(r.Numerator) / float64(den)
}

type Monitor struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	DisplayNum  int      `json:"displayNum"`
	IsPrimary   bool     `json:"isPrimary"`
	Inputs      []uint32 `json:"inputs,omitempty"`
	CurrentPort uint32   `json:"currentPort,omitempty"`
	Brightness  uint32   `json:"brightness,omitempty"`
}
