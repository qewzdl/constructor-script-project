package adapters

import (
	"context"
	"time"

	"constructor-script-backend/internal/background"
	"constructor-script-backend/pkg/pluginsdk"
)

// SchedulerAdapter adapts background scheduler to pluginsdk.Scheduler interface
type SchedulerAdapter struct {
	scheduler *background.Scheduler
}

func NewSchedulerAdapter(scheduler *background.Scheduler) pluginsdk.Scheduler {
	return &SchedulerAdapter{scheduler: scheduler}
}

func (a *SchedulerAdapter) Schedule(name string, interval time.Duration, fn func() error) error {
	job := background.Job{
		Name: name,
		Run: func(ctx context.Context) error {
			return fn()
		},
		Delay: 0,
	}
	return a.scheduler.Schedule(job)
}

func (a *SchedulerAdapter) ScheduleOnce(name string, delay time.Duration, fn func() error) error {
	job := background.Job{
		Name: name,
		Run: func(ctx context.Context) error {
			return fn()
		},
		Delay: delay,
	}
	return a.scheduler.Schedule(job)
}

func (a *SchedulerAdapter) Cancel(name string) error {
	// Cancel is not directly supported by background.Scheduler
	// Jobs complete on their own or during shutdown
	return nil
}
