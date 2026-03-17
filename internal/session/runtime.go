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

	actionMu sync.Mutex
	stateMu  sync.RWMutex
	refs     map[int]cachedRef
	mouse    point
	close    func() error
}

type cachedRef struct {
	meta    model.SnapshotElement
	element *rod.Element
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
		refs:           make(map[int]cachedRef),
		mouse:          point{X: 8, Y: 8},
		close:          closeFn,
	}
}

func (r *Runtime) UpdateRefs(elements []model.SnapshotElement, elementMap map[int]*rod.Element) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()

	r.refs = make(map[int]cachedRef, len(elements))
	for _, element := range elements {
		r.refs[element.RefID] = cachedRef{
			meta:    element,
			element: elementMap[element.RefID],
		}
	}
}

func (r *Runtime) Ref(refID int) (model.SnapshotElement, bool) {
	r.stateMu.RLock()
	defer r.stateMu.RUnlock()

	ref, ok := r.refs[refID]
	return ref.meta, ok
}

func (r *Runtime) Element(refID int) (*rod.Element, bool) {
	r.stateMu.RLock()
	defer r.stateMu.RUnlock()

	ref, ok := r.refs[refID]
	if !ok || ref.element == nil {
		return nil, false
	}

	return ref.element, true
}

func (r *Runtime) MousePosition() (float64, float64) {
	r.stateMu.RLock()
	defer r.stateMu.RUnlock()
	return r.mouse.X, r.mouse.Y
}

func (r *Runtime) SetMousePosition(x, y float64) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	r.mouse = point{X: x, Y: y}
}

func (r *Runtime) WithLock(fn func() error) error {
	r.actionMu.Lock()
	defer r.actionMu.Unlock()
	return fn()
}

func (r *Runtime) Close() error {
	if r.close == nil {
		return nil
	}
	return r.close()
}
