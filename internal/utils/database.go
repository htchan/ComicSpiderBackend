package utils

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	_ "github.com/lib/pq"
)

// open database for psql
func openPostgresDatabase(conf *config.DatabaseConfig) (*sql.DB, error) {
	conn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		conf.Host, conf.Port, conf.User, conf.Password, conf.Database,
	)

	database, err := sql.Open(conf.Driver, conn)
	if err != nil {
		return database, err
	}
	database.SetMaxIdleConns(5)
	database.SetMaxOpenConns(10)
	database.SetConnMaxIdleTime(5 * time.Second)
	database.SetConnMaxLifetime(5 * time.Second)
	return database, err
}

func OpenDatabase(conf *config.DatabaseConfig) (*sql.DB, error) {
	return openPostgresDatabase(conf)
}
