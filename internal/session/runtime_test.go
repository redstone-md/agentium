package session

import (
	"testing"
	"time"

	"agentium/internal/model"
)

func TestWithLockAllowsReadingCachedState(t *testing.T) {
	runtime := NewRuntime("id", model.SessionOptions{}, nil, nil, nil, nil, nil)
	runtime.UpdateRefs([]model.SnapshotElement{{RefID: 1, Text: "button"}}, nil)

	done := make(chan struct{})
	go func() {
		err := runtime.WithLock(func() error {
			if _, ok := runtime.Ref(1); !ok {
				t.Error("expected ref to be readable while action lock is held")
			}

			runtime.SetMousePosition(10, 20)
			x, y := runtime.MousePosition()
			if x != 10 || y != 20 {
				t.Errorf("unexpected mouse position: %f,%f", x, y)
			}

			return nil
		})
		if err != nil {
			t.Errorf("unexpected lock error: %v", err)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expected action lock path to complete without deadlock")
	}
}
