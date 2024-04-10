package sqlc

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"go.uber.org/goleak"
)

var connString string

const (
	user     = "test"
	password = "test"
	dbname   = "test"
)

func TestMain(m *testing.M) {
	leak := flag.Bool("leak", false, "check for memory leaks")
	flag.Parse()

	sqlcConnString, purge, err := setupContainer()
	if err != nil {
		purge()
		log.Fatalf("fail to setup docker: %v", err)
	}

	connString = sqlcConnString

	goleak.VerifyTestMain(m)

	if *leak {
		goleak.VerifyTestMain(m)
	} else {
		code := m.Run()

		purge()
		os.Exit(code)
	}
}

func setupMigrate(connString string) error {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return fmt.Errorf("migrate fail: %w", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance("file://../../../database/migrations", "postgres", driver)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil {
		return err
	}

	defer m.Close()

	return nil
}

func setupContainer() (string, func(), error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return "", func() {}, fmt.Errorf("init docker fail: %w", err)
	}

	containerName := "webhistory_test_sqlc_db"
	pool.RemoveContainerByName(containerName)

	resource, err := pool.RunWithOptions(
		&dockertest.RunOptions{
			Repository: "postgres",
			Tag:        "latest",
			Name:       containerName,
			Env: []string{
				fmt.Sprintf("POSTGRES_USER=%s", user),
				fmt.Sprintf("POSTGRES_PASSWORD=%s", password),
				fmt.Sprintf("POSTGRES_DB=%s", dbname),
			},
		},
		func(hc *docker.HostConfig) {
			hc.AutoRemove = true
			hc.RestartPolicy = docker.RestartPolicy{
				Name: "no",
			}
		},
	)
	if err != nil {
		return "", func() {}, fmt.Errorf("create resource fail: %w", err)
	}

	purge := func() {
		err := resource.Close()
		if err != nil {
			fmt.Println("purge error", err)
		}
	}

	connString := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", user, password, resource.GetHostPort("5432/tcp"), dbname)
	time.Sleep(3 * time.Second)

	err = setupMigrate(connString)
	if err != nil {
		return "", purge, fmt.Errorf("migrate fail: %w", err)
	}

	return connString, purge, nil
}

func populateData(db *sql.DB, uuid, title string) error {
	_, err := db.Exec("insert into websites (uuid, url, title, content, update_time) values ($1, $2, $3, 'content', $4)", uuid, "http://example.com/"+title, title, time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC))
	if err != nil {
		return err
	}
	_, err = db.Exec("insert into user_websites (website_uuid, user_uuid, group_name, access_time) values ($1, 'def', $2, $3)", uuid, title, time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC))
	return err
}
