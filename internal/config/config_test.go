package config

import "testing"

func TestLoadProjectConfigYAML(t *testing.T) {
	cfg, err := LoadFile("../../config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("expected HTTP addr :8080, got %s", cfg.HTTP.Addr)
	}
	if cfg.Database.DSN == "" {
		t.Fatal("expected database DSN to be configured")
	}
	if !cfg.BusinessConfigDB {
		t.Fatal("business configuration should be managed through database-backed APIs")
	}
}
