package appCore

import (
	"os"
	"testing"
)

func TestRuntimeEnvironmentDoesNotRequireBakedDotEnv(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(workingDir)
	for name, value := range map[string]string{"HOST": "localhost", "HTTP_PORT": "8085", "HTTPS_PORT": "8086", "CORE_ORIGINS": "https://localhost:8086", "API_PATH_PREFIX": "/api/v1", "DATA_SOURCE": "synthetic-test-dsn", "APP_TITLE": "Test", "DEV_MODE": "false", "EMAIL_DEV_MODE_ADDR": "", "CERT_OPTION": "0"} {
		t.Setenv(name, value)
	}
	var s settings
	if err := s.LoadEnv(); err != nil {
		t.Fatal(err)
	}
	if s.DataSource != "synthetic-test-dsn" {
		t.Fatal("runtime environment was not read")
	}
}
