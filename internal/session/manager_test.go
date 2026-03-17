package session

import (
	"testing"

	"agentium/internal/model"
)

func TestManagerDrainClearsSessions(t *testing.T) {
	manager := NewManager()
	first := NewRuntime("a", model.SessionOptions{}, nil, nil, nil, nil, nil)
	second := NewRuntime("b", model.SessionOptions{}, nil, nil, nil, nil, nil)

	manager.Add(first)
	manager.Add(second)

	runtimes := manager.Drain()
	if len(runtimes) != 2 {
		t.Fatalf("expected 2 runtimes, got %d", len(runtimes))
	}

	if _, err := manager.Get("a"); err == nil {
		t.Fatal("expected drained manager to be empty")
	}
}
