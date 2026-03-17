package model

import "time"

type SessionOptions struct {
	Proxy      string `json:"proxy,omitempty"`
	TimezoneID string `json:"timezone_id,omitempty"`
	Locale     string `json:"locale,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
}

type Viewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type BoundingBox struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

type SnapshotElement struct {
	RefID        int               `json:"ref_id"`
	Role         string            `json:"role"`
	Text         string            `json:"text"`
	Interactable bool              `json:"interactable"`
	Attributes   map[string]string `json:"attributes"`
	BBox         BoundingBox       `json:"bbox"`
}

type Snapshot struct {
	URL      string            `json:"url"`
	Title    string            `json:"title"`
	Viewport Viewport          `json:"viewport"`
	Elements []SnapshotElement `json:"elements"`
}

type PageText struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

type ActionType string

const (
	ActionClick           ActionType = "click"
	ActionFill            ActionType = "fill"
	ActionTypeText        ActionType = "type"
	ActionNavigate        ActionType = "navigate"
	ActionScroll          ActionType = "scroll"
	ActionWaitNetworkIdle ActionType = "wait_network_idle"
)

type ActionOptions struct {
	DelayMS int `json:"delay_ms,omitempty"`
}

type ActionRequest struct {
	Action  ActionType    `json:"action"`
	RefID   *int          `json:"ref_id,omitempty"`
	Value   *string       `json:"value,omitempty"`
	Options ActionOptions `json:"options,omitempty"`
}

type NetworkEvent struct {
	Timestamp    time.Time `json:"timestamp"`
	RequestID    string    `json:"request_id"`
	Method       string    `json:"method"`
	URL          string    `json:"url"`
	Status       int       `json:"status,omitempty"`
	ResourceType string    `json:"resource_type,omitempty"`
	Stage        string    `json:"stage"`
	ErrorText    string    `json:"error_text,omitempty"`
}

type ActionResult struct {
	Success       bool           `json:"success"`
	ErrorMsg      *string        `json:"error_msg"`
	NetworkEvents []NetworkEvent `json:"network_events"`
}
