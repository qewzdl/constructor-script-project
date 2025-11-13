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

func TestStringHelperFuncs(t *testing.T) {
	funcs := GetTemplateFuncs(nil)

	hasPrefix, ok := funcs["hasPrefix"].(func(string, string) bool)
	if !ok {
		t.Fatalf("hasPrefix func has unexpected signature")
	}
	if !hasPrefix("/archive/files/sample.pdf", "/archive/") {
		t.Errorf("hasPrefix should detect valid prefix")
	}
	if hasPrefix("/archive/files/sample.pdf", "/files/") {
		t.Errorf("hasPrefix should return false for invalid prefix")
	}

	hasSuffix, ok := funcs["hasSuffix"].(func(string, string) bool)
	if !ok {
		t.Fatalf("hasSuffix func has unexpected signature")
	}
	if !hasSuffix("document.pdf", ".pdf") {
		t.Errorf("hasSuffix should detect valid suffix")
	}
	if hasSuffix("document.pdf", ".txt") {
		t.Errorf("hasSuffix should return false for invalid suffix")
	}

	contains, ok := funcs["contains"].(func(string, string) bool)
	if !ok {
		t.Fatalf("contains func has unexpected signature")
	}
	if !contains("preview of pdf", "pdf") {
		t.Errorf("contains should detect substring")
	}
	if contains("preview of pdf", "docx") {
		t.Errorf("contains should return false when substring absent")
	}
}
