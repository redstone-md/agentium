package telemetry

import (
	"sync"
	"time"

	"agentium/internal/model"
)

type Tracker struct {
	mu           sync.RWMutex
	events       []model.NetworkEvent
	maxEvents    int
	lastActivity time.Time
}

func NewTracker(maxEvents int) *Tracker {
	return &Tracker{
		maxEvents:    maxEvents,
		lastActivity: time.Now(),
	}
}

func (t *Tracker) Push(event model.NetworkEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.events) == t.maxEvents {
		copy(t.events, t.events[1:])
		t.events[len(t.events)-1] = event
	} else {
		t.events = append(t.events, event)
	}

	t.lastActivity = event.Timestamp
}

func (t *Tracker) Since(start time.Time) []model.NetworkEvent {
	t.mu.RLock()
	defer t.mu.RUnlock()

	filtered := make([]model.NetworkEvent, 0, len(t.events))
	for _, event := range t.events {
		if !event.Timestamp.Before(start) {
			filtered = append(filtered, event)
		}
	}

	return filtered
}

func (t *Tracker) LastActivity() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastActivity
}
