package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"agentium/internal/model"
	"agentium/internal/session"
	"github.com/labstack/echo/v4"
)

type stubService struct {
	createSession func(context.Context, model.SessionOptions) (string, error)
	closeSession  func(context.Context, string) error
	getSnapshot   func(context.Context, string) (model.Snapshot, error)
	performAction func(context.Context, string, model.ActionRequest) (model.ActionResult, error)
	close         func() error
}

func (s stubService) CreateSession(ctx context.Context, options model.SessionOptions) (string, error) {
	return s.createSession(ctx, options)
}

func (s stubService) CloseSession(ctx context.Context, sessionID string) error {
	return s.closeSession(ctx, sessionID)
}

func (s stubService) GetSnapshot(ctx context.Context, sessionID string) (model.Snapshot, error) {
	return s.getSnapshot(ctx, sessionID)
}

func (s stubService) PerformAction(ctx context.Context, sessionID string, input model.ActionRequest) (model.ActionResult, error) {
	return s.performAction(ctx, sessionID, input)
}

func (s stubService) Close() error {
	if s.close != nil {
		return s.close()
	}
	return nil
}

func TestCreateSession(t *testing.T) {
	e := echo.New()
	handler := NewHandler(stubService{
		createSession: func(_ context.Context, options model.SessionOptions) (string, error) {
			if options.Locale != "en-GB" {
				t.Fatalf("unexpected locale: %q", options.Locale)
			}
			return "session-123", nil
		},
		closeSession: func(context.Context, string) error { return nil },
		getSnapshot:  func(context.Context, string) (model.Snapshot, error) { return model.Snapshot{}, nil },
		performAction: func(context.Context, string, model.ActionRequest) (model.ActionResult, error) {
			return model.ActionResult{}, nil
		},
	})
	handler.Register(e)

	body := bytes.NewBufferString(`{"locale":"en-GB"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload["session_id"] != "session-123" {
		t.Fatalf("unexpected session id: %q", payload["session_id"])
	}
}

func TestGetSnapshotNotFound(t *testing.T) {
	e := echo.New()
	handler := NewHandler(stubService{
		createSession: func(context.Context, model.SessionOptions) (string, error) { return "", nil },
		closeSession:  func(context.Context, string) error { return nil },
		getSnapshot: func(context.Context, string) (model.Snapshot, error) {
			return model.Snapshot{}, session.ErrSessionNotFound
		},
		performAction: func(context.Context, string, model.ActionRequest) (model.ActionResult, error) {
			return model.ActionResult{}, nil
		},
	})
	handler.Register(e)

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/missing/snapshot", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPerformActionBadRequest(t *testing.T) {
	e := echo.New()
	handler := NewHandler(stubService{
		createSession: func(context.Context, model.SessionOptions) (string, error) { return "", nil },
		closeSession:  func(context.Context, string) error { return nil },
		getSnapshot:   func(context.Context, string) (model.Snapshot, error) { return model.Snapshot{}, nil },
		performAction: func(context.Context, string, model.ActionRequest) (model.ActionResult, error) {
			return model.ActionResult{}, errors.New("action is invalid")
		},
	})
	handler.Register(e)

	body := bytes.NewBufferString(`{"action":"bad"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/s1/action", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHealthz(t *testing.T) {
	e := echo.New()
	handler := NewHandler(stubService{
		createSession: func(context.Context, model.SessionOptions) (string, error) { return "", nil },
		closeSession:  func(context.Context, string) error { return nil },
		getSnapshot:   func(context.Context, string) (model.Snapshot, error) { return model.Snapshot{}, nil },
		performAction: func(context.Context, string, model.ActionRequest) (model.ActionResult, error) {
			return model.ActionResult{}, nil
		},
	})
	handler.Register(e)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
