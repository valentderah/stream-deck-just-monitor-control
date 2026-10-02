package inspector

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

var refreshRateKeys = []string{"numerator", "denominator"}

func Verify(schema Schema, defaults any) error {
	values, err := toMap(defaults)
	if err != nil {
		return err
	}
	var errs []error
	report := func(f Field, format string, args ...any) {
		name := strings.Join(f.WrittenKeys(), ",")
		errs = append(errs, fmt.Errorf("field %q: %s", name, fmt.Sprintf(format, args...)))
	}

	seen := map[string]bool{}
	byKey := map[string]Field{}
	for _, f := range schema.Fields {
		if f.Label == "" {
			report(f, "missing label")
		}
		switch f.Kind {
		case KindMonitor, KindMonitors, KindSelect, KindNumber:
			if f.Key == "" || len(f.Keys) > 0 {
				report(f, "must set key and not keys")
			}
		case KindRefreshRate:
			if f.Key != "" || !slices.Equal(f.Keys, refreshRateKeys) {
				report(f, "must set keys numerator, denominator")
			}
		default:
			report(f, "unknown kind %q", f.Kind)
			continue
		}
		for _, k := range f.WrittenKeys() {
			if _, ok := values[k]; !ok {
				report(f, "key %q is not in settings", k)
			}
			if seen[k] {
				report(f, "duplicate key %q", k)
			}
			seen[k] = true
		}
		byKey[f.Key] = f

		if f.hasDefault() {
			if f.Default == nil {
				report(f, "missing default")
			} else if want := values[f.Key]; !reflect.DeepEqual(canon(f.Default), want) {
				report(f, "default %v does not match settings default %v", f.Default, want)
			}
		} else if f.Default != nil {
			report(f, "unexpected default %v", f.Default)
		}

		if f.Kind == KindSelect {
			hasOptions, hasSource := len(f.Options) > 0, f.Source != ""
			if hasOptions == hasSource {
				report(f, "needs exactly one of options or source")
			}
			if hasSource && len(f.Fallback) == 0 {
				report(f, "source needs a fallback")
			}
			if f.Type != TypeString && f.Type != TypeInt {
				report(f, "type must be string or int")
			}
			if hasOptions && f.Default != nil && !containsValue(f.Options, f.Default) {
				report(f, "default %v is not an option", f.Default)
			}
		} else if f.Type != "" {
			report(f, "type is only allowed on select")
		}
		if f.ZeroIsUnset && !f.hasDefault() {
			report(f, "zeroIsUnset is only allowed on select and number")
		}
		if (f.Min != nil || f.Max != nil) && f.Kind != KindNumber {
			report(f, "min and max are only allowed on number")
		}
	}

	for _, f := range schema.Fields {
		if f.VisibleWhen == nil {
			continue
		}
		target, ok := byKey[f.VisibleWhen.Key]
		if !ok || target.Kind != KindSelect || len(target.Options) == 0 {
			report(f, "visibleWhen must reference a static select, got %q", f.VisibleWhen.Key)
			continue
		}
		for _, v := range f.VisibleWhen.Values {
			if !containsValue(target.Options, v) {
				report(f, "visibleWhen value %v is not an option of %q", v, target.Key)
			}
		}
	}

	for _, f := range schema.Fields {
		if !f.ZeroIsUnset || !f.hasDefault() {
			continue
		}
		var zero any = 0
		if f.Type == TypeString {
			zero = ""
		}
		saved, _ := json.Marshal(map[string]any{f.Key: zero})
		got, err := Decode(schema, values, saved)
		if err != nil {
			report(f, "decoding zero: %v", err)
			continue
		}
		if !reflect.DeepEqual(canon(got[f.Key]), values[f.Key]) {
			report(f, "zero does not decode to default")
		}
	}
	return errors.Join(errs...)
}

func containsValue(options []Option, v any) bool {
	want := canon(v)
	return slices.ContainsFunc(options, func(o Option) bool {
		return reflect.DeepEqual(canon(o.Value), want)
	})
}
