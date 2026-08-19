package env

import (
	"strings"
	"testing"
	"time"
)

// env used to call getEnvOrPanic at package initialisation, which made every
// package importing it impossible to unit test: initialisation runs before
// TestMain, so the panic fired before a test could set anything up. The fact
// that this file runs at all is the regression test.
func TestPackageInitialisesWithoutRequiredVars(t *testing.T) {
	if Port == "" {
		t.Fatal("defaults should still be populated")
	}
}

func TestValidateReportsMissingRequiredVars(t *testing.T) {
	original := CoreDB
	t.Cleanup(func() { CoreDB = original })

	CoreDB = ""
	err := Validate()
	if err == nil {
		t.Fatal("expected an error when CORE_DB is unset")
	}
	if !strings.Contains(err.Error(), "CORE_DB") {
		t.Fatalf("error should name the missing variable, got %v", err)
	}

	CoreDB = "   "
	if Validate() == nil {
		t.Fatal("a whitespace-only value should count as missing")
	}

	CoreDB = "postgres://user:pass@host:5432/db"
	if err := Validate(); err != nil {
		t.Fatalf("expected no error with CORE_DB set, got %v", err)
	}
}

func TestGetEnvDurationFallsBackOnBadInput(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool // true if the fallback should be used
	}{
		{"valid", "30s", false},
		{"unparseable", "half an hour", true},
		{"below minimum", "1s", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_DURATION", tt.value)

			got := getEnvDuration("TEST_DURATION", 5*time.Minute, 10*time.Second)
			usedFallback := got == 5*time.Minute

			if usedFallback != tt.want {
				t.Fatalf("got %v (fallback=%v), want fallback=%v", got, usedFallback, tt.want)
			}
		})
	}
}

func TestGetEnvIntFallsBackOnBadInput(t *testing.T) {
	t.Setenv("TEST_INT", "not a number")
	if got := getEnvInt("TEST_INT", 42); got != 42 {
		t.Fatalf("got %d, want the fallback 42", got)
	}

	t.Setenv("TEST_INT", "-5")
	if got := getEnvInt("TEST_INT", 42); got != 42 {
		t.Fatalf("got %d, want the fallback 42 for a non-positive value", got)
	}

	t.Setenv("TEST_INT", "7")
	if got := getEnvInt("TEST_INT", 42); got != 7 {
		t.Fatalf("got %d, want 7", got)
	}
}
