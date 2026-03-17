package validation

import (
	"errors"
	"net/url"
	"strings"

	"agentium/internal/model"
)

func ValidateSessionOptions(input model.SessionOptions) error {
	if input.Proxy != "" {
		if _, err := url.ParseRequestURI(input.Proxy); err != nil {
			return errors.New("proxy must be a valid URL")
		}
	}

	if len(strings.TrimSpace(input.TimezoneID)) > 128 {
		return errors.New("timezone_id is too long")
	}

	if len(strings.TrimSpace(input.Locale)) > 32 {
		return errors.New("locale is too long")
	}

	if len(strings.TrimSpace(input.UserAgent)) > 512 {
		return errors.New("user_agent is too long")
	}

	if len(strings.TrimSpace(input.ProfileID)) > 128 {
		return errors.New("profile_id is too long")
	}

	return nil
}

func ValidateActionRequest(input model.ActionRequest) error {
	switch input.Action {
	case model.ActionClick, model.ActionFill, model.ActionTypeText, model.ActionNavigate, model.ActionScroll, model.ActionWaitNetworkIdle:
	default:
		return errors.New("action is invalid")
	}

	if input.Options.DelayMS < 0 || input.Options.DelayMS > 10000 {
		return errors.New("options.delay_ms must be between 0 and 10000")
	}

	switch input.Action {
	case model.ActionClick, model.ActionFill, model.ActionTypeText, model.ActionScroll:
		if input.RefID == nil {
			return errors.New("ref_id is required for this action")
		}
	}

	switch input.Action {
	case model.ActionFill, model.ActionTypeText, model.ActionNavigate:
		if input.Value == nil || strings.TrimSpace(*input.Value) == "" {
			return errors.New("value is required for this action")
		}
	}

	return nil
}
