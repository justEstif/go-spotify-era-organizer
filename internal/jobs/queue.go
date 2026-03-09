// Package jobs provides an in-memory background job queue with progress reporting.
package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Status represents the current state of a job.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// Job represents a background task with progress tracking.
type Job struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Type      string         `json:"type"`
	Status    Status         `json:"status"`
	Progress  int            `json:"progress"`
	Message   string         `json:"message"`
	Error     string         `json:"error,omitempty"`
	Meta      map[string]any `json:"-"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// JobHandler is a function that executes a job's work.
// The progress callback updates the job's progress (0-100) and message.
type JobHandler func(ctx context.Context, job *Job, progress func(pct int, msg string)) error

// Queue manages background jobs in-memory.
type Queue struct {
	mu       sync.RWMutex
	jobs     map[string]*Job
	handlers map[string]JobHandler
	subs     map[string][]chan *Job
}

// New creates a new job queue.
func New() *Queue {
	return &Queue{
		jobs:     make(map[string]*Job),
		handlers: make(map[string]JobHandler),
		subs:     make(map[string][]chan *Job),
	}
}

// Register registers a handler for a job type.
func (q *Queue) Register(jobType string, handler JobHandler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[jobType] = handler
}

// Submit creates and enqueues a job with no metadata.
func (q *Queue) Submit(userID, jobType string) (string, error) {
	return q.SubmitWithMeta(userID, jobType, nil)
}

// SubmitWithMeta creates and enqueues a job with arbitrary metadata.
func (q *Queue) SubmitWithMeta(userID, jobType string, meta map[string]any) (string, error) {
	q.mu.Lock()

	// Check for registered handler
	handler, ok := q.handlers[jobType]
	if !ok {
		q.mu.Unlock()
		return "", fmt.Errorf("no handler registered for job type %q", jobType)
	}

	// Prevent duplicate: reject if user already has a running/pending job of this type
	for _, j := range q.jobs {
		if j.UserID == userID && j.Type == jobType && (j.Status == StatusPending || j.Status == StatusRunning) {
			q.mu.Unlock()
			return j.ID, ErrDuplicateJob
		}
	}

	// Clean up old completed/failed jobs for this user (older than 1 hour)
	cutoff := time.Now().Add(-1 * time.Hour)
	for id, j := range q.jobs {
		if j.UserID == userID && (j.Status == StatusCompleted || j.Status == StatusFailed) && j.UpdatedAt.Before(cutoff) {
			delete(q.jobs, id)
			delete(q.subs, id)
		}
	}

	now := time.Now()
	job := &Job{
		ID:        uuid.New().String(),
		UserID:    userID,
		Type:      jobType,
		Status:    StatusPending,
		Progress:  0,
		Message:   "Queued",
		Meta:      meta,
		CreatedAt: now,
		UpdatedAt: now,
	}

	q.jobs[job.ID] = job
	q.mu.Unlock()

	// Launch goroutine to execute the job
	go q.run(job, handler)

	return job.ID, nil
}

// run executes a job's handler in a goroutine.
func (q *Queue) run(job *Job, handler JobHandler) {
	q.updateStatus(job.ID, StatusRunning, 0, "Starting...")

	ctx := context.Background()

	progress := func(pct int, msg string) {
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		q.updateStatus(job.ID, StatusRunning, pct, msg)
	}

	err := handler(ctx, job, progress)

	if err != nil {
		q.mu.Lock()
		j := q.jobs[job.ID]
		if j != nil {
			j.Status = StatusFailed
			j.Error = err.Error()
			j.UpdatedAt = time.Now()
		}
		q.mu.Unlock()
		q.notify(job.ID)
	} else {
		q.updateStatus(job.ID, StatusCompleted, 100, "Complete!")
	}
}

// updateStatus updates a job's status, progress, and message, then notifies subscribers.
func (q *Queue) updateStatus(jobID string, status Status, progress int, message string) {
	q.mu.Lock()
	j, ok := q.jobs[jobID]
	if ok {
		j.Status = status
		j.Progress = progress
		j.Message = message
		j.UpdatedAt = time.Now()
	}
	q.mu.Unlock()

	if ok {
		q.notify(jobID)
	}
}

// notify sends job state to all subscribers. Non-blocking: drops if channel is full.
func (q *Queue) notify(jobID string) {
	q.mu.RLock()
	j := q.jobs[jobID]
	subs := q.subs[jobID]
	q.mu.RUnlock()

	if j == nil {
		return
	}

	// Send a snapshot copy to avoid races
	snapshot := *j

	for _, ch := range subs {
		select {
		case ch <- &snapshot:
		default:
			// subscriber is slow, drop update
		}
	}
}

// Get returns a job by ID.
func (q *Queue) Get(jobID string) (*Job, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	j, ok := q.jobs[jobID]
	if !ok {
		return nil, false
	}
	snapshot := *j
	return &snapshot, true
}

// GetForUser returns all jobs for a user.
func (q *Queue) GetForUser(userID string) []*Job {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var result []*Job
	for _, j := range q.jobs {
		if j.UserID == userID {
			snapshot := *j
			result = append(result, &snapshot)
		}
	}
	return result
}

// GetActiveForUser returns running/pending jobs for a user.
func (q *Queue) GetActiveForUser(userID string) []*Job {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var result []*Job
	for _, j := range q.jobs {
		if j.UserID == userID && (j.Status == StatusPending || j.Status == StatusRunning) {
			snapshot := *j
			result = append(result, &snapshot)
		}
	}
	return result
}

// Subscribe returns a channel that receives job updates and an unsubscribe function.
func (q *Queue) Subscribe(jobID string) (<-chan *Job, func()) {
	ch := make(chan *Job, 16)

	q.mu.Lock()
	q.subs[jobID] = append(q.subs[jobID], ch)
	q.mu.Unlock()

	unsubscribe := func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		subs := q.subs[jobID]
		for i, s := range subs {
			if s == ch {
				q.subs[jobID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		close(ch)
	}

	return ch, unsubscribe
}

// ErrDuplicateJob is returned when a user already has a running job of the same type.
var ErrDuplicateJob = fmt.Errorf("a job of this type is already running")
