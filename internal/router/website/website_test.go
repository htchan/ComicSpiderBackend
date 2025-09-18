package website

import (
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"go.uber.org/goleak"
)

var connString string

func isNatsConnected() bool {
	nc, err := nats.Connect(connString)
	defer nc.Close()

	return err == nil && nc.IsConnected()
}

func TestMain(m *testing.M) {

	leak := flag.Bool("leak", false, "check for memory leaks")
	flag.Parse()

	sqlcConnString, purge, err := setupContainer()
	if err != nil {
		purge()
		log.Fatalf("fail to setup docker: %v", err)
	}

	connString = sqlcConnString

	// Wait for NATS to be ready
	natsConnected := false
	for range 100 {
		natsConnected = isNatsConnected()
		if natsConnected {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if !natsConnected {
		log.Printf("NATS container failed to connect")
		purge()
		os.Exit(1)
	}

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

	containerName := "webhistory_test_router"
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
			log.Println("purge error", err)
		}
	}

	connString := fmt.Sprintf("nats://%s", resource.GetHostPort("4222/tcp"))

	return connString, purge, nil
}
