package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mbogne/african-doers/internal/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InitStore initializes PostgreSQL and MongoDB from validated environment
// configuration. It never contains or logs credential values.
func InitStore(
	applicationConfig config.Config,
) error {
	connectionContext, cancel :=
		context.WithTimeout(
			context.Background(),
			applicationConfig.
				DatabaseConnectTimeout,
		)
	defer cancel()

	postgresDatabase, err := sql.Open(
		"postgres",
		applicationConfig.PostgresDSN,
	)
	if err != nil {
		return fmt.Errorf(
			"initialize PostgreSQL client: %w",
			err,
		)
	}

	postgresDatabase.SetMaxOpenConns(
		applicationConfig.
			PostgresMaxOpenConns,
	)
	postgresDatabase.SetMaxIdleConns(
		applicationConfig.
			PostgresMaxIdleConns,
	)
	postgresDatabase.SetConnMaxLifetime(
		applicationConfig.
			PostgresConnMaxLifetime,
	)

	if err := postgresDatabase.PingContext(
		connectionContext,
	); err != nil {
		_ = postgresDatabase.Close()

		return fmt.Errorf(
			"connect to PostgreSQL: %w",
			err,
		)
	}

	mongoClient, err := mongo.Connect(
		connectionContext,
		options.Client().ApplyURI(
			applicationConfig.MongoURI,
		),
	)
	if err != nil {
		_ = postgresDatabase.Close()

		return fmt.Errorf(
			"initialize MongoDB client: %w",
			err,
		)
	}

	if err := mongoClient.Ping(
		connectionContext,
		nil,
	); err != nil {
		_ = mongoClient.Disconnect(
			context.Background(),
		)
		_ = postgresDatabase.Close()

		return fmt.Errorf(
			"connect to MongoDB: %w",
			err,
		)
	}

	DB = &Database{
		PG: postgresDatabase,
		Mongo: mongoClient.Database(
			applicationConfig.MongoDatabase,
		),
	}

	setupPGSchema()

	return nil
}

// Close releases database connections during graceful shutdown.
func Close(ctx context.Context) error {
	if DB == nil {
		return nil
	}

	var closeErrors []error

	if DB.PG != nil {
		if err := DB.PG.Close(); err != nil {
			closeErrors = append(
				closeErrors,
				fmt.Errorf(
					"close PostgreSQL: %w",
					err,
				),
			)
		}
	}

	if DB.Mongo != nil {
		if err := DB.Mongo.Client().
			Disconnect(ctx); err != nil {
			closeErrors = append(
				closeErrors,
				fmt.Errorf(
					"close MongoDB: %w",
					err,
				),
			)
		}
	}

	return errors.Join(closeErrors...)
}
