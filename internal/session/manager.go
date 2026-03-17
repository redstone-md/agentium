package session

import (
	"errors"
	"sync"
)

var ErrSessionNotFound = errors.New("session not found")

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Runtime
}

func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Runtime),
	}
}

func (m *Manager) Add(runtime *Runtime) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[runtime.ID] = runtime
}

func (m *Manager) Get(id string) (*Runtime, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	runtime, ok := m.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}

	return runtime, nil
}

func (m *Manager) Delete(id string) (*Runtime, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime, ok := m.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}

	delete(m.sessions, id)
	return runtime, nil
}
