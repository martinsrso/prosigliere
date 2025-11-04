package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("AUTH_USERNAME", "testuser")
	defer os.Unsetenv("PORT")
	defer os.Unsetenv("AUTH_USERNAME")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Port)
	}

	if cfg.AuthUsername != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", cfg.AuthUsername)
	}
}

func TestLoadConfigFromEnvFile(t *testing.T) {
	envContent := `PORT=9999
AUTH_USERNAME=envuser
AUTH_PASSWORD=envpass
DATABASE_URL=postgres://test:test@localhost/test
`
	
	err := os.WriteFile(".env.test", []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}
	defer os.Remove(".env.test")

	os.Setenv("PORT", "9999")
	os.Setenv("AUTH_USERNAME", "envuser")
	defer os.Unsetenv("PORT")
	defer os.Unsetenv("AUTH_USERNAME")
	
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Port != 9999 {
		t.Errorf("Expected port 9999, got %d", cfg.Port)
	}
}
