package telemetry

import (
	"sync"
	"time"

	"agentium/internal/model"
)

type Tracker struct {
	mu                   sync.RWMutex
	events               []model.NetworkEvent
	indexByRequestID     map[string]int
	activeRelevant       map[string]struct{}
	maxEvents            int
	lastActivity         time.Time
	lastRelevantActivity time.Time
}

func NewTracker(maxEvents int) *Tracker {
	now := time.Now()
	return &Tracker{
		indexByRequestID:     make(map[string]int, maxEvents),
		activeRelevant:       make(map[string]struct{}),
		maxEvents:            maxEvents,
		lastActivity:         now,
		lastRelevantActivity: now,
	}
}

func (t *Tracker) OnRequest(event model.NetworkEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.lastActivity = event.Timestamp
	if isRelevantResource(event.ResourceType) {
		t.lastRelevantActivity = event.Timestamp
		t.activeRelevant[event.RequestID] = struct{}{}
	}

	if index, ok := t.indexByRequestID[event.RequestID]; ok {
		t.events[index] = mergeEvent(t.events[index], event)
		return
	}

	if len(t.events) == t.maxEvents {
		dropped := t.events[0]
		copy(t.events, t.events[1:])
		t.events[len(t.events)-1] = event
		delete(t.indexByRequestID, dropped.RequestID)
		t.rebuildIndexes()
		return
	}

	t.events = append(t.events, event)
	t.indexByRequestID[event.RequestID] = len(t.events) - 1
}

func (t *Tracker) OnResponse(requestID string, status int, timestamp time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.lastActivity = timestamp
	index, ok := t.indexByRequestID[requestID]
	if !ok {
		return
	}

	event := t.events[index]
	event.Status = status
	event.Stage = "response"
	t.events[index] = event
	if isRelevantResource(event.ResourceType) {
		t.lastRelevantActivity = timestamp
	}
}

func (t *Tracker) OnFinished(requestID string, timestamp time.Time) {
	t.finish(requestID, "finished", "", timestamp)
}

func (t *Tracker) OnFailed(requestID, errorText string, timestamp time.Time) {
	t.finish(requestID, "failed", errorText, timestamp)
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

func (t *Tracker) WaitForIdle(idleFor, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		if t.isIdle(idleFor) {
			return true
		}

		if time.Now().After(deadline) {
			return false
		}

		<-ticker.C
	}
}

func (t *Tracker) LastActivity() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastActivity
}

func (t *Tracker) finish(requestID, stage, errorText string, timestamp time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.lastActivity = timestamp
	index, ok := t.indexByRequestID[requestID]
	if !ok {
		return
	}

	event := t.events[index]
	event.Stage = stage
	event.ErrorText = errorText
	t.events[index] = event

	if isRelevantResource(event.ResourceType) {
		delete(t.activeRelevant, requestID)
		t.lastRelevantActivity = timestamp
	}
}

func (t *Tracker) isIdle(idleFor time.Duration) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.activeRelevant) > 0 {
		return false
	}

	return time.Since(t.lastRelevantActivity) >= idleFor
}

func (t *Tracker) rebuildIndexes() {
	for index, event := range t.events {
		t.indexByRequestID[event.RequestID] = index
	}
}

func mergeEvent(current, incoming model.NetworkEvent) model.NetworkEvent {
	if incoming.Method != "" {
		current.Method = incoming.Method
	}
	if incoming.URL != "" {
		current.URL = incoming.URL
	}
	if incoming.Status != 0 {
		current.Status = incoming.Status
	}
	if incoming.ResourceType != "" {
		current.ResourceType = incoming.ResourceType
	}
	if incoming.Stage != "" {
		current.Stage = incoming.Stage
	}
	if incoming.ErrorText != "" {
		current.ErrorText = incoming.ErrorText
	}
	if !incoming.Timestamp.IsZero() {
		current.Timestamp = incoming.Timestamp
	}
	return current
}

func isRelevantResource(resourceType string) bool {
	return resourceType == "XHR" || resourceType == "Fetch"
}
