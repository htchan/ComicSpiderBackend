package utils

import (
	"context"
	"fmt"

	"database/sql"

	"github.com/htchan/WebHistory/internal/config"
	_ "github.com/lib/pq"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Migrate(conf *config.DatabaseConfig) error {
	tr := otel.Tracer("github.com/htchan/WebHistory/migrate")
	_, span := tr.Start(context.Background(), "migrate", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		conf.User, conf.Password, conf.Host, conf.Port, conf.Database,
	)

	db, err := sql.Open(conf.Driver, connString)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return fmt.Errorf("migrate fail: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return fmt.Errorf("migrate fail: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file:///migrations", conf.Driver, driver)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return fmt.Errorf("migrate fail: %w", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return fmt.Errorf("migrate fail: %w", err)
	}

	defer m.Close()
	return nil
}
