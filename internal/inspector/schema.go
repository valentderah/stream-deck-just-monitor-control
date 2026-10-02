package inspector

type Kind string

const (
	KindMonitor     Kind = "monitor"
	KindMonitors    Kind = "monitors"
	KindSelect      Kind = "select"
	KindNumber      Kind = "number"
	KindRefreshRate Kind = "refreshRate"
)

type ValueType string

const (
	TypeString ValueType = "string"
	TypeInt    ValueType = "int"
)

const SourceMonitorInputs = "monitorInputs"

type Option struct {
	Value    any    `json:"value"`
	LabelKey string `json:"labelKey,omitempty"`
	Label    string `json:"label,omitempty"`
}

type Visibility struct {
	Key    string `json:"key"`
	Values []any  `json:"values"`
}

type Field struct {
	Key         string      `json:"key,omitempty"`
	Keys        []string    `json:"keys,omitempty"`
	Kind        Kind        `json:"kind"`
	Type        ValueType   `json:"type,omitempty"`
	Label       string      `json:"label"`
	Default     any         `json:"default,omitempty"`
	ZeroIsUnset bool        `json:"zeroIsUnset,omitempty"`
	Options     []Option    `json:"options,omitempty"`
	Source      string      `json:"source,omitempty"`
	Fallback    []Option    `json:"fallback,omitempty"`
	Min         *int        `json:"min,omitempty"`
	Max         *int        `json:"max,omitempty"`
	VisibleWhen *Visibility `json:"visibleWhen,omitempty"`
}

type Schema struct {
	Fields []Field `json:"fields"`
}

func LocalizedOption(value any, labelKey string) Option {
	return Option{Value: value, LabelKey: labelKey}
}

func LiteralOption(value any, label string) Option {
	return Option{Value: value, Label: label}
}

func MonitorField() Field {
	return Field{Key: "monitorId", Kind: KindMonitor, Label: "Monitors"}
}

func MonitorsField() Field {
	return Field{Key: "monitorIds", Kind: KindMonitors, Label: "Monitors"}
}

func RefreshRateField() Field {
	return Field{Keys: []string{"numerator", "denominator"}, Kind: KindRefreshRate, Label: "RefreshRate"}
}

func Select(key, label string, valueType ValueType, options ...Option) Field {
	return Field{Key: key, Kind: KindSelect, Type: valueType, Label: label, Options: options}
}

func SourceSelect(key, label, source string, fallback []Option) Field {
	return Field{Key: key, Kind: KindSelect, Type: TypeInt, Label: label, Source: source, Fallback: fallback}
}

func Number(key, label string) Field {
	return Field{Key: key, Kind: KindNumber, Label: label}
}

func (f Field) WithRange(min, max int) Field {
	f.Min, f.Max = &min, &max
	return f
}

func (f Field) WithZeroAsUnset() Field {
	f.ZeroIsUnset = true
	return f
}

func (f Field) VisibleIf(key string, values ...any) Field {
	f.VisibleWhen = &Visibility{Key: key, Values: values}
	return f
}

func (f Field) WrittenKeys() []string {
	if len(f.Keys) > 0 {
		return f.Keys
	}
	return []string{f.Key}
}

func (f Field) hasDefault() bool {
	return f.Kind == KindSelect || f.Kind == KindNumber
}
