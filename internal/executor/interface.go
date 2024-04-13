package executor

import (
	"context"
)

type Executor interface {
	Start()
	Register(...JobTrigger)
	Stop() error
}

type Job interface {
	Execute(context.Context, interface{}) error
}

type JobExec struct {
	Job    Job
	Params interface{}
}

type JobTrigger = chan *JobExec
