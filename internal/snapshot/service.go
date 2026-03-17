package snapshot

import (
	"encoding/json"
	"fmt"
	"strconv"

	"agentium/internal/model"
	"agentium/internal/session"
	"github.com/go-rod/rod"
)

const maxSnapshotBytes = 20 * 1024

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Capture(runtime *session.Runtime) (model.Snapshot, error) {
	result, err := runtime.Page.Eval(DistillScript)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("capture snapshot: %w", err)
	}

	var snapshot model.Snapshot
	if err := result.Value.Unmarshal(&snapshot); err != nil {
		payload, _ := json.Marshal(result.Value.Raw())
		return model.Snapshot{}, fmt.Errorf("decode snapshot: %w: %s", err, string(payload))
	}

	if snapshot.Elements == nil {
		snapshot.Elements = []model.SnapshotElement{}
	}

	if snapshot.Viewport.Width == 0 || snapshot.Viewport.Height == 0 {
		return model.Snapshot{}, fmt.Errorf("decode snapshot: viewport was empty")
	}

	snapshot, err = limitSnapshotSize(snapshot, maxSnapshotBytes)
	if err != nil {
		return model.Snapshot{}, err
	}

	elementMap, err := loadElementMap(runtime.Page, snapshot.Elements)
	if err != nil {
		return model.Snapshot{}, err
	}

	runtime.UpdateRefs(snapshot.Elements, elementMap)
	return snapshot, nil
}

func loadElementMap(page *rod.Page, elements []model.SnapshotElement) (map[int]*rod.Element, error) {
	allowed := make(map[int]struct{}, len(elements))
	for _, element := range elements {
		allowed[element.RefID] = struct{}{}
	}

	refs := make(map[int]*rod.Element, len(elements))
	nodes, err := page.Elements("[agentium-id]")
	if err != nil {
		return nil, fmt.Errorf("load snapshot elements: %w", err)
	}

	for _, node := range nodes {
		value, attrErr := node.Attribute("agentium-id")
		if attrErr != nil || value == nil {
			continue
		}

		refID, parseErr := strconv.Atoi(*value)
		if parseErr != nil {
			continue
		}

		if _, ok := allowed[refID]; ok {
			refs[refID] = node
		}
	}

	return refs, nil
}

func limitSnapshotSize(snapshot model.Snapshot, limit int) (model.Snapshot, error) {
	trimmed := snapshot
	for {
		payload, err := json.Marshal(trimmed)
		if err != nil {
			return model.Snapshot{}, fmt.Errorf("marshal snapshot: %w", err)
		}

		if len(payload) <= limit {
			return trimmed, nil
		}

		if len(trimmed.Elements) == 0 {
			return model.Snapshot{}, fmt.Errorf("snapshot exceeds %d bytes even without elements", limit)
		}

		trimmed.Elements = trimmed.Elements[:len(trimmed.Elements)-1]
	}
}
