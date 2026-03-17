package snapshot

import (
	"encoding/json"
	"strings"
	"testing"

	"agentium/internal/model"
)

func TestLimitSnapshotSizeTrimsElements(t *testing.T) {
	snapshot := model.Snapshot{
		URL:   "https://example.com",
		Title: "Example",
		Viewport: model.Viewport{
			Width:  1280,
			Height: 800,
		},
		Elements: make([]model.SnapshotElement, 0, 50),
	}

	for i := 0; i < 50; i++ {
		snapshot.Elements = append(snapshot.Elements, model.SnapshotElement{
			RefID: i + 1,
			Role:  "text",
			Text:  strings.Repeat("x", 200),
			Attributes: map[string]string{
				"aria-label": strings.Repeat("y", 100),
			},
		})
	}

	trimmed, err := limitSnapshotSize(snapshot, 2048)
	if err != nil {
		t.Fatalf("unexpected trim error: %v", err)
	}

	payload, err := json.Marshal(trimmed)
	if err != nil {
		t.Fatalf("marshal trimmed snapshot: %v", err)
	}

	if len(payload) > 2048 {
		t.Fatalf("expected payload <= 2048 bytes, got %d", len(payload))
	}

	if len(trimmed.Elements) >= len(snapshot.Elements) {
		t.Fatal("expected limiter to remove some elements")
	}
}
