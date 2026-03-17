package browser

import (
	"testing"
	"time"

	"agentium/internal/config"
)

func TestNormalizeProxyKey(t *testing.T) {
	if got := normalizeProxyKey(""); got != "direct" {
		t.Fatalf("expected direct key, got %q", got)
	}

	proxy := "http://user:pass@127.0.0.1:8080"
	if got := normalizeProxyKey(proxy); got != proxy {
		t.Fatalf("expected proxy key to be stable, got %q", got)
	}
}

func TestReleaseRootBrowserKeepsIdleBrowserUntilTTL(t *testing.T) {
	factory := NewFactory(config.Config{})
	factory.idleTTL = 20 * time.Millisecond
	factory.browsers["direct"] = &sharedBrowser{refCount: 1}

	factory.releaseRootBrowser("direct")

	factory.mu.Lock()
	_, existsImmediately := factory.browsers["direct"]
	factory.mu.Unlock()
	if !existsImmediately {
		t.Fatal("expected browser entry to remain available during idle TTL")
	}

	time.Sleep(50 * time.Millisecond)

	factory.mu.Lock()
	_, existsAfterTTL := factory.browsers["direct"]
	factory.mu.Unlock()
	if existsAfterTTL {
		t.Fatal("expected idle browser entry to be removed after TTL")
	}
}
