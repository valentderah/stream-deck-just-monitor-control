package inspector

import (
	"encoding/json"
	"strings"
	"testing"
)

type sampleMode string

const (
	sampleModeA sampleMode = "a"
	sampleModeB sampleMode = "b"
)

type sampleSettings struct {
	Mode      sampleMode `json:"mode"`
	Port      uint32     `json:"port"`
	Level     uint32     `json:"level"`
	MonitorID string     `json:"monitorId"`
	State     int        `json:"state"`
}

func sampleDefaults() sampleSettings {
	return sampleSettings{Mode: sampleModeA, Port: 15, Level: 50}
}

func modeField() Field {
	return Select("mode", "Mode", TypeString,
		LocalizedOption(sampleModeA, "A"),
		LocalizedOption(sampleModeB, "B"),
	).WithZeroAsUnset()
}

func sampleSchema() Schema {
	return MustBuild(sampleDefaults(),
		MonitorField(),
		modeField(),
		SourceSelect("port", "Port", SourceMonitorInputs, []Option{LiteralOption(15, "DisplayPort 1")}).
			WithZeroAsUnset().
			VisibleIf("mode", sampleModeB),
		Number("level", "Level").WithRange(0, 100),
	)
}

func TestBuildFillsDefaultsFromSettings(t *testing.T) {
	s := sampleSchema()
	if s.Fields[0].Default != nil {
		t.Errorf("monitor default = %v, want nil", s.Fields[0].Default)
	}
	if s.Fields[1].Default != "a" {
		t.Errorf("mode default = %#v", s.Fields[1].Default)
	}
	if s.Fields[2].Default != int64(15) {
		t.Errorf("port default = %#v", s.Fields[2].Default)
	}
	if s.Fields[3].Default != int64(50) {
		t.Errorf("level default = %#v", s.Fields[3].Default)
	}
}

func TestFieldJSON(t *testing.T) {
	got, err := json.Marshal(sampleSchema().Fields[2])
	if err != nil {
		t.Fatal(err)
	}
	want := `{"key":"port","kind":"select","type":"int","label":"Port","default":15,"zeroIsUnset":true,"source":"monitorInputs","fallback":[{"value":15,"label":"DisplayPort 1"}],"visibleWhen":{"key":"mode","values":["b"]}}`
	if string(got) != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestBuildRejectsFieldWithoutSettingsDefault(t *testing.T) {
	if _, err := Build(sampleDefaults(), Number("missing", "Missing")); err == nil {
		t.Fatal("Build returned nil error")
	}
}

func TestDecode(t *testing.T) {
	cases := []struct {
		name  string
		saved string
		want  sampleSettings
	}{
		{"missing keys keep defaults", `{"monitorId":"m1"}`, sampleSettings{Mode: "a", Port: 15, Level: 50, MonitorID: "m1"}},
		{"zero as unset", `{"mode":"","port":0,"level":0}`, sampleSettings{Mode: "a", Port: 15, Level: 0}},
		{"null as unset", `{"mode":null}`, sampleSettings{Mode: "a", Port: 15, Level: 50}},
		{"keys outside the schema survive", `{"state":1}`, sampleSettings{Mode: "a", Port: 15, Level: 50, State: 1}},
		{"empty payload", ``, sampleDefaults()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Decode(sampleSchema(), sampleDefaults(), json.RawMessage(c.saved))
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Fatalf("got %+v want %+v", got, c.want)
			}
		})
	}
}

func TestDecodeRejectsWrongType(t *testing.T) {
	got, err := Decode(sampleSchema(), sampleDefaults(), json.RawMessage(`{"port":"x"}`))
	if err == nil {
		t.Fatal("Decode returned nil error")
	}
	if got != sampleDefaults() {
		t.Fatalf("got %+v, want defaults", got)
	}
}

type rateSettings struct {
	Numerator   uint32 `json:"numerator"`
	Denominator uint32 `json:"denominator"`
	MonitorID   string `json:"monitorId"`
}

func TestDecodeRefreshRateDenominator(t *testing.T) {
	defaults := rateSettings{Denominator: 1}
	schema := MustBuild(defaults, MonitorField(), RefreshRateField())
	for saved, want := range map[string]rateSettings{
		`{"numerator":60,"denominator":0}`: {Numerator: 60, Denominator: 1},
		`{"numerator":144}`:                {Numerator: 144, Denominator: 1},
		`{"numerator":60,"denominator":2}`: {Numerator: 60, Denominator: 2},
	} {
		got, err := Decode(schema, defaults, json.RawMessage(saved))
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s: got %+v want %+v", saved, got, want)
		}
	}
}

func TestVerifyAcceptsValidSchema(t *testing.T) {
	if err := Verify(sampleSchema(), sampleDefaults()); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyReportsProblems(t *testing.T) {
	built := sampleSchema().Fields
	monitor, mode, level := built[0], built[1], built[3]

	withDefault := func(f Field, v any) Field { f.Default = v; return f }
	withZero := func(f Field) Field { f.ZeroIsUnset = true; return f }

	cases := []struct {
		name   string
		fields []Field
		want   string
	}{
		{"unknown key", []Field{{Key: "nope", Kind: KindNumber, Label: "X", Default: int64(1)}}, "not in settings"},
		{"duplicate key", []Field{monitor, monitor}, "duplicate key"},
		{"missing label", []Field{{Key: "monitorId", Kind: KindMonitor}}, "missing label"},
		{"unknown kind", []Field{{Key: "monitorId", Kind: "slider", Label: "X"}}, "unknown kind"},
		{"default mismatch", []Field{withDefault(level, int64(7))}, "does not match"},
		{"missing default", []Field{withDefault(level, nil)}, "missing default"},
		{"unexpected default", []Field{withDefault(monitor, "x")}, "unexpected default"},
		{"default not an option", []Field{withDefault(mode, "z")}, "is not an option"},
		{"options and source", []Field{func() Field { f := mode; f.Source = SourceMonitorInputs; return f }()}, "exactly one of options or source"},
		{"source without fallback", []Field{func() Field { f := built[2]; f.Fallback = nil; return f }()}, "needs a fallback"},
		{"zeroIsUnset on monitor", []Field{withZero(monitor)}, "zeroIsUnset"},
		{"refreshRate keys", []Field{{Keys: []string{"numerator"}, Kind: KindRefreshRate, Label: "RefreshRate"}}, "keys numerator, denominator"},
		{"visibleWhen unknown value", []Field{mode, level.VisibleIf("mode", "z")}, "visibleWhen value"},
		{"visibleWhen not a static select", []Field{level, mode.VisibleIf("level", 1)}, "visibleWhen must reference a static select"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := Verify(Schema{Fields: c.fields}, sampleDefaults())
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %v, want error containing %q", err, c.want)
			}
		})
	}
}
