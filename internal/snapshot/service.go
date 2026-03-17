package snapshot

import (
	"encoding/json"
	"fmt"

	"agentium/internal/model"
	"agentium/internal/session"
)

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

	runtime.UpdateRefs(snapshot.Elements)
	return snapshot, nil
}
