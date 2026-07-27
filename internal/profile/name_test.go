package profile

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateName_RejectsFormatViolations(t *testing.T) {
	cases := map[string]string{
		"uppercase":      "Work",
		"space":          "my profile",
		"path separator": "a/b",
		"dot-dot":        "..",
		"empty":          "",
		"too long":       strings.Repeat("a", 33),
		"leading dash":   "-abc",
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			err := ValidateName(input)
			if !errors.Is(err, ErrInvalidName) {
				t.Fatalf("ValidateName(%q) = %v, want ErrInvalidName", input, err)
			}
			if !strings.Contains(err.Error(), nameFormatHint) {
				t.Errorf("ValidateName(%q) error = %q, want it to contain the allowed format", input, err.Error())
			}
		})
	}
}

func TestValidateName_RejectsReservedName(t *testing.T) {
	err := ValidateName(DefaultName)
	if !errors.Is(err, ErrReservedName) {
		t.Fatalf("ValidateName(%q) = %v, want ErrReservedName", DefaultName, err)
	}
	if !strings.Contains(err.Error(), "reserved") {
		t.Errorf("ValidateName(%q) error = %q, want it to mention reserved", DefaultName, err.Error())
	}
}

func TestValidateName_AcceptsWellFormedNames(t *testing.T) {
	for _, name := range []string{"work", "a", "my-profile_2"} {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) = %v, want nil", name, err)
		}
	}
}
