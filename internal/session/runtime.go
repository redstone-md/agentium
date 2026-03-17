package session

import (
	"sync"

	"agentium/internal/model"
	"agentium/internal/telemetry"
	"github.com/go-rod/rod"
)

type Runtime struct {
	ID             string
	Options        model.SessionOptions
	RootBrowser    *rod.Browser
	ContextBrowser *rod.Browser
	Page           *rod.Page
	Tracker        *telemetry.Tracker

	mu    sync.RWMutex
	refs  map[int]model.SnapshotElement
	mouse point
	close func() error
}

type point struct {
	X float64
	Y float64
}

func NewRuntime(
	id string,
	options model.SessionOptions,
	rootBrowser *rod.Browser,
	contextBrowser *rod.Browser,
	page *rod.Page,
	tracker *telemetry.Tracker,
	closeFn func() error,
) *Runtime {
	return &Runtime{
		ID:             id,
		Options:        options,
		RootBrowser:    rootBrowser,
		ContextBrowser: contextBrowser,
		Page:           page,
		Tracker:        tracker,
		refs:           make(map[int]model.SnapshotElement),
		mouse:          point{X: 8, Y: 8},
		close:          closeFn,
	}
}

func (r *Runtime) UpdateRefs(elements []model.SnapshotElement) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refs = make(map[int]model.SnapshotElement, len(elements))
	for _, element := range elements {
		r.refs[element.RefID] = element
	}
}

func (r *Runtime) Ref(refID int) (model.SnapshotElement, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	element, ok := r.refs[refID]
	return element, ok
}

func (r *Runtime) MousePosition() (float64, float64) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.mouse.X, r.mouse.Y
}

func (r *Runtime) SetMousePosition(x, y float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mouse = point{X: x, Y: y}
}

func (r *Runtime) WithLock(fn func() error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return fn()
}

func (r *Runtime) Close() error {
	if r.close == nil {
		return nil
	}
	return r.close()
}
