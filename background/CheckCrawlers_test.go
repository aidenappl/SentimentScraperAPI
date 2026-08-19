package background

import "testing"

// This file existing and running is itself the regression test: background
// imports env, which used to panic during package initialisation when CORE_DB
// was unset. That panic fires before TestMain, so no test in this package
// could run at all — which is why the crawl-batch bug shipped without one.
func TestDomainOf(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"standard", "https://www.reuters.com/world/story", "www.reuters.com"},
		{"with port", "https://example.com:8443/story", "example.com"},
		{"unparseable", "://nope", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := domainOf(tt.url); got != tt.want {
				t.Fatalf("domainOf(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
