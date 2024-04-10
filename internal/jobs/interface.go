package jobs

import "github.com/htchan/WebHistory/internal/executor"

type Scheduler interface {
	Start()
	Stop() error
	Publisher() executor.JobTrigger
}
