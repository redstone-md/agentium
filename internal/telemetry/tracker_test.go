package telemetry

import (
	"testing"
	"time"

	"agentium/internal/model"
)

func TestTrackerMergesRequestLifecycle(t *testing.T) {
	tracker := NewTracker(10)
	start := time.Now()

	tracker.OnRequest(model.NetworkEvent{
		Timestamp:    start,
		RequestID:    "req-1",
		Method:       "POST",
		URL:          "https://example.com/api",
		ResourceType: "XHR",
		Stage:        "request",
	})
	tracker.OnResponse("req-1", 201, start.Add(10*time.Millisecond))
	tracker.OnFinished("req-1", start.Add(20*time.Millisecond))

	events := tracker.Since(start.Add(-time.Millisecond))
	if len(events) != 1 {
		t.Fatalf("expected 1 merged event, got %d", len(events))
	}

	event := events[0]
	if event.Method != "POST" || event.Status != 201 || event.Stage != "finished" {
		t.Fatalf("unexpected merged event: %+v", event)
	}
}

func TestTrackerWaitForIdleUsesRelevantRequests(t *testing.T) {
	tracker := NewTracker(10)
	now := time.Now()

	tracker.OnRequest(model.NetworkEvent{
		Timestamp:    now,
		RequestID:    "req-1",
		Method:       "GET",
		URL:          "https://example.com/data",
		ResourceType: "XHR",
		Stage:        "request",
	})

	done := make(chan bool, 1)
	go func() {
		done <- tracker.WaitForIdle(100*time.Millisecond, 2*time.Second)
	}()

	time.Sleep(150 * time.Millisecond)
	tracker.OnFinished("req-1", time.Now())

	select {
	case ok := <-done:
		if !ok {
			t.Fatal("expected tracker to become idle")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("wait for idle timed out")
	}
}
