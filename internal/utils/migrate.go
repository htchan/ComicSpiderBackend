package utils

import (
	"context"
	"fmt"

	"database/sql"

	"github.com/htchan/WebHistory/internal/config"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Migrate(conf *config.DatabaseConfig) error {
	tr := otel.Tracer("process")
	_, span := tr.Start(context.Background(), "migrate")
	defer span.End()

	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		conf.User, conf.Password, conf.Host, conf.Port, conf.Database,
	)

	db, err := sql.Open(conf.Driver, connString)
	if err != nil {
		return fmt.Errorf("migrate fail: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("migrate fail: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file:///migrations", conf.Driver, driver)
	if err != nil {
		return fmt.Errorf("migrate fail: %w", err)
	}

	err = m.Up()
	if err != nil {
		log.Warn().Err(err).Msg("migrate fail")
	}

	defer m.Close()
	return nil
}
