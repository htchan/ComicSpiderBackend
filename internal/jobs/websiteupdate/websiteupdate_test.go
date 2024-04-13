package websiteupdate

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/vendors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	leak := flag.Bool("leak", false, "check for memory leaks")
	flag.Parse()

	if *leak {
		goleak.VerifyTestMain(m)
	} else {
		os.Exit(m.Run())
	}
}

func TestSetup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		rpo                 repository.Repostory
		services            []vendors.VendorService
		conf                *config.WorkerBinConfig
		sleepInterval       time.Duration
		wantJob             *Job
		wantExecAtBeginning bool
	}{
		{
			name:     "happy flow",
			rpo:      nil,
			services: nil,
			conf: &config.WorkerBinConfig{
				WebsiteUpdateSleepInterval: 5 * time.Second,
				ExecAtBeginning:            true,
			},
			wantJob:             &Job{rpo: nil, sleepInterval: 5 * time.Second},
			wantExecAtBeginning: true,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			scheduler := Setup(test.rpo, test.conf, test.services)
			assert.Equal(t, test.wantJob, scheduler.job)
			assert.Equal(t, test.wantExecAtBeginning, scheduler.execAtBeginning)
		})
	}
}
