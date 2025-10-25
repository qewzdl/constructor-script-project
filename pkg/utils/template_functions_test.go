package utils

import (
	"reflect"
	"testing"
)

func TestDefaultTemplateFunc(t *testing.T) {
	funcs := GetTemplateFuncs(nil)
	defaultFunc, ok := funcs["default"].(func(interface{}, interface{}) interface{})
	if !ok {
		t.Fatalf("default func has unexpected signature")
	}

	testCases := []struct {
		name     string
		defaultV interface{}
		value    interface{}
		expected interface{}
	}{
		{"nil value", "fallback", nil, "fallback"},
		{"empty string", "fallback", "", "fallback"},
		{"non-empty string", "fallback", "value", "value"},
		{"boolean true", false, true, true},
		{"boolean false", true, false, false},
		{"zero int", 10, 0, 10},
		{"non-zero int", 10, 5, 5},
		{"empty slice", []string{"fallback"}, []string{}, []string{"fallback"}},
		{"non-empty slice", []string{"fallback"}, []string{"value"}, []string{"value"}},
	}

	for _, tc := range testCases {
		result := defaultFunc(tc.defaultV, tc.value)
		if !reflect.DeepEqual(result, tc.expected) {
			t.Errorf("%s: expected %v, got %v", tc.name, tc.expected, result)
		}
	}
}
