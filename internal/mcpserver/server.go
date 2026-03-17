package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"agentium/internal/app"
	"agentium/internal/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Server struct {
	service app.AgentiumService
}

type createSessionInput struct {
	Proxy      *string `json:"proxy,omitempty" jsonschema:"Optional upstream proxy URL"`
	TimezoneID *string `json:"timezone_id,omitempty" jsonschema:"Optional IANA timezone ID"`
	UserAgent  *string `json:"user_agent,omitempty" jsonschema:"Optional browser user agent"`
	ProfileID  *string `json:"profile_id,omitempty" jsonschema:"Optional stable fingerprint profile identifier"`
}

type sessionOutput struct {
	SessionID string `json:"session_id" jsonschema:"Created session identifier"`
}

type sessionRefInput struct {
	SessionID string `json:"session_id" jsonschema:"Agentium session identifier"`
}

type actionInput struct {
	SessionID string              `json:"session_id" jsonschema:"Agentium session identifier"`
	Action    model.ActionType    `json:"action" jsonschema:"click, fill, type, navigate, scroll, wait_network_idle"`
	RefID     *int                `json:"ref_id,omitempty" jsonschema:"Target snapshot ref_id"`
	Value     *string             `json:"value,omitempty" jsonschema:"Action value"`
	Options   model.ActionOptions `json:"options,omitempty" jsonschema:"Optional action parameters"`
}

func New(service app.AgentiumService) *Server {
	return &Server{service: service}
}

func (s *Server) Build() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "agentium",
		Version: "1.1.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentium_create_session",
		Description: "Create a browser session with optional proxy, timezone, and user agent overrides.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"proxy": map[string]any{
					"type":        "string",
					"description": "Optional upstream proxy URL",
				},
				"timezone_id": map[string]any{
					"type":        "string",
					"description": "Optional IANA timezone ID",
				},
				"user_agent": map[string]any{
					"type":        "string",
					"description": "Optional browser user agent",
				},
				"profile_id": map[string]any{
					"type":        "string",
					"description": "Optional stable fingerprint profile identifier",
				},
			},
			"additionalProperties": false,
		},
	}, s.createSession)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentium_get_snapshot",
		Description: "Return a distilled DOM snapshot for a live browser session.",
	}, s.getSnapshot)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentium_perform_action",
		Description: "Perform an action against the current session page and return resulting network telemetry.",
	}, s.performAction)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentium_close_session",
		Description: "Close a browser session and dispose the associated browser context.",
	}, s.closeSession)

	return server
}

func (s *Server) RunStdio(ctx context.Context) error {
	return s.Build().Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) createSession(ctx context.Context, _ *mcp.CallToolRequest, input createSessionInput) (*mcp.CallToolResult, sessionOutput, error) {
	options := model.SessionOptions{}
	if input.Proxy != nil {
		options.Proxy = *input.Proxy
	}
	if input.TimezoneID != nil {
		options.TimezoneID = *input.TimezoneID
	}
	if input.UserAgent != nil {
		options.UserAgent = *input.UserAgent
	}
	if input.ProfileID != nil {
		options.ProfileID = *input.ProfileID
	}

	id, err := s.service.CreateSession(ctx, options)
	if err != nil {
		return nil, sessionOutput{}, err
	}

	return nil, sessionOutput{SessionID: id}, nil
}

func (s *Server) getSnapshot(ctx context.Context, _ *mcp.CallToolRequest, input sessionRefInput) (*mcp.CallToolResult, map[string]string, error) {
	result, err := s.service.GetSnapshot(ctx, input.SessionID)
	if err != nil {
		return nil, nil, err
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal snapshot: %w", err)
	}

	return nil, map[string]string{"snapshot_json": string(payload)}, nil
}

func (s *Server) performAction(ctx context.Context, _ *mcp.CallToolRequest, input actionInput) (*mcp.CallToolResult, model.ActionResult, error) {
	result, err := s.service.PerformAction(ctx, input.SessionID, model.ActionRequest{
		Action:  input.Action,
		RefID:   input.RefID,
		Value:   input.Value,
		Options: input.Options,
	})
	return nil, result, err
}

func (s *Server) closeSession(ctx context.Context, _ *mcp.CallToolRequest, input sessionRefInput) (*mcp.CallToolResult, map[string]bool, error) {
	if err := s.service.CloseSession(ctx, input.SessionID); err != nil {
		return nil, nil, err
	}

	return nil, map[string]bool{"closed": true}, nil
}
