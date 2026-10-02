package inspector

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func Build(defaults any, fields ...Field) (Schema, error) {
	values, err := toMap(defaults)
	if err != nil {
		return Schema{}, err
	}
	out := make([]Field, len(fields))
	for i, f := range fields {
		f.Options = canonOptions(f.Options)
		f.Fallback = canonOptions(f.Fallback)
		if f.VisibleWhen != nil {
			vis := Visibility{Key: f.VisibleWhen.Key, Values: make([]any, len(f.VisibleWhen.Values))}
			for j, v := range f.VisibleWhen.Values {
				vis.Values[j] = canon(v)
			}
			f.VisibleWhen = &vis
		}
		if f.hasDefault() {
			v, ok := values[f.Key]
			if !ok {
				return Schema{}, fmt.Errorf("inspector: field %q has no settings default", f.Key)
			}
			f.Default = v
		}
		out[i] = f
	}
	return Schema{Fields: out}, nil
}

func MustBuild(defaults any, fields ...Field) Schema {
	s, err := Build(defaults, fields...)
	if err != nil {
		panic(err)
	}
	return s
}

func canonOptions(options []Option) []Option {
	if options == nil {
		return nil
	}
	out := make([]Option, len(options))
	for i, o := range options {
		o.Value = canon(o.Value)
		out[i] = o
	}
	return out
}

func toMap(v any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return decodeMap(b)
}

func decodeMap(b []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	for k, v := range m {
		m[k] = canonNumber(v)
	}
	return m, nil
}

func canon(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var out any
	if err := dec.Decode(&out); err != nil {
		return v
	}
	return canonNumber(out)
}

func canonNumber(v any) any {
	n, ok := v.(json.Number)
	if !ok {
		return v
	}
	if i, err := n.Int64(); err == nil {
		return i
	}
	if f, err := n.Float64(); err == nil {
		return f
	}
	return v
}
