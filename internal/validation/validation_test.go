package validation

import (
	"strings"
	"testing"

	"agentium/internal/model"
)

func TestValidateSessionOptionsRejectsBadProxy(t *testing.T) {
	err := ValidateSessionOptions(model.SessionOptions{Proxy: "://bad-proxy"})
	if err == nil {
		t.Fatal("expected bad proxy to fail validation")
	}
}

func TestValidateActionRequestRequiresRefIDForClick(t *testing.T) {
	err := ValidateActionRequest(model.ActionRequest{Action: model.ActionClick})
	if err == nil {
		t.Fatal("expected click without ref_id to fail validation")
	}
}

func TestValidateActionRequestAcceptsNavigate(t *testing.T) {
	value := "https://example.com"
	err := ValidateActionRequest(model.ActionRequest{
		Action: model.ActionNavigate,
		Value:  &value,
	})
	if err != nil {
		t.Fatalf("expected navigate request to pass validation: %v", err)
	}
}

func TestValidateSessionOptionsRejectsLongProfileID(t *testing.T) {
	err := ValidateSessionOptions(model.SessionOptions{ProfileID: strings.Repeat("a", 129)})
	if err == nil {
		t.Fatal("expected long profile_id to fail validation")
	}
}

func TestValidateSessionOptionsAcceptsPersistentMode(t *testing.T) {
	err := ValidateSessionOptions(model.SessionOptions{SessionMode: model.SessionModePersistent})
	if err != nil {
		t.Fatalf("expected persistent session mode to pass validation: %v", err)
	}
}

func TestValidateSessionOptionsRejectsUnknownSessionMode(t *testing.T) {
	err := ValidateSessionOptions(model.SessionOptions{SessionMode: "bad"})
	if err == nil {
		t.Fatal("expected invalid session_mode to fail validation")
	}
}
