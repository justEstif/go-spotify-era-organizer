package jobs

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSubmitAndComplete(t *testing.T) {
	q := New()
	q.Register("test", func(ctx context.Context, job *Job, progress func(int, string)) error {
		progress(50, "halfway")
		progress(100, "done")
		return nil
	})

	jobID, err := q.Submit("user1", "test")
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// Wait for completion
	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("job did not complete in time")
		default:
			j, ok := q.Get(jobID)
			if !ok {
				t.Fatal("job not found")
			}
			if j.Status == StatusCompleted {
				if j.Progress != 100 {
					t.Errorf("expected progress 100, got %d", j.Progress)
				}
				return
			}
			if j.Status == StatusFailed {
				t.Fatalf("job failed: %s", j.Error)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestSubmitFailed(t *testing.T) {
	q := New()
	q.Register("fail", func(ctx context.Context, job *Job, progress func(int, string)) error {
		return errors.New("something broke")
	})

	jobID, err := q.Submit("user1", "fail")
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("job did not complete in time")
		default:
			j, _ := q.Get(jobID)
			if j.Status == StatusFailed {
				if j.Error != "something broke" {
					t.Errorf("expected error 'something broke', got %q", j.Error)
				}
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestDuplicateJobPrevention(t *testing.T) {
	q := New()
	started := make(chan struct{}, 2)
	done := make(chan struct{})
	q.Register("slow", func(ctx context.Context, job *Job, progress func(int, string)) error {
		started <- struct{}{}
		<-done
		return nil
	})

	_, err := q.Submit("user1", "slow")
	if err != nil {
		t.Fatalf("first submit failed: %v", err)
	}

	<-started // wait for job to start running

	_, err = q.Submit("user1", "slow")
	if !errors.Is(err, ErrDuplicateJob) {
		t.Errorf("expected ErrDuplicateJob, got %v", err)
	}

	// Different user should be allowed
	_, err = q.Submit("user2", "slow")
	if err != nil {
		t.Errorf("different user submit should succeed: %v", err)
	}

	close(done)
}

func TestSubscribe(t *testing.T) {
	q := New()
	q.Register("sub", func(ctx context.Context, job *Job, progress func(int, string)) error {
		progress(25, "quarter")
		progress(50, "half")
		progress(100, "done")
		return nil
	})

	jobID, _ := q.Submit("user1", "sub")
	ch, unsub := q.Subscribe(jobID)
	defer unsub()

	var lastStatus Status
	deadline := time.After(2 * time.Second)
	for lastStatus != StatusCompleted {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for completion events")
		case update := <-ch:
			lastStatus = update.Status
		}
	}
}

func TestUnregisteredHandler(t *testing.T) {
	q := New()
	_, err := q.Submit("user1", "nonexistent")
	if err == nil {
		t.Error("expected error for unregistered handler")
	}
}
