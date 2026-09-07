package config

import "testing"

func TestLoadUsesDefaultLOMSURL(t *testing.T) {
	t.Setenv("LOMS_URL", "")
	if got := Load().LOMSURL; got != "http://localhost:8083" {
		t.Fatalf("LOMSURL = %q, want %q", got, "http://localhost:8083")
	}
}

func TestLoadUsesEnvironmentLOMSURL(t *testing.T) {
	t.Setenv("LOMS_URL", "http://loms:8080")
	if got := Load().LOMSURL; got != "http://loms:8080" {
		t.Fatalf("LOMSURL = %q, want %q", got, "http://loms:8080")
	}
}
