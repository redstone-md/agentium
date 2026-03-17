package app

import (
	"context"
	"fmt"
	"time"

	"agentium/internal/action"
	"agentium/internal/browser"
	"agentium/internal/config"
	"agentium/internal/model"
	"agentium/internal/session"
	"agentium/internal/snapshot"
	"agentium/internal/validation"
)

type Service struct {
	manager  *session.Manager
	factory  *browser.Factory
	snapshot *snapshot.Service
	actions  *action.Service
}

func NewService(cfg config.Config) *Service {
	return &Service{
		manager:  session.NewManager(),
		factory:  browser.NewFactory(cfg),
		snapshot: snapshot.NewService(),
		actions:  action.NewService(time.Now().UnixNano()),
	}
}

func (s *Service) CreateSession(_ context.Context, options model.SessionOptions) (string, error) {
	if err := validation.ValidateSessionOptions(options); err != nil {
		return "", err
	}

	runtime, err := s.factory.Create(options)
	if err != nil {
		return "", err
	}

	s.manager.Add(runtime)
	return runtime.ID, nil
}

func (s *Service) CloseSession(_ context.Context, sessionID string) error {
	runtime, err := s.manager.Delete(sessionID)
	if err != nil {
		return err
	}

	if err := runtime.Close(); err != nil {
		return fmt.Errorf("close session: %w", err)
	}

	return nil
}

func (s *Service) GetSnapshot(_ context.Context, sessionID string) (model.Snapshot, error) {
	runtime, err := s.manager.Get(sessionID)
	if err != nil {
		return model.Snapshot{}, err
	}

	var result model.Snapshot
	err = runtime.WithLock(func() error {
		snapshotResult, captureErr := s.snapshot.Capture(runtime)
		if captureErr != nil {
			return captureErr
		}
		result = snapshotResult
		return nil
	})
	if err != nil {
		return model.Snapshot{}, err
	}

	return result, nil
}

func (s *Service) PerformAction(_ context.Context, sessionID string, input model.ActionRequest) (model.ActionResult, error) {
	if err := validation.ValidateActionRequest(input); err != nil {
		return model.ActionResult{}, err
	}

	runtime, err := s.manager.Get(sessionID)
	if err != nil {
		return model.ActionResult{}, err
	}

	return s.actions.Execute(runtime, input)
}

func (s *Service) Close() error {
	runtimes := s.manager.Drain()

	var firstErr error
	for _, runtime := range runtimes {
		if err := runtime.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if err := s.factory.Close(); err != nil && firstErr == nil {
		firstErr = err
	}

	return firstErr
}
