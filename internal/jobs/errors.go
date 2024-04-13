package jobs

import "errors"

var (
	ErrInvalidParams    = errors.New("invalid params")
	ErrSchedulerStopped = errors.New("scheduler stopped")
)
