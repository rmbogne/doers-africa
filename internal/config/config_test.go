package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadReadsRequiredEnvironment(
	t *testing.T,
) {
	t.Setenv("APP_ENV", "development")
	t.Setenv(
		"POSTGRES_DSN",
		"host=localhost user=test password=test dbname=test sslmode=disable",
	)
	t.Setenv(
		"MONGO_URI",
		"mongodb://localhost:27017",
	)
	t.Setenv(
		"CSRF_SECRET",
		strings.Repeat("a", 32),
	)
	t.Setenv("COOKIE_SECURE", "false")

	loadedConfig, err := Load()
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}

	if loadedConfig.ServerAddress != ":8080" {
		t.Fatalf(
			"unexpected server address: %s",
			loadedConfig.ServerAddress,
		)
	}

	if loadedConfig.MongoDatabase !=
		"africandoers" {
		t.Fatalf(
			"unexpected Mongo database: %s",
			loadedConfig.MongoDatabase,
		)
	}

	if loadedConfig.DatabaseConnectTimeout !=
		10*time.Second {
		t.Fatalf(
			"unexpected connection timeout: %s",
			loadedConfig.DatabaseConnectTimeout,
		)
	}
}

func TestLoadRejectsShortCSRFSecret(
	t *testing.T,
) {
	t.Setenv(
		"POSTGRES_DSN",
		"host=localhost dbname=test",
	)
	t.Setenv(
		"MONGO_URI",
		"mongodb://localhost:27017",
	)
	t.Setenv(
		"CSRF_SECRET",
		"too-short",
	)

	_, err := Load()
	if err == nil {
		t.Fatal(
			"expected short CSRF secret to fail",
		)
	}
}

func TestLoadRequiresSecureCookiesInProduction(
	t *testing.T,
) {
	t.Setenv("APP_ENV", "production")
	t.Setenv(
		"POSTGRES_DSN",
		"host=localhost dbname=test",
	)
	t.Setenv(
		"MONGO_URI",
		"mongodb://localhost:27017",
	)
	t.Setenv(
		"CSRF_SECRET",
		strings.Repeat("b", 32),
	)
	t.Setenv("COOKIE_SECURE", "false")

	_, err := Load()
	if err == nil {
		t.Fatal(
			"expected insecure production cookies to fail",
		)
	}
}

func TestLoadRequestSizeDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("POSTGRES_DSN", "host=localhost dbname=test")
	t.Setenv("MONGO_URI", "mongodb://localhost:27017")
	t.Setenv("CSRF_SECRET", strings.Repeat("c", 32))
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("STANDARD_REQUEST_BODY_MAX_BYTES", "")
	t.Setenv("UPLOAD_REQUEST_BODY_MAX_BYTES", "")
	t.Setenv("REQUEST_BODY_MAX_BYTES", "")

	loadedConfig, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loadedConfig.StandardRequestBodyMaxBytes != 64<<10 {
		t.Fatalf("unexpected standard body limit: %d", loadedConfig.StandardRequestBodyMaxBytes)
	}
	if loadedConfig.UploadRequestBodyMaxBytes != 3<<20 {
		t.Fatalf("unexpected upload body limit: %d", loadedConfig.UploadRequestBodyMaxBytes)
	}
}
