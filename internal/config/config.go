package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultServerAddress        = ":8080"
	defaultMongoDatabase        = "africandoers"
	defaultDatabaseTimeout      = 10 * time.Second
	defaultPostgresMaxOpenConns = 20
	defaultPostgresMaxIdleConns = 5
	defaultPostgresConnLifetime = 30 * time.Minute
	defaultCSRFTTL              = 12 * time.Hour
	defaultRequestBodyMaxBytes  = 12 << 20
	defaultMultipartMemoryBytes = 2 << 20
	minimumCSRFSecretLength     = 32
)

type Config struct {
	Environment string

	ServerAddress string

	PostgresDSN             string
	PostgresMaxOpenConns    int
	PostgresMaxIdleConns    int
	PostgresConnMaxLifetime time.Duration

	MongoURI      string
	MongoDatabase string

	DatabaseConnectTimeout time.Duration

	CSRFSecret           string
	CSRFCookieSecure     bool
	CSRFTTL              time.Duration
	RequestBodyMaxBytes  int64
	MultipartMemoryBytes int64
}

// Load reads configuration from process environment variables.
//
// A local .env file is loaded when present. Existing process environment
// variables always take precedence over values in .env.
func Load() (Config, error) {
	if err := godotenv.Load(); err != nil &&
		!errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf(
			"load .env file: %w",
			err,
		)
	}

	environment := strings.ToLower(
		getEnv("APP_ENV", "development"),
	)

	if environment != "development" &&
		environment != "test" &&
		environment != "production" {
		return Config{}, fmt.Errorf(
			"APP_ENV must be development, test, or production",
		)
	}

	postgresDSN := firstNonEmpty(
		os.Getenv("POSTGRES_DSN"),
		os.Getenv("DATABASE_URL"),
	)

	mongoURI := firstNonEmpty(
		os.Getenv("MONGO_URI"),
		os.Getenv("MONGODB_URI"),
	)

	csrfSecret := strings.TrimSpace(
		os.Getenv("CSRF_SECRET"),
	)

	var missing []string

	if strings.TrimSpace(postgresDSN) == "" {
		missing = append(
			missing,
			"POSTGRES_DSN or DATABASE_URL",
		)
	}

	if strings.TrimSpace(mongoURI) == "" {
		missing = append(
			missing,
			"MONGO_URI or MONGODB_URI",
		)
	}

	if csrfSecret == "" {
		missing = append(
			missing,
			"CSRF_SECRET",
		)
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf(
			"missing required environment variables: %s",
			strings.Join(missing, ", "),
		)
	}

	if len(csrfSecret) < minimumCSRFSecretLength {
		return Config{}, fmt.Errorf(
			"CSRF_SECRET must contain at least %d characters",
			minimumCSRFSecretLength,
		)
	}

	cookieSecureDefault := environment == "production"

	cookieSecure, err := getBool(
		"COOKIE_SECURE",
		cookieSecureDefault,
	)
	if err != nil {
		return Config{}, err
	}

	if environment == "production" &&
		!cookieSecure {
		return Config{}, fmt.Errorf(
			"COOKIE_SECURE must be true in production",
		)
	}

	databaseTimeout, err := getDuration(
		"DATABASE_CONNECT_TIMEOUT",
		defaultDatabaseTimeout,
	)
	if err != nil {
		return Config{}, err
	}

	csrfTTL, err := getDuration(
		"CSRF_TTL",
		defaultCSRFTTL,
	)
	if err != nil {
		return Config{}, err
	}

	maxOpenConnections, err := getInt(
		"POSTGRES_MAX_OPEN_CONNS",
		defaultPostgresMaxOpenConns,
	)
	if err != nil {
		return Config{}, err
	}

	maxIdleConnections, err := getInt(
		"POSTGRES_MAX_IDLE_CONNS",
		defaultPostgresMaxIdleConns,
	)
	if err != nil {
		return Config{}, err
	}

	if maxIdleConnections > maxOpenConnections {
		return Config{}, fmt.Errorf(
			"POSTGRES_MAX_IDLE_CONNS cannot exceed POSTGRES_MAX_OPEN_CONNS",
		)
	}

	connectionLifetime, err := getDuration(
		"POSTGRES_CONN_MAX_LIFETIME",
		defaultPostgresConnLifetime,
	)
	if err != nil {
		return Config{}, err
	}

	requestBodyMaxBytes, err := getInt64(
		"REQUEST_BODY_MAX_BYTES",
		defaultRequestBodyMaxBytes,
	)
	if err != nil {
		return Config{}, err
	}

	multipartMemoryBytes, err := getInt64(
		"MULTIPART_MEMORY_BYTES",
		defaultMultipartMemoryBytes,
	)
	if err != nil {
		return Config{}, err
	}

	if multipartMemoryBytes >
		requestBodyMaxBytes {
		return Config{}, fmt.Errorf(
			"MULTIPART_MEMORY_BYTES cannot exceed REQUEST_BODY_MAX_BYTES",
		)
	}

	return Config{
		Environment: environment,

		ServerAddress: getEnv(
			"SERVER_ADDR",
			defaultServerAddress,
		),

		PostgresDSN:             postgresDSN,
		PostgresMaxOpenConns:    maxOpenConnections,
		PostgresMaxIdleConns:    maxIdleConnections,
		PostgresConnMaxLifetime: connectionLifetime,

		MongoURI: mongoURI,
		MongoDatabase: getEnv(
			"MONGO_DATABASE",
			defaultMongoDatabase,
		),

		DatabaseConnectTimeout: databaseTimeout,

		CSRFSecret:           csrfSecret,
		CSRFCookieSecure:     cookieSecure,
		CSRFTTL:              csrfTTL,
		RequestBodyMaxBytes:  requestBodyMaxBytes,
		MultipartMemoryBytes: multipartMemoryBytes,
	}, nil
}

func getEnv(
	name string,
	defaultValue string,
) string {
	value := strings.TrimSpace(
		os.Getenv(name),
	)

	if value == "" {
		return defaultValue
	}

	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}

func getBool(
	name string,
	defaultValue bool,
) (bool, error) {
	rawValue := strings.TrimSpace(
		os.Getenv(name),
	)
	if rawValue == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseBool(
		rawValue,
	)
	if err != nil {
		return false, fmt.Errorf(
			"%s must be true or false",
			name,
		)
	}

	return value, nil
}

func getInt(
	name string,
	defaultValue int,
) (int, error) {
	rawValue := strings.TrimSpace(
		os.Getenv(name),
	)
	if rawValue == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(rawValue)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf(
			"%s must be a positive integer",
			name,
		)
	}

	return value, nil
}

func getInt64(
	name string,
	defaultValue int64,
) (int64, error) {
	rawValue := strings.TrimSpace(
		os.Getenv(name),
	)
	if rawValue == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseInt(
		rawValue,
		10,
		64,
	)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf(
			"%s must be a positive integer",
			name,
		)
	}

	return value, nil
}

func getDuration(
	name string,
	defaultValue time.Duration,
) (time.Duration, error) {
	rawValue := strings.TrimSpace(
		os.Getenv(name),
	)
	if rawValue == "" {
		return defaultValue, nil
	}

	value, err := time.ParseDuration(
		rawValue,
	)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf(
			"%s must be a positive Go duration such as 10s, 30m, or 24h",
			name,
		)
	}

	return value, nil
}
