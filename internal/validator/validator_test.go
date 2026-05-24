package validator_test

import (
	"testing"

	"github.com/yaredow/greenlight/internal/validator"
)

func TestValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{name: "Valid email", email: "test@example.com", want: true},
		{name: "Empty email", email: "", want: false},
		{name: "Missing @", email: "testexample.com", want: false},
		{name: "Missing TLD", email: "test@example", want: false},
		{name: "Multiple @", email: "test@@example.com", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validator.Matches(tt.email, validator.EmailRX); got != tt.want {
				t.Errorf("Matches(%q, EmailRX) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestValidator_Check(t *testing.T) {
	v := validator.New()

	v.Check(true, "test", "should be valid")
	if !v.Valid() {
		t.Errorf("Expected validator to be valid, but got errors: %v", v.Errors)
	}

	v.Check(false, "test", "should be invalid")
	if v.Valid() {
		t.Errorf("Expected validator to be invalid")
	}

	if msg, ok := v.Errors["test"]; !ok || msg != "should be invalid" {
		t.Errorf("Expected error message 'should be invalid', got %q", msg)
	}
}
