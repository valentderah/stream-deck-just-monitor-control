package inspector

import (
	"encoding/json"
	"maps"
)

func Decode[T any](schema Schema, defaults T, saved json.RawMessage) (T, error) {
	base, err := toMap(defaults)
	if err != nil {
		return defaults, err
	}
	values := maps.Clone(base)
	if len(saved) > 0 {
		overlay, err := decodeMap(saved)
		if err != nil {
			return defaults, err
		}
		maps.Copy(values, overlay)
	}
	normalize(schema, base, values)
	b, err := json.Marshal(values)
	if err != nil {
		return defaults, err
	}
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		return defaults, err
	}
	return out, nil
}

func normalize(schema Schema, defaults, values map[string]any) {
	for _, f := range schema.Fields {
		switch f.Kind {
		case KindRefreshRate:
			if isZero(values["denominator"]) {
				values["denominator"] = int64(1)
			}
		case KindSelect, KindNumber:
			if f.ZeroIsUnset && isZero(values[f.Key]) {
				values[f.Key] = defaults[f.Key]
			}
		}
	}
}

func isZero(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case int64:
		return x == 0
	case float64:
		return x == 0
	case string:
		return x == ""
	default:
		return false
	}
}
