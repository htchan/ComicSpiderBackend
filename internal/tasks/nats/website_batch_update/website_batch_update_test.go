package websitebatchupdate

import (
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"go.uber.org/goleak"
)

var connString string

func TestMain(m *testing.M) {

	leak := flag.Bool("leak", false, "check for memory leaks")
	flag.Parse()

	sqlcConnString, purge, err := setupContainer()
	if err != nil {
		purge()
		log.Fatalf("fail to setup docker: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	connString = sqlcConnString

	if *leak {
		goleak.VerifyTestMain(m)
		purge()
	} else {
		code := m.Run()

		purge()
		os.Exit(code)
	}
}

func setupContainer() (string, func(), error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return "", func() {}, err
	}

	containerName := "webhistory_test_website_batch_update"
	pool.RemoveContainerByName(containerName)

	resource, err := pool.RunWithOptions(
		&dockertest.RunOptions{
			Repository: "nats",
			Tag:        "latest",
			Name:       containerName,
			Cmd:        []string{"-js"},
		},
		func(hc *docker.HostConfig) {
			hc.AutoRemove = true
			hc.RestartPolicy = docker.RestartPolicy{
				Name: "no",
			}
		},
	)
	if err != nil {
		return "", func() {}, err
	}

	purge := func() {
		err := pool.Purge(resource)
		if err != nil {
			fmt.Println("purge error", err)
		}
	}

	connString := fmt.Sprintf("nats://%s", resource.GetHostPort("4222/tcp"))

	return connString, purge, nil
}
