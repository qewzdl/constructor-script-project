package background

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"constructor-script-backend/pkg/logger"
)

type SchedulerConfig struct {
	WorkerCount int
	QueueSize   int
}

type RetryPolicy struct {
	MaxRetries int
	Backoff    time.Duration
}

type Job struct {
	Name        string
	Run         func(ctx context.Context) error
	Delay       time.Duration
	Timeout     time.Duration
	RetryPolicy RetryPolicy
}

var (
	ErrSchedulerNotStarted   = errors.New("scheduler not started")
	ErrJobAlreadyScheduled   = errors.New("job already scheduled")
	errSchedulerShuttingDown = errors.New("scheduler is shutting down")
)

type Scheduler struct {
	config SchedulerConfig

	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	started bool

	queue chan scheduledJob

	workerWG sync.WaitGroup
	jobWG    sync.WaitGroup

	activeJobs map[string]struct{}
}

type scheduledJob struct {
	job     Job
	attempt int
	unique  bool
}

var (
	metricsOnce        sync.Once
	jobRunsTotal       *prometheus.CounterVec
	jobDurationSeconds *prometheus.HistogramVec
	jobLastSuccess     *prometheus.GaugeVec
)

func initMetrics() {
	metricsOnce.Do(func() {
		jobRunsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "constructor_script",
			Subsystem: "background",
			Name:      "job_runs_total",
			Help:      "Total background job executions",
		}, []string{"job", "status"})

		jobDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "constructor_script",
			Subsystem: "background",
			Name:      "job_duration_seconds",
			Help:      "Duration of background job executions",
			Buckets:   prometheus.DefBuckets,
		}, []string{"job"})

		jobLastSuccess = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "constructor_script",
			Subsystem: "background",
			Name:      "job_last_success_timestamp",
			Help:      "Unix timestamp of the last successful background job execution",
		}, []string{"job"})
	})
}

func NewScheduler(cfg SchedulerConfig) *Scheduler {
	initMetrics()

	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 2
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 32
	}

	return &Scheduler{
		config:     cfg,
		queue:      make(chan scheduledJob, cfg.QueueSize),
		activeJobs: make(map[string]struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.started = true

	for i := 0; i < s.config.WorkerCount; i++ {
		s.workerWG.Add(1)
		go s.worker()
	}
}

func (s *Scheduler) worker() {
	defer s.workerWG.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case job := <-s.queue:
			s.execute(job)
		}
	}
}

func (s *Scheduler) execute(job scheduledJob) {
	if job.job.Delay > 0 {
		timer := time.NewTimer(job.job.Delay)
		select {
		case <-timer.C:
		case <-s.ctx.Done():
			timer.Stop()
			s.finishJob(job, context.Canceled)
			return
		}
	}

	s.jobWG.Add(1)
	defer s.jobWG.Done()

	if err := s.runJob(job); err != nil {
		if s.shouldRetry(job, err) {
			retry := job
			retry.attempt++
			if retry.job.RetryPolicy.Backoff > 0 {
				retry.job.Delay = retry.job.RetryPolicy.Backoff
			} else {
				retry.job.Delay = 0
			}

			if !s.enqueue(retry) {
				s.finishJob(job, err)
			}
			return
		}

		s.finishJob(job, err)
		return
	}

	s.finishJob(job, nil)
}

func (s *Scheduler) runJob(job scheduledJob) error {
	start := time.Now()
	status := "success"
	var runErr error

	ctx := s.ctx
	if job.job.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, job.job.Timeout)
		defer cancel()
	}

	defer func() {
		duration := time.Since(start)
		jobDurationSeconds.WithLabelValues(job.job.Name).Observe(duration.Seconds())
		jobRunsTotal.WithLabelValues(job.job.Name, status).Inc()
		if status == "success" {
			jobLastSuccess.WithLabelValues(job.job.Name).Set(float64(time.Now().Unix()))
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			runErr = fmt.Errorf("panic: %v", r)
			status = "failure"
			logger.Error(runErr, "Background job panicked", map[string]interface{}{"job": job.job.Name, "attempt": job.attempt})
		}
	}()

	select {
	case <-ctx.Done():
		status = "canceled"
		return ctx.Err()
	default:
	}

	runErr = job.job.Run(ctx)
	if runErr != nil {
		if errors.Is(runErr, context.Canceled) {
			status = "canceled"
		} else {
			status = "failure"
		}
		logger.Error(runErr, "Background job failed", map[string]interface{}{"job": job.job.Name, "attempt": job.attempt})
		return runErr
	}

	return nil
}

func (s *Scheduler) shouldRetry(job scheduledJob, err error) bool {
	if job.job.RetryPolicy.MaxRetries <= 0 {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	return job.attempt <= job.job.RetryPolicy.MaxRetries
}

func (s *Scheduler) enqueue(job scheduledJob) bool {
	for {
		select {
		case <-s.ctx.Done():
			return false
		case s.queue <- job:
			return true
		}
	}
}

func (s *Scheduler) finishJob(job scheduledJob, runErr error) {
	if job.unique {
		s.mu.Lock()
		delete(s.activeJobs, job.job.Name)
		s.mu.Unlock()
	}

	if runErr == nil {
		logger.Info("Background job completed", map[string]interface{}{"job": job.job.Name, "attempt": job.attempt})
		return
	}

	if errors.Is(runErr, context.Canceled) {
		logger.Warn("Background job canceled", map[string]interface{}{"job": job.job.Name, "attempt": job.attempt})
		return
	}

	logger.Error(runErr, "Background job finished with error", map[string]interface{}{"job": job.job.Name, "attempt": job.attempt})
}

func (s *Scheduler) Schedule(job Job) error {
	return s.schedule(job, false)
}

func (s *Scheduler) ScheduleUnique(job Job) error {
	return s.schedule(job, true)
}

func (s *Scheduler) schedule(job Job, unique bool) error {
	if job.Name == "" {
		return errors.New("job name is required")
	}
	if job.Run == nil {
		return errors.New("job runner is required")
	}

	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return ErrSchedulerNotStarted
	}
	if unique {
		if _, exists := s.activeJobs[job.Name]; exists {
			s.mu.Unlock()
			return ErrJobAlreadyScheduled
		}
		s.activeJobs[job.Name] = struct{}{}
	}
	s.mu.Unlock()

	scheduled := scheduledJob{job: job, attempt: 1, unique: unique}
	if !s.enqueue(scheduled) {
		if unique {
			s.mu.Lock()
			delete(s.activeJobs, job.Name)
			s.mu.Unlock()
		}
		return errSchedulerShuttingDown
	}

	return nil
}

func (s *Scheduler) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	cancel := s.cancel
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	done := make(chan struct{})
	go func() {
		s.workerWG.Wait()
		s.jobWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Scheduler) ActiveJobCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.activeJobs)
}
