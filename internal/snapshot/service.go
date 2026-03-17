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
	if err := json.Unmarshal([]byte(result.Value.String()), &snapshot); err != nil {
		return model.Snapshot{}, fmt.Errorf("decode snapshot: %w", err)
	}

	runtime.UpdateRefs(snapshot.Elements)
	return snapshot, nil
}
